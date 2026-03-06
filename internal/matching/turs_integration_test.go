package matching

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
)

// TestTURS_IntegrationScenario tests the TURS ranking algorithm with a realistic
// scenario matching the seeded test data in pkg/db/turs_test_seed.sql.
//
// Test Task: web_development, $250 budget, medium complexity, center (40.7128, -74.006), 10km radius
//
// Users with web_development skill:
//   - David   (Elite, ~0.3km, MAB=500, 95% accept, 99% complete)
//   - Grace   (Pro, ~3km, MAB=350, 85% accept, 95% complete)
//   - Jack    (Pro, ~5km, MAB=280, 78% accept, 92% complete)
//   - Kate    (Intermediate, center, MAB=200, 70% accept, 90% complete)
//   - Mia     (Intermediate, ~9km, MAB=220, 75% accept, 91% complete)
//   - Paul    (Beginner, center, MAB=120, 55% accept, 82% complete)
//   - Uma     (New/cold-start, ~0.3km, MAB=200, 0 history)
//   - Victor  (New/cold-start, ~1km, MAB=250, 2 completed)
func TestTURS_IntegrationScenario(t *testing.T) {
	svc := NewTURSService(DefaultWeights())

	// NYC center location for task
	taskLat := 40.7128
	taskLng := -74.006

	// web_development skill ID from seed
	webDevSkill := uuid.MustParse("a1000000-0000-0000-0000-000000000001")

	task := TaskInput{
		ID:             uuid.MustParse("c1000000-0000-0000-0000-000000000100"),
		Budget:         250,
		RequiredSkills: []uuid.UUID{webDevSkill},
		IsOnline:       false,
		Lat:            &taskLat,
		Lng:            &taskLng,
		RadiusKM:       10,
	}

	// Candidate locations (lat/lng)
	davidLat, davidLng := 40.7150, -74.0030   // ~0.3km from center
	graceLat, graceLng := 40.7400, -74.0060   // ~3km north
	jackLat, jackLng := 40.7500, -74.0300     // ~5km NW
	kateLat, kateLng := 40.7128, -74.0060     // center
	miaLat, miaLng := 40.7128, -73.9000       // ~9km east
	paulLat, paulLng := 40.7128, -74.0060     // center
	umaLat, umaLng := 40.7150, -74.0040       // ~0.3km
	victorLat, victorLng := 40.7200, -74.0100 // ~1km

	candidates := []CandidateInput{
		// David (Elite, web_dev + mobile_dev + data_analysis)
		{
			UserID:                uuid.MustParse("b1000000-0000-0000-0000-000000000004"),
			ExperienceLevel:       "elite",
			ExperienceMultiplier:  1.80,
			MAB:                   500,
			Skills:                []uuid.UUID{webDevSkill},
			AcceptanceRate:        0.95,
			MedianResponseSeconds: 45,
			PushOpenRate:          0.90,
			CompletionRate:        0.99,
			ReliabilityScore:      98.5,
			TotalTasksCompleted:   150,
			IsNewUser:             false,
			FixedLat:              &davidLat,
			FixedLng:              &davidLng,
		},
		// Grace (Pro, web_dev + ui_ux)
		{
			UserID:                uuid.MustParse("b1000000-0000-0000-0000-000000000007"),
			ExperienceLevel:       "pro",
			ExperienceMultiplier:  1.40,
			MAB:                   350,
			Skills:                []uuid.UUID{webDevSkill},
			AcceptanceRate:        0.85,
			MedianResponseSeconds: 90,
			PushOpenRate:          0.80,
			CompletionRate:        0.95,
			ReliabilityScore:      92.0,
			TotalTasksCompleted:   75,
			IsNewUser:             false,
			FixedLat:              &graceLat,
			FixedLng:              &graceLng,
		},
		// Jack (Pro, devops + web_dev)
		{
			UserID:                uuid.MustParse("b1000000-0000-0000-0000-000000000010"),
			ExperienceLevel:       "pro",
			ExperienceMultiplier:  1.30,
			MAB:                   280,
			Skills:                []uuid.UUID{webDevSkill},
			AcceptanceRate:        0.78,
			MedianResponseSeconds: 110,
			PushOpenRate:          0.72,
			CompletionRate:        0.92,
			ReliabilityScore:      87.0,
			TotalTasksCompleted:   55,
			IsNewUser:             false,
			FixedLat:              &jackLat,
			FixedLng:              &jackLng,
		},
		// Kate (Intermediate, web_dev)
		{
			UserID:                uuid.MustParse("b1000000-0000-0000-0000-000000000011"),
			ExperienceLevel:       "intermediate",
			ExperienceMultiplier:  1.15,
			MAB:                   200,
			Skills:                []uuid.UUID{webDevSkill},
			AcceptanceRate:        0.70,
			MedianResponseSeconds: 150,
			PushOpenRate:          0.65,
			CompletionRate:        0.90,
			ReliabilityScore:      82.0,
			TotalTasksCompleted:   35,
			IsNewUser:             false,
			FixedLat:              &kateLat,
			FixedLng:              &kateLng,
		},
		// Mia (Intermediate, data_analysis + web_dev)
		{
			UserID:                uuid.MustParse("b1000000-0000-0000-0000-000000000013"),
			ExperienceLevel:       "intermediate",
			ExperienceMultiplier:  1.20,
			MAB:                   220,
			Skills:                []uuid.UUID{webDevSkill},
			AcceptanceRate:        0.75,
			MedianResponseSeconds: 130,
			PushOpenRate:          0.70,
			CompletionRate:        0.91,
			ReliabilityScore:      84.0,
			TotalTasksCompleted:   40,
			IsNewUser:             false,
			FixedLat:              &miaLat,
			FixedLng:              &miaLng,
		},
		// Paul (Beginner, web_dev) - should rank lower due to budget mismatch
		{
			UserID:                uuid.MustParse("b1000000-0000-0000-0000-000000000016"),
			ExperienceLevel:       "beginner",
			ExperienceMultiplier:  0.95,
			MAB:                   120,
			Skills:                []uuid.UUID{webDevSkill},
			AcceptanceRate:        0.55,
			MedianResponseSeconds: 240,
			PushOpenRate:          0.50,
			CompletionRate:        0.82,
			ReliabilityScore:      70.0,
			TotalTasksCompleted:   10,
			IsNewUser:             false,
			FixedLat:              &paulLat,
			FixedLng:              &paulLng,
		},
		// Uma (New/cold-start, web_dev) - brand new user at 0.3km
		{
			UserID:                uuid.MustParse("b1000000-0000-0000-0000-000000000021"),
			ExperienceLevel:       "beginner",
			ExperienceMultiplier:  1.00,
			MAB:                   200,
			Skills:                []uuid.UUID{webDevSkill},
			AcceptanceRate:        0.00, // No history
			MedianResponseSeconds: 0,
			PushOpenRate:          0.00,
			CompletionRate:        1.00,
			ReliabilityScore:      100.0,
			TotalTasksCompleted:   0,
			IsNewUser:             true, // Cold start - get behavior floor
			FixedLat:              &umaLat,
			FixedLng:              &umaLng,
		},
		// Victor (New/cold-start, mobile_dev + web_dev) - 2 completed
		{
			UserID:                uuid.MustParse("b1000000-0000-0000-0000-000000000022"),
			ExperienceLevel:       "intermediate",
			ExperienceMultiplier:  1.00,
			MAB:                   250,
			Skills:                []uuid.UUID{webDevSkill},
			AcceptanceRate:        1.00,
			MedianResponseSeconds: 90,
			PushOpenRate:          1.00,
			CompletionRate:        1.00,
			ReliabilityScore:      100.0,
			TotalTasksCompleted:   2,
			IsNewUser:             true, // Cold start
			FixedLat:              &victorLat,
			FixedLng:              &victorLng,
		},
	}

	ranked := svc.RankCandidates(task, candidates)

	// Log the full ranking for visibility
	t.Log("=== TURS Rankings for Web Development Task ($250, medium, 10km radius) ===")
	for i, r := range ranked {
		name := getUserName(r.UserID)
		t.Logf("%d. %s (Score: %.2f)", i+1, name, r.Score)
		t.Logf("   SkillMatch: %.3f, Budget: %.3f, Geo: %.3f, Exp: %.3f, Behavior: %.3f, Speed: %.3f",
			r.Breakdown.SkillMatch,
			r.Breakdown.BudgetCompatibility,
			r.Breakdown.GeoRelevance,
			r.Breakdown.ExperienceFit,
			r.Breakdown.BehaviorIntent,
			r.Breakdown.SpeedProbability)
	}

	// Validate expected ordering based on TURS scoring logic
	if len(ranked) != 8 {
		t.Fatalf("expected 8 ranked candidates, got %d", len(ranked))
	}

	// For small budget tasks ($250 < $1000), the algorithm correctly:
	// 1. Penalizes elite users (experienceFit=0, budget incompatible)
	// 2. Penalizes pro users (experienceFit=0.3, often budget incompatible)
	// 3. Favors intermediate users (experienceFit=1.0)
	// 4. Supports beginners (experienceFit=0.8)
	// 5. Gives cold-start users a fair chance (behavior floor=0.5)

	// Verify Kate (intermediate, MAB=200, center) is #1 - best overall fit
	topCandidate := ranked[0]
	topName := getUserName(topCandidate.UserID)
	if topName != "Kate" {
		t.Errorf("expected Kate at #1 for budget-appropriate intermediate user, got %s", topName)
	}

	// Verify cold-start users (Uma, Victor) rank competitively (top 4)
	coldStartInTop4 := 0
	for i := 0; i < 4 && i < len(ranked); i++ {
		name := getUserName(ranked[i].UserID)
		if name == "Uma" || name == "Victor" {
			coldStartInTop4++
		}
	}
	if coldStartInTop4 < 1 {
		t.Errorf("expected at least 1 cold-start user in top 4, got %d", coldStartInTop4)
	}

	// Verify David (elite, MAB=500) ranks last due to:
	// - Budget: 0.0 (ratio 0.5 < 0.6)
	// - ExperienceFit: 0.0 (elite penalty for budget < $1000)
	lastCandidate := ranked[len(ranked)-1]
	lastName := getUserName(lastCandidate.UserID)
	if lastName != "David" {
		t.Errorf("expected David (elite, high MAB) at last position, got %s", lastName)
	}

	// Verify pro users (Grace, Jack) rank in bottom half due to experienceFit penalty
	proInBottom4 := 0
	for i := 4; i < len(ranked); i++ {
		name := getUserName(ranked[i].UserID)
		if name == "Grace" || name == "Jack" {
			proInBottom4++
		}
	}
	if proInBottom4 != 2 {
		t.Errorf("expected both pro users (Grace, Jack) in bottom 4, got %d", proInBottom4)
	}
}

