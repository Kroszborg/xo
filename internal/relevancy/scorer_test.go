package relevancy

import (
	"math"
	"testing"

	"github.com/google/uuid"
)

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func floatPtr(v float64) *float64 { return &v }
func uuidPtr(v uuid.UUID) *uuid.UUID { return &v }

func defaultGiver() GiverProfile {
	return GiverProfile{UserID: uuid.New()}
}

func goodGiver() GiverProfile {
	return GiverProfile{
		UserID:                uuid.New(),
		AvgReviewFromDoers:    4.5,
		TotalReviewsFromDoers: 20,
		TotalTasksPosted:      30,
		TotalTasksCompleted:   25,
	}
}

func badGiver() GiverProfile {
	return GiverProfile{
		UserID:                uuid.New(),
		AvgReviewFromDoers:    1.5,
		TotalReviewsFromDoers: 10,
		TotalTasksPosted:      20,
		TotalTasksCancelled:   10,
	}
}

func approxEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

// --------------------------------------------------------------------------
// TestSkillMatch
// --------------------------------------------------------------------------

func TestSkillMatch_NoRequiredSkills(t *testing.T) {
	task := TaskInput{ID: uuid.New()}
	cand := CandidateInput{UserID: uuid.New()}
	got := skillMatch(task, cand)
	if got != 0.5 {
		t.Errorf("no required skills: got %v, want 0.5", got)
	}
}

func TestSkillMatch_PerfectMatch(t *testing.T) {
	skillA := uuid.New()
	skillB := uuid.New()

	task := TaskInput{
		ID:               uuid.New(),
		RequiredSkillIDs: []uuid.UUID{skillA, skillB},
		MinProficiency:   []int{3, 2},
	}
	cand := CandidateInput{
		UserID:            uuid.New(),
		SkillIDs:          []uuid.UUID{skillA, skillB},
		ProficiencyLevels: []int{3, 2},
	}

	got := skillMatch(task, cand)
	// Exact match: ratio = 1.0, min(1.0, 1.2)/1.2 = 0.8333
	// Both skills have this score so overall base = 0.8333
	if got < 0.75 || got > 0.95 {
		t.Errorf("perfect match: got %v, want ~0.833", got)
	}
}

func TestSkillMatch_ExceedingProficiency(t *testing.T) {
	skillA := uuid.New()

	task := TaskInput{
		ID:               uuid.New(),
		RequiredSkillIDs: []uuid.UUID{skillA},
		MinProficiency:   []int{3},
	}
	exact := CandidateInput{
		UserID:            uuid.New(),
		SkillIDs:          []uuid.UUID{skillA},
		ProficiencyLevels: []int{3},
	}
	exceeding := CandidateInput{
		UserID:            uuid.New(),
		SkillIDs:          []uuid.UUID{skillA},
		ProficiencyLevels: []int{5},
	}

	exactScore := skillMatch(task, exact)
	exceedScore := skillMatch(task, exceeding)

	if exceedScore <= exactScore {
		t.Errorf("exceeding proficiency (%v) should score higher than exact (%v)", exceedScore, exactScore)
	}
}

func TestSkillMatch_MissingSkills(t *testing.T) {
	skillA := uuid.New()
	skillB := uuid.New()

	task := TaskInput{
		ID:               uuid.New(),
		RequiredSkillIDs: []uuid.UUID{skillA, skillB},
		MinProficiency:   []int{3, 2},
	}
	cand := CandidateInput{
		UserID:            uuid.New(),
		SkillIDs:          []uuid.UUID{}, // no skills at all
		ProficiencyLevels: []int{},
	}

	got := skillMatch(task, cand)
	if got != 0 {
		t.Errorf("missing all skills: got %v, want 0", got)
	}
}

func TestSkillMatch_CategoryBonus(t *testing.T) {
	skillA := uuid.New()

	task := TaskInput{
		ID:               uuid.New(),
		RequiredSkillIDs: []uuid.UUID{skillA},
		MinProficiency:   []int{3},
	}
	noCategory := CandidateInput{
		UserID:                 uuid.New(),
		SkillIDs:               []uuid.UUID{skillA},
		ProficiencyLevels:      []int{3},
		CategoryTasksCompleted: 0,
	}
	withCategory := CandidateInput{
		UserID:                 uuid.New(),
		SkillIDs:               []uuid.UUID{skillA},
		ProficiencyLevels:      []int{3},
		CategoryTasksCompleted: 10, // max bonus = 0.15
	}

	baseScore := skillMatch(task, noCategory)
	bonusScore := skillMatch(task, withCategory)

	diff := bonusScore - baseScore
	if diff < 0.10 || diff > 0.20 {
		t.Errorf("category bonus diff = %v, want ~0.15 (base=%v, bonus=%v)", diff, baseScore, bonusScore)
	}
}

