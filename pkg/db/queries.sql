-- ============================================================
-- USERS
-- ============================================================

-- name: CreateUser :one
INSERT INTO users (email, phone, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: UpdateUserStatus :exec
UPDATE users
SET status = $2
WHERE id = $1;


-- ============================================================
-- USER PROFILE
-- ============================================================

-- name: CreateUserProfile :exec
INSERT INTO user_profiles (
    user_id,
    experience_level,
    experience_multiplier,
    mab,
    radius_km,
    fixed_lat,
    fixed_lng,
    timezone,
    language,
    last_active_at
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW());

-- name: GetUserProfileForUpdate :one
SELECT *
FROM user_profiles
WHERE user_id = $1
FOR UPDATE;

-- name: UpdateExperienceMultiplier :exec
UPDATE user_profiles
SET experience_multiplier = $2,
    updated_at = NOW()
WHERE user_id = $1;

-- name: UpdateLastActive :exec
UPDATE user_profiles
SET last_active_at = NOW()
WHERE user_id = $1;


-- ============================================================
-- USER SKILLS
-- ============================================================

-- name: AddUserSkill :exec
INSERT INTO user_skills (user_id, skill_id, is_primary)
VALUES ($1,$2,$3)
ON CONFLICT DO NOTHING;

-- name: GetUserSkills :many
SELECT skill_id
FROM user_skills
WHERE user_id = $1;


-- ============================================================
-- USER BEHAVIOR METRICS
-- ============================================================

-- name: InitUserBehaviorMetrics :exec
INSERT INTO user_behavior_metrics (user_id)
VALUES ($1);

-- name: GetBehaviorMetrics :one
SELECT *
FROM user_behavior_metrics
WHERE user_id = $1;

-- name: IncrementAcceptedTasks :exec
UPDATE user_behavior_metrics
SET total_tasks_accepted = total_tasks_accepted + 1,
    updated_at = NOW()
WHERE user_id = $1;

-- name: IncrementCompletedTasks :exec
UPDATE user_behavior_metrics
SET total_tasks_completed = total_tasks_completed + 1,
    updated_at = NOW()
WHERE user_id = $1;

-- name: UpdateAcceptanceRate :exec
UPDATE user_behavior_metrics
SET acceptance_rate = $2,
    updated_at = NOW()
WHERE user_id = $1;


-- ============================================================
-- EXPERIENCE MULTIPLIER HISTORY
-- ============================================================

-- name: InsertEMHistory :exec
INSERT INTO experience_multiplier_history (
    user_id,
    old_multiplier,
    new_multiplier,
    accepted_budget,
    shown_budget,
    alpha
)
VALUES ($1,$2,$3,$4,$5,$6);


-- ============================================================
-- TASKS
-- ============================================================

-- name: CreateTaskPriority :one
INSERT INTO tasks (
    task_giver_id,
    category_id,
    budget,
    duration_hours,
    complexity_level,
    is_online,
    lat,
    lng,
    radius_km,
    state,
    priority_started_at,
    expires_at
)
VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,
    'priority',
    NOW(),
    NOW() + INTERVAL '24 hours'
)
RETURNING *;

-- name: GetTaskByID :one
SELECT *
FROM tasks
WHERE id = $1;

-- name: GetTaskForUpdate :one
SELECT *
FROM tasks
WHERE id = $1
FOR UPDATE;

-- name: MoveTaskToActive :exec
UPDATE tasks
SET state = 'active',
    active_started_at = NOW()
WHERE id = $1
AND state = 'priority';

-- name: AcceptTaskStateUpdate :exec
UPDATE tasks
SET state = 'accepted',
    accepted_at = NOW()
WHERE id = $1;

-- name: CompleteTask :exec
UPDATE tasks
SET state = 'completed',
    completed_at = NOW()
WHERE id = $1
AND state = 'accepted';

-- name: CancelTask :execrows
UPDATE tasks
SET state = 'cancelled'
WHERE id = $1
AND state IN ('priority', 'active');

-- name: UpdateTask :one
UPDATE tasks
SET budget = COALESCE($2, budget),
    duration_hours = COALESCE($3, duration_hours),
    complexity_level = COALESCE($4, complexity_level),
    is_online = COALESCE($5, is_online),
    lat = COALESCE($6, lat),
    lng = COALESCE($7, lng),
    radius_km = COALESCE($8, radius_km)
WHERE id = $1
AND state = 'active'
RETURNING *;

-- name: ListTasks :many
SELECT *
FROM tasks
WHERE (sqlc.narg('state')::text IS NULL OR state = sqlc.narg('state')::text)
  AND (sqlc.narg('category_id')::uuid IS NULL OR category_id = sqlc.narg('category_id')::uuid)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit_val') OFFSET sqlc.arg('offset_val');

-- name: ExpireTasks :exec
UPDATE tasks
SET state = 'expired'
WHERE state IN ('priority','active')
AND expires_at <= NOW();


-- ============================================================
-- TASK REQUIRED SKILLS
-- ============================================================

-- name: AddTaskRequiredSkill :exec
INSERT INTO task_required_skills (task_id, skill_id, is_core)
VALUES ($1,$2,$3);

-- name: GetTaskRequiredSkills :many
SELECT skill_id
FROM task_required_skills
WHERE task_id = $1;


