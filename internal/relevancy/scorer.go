package relevancy

import (
	"math"
	"sort"
)

// --------------------------------------------------------------------------
// Public API
// --------------------------------------------------------------------------

// ScoreCandidate computes the full relevancy score for a single candidate
// against a given task. The result includes dimension-level breakdowns for
// transparency and debugging.
//
// Formula: FinalScore = TaskFit * AcceptanceLikelihood * ColdStartMultiplier * 100
// The displayed score is capped at 100.
func ScoreCandidate(
	task TaskInput,
	candidate CandidateInput,
	giver GiverProfile,
	tfWeights TaskFitWeights,
	alWeights AcceptanceLikelihoodWeights,
) ScoreBreakdown {
	// Compute each TaskFit dimension (0-1 each)
	sm := skillMatch(task, candidate)
	qs := qualitySignal(candidate)
	rs := reliabilitySignal(candidate)
	bf := budgetFit(task, candidate)
	gf := GeoFitScore(task, candidate)
	rp := responsiveness(candidate)

	// Weighted sum for TaskFit
	tf := sm*tfWeights.SkillMatch +
		qs*tfWeights.QualitySignal +
		rs*tfWeights.ReliabilitySignal +
		bf*tfWeights.BudgetFit +
		gf*tfWeights.GeoFit +
		rp*tfWeights.Responsiveness

	// Compute each AcceptanceLikelihood dimension (0-1 each)
	ca := categoryAffinity(candidate)
	bd := budgetDesirability(task, candidate)
	gr := giverReputation(giver)
	st := similarTaskHistory(candidate)

	// Weighted sum for AcceptanceLikelihood
	al := ca*alWeights.CategoryAffinity +
		bd*alWeights.BudgetDesirability +
		gr*alWeights.GiverReputation +
		st*alWeights.SimilarTaskHistory

	// Cold start multiplier
	csm := ColdStartMultiplier(candidate.TotalTasksAccepted, candidate.TotalTasksCompleted)

	// Final score: capped at 100 for display
	raw := tf * al * csm * 100
	final := math.Min(raw, 100)
	final = math.Round(final*10000) / 10000 // 4 decimal places

	return ScoreBreakdown{
		UserID:  candidate.UserID,
		TaskID:  task.ID,

		TaskFit:              math.Round(tf*10000) / 10000,
		AcceptanceLikelihood: math.Round(al*10000) / 10000,
		ColdStartMultiplier:  math.Round(csm*100) / 100,
		FinalScore:           final,

		SkillMatch:        math.Round(sm*10000) / 10000,
		QualitySignal:     math.Round(qs*10000) / 10000,
		ReliabilitySignal: math.Round(rs*10000) / 10000,
		BudgetFit:         math.Round(bf*10000) / 10000,
		GeoFit:            math.Round(gf*10000) / 10000,
		Responsiveness:    math.Round(rp*10000) / 10000,

		CategoryAffinity:   math.Round(ca*10000) / 10000,
		BudgetDesirability: math.Round(bd*10000) / 10000,
		GiverReputation:    math.Round(gr*10000) / 10000,
		SimilarTaskHistory: math.Round(st*10000) / 10000,
	}
}

// RankCandidates scores all candidates for a task and returns them sorted
// by FinalScore descending. It automatically selects the appropriate weights
// based on task.IsOnline and task.Urgency.
func RankCandidates(
	task TaskInput,
	candidates []CandidateInput,
	giver GiverProfile,
) []ScoreBreakdown {
	tfWeights := TaskFitWeightsForTask(task.IsOnline).ApplyUrgency(task.Urgency)
	alWeights := DefaultAcceptanceLikelihoodWeights()

	results := make([]ScoreBreakdown, 0, len(candidates))
	for i := range candidates {
		sb := ScoreCandidate(task, candidates[i], giver, tfWeights, alWeights)
		results = append(results, sb)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})

	return results
}

// --------------------------------------------------------------------------
// TaskFit dimension functions (unexported, 0-1 each)
// --------------------------------------------------------------------------

