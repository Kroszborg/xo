# Implementation Steps

## Phase 1: Architecture Redesign (xo)
- [x] Design new architecture (REST API, dynamic weights, cold-start, notification interface)
- [x] Update architecture.md

## Phase 2: Database Changes (xo)
- [x] Add 'completed' state to tasks CHECK constraint + completed_at column
- [x] Add new queries: CancelTask, CompleteTask, ListTasks, GetNewUsersWithSkills, UpdateTask, GetTaskAcceptance
- [x] Regenerate sqlc

## Phase 3: Matching Engine Updates (xo)
- [x] Dynamic weights (online: redistribute geo weight proportionally)
- [x] Cold-start behavior-intent floor for new users (IsNewUser flag)
- [x] Update tests

## Phase 4: Notification Interface (xo)
- [x] Create Notifier interface
- [x] LogNotifier (development)
- [x] WebhookNotifier (production)
- [x] FCMNotifier (push notifications via FCM HTTP v1)

## Phase 5: REST API Layer (xo)
- [x] response.go — JSON response helpers
- [x] server.go — HTTP server, routes, middleware
- [x] handler.go — Task CRUD + accept + complete handlers

## Phase 6: Orchestrator Redesign (xo)
- [x] Integrate cold-start exploration slots (15% of wave)
- [x] Use Notifier interface instead of fmt.Printf
- [x] Integrate with API (called on task creation)

## Phase 7: Task Completion + EM Update (xo)
- [x] EM update formula with adaptive learning rate
- [x] Persist EM history + behavior metric updates

## Phase 8: Entry Point (xo)
- [x] Rewrite main.go as HTTP server with graceful shutdown

## Phase 9: Testing (xo)
- [x] Unit tests for matching (dynamic weights, cold-start)
- [x] Compilation check
- [x] Run all existing + new tests — 29/29 passing

## Gateway Implementation Phases (Python FastAPI)

### Phase 0: Scaffold
- [x] gateway/ directory, Dockerfile, requirements.txt
- [x] gateway/schema.sql — 17 gateway-owned tables
- [x] docker-compose.yml — gateway service + schema mount

### Phase 1: DB Layer
- [x] config.py (Pydantic BaseSettings)
- [x] database.py (asyncpg pool with lifespan)
- [x] schemas/envelope.py (ok/err/paginated helpers)
- [x] deps.py (get_db, get_current_user, CurrentUser, DBConn)

### Phase 2: Auth
- [x] auth.py (JWT create/decode, refresh token hashing)
- [x] routers/auth.py (register, login, send-otp, verify-otp, refresh, logout)
- [x] Role mapping: giver→task_giver, doer→task_doer

### Phase 3: Middleware
- [x] CORS, GZip, RequestID middleware
- [x] Validation error handler (envelope format)
- [x] Catch-all exception handler

### Phase 4: Profile & Onboarding
- [x] routers/profile.py (GET/PATCH profile, avatar upload, onboarding status)
- [x] 7-step onboarding endpoints (role, skills, experience, education, certificates, languages, bio)

### Phase 5: Verification & Payments
- [x] routers/verification.py (verification status, ID document upload)
- [x] Payment method CRUD (list, create, update, delete)

### Phase 6: Location/Uploads/Config
- [x] routers/location.py (PUT/GET location, address CRUD)
- [x] routers/config.py (categories, FAQs — public endpoints)
- [x] Syncs location to xo user_profiles.fixed_lat/lng

### Phase 7: xo Proxy
- [x] routers/tasks.py — httpx proxy for task CRUD + device tokens
- [x] page/limit → limit/offset translation

### Phase 8: Dashboard
- [x] routers/dashboard.py — aggregated stats across gateway + xo tables

### Phase 9: Testing
- [x] 29/29 gateway tests passing
- [x] xo tests: 29/29 still passing
- [x] xo build/vet: clean