func TestSkillMatch_ProficiencyCap(t *testing.T) {
	skillA := uuid.New()

	task := TaskInput{
		ID:               uuid.New(),
		RequiredSkillIDs: []uuid.UUID{skillA},
		MinProficiency:   []int{1},
	}

	// ratio = 10/1 = 10.0, capped at 1.2, so score = 1.2/1.2 = 1.0
	highProf := CandidateInput{
		UserID:            uuid.New(),
		SkillIDs:          []uuid.UUID{skillA},
		ProficiencyLevels: []int{10},
	}
	// ratio = 2/1 = 2.0, capped at 1.2, so score = 1.2/1.2 = 1.0
	modProf := CandidateInput{
		UserID:            uuid.New(),
		SkillIDs:          []uuid.UUID{skillA},
		ProficiencyLevels: []int{2},
	}

	highScore := skillMatch(task, highProf)
	modScore := skillMatch(task, modProf)

	// Both should be capped at the same value (1.0 base)
	if !approxEqual(highScore, modScore, 0.001) {
		t.Errorf("proficiency cap: high=%v, mod=%v, should be equal", highScore, modScore)
	}
}

// --------------------------------------------------------------------------
// TestQualitySignal
// --------------------------------------------------------------------------

func TestQualitySignal_NewUser(t *testing.T) {
	cand := CandidateInput{UserID: uuid.New()}
	got := qualitySignal(cand)
	if !approxEqual(got, 0.5, 0.01) {
		t.Errorf("new user quality: got %v, want ~0.5", got)
	}
}

func TestQualitySignal_ExperiencedGoodUser(t *testing.T) {
	cand := CandidateInput{
		UserID:                 uuid.New(),
		AvgReviewScore:         4.8,
		TotalReviewsReceived:   20,
		CompletionRate:         0.95,
		TotalTasksAccepted:     30,
		CategoryCompletionRate: 0.90,
	}
	got := qualitySignal(cand)
	if got < 0.85 {
		t.Errorf("experienced good user quality: got %v, want > 0.85", got)
	}
}

func TestQualitySignal_BadUser(t *testing.T) {
	cand := CandidateInput{
		UserID:                 uuid.New(),
		AvgReviewScore:         1.5,
		TotalReviewsReceived:   10,
		CompletionRate:         0.2,
		TotalTasksAccepted:     20,
		CategoryCompletionRate: 0.1,
	}
	got := qualitySignal(cand)
	if got > 0.35 {
		t.Errorf("bad user quality: got %v, want < 0.35", got)
	}
}

func TestQualitySignal_PartialConfidence(t *testing.T) {
	// 3 reviews out of 5 needed = 0.6 confidence
	cand := CandidateInput{
		UserID:               uuid.New(),
		AvgReviewScore:       4.5,
		TotalReviewsReceived: 3,
		CompletionRate:       0.9,
		TotalTasksAccepted:   3,
	}
	got := qualitySignal(cand)
	// Partial confidence should pull toward 0.5 from a high measured value
	if got < 0.5 || got > 0.85 {
		t.Errorf("partial confidence quality: got %v, want between 0.5 and 0.85", got)
	}
}

// --------------------------------------------------------------------------
// TestReliabilitySignal
// --------------------------------------------------------------------------

func TestReliabilitySignal_NewUser(t *testing.T) {
	cand := CandidateInput{UserID: uuid.New()}
	got := reliabilitySignal(cand)
	if !approxEqual(got, 0.5, 0.01) {
		t.Errorf("new user reliability: got %v, want ~0.5", got)
	}
}

func TestReliabilitySignal_ReliableUser(t *testing.T) {
	cand := CandidateInput{
		UserID:             uuid.New(),
		AcceptanceRate:     0.95,
		ConsistencyScore:   0.90,
		CompletionRate:     0.92,
		TotalTasksNotified: 50,
	}
	got := reliabilitySignal(cand)
	if got < 0.85 {
		t.Errorf("reliable user: got %v, want > 0.85", got)
	}
}

