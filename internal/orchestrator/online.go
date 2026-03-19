package orchestrator

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"xo/internal/relevancy"
)

// OnlineOrchestrator handles the materialization and lifecycle of online task
// relevancy scores. Unlike the offline orchestrator, online tasks do not push
// notifications — they populate a scored candidate list that both givers and
// doers can browse via paginated queries.
type OnlineOrchestrator struct {
	db *sql.DB
}

// NewOnlineOrchestrator creates a new OnlineOrchestrator.
func NewOnlineOrchestrator(db *sql.DB) *OnlineOrchestrator {
	return &OnlineOrchestrator{db: db}
}

// --------------------------------------------------------------------------
// ProcessTask — materialize scores for an online task
// --------------------------------------------------------------------------

// ProcessTask scores all eligible task doers for the given online task and
// inserts qualifying scores into the relevancy_scores table. It transitions
// the task from pending to active and sets a 48-hour TTL.
func (o *OnlineOrchestrator) ProcessTask(ctx context.Context, taskID uuid.UUID) error {
	// 1. Fetch task
	task, err := o.fetchTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("fetch task %s: %w", taskID, err)
	}

	// 2. Fetch giver profile
	giver, err := o.fetchGiverProfile(ctx, task.CreatedBy)
	if err != nil {
		return fmt.Errorf("fetch giver profile for task %s: %w", taskID, err)
	}

	// 3. Fetch ALL eligible task doers (no geo filter for online)
	candidates, err := o.fetchOnlineCandidates(ctx, task.CreatedBy)
	if err != nil {
		return fmt.Errorf("fetch candidates for task %s: %w", taskID, err)
	}

	if len(candidates) == 0 {
		log.Printf("online: no candidates for task %s", taskID)
		return nil
	}

	// 4. Batch-fetch skills for all candidates
	if err := o.batchLoadSkills(ctx, candidates); err != nil {
		return fmt.Errorf("batch load skills for task %s: %w", taskID, err)
	}

	// 5. Batch-fetch preference signals for all candidates
	if err := o.batchLoadPreferenceSignals(ctx, candidates, task.CategoryID); err != nil {
		return fmt.Errorf("batch load preference signals for task %s: %w", taskID, err)
	}

	// 6. Convert DB types to scoring types and score
	taskInput := dbTaskToRelevancy(task)
	candidateInputs := make([]relevancy.CandidateInput, len(candidates))
	for i := range candidates {
		candidateInputs[i] = dbCandidateToRelevancy(candidates[i])
	}
	giverInput := dbGiverToRelevancy(giver)

	ranked := relevancy.RankCandidates(taskInput, candidateInputs, giverInput)

	// 7. Filter below MinScoreOnline and batch INSERT into relevancy_scores
	insertCount, err := o.insertRelevancyScores(ctx, taskID, ranked)
	if err != nil {
		return fmt.Errorf("insert relevancy scores for task %s: %w", taskID, err)
	}

	log.Printf("online: task %s scored %d candidates, inserted %d above threshold %.0f",
		taskID, len(ranked), insertCount, relevancy.MinScoreOnline)

	// 8. Transition task: pending -> active
	if err := o.transitionTask(ctx, taskID, "pending", "active", nil); err != nil {
		return fmt.Errorf("transition task %s to active: %w", taskID, err)
	}

	// 9. Set expires_at = NOW() + OnlineTTLHours
	if err := o.setExpiry(ctx, taskID, time.Duration(relevancy.OnlineTTLHours)*time.Hour); err != nil {
		return fmt.Errorf("set expiry for task %s: %w", taskID, err)
	}

	return nil
}

// --------------------------------------------------------------------------
// HandleAccept — online task accepted
// --------------------------------------------------------------------------

