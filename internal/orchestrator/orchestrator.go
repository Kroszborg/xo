package orchestrator

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	db "xo/pkg/db/db"

	"xo/internal/matching"
)

const (
	priorityDuration = 10 * time.Minute
	waveInterval     = 60 * time.Second
	waveSize         = 15
)

// Orchestrator drives the priority flow for tasks: it scores candidates,
// sends wave-based notifications, and moves tasks to active if no one accepts
// within the priority window.
type Orchestrator struct {
	sqlDB *sql.DB
	q     *db.Queries
	turs  matching.TURSService
}

// New creates an Orchestrator backed by sqlDB and the provided TURSService.
func New(sqlDB *sql.DB, turs matching.TURSService) *Orchestrator {
	return &Orchestrator{
		sqlDB: sqlDB,
		q:     db.New(sqlDB),
		turs:  turs,
	}
}

// StartPriority spawns a goroutine that runs the full priority flow for the
// given task. The parent context controls cancellation.
func (o *Orchestrator) StartPriority(ctx context.Context, taskID uuid.UUID) {
	go o.runPriorityFlow(ctx, taskID)
}

// AcceptTask atomically records a task acceptance inside a transaction.
// It returns an error if the task has already been accepted.
func (o *Orchestrator) AcceptTask(
	ctx context.Context,
	taskID uuid.UUID,
	userID uuid.UUID,
	budget float64,
	responseTimeSec int,
) error {
	tx, err := o.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	qtx := o.q.WithTx(tx)

	task, err := qtx.GetTaskForUpdate(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task for update: %w", err)
	}
	if task.State != "priority" && task.State != "active" {
		err = fmt.Errorf("task %s is not open for acceptance (state=%s)", taskID, task.State)
		return err
	}

	err = qtx.InsertTaskAcceptance(ctx, db.InsertTaskAcceptanceParams{
		TaskID:              uuid.NullUUID{UUID: taskID, Valid: true},
		UserID:              uuid.NullUUID{UUID: userID, Valid: true},
		AcceptedBudget:      fmt.Sprintf("%.2f", budget),
		ResponseTimeSeconds: sql.NullInt32{Int32: int32(responseTimeSec), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("insert task acceptance: %w", err)
	}

	err = qtx.AcceptTaskStateUpdate(ctx, taskID)
	if err != nil {
		return fmt.Errorf("update task state: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// runPriorityFlow executes the full priority-window logic for a single task.
func (o *Orchestrator) runPriorityFlow(parent context.Context, taskID uuid.UUID) {
	ctx, cancel := context.WithTimeout(parent, priorityDuration)
	defer cancel()

	task, err := o.q.GetTaskByID(ctx, taskID)
	if err != nil {
		fmt.Printf("[orchestrator] failed to fetch task %s: %v\n", taskID, err)
		return
	}

	skillIDs, err := o.q.GetTaskRequiredSkills(ctx, taskID)
	if err != nil {
		fmt.Printf("[orchestrator] failed to fetch skills for task %s: %v\n", taskID, err)
		return
	}

	var firstSkillID uuid.UUID
	if len(skillIDs) > 0 {
		firstSkillID = skillIDs[0]
	}

	rows, err := o.q.GetEligibleCandidates(ctx, db.GetEligibleCandidatesParams{
		Mab:     task.Budget,
		SkillID: firstSkillID,
	})
	if err != nil {
		fmt.Printf("[orchestrator] failed to fetch candidates for task %s: %v\n", taskID, err)
		return
	}

	taskInput, err := toTaskInput(task, skillIDs)
	if err != nil {
		fmt.Printf("[orchestrator] failed to convert task %s: %v\n", taskID, err)
		return
	}

	candidates, err := o.toCandidateInputs(ctx, rows)
	if err != nil {
		fmt.Printf("[orchestrator] failed to convert candidates for task %s: %v\n", taskID, err)
		return
	}

	if len(candidates) == 0 {
		fmt.Printf("[orchestrator] no eligible candidates for task %s; moving to active\n", taskID)
		// Use a fresh background context: ctx may already be cancelled or the
		// parent context may have a shorter deadline than the cleanup needs.
		_ = o.q.MoveTaskToActive(context.Background(), taskID)
		return
	}

	ranked := o.turs.RankCandidates(taskInput, candidates)

	waveNum := 1
	for offset := 0; offset < len(ranked); offset += waveSize {
		current, err := o.q.GetTaskByID(ctx, taskID)
		if err == nil && current.State == "accepted" {
			fmt.Printf("[orchestrator] task %s already accepted; stopping waves\n", taskID)
			return
		}

		end := offset + waveSize
		if end > len(ranked) {
			end = len(ranked)
		}
		wave := ranked[offset:end]
		userIDs := make([]uuid.UUID, len(wave))
		for i, rc := range wave {
			userIDs[i] = rc.UserID
		}

		err = o.q.InsertTaskNotificationsBulk(ctx, db.InsertTaskNotificationsBulkParams{
			TaskID:     uuid.NullUUID{UUID: taskID, Valid: true},
			Unnest:     userIDs,
			WaveNumber: sql.NullInt32{Int32: int32(waveNum), Valid: true},
		})
		if err != nil {
			fmt.Printf("[orchestrator] failed to insert notifications (wave %d, task %s): %v\n", waveNum, taskID, err)
		}

		fmt.Printf("[orchestrator] wave %d: notified %d users for task %s\n", waveNum, len(userIDs), taskID)
		waveNum++

		select {
		case <-ctx.Done():
			// Priority window expired; use a fresh context because ctx is
			// already cancelled and DB operations on it would fail immediately.
			fmt.Printf("[orchestrator] priority window expired for task %s; moving to active\n", taskID)
			_ = o.q.MoveTaskToActive(context.Background(), taskID)
			return
		case <-time.After(waveInterval):
		}
	}

	// All ranked candidates have been notified; wait for window to expire.
	select {
	case <-ctx.Done():
		// Same rationale: ctx is cancelled, use a fresh context for cleanup.
		fmt.Printf("[orchestrator] priority window expired for task %s; moving to active\n", taskID)
		_ = o.q.MoveTaskToActive(context.Background(), taskID)
	}
}

// toTaskInput converts a DB Task row and its skill IDs to a matching.TaskInput.
func toTaskInput(t db.Task, skillIDs []uuid.UUID) (matching.TaskInput, error) {
	budget, err := strconv.ParseFloat(t.Budget, 64)
	if err != nil {
		return matching.TaskInput{}, fmt.Errorf("parse budget %q: %w", t.Budget, err)
	}

	var lat, lng *float64
	if t.Lat.Valid {
		v, err := strconv.ParseFloat(t.Lat.String, 64)
		if err != nil {
			return matching.TaskInput{}, fmt.Errorf("parse lat %q: %w", t.Lat.String, err)
		}
		lat = &v
	}
	if t.Lng.Valid {
		v, err := strconv.ParseFloat(t.Lng.String, 64)
		if err != nil {
			return matching.TaskInput{}, fmt.Errorf("parse lng %q: %w", t.Lng.String, err)
		}
		lng = &v
	}

	isOnline := true
	if t.IsOnline.Valid {
		isOnline = t.IsOnline.Bool
	}

	var radiusKM int
	if t.RadiusKm.Valid {
		radiusKM = int(t.RadiusKm.Int32)
	}

	var durationHours int
	if t.DurationHours.Valid {
		durationHours = int(t.DurationHours.Int32)
	}

	var complexityLevel string
	if t.ComplexityLevel.Valid {
		complexityLevel = t.ComplexityLevel.String
	}

	createdAt := time.Time{}
	if t.CreatedAt.Valid {
		createdAt = t.CreatedAt.Time
	}

	return matching.TaskInput{
		ID:              t.ID,
		Budget:          budget,
		CategoryID:      t.CategoryID,
		RequiredSkills:  skillIDs,
		IsOnline:        isOnline,
		Lat:             lat,
		Lng:             lng,
		RadiusKM:        radiusKM,
		DurationHours:   durationHours,
		ComplexityLevel: complexityLevel,
		CreatedAt:       createdAt,
	}, nil
}

// toCandidateInputs converts eligible candidate rows to matching.CandidateInput
// slices, fetching each candidate's skills individually.
func (o *Orchestrator) toCandidateInputs(
	ctx context.Context,
	rows []db.GetEligibleCandidatesRow,
) ([]matching.CandidateInput, error) {
	result := make([]matching.CandidateInput, 0, len(rows))
	for _, r := range rows {
		skills, err := o.q.GetUserSkills(ctx, r.UserID)
		if err != nil {
			return nil, fmt.Errorf("get skills for user %s: %w", r.UserID, err)
		}

		em, err := strconv.ParseFloat(r.ExperienceMultiplier, 64)
		if err != nil {
			return nil, fmt.Errorf("parse experience_multiplier for user %s: %w", r.UserID, err)
		}

		mab, err := strconv.ParseFloat(r.Mab, 64)
		if err != nil {
			return nil, fmt.Errorf("parse mab for user %s: %w", r.UserID, err)
		}

		var lat, lng *float64
		if r.FixedLat.Valid {
			v, err := strconv.ParseFloat(r.FixedLat.String, 64)
			if err != nil {
				return nil, fmt.Errorf("parse fixed_lat for user %s: %w", r.UserID, err)
			}
			lat = &v
		}
		if r.FixedLng.Valid {
			v, err := strconv.ParseFloat(r.FixedLng.String, 64)
			if err != nil {
				return nil, fmt.Errorf("parse fixed_lng for user %s: %w", r.UserID, err)
			}
			lng = &v
		}

		var acceptanceRate float64
		if r.AcceptanceRate.Valid {
			acceptanceRate, err = strconv.ParseFloat(r.AcceptanceRate.String, 64)
			if err != nil {
				return nil, fmt.Errorf("parse acceptance_rate for user %s: %w", r.UserID, err)
			}
		}

		var pushOpenRate float64
		if r.PushOpenRate.Valid {
			pushOpenRate, err = strconv.ParseFloat(r.PushOpenRate.String, 64)
			if err != nil {
				return nil, fmt.Errorf("parse push_open_rate for user %s: %w", r.UserID, err)
			}
		}

		var completionRate float64
		if r.CompletionRate.Valid {
			completionRate, err = strconv.ParseFloat(r.CompletionRate.String, 64)
			if err != nil {
				return nil, fmt.Errorf("parse completion_rate for user %s: %w", r.UserID, err)
			}
		}

		var reliabilityScore float64
		if r.ReliabilityScore.Valid {
			reliabilityScore, err = strconv.ParseFloat(r.ReliabilityScore.String, 64)
			if err != nil {
				return nil, fmt.Errorf("parse reliability_score for user %s: %w", r.UserID, err)
			}
		}

		var medianResponseSec int
		if r.MedianResponseSeconds.Valid {
			medianResponseSec = int(r.MedianResponseSeconds.Int32)
		}

		var totalCompleted int
		if r.TotalTasksCompleted.Valid {
			totalCompleted = int(r.TotalTasksCompleted.Int32)
		}

		result = append(result, matching.CandidateInput{
			UserID:                r.UserID,
			ExperienceLevel:       r.ExperienceLevel,
			ExperienceMultiplier:  em,
			MAB:                   mab,
			RadiusKM:              int(r.RadiusKm),
			FixedLat:              lat,
			FixedLng:              lng,
			Skills:                skills,
			AcceptanceRate:        acceptanceRate,
			MedianResponseSeconds: medianResponseSec,
			PushOpenRate:          pushOpenRate,
			CompletionRate:        completionRate,
			ReliabilityScore:      reliabilityScore,
			TotalTasksCompleted:   totalCompleted,
		})
	}
	return result, nil
}
