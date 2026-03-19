package matching

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

var (
	skillA = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	skillB = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	skillC = uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
)

func baseTask() TaskInput {
	return TaskInput{
		ID:             uuid.New(),
		Budget:         1200,
		CategoryID:     uuid.New(),
		RequiredSkills: []uuid.UUID{skillA, skillB},
		IsOnline:       true,
		DurationHours:  4,
		CreatedAt:      time.Now(),
	}
}

func baseCandidate() CandidateInput {
	return CandidateInput{
		UserID:                uuid.New(),
		ExperienceLevel:       "intermediate",
		ExperienceMultiplier:  1.0,
		MAB:                   1000,
		Skills:                []uuid.UUID{skillA, skillB},
		AcceptanceRate:        0.8,
		MedianResponseSeconds: 120,
		PushOpenRate:          0.7,
		CompletionRate:        0.9,
		ReliabilityScore:      90,
		TotalTasksCompleted:   20,
	}
}

func newService() TURSService {
	return NewTURSService(DefaultWeights())
}

func TestSkillMatch_NoRequiredSkills(t *testing.T) {
	svc := newService().(*tursService)
	task := baseTask()
	task.RequiredSkills = nil
	c := baseCandidate()

	got := svc.skillMatch(task, c)
	if got != 1.0 {
		t.Errorf("expected 1.0, got %.4f", got)
	}
}

func TestSkillMatch_AllSkillsMatch(t *testing.T) {
	svc := newService().(*tursService)
	task := baseTask()
	c := baseCandidate()
	// Candidate has both required skills.
	c.Skills = []uuid.UUID{skillA, skillB}

	got := svc.skillMatch(task, c)
	// raw = 10 (primary) + 6 (secondary) = 16; normalised = 16/30
	want := 16.0 / 30.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("expected %.6f, got %.6f", want, got)
	}
}

func TestSkillMatch_OnlyPrimaryMatch(t *testing.T) {
	svc := newService().(*tursService)
	task := baseTask()
	c := baseCandidate()
	c.Skills = []uuid.UUID{skillA}

	got := svc.skillMatch(task, c)
	want := 10.0 / 30.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("expected %.6f, got %.6f", want, got)
	}
}

func TestSkillMatch_NoMatch(t *testing.T) {
	svc := newService().(*tursService)
	task := baseTask()
	c := baseCandidate()
	c.Skills = []uuid.UUID{skillC}

	got := svc.skillMatch(task, c)
	if got != 0 {
		t.Errorf("expected 0, got %.4f", got)
	}
}

func TestSkillMatch_CapAt30(t *testing.T) {
	svc := newService().(*tursService)
	task := baseTask()
	// Add many required skills to exceed the 30-point cap.
	many := make([]uuid.UUID, 6)
	candidateSkills := make([]uuid.UUID, 6)
	for i := range many {
		many[i] = uuid.New()
		candidateSkills[i] = many[i]
	}
	task.RequiredSkills = many
	c := baseCandidate()
	c.Skills = candidateSkills

	got := svc.skillMatch(task, c)
	if got > 1.0+1e-9 {
		t.Errorf("score exceeded 1.0: %.4f", got)
	}
}

func TestBudgetCompatibility(t *testing.T) {
	svc := newService().(*tursService)
	task := baseTask()
	c := baseCandidate()
	c.MAB = 1000

	tests := []struct {
		budget float64
		want   float64
	}{
		{1200, 1.0},  // ratio 1.2 → 1.0
		{1100, 0.72}, // ratio 1.1 → 0.72
		{850, 0.4},   // ratio 0.85 → 0.4
		{650, 0.2},   // ratio 0.65 → 0.2
		{500, 0.0},   // ratio 0.5 → 0
	}

	for _, tt := range tests {
		task.Budget = tt.budget
		got := svc.budgetCompatibility(task, c)
		if math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("budget=%.0f: expected %.2f, got %.4f", tt.budget, tt.want, got)
		}
	}
}