// TestTURS_BudgetCompatibilityLevels verifies the budget ratio scoring
func TestTURS_BudgetCompatibilityLevels(t *testing.T) {
	svc := NewTURSService(DefaultWeights()).(*tursService)

	webDevSkill := uuid.MustParse("a1000000-0000-0000-0000-000000000001")

	testCases := []struct {
		name       string
		taskBudget float64
		userMAB    float64
		wantScore  float64
	}{
		// ratio = taskBudget / userMAB
		{"ratio >= 1.2 → 1.0", 300, 200, 1.0},          // 1.5 ratio
		{"ratio = 1.0 → 0.72", 200, 200, 0.72},         // 1.0 ratio
		{"ratio = 0.9 → 0.4", 180, 200, 0.4},           // 0.9 ratio
		{"ratio = 0.7 → 0.2", 140, 200, 0.2},           // 0.7 ratio
		{"ratio < 0.6 → 0", 100, 200, 0},               // 0.5 ratio
		{"David MAB=500 for $250 task", 250, 500, 0},   // 0.5 ratio → 0
		{"Grace MAB=350 for $250 task", 250, 350, 0.2}, // 0.71 ratio
		{"Kate MAB=200 for $250 task", 250, 200, 1.0},  // 1.25 ratio
	}

	taskLat, taskLng := 40.7128, -74.006
	userLat, userLng := 40.7128, -74.006

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task := TaskInput{
				ID:             uuid.New(),
				Budget:         tc.taskBudget,
				RequiredSkills: []uuid.UUID{webDevSkill},
				IsOnline:       false,
				Lat:            &taskLat,
				Lng:            &taskLng,
				RadiusKM:       10,
			}

			candidate := CandidateInput{
				UserID:                uuid.New(),
				ExperienceLevel:       "intermediate",
				MAB:                   tc.userMAB,
				Skills:                []uuid.UUID{webDevSkill},
				AcceptanceRate:        0.8,
				MedianResponseSeconds: 100,
				CompletionRate:        0.9,
				ReliabilityScore:      85,
				TotalTasksCompleted:   20,
				FixedLat:              &userLat,
				FixedLng:              &userLng,
			}

			got := svc.budgetCompatibility(task, candidate)
			if got != tc.wantScore {
				t.Errorf("budget ratio %.2f: expected %.2f, got %.2f",
					tc.taskBudget/tc.userMAB, tc.wantScore, got)
			}
		})
	}
}

