-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Users (password_hash NULLABLE for OAuth users)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT,  -- nullable for OAuth-only users
    role TEXT NOT NULL CHECK (role IN ('task_giver','task_doer','admin','both')),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TRIGGER set_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- OAuth providers
CREATE TABLE user_auth_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('google','facebook')),
    provider_user_id TEXT NOT NULL,
    provider_email TEXT,
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (provider, provider_user_id)
);
CREATE TRIGGER set_user_auth_providers_updated_at BEFORE UPDATE ON user_auth_providers FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- User profiles
CREATE TABLE user_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name TEXT,
    bio TEXT,
    avatar_url TEXT,
    phone TEXT,
    latitude NUMERIC(10,7),
    longitude NUMERIC(10,7),
    city TEXT,
    state TEXT,
    country TEXT,
    max_distance_km INT DEFAULT 50,
    preferred_budget_min NUMERIC(12,2),
    preferred_budget_max NUMERIC(12,2),
    is_online BOOLEAN DEFAULT FALSE,
    onboarding_step INT DEFAULT 0,
    onboarding_completed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TRIGGER set_user_profiles_updated_at BEFORE UPDATE ON user_profiles FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Skills
CREATE TABLE skills (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User skills (many-to-many)
CREATE TABLE user_skills (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    proficiency_level INT DEFAULT 1 CHECK (proficiency_level BETWEEN 1 AND 5),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, skill_id)
);

-- User behavior metrics (adds total_tasks_notified, average_review_score, total_reviews_received)
CREATE TABLE user_behavior_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_tasks_completed INT DEFAULT 0,
    total_tasks_accepted INT DEFAULT 0,
    total_tasks_notified INT DEFAULT 0,
    total_reviews_received INT DEFAULT 0,
    average_response_time_minutes NUMERIC(10,2) DEFAULT 0,
    completion_rate NUMERIC(5,4) DEFAULT 0,
    acceptance_rate NUMERIC(5,4) DEFAULT 0,
    reliability_score NUMERIC(5,2) DEFAULT 50,
    average_review_score NUMERIC(3,2) DEFAULT 0,
    consistency_score NUMERIC(5,4) DEFAULT 0.5,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TRIGGER set_user_behavior_metrics_updated_at BEFORE UPDATE ON user_behavior_metrics FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Experience multiplier history
CREATE TABLE experience_multiplier_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    old_multiplier NUMERIC(5,4),
    new_multiplier NUMERIC(5,4),
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Task categories (fixed admin-managed set)
CREATE TABLE task_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    icon_url TEXT,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Tasks (adds title, description, client_type, slm_category_id, slm_category_confidence, in_progress state)
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_by UUID NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    description TEXT,
    budget NUMERIC(12,2) NOT NULL,
    latitude NUMERIC(10,7),
    longitude NUMERIC(10,7),
    radius NUMERIC(10,2) DEFAULT 50.0,
    city TEXT,
    location_name TEXT,
    is_online BOOLEAN DEFAULT FALSE,
    urgency TEXT DEFAULT 'normal' CHECK (urgency IN ('low','normal','high','critical')),
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending','matching','matched','active','in_progress','completed','cancelled','expired')),
    client_type TEXT DEFAULT 'web' CHECK (client_type IN ('web','mobile_android','mobile_ios')),
    category_id UUID REFERENCES task_categories(id),
    slm_category_id UUID REFERENCES task_categories(id),
    slm_category_confidence NUMERIC(5,4),
    accepted_by UUID REFERENCES users(id),
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TRIGGER set_tasks_updated_at BEFORE UPDATE ON tasks FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Task required skills
CREATE TABLE task_required_skills (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    minimum_proficiency INT DEFAULT 1 CHECK (minimum_proficiency BETWEEN 1 AND 5),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (task_id, skill_id)
);

-- Task acceptances
CREATE TABLE task_acceptances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending','accepted','rejected','expired')),
    responded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (task_id, user_id)
);