func TestReliabilitySignal_UnreliableUser(t *testing.T) {
	cand := CandidateInput{
		UserID:             uuid.New(),
		AcceptanceRate:     0.1,
		ConsistencyScore:   0.2,
		CompletionRate:     0.15,
		TotalTasksNotified: 30,
	}
	got := reliabilitySignal(cand)
	if got > 0.25 {
		t.Errorf("unreliable user: got %v, want < 0.25", got)
	}
}

// --------------------------------------------------------------------------
// TestBudgetFit
// --------------------------------------------------------------------------

func TestBudgetFit_NoPreferences(t *testing.T) {
	task := TaskInput{Budget: 100}
	cand := CandidateInput{UserID: uuid.New()}
	got := budgetFit(task, cand)
	if got != 0.5 {
		t.Errorf("no preferences: got %v, want 0.5", got)
	}
}

func TestBudgetFit_InRangeCenter(t *testing.T) {
	task := TaskInput{Budget: 150}
	cand := CandidateInput{
		UserID:             uuid.New(),
		PreferredBudgetMin: 100,
		PreferredBudgetMax: 200,
	}
	got := budgetFit(task, cand)
	// Budget at center: deviation = 0, score = 1.0
	if !approxEqual(got, 1.0, 0.01) {
		t.Errorf("center of range: got %v, want ~1.0", got)
	}
}

func TestBudgetFit_UnderMin(t *testing.T) {
	task := TaskInput{Budget: 50}
	cand := CandidateInput{
		UserID:             uuid.New(),
		PreferredBudgetMin: 100,
		PreferredBudgetMax: 200,
	}
	got := budgetFit(task, cand)
	// Under budget: (50/100) * 0.7 = 0.35
	if !approxEqual(got, 0.35, 0.01) {
		t.Errorf("under min: got %v, want ~0.35", got)
	}
}

func TestBudgetFit_OverMax(t *testing.T) {
	task := TaskInput{Budget: 300}
	cand := CandidateInput{
		UserID:             uuid.New(),
		PreferredBudgetMin: 100,
		PreferredBudgetMax: 200,
	}
	got := budgetFit(task, cand)
	// Over budget: 0.7 + (200/300)*0.2 = 0.7 + 0.133 = 0.833
	if !approxEqual(got, 0.833, 0.02) {
		t.Errorf("over max: got %v, want ~0.833", got)
	}
}

func TestBudgetFit_AsymmetryTest(t *testing.T) {
	cand := CandidateInput{
		UserID:             uuid.New(),
		PreferredBudgetMin: 100,
		PreferredBudgetMax: 200,
	}

	// Way under: budget = 20
	taskUnder := TaskInput{Budget: 20}
	underScore := budgetFit(taskUnder, cand)

	// Way over: budget = 500
	taskOver := TaskInput{Budget: 500}
	overScore := budgetFit(taskOver, cand)

	// Over-budget should be gentler (higher score) than under-budget
	if overScore <= underScore {
		t.Errorf("asymmetry: over-budget (%v) should score higher than under-budget (%v)", overScore, underScore)
	}
}

// --------------------------------------------------------------------------
// TestGeoFit
// --------------------------------------------------------------------------

func TestGeoFit_OnlineTask(t *testing.T) {
	task := TaskInput{
		IsOnline:  true,
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
	}
	cand := CandidateInput{
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
	}
	got := GeoFitScore(task, cand)
	if got != 0 {
		t.Errorf("online task geo: got %v, want 0", got)
	}
}

func TestGeoFit_AtTaskLocation(t *testing.T) {
	task := TaskInput{
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
		Radius:    50,
	}
	cand := CandidateInput{
		Latitude:      floatPtr(40.7128),
		Longitude:     floatPtr(-74.0060),
		MaxDistanceKM: 50,
	}
	got := GeoFitScore(task, cand)
	if !approxEqual(got, 1.0, 0.01) {
		t.Errorf("at location: got %v, want ~1.0", got)
	}
}

func TestGeoFit_AtEdgeOfRadius(t *testing.T) {
	// Distance ~49.7 km (just under 50km radius)
	task := TaskInput{
		Latitude:  floatPtr(40.0),
		Longitude: floatPtr(-74.0),
		Radius:    50,
	}
	// Approximately 50km away
	cand := CandidateInput{
		Latitude:      floatPtr(40.45),
		Longitude:     floatPtr(-74.0),
		MaxDistanceKM: 50,
	}
	got := GeoFitScore(task, cand)
	// At edge: 1.0 - (distance/radius * 0.4) ~ 0.6
	if got < 0.55 || got > 0.75 {
		t.Errorf("at edge: got %v, want ~0.6", got)
	}
}