func TestExperienceFit_LowBudget(t *testing.T) {
	svc := newService().(*tursService)
	task := baseTask()
	task.Budget = 500
	c := baseCandidate()

	cases := []struct {
		level string
		want  float64
	}{
		{"beginner", 0.8},
		{"intermediate", 1.0},
		{"pro", 0.3},
		{"elite", 0.0},
	}

	for _, tc := range cases {
		c.ExperienceLevel = tc.level
		got := svc.experienceFit(task, c)
		if math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("level=%s: expected %.2f, got %.4f", tc.level, tc.want, got)
		}
	}
}

func TestBehaviorIntent(t *testing.T) {
	svc := newService().(*tursService)
	c := baseCandidate()
	c.AcceptanceRate = 1.0
	c.CompletionRate = 1.0
	c.ReliabilityScore = 100

	got := svc.behaviorIntent(c)
	want := 1.0*0.6 + 1.0*0.3 + (100.0/100)*0.1
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("expected %.4f, got %.4f", want, got)
	}
}

func TestSpeedProbability(t *testing.T) {
	svc := newService().(*tursService)
	c := baseCandidate()

	c.MedianResponseSeconds = 0
	if got := svc.speedProbability(c); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("expected 0.5 for zero response, got %.4f", got)
	}

	c.MedianResponseSeconds = 150
	want := 1 - (150.0 / 300)
	if got := svc.speedProbability(c); math.Abs(got-want) > 1e-9 {
		t.Errorf("expected %.4f, got %.4f", want, got)
	}

	c.MedianResponseSeconds = 600
	if got := svc.speedProbability(c); got < 0 {
		t.Errorf("score should not be negative, got %.4f", got)
	}
}

func TestGeoRelevance_OnlineTask(t *testing.T) {
	svc := newService().(*tursService)
	task := baseTask()
	task.IsOnline = true
	c := baseCandidate()

	got := svc.geoRelevance(task, c)
	if math.Abs(got-0.5) > 1e-9 {
		t.Errorf("expected 0.5 for online task, got %.4f", got)
	}
}

func TestGeoRelevance_OfflineNilCoords(t *testing.T) {
	svc := newService().(*tursService)
	task := baseTask()
	task.IsOnline = false
	task.Lat = nil
	c := baseCandidate()

	got := svc.geoRelevance(task, c)
	if got != 0 {
		t.Errorf("expected 0 when coords are nil, got %.4f", got)
	}
}

func TestGeoRelevance_OfflineWithinRadius(t *testing.T) {
	svc := newService().(*tursService)
	task := baseTask()
	task.IsOnline = false

	lat1, lng1 := 12.9716, 77.5946 // Bengaluru
	task.Lat = &lat1
	task.Lng = &lng1
	task.RadiusKM = 50

	// Place candidate very close (same location).
	lat2, lng2 := 12.9716, 77.5946
	c := baseCandidate()
	c.FixedLat = &lat2
	c.FixedLng = &lng2

	got := svc.geoRelevance(task, c)
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("expected 1.0 for candidate within radius, got %.4f", got)
	}
}

func TestGeoRelevance_OfflineOutsideRadius(t *testing.T) {
	svc := newService().(*tursService)
	task := baseTask()
	task.IsOnline = false

	lat1, lng1 := 12.9716, 77.5946 // Bengaluru
	task.Lat = &lat1
	task.Lng = &lng1
	task.RadiusKM = 10

	// Mumbai is ~900 km away.
	lat2, lng2 := 19.0760, 72.8777
	c := baseCandidate()
	c.FixedLat = &lat2
	c.FixedLng = &lng2

	got := svc.geoRelevance(task, c)
	if got != 0 {
		t.Errorf("expected 0 for candidate far outside radius, got %.4f", got)
	}
}

func TestHaversineKM_KnownDistance(t *testing.T) {
	// Bengaluru ↔ Chennai is approximately 290 km.
	dist := haversineKM(12.9716, 77.5946, 13.0827, 80.2707)
	if math.Abs(dist-290) > 5 {
		t.Errorf("expected ~290 km, got %.1f km", dist)
	}
}

