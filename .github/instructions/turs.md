# TURS (Task–User Relevancy Score): End-to-End Algorithm Specification

**Purpose:**  
TURS is Sayzo’s real-time scoring framework used to decide *who should receive task notifications first*. It predicts the likelihood that a given user will **accept a specific task quickly and successfully**, while protecting the marketplace from spam, low-quality matches, and budget/expectation mismatch.

---

## 1) System Overview

**High-level flow:**

1. **Hard Filters (Eligibility Gate)**
   - Required skills present
   - Active recently (e.g., last 7 days)
   - Experience Modifier gate (task size vs user level)
   - Budget sanity gate (Task budget ≥ 50% of user’s MAB)

2. **TURS Scoring (Soft Ranking)**
   - Eligible users are scored using multiple signals.
   - Top *N* (e.g., 30) are notified in waves to hit the “accept in 5 minutes” SLA.

3. **Feedback Loop**
   - Accept/reject/response-time updates continuously tune pricing expectations and behavior models.

---

## 2) Final TURS Formula

\[
\textbf{TURS}(u, t) =
0.30 \cdot \text{SkillMatch}
+ 0.25 \cdot \text{BudgetCompatibility}
+ 0.15 \cdot \text{GeoRelevance}
+ 0.15 \cdot \text{ExperienceFit}
+ 0.10 \cdot \text{BehaviorIntent}
+ 0.05 \cdot \text{SpeedProbability}
\]

> **Range:** 0–100  
> **Interpretation:** Higher TURS ⇒ higher likelihood of fast, successful acceptance.

> **Note:** For *online* tasks, GeoRelevance becomes a soft tie-breaker (time/context weighted). For *offline/hyperlocal* tasks, GeoRelevance can act as a hard gate.

---

## 3) Parameter Definitions & Calculations

### 3.1 Skill Match (0–30)

**What it measures:** Capability fit for the task’s required skills and domain.

**Inputs:**
- Core skills match (exact tools/skills)
- Supporting skills match
- Domain/niche experience (e.g., real estate marketing)
- Recency of similar task execution

**Example scoring:**
- Primary skill in core profile: +10  
- Secondary required skills in core: +6 each  
- Domain tag match: +5  
- Similar task in last 30 days: +3  
**Cap:** 30

**Relevance:** Prevents under-qualified matches and reduces failure rate.

---

### 3.2 Budget Compatibility (0–25)

**What it measures:** Likelihood the user will accept the task at the given budget.

**Key concept: MAB (Minimum Acceptable Budget)**  
\[
\textbf{MAB}_u = \text{MarketBaseBudget}_{(category,duration)} \times \text{ExperienceMultiplier}_u
\]

**Scoring bands:**
- Task ≥ 120% of MAB → 25  
- 100–119% → 18  
- 80–99% → 10  
- 60–79% → 5  
- < 60% → 0

**Relevance:** Biggest predictor of acceptance. Protects pros from low-budget spam and routes budget-appropriate tasks to the right supply tier.

---

### 3.3 Geo Relevance (0–15)

**What it measures:** Location suitability for task execution.

**Offline/Hyperlocal tasks (hard relevance):**
- Same city → 15  
- Same state/region → 8–10  
- Else → 0

**Online tasks (soft relevance):**
- Time-zone compatibility (0–6)
- Market/context familiarity (0–4)
- Language match (0–3)
- Minor proximity bias (0–2)  
**Cap:** 15

**Relevance:** Reduces coordination friction (time zones, context, communication). For online tasks, geo is a nudge, not a blocker.

---

### 3.4 Experience Fit (0–15)

**What it measures:** Right-sizing task complexity vs user level.

**Heuristic example (for low-budget, short tasks):**
- Beginner → 12  
- Intermediate → 15  
- Pro → 5  
- Elite → 0

**Relevance:** Avoids over-qualifying (pros ignoring small gigs) and under-qualifying (beginners failing complex work). This improves acceptance speed and completion quality.

---

### 3.5 Behavior & Intent (0–10)

**What it measures:** Readiness to act *now*.

**New users (cold start):**
- Profile completeness (0–3)
- Task interactions in first session (0–3)
- First notification open speed (0–3)
- Recency/time-of-day match (0–1)

**Experienced users:**
- Historical acceptance rate
- Median response time
- Completion reliability

**Relevance:** Drives the “accept in 5 minutes” promise by prioritizing users who are active and responsive.

---

### 3.6 Speed Probability (0–5)

**What it measures:** Probability of fast response given historical notification behavior.

**Inputs:**
- Push open rate
- Median response latency
- Current active window match

**Relevance:** Final tie-breaker to maximize real-time acceptance speed.

---

## 4) Experience Multiplier (Pricing Expectation Engine)

**Purpose:** Estimates how expensive a user’s time is relative to market average.

**Base (Day-0):**
- Beginner: 0.6  
- Intermediate: 1.0  
- Pro: 1.4  
- Elite: 1.8

**Update on acceptance:**
\[
\textbf{EM}_{new} =
\textbf{clamp}\Big(
\textbf{EM}_{old} \cdot (1-\alpha)
+ \textbf{EM}_{old} \cdot \frac{\text{AcceptedBudget}}{\text{ShownBudget}} \cdot \alpha,
\; 0.5,\; 2.0
\Big)
\]
- Learning rate \(\alpha\): 0.2 (first 5 accepts), 0.1 (next 5), 0.05 (steady-state)

**Relevance:** Aligns future notifications with what the user *actually accepts*, not what they claim.

---

## 5) Experience Modifier (Eligibility Gate)

**Purpose:** A coarse filter deciding whether a task should be shown to a user at all based on task size/complexity vs user level.

**Example rule:**
- ₹700, 3-day marketing task → allow Beginner/Intermediate; de-prioritize Pro; block Elite.

**Relevance:** Prevents supply churn and notification fatigue.

---

## 6) Cold-Start Handling (New Users)

- **Initial MAB:** Category median × temporary Experience Multiplier.
- **Controlled exposure:** Show adjacent budget bands first to learn price tolerance quickly.
- **Confidence weighting:** Early signals are discounted until 5–10 accepts are observed.
- **Exploration quota:** Keep 10–15% randomness to surface hidden talent and avoid overfitting.

---

## 7) Notification Strategy

- Rank eligible users by TURS.
- Notify in waves (e.g., 15 immediately, 15 after 60–90s if no accept).
- Stop once accepted.
- Feedback updates Behavior & Intent, Speed Probability, and EM.

---

## 8) Metrics to Monitor

- **Acceptance latency (P50/P90)**
- **Notification waste** = ignored / sent
- **Budget drift** = |initial MAB − true MAB|
- **Completion success rate**
- **Supply churn (pro opt-outs due to spam)**

---

## 9) Design Principles

- **Predict acceptance, not “best talent.”**
- **Protect both sides:** no spam to pros, no overwhelm for beginners.
- **Behavior > self-reported claims.**
- **Soft preferences for online geo; hard gates for offline geo.**
- **Always keep exploration (10–15%).**

---

## 10) One-Line Summary

> **TURS is a real-time, behavior-driven matching score that optimizes for fast acceptance and successful completion by combining skill fit, budget fit, context fit, and readiness to act.**
