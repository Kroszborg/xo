-- seed.sql: Dummy data for 3 users (role=both) all within a 10 km radius and 10 tasks.
--
-- Center point: 40.7128° N, 74.0060° W  (New York area)
-- User 1 (Alice): 40.712800, -74.006000  — center
-- User 2 (Bob):   40.739800, -74.006000  — ≈ 3.0 km north
-- User 3 (Carol): 40.712800, -73.947000  — ≈ 5.0 km east
-- Max inter-user distance (Bob ↔ Carol):  ≈ 5.8 km  — all within 10 km
--
-- Fixed UUIDs are used so the script is idempotent via ON CONFLICT clauses.
-- All password_hash values are bcrypt(cost=10) hashes of 'password123'.


-- ─── Skills ──────────────────────────────────────────────────────────────────

INSERT INTO skills (id, name) VALUES
    ('a1000000-0000-0000-0000-000000000001', 'web_development'),
    ('a1000000-0000-0000-0000-000000000002', 'mobile_development'),
    ('a1000000-0000-0000-0000-000000000003', 'data_analysis'),
    ('a1000000-0000-0000-0000-000000000004', 'graphic_design'),
    ('a1000000-0000-0000-0000-000000000005', 'content_writing'),
    ('a1000000-0000-0000-0000-000000000006', 'video_editing')
ON CONFLICT DO NOTHING;


-- ─── Users ───────────────────────────────────────────────────────────────────
-- role='both' makes every user simultaneously a task doer and a task giver.

INSERT INTO users (id, email, phone, password_hash, role, status) VALUES
    (
        'b1000000-0000-0000-0000-000000000001',
        'alice@example.com',
        '+15550001001',
        '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6',
        'both',
        'active'
    ),
    (
        'b1000000-0000-0000-0000-000000000002',
        'bob@example.com',
        '+15550001002',
        '$2b$10$kanyYdVMptH4DBSfxUUcveuNwkua2nOeM6z4MezaG5duTwDPJ1daG',
        'both',
        'active'
    ),
    (
        'b1000000-0000-0000-0000-000000000003',
        'carol@example.com',
        '+15550001003',
        '$2b$10$rQYj7a8mOc62Nnftm.v10uZlR1iY40UkXODh4KoMuw2.9irf4rkF.',
        'both',
        'active'
    )
ON CONFLICT DO NOTHING;


-- ─── User Profiles ───────────────────────────────────────────────────────────
-- radius_km=10 means each user is willing to take/give tasks within 10 km.

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
    profile_completion_score,
    last_active_at
) VALUES
    (
        'b1000000-0000-0000-0000-000000000001',
        'intermediate', 1.10, 150.00,
        10, 40.712800, -74.006000,
        'America/New_York', 'en', 85,
        NOW() - INTERVAL '1 day'
    ),
    (
        'b1000000-0000-0000-0000-000000000002',
        'pro', 1.30, 200.00,
        10, 40.739800, -74.006000,
        'America/New_York', 'en', 90,
        NOW() - INTERVAL '2 days'
    ),
    (
        'b1000000-0000-0000-0000-000000000003',
        'beginner', 0.90, 100.00,
        10, 40.712800, -73.947000,
        'America/New_York', 'en', 70,
        NOW() - INTERVAL '3 days'
    )
ON CONFLICT DO NOTHING;


-- ─── User Skills ─────────────────────────────────────────────────────────────

INSERT INTO user_skills (user_id, skill_id, is_primary) VALUES
    -- Alice: web_development (primary), data_analysis
    ('b1000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000001', TRUE),
    ('b1000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000003', FALSE),
    -- Bob: mobile_development (primary), web_development
    ('b1000000-0000-0000-0000-000000000002', 'a1000000-0000-0000-0000-000000000002', TRUE),
    ('b1000000-0000-0000-0000-000000000002', 'a1000000-0000-0000-0000-000000000001', FALSE),
    -- Carol: graphic_design (primary), content_writing
    ('b1000000-0000-0000-0000-000000000003', 'a1000000-0000-0000-0000-000000000004', TRUE),
    ('b1000000-0000-0000-0000-000000000003', 'a1000000-0000-0000-0000-000000000005', FALSE)
