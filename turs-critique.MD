# TURS Algorithm — Critical Evaluation

## Executive Summary

TURS is a **solid MVP-grade heuristic ranker**, not a true "algorithm" in the ML/optimization sense. It's a hand-tuned weighted linear combination — the simplest possible approach to multi-criteria scoring. This isn't necessarily bad for an MVP, but it has significant limitations that will surface at scale.

---

## Part 1: What TURS Gets Right

### Clean separation of concerns

The code architecture is genuinely well-structured. The scoring functions are pure (no DB access, no side effects), the types are clearly defined, and the orchestrator cleanly separates "who to match" from "how to deliver." This makes the algorithm testable and replaceable — critical for an MVP that will evolve.

### The 6 dimensions are reasonable

SkillMatch, BudgetCompatibility, GeoRelevance, ExperienceFit, BehaviorIntent, SpeedProbability — these cover the primary axes a task marketplace cares about. No obvious missing *category* of signal (though individual signals within categories are thin — more on this later).

### Online task weight redistribution

The `ForTask(isOnline)` proportional redistribution of GeoRelevance weight is the correct mathematical approach. When geo is irrelevant, you redistribute proportionally rather than just zeroing it out, preserving the relative importance of other dimensions.

### Cold-start warmup

The concept of a graduated warmup with behavior floor + score boost is sound. New users need a chance to be seen, and the dual mechanism (floor prevents penalizing sparse data, boost provides visibility) addresses both aspects of the cold-start problem.

### Haversine implementation

Correctly implemented. Tested. Does what it should.

---

## Part 2: Significant Problems

### Problem 1: Static weights are a ticking time bomb

The weights (0.30, 0.25, 0.15, 0.15, 0.10, 0.05) are hardcoded constants with **no mechanism for learning or adaptation**.

In production task marketplaces (TaskRabbit, Upwork, Thumbtack), matching weights are:
- **Category-specific**: A plumbing task should weight SkillMatch and GeoRelevance much higher than an online tutoring task weights SpeedProbability.
- **Market-condition-aware**: When supply is thin, you relax skill/geo requirements. When demand is thin, you tighten.
- **User-feedback-driven**: Click-through rates, acceptance rates, and completion rates on matched tasks feed back to calibrate weights via A/B testing or bandits.

TURS has none of this. The same weights apply to "Deep clean 2BR apartment" and "Fix slow laptop."

**Impact**: Moderate now (MVP), severe at scale. You'll see mismatches that *feel* wrong to users but score well algorithmically.

### Problem 2: SkillMatch proficiency capping is wrong

```go
ratio := float64(candProf) / float64(minProf)
if ratio > 1.0 {
    ratio = 1.0
}
```

A candidate with proficiency 5 on a skill requiring minimum 1 scores **identically** to a candidate with proficiency 1. There's zero credit for exceeding requirements. This creates a significant ranking distortion:

**Example**: Task requires "Cleaning" at proficiency 1.
- Candidate A: Cleaning proficiency 5 → score 1.0
- Candidate B: Cleaning proficiency 1 → score 1.0

These candidates are rated equal on skill match, but any task giver would prefer the expert. The algorithm should apply a bonus (even a mild one) for exceeding the minimum, not just cap at 1.0.

### Problem 3: BudgetCompatibility has asymmetric edge cases

```go
if budget < bMin {
    ratio := budget / bMin
    return math.Max(0, ratio)
}
// budget > bMax
ratio := bMax / budget
return math.Max(0, ratio)
```

Consider: Task budget = $200, candidate preferred range = $30–$100.

- `ratio = 100/200 = 0.5`

But the candidate might be *thrilled* to take a $200 job — the budget *exceeding* their max is often a good signal, not a penalty. The algorithm treats "budget too high" the same as "budget too low." A $10 job when you want $100 minimum is very different from a $200 job when your max preference is $100.

The decay should be asymmetric: penalize under-budget harshly, penalize over-budget mildly (or even bonus it).

### Problem 4: `Urgency` field is declared but never used

`TaskInput.Urgency` is a string field with values `low/normal/high/critical`, but it appears **nowhere** in any scoring function. It's fetched from the DB, stored in the struct, converted through the type system... and then ignored.

For a task marketplace, urgency should be a multiplicative modifier:
- High urgency → boost SpeedProbability weight
- Critical urgency → boost SpeedProbability *and* narrow geo radius
- Low urgency → increase exploration percentage for cold-start users

This is dead data right now.

### Problem 5: `AvgReviewScore` is fetched but never used in scoring

`CandidateInput.AvgReviewScore` is populated from `user_behavior_metrics` but never referenced by any scoring function. Review scores are arguably the **single most important quality signal** a marketplace has. Not using them is a significant oversight.

You'd expect review score to factor into either ExperienceFit or BehaviorIntent, or as its own dimension.

### Problem 6: GeoRelevance uses (0,0) as "no location" sentinel

```go
if task.Latitude == 0 && task.Longitude == 0 {
    return 0.5
}
if c.Latitude == 0 && c.Longitude == 0 {
    return 0.3
}
```

