# XO — Task Matching Platform

## What This Is

XO is a task matching platform. Task givers post tasks, the system scores and ranks eligible task doers using the TURS algorithm, then delivers wave-based notifications via an orchestrator. Includes SLM-powered chat moderation, social OAuth, bidirectional reviews, and graduated cold-start warmup.

## Current Status (2026-03-15)

Branch: `fresh-start`. Core platform is feature-complete for MVP.

### Implemented
- **xo service**: TURS scoring (6 dimensions + warmup), wave-based orchestrator, multi-channel notifications (FCM/WebPush/InApp), SLM chat moderation + task categorization, chat service, bidirectional reviews, nearby search, category management
- **gateway service**: JWT auth with refresh tokens, argon2 passwords, Google + Facebook OAuth (web + mobile), user profiles, onboarding flow (experience/education/certificates/languages/skills), WebSocket chat relay, proxy to xo for all task/chat/review/notification/category routes
- **database**: ~25 tables, UUID PKs, updated_at triggers, indexes, seed data
- **docker**: Full compose stack with health checks

### Not Yet Implemented
- Frontend client
- CI/CD pipeline
- Production secrets management
- Rate limiting / abuse prevention
- Monitoring, logging, alerting (observability)
- Load testing / performance benchmarks
- Email notifications
- Payment / escrow integration
- Admin dashboard
- File/image upload (avatars, task attachments)
- Search / filtering enhancements (full-text, faceted)

## Architecture

### Services
| Service | Stack | Port | Purpose |
|---------|-------|------|---------|
| **xo** | Go 1.23, net/http, lib/pq | 8080 | Core engine — TURS scoring, orchestrator, notifications, chat, reviews, SLM |
| **gateway** | Python 3.12, FastAPI, asyncpg, PyJWT | 8000 | Auth (JWT + OAuth), profiles, WebSocket, proxy to xo |
| **postgres** | PostgreSQL 16 | 5432 | Shared database |
| **ollama** | Ollama (phi4-mini 3.8B) | 11434 | Chat moderation + task categorization |

### System Topology
Frontend → Gateway (HTTPS/WSS) → xo (HTTP internal) → Ollama (HTTP internal). Both services → Postgres (TCP).

## TURS Algorithm (Task-User Relevance Score)

### Scoring Dimensions
| Dimension | Weight | Source |
|-----------|--------|--------|
| SkillMatch | 0.30 | `internal/matching/turs.go` |
| BudgetCompatibility | 0.25 | `internal/matching/turs.go` |
| GeoRelevance | 0.15 | `internal/matching/turs.go` — redistributed for online tasks |
| ExperienceFit | 0.15 | `internal/matching/turs.go` |
| BehaviorIntent | 0.10 | `internal/matching/turs.go` |
| SpeedProbability | 0.05 | `internal/matching/turs.go` |

Online tasks redistribute GeoRelevance (15%) proportionally across other dimensions.

### Warmup (Cold-Start)
```
WarmupFactor = max(0, 1.0 - (completed_tasks / 20))
BehaviorIntent floor = 0.5 * WarmupFactor
Score boost = 5.0 * WarmupFactor
```
Source: `internal/matching/warmup.go`

### Orchestrator Waves
- Interval: 60s, size: 15 users/wave
- 15% exploration slots for cold-start users (warmupFactor > 0)
- Channels: FCM → WebPush → InApp (in priority order)
- Source: `internal/orchestrator/orchestrator.go`

## Go Package Layout
```
cmd/xo/main.go                     # Entry point
internal/api/                       # HTTP handlers (server, middleware, response, task/chat/review/device/notification/category handlers)
internal/matching/                  # TURS scoring (turs.go, types.go, weights.go, warmup.go)
internal/orchestrator/              # Wave-based matching pipeline
internal/notification/              # Notifier interface + FCM/WebPush/InApp dispatchers
internal/slm/                       # Ollama client, moderator, categorizer, prompts
internal/chat/                      # Chat service + types
internal/review/                    # Review service + metric recalculation
pkg/db/                             # schema.sql, seed.sql, queries.sql, sqlc.yaml
```

## Gateway Layout
```
app/main.py                         # FastAPI app, lifespan, middleware
app/config.py, database.py, auth.py, deps.py
app/oauth/                          # Google + Facebook OAuth (redirect, callback, mobile token)
app/routers/                        # auth, oauth, profile, tasks, chat, reviews, categories, notifications, nearby, onboarding
app/schemas/                        # Pydantic models + envelope helpers
app/ws/                             # WebSocket manager + chat relay
```

## Database Schema Summary

25 tables across auth, profiles, tasks, matching, chat, reviews, notifications:

