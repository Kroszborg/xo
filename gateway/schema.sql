-- Gateway schema: tables owned by the API gateway.
-- Loaded AFTER xo's 01-schema.sql and 02-seed.sql so the users table exists.

-- xo defines trigger_set_updated_at(); create the alias name used by gateway triggers.
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ─── Refresh Tokens ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens(expires_at);

-- ─── OTP Codes ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS otp_codes (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code        TEXT NOT NULL,
    channel     TEXT NOT NULL CHECK (channel IN ('sms', 'email')),
    purpose     TEXT NOT NULL CHECK (purpose IN ('login', 'verify_phone', 'verify_email', 'reset_password')),
    expires_at  TIMESTAMPTZ NOT NULL,
    verified    BOOLEAN NOT NULL DEFAULT FALSE,
    attempts    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_otp_codes_user ON otp_codes(user_id);

-- ─── User Profile (gateway-owned extended profile) ───────────────────────────
-- This extends xo's user_profiles with onboarding-specific fields.
CREATE TABLE IF NOT EXISTS gateway_user_profile (
    user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name    TEXT,
    first_name      TEXT,
    last_name       TEXT,
    bio             TEXT,
    avatar_url      TEXT,
    date_of_birth   DATE,
    gender          TEXT CHECK (gender IN ('male', 'female', 'other', 'prefer_not_to_say')),
    onboarding_step INT NOT NULL DEFAULT 0 CHECK (onboarding_step BETWEEN 0 AND 7),
    onboarding_done BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_gateway_profile_updated
    BEFORE UPDATE ON gateway_user_profile
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ─── Core Skills (onboarding step 2) ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_core_skills (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_name  TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, skill_name)
);

CREATE INDEX IF NOT EXISTS idx_user_core_skills_user ON user_core_skills(user_id);

-- ─── Other Skills (onboarding step 2) ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_other_skills (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_name  TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, skill_name)
);

CREATE INDEX IF NOT EXISTS idx_user_other_skills_user ON user_other_skills(user_id);

-- ─── Experience (onboarding step 3) ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_experience (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    company     TEXT,
    start_date  DATE,
    end_date    DATE,
    current     BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_experience_user ON user_experience(user_id);

CREATE TRIGGER trg_user_experience_updated
    BEFORE UPDATE ON user_experience
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ─── Education (onboarding step 4) ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_education (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    institution     TEXT NOT NULL,
    degree          TEXT,
    field_of_study  TEXT,
    start_date      DATE,
    end_date        DATE,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_education_user ON user_education(user_id);

CREATE TRIGGER trg_user_education_updated
    BEFORE UPDATE ON user_education
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ─── Certificates (onboarding step 5) ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_certificates (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    issuing_org     TEXT,
    issue_date      DATE,
    expiry_date     DATE,
    credential_id   TEXT,
    credential_url  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_certificates_user ON user_certificates(user_id);

CREATE TRIGGER trg_user_certificates_updated
    BEFORE UPDATE ON user_certificates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ─── Languages (onboarding step 6) ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_languages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language        TEXT NOT NULL,
    proficiency     TEXT CHECK (proficiency IN ('basic', 'conversational', 'fluent', 'native')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, language)
);

CREATE INDEX IF NOT EXISTS idx_user_languages_user ON user_languages(user_id);

-- ─── Payment Methods ─────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_payment_methods (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method_type     TEXT NOT NULL CHECK (method_type IN ('bank_account', 'mobile_money', 'paypal', 'stripe')),
    provider        TEXT,
    account_ref     TEXT NOT NULL,
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_payment_methods_user ON user_payment_methods(user_id);

CREATE TRIGGER trg_user_payment_methods_updated
    BEFORE UPDATE ON user_payment_methods
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ─── Addresses ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_addresses (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label       TEXT,
    line1       TEXT NOT NULL,
    line2       TEXT,
    city        TEXT NOT NULL,
    state       TEXT,
    postal_code TEXT,
    country     TEXT NOT NULL DEFAULT 'US',
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_addresses_user ON user_addresses(user_id);

CREATE TRIGGER trg_user_addresses_updated
    BEFORE UPDATE ON user_addresses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ─── User Location (live/last-known) ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_location (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    lat         NUMERIC(9,6) NOT NULL,
    lng         NUMERIC(9,6) NOT NULL,
    accuracy_m  NUMERIC(8,2),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_user_location_updated
    BEFORE UPDATE ON user_location
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ─── Verification ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_verification (
    user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    phone_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    id_verified     BOOLEAN NOT NULL DEFAULT FALSE,
    id_document_url TEXT,
    verified_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_user_verification_updated
    BEFORE UPDATE ON user_verification
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ─── Categories ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS categories (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL UNIQUE,
    icon_url    TEXT,
    sort_order  INT NOT NULL DEFAULT 0,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── FAQs ────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS faqs (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    question    TEXT NOT NULL,
    answer      TEXT NOT NULL,
    category    TEXT,
    sort_order  INT NOT NULL DEFAULT 0,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── File Uploads ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS file_uploads (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_type   TEXT NOT NULL CHECK (file_type IN ('avatar', 'id_document', 'certificate', 'attachment')),
    file_name   TEXT NOT NULL,
    file_path   TEXT NOT NULL,
    mime_type   TEXT,
    size_bytes  BIGINT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_file_uploads_user ON file_uploads(user_id);
