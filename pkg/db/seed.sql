-- ============================================================
-- XO Platform Seed Data
-- ============================================================
-- Deterministic UUIDs for easy cross-referencing:
--   Categories:  aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa0XX
--   Skills:      bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb0XX
--   Users:       cccccccc-cccc-cccc-cccc-cccccccccc0X
--   Tasks:       dddddddd-dddd-dddd-dddd-ddddddddd00X
-- ============================================================

BEGIN;

-- ============================================================
-- 1. TASK CATEGORIES (15)
-- ============================================================

INSERT INTO task_categories (id, name, description) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', 'Home Cleaning', 'House and apartment cleaning services'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', 'Plumbing', 'Pipe repair, faucet installation, drain clearing'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa003', 'Electrical Work', 'Wiring, outlet installation, lighting'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa004', 'Furniture Assembly', 'Assembling flat-pack and custom furniture'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa005', 'Moving Help', 'Packing, loading, unloading assistance'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa006', 'Lawn Care', 'Mowing, trimming, landscaping'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa007', 'Painting', 'Interior and exterior painting'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa008', 'Handyman', 'General repairs and maintenance'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa009', 'Tutoring', 'Academic and skill tutoring'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa00a', 'Pet Care', 'Pet sitting, walking, grooming'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa00b', 'Delivery', 'Package and item delivery'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa00c', 'Personal Shopping', 'Grocery and personal item shopping'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa00d', 'Event Help', 'Event setup, serving, cleanup'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa00e', 'Tech Support', 'Computer repair, network setup, software help'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa00f', 'Photography', 'Event, portrait, and product photography')
ON CONFLICT (id) DO NOTHING;


-- ============================================================
-- 2. SKILLS (20)
-- ============================================================

INSERT INTO skills (id, name) VALUES
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb001', 'Cleaning'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb002', 'Pipe Repair'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb003', 'Wiring'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb004', 'Assembly'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb005', 'Heavy Lifting'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb006', 'Mowing'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb007', 'Painting'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb008', 'General Repair'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb009', 'Teaching'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb00a', 'Animal Care'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb00b', 'Driving'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb00c', 'Shopping'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb00d', 'Event Setup'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb00e', 'Computer Repair'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb00f', 'Photography'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb010', 'Carpentry'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb011', 'Cooking'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb012', 'Organizing'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb013', 'Sewing'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb014', 'Gardening')
ON CONFLICT (id) DO NOTHING;


-- ============================================================
-- 3. TEST USERS (6)
-- ============================================================
-- Password for all: password123
-- Argon2 hash (gateway uses argon2, but seed uses bcrypt-style placeholder)
-- In practice, register via API; these are for DB-level testing only.