func TestGeoFit_NilCoordinates(t *testing.T) {
	task := TaskInput{
		Latitude:  nil,
		Longitude: nil,
		Radius:    50,
	}
	cand := CandidateInput{
		Latitude:      floatPtr(40.7128),
		Longitude:     floatPtr(-74.0060),
		MaxDistanceKM: 50,
	}
	got := GeoFitScore(task, cand)
	if got != 0.5 {
		t.Errorf("nil task coords: got %v, want 0.5", got)
	}

	// Candidate nil coords
	task2 := TaskInput{
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
		Radius:    50,
	}
	cand2 := CandidateInput{
		Latitude:  nil,
		Longitude: nil,
	}
	got2 := GeoFitScore(task2, cand2)
	if got2 != 0.5 {
		t.Errorf("nil candidate coords: got %v, want 0.5", got2)
	}
}

// --------------------------------------------------------------------------
// TestResponsiveness
// --------------------------------------------------------------------------

func TestResponsiveness_NoData(t *testing.T) {
	cand := CandidateInput{UserID: uuid.New(), TotalTasksNotified: 0}
	got := responsiveness(cand)
	if got != 0.5 {
		t.Errorf("no data: got %v, want 0.5", got)
	}
}

func TestResponsiveness_FastResponder(t *testing.T) {
	cand := CandidateInput{
		UserID:             uuid.New(),
		TotalTasksNotified: 10,
		AvgResponseMinutes: 2.0,
	}
	got := responsiveness(cand)
	if got != 1.0 {
		t.Errorf("fast responder (2 min): got %v, want 1.0", got)
	}
}

func TestResponsiveness_SlowResponder(t *testing.T) {
	cand := CandidateInput{
		UserID:             uuid.New(),
		TotalTasksNotified: 10,
		AvgResponseMinutes: 60.0,
	}
	got := responsiveness(cand)
	if got != 0.0 {
		t.Errorf("slow responder (60 min): got %v, want 0.0", got)
	}
}

func TestResponsiveness_MediumResponder(t *testing.T) {
	cand := CandidateInput{
		UserID:             uuid.New(),
		TotalTasksNotified: 10,
		AvgResponseMinutes: 30.0,
	}
	got := responsiveness(cand)
	// Linear: 1.0 - (30-5)/(60-5) = 1.0 - 25/55 = 0.5454...
	if !approxEqual(got, 0.5454, 0.01) {
		t.Errorf("medium responder (30 min): got %v, want ~0.545", got)
	}
}

// --------------------------------------------------------------------------
// TestCategoryAffinity
// --------------------------------------------------------------------------

func TestCategoryAffinity_NoSignal(t *testing.T) {
	cand := CandidateInput{UserID: uuid.New()}
	got := categoryAffinity(cand)
	if got != 0.5 {
		t.Errorf("no signal, no skills: got %v, want 0.5", got)
	}
}

func TestCategoryAffinity_NoSignalWithSkills(t *testing.T) {
	cand := CandidateInput{
		UserID:   uuid.New(),
		SkillIDs: []uuid.UUID{uuid.New()},
	}
	got := categoryAffinity(cand)
	if got != 0.6 {
		t.Errorf("no signal with skills: got %v, want 0.6", got)
	}
}

func TestCategoryAffinity_StrongPositive(t *testing.T) {
	cand := CandidateInput{
		UserID: uuid.New(),
		CategoryAffinity: &PreferenceSignal{
			SignalValue: 0.9,
			SampleSize:  20,
		},
	}
	got := categoryAffinity(cand)
	if got < 0.85 {
		t.Errorf("strong positive: got %v, want > 0.85", got)
	}
}

func TestCategoryAffinity_StrongNegative(t *testing.T) {
	cand := CandidateInput{
		UserID: uuid.New(),
		CategoryAffinity: &PreferenceSignal{
			SignalValue: 0.1,
			SampleSize:  20,
		},
	}
	got := categoryAffinity(cand)
	if got > 0.15 {
		t.Errorf("strong negative: got %v, want < 0.15", got)
	}
}

