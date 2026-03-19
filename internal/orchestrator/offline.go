package orchestrator

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"xo/internal/notification"
	"xo/internal/relevancy"
)

// OfflineOrchestrator manages the active-queue matching pipeline for offline tasks.
// It replaces the wave-based approach with a persistent matching_queue model:
// score once, notify in batches, pull replacements on decline.
type OfflineOrchestrator struct {
	db       *sql.DB
	notifier notification.Notifier
	signals  *relevancy.SignalUpdater
}

// NewOfflineOrchestrator creates a new OfflineOrchestrator.
func NewOfflineOrchestrator(db *sql.DB, notifier notification.Notifier) *OfflineOrchestrator {
	return &OfflineOrchestrator{
		db:       db,
		notifier: notifier,
		signals:  &relevancy.SignalUpdater{},
	}
}

// --------------------------------------------------------------------------
// ProcessTask — main entry point (score, queue, notify first batch)
// --------------------------------------------------------------------------

// ProcessTask runs the full offline matching pipeline for a task:
// 1. Fetch task, giver profile, and eligible candidates (batch, no N+1)
// 2. Geo-filter with dual radius
// 3. Score via relevancy engine
// 4. Filter below MinScoreOffline
// 5. Populate matching_queue
// 6. Mark top N as active, transition task state, push-notify
func (o *OfflineOrchestrator) ProcessTask(ctx context.Context, taskID uuid.UUID) error {
	task, err := o.fetchTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("fetch task %s: %w", taskID, err)
	}

	giver, err := o.fetchGiverProfile(ctx, task.CreatedBy)
	if err != nil {
		return fmt.Errorf("fetch giver profile for task %s: %w", taskID, err)
	}

	candidates, err := o.fetchCandidatesWithSkills(ctx, task.CreatedBy)
	if err != nil {
		return fmt.Errorf("fetch candidates for task %s: %w", taskID, err)
	}

	if len(candidates) == 0 {
		log.Printf("offline: no candidates for task %s", taskID)
		return nil
	}

	// Geo-filter: dual radius check
	var filtered []relevancy.CandidateInput
	for i := range candidates {
		if relevancy.WithinDualRadius(task, candidates[i]) {
			filtered = append(filtered, candidates[i])
		}
	}

	if len(filtered) == 0 {
		log.Printf("offline: no candidates within radius for task %s", taskID)
		return nil
	}

	// Fetch preference signals for all filtered candidates (batch)
	candidateIDs := make([]uuid.UUID, len(filtered))
	for i := range filtered {
		candidateIDs[i] = filtered[i].UserID
	}
	signalMap, err := o.fetchPreferenceSignals(ctx, candidateIDs, task.CategoryID)
	if err != nil {
		return fmt.Errorf("fetch preference signals for task %s: %w", taskID, err)
	}

	// Attach signals to candidates
	for i := range filtered {
		uid := filtered[i].UserID
		if sigs, ok := signalMap[uid]; ok {
			filtered[i].CategoryAffinity = sigs.CategoryAffinity
			filtered[i].BudgetAcceptAvg = sigs.BudgetAcceptAvg
			filtered[i].BudgetRejectAvg = sigs.BudgetRejectAvg
			filtered[i].IgnoreCount = sigs.IgnoreCount
		}
	}

	// Score and rank
	ranked := relevancy.RankCandidates(task, filtered, giver)

	// Filter below minimum offline score
	var qualifying []relevancy.ScoreBreakdown
	for _, sb := range ranked {
		if sb.FinalScore >= relevancy.MinScoreOffline {
			qualifying = append(qualifying, sb)
		}
	}

	if len(qualifying) == 0 {
		log.Printf("offline: no candidates above score threshold for task %s", taskID)
		return nil
	}

	// Transition: pending -> matching
	if err := o.transitionTask(ctx, nil, taskID, "pending", "matching", nil, nil); err != nil {
		return fmt.Errorf("transition task %s to matching: %w", taskID, err)
	}

	// Populate matching_queue (batch INSERT)
	if err := o.populateQueue(ctx, taskID, qualifying); err != nil {
		return fmt.Errorf("populate queue for task %s: %w", taskID, err)
	}

	// Mark top N as 'active' and transition task
	batchSize := relevancy.BatchSizeForUrgency(task.Urgency)
	if batchSize > len(qualifying) {
		batchSize = len(qualifying)
	}

	// TX2: Notification batch
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin notification tx for task %s: %w", taskID, err)
	}
	defer tx.Rollback()

	// Update top N entries from queued -> active -> notified
	for i := 0; i < batchSize; i++ {
		_, err := tx.ExecContext(ctx,
			`UPDATE matching_queue SET status = 'notified', notified_at = NOW()
			 WHERE task_id = $1 AND user_id = $2 AND status = 'queued'`,
			taskID, qualifying[i].UserID,
		)
		if err != nil {
			return fmt.Errorf("update queue entry to notified: %w", err)
		}

		// Insert task_notification record
		_, err = tx.ExecContext(ctx,
			`INSERT INTO task_notifications (task_id, user_id, wave_number, score, is_exploration, channel, status, sent_at)
			 VALUES ($1, $2, 1, $3, FALSE, 'fcm', 'sent', NOW())`,
			taskID, qualifying[i].UserID, qualifying[i].FinalScore,
		)
		if err != nil {
			return fmt.Errorf("insert task notification: %w", err)
		}
	}

	// Transition: matching -> matched
	if err := o.transitionTask(ctx, tx, taskID, "matching", "matched", nil, nil); err != nil {
		return fmt.Errorf("transition task %s to matched: %w", taskID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit notification tx for task %s: %w", taskID, err)
	}

	// Async: send push notification to each notified candidate
	for i := 0; i < batchSize; i++ {
		if err := o.notifier.Notify(ctx, taskID, []uuid.UUID{qualifying[i].UserID}, 1); err != nil {
			log.Printf("offline: failed to push-notify %s for task %s: %v",
				qualifying[i].UserID, taskID, err)
		}
	}

	return nil
}

