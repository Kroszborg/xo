package orchestrator

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	db "xo/pkg/db/db"

	"xo/internal/matching"
	"xo/internal/notification"
)

const (
	priorityDuration   = 10 * time.Minute
	waveInterval       = 60 * time.Second
	waveSize           = 15
	coldStartThreshold = 5  // users with fewer completed tasks are "new"
	explorationPercent = 15 // percent of each wave reserved for new users
)

// Orchestrator drives the priority flow for tasks: it scores candidates,
// sends wave-based notifications (with cold-start exploration slots), and
// manages task lifecycle transitions including completion and EM updates.
type Orchestrator struct {
	sqlDB    *sql.DB
	q        *db.Queries
	turs     matching.TURSService
	notifier notification.Notifier
}

// New creates an Orchestrator backed by sqlDB, the provided TURSService, and a Notifier.
func New(sqlDB *sql.DB, turs matching.TURSService, notifier notification.Notifier) *Orchestrator {
	return &Orchestrator{
		sqlDB:    sqlDB,
		q:        db.New(sqlDB),
		turs:     turs,
		notifier: notifier,
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

// CompleteTask marks a task as completed and updates the accepting user's
// Experience Multiplier using the adaptive EM formula. All writes happen
// inside a single transaction.
func (o *Orchestrator) CompleteTask(ctx context.Context, taskID uuid.UUID) error {
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

	// 1. Transition state to completed.
	err = qtx.CompleteTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("complete task: %w", err)
	}

	// 2. Fetch acceptance record for the budget and user ID.
	acceptance, err := qtx.GetTaskAcceptance(ctx, uuid.NullUUID{UUID: taskID, Valid: true})
	if err != nil {
		return fmt.Errorf("get task acceptance: %w", err)
	}

	userID := acceptance.UserID.UUID

	// 3. Get shown budget from the task.
	task, err := qtx.GetTaskByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	// 4. Get current EM from user profile (locked for update).
	profile, err := qtx.GetUserProfileForUpdate(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user profile: %w", err)
	}

	// 5. Get behavior metrics to determine alpha.
	metrics, err := qtx.GetBehaviorMetrics(ctx, userID)
	if err != nil {
		return fmt.Errorf("get behavior metrics: %w", err)
	}

	// 6. Compute adaptive alpha.
	totalAccepted := 0
	if metrics.TotalTasksAccepted.Valid {
		totalAccepted = int(metrics.TotalTasksAccepted.Int32)
	}
	alpha := adaptiveAlpha(totalAccepted)

	// 7. Calculate new EM.
	acceptedBudget, _ := strconv.ParseFloat(acceptance.AcceptedBudget, 64)
	shownBudget, _ := strconv.ParseFloat(task.Budget, 64)
	oldEM, _ := strconv.ParseFloat(profile.ExperienceMultiplier, 64)

	newEM := oldEM*(1-alpha) + oldEM*(acceptedBudget/shownBudget)*alpha
	newEM = math.Max(0.5, math.Min(newEM, 2.0))

	// 8. Persist EM update.
	err = qtx.UpdateExperienceMultiplier(ctx, db.UpdateExperienceMultiplierParams{
		UserID:               userID,
		ExperienceMultiplier: fmt.Sprintf("%.2f", newEM),
	})
	if err != nil {
		return fmt.Errorf("update experience multiplier: %w", err)
	}

	// 9. Record EM history.
	err = qtx.InsertEMHistory(ctx, db.InsertEMHistoryParams{
		UserID:         uuid.NullUUID{UUID: userID, Valid: true},
		OldMultiplier:  sql.NullString{String: fmt.Sprintf("%.2f", oldEM), Valid: true},
		NewMultiplier:  sql.NullString{String: fmt.Sprintf("%.2f", newEM), Valid: true},
		AcceptedBudget: sql.NullString{String: acceptance.AcceptedBudget, Valid: true},
		ShownBudget:    sql.NullString{String: task.Budget, Valid: true},
		Alpha:          sql.NullString{String: fmt.Sprintf("%.2f", alpha), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("insert em history: %w", err)
	}

	// 10. Increment completed tasks counter.
	err = qtx.IncrementCompletedTasks(ctx, userID)
	if err != nil {
		return fmt.Errorf("increment completed tasks: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// adaptiveAlpha returns the EM learning rate based on the number of tasks
// the user has accepted so far.
func adaptiveAlpha(totalAccepted int) float64 {
	switch {
	case totalAccepted < 5:
		return 0.20
	case totalAccepted < 10:
		return 0.10
	default:
		return 0.05
	}
}

// ---------------------------------------------------------------------------
// Priority Flow
// ---------------------------------------------------------------------------

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

	// Fetch regular eligible candidates.
	regularRows, err := o.q.GetEligibleCandidates(ctx, db.GetEligibleCandidatesParams{
		Mab:    task.Budget,
		TaskID: taskID,
	})
	if err != nil {
		fmt.Printf("[orchestrator] failed to fetch candidates for task %s: %v\n", taskID, err)
		return
	}

	// Fetch cold-start (new user) candidates.
	newUserRows, err := o.q.GetNewUserCandidates(ctx, db.GetNewUserCandidatesParams{
		Mab:                 task.Budget,
		TaskID:              taskID,
		TotalTasksCompleted: sql.NullInt32{Int32: int32(coldStartThreshold), Valid: true},
	})
	if err != nil {
		fmt.Printf("[orchestrator] failed to fetch new user candidates for task %s: %v\n", taskID, err)
		return
	}

	taskInput, err := toTaskInput(task, skillIDs)
	if err != nil {
		fmt.Printf("[orchestrator] failed to convert task %s: %v\n", taskID, err)
		return
	}

	// Build a set of new user IDs for deduplication.
	newUserSet := make(map[uuid.UUID]struct{}, len(newUserRows))
	for _, r := range newUserRows {
		newUserSet[r.UserID] = struct{}{}
	}

	// Convert regular candidates, excluding those already in the new-user set.
	var regularCandidates []matching.CandidateInput
	for _, r := range regularRows {
		if _, isNew := newUserSet[r.UserID]; isNew {
			continue
		}
		c, err := toRegularCandidate(ctx, o.q, r)
		if err != nil {
			fmt.Printf("[orchestrator] skip regular candidate %s: %v\n", r.UserID, err)
			continue
		}
		regularCandidates = append(regularCandidates, c)
	}

	// Convert new user candidates.
	var newCandidates []matching.CandidateInput
	for _, r := range newUserRows {
		c, err := toNewUserCandidate(ctx, o.q, r)
		if err != nil {
			fmt.Printf("[orchestrator] skip new candidate %s: %v\n", r.UserID, err)
			continue
		}
		newCandidates = append(newCandidates, c)
	}

	if len(regularCandidates) == 0 && len(newCandidates) == 0 {
		fmt.Printf("[orchestrator] no eligible candidates for task %s; moving to active\n", taskID)
		_ = o.q.MoveTaskToActive(context.Background(), taskID)
		return
	}

	// Score and rank both pools.
	rankedRegular := o.turs.RankCandidates(taskInput, regularCandidates)
	rankedNew := o.turs.RankCandidates(taskInput, newCandidates)

	// Send waves with exploration slots.
	o.sendWaves(ctx, taskID, rankedRegular, rankedNew)
}

// sendWaves delivers ranked candidates in waves, reserving explorationPercent
// of each wave for new users to ensure cold-start visibility.
func (o *Orchestrator) sendWaves(
	ctx context.Context,
	taskID uuid.UUID,
	regular []matching.RankedCandidate,
	newUsers []matching.RankedCandidate,
) {
	explorationSlots := max(1, waveSize*explorationPercent/100) // at least 1 slot
	regularSlots := waveSize - explorationSlots

	regIdx, newIdx := 0, 0
	waveNum := 1

	for regIdx < len(regular) || newIdx < len(newUsers) {
		// Check if task was already accepted.
		current, err := o.q.GetTaskByID(ctx, taskID)
		if err == nil && current.State == "accepted" {
			fmt.Printf("[orchestrator] task %s accepted; stopping waves\n", taskID)
			return
		}

		var wave []uuid.UUID

		// Fill regular slots.
		for i := 0; i < regularSlots && regIdx < len(regular); i++ {
			wave = append(wave, regular[regIdx].UserID)
			regIdx++
		}

		// Fill exploration slots with new users.
		for i := 0; i < explorationSlots && newIdx < len(newUsers); i++ {
			wave = append(wave, newUsers[newIdx].UserID)
			newIdx++
		}

		// If either pool is exhausted, fill remaining from the other.
		for len(wave) < waveSize && regIdx < len(regular) {
			wave = append(wave, regular[regIdx].UserID)
			regIdx++
		}
		for len(wave) < waveSize && newIdx < len(newUsers) {
			wave = append(wave, newUsers[newIdx].UserID)
			newIdx++
		}

		if len(wave) == 0 {
			break
		}

		// Persist notifications in DB.
		err = o.q.InsertTaskNotificationsBulk(ctx, db.InsertTaskNotificationsBulkParams{
			TaskID:     uuid.NullUUID{UUID: taskID, Valid: true},
			Unnest:     pq.Array(wave),
			WaveNumber: sql.NullInt32{Int32: int32(waveNum), Valid: true},
		})
		if err != nil {
			fmt.Printf("[orchestrator] failed to insert notifications (wave %d, task %s): %v\n", waveNum, taskID, err)
		}

		// Deliver notifications via the Notifier interface.
		if notifyErr := o.notifier.Notify(ctx, taskID, wave, waveNum); notifyErr != nil {
			fmt.Printf("[orchestrator] notification delivery failed (wave %d, task %s): %v\n", waveNum, taskID, notifyErr)
		}

		waveNum++

		select {
		case <-ctx.Done():
			fmt.Printf("[orchestrator] priority window expired for task %s; moving to active\n", taskID)
			_ = o.q.MoveTaskToActive(context.Background(), taskID)
			return
		case <-time.After(waveInterval):
		}
	}

	// All candidates notified; wait for priority window to expire.
	select {
	case <-ctx.Done():
		fmt.Printf("[orchestrator] priority window expired for task %s; moving to active\n", taskID)
		_ = o.q.MoveTaskToActive(context.Background(), taskID)
	}
}

// ---------------------------------------------------------------------------
// Model conversions
// ---------------------------------------------------------------------------

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

// toRegularCandidate converts a GetEligibleCandidatesRow to a CandidateInput.
func toRegularCandidate(ctx context.Context, q *db.Queries, r db.GetEligibleCandidatesRow) (matching.CandidateInput, error) {
	skills, err := q.GetUserSkills(ctx, r.UserID)
	if err != nil {
		return matching.CandidateInput{}, fmt.Errorf("get skills for user %s: %w", r.UserID, err)
	}

	em, _ := strconv.ParseFloat(r.ExperienceMultiplier, 64)
	mab, _ := strconv.ParseFloat(r.Mab, 64)

	c := matching.CandidateInput{
		UserID:               r.UserID,
		ExperienceLevel:      r.ExperienceLevel,
		ExperienceMultiplier: em,
		MAB:                  mab,
		RadiusKM:             int(r.RadiusKm),
		Skills:               skills,
		IsNewUser:            false,
	}

	if r.FixedLat.Valid {
		v, _ := strconv.ParseFloat(r.FixedLat.String, 64)
		c.FixedLat = &v
	}
	if r.FixedLng.Valid {
		v, _ := strconv.ParseFloat(r.FixedLng.String, 64)
		c.FixedLng = &v
	}
	if r.AcceptanceRate.Valid {
		c.AcceptanceRate, _ = strconv.ParseFloat(r.AcceptanceRate.String, 64)
	}
	if r.PushOpenRate.Valid {
		c.PushOpenRate, _ = strconv.ParseFloat(r.PushOpenRate.String, 64)
	}
	if r.CompletionRate.Valid {
		c.CompletionRate, _ = strconv.ParseFloat(r.CompletionRate.String, 64)
	}
	if r.ReliabilityScore.Valid {
		c.ReliabilityScore, _ = strconv.ParseFloat(r.ReliabilityScore.String, 64)
	}
	if r.MedianResponseSeconds.Valid {
		c.MedianResponseSeconds = int(r.MedianResponseSeconds.Int32)
	}
	if r.TotalTasksCompleted.Valid {
		c.TotalTasksCompleted = int(r.TotalTasksCompleted.Int32)
	}

	return c, nil
}

// toNewUserCandidate converts a GetNewUserCandidatesRow to a CandidateInput
// with IsNewUser=true so the scoring engine applies the behavior-intent floor.
func toNewUserCandidate(ctx context.Context, q *db.Queries, r db.GetNewUserCandidatesRow) (matching.CandidateInput, error) {
	skills, err := q.GetUserSkills(ctx, r.UserID)
	if err != nil {
		return matching.CandidateInput{}, fmt.Errorf("get skills for user %s: %w", r.UserID, err)
	}

	em, _ := strconv.ParseFloat(r.ExperienceMultiplier, 64)
	mab, _ := strconv.ParseFloat(r.Mab, 64)

	c := matching.CandidateInput{
		UserID:               r.UserID,
		ExperienceLevel:      r.ExperienceLevel,
		ExperienceMultiplier: em,
		MAB:                  mab,
		RadiusKM:             int(r.RadiusKm),
		Skills:               skills,
		IsNewUser:            true,
	}

	if r.FixedLat.Valid {
		v, _ := strconv.ParseFloat(r.FixedLat.String, 64)
		c.FixedLat = &v
	}
	if r.FixedLng.Valid {
		v, _ := strconv.ParseFloat(r.FixedLng.String, 64)
		c.FixedLng = &v
	}
	if r.AcceptanceRate.Valid {
		c.AcceptanceRate, _ = strconv.ParseFloat(r.AcceptanceRate.String, 64)
	}
	if r.PushOpenRate.Valid {
		c.PushOpenRate, _ = strconv.ParseFloat(r.PushOpenRate.String, 64)
	}
	if r.CompletionRate.Valid {
		c.CompletionRate, _ = strconv.ParseFloat(r.CompletionRate.String, 64)
	}
	if r.ReliabilityScore.Valid {
		c.ReliabilityScore, _ = strconv.ParseFloat(r.ReliabilityScore.String, 64)
	}
	if r.MedianResponseSeconds.Valid {
		c.MedianResponseSeconds = int(r.MedianResponseSeconds.Int32)
	}
	if r.TotalTasksCompleted.Valid {
		c.TotalTasksCompleted = int(r.TotalTasksCompleted.Int32)
	}

	return c, nil
}