func TestCategoryAffinity_PartialConfidence(t *testing.T) {
	// 2 samples out of 5 needed = 0.4 confidence
	cand := CandidateInput{
		UserID: uuid.New(),
		CategoryAffinity: &PreferenceSignal{
			SignalValue: 0.9,
			SampleSize:  2,
		},
	}
	got := categoryAffinity(cand)
	// 0.4 * 0.9 + 0.6 * 0.5 = 0.36 + 0.30 = 0.66
	if !approxEqual(got, 0.66, 0.01) {
		t.Errorf("partial confidence: got %v, want ~0.66", got)
	}
}

// --------------------------------------------------------------------------
// TestBudgetDesirability
// --------------------------------------------------------------------------

func TestBudgetDesirability_NoHistory(t *testing.T) {
	task := TaskInput{Budget: 100}
	cand := CandidateInput{
		UserID:             uuid.New(),
		PreferredBudgetMin: 80,
		PreferredBudgetMax: 120,
	}
	// Should fall back to budgetFit
	got := budgetDesirability(task, cand)
	expected := budgetFit(task, cand)
	if !approxEqual(got, expected, 0.001) {
		t.Errorf("no history: got %v, want %v (budgetFit fallback)", got, expected)
	}
}

func TestBudgetDesirability_MatchingAcceptHistory(t *testing.T) {
	task := TaskInput{Budget: 100}
	cand := CandidateInput{
		UserID: uuid.New(),
		BudgetAcceptAvg: &PreferenceSignal{
			SignalValue: 100, // accepts tasks at this budget
			SampleSize:  10,
		},
	}
	got := budgetDesirability(task, cand)
	// Budget matches accept average perfectly -> high score
	if got < 0.8 {
		t.Errorf("matching accept history: got %v, want > 0.8", got)
	}
}

func TestBudgetDesirability_MismatchingHistory(t *testing.T) {
	task := TaskInput{Budget: 100}
	cand := CandidateInput{
		UserID: uuid.New(),
		BudgetAcceptAvg: &PreferenceSignal{
			SignalValue: 500, // accepts much higher budgets
			SampleSize:  10,
		},
		BudgetRejectAvg: &PreferenceSignal{
			SignalValue: 100, // rejects at this budget level
			SampleSize:  10,
		},
	}
	got := budgetDesirability(task, cand)
	// Budget matches reject average, far from accept average -> low score
	if got > 0.3 {
		t.Errorf("mismatching history: got %v, want < 0.3", got)
	}
}

// --------------------------------------------------------------------------
// TestGiverReputation
// --------------------------------------------------------------------------

func TestGiverReputation_NoReviews(t *testing.T) {
	giver := GiverProfile{UserID: uuid.New()}
	got := giverReputation(giver)
	if got != 0.5 {
		t.Errorf("no reviews: got %v, want 0.5", got)
	}
}

func TestGiverReputation_GoodGiver(t *testing.T) {
	giver := GiverProfile{
		UserID:                uuid.New(),
		AvgReviewFromDoers:    4.5,
		TotalReviewsFromDoers: 20,
	}
	got := giverReputation(giver)
	if got < 0.85 {
		t.Errorf("good giver: got %v, want > 0.85", got)
	}
}

func TestGiverReputation_BadGiver(t *testing.T) {
	giver := GiverProfile{
		UserID:                uuid.New(),
		AvgReviewFromDoers:    1.5,
		TotalReviewsFromDoers: 15,
	}
	got := giverReputation(giver)
	if got > 0.35 {
		t.Errorf("bad giver: got %v, want < 0.35", got)
	}
}

// --------------------------------------------------------------------------
// TestSimilarTaskHistory
// --------------------------------------------------------------------------

func TestSimilarTaskHistory_NoSimilar(t *testing.T) {
	cand := CandidateInput{UserID: uuid.New()}
	got := similarTaskHistory(cand)
	if got != 0.5 {
		t.Errorf("no similar tasks: got %v, want 0.5", got)
	}
}

func TestSimilarTaskHistory_PreviouslyAccepted(t *testing.T) {
	cand := CandidateInput{
		UserID:              uuid.New(),
		SimilarTaskAccepted: 8,
		SimilarTaskRejected: 2,
	}
	got := similarTaskHistory(cand)
	// confidence = min(1.0, 10/5) = 1.0
	// measured = 8/10 = 0.8
	// result = 1.0 * 0.8 + 0 * 0.5 = 0.8
	if !approxEqual(got, 0.8, 0.01) {
		t.Errorf("previously accepted: got %v, want ~0.8", got)
	}
}