-- Task notifications (adds channel, delivered status)
CREATE TABLE task_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    wave_number INT NOT NULL,
    score NUMERIC(10,4),
    is_exploration BOOLEAN DEFAULT FALSE,
    channel TEXT DEFAULT 'fcm' CHECK (channel IN ('fcm','webpush','inapp')),
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending','sent','delivered','failed','expired')),
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Task state transitions (adds metadata JSONB)
CREATE TABLE task_state_transitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    from_status TEXT,
    to_status TEXT NOT NULL,
    triggered_by UUID REFERENCES users(id),
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Device tokens (FCM)
CREATE TABLE device_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    platform TEXT NOT NULL CHECK (platform IN ('android','ios')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, token)
);
CREATE TRIGGER set_device_tokens_updated_at BEFORE UPDATE ON device_tokens FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Web push subscriptions (VAPID)
CREATE TABLE web_push_subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    p256dh_key TEXT NOT NULL,
    auth_key TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, endpoint)
);

-- In-app notification feed
CREATE TABLE inapp_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('task_match','task_accepted','task_completed','review_received','chat_message','system')),
    title TEXT NOT NULL,
    body TEXT,
    payload JSONB,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Conversations (1-on-1 chat, unlocks after task acceptance)
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    participant_a UUID NOT NULL REFERENCES users(id),
    participant_b UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (task_id, participant_a, participant_b)
);
CREATE TRIGGER set_conversations_updated_at BEFORE UPDATE ON conversations FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Chat messages
CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    content_moderated TEXT,
    moderation_flags JSONB,
    moderation_status TEXT DEFAULT 'pending' CHECK (moderation_status IN ('pending','clean','flagged','blocked')),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Task reviews (bidirectional, independent, immediately visible)
CREATE TABLE task_reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES users(id),
    reviewee_id UUID NOT NULL REFERENCES users(id),
    reviewer_role TEXT NOT NULL CHECK (reviewer_role IN ('task_giver','task_doer')),
    rating INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (task_id, reviewer_id)
);

-- Disputes
CREATE TABLE disputes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    initiated_by UUID NOT NULL REFERENCES users(id),
    against_user UUID NOT NULL REFERENCES users(id),
    reason TEXT NOT NULL,
    status TEXT DEFAULT 'open' CHECK (status IN ('open','under_review','resolved','dismissed')),
    resolution TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- Gateway tables
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE otp_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    purpose TEXT NOT NULL CHECK (purpose IN ('login','verify_email','verify_phone','password_reset')),
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User experience (onboarding)
CREATE TABLE user_experience (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    company TEXT,
    start_date DATE,
    end_date DATE,
    current BOOLEAN DEFAULT FALSE,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TRIGGER set_user_experience_updated_at BEFORE UPDATE ON user_experience FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- User education (onboarding)
CREATE TABLE user_education (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    institution TEXT NOT NULL,
    degree TEXT,
    field_of_study TEXT,
    start_date DATE,
    end_date DATE,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TRIGGER set_user_education_updated_at BEFORE UPDATE ON user_education FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- User certificates (onboarding)
CREATE TABLE user_certificates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    issuing_org TEXT,
    issue_date DATE,
    expiry_date DATE,
    credential_id TEXT,
    credential_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TRIGGER set_user_certificates_updated_at BEFORE UPDATE ON user_certificates FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- User languages (onboarding)
CREATE TABLE user_languages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    proficiency TEXT CHECK (proficiency IN ('beginner','intermediate','advanced','native')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, language)
);
CREATE TRIGGER set_user_languages_updated_at BEFORE UPDATE ON user_languages FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- User preference signals (behavioral patterns from accept/reject history)
CREATE TABLE user_preference_signals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID REFERENCES task_categories(id) ON DELETE CASCADE,
    signal_type TEXT NOT NULL CHECK (signal_type IN (
        'category_affinity',
        'budget_accept_avg', 'budget_accept_count',
        'budget_reject_avg', 'budget_reject_count',
        'geo_avg_distance_accepted',
        'ignore_count'
    )),
    signal_value NUMERIC(10,4) NOT NULL,
    sample_size INT NOT NULL DEFAULT 0,
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, category_id, signal_type)
);
CREATE INDEX idx_preference_signals_user ON user_preference_signals(user_id);
CREATE INDEX idx_preference_signals_user_category ON user_preference_signals(user_id, category_id);

