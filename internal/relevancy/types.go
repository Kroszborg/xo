package relevancy

import (
	"time"

	"github.com/google/uuid"
)

// --------------------------------------------------------------------------
// Input types — populated from DB before scoring
// --------------------------------------------------------------------------

// TaskInput contains all task data needed for scoring.
type TaskInput struct {
	ID               uuid.UUID
	CreatedBy        uuid.UUID
	CategoryID       *uuid.UUID
	RequiredSkillIDs []uuid.UUID
	MinProficiency   []int // parallel to RequiredSkillIDs
	Budget           float64
	Latitude         *float64 // nullable — nil means unset
	Longitude        *float64
	Radius           float64 // task giver's radius in km (offline only)
	IsOnline         bool
	Urgency          string // low, normal, high, critical
	CreatedAt        time.Time
}

// CandidateInput contains all candidate data needed for scoring.
type CandidateInput struct {
	UserID             uuid.UUID
	SkillIDs           []uuid.UUID
	ProficiencyLevels  []int // parallel to SkillIDs
	Latitude           *float64
	Longitude          *float64
	MaxDistanceKM      int
	PreferredBudgetMin float64
	PreferredBudgetMax float64

	// Behavior metrics
	TotalTasksCompleted int
	TotalTasksAccepted  int
	TotalTasksNotified  int
	TotalReviewsReceived int
	AvgResponseMinutes  float64
	CompletionRate      float64
	AcceptanceRate      float64
	ReliabilityScore    float64
	AvgReviewScore      float64
	ConsistencyScore    float64

	// Category-specific metrics (for the task's category)
	CategoryTasksCompleted int
	CategoryCompletionRate float64

	// Preference signals (pre-loaded from user_preference_signals)
	CategoryAffinity    *PreferenceSignal
	BudgetAcceptAvg     *PreferenceSignal
	BudgetRejectAvg     *PreferenceSignal
	IgnoreCount         *PreferenceSignal
	SimilarTaskAccepted int
	SimilarTaskRejected int
}

// GiverProfile contains the task giver's reputation data.
type GiverProfile struct {
	UserID              uuid.UUID
	AvgReviewFromDoers  float64
	TotalReviewsFromDoers int
	TotalTasksPosted    int
	TotalTasksCompleted int
	TotalTasksCancelled int
	RepostCount         int
	LastRepostAt        *time.Time
}

// PreferenceSignal represents a single behavioral signal from user_preference_signals.
type PreferenceSignal struct {
	SignalValue float64
	SampleSize  int
}

// Confidence returns confidence level [0, 1] based on sample size and required threshold.
func (p *PreferenceSignal) Confidence(requiredSamples int) float64 {
	if p == nil || requiredSamples <= 0 {
		return 0
	}
	c := float64(p.SampleSize) / float64(requiredSamples)
	if c > 1.0 {
		return 1.0
	}
	return c
}

// --------------------------------------------------------------------------
// Scoring output types
// --------------------------------------------------------------------------

// ScoreBreakdown provides full transparency into how a score was computed.
type ScoreBreakdown struct {
	UserID  uuid.UUID `json:"user_id"`
	TaskID  uuid.UUID `json:"task_id"`

	// Component scores (0-1 each)
	TaskFit              float64 `json:"task_fit"`
	AcceptanceLikelihood float64 `json:"acceptance_likelihood"`
	ColdStartMultiplier  float64 `json:"cold_start_multiplier"`

	// Final combined score (0-100)
	FinalScore float64 `json:"final_score"`

	// TaskFit dimension breakdown
	SkillMatch       float64 `json:"skill_match"`
	QualitySignal    float64 `json:"quality_signal"`
	ReliabilitySignal float64 `json:"reliability_signal"`
	BudgetFit        float64 `json:"budget_fit"`
	GeoFit           float64 `json:"geo_fit"`
	Responsiveness   float64 `json:"responsiveness"`

	// AcceptanceLikelihood dimension breakdown
	CategoryAffinity   float64 `json:"category_affinity"`
	BudgetDesirability float64 `json:"budget_desirability"`
	GiverReputation    float64 `json:"giver_reputation"`
	SimilarTaskHistory float64 `json:"similar_task_history"`
}

// --------------------------------------------------------------------------
// Matching queue types
// --------------------------------------------------------------------------

// QueueEntry represents a single entry in the offline matching queue.
type QueueEntry struct {
	ID                   uuid.UUID
	TaskID               uuid.UUID
	UserID               uuid.UUID
	Score                float64
	TaskFit              float64
	AcceptanceLikelihood float64
	Status               QueueStatus
	Position             int
	NotifiedAt           *time.Time
	RespondedAt          *time.Time
}

// QueueStatus represents the state of a matching queue entry.
type QueueStatus string

const (
	QueueStatusQueued    QueueStatus = "queued"
	QueueStatusActive    QueueStatus = "active"
	QueueStatusNotified  QueueStatus = "notified"
	QueueStatusAccepted  QueueStatus = "accepted"
	QueueStatusDeclined  QueueStatus = "declined"
	QueueStatusIgnored   QueueStatus = "ignored"
	QueueStatusCancelled QueueStatus = "cancelled"
	QueueStatusFiltered  QueueStatus = "filtered"
)

// --------------------------------------------------------------------------
// Constants
// --------------------------------------------------------------------------

const (
	MinScoreOnline  = 15.0  // Minimum score to insert into relevancy_scores
	MinScoreOffline = 25.0  // Minimum score to insert into matching_queue
	ColdStartCap    = 10    // accepted+completed to reach multiplier 1.0
	ColdStartBoost  = 0.3   // Max boost for brand-new users (1.3×)

	OfflineBatchDefault  = 3     // Default parallel notification batch size
	OfflineBatchHigh     = 5     // Batch size for high urgency
	OfflineBatchCritical = 6     // Batch size for critical urgency
	OfflineTimeoutMin    = 10    // Minutes before offline task expires
	OfflineTimeoutCritical = 5   // Minutes for critical urgency

	CooldownBaseSec = 30    // Base repost cooldown in seconds
	CooldownCapSec  = 3600  // Max cooldown (1 hour)

	OnlineTTLHours = 48 // Default online task lifetime

	ConfidenceReviews  = 5  // Reviews needed for full confidence
	ConfidenceBehavior = 10 // Tasks needed for full behavioral confidence
	ConfidenceCategory = 5  // Category interactions for full confidence

	IgnoreRejectionWeight = 0.5 // How much an ignored notification counts as rejection
)