func TestSimilarTaskHistory_PreviouslyRejected(t *testing.T) {
	cand := CandidateInput{
		UserID:              uuid.New(),
		SimilarTaskAccepted: 1,
		SimilarTaskRejected: 9,
	}
	got := similarTaskHistory(cand)
	// confidence = 1.0, measured = 1/10 = 0.1
	if !approxEqual(got, 0.1, 0.01) {
		t.Errorf("previously rejected: got %v, want ~0.1", got)
	}
}

// --------------------------------------------------------------------------
// TestScoreCandidate_Integration
// --------------------------------------------------------------------------

func TestScoreCandidate_Integration(t *testing.T) {
	skillA := uuid.New()
	skillB := uuid.New()

	task := TaskInput{
		ID:               uuid.New(),
		CreatedBy:        uuid.New(),
		RequiredSkillIDs: []uuid.UUID{skillA, skillB},
		MinProficiency:   []int{3, 2},
		Budget:           100,
		Latitude:         floatPtr(40.7128),
		Longitude:        floatPtr(-74.0060),
		Radius:           50,
		IsOnline:         false,
		Urgency:          "normal",
	}

	cand := CandidateInput{
		UserID:                 uuid.New(),
		SkillIDs:               []uuid.UUID{skillA, skillB},
		ProficiencyLevels:      []int{4, 3},
		Latitude:               floatPtr(40.7580),
		Longitude:              floatPtr(-73.9855),
		MaxDistanceKM:          50,
		PreferredBudgetMin:     50,
		PreferredBudgetMax:     200,
		TotalTasksCompleted:    30,
		TotalTasksAccepted:     35,
		TotalTasksNotified:     40,
		TotalReviewsReceived:   20,
		AvgResponseMinutes:     3,
		CompletionRate:         0.85,
		AcceptanceRate:         0.87,
		ReliabilityScore:       85,
		AvgReviewScore:         4.5,
		ConsistencyScore:       0.8,
		CategoryTasksCompleted: 5,
		CategoryCompletionRate: 0.9,
		CategoryAffinity: &PreferenceSignal{
			SignalValue: 0.8,
			SampleSize:  10,
		},
		SimilarTaskAccepted: 5,
		SimilarTaskRejected: 1,
	}

	giver := goodGiver()
	tfWeights := OfflineTaskFitWeights()
	alWeights := DefaultAcceptanceLikelihoodWeights()

	score := ScoreCandidate(task, cand, giver, tfWeights, alWeights)

	// Verify formula: FinalScore = TaskFit * AcceptanceLikelihood * ColdStart * 100
	expected := score.TaskFit * score.AcceptanceLikelihood * score.ColdStartMultiplier * 100
	expected = math.Min(expected, 100)
	expected = math.Round(expected*10000) / 10000

	if !approxEqual(score.FinalScore, expected, 0.01) {
		t.Errorf("FinalScore %v != TaskFit(%v) * AcceptanceLikelihood(%v) * ColdStart(%v) * 100 = %v",
			score.FinalScore, score.TaskFit, score.AcceptanceLikelihood, score.ColdStartMultiplier, expected)
	}

	// Experienced user (30+35 >= 10): cold start multiplier should be 1.0
	if score.ColdStartMultiplier != 1.0 {
		t.Errorf("experienced user cold start: got %v, want 1.0", score.ColdStartMultiplier)
	}

	// Score should be positive and meaningful
	if score.FinalScore < 20 {
		t.Errorf("good candidate FinalScore too low: %v", score.FinalScore)
	}
}

