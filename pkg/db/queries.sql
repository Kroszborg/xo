-- =====================
-- Users
-- =====================

-- name: GetUserByID :one
SELECT id, email, password_hash, role, is_active, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, is_active, created_at, updated_at
FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING id, email, password_hash, role, is_active, created_at, updated_at;

-- =====================
-- User Profiles
-- =====================

-- name: GetUserProfile :one
SELECT up.*, u.email, u.role
FROM user_profiles up
JOIN users u ON u.id = up.user_id
WHERE up.user_id = $1;

-- name: CreateUserProfile :one
INSERT INTO user_profiles (user_id) VALUES ($1)
RETURNING *;

-- =====================
-- User Behavior Metrics
-- =====================

-- name: GetUserBehaviorMetrics :one
SELECT * FROM user_behavior_metrics WHERE user_id = $1;

-- name: CreateUserBehaviorMetrics :one
INSERT INTO user_behavior_metrics (user_id) VALUES ($1)
RETURNING *;

-- name: UpdateUserBehaviorMetrics :exec
UPDATE user_behavior_metrics
SET total_tasks_completed = $2,
    total_tasks_accepted = $3,
    total_tasks_notified = $4,
    total_reviews_received = $5,
    average_response_time_minutes = $6,
    completion_rate = $7,
    acceptance_rate = $8,
    reliability_score = $9,
    average_review_score = $10,
    consistency_score = $11
WHERE user_id = $1;

-- =====================
-- Skills
-- =====================

-- name: ListSkills :many
SELECT id, name, created_at FROM skills ORDER BY name;

-- name: GetSkillByName :one
SELECT id, name, created_at FROM skills WHERE name = $1;

-- =====================
-- User Skills
-- =====================

-- name: GetUserSkills :many
SELECT us.id, us.user_id, us.skill_id, s.name AS skill_name, us.proficiency_level, us.created_at
FROM user_skills us
JOIN skills s ON s.id = us.skill_id
WHERE us.user_id = $1;

-- name: AddUserSkill :one
INSERT INTO user_skills (user_id, skill_id, proficiency_level)
VALUES ($1, $2, $3)
RETURNING *;

-- =====================
-- Task Categories
-- =====================

-- name: ListActiveCategories :many
SELECT id, name, description, icon_url, active, created_at
FROM task_categories
WHERE active = TRUE
ORDER BY name;

-- name: GetCategoryByID :one
SELECT * FROM task_categories WHERE id = $1;

-- =====================
-- Tasks
-- =====================

-- name: CreateTask :one
INSERT INTO tasks (
    created_by, title, description, budget, latitude, longitude, city,
    is_online, urgency, client_type, category_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetTaskByID :one
SELECT * FROM tasks WHERE id = $1;

-- name: ListTasks :many
SELECT * FROM tasks
WHERE (CASE WHEN @cursor::text != '' THEN created_at < (SELECT created_at FROM tasks WHERE id = @cursor::uuid) ELSE TRUE END)
ORDER BY created_at DESC
LIMIT @page_limit;

-- name: ListTasksByUser :many
SELECT * FROM tasks
WHERE created_by = $1
ORDER BY created_at DESC;

-- name: UpdateTaskStatus :exec
UPDATE tasks SET status = $2 WHERE id = $1;

-- name: UpdateTaskAcceptedBy :exec
UPDATE tasks SET accepted_by = $2, status = 'in_progress' WHERE id = $1;

-- name: CompleteTask :exec
UPDATE tasks SET status = 'completed', completed_at = NOW() WHERE id = $1;

-- name: UpdateTaskSLMCategory :exec
UPDATE tasks SET slm_category_id = $2, slm_category_confidence = $3 WHERE id = $1;

-- =====================
-- Task Required Skills
-- =====================

-- name: GetTaskRequiredSkills :many
SELECT trs.*, s.name AS skill_name
FROM task_required_skills trs
JOIN skills s ON s.id = trs.skill_id
WHERE trs.task_id = $1;

-- name: AddTaskRequiredSkill :one
INSERT INTO task_required_skills (task_id, skill_id, minimum_proficiency)
VALUES ($1, $2, $3)
RETURNING *;

-- =====================
-- Task Acceptances
-- =====================

-- name: CreateTaskAcceptance :one
INSERT INTO task_acceptances (task_id, user_id, status)
VALUES ($1, $2, 'accepted')
RETURNING *;

-- name: GetTaskAcceptance :one
SELECT * FROM task_acceptances
WHERE task_id = $1 AND user_id = $2;

-- =====================
-- Task Notifications
-- =====================

-- name: CreateTaskNotification :one
INSERT INTO task_notifications (task_id, user_id, wave_number, score, is_exploration, channel)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetPendingNotifications :many
SELECT * FROM task_notifications
WHERE task_id = $1 AND status = 'pending'
ORDER BY score DESC;

-- =====================
-- Task State Transitions
-- =====================

-- name: CreateTaskStateTransition :one
INSERT INTO task_state_transitions (task_id, from_status, to_status, triggered_by, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- =====================
-- Device Tokens
-- =====================

-- name: UpsertDeviceToken :one
INSERT INTO device_tokens (user_id, token, platform)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, token) DO UPDATE SET updated_at = NOW()
RETURNING *;

-- name: DeleteDeviceToken :exec
DELETE FROM device_tokens WHERE user_id = $1 AND token = $2;

-- name: GetUserDeviceTokens :many
SELECT * FROM device_tokens WHERE user_id = $1;

-- =====================
-- Web Push Subscriptions
-- =====================

-- name: CreateWebPushSubscription :one
INSERT INTO web_push_subscriptions (user_id, endpoint, p256dh_key, auth_key)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, endpoint) DO NOTHING
RETURNING *;

