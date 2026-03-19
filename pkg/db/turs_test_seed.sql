-- TURS test seed: 20 users with varying stats + 1 test task
-- This file supplements the existing seed.sql
-- Run this AFTER seed.sql to add test data for TURS algorithm testing

-- ─── Additional Skills ───────────────────────────────────────────────────────

INSERT INTO skills (id, name) VALUES
    ('a1000000-0000-0000-0000-000000000007', 'machine_learning'),
    ('a1000000-0000-0000-0000-000000000008', 'project_management'),
    ('a1000000-0000-0000-0000-000000000009', 'ui_ux_design'),
    ('a1000000-0000-0000-0000-000000000010', 'devops')
ON CONFLICT DO NOTHING;


-- ─── Test Users (20 users with diverse profiles) ─────────────────────────────
-- Center: 40.7128° N, 74.0060° W (NYC)
-- Users distributed across different distances, experience levels, and behaviors

INSERT INTO users (id, email, phone, password_hash, role, status) VALUES
    -- Elite performers (users 4-6)
    ('b1000000-0000-0000-0000-000000000004', 'david@example.com', '+15550001004', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000005', 'emma@example.com', '+15550001005', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000006', 'frank@example.com', '+15550001006', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'both', 'active'),
    
    -- Pro performers (users 7-10)
    ('b1000000-0000-0000-0000-000000000007', 'grace@example.com', '+15550001007', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000008', 'henry@example.com', '+15550001008', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000009', 'isabel@example.com', '+15550001009', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'both', 'active'),
    ('b1000000-0000-0000-0000-000000000010', 'jack@example.com', '+15550001010', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    
    -- Intermediate users (users 11-15)
    ('b1000000-0000-0000-0000-000000000011', 'kate@example.com', '+15550001011', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000012', 'leo@example.com', '+15550001012', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000013', 'mia@example.com', '+15550001013', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'both', 'active'),
    ('b1000000-0000-0000-0000-000000000014', 'noah@example.com', '+15550001014', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000015', 'olivia@example.com', '+15550001015', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    
    -- Beginners (users 16-20)
    ('b1000000-0000-0000-0000-000000000016', 'paul@example.com', '+15550001016', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000017', 'quinn@example.com', '+15550001017', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000018', 'ruby@example.com', '+15550001018', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'both', 'active'),
    ('b1000000-0000-0000-0000-000000000019', 'sam@example.com', '+15550001019', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000020', 'tara@example.com', '+15550001020', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    
    -- New users with minimal history (users 21-23) - cold start testing
    ('b1000000-0000-0000-0000-000000000021', 'uma@example.com', '+15550001021', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000022', 'victor@example.com', '+15550001022', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active'),
    ('b1000000-0000-0000-0000-000000000023', 'wendy@example.com', '+15550001023', '$2b$10$qs/CEiSphJI30W5b67qyFuJHEcs48DUi5OwCJVJq8YB3WRFllpBc6', 'task_doer', 'active')
ON CONFLICT DO NOTHING;


-- ─── User Profiles (varying locations, MAB, experience) ──────────────────────
-- Distances from center (40.7128, -74.006):
-- ~1km: ±0.009 lat/lng, ~3km: ±0.027, ~5km: ±0.045, ~8km: ±0.072, ~15km: ±0.135

INSERT INTO user_profiles (
    user_id, experience_level, experience_multiplier, mab, radius_km,
    fixed_lat, fixed_lng, timezone, language, profile_completion_score, last_active_at
) VALUES
    -- Elite: High MAB, perfect profile, close locations, high multipliers
    ('b1000000-0000-0000-0000-000000000004', 'elite', 1.80, 500.00, 15, 40.7150, -74.0030, 'America/New_York', 'en', 100, NOW() - INTERVAL '1 hour'),
    ('b1000000-0000-0000-0000-000000000005', 'elite', 1.90, 600.00, 20, 40.7100, -74.0100, 'America/New_York', 'en', 98, NOW() - INTERVAL '30 minutes'),
    ('b1000000-0000-0000-0000-000000000006', 'elite', 1.75, 450.00, 12, 40.7200, -74.0000, 'America/New_York', 'en', 95, NOW() - INTERVAL '2 hours'),
    
    -- Pro: Good MAB, various distances
    ('b1000000-0000-0000-0000-000000000007', 'pro', 1.40, 350.00, 10, 40.7400, -74.0060, 'America/New_York', 'en', 90, NOW() - INTERVAL '3 hours'),   -- ~3km north
    ('b1000000-0000-0000-0000-000000000008', 'pro', 1.35, 300.00, 8,  40.7128, -73.9500, 'America/New_York', 'en', 88, NOW() - INTERVAL '1 day'),     -- ~5km east
    ('b1000000-0000-0000-0000-000000000009', 'pro', 1.50, 400.00, 15, 40.6900, -74.0060, 'America/New_York', 'en', 92, NOW() - INTERVAL '4 hours'),   -- ~2.5km south
    ('b1000000-0000-0000-0000-000000000010', 'pro', 1.30, 280.00, 10, 40.7500, -74.0300, 'America/New_York', 'en', 85, NOW() - INTERVAL '6 hours'),   -- ~5km northwest
    
    -- Intermediate: Moderate stats, mixed locations
    ('b1000000-0000-0000-0000-000000000011', 'intermediate', 1.15, 200.00, 8,  40.7128, -74.0060, 'America/New_York', 'en', 80, NOW() - INTERVAL '12 hours'), -- center
    ('b1000000-0000-0000-0000-000000000012', 'intermediate', 1.10, 180.00, 10, 40.7800, -74.0060, 'America/New_York', 'en', 75, NOW() - INTERVAL '1 day'),    -- ~7.5km north
    ('b1000000-0000-0000-0000-000000000013', 'intermediate', 1.20, 220.00, 12, 40.7128, -73.9000, 'America/New_York', 'en', 82, NOW() - INTERVAL '8 hours'),   -- ~9km east
    ('b1000000-0000-0000-0000-000000000014', 'intermediate', 1.05, 160.00, 6,  40.6500, -74.0060, 'America/New_York', 'en', 70, NOW() - INTERVAL '2 days'),    -- ~7km south
    ('b1000000-0000-0000-0000-000000000015', 'intermediate', 1.25, 250.00, 15, 40.7128, -74.0800, 'America/New_York', 'en', 78, NOW() - INTERVAL '10 hours'),  -- ~6km west
    
    -- Beginners: Lower MAB, smaller radius, varying locations
    ('b1000000-0000-0000-0000-000000000016', 'beginner', 0.95, 120.00, 5,  40.7128, -74.0060, 'America/New_York', 'en', 60, NOW() - INTERVAL '3 days'),  -- center
    ('b1000000-0000-0000-0000-000000000017', 'beginner', 0.85, 100.00, 5,  40.8000, -74.0060, 'America/New_York', 'en', 55, NOW() - INTERVAL '4 days'),  -- ~10km north
    ('b1000000-0000-0000-0000-000000000018', 'beginner', 0.90, 130.00, 8,  40.7128, -73.8500, 'America/New_York', 'en', 65, NOW() - INTERVAL '2 days'),  -- ~13km east
    ('b1000000-0000-0000-0000-000000000019', 'beginner', 0.80, 90.00,  5,  40.6000, -74.0060, 'America/New_York', 'en', 50, NOW() - INTERVAL '5 days'),  -- ~12.5km south
    ('b1000000-0000-0000-0000-000000000020', 'beginner', 1.00, 150.00, 10, 40.7128, -74.1500, 'America/New_York', 'en', 68, NOW() - INTERVAL '1 day'),   -- ~12km west
    
    -- New users (cold start): minimal data, recent signups
    ('b1000000-0000-0000-0000-000000000021', 'beginner', 1.00, 200.00, 10, 40.7150, -74.0040, 'America/New_York', 'en', 40, NOW() - INTERVAL '1 hour'),  -- ~0.3km 
    ('b1000000-0000-0000-0000-000000000022', 'intermediate', 1.00, 250.00, 15, 40.7200, -74.0100, 'America/New_York', 'en', 50, NOW() - INTERVAL '2 hours'),
    ('b1000000-0000-0000-0000-000000000023', 'pro', 1.00, 300.00, 12, 40.7100, -74.0000, 'America/New_York', 'en', 45, NOW() - INTERVAL '30 minutes')
ON CONFLICT DO NOTHING;


-- ─── User Skills (diverse skill sets) ────────────────────────────────────────

INSERT INTO user_skills (user_id, skill_id, is_primary) VALUES
    -- David (Elite): web_dev (primary), mobile_dev, data_analysis
    ('b1000000-0000-0000-0000-000000000004', 'a1000000-0000-0000-0000-000000000001', TRUE),
    ('b1000000-0000-0000-0000-000000000004', 'a1000000-0000-0000-0000-000000000002', FALSE),
    ('b1000000-0000-0000-0000-000000000004', 'a1000000-0000-0000-0000-000000000003', FALSE),
    
    -- Emma (Elite): data_analysis (primary), machine_learning
    ('b1000000-0000-0000-0000-000000000005', 'a1000000-0000-0000-0000-000000000003', TRUE),
    ('b1000000-0000-0000-0000-000000000005', 'a1000000-0000-0000-0000-000000000007', FALSE),
    
    -- Frank (Elite): mobile_dev (primary), devops
    ('b1000000-0000-0000-0000-000000000006', 'a1000000-0000-0000-0000-000000000002', TRUE),
    ('b1000000-0000-0000-0000-000000000006', 'a1000000-0000-0000-0000-000000000010', FALSE),
    
    -- Grace (Pro): web_dev (primary), ui_ux
    ('b1000000-0000-0000-0000-000000000007', 'a1000000-0000-0000-0000-000000000001', TRUE),
    ('b1000000-0000-0000-0000-000000000007', 'a1000000-0000-0000-0000-000000000009', FALSE),
    
    -- Henry (Pro): graphic_design (primary), video_editing
    ('b1000000-0000-0000-0000-000000000008', 'a1000000-0000-0000-0000-000000000004', TRUE),
    ('b1000000-0000-0000-0000-000000000008', 'a1000000-0000-0000-0000-000000000006', FALSE),
    
    -- Isabel (Pro): content_writing (primary), project_management
    ('b1000000-0000-0000-0000-000000000009', 'a1000000-0000-0000-0000-000000000005', TRUE),
    ('b1000000-0000-0000-0000-000000000009', 'a1000000-0000-0000-0000-000000000008', FALSE),
    
    -- Jack (Pro): devops (primary), web_dev
    ('b1000000-0000-0000-0000-000000000010', 'a1000000-0000-0000-0000-000000000010', TRUE),
    ('b1000000-0000-0000-0000-000000000010', 'a1000000-0000-0000-0000-000000000001', FALSE),
    
    -- Kate (Intermediate): web_dev (primary)
    ('b1000000-0000-0000-0000-000000000011', 'a1000000-0000-0000-0000-000000000001', TRUE),
    
    -- Leo (Intermediate): mobile_dev (primary)
    ('b1000000-0000-0000-0000-000000000012', 'a1000000-0000-0000-0000-000000000002', TRUE),
    
    -- Mia (Intermediate): data_analysis (primary), web_dev
    ('b1000000-0000-0000-0000-000000000013', 'a1000000-0000-0000-0000-000000000003', TRUE),
    ('b1000000-0000-0000-0000-000000000013', 'a1000000-0000-0000-0000-000000000001', FALSE),
    
    -- Noah (Intermediate): graphic_design (primary)
    ('b1000000-0000-0000-0000-000000000014', 'a1000000-0000-0000-0000-000000000004', TRUE),
    
    -- Olivia (Intermediate): content_writing (primary)
    ('b1000000-0000-0000-0000-000000000015', 'a1000000-0000-0000-0000-000000000005', TRUE),
    
    -- Paul (Beginner): web_dev (primary)
    ('b1000000-0000-0000-0000-000000000016', 'a1000000-0000-0000-0000-000000000001', TRUE),
    
    -- Quinn (Beginner): graphic_design (primary)
    ('b1000000-0000-0000-0000-000000000017', 'a1000000-0000-0000-0000-000000000004', TRUE),
    
    -- Ruby (Beginner): content_writing (primary)
    ('b1000000-0000-0000-0000-000000000018', 'a1000000-0000-0000-0000-000000000005', TRUE),
    
    -- Sam (Beginner): data_analysis (primary)
    ('b1000000-0000-0000-0000-000000000019', 'a1000000-0000-0000-0000-000000000003', TRUE),
    
    -- Tara (Beginner): video_editing (primary)
    ('b1000000-0000-0000-0000-000000000020', 'a1000000-0000-0000-0000-000000000006', TRUE),
    
    -- Uma (New): web_dev (primary)
    ('b1000000-0000-0000-0000-000000000021', 'a1000000-0000-0000-0000-000000000001', TRUE),
    
    -- Victor (New): mobile_dev (primary), web_dev
    ('b1000000-0000-0000-0000-000000000022', 'a1000000-0000-0000-0000-000000000002', TRUE),
    ('b1000000-0000-0000-0000-000000000022', 'a1000000-0000-0000-0000-000000000001', FALSE),
    
    -- Wendy (New): data_analysis (primary)
    ('b1000000-0000-0000-0000-000000000023', 'a1000000-0000-0000-0000-000000000003', TRUE)
ON CONFLICT DO NOTHING;


-- ─── User Behavior Metrics (diverse performance profiles) ───────────────────
-- Key metrics: acceptance_rate, median_response_seconds, push_open_rate, 
--              completion_rate, reliability_score, total_tasks_completed

INSERT INTO user_behavior_metrics (
    user_id, acceptance_rate, median_response_seconds, push_open_rate,
    completion_rate, reliability_score, total_tasks_completed, total_tasks_accepted
) VALUES
    -- Elite performers: High acceptance, fast response, excellent completion
    ('b1000000-0000-0000-0000-000000000004', 0.9500, 45,  0.9000, 0.9900, 98.50, 150, 158),
    ('b1000000-0000-0000-0000-000000000005', 0.9200, 60,  0.8500, 0.9800, 97.00, 120, 130),
    ('b1000000-0000-0000-0000-000000000006', 0.8800, 75,  0.8800, 0.9700, 96.50, 100, 114),
    
    -- Pro performers: Good metrics overall
    ('b1000000-0000-0000-0000-000000000007', 0.8500, 90,  0.8000, 0.9500, 92.00, 75, 88),
    ('b1000000-0000-0000-0000-000000000008', 0.8000, 120, 0.7500, 0.9300, 89.00, 60, 75),
    ('b1000000-0000-0000-0000-000000000009', 0.8300, 100, 0.7800, 0.9400, 91.00, 70, 84),
    ('b1000000-0000-0000-0000-000000000010', 0.7800, 110, 0.7200, 0.9200, 87.00, 55, 71),
    
    -- Intermediate: Moderate metrics, some variability
    ('b1000000-0000-0000-0000-000000000011', 0.7000, 150, 0.6500, 0.9000, 82.00, 35, 50),
    ('b1000000-0000-0000-0000-000000000012', 0.6500, 180, 0.6000, 0.8800, 78.00, 28, 43),
    ('b1000000-0000-0000-0000-000000000013', 0.7500, 130, 0.7000, 0.9100, 84.00, 40, 53),
    ('b1000000-0000-0000-0000-000000000014', 0.6000, 200, 0.5500, 0.8600, 75.00, 22, 37),
    ('b1000000-0000-0000-0000-000000000015', 0.7200, 160, 0.6800, 0.8900, 80.00, 32, 44),
    
    -- Beginners: Lower metrics, less experience
    ('b1000000-0000-0000-0000-000000000016', 0.5500, 240, 0.5000, 0.8200, 70.00, 10, 18),
    ('b1000000-0000-0000-0000-000000000017', 0.4500, 300, 0.4000, 0.7800, 65.00, 6, 13),
    ('b1000000-0000-0000-0000-000000000018', 0.5000, 280, 0.4500, 0.8000, 68.00, 8, 16),
    ('b1000000-0000-0000-0000-000000000019', 0.4000, 360, 0.3500, 0.7500, 60.00, 5, 12),
    ('b1000000-0000-0000-0000-000000000020', 0.6000, 220, 0.5500, 0.8400, 72.00, 12, 20),
    
    -- New users (cold start): Minimal history, below threshold (<5 completed)
    ('b1000000-0000-0000-0000-000000000021', 0.0000, 0,   0.0000, 1.0000, 100.00, 0, 0),   -- brand new
    ('b1000000-0000-0000-0000-000000000022', 1.0000, 90,  1.0000, 1.0000, 100.00, 2, 2),   -- 2 completed (cold start)
    ('b1000000-0000-0000-0000-000000000023', 0.7500, 120, 0.8000, 1.0000, 100.00, 4, 5)    -- 4 completed (cold start threshold)
ON CONFLICT DO NOTHING;


-- ─── Test Task for TURS Algorithm ────────────────────────────────────────────
-- A web_development task near center with medium budget
-- This should rank users based on TURS dimensions:
--   - SkillMatch: users with web_development skill
--   - BudgetCompatibility: MAB >= 250
--   - GeoRelevance: distance from (40.7128, -74.006) within 10km
--   - ExperienceFit: medium complexity → intermediate/pro preferred
--   - BehaviorIntent: higher acceptance/completion/reliability = better
--   - SpeedProbability: faster response time = better

INSERT INTO tasks (
    id, task_giver_id, category_id, budget, duration_hours, complexity_level,
    is_online, lat, lng, radius_km, state, priority_started_at, active_started_at, expires_at
) VALUES
    (
        'c1000000-0000-0000-0000-000000000100',
        'b1000000-0000-0000-0000-000000000001',  -- Alice creates the task
        'd1000000-0000-0000-0000-000000000001',
        250.00,    -- Medium budget
        6,         -- 6 hours
        'medium',  -- Medium complexity
        FALSE,     -- Physical task
        40.7128,   -- Center latitude
        -74.0060,  -- Center longitude
        10,        -- 10km radius
        'active',
        NOW() - INTERVAL '1 hour',
        NOW() - INTERVAL '30 minutes',
        NOW() + INTERVAL '23 hours'
    )
ON CONFLICT DO NOTHING;

INSERT INTO task_required_skills (task_id, skill_id, is_core) VALUES
    ('c1000000-0000-0000-0000-000000000100', 'a1000000-0000-0000-0000-000000000001', TRUE),  -- web_development (core)
    ('c1000000-0000-0000-0000-000000000003', 'a1000000-0000-0000-0000-000000000003', FALSE)  -- data_analysis (supporting)
ON CONFLICT DO NOTHING;


-- ─── Summary of Test Users ───────────────────────────────────────────────────
-- 
-- Users with web_development skill (should rank higher):
--   David   (Elite, ~0.3km, MAB=500, 95% accept, 99% complete)
--   Grace   (Pro, ~3km N, MAB=350, 85% accept, 95% complete)
--   Jack    (Pro, ~5km NW, MAB=280, 78% accept, 92% complete) 
--   Kate    (Intermediate, center, MAB=200, 70% accept, 90% complete)
--   Mia     (Intermediate, ~9km E, MAB=220, 75% accept, 91% complete)
--   Paul    (Beginner, center, MAB=120, 55% accept, 82% complete)
--   Uma     (New, ~0.3km, MAB=200, 0 history - cold start)
--   Victor  (New, ~1km, MAB=250, 2 completed - cold start)
--
-- Expected ranking (roughly):
--   1. David  - best combo of skill, proximity, behavior, experience
--   2. Grace  - strong metrics, close location
--   3. Kate   - center location, but lower experience
--   4. Jack   - good pro, but further away
--   5. Victor - cold start, but has skill + proximity
--   6. Uma    - cold start, close, but no history
--   7. Mia    - has skill, but too far (9km), budget might not match
--   8. Paul   - beginner at center, but low MAB (120 < 250)