Latitude 0, Longitude 0 is a real place — it's in the Gulf of Guinea off the coast of West Africa. Using `(0,0)` as a sentinel for "unset" is a classic bug. The types should use `*float64` (nullable) rather than zero-value semantics. The DB schema actually uses `NUMERIC(10,7)` which is nullable — but the `CandidateInput` and `TaskInput` structs use plain `float64`, losing nullability.

**Impact**: Any user near the equator/prime meridian intersection gets incorrect matching. Unlikely for an India-focused MVP, but architecturally sloppy.

### Problem 7: N+1 query in fetchCandidates

```go
for rows.Next() {
    // ... scan candidate ...
    skillRows, err := o.db.QueryContext(ctx,
        `SELECT skill_id, proficiency_level FROM user_skills WHERE user_id = $1`,
        c.UserID,
    )
```

For every candidate, a separate query fetches their skills. With 1,000 candidates, that's 1,001 queries. This should be a single query with a JOIN or a `WHERE user_id = ANY($1)` batch.

**Impact**: Severe at scale. At 100 task doers, you're fine. At 10,000, this is a bottleneck.

### Problem 8: No minimum score threshold

`RankCandidates` returns ALL candidates sorted by score. There's no cutoff. A candidate who scores 3/100 still gets notified if they're in the top 15. The orchestrator should have a minimum score threshold below which candidates are filtered out, regardless of wave size.

---

## Part 3: Test Coverage Assessment

| What's Tested | Quality |
|---|---|
| WarmupFactor | Good — boundary conditions covered |
| Weight normalization (online/offline) | Good |
| Basic scoring (experienced candidate) | Shallow — only checks `> 0` |
| Cold-start scoring | Decent — checks warmup values and floor |
| Ranking order | Decent — strong vs weak ordering |
| Haversine | Good — known distances + same-point |

**What's NOT tested**:
- Individual dimension functions (`skillMatch`, `budgetCompatibility`, `geoRelevance`, `experienceFit`, `speedProbability`) — none have direct unit tests
- Edge cases: zero skills required, candidate with no skills, budget = 0, same location
- The orchestrator `splitByWarmup` and wave logic — no tests at all
- Score ranges — nothing verifies scores stay in [0, 105]

The test suite would catch a catastrophic regression but would miss subtle scoring bugs.

---

## Part 4: Comparison to Industry Approaches

| Feature | TURS | TaskRabbit/Upwork-class | Gap |
|---|---|---|---|
| Scoring method | Weighted linear combination | ML-based (gradient boosted trees, neural) + hand-tuned business rules | Significant — but acceptable for MVP |
| Feature count | 6 dimensions, ~12 raw signals | 50-200+ features including behavioral sequences, time-of-day patterns, task description NLP | Expected gap |
| Personalization | None — same weights for all | Per-user, per-category, per-market | Expected gap |
| Feedback loop | None | A/B tested, continuously retrained | Critical missing piece for post-MVP |
| Cold-start | Linear warmup with fixed threshold (20 tasks) | Multi-armed bandit exploration, collaborative filtering, transfer learning from profile signals | TURS approach is reasonable for MVP |
| Real-time factors | None | Online/busy status, current task load, time-of-day response patterns | Missing but not MVP-critical |
| Explainability | Excellent (ScoreBreakdown) | Often poor | **TURS wins here** — the transparency is a genuine advantage |

---

## Part 5: Verdict

**TURS is a B- matching algorithm for an MVP, with a few fixable bugs that drop it to C+ in its current state.**

### What makes it work for now:
- Clean architecture that's easy to iterate on
- Covers the right dimensions conceptually
- Cold-start handling is thoughtful
- Score transparency (ScoreBreakdown) is excellent for debugging and user trust
- Wave-based delivery with exploration slots is a sound notification strategy

### What needs fixing before production (priority order):

1. **Use `AvgReviewScore`** — this is the most impactful missing signal. Add it to ExperienceFit or create a 7th dimension.
2. **Use `Urgency`** — it's already in the data model, just wire it in as a weight modifier.
3. **Fix proficiency cap** — award partial bonus for exceeding requirements (e.g., `min(ratio, 1.2)` instead of `min(ratio, 1.0)`).
4. **Fix budget asymmetry** — over-budget should decay slower than under-budget.
5. **Fix N+1 query** — batch the skill fetch before it becomes a scaling wall.
6. **Add minimum score threshold** — don't notify candidates who scored below, say, 20/100.
7. **Fix (0,0) sentinel** — use nullable types through the full stack.

### What's fine to defer:
- ML-based scoring (wait until you have real behavioral data)
- Per-category weights (wait until you see which categories dominate)
- Real-time signals (wait until you have enough users for it to matter)

### Confidence level: High (8/10)

Based on reading every line of the scoring, orchestrator, types, tests, seed data, and schema. The analysis is based on the actual implementation, not assumptions.

The fundamental architecture is sound enough that these issues are all fixable without a rewrite. The biggest risk isn't the algorithm quality — it's that you go to production without the feedback loop needed to *know* whether the weights are right. Instrument everything: log the scores, track which notifications lead to acceptances, and build the data pipeline to close that loop early.