func TestScoreCandidate_MultiplicationProperty(t *testing.T) {
	// A candidate who is a great fit but won't accept should score low
	skillA := uuid.New()

	task := TaskInput{
		ID:               uuid.New(),
		CreatedBy:        uuid.New(),
		RequiredSkillIDs: []uuid.UUID{skillA},
		MinProficiency:   []int{3},
		Budget:           100,
		IsOnline:         true,
		Urgency:          "normal",
	}

	// Great skills but hates this category and giver has terrible reviews
	goodFitBadAccept := CandidateInput{
		UserID:              uuid.New(),
		SkillIDs:            []uuid.UUID{skillA},
		ProficiencyLevels:   []int{5},
		PreferredBudgetMin:  80,
		PreferredBudgetMax:  120,
		TotalTasksCompleted: 20,
		TotalTasksAccepted:  25,
		TotalTasksNotified:  30,
		TotalReviewsReceived: 15,
		CompletionRate:      0.9,
		AcceptanceRate:      0.9,
		ConsistencyScore:    0.85,
		AvgReviewScore:      4.5,
		AvgResponseMinutes:  3,
		CategoryAffinity: &PreferenceSignal{
			SignalValue: 0.05, // hates this category
			SampleSize:  20,
		},
		SimilarTaskAccepted: 0,
		SimilarTaskRejected: 10,
	}

	badGiver := badGiver()

	tfWeights := OnlineTaskFitWeights()
	alWeights := DefaultAcceptanceLikelihoodWeights()

	score := ScoreCandidate(task, goodFitBadAccept, badGiver, tfWeights, alWeights)

	// TaskFit should be high (good skills, good metrics)
	if score.TaskFit < 0.6 {
		t.Errorf("expected high TaskFit, got %v", score.TaskFit)
	}

	// AcceptanceLikelihood should be low (hates category, bad giver, rejected similar)
	if score.AcceptanceLikelihood > 0.45 {
		t.Errorf("expected low AcceptanceLikelihood, got %v", score.AcceptanceLikelihood)
	}

	// FinalScore should be pulled down by the multiplication — significantly
	// less than what TaskFit alone would suggest (TaskFit * 100)
	expectedWithoutAcceptance := score.TaskFit * 100
	if score.FinalScore > expectedWithoutAcceptance*0.5 {
		t.Errorf("multiplication property: FinalScore %v should be well below TaskFit-only estimate %v",
			score.FinalScore, expectedWithoutAcceptance)
	}
}

// --------------------------------------------------------------------------
// TestRankCandidates
// --------------------------------------------------------------------------

func TestRankCandidates_Ordering(t *testing.T) {
	skillA := uuid.New()

	task := TaskInput{
		ID:               uuid.New(),
		CreatedBy:        uuid.New(),
		RequiredSkillIDs: []uuid.UUID{skillA},
		MinProficiency:   []int{3},
		Budget:           100,
		IsOnline:         true,
		Urgency:          "normal",
	}

	strong := CandidateInput{
		UserID:               uuid.New(),
		SkillIDs:             []uuid.UUID{skillA},
		ProficiencyLevels:    []int{5},
		PreferredBudgetMin:   50,
		PreferredBudgetMax:   150,
		TotalTasksCompleted:  40,
		TotalTasksAccepted:   45,
		TotalTasksNotified:   50,
		TotalReviewsReceived: 30,
		CompletionRate:       0.95,
		AcceptanceRate:       0.90,
		ReliabilityScore:     90,
		AvgReviewScore:       4.8,
		ConsistencyScore:     0.9,
		AvgResponseMinutes:   2,
	}

	weak := CandidateInput{
		UserID:              uuid.New(),
		TotalTasksCompleted: 5,
		TotalTasksAccepted:  8,
		CompletionRate:      0.3,
		AcceptanceRate:      0.2,
		ReliabilityScore:    30,
		ConsistencyScore:    0.3,
	}

	giver := defaultGiver()
	results := RankCandidates(task, []CandidateInput{weak, strong}, giver)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].UserID != strong.UserID {
		t.Errorf("expected strong candidate ranked first")
	}
	if results[0].FinalScore <= results[1].FinalScore {
		t.Errorf("strong (%v) should score higher than weak (%v)",
			results[0].FinalScore, results[1].FinalScore)
	}
}

