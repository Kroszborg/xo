# XO — Task Matching Platform (Backend Dev's Final Version)

## What This Is

XO is a task matching platform. Task givers post tasks, the system scores and ranks eligible task doers using the TURS/Relevancy algorithm, then delivers wave-based notifications. Gateway handles JWT auth, profiles, onboarding, OTP, payments, addresses, verification, and proxies tasks to the xo service.

## Current Status

This is the backend dev's final version (`programming/xo`), merged in to replace the previous version.

### Implemented
- **xo service**: TURS scoring, relevancy engine, wave-based orchestrator, FCM notifications, device registration, nearby search
- **gateway service**: JWT auth with refresh tokens + OTP, argon2 passwords, user profiles, onboarding flow (steps 1–7), verification/KYC, payment methods CRUD, addresses CRUD, location tracking, dashboard stats, avatar upload, proxy to xo for task routes
- **database**: schema.sql (xo tables) + gateway/schema.sql (gateway tables), seed.sql, turs_test_seed.sql
- **docker**: Full compose stack with health checks (no ollama)

### Not Yet Implemented
- Chat / WebSocket relay
- Social OAuth (Google, Facebook)
- Reviews / disputes
- Notifications push (FCM project ID not configured)
- Rate limiting / abuse prevention

## Architecture

### Services
| Service | Stack | Port | Purpose |
|---------|-------|------|---------|
| **xo** | Go 1.23, net/http | 8080 | Core engine — TURS scoring, orchestrator, notifications, nearby |
| **gateway** | Python 3.12, FastAPI, asyncpg, PyJWT | 8000 | Auth (JWT+OTP), profiles, onboarding, verification, payments, proxy to xo |
| **postgres** | PostgreSQL 16 | 5432 | Shared database |

### System Topology
Frontend → Gateway (HTTPS) → xo (HTTP internal) → Postgres

## Go Package Layout
```
cmd/xo/main.go                     # Entry point
internal/api/                       # HTTP handlers (server, handler, response, device_handler)
internal/matching/                  # TURS matching
internal/relevancy/                 # Relevancy scoring engine
internal/orchestrator/              # Wave-based matching pipeline (offline/online)
internal/notification/              # Notifier interface + FCM
pkg/db/                             # schema.sql, seed.sql, queries.sql, sqlc.yaml, turs_test_seed.sql
```

## Gateway Layout
```
app/main.py                         # FastAPI app, lifespan, middleware
app/config.py, database.py, auth.py, deps.py
app/routers/                        # auth, config, dashboard, location, profile, tasks, verification
app/schemas/                        # Pydantic models + envelope helpers
gateway/schema.sql                  # Gateway-owned tables (loaded as 03-gateway-schema.sql)
```

## Database Schema
- `pkg/db/schema.sql` — xo core tables (users, tasks, matching, notifications, devices)
- `pkg/db/seed.sql` — seed data
- `gateway/schema.sql` — gateway tables (refresh_tokens, otp_codes, gateway_user_profile, user_core_skills, user_other_skills, user_experience, user_education, user_certificates, user_languages, user_payment_methods, user_addresses, user_location, user_verification, categories, faqs, file_uploads)
- Loaded in order: 01-schema.sql → 02-seed.sql → 03-gateway-schema.sql → 04-turs-test-seed.sql

## Key Dependencies

### Go (go.mod)
- `github.com/google/uuid` — UUID generation
- `github.com/lib/pq` — PostgreSQL driver
- `firebase.google.com/go/v4` — FCM push notifications

### Python (gateway)
- `fastapi[standard]`, `uvicorn[standard]` — HTTP server
- `asyncpg` — async PostgreSQL
- `pyjwt`, `argon2-cffi` — auth
- `httpx` — proxy to xo
- `python-multipart` — file uploads
- `pydantic[email]`, `pydantic-settings` — validation + config

## Commands

```bash
# Full stack
docker compose up -d

# Fresh start (wipe DB)
docker compose down -v && docker compose up -d

# Production (with Caddy)
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Go tests
go test ./...

# Connect to DB
docker compose exec postgres psql -U xo -d xo
```

## Environment Variables

### xo service
- `DATABASE_URL` — postgres connection string
- `LISTEN_ADDR` — e.g. `:8080`
- `NOTIFICATION_WEBHOOK_URL` — external push service (empty = log to stdout)
- `FCM_PROJECT_ID` — Firebase project ID
- `GOOGLE_APPLICATION_CREDENTIALS` — path to Firebase service account JSON

### gateway service
- `DATABASE_URL` — postgres connection string (postgresql:// format)
- `XO_SERVICE_URL` — e.g. `http://xo:8080`
- `JWT_SECRET` — secret key
- `JWT_ALGORITHM` — `HS256`
- `JWT_ACCESS_TOKEN_MINUTES` — `30`
- `JWT_REFRESH_TOKEN_DAYS` — `30`
- `LISTEN_HOST` — `0.0.0.0`
- `LISTEN_PORT` — `8000`
- `UPLOAD_DIR` — `/app/uploads`
- `CORS_ORIGINS` — JSON array string, e.g. `'["*"]'`

## Rules

- Always run `go test ./...` after modifying Go code
- Never commit secrets, `.env` files, or Firebase service account keys
- Gateway is the SINGLE entry point — xo routes are not exposed to frontend
- OTP is a stub (logs to stdout) — wire up real SMS/email provider before production
