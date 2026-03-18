package orchestrator

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hakaitech/xo/internal/matching"
	"github.com/hakaitech/xo/internal/notification"
)

const (
	PriorityWindowMinutes = 10
	WaveIntervalSeconds   = 60
	WaveSize              = 15
	ExplorationPercent    = 0.15
)

// Orchestrator manages the task matching and notification pipeline.
type Orchestrator struct {
	db       *sql.DB
	notifier notification.Notifier
}

// NewOrchestrator creates a new Orchestrator.
func NewOrchestrator(db *sql.DB, notifier notification.Notifier) *Orchestrator {
	return &Orchestrator{db: db, notifier: notifier}
}

// ProcessTask runs the full matching pipeline for a task:
// 1. Fetch eligible candidates
// 2. Score and rank using TURS
// 3. Send notifications in waves
func (o *Orchestrator) ProcessTask(ctx context.Context, taskID uuid.UUID) error {
	// Fetch task
	task, err := o.fetchTask(ctx, taskID)
	if err != nil {
		return err
	}

	// Transition to matching
	if err := o.transitionTask(ctx, taskID, "pending", "matching", nil); err != nil {
		return err
	}

	// Fetch candidates
	candidates, err := o.fetchCandidates(ctx, task.CreatedBy)
	if err != nil {
		return err
	}

	if len(candidates) == 0 {
		log.Printf("no candidates for task %s", taskID)
		return nil
	}

	// Convert and score
	taskInput := ToMatchingTask(task)
	var candidateInputs []matching.CandidateInput
	for _, c := range candidates {
		candidateInputs = append(candidateInputs, ToMatchingCandidate(c))
	}

	ranked := matching.RankCandidates(taskInput, candidateInputs)

	// Split into regular and exploration pools
	regular, exploration := splitByWarmup(ranked)

	// Send waves
	return o.sendWaves(ctx, taskID, regular, exploration)
}

// splitByWarmup separates candidates with warmup > 0 for exploration slots.
func splitByWarmup(ranked []matching.ScoreBreakdown) (regular, exploration []matching.ScoreBreakdown) {
	for _, r := range ranked {
		if r.WarmupFactor > 0 {
			exploration = append(exploration, r)
		} else {
			regular = append(regular, r)
		}
	}
	return
}

// sendWaves dispatches notifications in waves of WaveSize with exploration slots.
func (o *Orchestrator) sendWaves(ctx context.Context, taskID uuid.UUID, regular, exploration []matching.ScoreBreakdown) error {
	waveNum := 1
	regIdx := 0
	expIdx := 0

	for regIdx < len(regular) || expIdx < len(exploration) {
		wave := make([]matching.ScoreBreakdown, 0, WaveSize)

		// Calculate exploration slots for this wave
		expSlots := int(math.Ceil(float64(WaveSize) * ExplorationPercent))
		regSlots := WaveSize - expSlots

		// Fill exploration slots
		for i := 0; i < expSlots && expIdx < len(exploration); i++ {
			wave = append(wave, exploration[expIdx])
			expIdx++
		}

		// Fill regular slots
		for i := 0; i < regSlots && regIdx < len(regular); i++ {
			wave = append(wave, regular[regIdx])
			regIdx++
		}

		// Fill remaining slots from whichever pool has candidates
		for len(wave) < WaveSize && regIdx < len(regular) {
			wave = append(wave, regular[regIdx])
			regIdx++
		}
		for len(wave) < WaveSize && expIdx < len(exploration) {
			wave = append(wave, exploration[expIdx])
			expIdx++
		}

		if len(wave) == 0 {
			break
		}

		// Send notifications for this wave
		for _, candidate := range wave {
			isExploration := candidate.WarmupFactor > 0
			if err := o.sendNotification(ctx, taskID, candidate, waveNum, isExploration); err != nil {
				log.Printf("failed to notify %s for task %s: %v", candidate.UserID, taskID, err)
			}
		}

		waveNum++

		// Wait for wave interval before next wave (unless context cancelled)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(WaveIntervalSeconds) * time.Second):
		}
	}

	return nil
}

// sendNotification records and dispatches a single notification.
func (o *Orchestrator) sendNotification(ctx context.Context, taskID uuid.UUID, candidate matching.ScoreBreakdown, waveNum int, isExploration bool) error {
	// Record in task_notifications
	_, err := o.db.ExecContext(ctx,
		`INSERT INTO task_notifications (task_id, user_id, wave_number, score, is_exploration, channel, status, sent_at)
		 VALUES ($1, $2, $3, $4, $5, 'fcm', 'sent', NOW())`,
		taskID, candidate.UserID, waveNum, candidate.TotalScore, isExploration,
	)
	if err != nil {
		return err
	}

	// Dispatch via notifier
	return o.notifier.Notify(ctx, notification.Message{
		UserID: candidate.UserID,
		Type:   "task_match",
		Title:  "New task match!",
		Body:   "A new task matches your skills. Tap to view.",
		Data: map[string]string{
			"task_id": taskID.String(),
			"score":   formatScore(candidate.TotalScore),
			"wave":    formatInt(waveNum),
		},
	})
}