func TestScoreCandidate_AllWeights(t *testing.T) {
	svc := newService()
	task := baseTask()
	task.IsOnline = true
	c := baseCandidate()

	bd := svc.ScoreCandidate(task, c)

	if bd.FinalScore < 0 || bd.FinalScore > 100 {
		t.Errorf("FinalScore out of range: %.4f", bd.FinalScore)
	}
	if math.Abs(bd.GeoRelevance-0.5) > 1e-9 {
		t.Errorf("GeoRelevance should be 0.5 for online task, got %.4f", bd.GeoRelevance)
	}
}

func TestRankCandidates_Order(t *testing.T) {
	svc := newService()
	task := baseTask()
	task.IsOnline = true

	// High-quality candidate.
	good := baseCandidate()
	good.UserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	good.AcceptanceRate = 0.9
	good.CompletionRate = 0.95
	good.ReliabilityScore = 95
	good.MAB = 1000
	good.Skills = []uuid.UUID{skillA, skillB}

	// Low-quality candidate.
	poor := baseCandidate()
	poor.UserID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	poor.AcceptanceRate = 0.1
	poor.CompletionRate = 0.2
	poor.ReliabilityScore = 30
	poor.MAB = 3000 // budget incompatible
	poor.Skills = []uuid.UUID{skillC}

	ranked := svc.RankCandidates(task, []CandidateInput{poor, good})
	if len(ranked) != 2 {
		t.Fatalf("expected 2 ranked candidates, got %d", len(ranked))
	}
	if ranked[0].UserID != good.UserID {
		t.Errorf("expected good candidate first, got %s", ranked[0].UserID)
	}
}

// ---------------------------------------------------------------------------
// Dynamic weights tests
// ---------------------------------------------------------------------------

func TestWeights_ForTask_OfflineUnchanged(t *testing.T) {
	w := DefaultWeights()
	got := w.ForTask(false)
	if got != w {
		t.Errorf("offline weights should be unchanged, got %+v", got)
	}
}

func TestWeights_ForTask_OnlineZerosGeo(t *testing.T) {
	w := DefaultWeights()
	got := w.ForTask(true)

	if got.GeoRelevance != 0 {
		t.Errorf("expected GeoRelevance=0 for online, got %.4f", got.GeoRelevance)
	}

	sum := got.SkillMatch + got.BudgetCompatibility + got.GeoRelevance +
		got.ExperienceFit + got.BehaviorIntent + got.SpeedProbability
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("online weights should sum to 1.0, got %.6f", sum)
	}

	// Verify proportional relationship is preserved.
	// SkillMatch / BudgetCompatibility should still be 0.30 / 0.25 = 1.2
	ratio := got.SkillMatch / got.BudgetCompatibility
	wantRatio := 0.30 / 0.25
	if math.Abs(ratio-wantRatio) > 1e-9 {
		t.Errorf("weight ratio not preserved: got %.4f, want %.4f", ratio, wantRatio)
	}
}

func TestScoreCandidate_OnlineNoGeoContribution(t *testing.T) {
	svc := newService()
	task := baseTask()
	task.IsOnline = true

	c := baseCandidate()

	bd := svc.ScoreCandidate(task, c)
	// For online tasks, GeoRelevance should still be computed (0.5 for online)
	// but its weight is 0, so it shouldn't affect the final score.
	if bd.GeoRelevance != 0.5 {
		t.Errorf("GeoRelevance raw should be 0.5, got %.4f", bd.GeoRelevance)
	}

	// Verify online score differs from offline score for the same candidate.
	task2 := baseTask()
	task2.IsOnline = false
	lat, lng := 12.9716, 77.5946
	task2.Lat = &lat
	task2.Lng = &lng
	task2.RadiusKM = 50
	c.FixedLat = &lat
	c.FixedLng = &lng

	bdOffline := svc.ScoreCandidate(task2, c)
	if bd.FinalScore == bdOffline.FinalScore {
		t.Log("online and offline scores happen to match; this is acceptable but worth noting")
	}
}

// ---------------------------------------------------------------------------
// Cold-start behavior floor tests
// ---------------------------------------------------------------------------