ON CONFLICT DO NOTHING;


-- ─── User Behavior Metrics ───────────────────────────────────────────────────

INSERT INTO user_behavior_metrics (
    user_id,
    acceptance_rate,
    median_response_seconds,
    push_open_rate,
    completion_rate,
    reliability_score,
    total_tasks_completed,
    total_tasks_accepted
) VALUES
    ('b1000000-0000-0000-0000-000000000001', 0.7500, 120, 0.6000, 0.9500,  88.00, 15, 20),
    ('b1000000-0000-0000-0000-000000000002', 0.8200,  90, 0.7500, 0.9800,  94.00, 25, 31),
    ('b1000000-0000-0000-0000-000000000003', 0.6000, 180, 0.5000, 0.8500,  75.00,  8, 13)
ON CONFLICT DO NOTHING;


-- ─── Tasks ───────────────────────────────────────────────────────────────────
-- 10 tasks distributed across the 3 users.
-- Category UUIDs (d1…) are fixed references. Note: the tasks.category_id column
-- carries no REFERENCES constraint in the schema, so arbitrary UUIDs are valid.
-- Physical tasks carry coordinates within the 10 km cluster; online tasks have NULL coords.
-- 'active' tasks carry both priority_started_at and active_started_at.

INSERT INTO tasks (
    id,
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
    active_started_at,
    expires_at
) VALUES
    -- Task 1: Alice posts a web-dev job near center (priority)
    (
        'c1000000-0000-0000-0000-000000000001',
        'b1000000-0000-0000-0000-000000000001',
        'd1000000-0000-0000-0000-000000000001',
        200.00, 4, 'medium', FALSE,
        40.714000, -74.003000, 10,
        'priority', NOW(), NULL, NOW() + INTERVAL '24 hours'
    ),
    -- Task 2: Bob posts a mobile-dev job 2.5 km north (active)
    (
        'c1000000-0000-0000-0000-000000000002',
        'b1000000-0000-0000-0000-000000000002',
        'd1000000-0000-0000-0000-000000000001',
        350.00, 8, 'high', FALSE,
        40.736000, -74.008000, 10,
        'active',
        NOW() - INTERVAL '2 hours',
        NOW() - INTERVAL '1 hour',
        NOW() + INTERVAL '22 hours'
    ),
    -- Task 3: Carol posts an online graphic-design job (active)
    (
        'c1000000-0000-0000-0000-000000000003',
        'b1000000-0000-0000-0000-000000000003',
        'd1000000-0000-0000-0000-000000000002',
        150.00, 2, 'low', TRUE,
        NULL, NULL, 0,
        'active',
        NOW() - INTERVAL '1 hour',
        NOW() - INTERVAL '30 minutes',
        NOW() + INTERVAL '23 hours'
    ),
    -- Task 4: Alice posts a high-budget web+data job near center (priority)
    (
        'c1000000-0000-0000-0000-000000000004',
        'b1000000-0000-0000-0000-000000000001',
        'd1000000-0000-0000-0000-000000000002',
        500.00, 16, 'high', FALSE,
        40.710000, -74.015000, 10,
        'priority', NOW(), NULL, NOW() + INTERVAL '24 hours'
    ),
    -- Task 5: Bob posts an online content-writing job (active)
    (
        'c1000000-0000-0000-0000-000000000005',
        'b1000000-0000-0000-0000-000000000002',
        'd1000000-0000-0000-0000-000000000003',
        80.00, 1, 'low', TRUE,
        NULL, NULL, 0,
        'active',
        NOW() - INTERVAL '3 hours',
        NOW() - INTERVAL '2 hours',
        NOW() + INTERVAL '21 hours'
    ),
    -- Task 6: Carol posts a graphic-design job ~4.8 km east (priority)
    (
        'c1000000-0000-0000-0000-000000000006',
        'b1000000-0000-0000-0000-000000000003',
        'd1000000-0000-0000-0000-000000000003',
        120.00, 3, 'medium', FALSE,
        40.715000, -73.950000, 8,
        'priority', NOW(), NULL, NOW() + INTERVAL '24 hours'
    ),
    -- Task 7: Alice posts a full-stack job near center (active)
    (
        'c1000000-0000-0000-0000-000000000007',
        'b1000000-0000-0000-0000-000000000001',
        'd1000000-0000-0000-0000-000000000004',
        300.00, 6, 'medium', FALSE,
        40.718000, -74.000000, 10,
        'active',
        NOW() - INTERVAL '4 hours',
        NOW() - INTERVAL '3 hours',
        NOW() + INTERVAL '20 hours'
    ),
    -- Task 8: Bob posts an online video-editing job (active)
    (
        'c1000000-0000-0000-0000-000000000008',
        'b1000000-0000-0000-0000-000000000002',
        'd1000000-0000-0000-0000-000000000004',
        250.00, 5, 'medium', TRUE,
        NULL, NULL, 0,
        'active',
        NOW() - INTERVAL '1 hour',
        NOW() - INTERVAL '30 minutes',
        NOW() + INTERVAL '23 hours'
    ),
    -- Task 9: Carol posts a data-analysis job ~4.5 km east (priority)
    (
        'c1000000-0000-0000-0000-000000000009',
        'b1000000-0000-0000-0000-000000000003',
        'd1000000-0000-0000-0000-000000000001',
        400.00, 10, 'high', FALSE,
        40.711000, -73.955000, 8,
        'priority', NOW(), NULL, NOW() + INTERVAL '24 hours'
    ),
    -- Task 10: Alice posts an online content-writing job (active)
    (
        'c1000000-0000-0000-0000-000000000010',
        'b1000000-0000-0000-0000-000000000001',
        'd1000000-0000-0000-0000-000000000002',
        175.00, 3, 'low', TRUE,
        NULL, NULL, 0,
        'active',
        NOW() - INTERVAL '2 hours',
        NOW() - INTERVAL '1 hour',
        NOW() + INTERVAL '22 hours'
    )