// HandleAccept processes an acceptance of an online task. It transitions the
// task to in_progress, records the acceptance, creates a conversation, updates
// behavior metrics, and removes the materialized relevancy scores.
func (o *OnlineOrchestrator) HandleAccept(ctx context.Context, taskID, userID uuid.UUID) error {
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Update task -> in_progress, set accepted_by
	_, err = tx.ExecContext(ctx,
		`UPDATE tasks SET status = 'in_progress', accepted_by = $2 WHERE id = $1`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	// 2. Insert task_state_transition
	_, err = tx.ExecContext(ctx,
		`INSERT INTO task_state_transitions (task_id, from_status, to_status, triggered_by)
		 VALUES ($1, 'active', 'in_progress', $2)`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("insert state transition: %w", err)
	}

	// 3. Insert task_acceptances
	_, err = tx.ExecContext(ctx,
		`INSERT INTO task_acceptances (task_id, user_id, status, responded_at)
		 VALUES ($1, $2, 'accepted', NOW())
		 ON CONFLICT (task_id, user_id) DO UPDATE SET status = 'accepted', responded_at = NOW()`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("insert task acceptance: %w", err)
	}

	// 4. Create conversation between giver and doer
	var giverID uuid.UUID
	err = tx.QueryRowContext(ctx,
		`SELECT created_by FROM tasks WHERE id = $1`, taskID,
	).Scan(&giverID)
	if err != nil {
		return fmt.Errorf("fetch task giver: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO conversations (task_id, participant_a, participant_b)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (task_id, participant_a, participant_b) DO NOTHING`,
		taskID, giverID, userID,
	)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}

	// 5. Update user_behavior_metrics (total_tasks_accepted++, acceptance_rate)
	_, err = tx.ExecContext(ctx,
		`UPDATE user_behavior_metrics
		 SET total_tasks_accepted = total_tasks_accepted + 1,
		     acceptance_rate = CASE
		         WHEN total_tasks_notified > 0
		         THEN (total_tasks_accepted + 1)::NUMERIC / total_tasks_notified
		         ELSE acceptance_rate
		     END
		 WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("update behavior metrics: %w", err)
	}

	// 6. Delete all materialized relevancy scores for this task
	_, err = tx.ExecContext(ctx,
		`DELETE FROM relevancy_scores WHERE task_id = $1`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("delete relevancy scores: %w", err)
	}

	return tx.Commit()
}

// --------------------------------------------------------------------------
// HandleExpiry — TTL expired
// --------------------------------------------------------------------------

// HandleExpiry processes an online task whose TTL has expired. It transitions
// the task to expired, removes materialized scores, and updates giver metrics.
func (o *OnlineOrchestrator) HandleExpiry(ctx context.Context, taskID uuid.UUID) error {
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Update task -> expired
	_, err = tx.ExecContext(ctx,
		`UPDATE tasks SET status = 'expired' WHERE id = $1`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	// 2. Insert task_state_transition
	_, err = tx.ExecContext(ctx,
		`INSERT INTO task_state_transitions (task_id, from_status, to_status, metadata)
		 VALUES ($1, 'active', 'expired', '{"reason": "ttl_expired"}')`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("insert state transition: %w", err)
	}

	// 3. Delete materialized relevancy scores
	_, err = tx.ExecContext(ctx,
		`DELETE FROM relevancy_scores WHERE task_id = $1`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("delete relevancy scores: %w", err)
	}

	// 4. Update giver_behavior_metrics (total_tasks_expired++)
	_, err = tx.ExecContext(ctx,
		`UPDATE giver_behavior_metrics
		 SET total_tasks_expired = total_tasks_expired + 1
		 WHERE user_id = (SELECT created_by FROM tasks WHERE id = $1)`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("update giver behavior metrics: %w", err)
	}

	return tx.Commit()
}

// --------------------------------------------------------------------------
// GetCandidatesForTask — Direction A: giver views candidates
// --------------------------------------------------------------------------

// RelevancyRow represents a single row from the relevancy_scores table.
type RelevancyRow struct {
	UserID               uuid.UUID `json:"user_id"`
	TaskID               uuid.UUID `json:"task_id"`
	TaskFit              float64   `json:"task_fit"`
	AcceptanceLikelihood float64   `json:"acceptance_likelihood"`
	ColdStartMultiplier  float64   `json:"cold_start_multiplier"`
	FinalScore           float64   `json:"final_score"`
}

// GetCandidatesForTask returns a paginated list of scored candidates for a
// given task, ordered by final_score descending. This is "Direction A" — the
// task giver browses ranked candidates.
func (o *OnlineOrchestrator) GetCandidatesForTask(ctx context.Context, taskID uuid.UUID, limit, offset int) ([]RelevancyRow, error) {
	rows, err := o.db.QueryContext(ctx,
		`SELECT user_id, task_fit, acceptance_likelihood, cold_start_multiplier, final_score
		 FROM relevancy_scores
		 WHERE task_id = $1
		 ORDER BY final_score DESC
		 LIMIT $2 OFFSET $3`,
		taskID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query candidates for task %s: %w", taskID, err)
	}
	defer rows.Close()

	var results []RelevancyRow
	for rows.Next() {
		var r RelevancyRow
		r.TaskID = taskID
		if err := rows.Scan(&r.UserID, &r.TaskFit, &r.AcceptanceLikelihood, &r.ColdStartMultiplier, &r.FinalScore); err != nil {
			return nil, fmt.Errorf("scan candidate row: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// --------------------------------------------------------------------------
// GetTasksForUser — Direction B: doer views task feed
// --------------------------------------------------------------------------

// GetTasksForUser returns a paginated list of scored tasks for a given user,
// ordered by final_score descending. This is "Direction B" — the task doer
// browses their personalized task feed.
func (o *OnlineOrchestrator) GetTasksForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]RelevancyRow, error) {
	rows, err := o.db.QueryContext(ctx,
		`SELECT rs.task_id, rs.task_fit, rs.acceptance_likelihood, rs.cold_start_multiplier, rs.final_score
		 FROM relevancy_scores rs
		 JOIN tasks t ON t.id = rs.task_id
		 WHERE rs.user_id = $1 AND t.status = 'active' AND t.is_online = true
		 ORDER BY rs.final_score DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query tasks for user %s: %w", userID, err)
	}
	defer rows.Close()

	var results []RelevancyRow
	for rows.Next() {
		var r RelevancyRow
		r.UserID = userID
		if err := rows.Scan(&r.TaskID, &r.TaskFit, &r.AcceptanceLikelihood, &r.ColdStartMultiplier, &r.FinalScore); err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// --------------------------------------------------------------------------
// Internal DB helpers
// --------------------------------------------------------------------------

// dbOnlineCandidate holds all candidate data fetched from DB before conversion.
type dbOnlineCandidate struct {
	UserID               uuid.UUID
	Latitude             *float64
	Longitude            *float64
	PreferredBudgetMin   *float64
	PreferredBudgetMax   *float64
	MaxDistanceKM        *int
	TotalTasksCompleted  int
	TotalTasksAccepted   int
	TotalTasksNotified   int
	TotalReviewsReceived int
	AvgResponseMinutes   float64
	CompletionRate       float64
	AcceptanceRate       float64
	ReliabilityScore     float64
	AvgReviewScore       float64
	ConsistencyScore     float64
	SkillIDs             []uuid.UUID
	ProficiencyLevels    []int

	// Category-specific (populated after batch load)
	CategoryTasksCompleted int
	CategoryCompletionRate float64

	// Preference signals (populated after batch load)
	CategoryAffinity    *relevancy.PreferenceSignal
	BudgetAcceptAvg     *relevancy.PreferenceSignal
	BudgetRejectAvg     *relevancy.PreferenceSignal
	IgnoreCount         *relevancy.PreferenceSignal
	SimilarTaskAccepted int
	SimilarTaskRejected int
}

// dbOnlineTask holds task data fetched from DB.
type dbOnlineTask struct {
	ID               uuid.UUID
	CreatedBy        uuid.UUID
	CategoryID       *uuid.UUID
	Budget           float64
	Latitude         *float64
	Longitude        *float64
	Radius           float64
	IsOnline         bool
	Urgency          string
	RequiredSkillIDs []uuid.UUID
	MinProficiency   []int
}

// dbOnlineGiver holds giver profile data.
type dbOnlineGiver struct {
	UserID                uuid.UUID
	AvgReviewFromDoers    float64
	TotalReviewsFromDoers int
	TotalTasksPosted      int
	TotalTasksCompleted   int
	TotalTasksCancelled   int
	RepostCount           int
	LastRepostAt          *time.Time
}

func (o *OnlineOrchestrator) fetchTask(ctx context.Context, taskID uuid.UUID) (dbOnlineTask, error) {
	var t dbOnlineTask
	var lat, lng sql.NullFloat64
	var categoryID sql.NullString

	err := o.db.QueryRowContext(ctx,
		`SELECT id, created_by, category_id, budget, latitude, longitude, radius, is_online, urgency
		 FROM tasks WHERE id = $1`,
		taskID,
	).Scan(&t.ID, &t.CreatedBy, &categoryID, &t.Budget, &lat, &lng, &t.Radius, &t.IsOnline, &t.Urgency)
	if err != nil {
		return t, err
	}

	if lat.Valid {
		t.Latitude = &lat.Float64
	}
	if lng.Valid {
		t.Longitude = &lng.Float64
	}
	if categoryID.Valid {
		parsed, err := uuid.Parse(categoryID.String)
		if err == nil {
			t.CategoryID = &parsed
		}
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

func (o *OnlineOrchestrator) fetchGiverProfile(ctx context.Context, userID uuid.UUID) (dbOnlineGiver, error) {
	var g dbOnlineGiver
	g.UserID = userID

	var lastRepost sql.NullTime
	err := o.db.QueryRowContext(ctx,
		`SELECT total_tasks_posted, total_tasks_completed, total_tasks_cancelled,
		        total_tasks_expired, avg_review_from_doers, total_reviews_from_doers,
		        repost_count, last_repost_at
		 FROM giver_behavior_metrics WHERE user_id = $1`,
		userID,
	).Scan(
		&g.TotalTasksPosted, &g.TotalTasksCompleted, &g.TotalTasksCancelled,
		new(int), // total_tasks_expired — not stored in struct, consumed and discarded
		&g.AvgReviewFromDoers, &g.TotalReviewsFromDoers,
		&g.RepostCount, &lastRepost,
	)
	if err == sql.ErrNoRows {
		// Giver has no metrics row yet — return zero-value profile
		return g, nil
	}
	if err != nil {
		return g, err
	}
	if lastRepost.Valid {
		g.LastRepostAt = &lastRepost.Time
	}
	return g, nil
}

// fetchOnlineCandidates loads all eligible task doers (no geo filter).
func (o *OnlineOrchestrator) fetchOnlineCandidates(ctx context.Context, excludeUserID uuid.UUID) ([]*dbOnlineCandidate, error) {
	rows, err := o.db.QueryContext(ctx,
		`SELECT
			u.id,
			up.latitude, up.longitude,
			up.preferred_budget_min, up.preferred_budget_max,
			up.max_distance_km,
			ubm.total_tasks_completed, ubm.total_tasks_accepted,
			ubm.total_tasks_notified, ubm.total_reviews_received,
			ubm.average_response_time_minutes,
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

	var candidates []*dbOnlineCandidate
	for rows.Next() {
		c := &dbOnlineCandidate{}
		var lat, lng, budMin, budMax sql.NullFloat64
		var maxDist sql.NullInt32

		err := rows.Scan(
			&c.UserID,
			&lat, &lng,
			&budMin, &budMax,
			&maxDist,
			&c.TotalTasksCompleted, &c.TotalTasksAccepted,
			&c.TotalTasksNotified, &c.TotalReviewsReceived,
			&c.AvgResponseMinutes,
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

		candidates = append(candidates, c)
	}

	return candidates, rows.Err()
}

// batchLoadSkills fetches skills for all candidates in a single query using
// WHERE user_id = ANY($1), avoiding N+1 queries.
func (o *OnlineOrchestrator) batchLoadSkills(ctx context.Context, candidates []*dbOnlineCandidate) error {
	if len(candidates) == 0 {
		return nil
	}

	userIDs := make([]uuid.UUID, len(candidates))
	userMap := make(map[uuid.UUID]*dbOnlineCandidate, len(candidates))
	for i, c := range candidates {
		userIDs[i] = c.UserID
		userMap[c.UserID] = c
	}

	rows, err := o.db.QueryContext(ctx,
		`SELECT user_id, skill_id, proficiency_level
		 FROM user_skills
		 WHERE user_id = ANY($1)`,
		uuidSliceToArray(userIDs),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var uid, sid uuid.UUID
		var prof int
		if err := rows.Scan(&uid, &sid, &prof); err != nil {
			return err
		}
		if c, ok := userMap[uid]; ok {
			c.SkillIDs = append(c.SkillIDs, sid)
			c.ProficiencyLevels = append(c.ProficiencyLevels, prof)
		}
	}

	return rows.Err()
}

// batchLoadPreferenceSignals fetches preference signals for all candidates
// in a single query. It also loads category-specific task completion metrics
// and similar task history.
func (o *OnlineOrchestrator) batchLoadPreferenceSignals(ctx context.Context, candidates []*dbOnlineCandidate, categoryID *uuid.UUID) error {
	if len(candidates) == 0 {
		return nil
	}

	userIDs := make([]uuid.UUID, len(candidates))
	userMap := make(map[uuid.UUID]*dbOnlineCandidate, len(candidates))
	for i, c := range candidates {
		userIDs[i] = c.UserID
		userMap[c.UserID] = c
	}

	// Load preference signals
	rows, err := o.db.QueryContext(ctx,
		`SELECT user_id, signal_type, signal_value, sample_size
		 FROM user_preference_signals
		 WHERE user_id = ANY($1) AND (category_id = $2 OR category_id IS NULL)`,
		uuidSliceToArray(userIDs), categoryID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var uid uuid.UUID
		var signalType string
		var signalValue float64
		var sampleSize int

		if err := rows.Scan(&uid, &signalType, &signalValue, &sampleSize); err != nil {
			return err
		}

		c, ok := userMap[uid]
		if !ok {
			continue
		}

		sig := &relevancy.PreferenceSignal{SignalValue: signalValue, SampleSize: sampleSize}
		switch signalType {
		case "category_affinity":
			c.CategoryAffinity = sig
		case "budget_accept_avg":
			c.BudgetAcceptAvg = sig
		case "budget_reject_avg":
			c.BudgetRejectAvg = sig
		case "ignore_count":
			c.IgnoreCount = sig
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Load category-specific completion metrics if a category is specified
	if categoryID != nil {
		catRows, err := o.db.QueryContext(ctx,
			`SELECT ta.user_id,
			        COUNT(*) FILTER (WHERE t.status = 'completed') as completed,
			        COUNT(*) as total
			 FROM task_acceptances ta
			 JOIN tasks t ON t.id = ta.task_id
			 WHERE ta.user_id = ANY($1) AND t.category_id = $2 AND ta.status = 'accepted'
			 GROUP BY ta.user_id`,
			uuidSliceToArray(userIDs), categoryID,
		)
		if err != nil {
			return err
		}
		defer catRows.Close()

		for catRows.Next() {
			var uid uuid.UUID
			var completed, total int
			if err := catRows.Scan(&uid, &completed, &total); err != nil {
				return err
			}
			if c, ok := userMap[uid]; ok {
				c.CategoryTasksCompleted = completed
				if total > 0 {
					c.CategoryCompletionRate = float64(completed) / float64(total)
				}
			}
		}
		if err := catRows.Err(); err != nil {
			return err
		}
	}

	return nil
}

// insertRelevancyScores batch-inserts scored candidates into relevancy_scores,
// filtering those below MinScoreOnline. Returns the count of inserted rows.
func (o *OnlineOrchestrator) insertRelevancyScores(ctx context.Context, taskID uuid.UUID, ranked []relevancy.ScoreBreakdown) (int, error) {
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO relevancy_scores (task_id, user_id, task_fit, acceptance_likelihood, cold_start_multiplier, final_score)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
	)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	count := 0
	for _, r := range ranked {
		if r.FinalScore < relevancy.MinScoreOnline {
			continue
		}
		_, err := stmt.ExecContext(ctx, taskID, r.UserID, r.TaskFit, r.AcceptanceLikelihood, r.ColdStartMultiplier, r.FinalScore)
		if err != nil {
			return count, err
		}
		count++
	}

	return count, tx.Commit()
}

// transitionTask records a state transition within a transaction.
func (o *OnlineOrchestrator) transitionTask(ctx context.Context, taskID uuid.UUID, from, to string, triggeredBy *uuid.UUID) error {
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

// setExpiry sets the expires_at field on a task to NOW() + duration.
func (o *OnlineOrchestrator) setExpiry(ctx context.Context, taskID uuid.UUID, ttl time.Duration) error {
	_, err := o.db.ExecContext(ctx,
		`UPDATE tasks SET expires_at = NOW() + $2 * INTERVAL '1 second' WHERE id = $1`,
		taskID, int(ttl.Seconds()),
	)
	return err
}

// --------------------------------------------------------------------------
// Type converters (DB types -> relevancy scoring types)
// --------------------------------------------------------------------------

func dbTaskToRelevancy(t dbOnlineTask) relevancy.TaskInput {
	return relevancy.TaskInput{
		ID:               t.ID,
		CreatedBy:        t.CreatedBy,
		CategoryID:       t.CategoryID,
		RequiredSkillIDs: t.RequiredSkillIDs,
		MinProficiency:   t.MinProficiency,
		Budget:           t.Budget,
		Latitude:         t.Latitude,
		Longitude:        t.Longitude,
		Radius:           t.Radius,
		IsOnline:         t.IsOnline,
		Urgency:          t.Urgency,
	}
}

func dbCandidateToRelevancy(c *dbOnlineCandidate) relevancy.CandidateInput {
	ci := relevancy.CandidateInput{
		UserID:                 c.UserID,
		SkillIDs:               c.SkillIDs,
		ProficiencyLevels:      c.ProficiencyLevels,
		TotalTasksCompleted:    c.TotalTasksCompleted,
		TotalTasksAccepted:     c.TotalTasksAccepted,
		TotalTasksNotified:     c.TotalTasksNotified,
		TotalReviewsReceived:   c.TotalReviewsReceived,
		AvgResponseMinutes:     c.AvgResponseMinutes,
		CompletionRate:         c.CompletionRate,
		AcceptanceRate:         c.AcceptanceRate,
		ReliabilityScore:       c.ReliabilityScore,
		AvgReviewScore:         c.AvgReviewScore,
		ConsistencyScore:       c.ConsistencyScore,
		CategoryTasksCompleted: c.CategoryTasksCompleted,
		CategoryCompletionRate: c.CategoryCompletionRate,
		CategoryAffinity:       c.CategoryAffinity,
		BudgetAcceptAvg:        c.BudgetAcceptAvg,
		BudgetRejectAvg:        c.BudgetRejectAvg,
		IgnoreCount:            c.IgnoreCount,
		SimilarTaskAccepted:    c.SimilarTaskAccepted,
		SimilarTaskRejected:    c.SimilarTaskRejected,
	}
	if c.Latitude != nil {
		ci.Latitude = c.Latitude
	}
	if c.Longitude != nil {
		ci.Longitude = c.Longitude
	}
	if c.PreferredBudgetMin != nil {
		ci.PreferredBudgetMin = *c.PreferredBudgetMin
	}
	if c.PreferredBudgetMax != nil {
		ci.PreferredBudgetMax = *c.PreferredBudgetMax
	}
	if c.MaxDistanceKM != nil {
		ci.MaxDistanceKM = *c.MaxDistanceKM
	}
	return ci
}

func dbGiverToRelevancy(g dbOnlineGiver) relevancy.GiverProfile {
	return relevancy.GiverProfile{
		UserID:                g.UserID,
		AvgReviewFromDoers:    g.AvgReviewFromDoers,
		TotalReviewsFromDoers: g.TotalReviewsFromDoers,
		TotalTasksPosted:      g.TotalTasksPosted,
		TotalTasksCompleted:   g.TotalTasksCompleted,
		TotalTasksCancelled:   g.TotalTasksCancelled,
		RepostCount:           g.RepostCount,
		LastRepostAt:          g.LastRepostAt,
	}
}

// uuidSliceToArray converts a []uuid.UUID to a lib/pq-compatible array
// representation. Uses a simple string concatenation approach compatible
// with PostgreSQL's UUID array syntax.
func uuidSliceToArray(ids []uuid.UUID) interface{} {
	// lib/pq supports passing arrays as string format: {uuid1,uuid2,...}
	if len(ids) == 0 {
		return "{}"
	}
	s := "{"
	for i, id := range ids {
		if i > 0 {
			s += ","
		}
		s += id.String()
	}
	s += "}"
	return s
}