**Auth**: `users`, `user_auth_providers`, `refresh_tokens`, `otp_codes`
**Profiles**: `user_profiles`, `user_skills`, `skills`, `user_behavior_metrics`, `experience_multiplier_history`, `user_experience`, `user_education`, `user_certificates`, `user_languages`
**Tasks**: `tasks`, `task_categories`, `task_required_skills`, `task_acceptances`, `task_notifications`, `task_state_transitions`
**Chat**: `conversations`, `chat_messages`
**Reviews**: `task_reviews`, `disputes`
**Notifications**: `device_tokens`, `web_push_subscriptions`, `inapp_notifications`

Schema: `pkg/db/schema.sql` | Seed: `pkg/db/seed.sql` | Queries: `pkg/db/queries.sql`

## Key Dependencies

### Go (go.mod)
- `github.com/google/uuid` — UUID generation
- `github.com/lib/pq` — PostgreSQL driver
- `firebase.google.com/go/v4` — FCM push notifications

### Python (gateway)
- `fastapi`, `uvicorn` — HTTP/WebSocket server
- `asyncpg` — async PostgreSQL
- `pyjwt` — JWT tokens
- `argon2-cffi` — password hashing
- `httpx` — HTTP client for proxying to xo
- `python-multipart` — form data parsing
- `websockets` — WebSocket protocol

## Conventions

### Go
- Standard library `net/http` with `ServeMux` — no frameworks
- `lib/pq` for PostgreSQL — not pgx
- `sqlc` for type-safe query generation
- UUIDs via `github.com/google/uuid`
- Package layout: `cmd/`, `internal/`, `pkg/`

### Python (Gateway)
- FastAPI with asyncpg
- PyJWT for token handling, argon2 for passwords
- httpx for proxying to xo service

### Database
- PostgreSQL 16
- `uuid-ossp` extension for UUID generation
- All tables use UUID primary keys
- `updated_at` triggers on mutable tables
- Dev credentials: `xo/xo/xo` (user/pass/db)

### API Response Format
- Success: `{"data": {...}}` or `{"data": [...], "cursor": {"next": "...", "has_more": true}}`
- Error: `{"error": {"code": "...", "message": "...", "details": [...]}}`

### Docker
- `docker-compose.yml` orchestrates: postgres, ollama, ollama-init, xo, gateway
- Schema + seed mounted as postgres init scripts
- Health checks on postgres and ollama before dependent services start

## Deployment Strategy

### Budget
INR 50,000–75,000/month (~$540–$810 USD)

### Recommended Tiers
| Tier | Cost | Setup |
|------|------|-------|
| **1 — MVP** | ~$107/INR 9,900/mo | AWS Mumbai: t3.small (xo + gateway), t3.medium (Ollama CPU), RDS db.t3.small |
| **2 — GPU Value** | ~$270/INR 25K/mo | AWS Mumbai (app + DB) + Hetzner GEX44 (RTX 4000 Ada for inference) |
| **3 — Scale** | ~$530/INR 49K/mo | All-AWS Mumbai: ECS/EC2, g4dn.xlarge GPU, Multi-AZ RDS |

Start with Tier 1. CPU inference (6–10 tok/sec on 4 vCPU) is adequate for <10 QPS moderation at early stage. Detailed analysis in `CLOUD_HOSTING_ANALYSIS.md`.

### Production Checklist
- [ ] Replace `JWT_SECRET` with strong random secret
- [ ] Set `CORS_ORIGINS` to actual frontend domain(s)
- [ ] Configure `GOOGLE_APPLICATION_CREDENTIALS` for FCM
- [ ] Set Google/Facebook OAuth client IDs and secrets
- [ ] Enable SSL/TLS termination at load balancer
- [ ] Set up database backups (automated snapshots)
- [ ] WebSocket-compatible load balancer/reverse proxy
- [ ] Container registry for xo and gateway images
- [ ] Health check endpoints for orchestration
- [ ] Log aggregation and error alerting

## Commands

```bash
# Full stack
docker compose up -d

# Fresh start (wipe DB)
docker compose down -v && docker compose up -d

# Go tests
go test ./...

# Gateway tests (in venv)
cd gateway && source .venv/bin/activate && pytest

# Connect to DB
docker compose exec postgres psql -U xo -d xo
```

## Rules

- Always run `go test ./...` after modifying Go code
- Always run `pytest` after modifying gateway code
- Never commit secrets, `.env` files, or Firebase service account keys
- Use transactions for multi-step DB operations (see orchestrator pattern)
- Test TURS changes against the seed data scenarios
- Keep sqlc queries in `pkg/db/queries.sql` and regenerate with `sqlc generate`
- Gateway is the SINGLE entry point — xo `/internal/*` routes are not exposed to frontend
- `X-Client-Type` header (`web`, `mobile_android`, `mobile_ios`) enforces offline task restrictions
- Chat unlocks after task acceptance (conversation auto-created in AcceptTask)
- Reviews are independent and immediately visible; either party can review post-completion