func TestRankCandidates_ColdStartCompetitiveNotDominant(t *testing.T) {
	skillA := uuid.New()

	task := TaskInput{
		ID:               uuid.New(),
		CreatedBy:        uuid.New(),
		RequiredSkillIDs: []uuid.UUID{skillA},
		MinProficiency:   []int{3},
		Budget:           100,
		IsOnline:         true,
		Urgency:          "normal",
	}

	// Experienced, strong candidate
	experienced := CandidateInput{
		UserID:               uuid.New(),
		SkillIDs:             []uuid.UUID{skillA},
		ProficiencyLevels:    []int{5},
		PreferredBudgetMin:   50,
		PreferredBudgetMax:   150,
		TotalTasksCompleted:  40,
		TotalTasksAccepted:   45,
		TotalTasksNotified:   50,
		TotalReviewsReceived: 30,
		CompletionRate:       0.95,
		AcceptanceRate:       0.90,
		AvgReviewScore:       4.8,
		ConsistencyScore:     0.9,
		AvgResponseMinutes:   2,
	}

	// New user (cold-start) with some skills
	newUser := CandidateInput{
		UserID:            uuid.New(),
		SkillIDs:          []uuid.UUID{skillA},
		ProficiencyLevels: []int{3},
	}

	giver := defaultGiver()
	results := RankCandidates(task, []CandidateInput{newUser, experienced}, giver)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// New user should get cold-start multiplier > 1.0
	var newResult, expResult ScoreBreakdown
	for _, r := range results {
		if r.UserID == newUser.UserID {
			newResult = r
		} else {
			expResult = r
		}
	}

	if newResult.ColdStartMultiplier <= 1.0 {
		t.Errorf("cold-start user should have multiplier > 1.0, got %v", newResult.ColdStartMultiplier)
	}
	if expResult.ColdStartMultiplier != 1.0 {
		t.Errorf("experienced user should have multiplier 1.0, got %v", expResult.ColdStartMultiplier)
	}

	// Cold-start should be competitive (non-zero score) but not dominant
	if newResult.FinalScore <= 0 {
		t.Errorf("cold-start user should have non-zero score, got %v", newResult.FinalScore)
	}
	if newResult.FinalScore > expResult.FinalScore {
		t.Errorf("cold-start user (%v) should not dominate experienced user (%v)",
			newResult.FinalScore, expResult.FinalScore)
	}
}

// --------------------------------------------------------------------------
// TestColdStartMultiplier
// --------------------------------------------------------------------------

func TestColdStartMultiplier(t *testing.T) {
	tests := []struct {
		name      string
		accepted  int
		completed int
		want      float64
	}{
		{"zero tasks", 0, 0, 1.3},
		{"5 tasks (2+3)", 2, 3, 1.15},
		{"10 tasks (5+5)", 5, 5, 1.0},
		{"15 tasks (8+7)", 8, 7, 1.0},
		{"mid progress (3+0)", 3, 0, 1.21},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ColdStartMultiplier(tt.accepted, tt.completed)
			if !approxEqual(got, tt.want, 0.01) {
				t.Errorf("ColdStartMultiplier(%d, %d) = %v, want %v",
					tt.accepted, tt.completed, got, tt.want)
			}
		})
	}
}

// --------------------------------------------------------------------------
// TestWeights
// --------------------------------------------------------------------------

func TestOfflineWeightsSum(t *testing.T) {
	w := OfflineTaskFitWeights()
	sum := w.SkillMatch + w.QualitySignal + w.BudgetFit +
		w.ReliabilitySignal + w.GeoFit + w.Responsiveness
	if !approxEqual(sum, 1.0, 0.001) {
		t.Errorf("offline weights sum = %v, want 1.0", sum)
	}
}

func TestOnlineWeightsSum(t *testing.T) {
	w := OnlineTaskFitWeights()
	sum := w.SkillMatch + w.QualitySignal + w.BudgetFit +
		w.ReliabilitySignal + w.GeoFit + w.Responsiveness
	if !approxEqual(sum, 1.0, 0.001) {
		t.Errorf("online weights sum = %v, want 1.0", sum)
	}
}

func TestOnlineWeightsGeoZero(t *testing.T) {
	w := OnlineTaskFitWeights()
	if w.GeoFit != 0 {
		t.Errorf("online GeoFit weight = %v, want 0", w.GeoFit)
	}
}

func TestUrgencyModifierRenormalization(t *testing.T) {
	tests := []struct {
		urgency string
	}{
		{"high"},
		{"critical"},
	}

	for _, tt := range tests {
		t.Run(tt.urgency, func(t *testing.T) {
			w := OfflineTaskFitWeights().ApplyUrgency(tt.urgency)
			sum := w.SkillMatch + w.QualitySignal + w.BudgetFit +
				w.ReliabilitySignal + w.GeoFit + w.Responsiveness
			if !approxEqual(sum, 1.0, 0.001) {
				t.Errorf("urgency=%s: weights sum = %v, want 1.0", tt.urgency, sum)
			}
		})
	}
}

func TestAcceptanceLikelihoodWeightsSum(t *testing.T) {
	w := DefaultAcceptanceLikelihoodWeights()
	sum := w.CategoryAffinity + w.BudgetDesirability + w.GiverReputation + w.SimilarTaskHistory
	if !approxEqual(sum, 1.0, 0.001) {
		t.Errorf("acceptance likelihood weights sum = %v, want 1.0", sum)
	}
}