// TestTURS_ExperienceFitForSmallBudgets verifies experienceFit for budget < $1000
func TestTURS_ExperienceFitForSmallBudgets(t *testing.T) {
	svc := NewTURSService(DefaultWeights()).(*tursService)

	testCases := []struct {
		level     string
		wantScore float64
	}{
		{"beginner", 0.8},
		{"intermediate", 1.0},
		{"pro", 0.3},
		{"elite", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			task := TaskInput{Budget: 250} // < $1000
			candidate := CandidateInput{ExperienceLevel: tc.level}

			got := svc.experienceFit(task, candidate)
			if got != tc.wantScore {
				t.Errorf("%s: expected %.2f, got %.2f", tc.level, tc.wantScore, got)
			}
		})
	}
}

// TestTURS_ColdStartBehaviorFloor verifies new users get 0.5 behavior floor
func TestTURS_ColdStartBehaviorFloor(t *testing.T) {
	svc := NewTURSService(DefaultWeights()).(*tursService)

	// Uma: brand new, 0% acceptance, should get 0.5 floor
	newUser := CandidateInput{
		AcceptanceRate:      0.0,
		CompletionRate:      1.0,
		ReliabilityScore:    100,
		TotalTasksCompleted: 0,
		IsNewUser:           true,
	}

	got := svc.behaviorIntent(newUser)
	if got != 0.5 {
		t.Errorf("new user behavior floor: expected 0.5, got %.4f", got)
	}

	// Established user with same rates should get raw score
	establishedUser := CandidateInput{
		AcceptanceRate:      0.0,
		CompletionRate:      1.0,
		ReliabilityScore:    100,
		TotalTasksCompleted: 50,
		IsNewUser:           false,
	}

	got = svc.behaviorIntent(establishedUser)
	// raw = (0.0 * 0.6) + (1.0 * 0.3) + (1.0 * 0.1) = 0.4
	want := 0.4
	if got != want {
		t.Errorf("established user: expected %.2f, got %.4f", want, got)
	}
}

