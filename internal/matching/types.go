package matching

import (
	"time"

	"github.com/google/uuid"
)

type TaskInput struct {
	ID              uuid.UUID
	Budget          float64
	CategoryID      uuid.UUID
	RequiredSkills  []uuid.UUID
	IsOnline        bool
	Lat             *float64
	Lng             *float64
	RadiusKM        int
	DurationHours   int
	ComplexityLevel string
	CreatedAt       time.Time
}

type CandidateInput struct {
	UserID               uuid.UUID
	ExperienceLevel      string
	ExperienceMultiplier float64
	MAB                  float64
	RadiusKM             int
	FixedLat             *float64
	FixedLng             *float64

	AcceptanceRate        float64
	MedianResponseSeconds int
	PushOpenRate          float64
	CompletionRate        float64
	ReliabilityScore      float64
	TotalTasksCompleted   int
}

type ScoreBreakdown struct {
	SkillMatch          float64
	BudgetCompatibility float64
	ExperienceFit       float64
	BehaviorIntent      float64
	SpeedProbability    float64
	FinalScore          float64
}
