package review

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/google/uuid"
)

// Service handles review operations and metric recalculation.
type Service struct {
	db *sql.DB
}

// NewService creates a new review Service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// CreateReview creates a review and recalculates the reviewee's metrics.
func (s *Service) CreateReview(ctx context.Context, input CreateReviewInput) (*Review, error) {
	// Fetch task to determine roles
	var taskCreatedBy, taskAcceptedBy uuid.UUID
	var taskStatus string
	err := s.db.QueryRowContext(ctx,
		`SELECT created_by, accepted_by, status FROM tasks WHERE id = $1`,
		input.TaskID,
	).Scan(&taskCreatedBy, &taskAcceptedBy, &taskStatus)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Verify task is completed
	if taskStatus != "completed" {
		return nil, fmt.Errorf("can only review completed tasks")
	}

	// Determine roles
	var reviewerRole string
	var revieweeID uuid.UUID

	if input.ReviewerID == taskCreatedBy {
		reviewerRole = "task_giver"
		revieweeID = taskAcceptedBy
	} else if input.ReviewerID == taskAcceptedBy {
		reviewerRole = "task_doer"
		revieweeID = taskCreatedBy
	} else {
		return nil, fmt.Errorf("reviewer is not a participant in this task")
	}

	// Insert review
	var review Review
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO task_reviews (task_id, reviewer_id, reviewee_id, reviewer_role, rating, comment)
         VALUES ($1, $2, $3, $4, $5, $6)
         RETURNING id, task_id, reviewer_id, reviewee_id, reviewer_role, rating, comment, created_at`,
		input.TaskID, input.ReviewerID, revieweeID, reviewerRole, input.Rating, input.Comment,
	).Scan(&review.ID, &review.TaskID, &review.ReviewerID, &review.RevieweeID,
		&review.ReviewerRole, &review.Rating, &review.Comment, &review.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert review: %w", err)
	}

	// Recalculate reviewee's behavior metrics
	if err := s.recalculateMetrics(ctx, revieweeID); err != nil {
		// Log but don't fail the review creation
		fmt.Printf("failed to recalculate metrics for %s: %v\n", revieweeID, err)
	}

	return &review, nil
}

// recalculateMetrics updates the reviewee's behavior metrics based on all reviews.
func (s *Service) recalculateMetrics(ctx context.Context, userID uuid.UUID) error {
	// Get average rating and count
	var avgRating float64
	var totalReviews int
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(AVG(rating), 0), COUNT(*) FROM task_reviews WHERE reviewee_id = $1`,
		userID,
	).Scan(&avgRating, &totalReviews)
	if err != nil {
		return err
	}

	// Get task stats
	var completed, accepted, notified int
	s.db.QueryRowContext(ctx,
		`SELECT total_tasks_completed, total_tasks_accepted, total_tasks_notified
         FROM user_behavior_metrics WHERE user_id = $1`,
		userID,
	).Scan(&completed, &accepted, &notified)

	// Calculate rates
	completionRate := float64(completed) / math.Max(1, float64(accepted))
	acceptanceRate := float64(accepted) / math.Max(1, float64(notified))

	// Calculate reliability score
	// (avgRating/5 * 0.4) + (completionRate * 0.4) + (consistency * 0.2) * 100
	var consistency float64
	s.db.QueryRowContext(ctx,
		`SELECT consistency_score FROM user_behavior_metrics WHERE user_id = $1`,
		userID,
	).Scan(&consistency)

	reliability := ((avgRating / 5.0 * 0.4) + (completionRate * 0.4) + (consistency * 0.2)) * 100

	// Update metrics
	_, err = s.db.ExecContext(ctx,
		`UPDATE user_behavior_metrics
         SET average_review_score = $2,
             total_reviews_received = $3,
             completion_rate = $4,
             acceptance_rate = $5,
             reliability_score = $6
         WHERE user_id = $1`,
		userID,
		math.Round(avgRating*100)/100,
		totalReviews,
		math.Round(completionRate*10000)/10000,
		math.Round(acceptanceRate*10000)/10000,
		math.Round(reliability*100)/100,
	)
	return err
}

// GetTaskReviews gets all reviews for a task.
func (s *Service) GetTaskReviews(ctx context.Context, taskID uuid.UUID) ([]Review, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, task_id, reviewer_id, reviewee_id, reviewer_role, rating, comment, created_at
         FROM task_reviews WHERE task_id = $1 ORDER BY created_at DESC`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		var comment sql.NullString
		if err := rows.Scan(&r.ID, &r.TaskID, &r.ReviewerID, &r.RevieweeID, &r.ReviewerRole, &r.Rating, &comment, &r.CreatedAt); err != nil {
			return nil, err
		}
		if comment.Valid {
			r.Comment = comment.String
		}
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}

// GetUserReviews gets all reviews received by a user.
func (s *Service) GetUserReviews(ctx context.Context, userID uuid.UUID) ([]Review, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, task_id, reviewer_id, reviewee_id, reviewer_role, rating, comment, created_at
         FROM task_reviews WHERE reviewee_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		var comment sql.NullString
		if err := rows.Scan(&r.ID, &r.TaskID, &r.ReviewerID, &r.RevieweeID, &r.ReviewerRole, &r.Rating, &comment, &r.CreatedAt); err != nil {
			return nil, err
		}
		if comment.Valid {
			r.Comment = comment.String
		}
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}

// CreateDispute creates a dispute for a task.
func (s *Service) CreateDispute(ctx context.Context, taskID, initiatedBy, againstUser uuid.UUID, reason string) (*Dispute, error) {
	var d Dispute
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO disputes (task_id, initiated_by, against_user, reason)
         VALUES ($1, $2, $3, $4)
         RETURNING id, task_id, initiated_by, against_user, reason, status, created_at`,
		taskID, initiatedBy, againstUser, reason,
	).Scan(&d.ID, &d.TaskID, &d.InitiatedBy, &d.AgainstUser, &d.Reason, &d.Status, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