// TestTURS_GeoRelevanceDistances verifies geo scoring for different distances
func TestTURS_GeoRelevanceDistances(t *testing.T) {
	svc := NewTURSService(DefaultWeights()).(*tursService)

	taskLat, taskLng := 40.7128, -74.006
	radius := 10

	testCases := []struct {
		name      string
		userLat   float64
		userLng   float64
		wantScore float64
	}{
		{"center (0km)", 40.7128, -74.006, 1.0},
		{"~3km north (within radius)", 40.7400, -74.006, 1.0},
		{"~9km east (within radius)", 40.7128, -73.9000, 1.0},
		{"~15km north (1.5x radius)", 40.8500, -74.006, 0.5}, // within 2x
		{"~25km away (> 2x radius)", 40.9500, -74.006, 0.0},  // beyond 2x
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task := TaskInput{
				IsOnline: false,
				Lat:      &taskLat,
				Lng:      &taskLng,
				RadiusKM: radius,
			}

			candidate := CandidateInput{
				FixedLat: &tc.userLat,
				FixedLng: &tc.userLng,
			}

			got := svc.geoRelevance(task, candidate)
			if got != tc.wantScore {
				t.Errorf("%s: expected %.2f, got %.4f", tc.name, tc.wantScore, got)
			}
		})
	}
}

