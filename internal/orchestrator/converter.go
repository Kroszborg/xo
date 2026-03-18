package orchestrator

import (
	"github.com/google/uuid"
	"github.com/hakaitech/xo/internal/matching"
)

// DBCandidate represents a row from the GetEligibleCandidates query.
type DBCandidate struct {
	UserID              uuid.UUID
	Latitude            *float64
	Longitude           *float64
	PreferredBudgetMin  *float64
	PreferredBudgetMax  *float64
	MaxDistanceKM       *int
	TotalTasksCompleted int
	TotalTasksAccepted  int
	TotalTasksNotified  int
	AvgResponseMinutes  float64
	CompletionRate      float64
	AcceptanceRate      float64
	ReliabilityScore    float64
	AvgReviewScore      float64
	ConsistencyScore    float64
	SkillIDs            []uuid.UUID
	ProficiencyLevels   []int
}

// DBTask represents the task data needed for scoring.
type DBTask struct {
	ID               uuid.UUID
	CreatedBy        uuid.UUID
	Budget           float64
	Latitude         *float64
	Longitude        *float64
	IsOnline         bool
	Urgency          string
	RequiredSkillIDs []uuid.UUID
	MinProficiency   []int
}

// ToMatchingCandidate converts a DBCandidate to a matching.CandidateInput.
func ToMatchingCandidate(db DBCandidate) matching.CandidateInput {
	c := matching.CandidateInput{
		UserID:              db.UserID,
		SkillIDs:            db.SkillIDs,
		ProficiencyLevels:   db.ProficiencyLevels,
		TotalTasksCompleted: db.TotalTasksCompleted,
		TotalTasksAccepted:  db.TotalTasksAccepted,
		TotalTasksNotified:  db.TotalTasksNotified,
		AvgResponseMinutes:  db.AvgResponseMinutes,
		CompletionRate:      db.CompletionRate,
		AcceptanceRate:      db.AcceptanceRate,
		ReliabilityScore:    db.ReliabilityScore,
		AvgReviewScore:      db.AvgReviewScore,
		ConsistencyScore:    db.ConsistencyScore,
	}
	if db.Latitude != nil {
		c.Latitude = *db.Latitude
	}
	if db.Longitude != nil {
		c.Longitude = *db.Longitude
	}
	if db.PreferredBudgetMin != nil {
		c.PreferredBudgetMin = *db.PreferredBudgetMin
	}
	if db.PreferredBudgetMax != nil {
		c.PreferredBudgetMax = *db.PreferredBudgetMax
	}
	if db.MaxDistanceKM != nil {
		c.MaxDistanceKM = *db.MaxDistanceKM
	}
	return c
}

// ToMatchingTask converts a DBTask to a matching.TaskInput.
func ToMatchingTask(db DBTask) matching.TaskInput {
	t := matching.TaskInput{
		ID:               db.ID,
		RequiredSkillIDs: db.RequiredSkillIDs,
		MinProficiency:   db.MinProficiency,
		Budget:           db.Budget,
		IsOnline:         db.IsOnline,
		Urgency:          db.Urgency,
	}
	if db.Latitude != nil {
		t.Latitude = *db.Latitude
	}
	if db.Longitude != nil {
		t.Longitude = *db.Longitude
	}
	return t
}
