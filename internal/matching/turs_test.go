package matching

import (
	"testing"

	"github.com/google/uuid"
)

func TestWarmupFactor(t *testing.T) {
	tests := []struct {
		completed int
		expected  float64
	}{
		{0, 1.0},
		{5, 0.75},
		{10, 0.50},
		{15, 0.25},
		{20, 0.0},
		{25, 0.0},
	}
	for _, tt := range tests {
		got := WarmupFactor(tt.completed)
		if got != tt.expected {
			t.Errorf("WarmupFactor(%d) = %v, want %v", tt.completed, got, tt.expected)
		}
	}
}

func TestWeightsForOnlineTask(t *testing.T) {
	w := DefaultWeights().ForTask(true)
	if w.GeoRelevance != 0 {
		t.Errorf("online task GeoRelevance = %v, want 0", w.GeoRelevance)
	}
	total := w.SkillMatch + w.BudgetCompat + w.GeoRelevance +
		w.ExperienceFit + w.BehaviorIntent + w.SpeedProbability
	if diff := total - 1.0; diff > 0.001 || diff < -0.001 {
		t.Errorf("online weights sum = %v, want ~1.0", total)
	}
}

func TestWeightsForOfflineTask(t *testing.T) {
	w := DefaultWeights().ForTask(false)
	total := w.SkillMatch + w.BudgetCompat + w.GeoRelevance +
		w.ExperienceFit + w.BehaviorIntent + w.SpeedProbability
	if diff := total - 1.0; diff > 0.001 || diff < -0.001 {
		t.Errorf("offline weights sum = %v, want 1.0", total)
	}
}

func TestScoreCandidateBasic(t *testing.T) {
	skillA := uuid.New()
	skillB := uuid.New()

	task := TaskInput{
		ID:               uuid.New(),
		RequiredSkillIDs: []uuid.UUID{skillA, skillB},
		MinProficiency:   []int{3, 2},
		Budget:           100,
		Latitude:         40.7128,
		Longitude:        -74.0060,
		IsOnline:         false,
		Urgency:          "normal",
	}

	// Experienced candidate with matching skills
	experienced := CandidateInput{
		UserID:              uuid.New(),
		SkillIDs:            []uuid.UUID{skillA, skillB},
		ProficiencyLevels:   []int{4, 3},
		Latitude:            40.7580,
		Longitude:           -73.9855,
		MaxDistanceKM:       50,
		PreferredBudgetMin:  50,
		PreferredBudgetMax:  200,
		TotalTasksCompleted: 30,
		TotalTasksAccepted:  35,
		TotalTasksNotified:  40,
		AvgResponseMinutes:  3,
		CompletionRate:      0.85,
		AcceptanceRate:      0.87,
		ReliabilityScore:    85,
		AvgReviewScore:      4.5,
		ConsistencyScore:    0.8,
	}

	w := DefaultWeights().ForTask(false)
	score := ScoreCandidate(task, experienced, w)

	if score.TotalScore <= 0 {
		t.Errorf("experienced candidate score = %v, want > 0", score.TotalScore)
	}
	if score.WarmupFactor != 0 {
		t.Errorf("experienced candidate warmup = %v, want 0", score.WarmupFactor)
	}
	if score.WarmupBoost != 0 {
		t.Errorf("experienced candidate warmup boost = %v, want 0", score.WarmupBoost)
	}
}

func TestScoreCandidateColdStart(t *testing.T) {
	task := TaskInput{
		ID:       uuid.New(),
		Budget:   100,
		IsOnline: true,
		Urgency:  "normal",
	}

	newUser := CandidateInput{
		UserID:              uuid.New(),
		TotalTasksCompleted: 2,
		ConsistencyScore:    0.5,
		ReliabilityScore:    50,
	}

	w := DefaultWeights().ForTask(true)
	score := ScoreCandidate(task, newUser, w)

	if score.WarmupFactor != 0.9 {
		t.Errorf("cold start warmup = %v, want 0.9", score.WarmupFactor)
	}
	if score.WarmupBoost <= 0 {
		t.Errorf("cold start should have warmup boost > 0, got %v", score.WarmupBoost)
	}
	if score.BehaviorIntent < 0.45 {
		t.Errorf("cold start behavior floor should be >= 0.45, got %v", score.BehaviorIntent)
	}
}

func TestRankCandidatesOrdering(t *testing.T) {
	skillA := uuid.New()

	task := TaskInput{
		ID:               uuid.New(),
		RequiredSkillIDs: []uuid.UUID{skillA},
		MinProficiency:   []int{3},
		Budget:           100,
		IsOnline:         true,
		Urgency:          "normal",
	}

	strong := CandidateInput{
		UserID:              uuid.New(),
		SkillIDs:            []uuid.UUID{skillA},
		ProficiencyLevels:   []int{5},
		PreferredBudgetMin:  50,
		PreferredBudgetMax:  150,
		TotalTasksCompleted: 40,
		CompletionRate:      0.95,
		AcceptanceRate:      0.90,
		ReliabilityScore:    90,
		AvgReviewScore:      4.8,
		ConsistencyScore:    0.9,
		AvgResponseMinutes:  2,
		TotalTasksNotified:  50,
	}

	weak := CandidateInput{
		UserID:              uuid.New(),
		TotalTasksCompleted: 5,
		CompletionRate:      0.3,
		AcceptanceRate:      0.2,
		ReliabilityScore:    30,
		ConsistencyScore:    0.3,
	}

	results := RankCandidates(task, []CandidateInput{weak, strong})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].UserID != strong.UserID {
		t.Errorf("expected strong candidate ranked first")
	}
	if results[0].TotalScore <= results[1].TotalScore {
		t.Errorf("strong (%v) should score higher than weak (%v)",
			results[0].TotalScore, results[1].TotalScore)
	}
}

func TestHaversine(t *testing.T) {
	// NYC to LA ~ 3944 km
	dist := haversine(40.7128, -74.0060, 34.0522, -118.2437)
	if dist < 3900 || dist > 4000 {
		t.Errorf("NYC to LA = %v km, want ~3944", dist)
	}
	// Same point
	dist = haversine(40.7128, -74.0060, 40.7128, -74.0060)
	if dist != 0 {
		t.Errorf("same point distance = %v, want 0", dist)
	}
}
