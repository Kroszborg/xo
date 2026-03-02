CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT UNIQUE NOT NULL,
    phone TEXT UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('task_doer','task_giver','both')),
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_status ON users(status);

CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    experience_level TEXT NOT NULL CHECK (
        experience_level IN ('beginner','intermediate','pro','elite')
    ),

    experience_multiplier NUMERIC(3,2) NOT NULL CHECK (
        experience_multiplier BETWEEN 0.5 AND 2.0
    ),

    mab NUMERIC(10,2) NOT NULL CHECK (mab >= 0),

    radius_km INT NOT NULL CHECK (radius_km >= 0),

    fixed_lat NUMERIC(9,6),
    fixed_lng NUMERIC(9,6),

    timezone TEXT,
    language TEXT,

    profile_completion_score INT DEFAULT 0 CHECK (
        profile_completion_score BETWEEN 0 AND 100
    ),

    last_active_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_profiles_last_active ON user_profiles(last_active_at);
CREATE INDEX idx_profiles_mab ON user_profiles(mab);

CREATE TABLE skills (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    category_id UUID,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE user_skills (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    skill_id UUID REFERENCES skills(id) ON DELETE CASCADE,
    is_primary BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (user_id, skill_id)
);

CREATE INDEX idx_user_skills_skill ON user_skills(skill_id);


CREATE TABLE user_behavior_metrics (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    acceptance_rate NUMERIC(5,4) DEFAULT 0 CHECK (
        acceptance_rate BETWEEN 0 AND 1
    ),

    median_response_seconds INT DEFAULT 0 CHECK (
        median_response_seconds >= 0
    ),

    push_open_rate NUMERIC(5,4) DEFAULT 0 CHECK (
        push_open_rate BETWEEN 0 AND 1
    ),

    completion_rate NUMERIC(5,4) DEFAULT 1 CHECK (
        completion_rate BETWEEN 0 AND 1
    ),

    reliability_score NUMERIC(5,2) DEFAULT 100 CHECK (
        reliability_score BETWEEN 0 AND 100
    ),

    total_tasks_completed INT DEFAULT 0 CHECK (
        total_tasks_completed >= 0
    ),

    total_tasks_accepted INT DEFAULT 0 CHECK (
        total_tasks_accepted >= 0
    ),

    updated_at TIMESTAMP DEFAULT NOW()
);


CREATE TABLE experience_multiplier_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID REFERENCES users(id) ON DELETE CASCADE,

    old_multiplier NUMERIC(3,2),
    new_multiplier NUMERIC(3,2),

    accepted_budget NUMERIC(10,2),
    shown_budget NUMERIC(10,2),

    alpha NUMERIC(3,2),

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_em_history_user ON experience_multiplier_history(user_id);


CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    task_giver_id UUID REFERENCES users(id),

    category_id UUID NOT NULL,

    budget NUMERIC(10,2) NOT NULL CHECK (budget > 0),

    duration_hours INT,
    complexity_level TEXT,

    is_online BOOLEAN DEFAULT TRUE,

    lat NUMERIC(9,6),
    lng NUMERIC(9,6),
    radius_km INT CHECK (radius_km >= 0),

    state TEXT NOT NULL CHECK (
        state IN (
            'draft',
            'priority',
            'active',
            'accepted',
            'completed',
            'expired',
            'cancelled'
        )
    ),

    priority_started_at TIMESTAMP,
    active_started_at TIMESTAMP,
    expires_at TIMESTAMP,
    accepted_at TIMESTAMP,
    completed_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_tasks_state ON tasks(state);
CREATE INDEX idx_tasks_category ON tasks(category_id);
CREATE INDEX idx_tasks_priority_started ON tasks(priority_started_at);
CREATE INDEX idx_tasks_expires ON tasks(expires_at);


CREATE TABLE task_required_skills (
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    skill_id UUID REFERENCES skills(id) ON DELETE CASCADE,
    is_core BOOLEAN DEFAULT TRUE,
    PRIMARY KEY (task_id, skill_id)
);

CREATE INDEX idx_task_required_skills_skill ON task_required_skills(skill_id);

CREATE TABLE task_acceptances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    task_id UUID UNIQUE REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),

    accepted_budget NUMERIC(10,2) NOT NULL CHECK (accepted_budget > 0),
    response_time_seconds INT CHECK (response_time_seconds >= 0),

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_task_acceptances_user ON task_acceptances(user_id);

CREATE TABLE task_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,

    wave_number INT CHECK (wave_number >= 1),

    opened_at TIMESTAMP,
    responded_at TIMESTAMP,

    status TEXT CHECK (
        status IN ('sent','opened','ignored','accepted','expired')
    ),

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notifications_task ON task_notifications(task_id);
CREATE INDEX idx_notifications_user ON task_notifications(user_id);


CREATE TABLE task_state_transitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,

    from_state TEXT,
    to_state TEXT,

    triggered_by TEXT,

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_task_state_transitions_task
    ON task_state_transitions(task_id);


CREATE TABLE device_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    token TEXT NOT NULL,

    platform TEXT NOT NULL CHECK (platform IN ('android', 'ios')),

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE (user_id, token)
);

CREATE INDEX idx_device_tokens_user ON device_tokens(user_id);
CREATE INDEX idx_device_tokens_token ON device_tokens(token);


CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
   RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_profiles_updated
BEFORE UPDATE ON user_profiles
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_tasks_updated
BEFORE UPDATE ON tasks
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_device_tokens_updated
BEFORE UPDATE ON device_tokens
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
