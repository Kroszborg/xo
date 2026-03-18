package matching

import "github.com/google/uuid"

// TaskInput contains the task parameters needed for scoring.
type TaskInput struct {
	ID               uuid.UUID
	RequiredSkillIDs []uuid.UUID
	MinProficiency   []int // parallel to RequiredSkillIDs
	Budget           float64
	Latitude         float64
	Longitude        float64
	IsOnline         bool
	Urgency          string // low, normal, high, critical
}

// CandidateInput contains a candidate's data for scoring.
type CandidateInput struct {
	UserID              uuid.UUID
	SkillIDs            []uuid.UUID
	ProficiencyLevels   []int // parallel to SkillIDs
	Latitude            float64
	Longitude           float64
	MaxDistanceKM       int
	PreferredBudgetMin  float64
	PreferredBudgetMax  float64
	TotalTasksCompleted int
	TotalTasksAccepted  int
	TotalTasksNotified  int
	AvgResponseMinutes  float64
	CompletionRate      float64
	AcceptanceRate      float64
	ReliabilityScore    float64
	AvgReviewScore      float64
	ConsistencyScore    float64
}

// ScoreBreakdown provides per-dimension scores for transparency.
type ScoreBreakdown struct {
	UserID           uuid.UUID `json:"user_id"`
	TotalScore       float64   `json:"total_score"`
	SkillMatch       float64   `json:"skill_match"`
	BudgetCompat     float64   `json:"budget_compat"`
	GeoRelevance     float64   `json:"geo_relevance"`
	ExperienceFit    float64   `json:"experience_fit"`
	BehaviorIntent   float64   `json:"behavior_intent"`
	SpeedProbability float64   `json:"speed_probability"`
	WarmupFactor     float64   `json:"warmup_factor"`
	WarmupBoost      float64   `json:"warmup_boost"`
}