// skillMatch evaluates how well the candidate's skills match the task requirements.
// Returns 0.5 when the task has no required skills.
func skillMatch(task TaskInput, candidate CandidateInput) float64 {
	if len(task.RequiredSkillIDs) == 0 {
		return 0.5
	}

	// Build a lookup of candidate skills: skillID -> proficiency level
	candidateSkills := make(map[string]int, len(candidate.SkillIDs))
	for i, sid := range candidate.SkillIDs {
		prof := 0
		if i < len(candidate.ProficiencyLevels) {
			prof = candidate.ProficiencyLevels[i]
		}
		candidateSkills[sid.String()] = prof
	}

	var weightedScore, totalWeight float64

	for i, reqSkillID := range task.RequiredSkillIDs {
		reqProf := 1 // default minimum proficiency
		if i < len(task.MinProficiency) && task.MinProficiency[i] > 0 {
			reqProf = task.MinProficiency[i]
		}

		weight := float64(reqProf) // higher required proficiency = more important
		totalWeight += weight

		candProf, hasSkill := candidateSkills[reqSkillID.String()]
		if !hasSkill {
			// Candidate lacks this skill entirely — score 0 for this skill
			continue
		}

		ratio := float64(candProf) / float64(reqProf)
		// Cap the bonus at 1.2x — exceeding the requirement is good but bounded
		score := math.Min(ratio, 1.2) / 1.2
		weightedScore += score * weight
	}

	if totalWeight == 0 {
		return 0.5
	}

	baseScore := weightedScore / totalWeight

	// Category experience bonus: rewarding candidates who have completed
	// tasks in the same category before
	categoryBonus := math.Min(float64(candidate.CategoryTasksCompleted)/10.0, 0.15)

	return clamp(baseScore+categoryBonus, 0, 1.0)
}

// qualitySignal assesses the candidate's overall quality based on reviews
// and completion rates. Uses confidence weighting so new users regress to 0.5.
func qualitySignal(candidate CandidateInput) float64 {
	// Review component
	reviewMeasured := candidate.AvgReviewScore / 5.0
	reviewConfidence := math.Min(1.0, float64(candidate.TotalReviewsReceived)/float64(ConfidenceReviews))
	effectiveReview := reviewConfidence*reviewMeasured + (1.0-reviewConfidence)*0.5

	// Completion component
	completionMeasured := candidate.CompletionRate
	completionConfidence := math.Min(1.0, float64(candidate.TotalTasksAccepted)/float64(ConfidenceBehavior))
	effectiveCompletion := completionConfidence*completionMeasured + (1.0-completionConfidence)*0.5

	// Category completion component (uses same completion confidence)
	categoryCompletionMeasured := candidate.CategoryCompletionRate
	effectiveCategoryCompletion := completionConfidence*categoryCompletionMeasured + (1.0-completionConfidence)*0.5

	return effectiveReview*0.6 + effectiveCompletion*0.2 + effectiveCategoryCompletion*0.2
}

// reliabilitySignal measures how dependable a candidate is based on acceptance
// rate, consistency score, and completion rate.
func reliabilitySignal(candidate CandidateInput) float64 {
	raw := candidate.AcceptanceRate*0.4 +
		candidate.ConsistencyScore*0.3 +
		candidate.CompletionRate*0.3

	confidence := math.Min(1.0, float64(candidate.TotalTasksNotified)/float64(ConfidenceBehavior))
	return confidence*raw + (1.0-confidence)*0.5
}

// budgetFit evaluates how well the task budget aligns with the candidate's
// budget preferences. Returns 0.5 when the candidate has no preferences.
func budgetFit(task TaskInput, candidate CandidateInput) float64 {
	bMin := candidate.PreferredBudgetMin
	bMax := candidate.PreferredBudgetMax
	budget := task.Budget

	// No preferences set
	if bMin == 0 && bMax == 0 {
		return 0.5
	}

	// Default bMax to bMin * 3 if unset
	if bMax == 0 {
		bMax = bMin * 3
	}

	// Ensure valid range
	if bMin > bMax {
		bMin, bMax = bMax, bMin
	}

	switch {
	case budget >= bMin && budget <= bMax:
		// In range: slight penalty for being off-center
		rangeWidth := bMax - bMin
		if rangeWidth == 0 {
			return 1.0
		}
		center := (bMin + bMax) / 2
		deviation := math.Abs(budget-center) / (rangeWidth) * 2
		return 1.0 - (deviation * 0.1)

	case budget < bMin:
		// Under budget: sharp decay
		if bMin == 0 {
			return 0.5
		}
		return (budget / bMin) * 0.7

	default:
		// Over budget: gentle decay
		if budget == 0 {
			return 0.5
		}
		return 0.7 + (bMax/budget)*0.2
	}
}

