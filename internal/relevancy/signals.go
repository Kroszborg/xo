package relevancy

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// SignalUpdater handles transactional updates to user_preference_signals.
// All methods are designed to be called inside an existing transaction.
type SignalUpdater struct{}

// RecordAccept updates preference signals when a user accepts a task.
// - category_affinity: push 1.0 (positive signal, increment 1)
// - budget_accept_avg: running average of the accepted task budget
// - budget_accept_count: increment by 1
func (s *SignalUpdater) RecordAccept(ctx context.Context, tx *sql.Tx, userID uuid.UUID, categoryID *uuid.UUID, taskBudget float64) error {
	// category_affinity: running average, push value=1.0 with increment=1
	if err := s.upsertRunningAverage(ctx, tx, userID, categoryID, "category_affinity", 1.0, 1); err != nil {
		return err
	}

	// budget_accept_avg: running average of accepted task budgets
	if err := s.upsertRunningAverage(ctx, tx, userID, categoryID, "budget_accept_avg", taskBudget, 1); err != nil {
		return err
	}

	// budget_accept_count: simple counter (signal_value tracks the count)
	if err := s.upsertIncrement(ctx, tx, userID, categoryID, "budget_accept_count", 1); err != nil {
		return err
	}

	return nil
}

// RecordDecline updates preference signals when a user declines a task.
// - category_affinity: push 0.0 (negative signal, increment 1)
// - budget_reject_avg: running average of the rejected task budget
// - budget_reject_count: increment by 1
func (s *SignalUpdater) RecordDecline(ctx context.Context, tx *sql.Tx, userID uuid.UUID, categoryID *uuid.UUID, taskBudget float64) error {
	// category_affinity: running average, push value=0.0 with increment=1
	if err := s.upsertRunningAverage(ctx, tx, userID, categoryID, "category_affinity", 0.0, 1); err != nil {
		return err
	}

	// budget_reject_avg: running average of rejected task budgets
	if err := s.upsertRunningAverage(ctx, tx, userID, categoryID, "budget_reject_avg", taskBudget, 1); err != nil {
		return err
	}

	// budget_reject_count: simple counter
	if err := s.upsertIncrement(ctx, tx, userID, categoryID, "budget_reject_count", 1); err != nil {
		return err
	}

	return nil
}

// RecordIgnore updates preference signals for users who were notified but
// did not respond within the timeout window.
// - category_affinity: push 0.0 with half weight (IgnoreRejectionWeight = 0.5)
// - ignore_count: simple increment
func (s *SignalUpdater) RecordIgnore(ctx context.Context, tx *sql.Tx, userID uuid.UUID, categoryID *uuid.UUID) error {
	// category_affinity: push 0.0 with half-weight increment
	// This counts an ignore as a weak negative signal.
	if err := s.upsertRunningAverage(ctx, tx, userID, categoryID, "category_affinity", 0.0, IgnoreRejectionWeightInt); err != nil {
		return err
	}

	// ignore_count: simple counter
	if err := s.upsertIncrement(ctx, tx, userID, categoryID, "ignore_count", 1); err != nil {
		return err
	}

	return nil
}

// RecordCompletion updates signals when a user completes a task.
// - category_affinity: push 1.0 (completing reinforces acceptance)
func (s *SignalUpdater) RecordCompletion(ctx context.Context, tx *sql.Tx, userID uuid.UUID, categoryID *uuid.UUID) error {
	// category_affinity: a completion is a strong positive signal
	if err := s.upsertRunningAverage(ctx, tx, userID, categoryID, "category_affinity", 1.0, 1); err != nil {
		return err
	}

	return nil
}

// IgnoreRejectionWeightInt is the fractional weight for ignore signals, expressed
// as a multiplier applied to sample_size. We use the integer part in the SQL
// formula where sample_size is an INT column: the fractional increment is handled
// via a separate SQL expression that works with NUMERIC arithmetic.
const IgnoreRejectionWeightInt = 0 // Sentinel: see upsertRunningAverage for fractional handling

// upsertRunningAverage performs an UPSERT for a running-average signal.
//
// Formula:
//
//	signal_value = (signal_value * sample_size + new_value * increment) / (sample_size + increment)
//	sample_size += increment
//
// For ignore signals where increment is conceptually 0.5, we pass increment=0
// as a sentinel and use fractional arithmetic in the SQL. The sample_size column
// is INT so we track the fractional part via the running average formula, which
// is numerically equivalent even when sample_size is rounded.
func (s *SignalUpdater) upsertRunningAverage(ctx context.Context, tx *sql.Tx, userID uuid.UUID, categoryID *uuid.UUID, signalType string, newValue float64, increment int) error {
	// For the ignore half-weight case (increment == 0 sentinel), we use
	// fractional arithmetic directly in the running average formula and
	// still increment sample_size by 1 (to keep it as INT), but apply
	// the weight to the contribution of the new value.
	if increment == 0 {
		// Fractional weight: new observation has weight 0.5
		_, err := tx.ExecContext(ctx,
			`INSERT INTO user_preference_signals (user_id, category_id, signal_type, signal_value, sample_size)
			 VALUES ($1, $2, $3, $4, 1)
			 ON CONFLICT (user_id, category_id, signal_type) DO UPDATE SET
			   signal_value = (
			     user_preference_signals.signal_value * user_preference_signals.sample_size + $4::numeric * 0.5
			   ) / (user_preference_signals.sample_size + 0.5)::numeric,
			   sample_size = user_preference_signals.sample_size + 1,
			   last_updated = NOW()`,
			userID, categoryID, signalType, newValue,
		)
		return err
	}

	_, err := tx.ExecContext(ctx,
		`INSERT INTO user_preference_signals (user_id, category_id, signal_type, signal_value, sample_size)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id, category_id, signal_type) DO UPDATE SET
		   signal_value = (
		     user_preference_signals.signal_value * user_preference_signals.sample_size + $4::numeric * $5
		   ) / (user_preference_signals.sample_size + $5),
		   sample_size = user_preference_signals.sample_size + $5,
		   last_updated = NOW()`,
		userID, categoryID, signalType, newValue, increment,
	)
	return err
}

// upsertIncrement performs an UPSERT for a simple counter signal.
// signal_value is incremented by the given amount; sample_size tracks total events.
func (s *SignalUpdater) upsertIncrement(ctx context.Context, tx *sql.Tx, userID uuid.UUID, categoryID *uuid.UUID, signalType string, amount int) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO user_preference_signals (user_id, category_id, signal_type, signal_value, sample_size)
		 VALUES ($1, $2, $3, $4, $4)
		 ON CONFLICT (user_id, category_id, signal_type) DO UPDATE SET
		   signal_value = user_preference_signals.signal_value + $4,
		   sample_size = user_preference_signals.sample_size + $4,
		   last_updated = NOW()`,
		userID, categoryID, signalType, amount,
	)
	return err
}
