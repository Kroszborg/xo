package review

import (
	"time"

	"github.com/google/uuid"
)

// Review represents a task review.
type Review struct {
	ID           uuid.UUID `json:"id"`
	TaskID       uuid.UUID `json:"task_id"`
	ReviewerID   uuid.UUID `json:"reviewer_id"`
	RevieweeID   uuid.UUID `json:"reviewee_id"`
	ReviewerRole string    `json:"reviewer_role"`
	Rating       int       `json:"rating"`
	Comment      string    `json:"comment,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Dispute represents a task dispute.
type Dispute struct {
	ID          uuid.UUID  `json:"id"`
	TaskID      uuid.UUID  `json:"task_id"`
	InitiatedBy uuid.UUID  `json:"initiated_by"`
	AgainstUser uuid.UUID  `json:"against_user"`
	Reason      string     `json:"reason"`
	Status      string     `json:"status"`
	Resolution  *string    `json:"resolution,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// CreateReviewInput holds the input for creating a review.
type CreateReviewInput struct {
	TaskID     uuid.UUID
	ReviewerID uuid.UUID
	Rating     int
	Comment    string
}