// --------------------------------------------------------------------------
// HandleDecline — TX3: Decline + Replacement
// --------------------------------------------------------------------------

// HandleDecline processes a user's decline of a task offer.
// Within a single transaction it:
// 1. Marks the matching_queue entry as declined
// 2. Upserts task_acceptances to rejected
// 3. Records negative preference signals
// 4. Pulls the next candidate from the reserve queue
// 5. Updates user_behavior_metrics
// After commit, sends a push notification to the replacement (if any).
func (o *OfflineOrchestrator) HandleDecline(ctx context.Context, taskID, userID uuid.UUID) error {
	// Fetch task for signal context (budget, category)
	task, err := o.fetchTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("handle decline fetch task %s: %w", taskID, err)
	}

	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("handle decline begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Update matching_queue entry -> declined
	_, err = tx.ExecContext(ctx,
		`UPDATE matching_queue SET status = 'declined', responded_at = NOW()
		 WHERE task_id = $1 AND user_id = $2 AND status IN ('active', 'notified')`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("decline queue entry: %w", err)
	}

	// 2. Upsert task_acceptances -> rejected
	_, err = tx.ExecContext(ctx,
		`INSERT INTO task_acceptances (task_id, user_id, status, responded_at)
		 VALUES ($1, $2, 'rejected', NOW())
		 ON CONFLICT (task_id, user_id) DO UPDATE SET
		   status = 'rejected', responded_at = NOW()`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("upsert task acceptance: %w", err)
	}

	// 3. Record preference signals (decline)
	if err := o.signals.RecordDecline(ctx, tx, userID, task.CategoryID, task.Budget); err != nil {
		return fmt.Errorf("record decline signals: %w", err)
	}

	// 4. Pull next from reserve queue
	var replacementID uuid.UUID
	var hasReplacement bool
	err = tx.QueryRowContext(ctx,
		`SELECT user_id FROM matching_queue
		 WHERE task_id = $1 AND status = 'queued' AND score >= $2
		 ORDER BY position ASC
		 LIMIT 1`,
		taskID, relevancy.MinScoreOffline,
	).Scan(&replacementID)
	if err == nil {
		hasReplacement = true
		// Promote replacement to notified
		_, err = tx.ExecContext(ctx,
			`UPDATE matching_queue SET status = 'notified', notified_at = NOW()
			 WHERE task_id = $1 AND user_id = $2 AND status = 'queued'`,
			taskID, replacementID,
		)
		if err != nil {
			return fmt.Errorf("promote replacement: %w", err)
		}

		// Record notification
		_, err = tx.ExecContext(ctx,
			`INSERT INTO task_notifications (task_id, user_id, wave_number, score, is_exploration, channel, status, sent_at)
			 VALUES ($1, $2, 1, (SELECT score FROM matching_queue WHERE task_id = $1 AND user_id = $2), FALSE, 'fcm', 'sent', NOW())`,
			taskID, replacementID,
		)
		if err != nil {
			return fmt.Errorf("insert replacement notification: %w", err)
		}
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("query reserve queue: %w", err)
	}

	// 5. Update user_behavior_metrics: recalculate acceptance_rate
	_, err = tx.ExecContext(ctx,
		`UPDATE user_behavior_metrics SET
		   acceptance_rate = CASE
		     WHEN total_tasks_notified > 0
		     THEN total_tasks_accepted::numeric / total_tasks_notified::numeric
		     ELSE 0
		   END
		 WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("update behavior metrics on decline: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit decline tx: %w", err)
	}

	// Async: push-notify replacement
	if hasReplacement {
		if err := o.notifier.Notify(ctx, taskID, []uuid.UUID{replacementID}, 1); err != nil {
			log.Printf("offline: failed to notify replacement %s for task %s: %v",
				replacementID, taskID, err)
		}
	}

	return nil
}

// --------------------------------------------------------------------------
// HandleAccept — TX4: Accept
// --------------------------------------------------------------------------

// HandleAccept processes a user's acceptance of a task offer.
// Within a single transaction it:
// 1. Updates task status to in_progress with accepted_by
// 2. Records state transition
// 3. Updates matching_queue: accepter -> accepted, all others -> cancelled
// 4. Updates task_acceptances: accepter -> accepted, others -> expired
// 5. Creates a conversation for chat
// 6. Records positive preference signals
// 7. Updates user_behavior_metrics
func (o *OfflineOrchestrator) HandleAccept(ctx context.Context, taskID, userID uuid.UUID) error {
	task, err := o.fetchTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("handle accept fetch task %s: %w", taskID, err)
	}

	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("handle accept begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Update task status -> in_progress
	_, err = tx.ExecContext(ctx,
		`UPDATE tasks SET status = 'in_progress', accepted_by = $2
		 WHERE id = $1 AND status = 'matched'`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("update task to in_progress: %w", err)
	}

	// 2. State transition
	_, err = tx.ExecContext(ctx,
		`INSERT INTO task_state_transitions (task_id, from_status, to_status, triggered_by)
		 VALUES ($1, 'matched', 'in_progress', $2)`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("insert state transition: %w", err)
	}

	// 3. Update matching_queue: accepter -> accepted, all others -> cancelled
	_, err = tx.ExecContext(ctx,
		`UPDATE matching_queue SET status = 'accepted', responded_at = NOW()
		 WHERE task_id = $1 AND user_id = $2`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("accept queue entry: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE matching_queue SET status = 'cancelled'
		 WHERE task_id = $1 AND user_id != $2 AND status IN ('queued', 'active', 'notified')`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("cancel other queue entries: %w", err)
	}

	// 4. Update task_acceptances: accepter -> accepted, others -> expired
	_, err = tx.ExecContext(ctx,
		`INSERT INTO task_acceptances (task_id, user_id, status, responded_at)
		 VALUES ($1, $2, 'accepted', NOW())
		 ON CONFLICT (task_id, user_id) DO UPDATE SET
		   status = 'accepted', responded_at = NOW()`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("accept task acceptance: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE task_acceptances SET status = 'expired'
		 WHERE task_id = $1 AND user_id != $2 AND status = 'pending'`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("expire other acceptances: %w", err)
	}

	// 5. Create conversation
	_, err = tx.ExecContext(ctx,
		`INSERT INTO conversations (task_id, participant_a, participant_b)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (task_id, participant_a, participant_b) DO NOTHING`,
		taskID, task.CreatedBy, userID,
	)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}

	// 6. Record preference signals (accept)
	if err := o.signals.RecordAccept(ctx, tx, userID, task.CategoryID, task.Budget); err != nil {
		return fmt.Errorf("record accept signals: %w", err)
	}

	// 7. Update user_behavior_metrics
	_, err = tx.ExecContext(ctx,
		`UPDATE user_behavior_metrics SET
		   total_tasks_accepted = total_tasks_accepted + 1,
		   acceptance_rate = CASE
		     WHEN total_tasks_notified > 0
		     THEN (total_tasks_accepted + 1)::numeric / total_tasks_notified::numeric
		     ELSE 0
		   END
		 WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("update behavior metrics on accept: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit accept tx: %w", err)
	}

	return nil
}

// --------------------------------------------------------------------------
// HandleTimeout — TX5: Timeout (10-min offline expiry)
// --------------------------------------------------------------------------

// HandleTimeout processes the expiry of an offline task's acceptance window.
// Within a single transaction it:
// 1. Updates task status to expired
// 2. Records state transition with metadata
// 3. Cancels all queued/active entries, marks notified-no-response as ignored
// 4. Records ignore signals for each ignored user
// 5. Updates giver_behavior_metrics
func (o *OfflineOrchestrator) HandleTimeout(ctx context.Context, taskID uuid.UUID) error {
	task, currentStatus, err := o.fetchTaskWithStatus(ctx, taskID)
	if err != nil {
		return fmt.Errorf("handle timeout fetch task %s: %w", taskID, err)
	}

	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("handle timeout begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Update task -> expired
	_, err = tx.ExecContext(ctx,
		`UPDATE tasks SET status = 'expired'
		 WHERE id = $1 AND status IN ('matching', 'matched')`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("expire task: %w", err)
	}

	// Count statuses for metadata
	var totalNotified, totalDeclined, totalQueued int
	err = tx.QueryRowContext(ctx,
		`SELECT
		   COUNT(*) FILTER (WHERE status = 'notified'),
		   COUNT(*) FILTER (WHERE status = 'declined'),
		   COUNT(*) FILTER (WHERE status = 'queued')
		 FROM matching_queue WHERE task_id = $1`,
		taskID,
	).Scan(&totalNotified, &totalDeclined, &totalQueued)
	if err != nil {
		return fmt.Errorf("count queue statuses: %w", err)
	}

	// 2. State transition with metadata
	metadata := map[string]interface{}{
		"reason":          "timeout",
		"notified_count":  totalNotified,
		"declined_count":  totalDeclined,
		"queued_count":    totalQueued,
		"timeout_minutes": relevancy.TimeoutForUrgency(task.Urgency),
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal timeout metadata: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO task_state_transitions (task_id, from_status, to_status, metadata)
		 VALUES ($1, $2, 'expired', $3)`,
		taskID, currentStatus, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("insert timeout state transition: %w", err)
	}

	// 3. Collect ignored users (notified but no response)
	rows, err := tx.QueryContext(ctx,
		`SELECT user_id FROM matching_queue
		 WHERE task_id = $1 AND status = 'notified'`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("query notified users: %w", err)
	}
	var ignoredUserIDs []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			rows.Close()
			return fmt.Errorf("scan ignored user: %w", err)
		}
		ignoredUserIDs = append(ignoredUserIDs, uid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate ignored users: %w", err)
	}

	// Update queue: queued/active -> cancelled, notified -> ignored
	_, err = tx.ExecContext(ctx,
		`UPDATE matching_queue SET status = 'cancelled'
		 WHERE task_id = $1 AND status IN ('queued', 'active')`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("cancel queued entries: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE matching_queue SET status = 'ignored'
		 WHERE task_id = $1 AND status = 'notified'`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("mark ignored entries: %w", err)
	}

	// 4. Record ignore signals for each ignored user
	for _, uid := range ignoredUserIDs {
		if err := o.signals.RecordIgnore(ctx, tx, uid, task.CategoryID); err != nil {
			return fmt.Errorf("record ignore signal for %s: %w", uid, err)
		}
	}

	// 5. Update giver_behavior_metrics
	_, err = tx.ExecContext(ctx,
		`INSERT INTO giver_behavior_metrics (user_id, total_tasks_expired)
		 VALUES ($1, 1)
		 ON CONFLICT (user_id) DO UPDATE SET
		   total_tasks_expired = giver_behavior_metrics.total_tasks_expired + 1,
		   last_repost_at = NOW()`,
		task.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("update giver metrics on timeout: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit timeout tx: %w", err)
	}

	return nil
}

// --------------------------------------------------------------------------
// Helper: fetchTask
// --------------------------------------------------------------------------

func (o *OfflineOrchestrator) fetchTask(ctx context.Context, taskID uuid.UUID) (relevancy.TaskInput, error) {
	var t relevancy.TaskInput
	var lat, lng sql.NullFloat64
	var categoryID uuid.NullUUID
	var radius sql.NullFloat64

	err := o.db.QueryRowContext(ctx,
		`SELECT id, created_by, budget, latitude, longitude, radius,
		        is_online, urgency, category_id
		 FROM tasks WHERE id = $1`,
		taskID,
	).Scan(&t.ID, &t.CreatedBy, &t.Budget, &lat, &lng, &radius,
		&t.IsOnline, &t.Urgency, &categoryID)
	if err != nil {
		return t, err
	}

	if lat.Valid {
		t.Latitude = &lat.Float64
	}
	if lng.Valid {
		t.Longitude = &lng.Float64
	}
	if radius.Valid {
		t.Radius = radius.Float64
	} else {
		t.Radius = 50.0 // default
	}
	if categoryID.Valid {
		t.CategoryID = &categoryID.UUID
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

// --------------------------------------------------------------------------
// Helper: fetchGiverProfile
// --------------------------------------------------------------------------

func (o *OfflineOrchestrator) fetchGiverProfile(ctx context.Context, giverID uuid.UUID) (relevancy.GiverProfile, error) {
	var g relevancy.GiverProfile
	g.UserID = giverID

	err := o.db.QueryRowContext(ctx,
		`SELECT
		   COALESCE(total_tasks_posted, 0),
		   COALESCE(total_tasks_completed, 0),
		   COALESCE(total_tasks_cancelled, 0),
		   COALESCE(avg_review_from_doers, 0),
		   COALESCE(total_reviews_from_doers, 0),
		   COALESCE(repost_count, 0),
		   last_repost_at
		 FROM giver_behavior_metrics WHERE user_id = $1`,
		giverID,
	).Scan(
		&g.TotalTasksPosted,
		&g.TotalTasksCompleted,
		&g.TotalTasksCancelled,
		&g.AvgReviewFromDoers,
		&g.TotalReviewsFromDoers,
		&g.RepostCount,
		&g.LastRepostAt,
	)
	if err == sql.ErrNoRows {
		// Giver has no metrics yet — return zero-valued profile
		return g, nil
	}
	return g, err
}

// --------------------------------------------------------------------------
// Helper: fetchCandidatesWithSkills (batch, NO N+1)
// --------------------------------------------------------------------------

func (o *OfflineOrchestrator) fetchCandidatesWithSkills(ctx context.Context, excludeUserID uuid.UUID) ([]relevancy.CandidateInput, error) {
	// Step 1: Fetch all eligible candidates
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

	candidateMap := make(map[uuid.UUID]int) // userID -> index in slice
	var candidates []relevancy.CandidateInput

	for rows.Next() {
		var c relevancy.CandidateInput
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
			c.PreferredBudgetMin = budMin.Float64
		}
		if budMax.Valid {
			c.PreferredBudgetMax = budMax.Float64
		}
		if maxDist.Valid {
			c.MaxDistanceKM = int(maxDist.Int32)
		}

		candidateMap[c.UserID] = len(candidates)
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return candidates, nil
	}

	// Step 2: Batch fetch all skills for these candidates
	userIDs := make([]uuid.UUID, len(candidates))
	for i := range candidates {
		userIDs[i] = candidates[i].UserID
	}

	skillRows, err := o.db.QueryContext(ctx,
		`SELECT user_id, skill_id, proficiency_level
		 FROM user_skills
		 WHERE user_id = ANY($1)`,
		uuidArrayParam(userIDs),
	)
	if err != nil {
		return nil, fmt.Errorf("batch fetch skills: %w", err)
	}
	defer skillRows.Close()

	for skillRows.Next() {
		var uid, sid uuid.UUID
		var prof int
		if err := skillRows.Scan(&uid, &sid, &prof); err != nil {
			return nil, err
		}
		if idx, ok := candidateMap[uid]; ok {
			candidates[idx].SkillIDs = append(candidates[idx].SkillIDs, sid)
			candidates[idx].ProficiencyLevels = append(candidates[idx].ProficiencyLevels, prof)
		}
	}

	return candidates, skillRows.Err()
}

// --------------------------------------------------------------------------
// Helper: fetchPreferenceSignals (batch)
// --------------------------------------------------------------------------

// candidateSignals holds the preference signals for a single candidate.
type candidateSignals struct {
	CategoryAffinity *relevancy.PreferenceSignal
	BudgetAcceptAvg  *relevancy.PreferenceSignal
	BudgetRejectAvg  *relevancy.PreferenceSignal
	IgnoreCount      *relevancy.PreferenceSignal
}

func (o *OfflineOrchestrator) fetchPreferenceSignals(ctx context.Context, userIDs []uuid.UUID, categoryID *uuid.UUID) (map[uuid.UUID]*candidateSignals, error) {
	result := make(map[uuid.UUID]*candidateSignals, len(userIDs))

	if len(userIDs) == 0 {
		return result, nil
	}

	rows, err := o.db.QueryContext(ctx,
		`SELECT user_id, signal_type, signal_value, sample_size
		 FROM user_preference_signals
		 WHERE user_id = ANY($1)
		   AND (category_id = $2 OR category_id IS NULL)`,
		uuidArrayParam(userIDs), categoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("query preference signals: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var uid uuid.UUID
		var signalType string
		var signalValue float64
		var sampleSize int

		if err := rows.Scan(&uid, &signalType, &signalValue, &sampleSize); err != nil {
			return nil, err
		}

		sigs, ok := result[uid]
		if !ok {
			sigs = &candidateSignals{}
			result[uid] = sigs
		}

		sig := &relevancy.PreferenceSignal{
			SignalValue: signalValue,
			SampleSize:  sampleSize,
		}

		switch signalType {
		case "category_affinity":
			sigs.CategoryAffinity = sig
		case "budget_accept_avg":
			sigs.BudgetAcceptAvg = sig
		case "budget_reject_avg":
			sigs.BudgetRejectAvg = sig
		case "ignore_count":
			sigs.IgnoreCount = sig
		}
	}

	return result, rows.Err()
}

// --------------------------------------------------------------------------
// Helper: populateQueue (batch INSERT)
// --------------------------------------------------------------------------

func (o *OfflineOrchestrator) populateQueue(ctx context.Context, taskID uuid.UUID, scored []relevancy.ScoreBreakdown) error {
	if len(scored) == 0 {
		return nil
	}

	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO matching_queue (task_id, user_id, score, task_fit, acceptance_likelihood, status, position)
		 VALUES ($1, $2, $3, $4, $5, 'queued', $6)
		 ON CONFLICT (task_id, user_id) DO NOTHING`,
	)
	if err != nil {
		return fmt.Errorf("prepare queue insert: %w", err)
	}
	defer stmt.Close()

	for i, sb := range scored {
		_, err := stmt.ExecContext(ctx,
			taskID, sb.UserID, sb.FinalScore,
			sb.TaskFit, sb.AcceptanceLikelihood, i+1, // position is 1-based
		)
		if err != nil {
			return fmt.Errorf("insert queue entry %d: %w", i, err)
		}
	}

	return tx.Commit()
}

// --------------------------------------------------------------------------
// Helper: transitionTask
// --------------------------------------------------------------------------

// transitionTask records a state transition. If tx is nil, it creates its own transaction.
func (o *OfflineOrchestrator) transitionTask(ctx context.Context, tx *sql.Tx, taskID uuid.UUID, from, to string, triggeredBy *uuid.UUID, metadata []byte) error {
	execer := txOrDB(tx, o.db)

	_, err := execer.ExecContext(ctx,
		`UPDATE tasks SET status = $2 WHERE id = $1`,
		taskID, to,
	)
	if err != nil {
		return err
	}

	_, err = execer.ExecContext(ctx,
		`INSERT INTO task_state_transitions (task_id, from_status, to_status, triggered_by, metadata)
		 VALUES ($1, $2, $3, $4, $5)`,
		taskID, from, to, triggeredBy, metadata,
	)
	return err
}

// --------------------------------------------------------------------------
// Internal utilities
// --------------------------------------------------------------------------

// execContext is a common interface for *sql.DB and *sql.Tx.
type execContext interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func txOrDB(tx *sql.Tx, db *sql.DB) execContext {
	if tx != nil {
		return tx
	}
	return db
}

// uuidArrayParam converts a UUID slice to a lib/pq compatible array parameter.
// lib/pq handles []byte and string arrays, but for UUID arrays we need to format
// them as a PostgreSQL array literal.
func uuidArrayParam(ids []uuid.UUID) interface{} {
	if len(ids) == 0 {
		return "{}"
	}
	// Build PostgreSQL array literal: {uuid1,uuid2,...}
	result := "{"
	for i, id := range ids {
		if i > 0 {
			result += ","
		}
		result += id.String()
	}
	result += "}"
	return result
}

// --------------------------------------------------------------------------
// TaskInput extension for status (used in timeout handler)
// --------------------------------------------------------------------------

// fetchTaskWithStatus loads a task and its current status.
// This is used by HandleTimeout where we need the status for the transition.
func (o *OfflineOrchestrator) fetchTaskWithStatus(ctx context.Context, taskID uuid.UUID) (relevancy.TaskInput, string, error) {
	var t relevancy.TaskInput
	var lat, lng sql.NullFloat64
	var categoryID uuid.NullUUID
	var radius sql.NullFloat64
	var status string
	var createdAt time.Time

	err := o.db.QueryRowContext(ctx,
		`SELECT id, created_by, budget, latitude, longitude, radius,
		        is_online, urgency, category_id, status, created_at
		 FROM tasks WHERE id = $1`,
		taskID,
	).Scan(&t.ID, &t.CreatedBy, &t.Budget, &lat, &lng, &radius,
		&t.IsOnline, &t.Urgency, &categoryID, &status, &createdAt)
	if err != nil {
		return t, "", err
	}

	if lat.Valid {
		t.Latitude = &lat.Float64
	}
	if lng.Valid {
		t.Longitude = &lng.Float64
	}
	if radius.Valid {
		t.Radius = radius.Float64
	} else {
		t.Radius = 50.0
	}
	if categoryID.Valid {
		t.CategoryID = &categoryID.UUID
	}
	t.CreatedAt = createdAt

	return t, status, nil
}