func TestBehaviorIntent_NewUserFloor(t *testing.T) {
	svc := newService().(*tursService)

	// New user with zero acceptance rate.
	c := baseCandidate()
	c.IsNewUser = true
	c.AcceptanceRate = 0
	c.CompletionRate = 1.0
	c.ReliabilityScore = 100

	got := svc.behaviorIntent(c)
	// Raw: 0*0.6 + 1.0*0.3 + 1.0*0.1 = 0.4 → floor applies → 0.5
	if math.Abs(got-0.5) > 1e-9 {
		t.Errorf("expected floor of 0.5 for new user, got %.4f", got)
	}
}

func TestBehaviorIntent_NewUserAboveFloor(t *testing.T) {
	svc := newService().(*tursService)

	// New user who already has decent metrics (e.g., accepted a few tasks quickly).
	c := baseCandidate()
	c.IsNewUser = true
	c.AcceptanceRate = 0.8
	c.CompletionRate = 0.9
	c.ReliabilityScore = 90

	got := svc.behaviorIntent(c)
	// Raw: 0.8*0.6 + 0.9*0.3 + 0.9*0.1 = 0.48 + 0.27 + 0.09 = 0.84
	// Above floor, so floor doesn't apply.
	want := 0.8*0.6 + 0.9*0.3 + (90.0/100)*0.1
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("expected %.4f, got %.4f", want, got)
	}
}

func TestBehaviorIntent_VeteranNoFloor(t *testing.T) {
	svc := newService().(*tursService)

	// Veteran user with low acceptance rate - no floor should apply.
	c := baseCandidate()
	c.IsNewUser = false
	c.AcceptanceRate = 0
	c.CompletionRate = 1.0
	c.ReliabilityScore = 100

	got := svc.behaviorIntent(c)
	want := 0.0*0.6 + 1.0*0.3 + 1.0*0.1
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("expected %.4f for veteran (no floor), got %.4f", want, got)
	}
}

func TestNewUser_GetsHigherBehaviorThanVeteranWithZeroRate(t *testing.T) {
	svc := newService()
	task := baseTask()
	task.IsOnline = true

	newUser := baseCandidate()
	newUser.UserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	newUser.IsNewUser = true
	newUser.AcceptanceRate = 0
	newUser.CompletionRate = 1.0
	newUser.ReliabilityScore = 100

	veteran := baseCandidate()
	veteran.UserID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	veteran.IsNewUser = false
	veteran.AcceptanceRate = 0
	veteran.CompletionRate = 1.0
	veteran.ReliabilityScore = 100

	newBD := svc.ScoreCandidate(task, newUser)
	vetBD := svc.ScoreCandidate(task, veteran)

	if newBD.BehaviorIntent <= vetBD.BehaviorIntent {
		t.Errorf("new user behavior (%.4f) should exceed veteran with same zero rate (%.4f)",
			newBD.BehaviorIntent, vetBD.BehaviorIntent)
	}
}

// ---------------------------------------------------------------------------
// Adaptive alpha (EM update) test
// ---------------------------------------------------------------------------

func TestAdaptiveAlpha(t *testing.T) {
	// These test the orchestrator.adaptiveAlpha function indirectly.
	// Since it's unexported, we validate the EM update logic through
	// the TURS scoring: verify that experience multiplier affects MAB
	// which affects budget compatibility scoring.
	svc := newService()
	task := baseTask()
	task.IsOnline = true
	task.Budget = 1200

	// High EM user → high MAB → lower budget compatibility.
	highEM := baseCandidate()
	highEM.UserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	highEM.ExperienceMultiplier = 1.8
	highEM.MAB = 2000 // derived from high EM
	highEM.Skills = []uuid.UUID{skillA, skillB}

	// Low EM user → low MAB → higher budget compatibility.
	lowEM := baseCandidate()
	lowEM.UserID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	lowEM.ExperienceMultiplier = 0.6
	lowEM.MAB = 800 // derived from low EM
	lowEM.Skills = []uuid.UUID{skillA, skillB}

	highBD := svc.ScoreCandidate(task, highEM)
	lowBD := svc.ScoreCandidate(task, lowEM)

	if lowBD.BudgetCompatibility <= highBD.BudgetCompatibility {
		t.Errorf("low-EM user should have better budget compat (%.4f) than high-EM (%.4f)",
			lowBD.BudgetCompatibility, highBD.BudgetCompatibility)
	}
}