ON CONFLICT DO NOTHING;


-- ─── Task Required Skills ─────────────────────────────────────────────────────

INSERT INTO task_required_skills (task_id, skill_id, is_core) VALUES
    -- Task 1: web_development (core)
    ('c1000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000001', TRUE),
    -- Task 2: mobile_development (core)
    ('c1000000-0000-0000-0000-000000000002', 'a1000000-0000-0000-0000-000000000002', TRUE),
    -- Task 3: graphic_design (core), content_writing (supporting)
    ('c1000000-0000-0000-0000-000000000003', 'a1000000-0000-0000-0000-000000000004', TRUE),
    ('c1000000-0000-0000-0000-000000000003', 'a1000000-0000-0000-0000-000000000005', FALSE),
    -- Task 4: web_development (core), data_analysis (supporting)
    ('c1000000-0000-0000-0000-000000000004', 'a1000000-0000-0000-0000-000000000001', TRUE),
    ('c1000000-0000-0000-0000-000000000004', 'a1000000-0000-0000-0000-000000000003', FALSE),
    -- Task 5: content_writing (core)
    ('c1000000-0000-0000-0000-000000000005', 'a1000000-0000-0000-0000-000000000005', TRUE),
    -- Task 6: graphic_design (core)
    ('c1000000-0000-0000-0000-000000000006', 'a1000000-0000-0000-0000-000000000004', TRUE),
    -- Task 7: web_development (core), mobile_development (supporting)
    ('c1000000-0000-0000-0000-000000000007', 'a1000000-0000-0000-0000-000000000001', TRUE),
    ('c1000000-0000-0000-0000-000000000007', 'a1000000-0000-0000-0000-000000000002', FALSE),
    -- Task 8: video_editing (core)
    ('c1000000-0000-0000-0000-000000000008', 'a1000000-0000-0000-0000-000000000006', TRUE),
    -- Task 9: data_analysis (core), web_development (supporting)
    ('c1000000-0000-0000-0000-000000000009', 'a1000000-0000-0000-0000-000000000003', TRUE),
    ('c1000000-0000-0000-0000-000000000009', 'a1000000-0000-0000-0000-000000000001', FALSE),
    -- Task 10: content_writing (core)
    ('c1000000-0000-0000-0000-000000000010', 'a1000000-0000-0000-0000-000000000005', TRUE)
ON CONFLICT DO NOTHING;