INSERT INTO users (id, email, password_hash, role) VALUES
    ('cccccccc-cccc-cccc-cccc-cccccccccc01', 'alice@test.com',   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'task_giver'),
    ('cccccccc-cccc-cccc-cccc-cccccccccc02', 'bob@test.com',     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'task_giver'),
    ('cccccccc-cccc-cccc-cccc-cccccccccc03', 'charlie@test.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'task_doer'),
    ('cccccccc-cccc-cccc-cccc-cccccccccc04', 'diana@test.com',   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'task_doer'),
    ('cccccccc-cccc-cccc-cccc-cccccccccc05', 'eve@test.com',     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'task_doer'),
    ('cccccccc-cccc-cccc-cccc-cccccccccc06', 'admin@test.com',   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin')
ON CONFLICT (id) DO NOTHING;


-- ============================================================
-- 4. USER PROFILES
-- ============================================================

INSERT INTO user_profiles (user_id, full_name, latitude, longitude, city, state, country, max_distance_km, preferred_budget_min, preferred_budget_max, is_online, onboarding_completed) VALUES
    -- Alice (task_giver) - San Francisco
    ('cccccccc-cccc-cccc-cccc-cccccccccc01', 'Alice Johnson', 37.7749, -122.4194, 'San Francisco', 'CA', 'US', 50, NULL, NULL, false, true),
    -- Bob (task_giver) - New York
    ('cccccccc-cccc-cccc-cccc-cccccccccc02', 'Bob Smith', 40.7128, -74.0060, 'New York', 'NY', 'US', 50, NULL, NULL, false, true),
    -- Charlie (task_doer, experienced) - San Francisco
    ('cccccccc-cccc-cccc-cccc-cccccccccc03', 'Charlie Brown', 37.7849, -122.4094, 'San Francisco', 'CA', 'US', 25, 30.00, 250.00, false, true),
    -- Diana (task_doer, moderate) - Austin
    ('cccccccc-cccc-cccc-cccc-cccccccccc04', 'Diana Prince', 30.2672, -97.7431, 'Austin', 'TX', 'US', 40, 20.00, 150.00, false, true),
    -- Eve (task_doer, cold-start) - Chicago
    ('cccccccc-cccc-cccc-cccc-cccccccccc05', 'Eve Williams', 41.8781, -87.6298, 'Chicago', 'IL', 'US', 30, 15.00, 100.00, false, true),
    -- Admin - Denver
    ('cccccccc-cccc-cccc-cccc-cccccccccc06', 'Admin User', 39.7392, -104.9903, 'Denver', 'CO', 'US', 50, NULL, NULL, false, true)
ON CONFLICT (user_id) DO NOTHING;


-- ============================================================
-- 5. USER SKILLS (task_doers only, 3-5 skills each)
-- ============================================================

-- Charlie: experienced handyman (5 skills)
INSERT INTO user_skills (user_id, skill_id, proficiency_level) VALUES
    ('cccccccc-cccc-cccc-cccc-cccccccccc03', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb008', 5),  -- General Repair
    ('cccccccc-cccc-cccc-cccc-cccccccccc03', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb004', 4),  -- Assembly
    ('cccccccc-cccc-cccc-cccc-cccccccccc03', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb007', 4),  -- Painting
    ('cccccccc-cccc-cccc-cccc-cccccccccc03', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb010', 3),  -- Carpentry
    ('cccccccc-cccc-cccc-cccc-cccccccccc03', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb005', 3)   -- Heavy Lifting
ON CONFLICT (user_id, skill_id) DO NOTHING;

-- Diana: cleaning & organizing specialist (4 skills)
INSERT INTO user_skills (user_id, skill_id, proficiency_level) VALUES
    ('cccccccc-cccc-cccc-cccc-cccccccccc04', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb001', 5),  -- Cleaning
    ('cccccccc-cccc-cccc-cccc-cccccccccc04', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb012', 4),  -- Organizing
    ('cccccccc-cccc-cccc-cccc-cccccccccc04', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb00c', 3),  -- Shopping
    ('cccccccc-cccc-cccc-cccc-cccccccccc04', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb011', 3)   -- Cooking
ON CONFLICT (user_id, skill_id) DO NOTHING;

-- Eve: new user, tech-oriented (3 skills)
INSERT INTO user_skills (user_id, skill_id, proficiency_level) VALUES
    ('cccccccc-cccc-cccc-cccc-cccccccccc05', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb00e', 3),  -- Computer Repair
    ('cccccccc-cccc-cccc-cccc-cccccccccc05', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb009', 2),  -- Teaching
    ('cccccccc-cccc-cccc-cccc-cccccccccc05', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb003', 2)   -- Wiring
ON CONFLICT (user_id, skill_id) DO NOTHING;


-- ============================================================
-- 6. USER BEHAVIOR METRICS (task_doers)
-- ============================================================

-- Charlie: experienced (25 completed, high reliability)
INSERT INTO user_behavior_metrics (
    user_id, total_tasks_completed, total_tasks_accepted, total_tasks_notified,
    total_reviews_received, average_response_time_minutes,
    completion_rate, acceptance_rate, reliability_score,
    average_review_score, consistency_score
) VALUES (
    'cccccccc-cccc-cccc-cccc-cccccccccc03',
    25, 28, 35,
    18, 3.5,
    0.8929, 0.8000, 85.00,
    4.50, 0.80
) ON CONFLICT (user_id) DO NOTHING;

-- Diana: moderate experience (12 completed, medium reliability)
INSERT INTO user_behavior_metrics (
    user_id, total_tasks_completed, total_tasks_accepted, total_tasks_notified,
    total_reviews_received, average_response_time_minutes,
    completion_rate, acceptance_rate, reliability_score,
    average_review_score, consistency_score
) VALUES (
    'cccccccc-cccc-cccc-cccc-cccccccccc04',
    12, 15, 22,
    8, 8.0,
    0.8000, 0.6818, 72.00,
    3.80, 0.65
) ON CONFLICT (user_id) DO NOTHING;

-- Eve: cold-start user (2 completed, limited data)
INSERT INTO user_behavior_metrics (
    user_id, total_tasks_completed, total_tasks_accepted, total_tasks_notified,
    total_reviews_received, average_response_time_minutes,
    completion_rate, acceptance_rate, reliability_score,
    average_review_score, consistency_score
) VALUES (
    'cccccccc-cccc-cccc-cccc-cccccccccc05',
    2, 3, 5,
    1, 15.0,
    0.6667, 0.6000, 50.00,
    4.00, 0.50
) ON CONFLICT (user_id) DO NOTHING;


-- ============================================================
-- 6b. GIVER BEHAVIOR METRICS (task_givers)
-- ============================================================

-- Alice: 3 tasks posted (tasks 1, 3, 5), 1 completed, 0 cancelled, 0 expired
INSERT INTO giver_behavior_metrics (
    user_id, total_tasks_posted, total_tasks_completed,
    total_tasks_cancelled, total_tasks_expired,
    avg_review_from_doers, total_reviews_from_doers,
    repost_count
) VALUES (
    'cccccccc-cccc-cccc-cccc-cccccccccc01',
    3, 1, 0, 0,
    0, 0,
    0
) ON CONFLICT (user_id) DO NOTHING;

-- Bob: 2 tasks posted (tasks 2, 4), 0 completed, 0 cancelled, 0 expired
INSERT INTO giver_behavior_metrics (
    user_id, total_tasks_posted, total_tasks_completed,
    total_tasks_cancelled, total_tasks_expired,
    avg_review_from_doers, total_reviews_from_doers,
    repost_count
) VALUES (
    'cccccccc-cccc-cccc-cccc-cccccccccc02',
    2, 0, 0, 0,
    0, 0,
    0
) ON CONFLICT (user_id) DO NOTHING;


-- ============================================================
-- 7. SAMPLE TASKS (5)
-- ============================================================

-- Task 1: PENDING - Home cleaning in San Francisco (Alice)
INSERT INTO tasks (id, created_by, title, description, budget, latitude, longitude, city, is_online, urgency, status, client_type, category_id, expires_at) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd001', 'cccccccc-cccc-cccc-cccc-cccccccccc01',
     'Deep clean 2BR apartment', 'Need thorough cleaning of a 2-bedroom apartment including kitchen and bathrooms',
     75.00, 37.7749, -122.4194, 'San Francisco', false, 'normal', 'pending', 'web',
     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', NOW() + INTERVAL '24 hours');

-- Task 2: PENDING - Furniture assembly in New York (Bob)
INSERT INTO tasks (id, created_by, title, description, budget, latitude, longitude, city, is_online, urgency, status, client_type, category_id, expires_at) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd002', 'cccccccc-cccc-cccc-cccc-cccccccccc02',
     'Assemble IKEA bookshelf and desk', 'Need help assembling a Billy bookcase and Malm desk',
     120.00, 40.7128, -74.0060, 'New York', false, 'normal', 'pending', 'mobile_android',
     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa004', NOW() + INTERVAL '24 hours');

-- Task 3: MATCHING - Online tech support (Alice)
INSERT INTO tasks (id, created_by, title, description, budget, is_online, urgency, status, client_type, category_id, expires_at) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd003', 'cccccccc-cccc-cccc-cccc-cccccccccc01',
     'Fix slow laptop', 'My laptop is running very slow, need someone to diagnose and fix remotely',
     50.00, true, 'high', 'matching', 'web',
     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa00e', NOW() + INTERVAL '12 hours');

-- Task 4: IN_PROGRESS - Painting in San Francisco (Bob, accepted by Charlie)
INSERT INTO tasks (id, created_by, title, description, budget, latitude, longitude, city, is_online, urgency, status, client_type, category_id, accepted_by, expires_at) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd004', 'cccccccc-cccc-cccc-cccc-cccccccccc02',
     'Paint living room walls', 'Living room needs two coats of paint, approximately 400 sqft',
     200.00, 37.7849, -122.4094, 'San Francisco', false, 'normal', 'in_progress', 'mobile_ios',
     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa007', 'cccccccc-cccc-cccc-cccc-cccccccccc03',
     NOW() + INTERVAL '48 hours');

-- Task 5: COMPLETED - Handyman work in San Francisco (Alice, completed by Charlie)
INSERT INTO tasks (id, created_by, title, description, budget, latitude, longitude, city, is_online, urgency, status, client_type, category_id, accepted_by, completed_at, expires_at) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd005', 'cccccccc-cccc-cccc-cccc-cccccccccc01',
     'Fix leaky faucet and door hinge', 'Kitchen faucet dripping and bedroom door hinge loose',
     90.00, 37.7749, -122.4194, 'San Francisco', false, 'normal', 'completed', 'mobile_android',
     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa008', 'cccccccc-cccc-cccc-cccc-cccccccccc03',
     NOW() - INTERVAL '2 days', NOW() + INTERVAL '24 hours');


-- ============================================================
-- 8. TASK REQUIRED SKILLS
-- ============================================================

-- Task 1 (Home Cleaning): Cleaning + Organizing
INSERT INTO task_required_skills (task_id, skill_id, minimum_proficiency) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd001', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb001', 3),
    ('dddddddd-dddd-dddd-dddd-ddddddddd001', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb012', 2);

-- Task 2 (Furniture Assembly): Assembly + Heavy Lifting
INSERT INTO task_required_skills (task_id, skill_id, minimum_proficiency) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd002', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb004', 3),
    ('dddddddd-dddd-dddd-dddd-ddddddddd002', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb005', 2);

-- Task 3 (Tech Support): Computer Repair + Wiring
INSERT INTO task_required_skills (task_id, skill_id, minimum_proficiency) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd003', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb00e', 3),
    ('dddddddd-dddd-dddd-dddd-ddddddddd003', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb003', 1);

-- Task 4 (Painting): Painting + General Repair
INSERT INTO task_required_skills (task_id, skill_id, minimum_proficiency) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd004', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb007', 3),
    ('dddddddd-dddd-dddd-dddd-ddddddddd004', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb008', 2);

-- Task 5 (Handyman): General Repair + Carpentry + Assembly
INSERT INTO task_required_skills (task_id, skill_id, minimum_proficiency) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd005', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb008', 4),
    ('dddddddd-dddd-dddd-dddd-ddddddddd005', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb010', 2),
    ('dddddddd-dddd-dddd-dddd-ddddddddd005', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb004', 2);


-- ============================================================
-- 9. TASK ACCEPTANCES (for in_progress & completed tasks)
-- ============================================================

INSERT INTO task_acceptances (task_id, user_id, status, responded_at) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd004', 'cccccccc-cccc-cccc-cccc-cccccccccc03', 'accepted', NOW() - INTERVAL '1 hour'),
    ('dddddddd-dddd-dddd-dddd-ddddddddd005', 'cccccccc-cccc-cccc-cccc-cccccccccc03', 'accepted', NOW() - INTERVAL '3 days');


-- ============================================================
-- 10. TASK NOTIFICATIONS (sample waves)
-- ============================================================

INSERT INTO task_notifications (task_id, user_id, wave_number, score, is_exploration, channel, status, sent_at) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd003', 'cccccccc-cccc-cccc-cccc-cccccccccc05', 1, 72.50, true, 'fcm', 'sent', NOW() - INTERVAL '30 minutes'),
    ('dddddddd-dddd-dddd-dddd-ddddddddd004', 'cccccccc-cccc-cccc-cccc-cccccccccc03', 1, 88.30, false, 'fcm', 'delivered', NOW() - INTERVAL '2 hours'),
    ('dddddddd-dddd-dddd-dddd-ddddddddd005', 'cccccccc-cccc-cccc-cccc-cccccccccc03', 1, 91.10, false, 'fcm', 'delivered', NOW() - INTERVAL '4 days');


-- ============================================================
-- 11. TASK STATE TRANSITIONS
-- ============================================================

INSERT INTO task_state_transitions (task_id, from_status, to_status, triggered_by) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd004', 'pending', 'matching', NULL),
    ('dddddddd-dddd-dddd-dddd-ddddddddd004', 'matching', 'in_progress', 'cccccccc-cccc-cccc-cccc-cccccccccc03'),
    ('dddddddd-dddd-dddd-dddd-ddddddddd005', 'pending', 'matching', NULL),
    ('dddddddd-dddd-dddd-dddd-ddddddddd005', 'matching', 'in_progress', 'cccccccc-cccc-cccc-cccc-cccccccccc03'),
    ('dddddddd-dddd-dddd-dddd-ddddddddd005', 'in_progress', 'completed', 'cccccccc-cccc-cccc-cccc-cccccccccc01');


-- ============================================================
-- 12. DEVICE TOKENS
-- ============================================================

INSERT INTO device_tokens (user_id, token, platform) VALUES
    ('cccccccc-cccc-cccc-cccc-cccccccccc03', 'fcm_token_charlie_android_001', 'android'),
    ('cccccccc-cccc-cccc-cccc-cccccccccc04', 'fcm_token_diana_ios_001', 'ios'),
    ('cccccccc-cccc-cccc-cccc-cccccccccc05', 'fcm_token_eve_android_001', 'android')
ON CONFLICT (user_id, token) DO NOTHING;


-- ============================================================
-- 13. CONVERSATIONS (for in_progress task)
-- ============================================================

INSERT INTO conversations (task_id, participant_a, participant_b) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd004', 'cccccccc-cccc-cccc-cccc-cccccccccc02', 'cccccccc-cccc-cccc-cccc-cccccccccc03');


-- ============================================================
-- 14. SAMPLE REVIEW (for completed task)
-- ============================================================

INSERT INTO task_reviews (task_id, reviewer_id, reviewee_id, reviewer_role, rating, comment) VALUES
    ('dddddddd-dddd-dddd-dddd-ddddddddd005', 'cccccccc-cccc-cccc-cccc-cccccccccc01', 'cccccccc-cccc-cccc-cccc-cccccccccc03', 'task_giver', 5, 'Charlie did an excellent job! Fixed everything quickly.');


COMMIT;