// TestTURS_SpeedProbabilityScoring verifies response time scoring
func TestTURS_SpeedProbabilityScoring(t *testing.T) {
	svc := NewTURSService(DefaultWeights()).(*tursService)

	testCases := []struct {
		name         string
		responseTime int
		wantScore    float64
	}{
		{"instant (0s)", 0, 0.5},       // neutral for unknown
		{"fast (45s)", 45, 0.85},       // 1 - 45/300 = 0.85
		{"moderate (150s)", 150, 0.5},  // 1 - 150/300 = 0.5
		{"slow (300s)", 300, 0.0},      // 1 - 300/300 = 0
		{"very slow (360s)", 360, 0.0}, // clamped to 0
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := CandidateInput{MedianResponseSeconds: tc.responseTime}

			got := svc.speedProbability(candidate)
			if got < tc.wantScore-0.01 || got > tc.wantScore+0.01 {
				t.Errorf("%s: expected %.2f, got %.4f", tc.name, tc.wantScore, got)
			}
		})
	}
}

// getUserName returns the user name based on the UUID from seed data
func getUserName(id uuid.UUID) string {
	names := map[string]string{
		"b1000000-0000-0000-0000-000000000004": "David",
		"b1000000-0000-0000-0000-000000000005": "Emma",
		"b1000000-0000-0000-0000-000000000006": "Frank",
		"b1000000-0000-0000-0000-000000000007": "Grace",
		"b1000000-0000-0000-0000-000000000008": "Henry",
		"b1000000-0000-0000-0000-000000000009": "Isabel",
		"b1000000-0000-0000-0000-000000000010": "Jack",
		"b1000000-0000-0000-0000-000000000011": "Kate",
		"b1000000-0000-0000-0000-000000000012": "Leo",
		"b1000000-0000-0000-0000-000000000013": "Mia",
		"b1000000-0000-0000-0000-000000000014": "Noah",
		"b1000000-0000-0000-0000-000000000015": "Olivia",
		"b1000000-0000-0000-0000-000000000016": "Paul",
		"b1000000-0000-0000-0000-000000000017": "Quinn",
		"b1000000-0000-0000-0000-000000000018": "Ruby",
		"b1000000-0000-0000-0000-000000000019": "Sam",
		"b1000000-0000-0000-0000-000000000020": "Tara",
		"b1000000-0000-0000-0000-000000000021": "Uma",
		"b1000000-0000-0000-0000-000000000022": "Victor",
		"b1000000-0000-0000-0000-000000000023": "Wendy",
	}
	if name, ok := names[id.String()]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%s)", id.String()[:8])
}