// transitionTask records a state transition.
func (o *Orchestrator) transitionTask(ctx context.Context, taskID uuid.UUID, from, to string, triggeredBy *uuid.UUID) error {
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`UPDATE tasks SET status = $2 WHERE id = $1`,
		taskID, to,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO task_state_transitions (task_id, from_status, to_status, triggered_by)
		 VALUES ($1, $2, $3, $4)`,
		taskID, from, to, triggeredBy,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// fetchTask loads a task from the database.
func (o *Orchestrator) fetchTask(ctx context.Context, taskID uuid.UUID) (DBTask, error) {
	var t DBTask
	var lat, lng sql.NullFloat64

	err := o.db.QueryRowContext(ctx,
		`SELECT id, created_by, budget, latitude, longitude, is_online, urgency FROM tasks WHERE id = $1`,
		taskID,
	).Scan(&t.ID, &t.CreatedBy, &t.Budget, &lat, &lng, &t.IsOnline, &t.Urgency)
	if err != nil {
		return t, err
	}

	if lat.Valid {
		t.Latitude = &lat.Float64
	}
	if lng.Valid {
		t.Longitude = &lng.Float64
	}

	// Fetch required skills
	rows, err := o.db.QueryContext(ctx,
		`SELECT skill_id, minimum_proficiency FROM task_required_skills WHERE task_id = $1`,
		taskID,
	)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	for rows.Next() {
		var sid uuid.UUID
		var prof int
		if err := rows.Scan(&sid, &prof); err != nil {
			return t, err
		}
		t.RequiredSkillIDs = append(t.RequiredSkillIDs, sid)
		t.MinProficiency = append(t.MinProficiency, prof)
	}

	return t, rows.Err()
}

// fetchCandidates loads all eligible task doers.
func (o *Orchestrator) fetchCandidates(ctx context.Context, excludeUserID uuid.UUID) ([]DBCandidate, error) {
	rows, err := o.db.QueryContext(ctx,
		`SELECT
			u.id,
			up.latitude, up.longitude,
			up.preferred_budget_min, up.preferred_budget_max,
			up.max_distance_km,
			ubm.total_tasks_completed, ubm.total_tasks_accepted,
			ubm.total_tasks_notified, ubm.average_response_time_minutes,
			ubm.completion_rate, ubm.acceptance_rate,
			ubm.reliability_score, ubm.average_review_score,
			ubm.consistency_score
		FROM users u
		JOIN user_profiles up ON up.user_id = u.id
		JOIN user_behavior_metrics ubm ON ubm.user_id = u.id
		WHERE u.role = 'task_doer' AND u.is_active = TRUE AND u.id != $1`,
		excludeUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []DBCandidate
	for rows.Next() {
		var c DBCandidate
		var lat, lng, budMin, budMax sql.NullFloat64
		var maxDist sql.NullInt32

		err := rows.Scan(
			&c.UserID,
			&lat, &lng,
			&budMin, &budMax,
			&maxDist,
			&c.TotalTasksCompleted, &c.TotalTasksAccepted,
			&c.TotalTasksNotified, &c.AvgResponseMinutes,
			&c.CompletionRate, &c.AcceptanceRate,
			&c.ReliabilityScore, &c.AvgReviewScore,
			&c.ConsistencyScore,
		)
		if err != nil {
			return nil, err
		}

		if lat.Valid {
			c.Latitude = &lat.Float64
		}
		if lng.Valid {
			c.Longitude = &lng.Float64
		}
		if budMin.Valid {
			c.PreferredBudgetMin = &budMin.Float64
		}
		if budMax.Valid {
			c.PreferredBudgetMax = &budMax.Float64
		}
		if maxDist.Valid {
			v := int(maxDist.Int32)
			c.MaxDistanceKM = &v
		}

		// Fetch skills for this candidate
		skillRows, err := o.db.QueryContext(ctx,
			`SELECT skill_id, proficiency_level FROM user_skills WHERE user_id = $1`,
			c.UserID,
		)
		if err != nil {
			return nil, err
		}
		for skillRows.Next() {
			var sid uuid.UUID
			var prof int
			if err := skillRows.Scan(&sid, &prof); err != nil {
				skillRows.Close()
				return nil, err
			}
			c.SkillIDs = append(c.SkillIDs, sid)
			c.ProficiencyLevels = append(c.ProficiencyLevels, prof)
		}
		skillRows.Close()

		candidates = append(candidates, c)
	}

	return candidates, rows.Err()
}

func formatScore(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}