// responsiveness scores how quickly a candidate typically responds to notifications.
// Uses linear interpolation between 5 min (perfect) and 60 min (zero).
func responsiveness(candidate CandidateInput) float64 {
	if candidate.TotalTasksNotified == 0 {
		return 0.5
	}

	avgMin := candidate.AvgResponseMinutes

	if avgMin <= 5.0 {
		return 1.0
	}
	if avgMin >= 60.0 {
		return 0.0
	}

	// Linear interpolation: 5 min -> 1.0, 60 min -> 0.0
	return 1.0 - (avgMin-5.0)/(60.0-5.0)
}

// --------------------------------------------------------------------------
// AcceptanceLikelihood dimension functions (unexported, 0-1 each)
// --------------------------------------------------------------------------

// categoryAffinity predicts how likely the candidate is to accept tasks in
// this category based on historical accept/reject/ignore behavior.
func categoryAffinity(candidate CandidateInput) float64 {
	if candidate.CategoryAffinity == nil {
		// No signal available — use skill presence as a weak proxy
		if len(candidate.SkillIDs) > 0 {
			return 0.6
		}
		return 0.5
	}

	signal := candidate.CategoryAffinity
	confidence := signal.Confidence(ConfidenceCategory)
	measured := signal.SignalValue // already represents acceptance ratio

	return confidence*measured + (1.0-confidence)*0.5
}

// budgetDesirability predicts whether the candidate finds the task budget
// appealing based on their historical accept/reject budget patterns.
func budgetDesirability(task TaskInput, candidate CandidateInput) float64 {
	acceptSig := candidate.BudgetAcceptAvg
	rejectSig := candidate.BudgetRejectAvg

	// No accept history at all — fall back to budgetFit
	if acceptSig == nil {
		return budgetFit(task, candidate)
	}

	budget := task.Budget
	acceptAvg := acceptSig.SignalValue
	acceptSamples := acceptSig.SampleSize

	// Have both accept and reject data — compare distance to each average
	if rejectSig != nil && rejectSig.SampleSize > 0 {
		rejectAvg := rejectSig.SignalValue
		totalSamples := acceptSamples + rejectSig.SampleSize
		confidence := math.Min(1.0, float64(totalSamples)/5.0)

		// How close is budget to accept avg vs reject avg?
		distToAccept := math.Abs(budget - acceptAvg)
		distToReject := math.Abs(budget - rejectAvg)

		totalDist := distToAccept + distToReject
		if totalDist == 0 {
			// Budget is equidistant (or equal) to both — moderate signal
			measured := 0.5
			return confidence*measured + (1.0-confidence)*budgetFit(task, candidate)
		}

		// Closer to accept avg => higher score
		measured := distToReject / totalDist
		return confidence*measured + (1.0-confidence)*budgetFit(task, candidate)
	}

	// Only accept data — measure based on distance from accept average
	confidence := math.Min(1.0, float64(acceptSamples)/5.0)

	// Compute how close the budget is to the historical accept average
	if acceptAvg == 0 {
		return budgetFit(task, candidate)
	}
	diff := math.Abs(budget-acceptAvg) / acceptAvg
	// diff of 0 => 1.0, diff of 1.0+ => 0.5
	measured := clamp(1.0-diff*0.5, 0.5, 1.0)

	return confidence*measured + (1.0-confidence)*budgetFit(task, candidate)
}

// giverReputation reflects the task giver's reputation as seen by previous
// task doers. A giver with no reviews regresses to neutral (0.5).
func giverReputation(giver GiverProfile) float64 {
	if giver.TotalReviewsFromDoers == 0 {
		return 0.5
	}

	confidence := math.Min(1.0, float64(giver.TotalReviewsFromDoers)/float64(ConfidenceReviews))
	measured := giver.AvgReviewFromDoers / 5.0
	return confidence*measured + (1.0-confidence)*0.5
}

// similarTaskHistory predicts acceptance based on the candidate's past
// behavior with tasks in the same category and similar budget range.
func similarTaskHistory(candidate CandidateInput) float64 {
	accepted := candidate.SimilarTaskAccepted
	rejected := candidate.SimilarTaskRejected
	total := accepted + rejected

	if total == 0 {
		return 0.5
	}

	confidence := math.Min(1.0, float64(total)/float64(ConfidenceCategory))
	measured := float64(accepted) / float64(total)
	return confidence*measured + (1.0-confidence)*0.5
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// clamp restricts val to the range [min, max].
func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