-- ============================================================
-- HARD FILTER CANDIDATE QUERY (LIB/PQ SAFE)
-- ============================================================

-- $1 = task_budget (numeric)
-- $2 = task_id (uuid)

-- name: GetEligibleCandidates :many
SELECT
    u.id AS user_id,
    up.experience_level,
    up.experience_multiplier,
    up.mab,
    up.radius_km,
    up.fixed_lat,
    up.fixed_lng,
    ubm.acceptance_rate,
    ubm.median_response_seconds,
    ubm.push_open_rate,
    ubm.completion_rate,
    ubm.reliability_score,
    ubm.total_tasks_completed
FROM users u
JOIN user_profiles up ON up.user_id = u.id
JOIN user_behavior_metrics ubm ON ubm.user_id = u.id
WHERE
    u.status = 'active'
    AND up.last_active_at >= NOW() - INTERVAL '7 days'
    AND $1 >= (0.5 * up.mab)
    AND EXISTS (
        SELECT 1
        FROM user_skills us
        JOIN task_required_skills trs ON trs.skill_id = us.skill_id AND trs.task_id = $2
        WHERE us.user_id = u.id
    );


-- ============================================================
-- TASK ACCEPTANCE (RACE SAFE)
-- ============================================================

-- name: InsertTaskAcceptance :exec
INSERT INTO task_acceptances (
    task_id,
    user_id,
    accepted_budget,
    response_time_seconds
)
VALUES ($1,$2,$3,$4);


-- ============================================================
-- NOTIFICATIONS
-- ============================================================

-- $2 = []uuid via pq.Array()

-- name: InsertTaskNotificationsBulk :exec
INSERT INTO task_notifications (task_id, user_id, wave_number, status)
SELECT $1, unnest($2), $3, 'sent';

-- name: MarkNotificationOpened :exec
UPDATE task_notifications
SET opened_at = NOW(),
    status = 'opened'
WHERE task_id = $1
AND user_id = $2;

-- name: MarkNotificationResponded :exec
UPDATE task_notifications
SET responded_at = NOW(),
    status = $3
WHERE task_id = $1
AND user_id = $2;


-- ============================================================
-- ACTIVE MARKETPLACE
-- ============================================================

-- $1 = []uuid categories
-- $2 = min_budget
-- $3 = limit

-- name: GetActiveTasksForUser :many
SELECT *
FROM tasks
WHERE state = 'active'
AND expires_at > NOW()
AND category_id = ANY($1)
AND budget >= $2
ORDER BY created_at DESC
LIMIT $3;


-- ============================================================
-- METRICS
-- ============================================================

-- name: GetAcceptanceLatencyP50 :one
SELECT percentile_cont(0.5)
WITHIN GROUP (ORDER BY response_time_seconds)
FROM task_acceptances
WHERE created_at >= NOW() - INTERVAL '7 days';

-- name: GetNotificationWasteRatio :one
SELECT
    COUNT(*) FILTER (WHERE status = 'ignored')::decimal
    / NULLIF(COUNT(*),0)
FROM task_notifications
WHERE created_at >= NOW() - INTERVAL '7 days';


-- ============================================================
-- COLD-START: NEW USERS WITH MATCHING SKILLS
-- ============================================================

-- $1 = task_budget (numeric)
-- $2 = task_id (uuid)
-- $3 = cold_start_threshold (int, e.g. 5)

-- name: GetNewUserCandidates :many
SELECT
    u.id AS user_id,
    up.experience_level,
    up.experience_multiplier,
    up.mab,
    up.radius_km,
    up.fixed_lat,
    up.fixed_lng,
    ubm.acceptance_rate,
    ubm.median_response_seconds,
    ubm.push_open_rate,
    ubm.completion_rate,
    ubm.reliability_score,
    ubm.total_tasks_completed
FROM users u
JOIN user_profiles up ON up.user_id = u.id
JOIN user_behavior_metrics ubm ON ubm.user_id = u.id
WHERE
    u.status = 'active'
    AND ubm.total_tasks_completed < $3
    AND $1 >= (0.5 * up.mab)
    AND EXISTS (
        SELECT 1
        FROM user_skills us
        JOIN task_required_skills trs ON trs.skill_id = us.skill_id AND trs.task_id = $2
        WHERE us.user_id = u.id
    );


-- ============================================================
-- TASK ACCEPTANCE LOOKUP
-- ============================================================

-- name: GetTaskAcceptance :one
SELECT *
FROM task_acceptances
WHERE task_id = $1;


-- ============================================================
-- DEVICE TOKENS (FCM)
-- ============================================================

-- name: UpsertDeviceToken :one
INSERT INTO device_tokens (user_id, token, platform)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, token)
DO UPDATE SET platform = EXCLUDED.platform, updated_at = NOW()
RETURNING *;

-- name: DeleteDeviceToken :exec
DELETE FROM device_tokens
WHERE user_id = $1 AND token = $2;

-- name: DeleteDeviceTokenByToken :exec
DELETE FROM device_tokens
WHERE token = $1;

-- name: GetDeviceTokensByUserID :many
SELECT *
FROM device_tokens
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetDeviceTokensByUserIDs :many
SELECT *
FROM device_tokens
WHERE user_id = ANY($1::uuid[])
ORDER BY user_id, created_at DESC;