-- Relevancy scores (materialized for online tasks)
CREATE TABLE relevancy_scores (
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_fit NUMERIC(6,4) NOT NULL,
    acceptance_likelihood NUMERIC(6,4) NOT NULL,
    cold_start_multiplier NUMERIC(4,2) NOT NULL DEFAULT 1.0,
    final_score NUMERIC(8,4) NOT NULL,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, user_id)
);
CREATE INDEX idx_relevancy_task_score ON relevancy_scores(task_id, final_score DESC);
CREATE INDEX idx_relevancy_user_score ON relevancy_scores(user_id, final_score DESC);
CREATE INDEX idx_relevancy_task ON relevancy_scores(task_id);

-- Matching queue (persistent queue for offline orchestration)
CREATE TABLE matching_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    score NUMERIC(8,4) NOT NULL,
    task_fit NUMERIC(6,4) NOT NULL,
    acceptance_likelihood NUMERIC(6,4) NOT NULL,
    status TEXT DEFAULT 'queued' CHECK (status IN (
        'queued', 'active', 'notified', 'accepted',
        'declined', 'ignored', 'cancelled', 'filtered'
    )),
    position INT NOT NULL,
    notified_at TIMESTAMPTZ,
    responded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(task_id, user_id)
);
CREATE INDEX idx_matching_queue_task_status ON matching_queue(task_id, status, position);
CREATE INDEX idx_matching_queue_task_active ON matching_queue(task_id) WHERE status IN ('queued', 'active', 'notified');

-- Giver behavior metrics (task giver reputation)
CREATE TABLE giver_behavior_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_tasks_posted INT DEFAULT 0,
    total_tasks_completed INT DEFAULT 0,
    total_tasks_cancelled INT DEFAULT 0,
    total_tasks_expired INT DEFAULT 0,
    avg_review_from_doers NUMERIC(3,2) DEFAULT 0,
    total_reviews_from_doers INT DEFAULT 0,
    repost_count INT DEFAULT 0,
    last_repost_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TRIGGER set_giver_behavior_metrics_updated_at
    BEFORE UPDATE ON giver_behavior_metrics
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Indexes for performance
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created_by ON tasks(created_by);
CREATE INDEX idx_tasks_accepted_by ON tasks(accepted_by);
CREATE INDEX idx_tasks_category ON tasks(category_id);
CREATE INDEX idx_task_notifications_task ON task_notifications(task_id);
CREATE INDEX idx_task_notifications_user ON task_notifications(user_id);
CREATE INDEX idx_task_acceptances_task ON task_acceptances(task_id);
CREATE INDEX idx_task_acceptances_user ON task_acceptances(user_id);
CREATE INDEX idx_user_skills_user ON user_skills(user_id);
CREATE INDEX idx_user_skills_skill ON user_skills(skill_id);
CREATE INDEX idx_chat_messages_conversation ON chat_messages(conversation_id);
CREATE INDEX idx_chat_messages_created ON chat_messages(created_at);
CREATE INDEX idx_inapp_notifications_user ON inapp_notifications(user_id);
CREATE INDEX idx_inapp_notifications_unread ON inapp_notifications(user_id) WHERE read_at IS NULL;
CREATE INDEX idx_task_reviews_task ON task_reviews(task_id);
CREATE INDEX idx_task_reviews_reviewee ON task_reviews(reviewee_id);
CREATE INDEX idx_conversations_participants ON conversations(participant_a, participant_b);
CREATE INDEX idx_device_tokens_user ON device_tokens(user_id);
CREATE INDEX idx_web_push_user ON web_push_subscriptions(user_id);
CREATE INDEX idx_user_auth_providers_user ON user_auth_providers(user_id);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_user_experience_user ON user_experience(user_id);
CREATE INDEX idx_user_education_user ON user_education(user_id);
CREATE INDEX idx_user_certificates_user ON user_certificates(user_id);
CREATE INDEX idx_user_languages_user ON user_languages(user_id);
CREATE INDEX idx_user_profiles_location ON user_profiles(latitude, longitude) WHERE latitude IS NOT NULL AND longitude IS NOT NULL;
CREATE INDEX idx_relevancy_scores_cleanup ON relevancy_scores(task_id);
CREATE INDEX idx_matching_queue_task ON matching_queue(task_id);
CREATE INDEX idx_giver_behavior_metrics_user ON giver_behavior_metrics(user_id);
CREATE INDEX idx_user_preference_signals_lookup ON user_preference_signals(user_id, category_id, signal_type);