-- name: DeleteWebPushSubscription :exec
DELETE FROM web_push_subscriptions WHERE user_id = $1 AND endpoint = $2;

-- name: GetUserWebPushSubscriptions :many
SELECT * FROM web_push_subscriptions WHERE user_id = $1;

-- =====================
-- In-App Notifications
-- =====================

-- name: CreateInAppNotification :one
INSERT INTO inapp_notifications (user_id, type, title, body, payload)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListUserNotifications :many
SELECT * FROM inapp_notifications
WHERE user_id = $1
  AND (CASE WHEN @cursor::text != '' THEN created_at < (SELECT created_at FROM inapp_notifications WHERE id = @cursor::uuid) ELSE TRUE END)
ORDER BY created_at DESC
LIMIT @page_limit;

-- name: MarkNotificationRead :exec
UPDATE inapp_notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2;

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM inapp_notifications WHERE user_id = $1 AND read_at IS NULL;

-- =====================
-- Conversations
-- =====================

-- name: CreateConversation :one
INSERT INTO conversations (task_id, participant_a, participant_b)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetConversation :one
SELECT * FROM conversations WHERE id = $1;

-- name: GetConversationByTask :one
SELECT * FROM conversations WHERE task_id = $1;

-- name: GetUserConversations :many
SELECT * FROM conversations
WHERE participant_a = $1 OR participant_b = $1
ORDER BY updated_at DESC;

-- =====================
-- Chat Messages
-- =====================

-- name: CreateChatMessage :one
INSERT INTO chat_messages (conversation_id, sender_id, content, content_moderated, moderation_flags, moderation_status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetChatMessages :many
SELECT * FROM chat_messages
WHERE conversation_id = $1
  AND (CASE WHEN @cursor::text != '' THEN created_at < (SELECT created_at FROM chat_messages WHERE id = @cursor::uuid) ELSE TRUE END)
ORDER BY created_at DESC
LIMIT @page_limit;

-- =====================
-- Task Reviews
-- =====================

-- name: CreateTaskReview :one
INSERT INTO task_reviews (task_id, reviewer_id, reviewee_id, reviewer_role, rating, comment)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetTaskReviews :many
SELECT * FROM task_reviews WHERE task_id = $1 ORDER BY created_at DESC;

-- name: GetUserReviews :many
SELECT tr.*, t.title AS task_title
FROM task_reviews tr
JOIN tasks t ON t.id = tr.task_id
WHERE tr.reviewee_id = $1
ORDER BY tr.created_at DESC;

-- name: GetUserAverageRating :one
SELECT COALESCE(AVG(rating), 0)::numeric(3,2) AS avg_rating,
       COUNT(*)::int AS total_reviews
FROM task_reviews WHERE reviewee_id = $1;

-- =====================
-- Disputes
-- =====================

-- name: CreateDispute :one
INSERT INTO disputes (task_id, initiated_by, against_user, reason)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetDispute :one
SELECT * FROM disputes WHERE id = $1;

-- =====================
-- Eligible Candidates (for TURS scoring)
-- =====================

-- name: GetEligibleCandidates :many
SELECT
    u.id AS user_id,
    up.latitude,
    up.longitude,
    up.preferred_budget_min,
    up.preferred_budget_max,
    up.max_distance_km,
    ubm.total_tasks_completed,
    ubm.total_tasks_accepted,
    ubm.total_tasks_notified,
    ubm.average_response_time_minutes,
    ubm.completion_rate,
    ubm.acceptance_rate,
    ubm.reliability_score,
    ubm.average_review_score,
    ubm.consistency_score,
    ARRAY_AGG(us.skill_id) AS skill_ids,
    ARRAY_AGG(us.proficiency_level) AS proficiency_levels
FROM users u
JOIN user_profiles up ON up.user_id = u.id
JOIN user_behavior_metrics ubm ON ubm.user_id = u.id
LEFT JOIN user_skills us ON us.user_id = u.id
WHERE u.role = 'task_doer'
  AND u.is_active = TRUE
  AND u.id != $1
GROUP BY u.id, up.latitude, up.longitude, up.preferred_budget_min,
         up.preferred_budget_max, up.max_distance_km,
         ubm.total_tasks_completed, ubm.total_tasks_accepted,
         ubm.total_tasks_notified, ubm.average_response_time_minutes,
         ubm.completion_rate, ubm.acceptance_rate, ubm.reliability_score,
         ubm.average_review_score, ubm.consistency_score;

-- =====================
-- OAuth Providers
-- =====================

-- name: GetUserAuthProvider :one
SELECT * FROM user_auth_providers
WHERE provider = $1 AND provider_user_id = $2;

-- name: CreateUserAuthProvider :one
INSERT INTO user_auth_providers (user_id, provider, provider_user_id, provider_email, access_token, refresh_token, token_expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetUserAuthProviders :many
SELECT * FROM user_auth_providers WHERE user_id = $1;
