# Active Context

## Current Work
API Gateway (Python FastAPI) fully implemented and tested. Docker compose integration verified.

## Key Decisions
- Go 1.25 net/http ServeMux for xo routing (no third-party router)
- Online tasks: GeoRelevance weight = 0, redistributed proportionally to remaining 5 dimensions
- Cold-start: users with < 5 completed tasks get exploration slots (15% per wave) + behavior-intent floor of 0.5
- Notification: interface-based (LogNotifier for dev, WebhookNotifier for webhook, FCMNotifier for push)
- FCM: integrated directly into xo via FCM HTTP v1 API + golang.org/x/oauth2/google ADC
- FCM auth: GOOGLE_APPLICATION_CREDENTIALS + FCM_PROJECT_ID env vars
- Device tokens: stored in device_tokens table with user_id + token + platform (android/ios)
- Stale token cleanup: automatic on FCM NOT_FOUND/UNREGISTERED errors
- EM update happens on POST /tasks/{id}/complete using adaptive alpha (0.20 → 0.10 → 0.05)
- Task states: priority → active → accepted → completed (also: expired, cancelled)

### Gateway Decisions
- Python 3.12, FastAPI 0.115, Uvicorn, asyncpg, PyJWT (HS256), argon2-cffi, httpx
- Gateway lives inside xo repo at gateway/
- Shares same Postgres DB (separate init file: 03-gateway-schema.sql)
- Gateway owns: auth, profile, onboarding, verification, payments, location, uploads, dashboard, config
- Gateway proxies: task CRUD + device tokens → xo service via httpx
- Role mapping: frontend giver→task_giver, doer→task_doer, both→both
- OTP: stub (logs to stdout) — real SMS/email provider integration later
- File uploads: local filesystem in dev (UPLOAD_DIR env var) — S3 later
- No Alembic yet — raw SQL init matching xo's approach
- All API responses wrapped in {success, data, error, message} envelope

## Files Being Modified
### xo service (Go)
- pkg/db/schema.sql — core DB schema (users, tasks, etc.)
- pkg/db/queries.sql — sqlc queries
- internal/matching/ — TURS scoring engine
- internal/api/ — REST API (server.go, handler.go, response.go)
- internal/notification/ — FCM + webhook notifiers
- internal/orchestrator/ — task lifecycle orchestrator
- cmd/xo/main.go — HTTP server entry point

### Gateway (Python)
- gateway/schema.sql — 17 gateway-owned tables
- gateway/Dockerfile — Python 3.12 multi-stage build
- gateway/requirements.txt — pinned dependencies
- gateway/app/main.py — FastAPI app with lifespan, middleware, routers
- gateway/app/config.py — Pydantic BaseSettings
- gateway/app/database.py — asyncpg connection pool
- gateway/app/auth.py — JWT token creation/verification
- gateway/app/deps.py — FastAPI dependencies (get_db, get_current_user)
- gateway/app/schemas/ — Pydantic models (envelope, auth, profile)
- gateway/app/routers/auth.py — register, login, OTP, refresh, logout
- gateway/app/routers/profile.py — profile CRUD + 7-step onboarding
- gateway/app/routers/verification.py — verification + payment methods
- gateway/app/routers/location.py — location + addresses
- gateway/app/routers/config.py — categories + FAQs
- gateway/app/routers/tasks.py — xo proxy (task CRUD + devices)
- gateway/app/routers/dashboard.py — aggregated user stats
- gateway/tests/ — 67 tests (conftest, auth, profile, endpoints, envelope, health, tasks_proxy, edge_cases)

### Infrastructure
- docker-compose.yml — postgres + xo + gateway services
- .dockerignore — excludes gateway venv
