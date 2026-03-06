# XO Architecture

## Overview

XO is a task matching platform with a Go backend (xo service) and Python FastAPI gateway.

## Services

### xo (Go 1.25)
- **Location**: Root directory
- **Port**: 8080
- **Purpose**: Core task matching, TURS algorithm, orchestrator, notifications
- **Stack**: net/http ServeMux, pgx/v5, sqlc

### Gateway (Python 3.12)
- **Location**: `gateway/`
- **Port**: 8000  
- **Purpose**: User authentication, profile management, proxy to xo
- **Stack**: FastAPI 0.115, asyncpg 0.30.0, PyJWT, bcrypt

### PostgreSQL
- **Image**: postgres:16-alpine
- **Port**: 5432
- **User/Pass/DB**: xo/xo/xo

## Key Components

### TURS Algorithm (Task-User Relevance Score)
- **Location**: `internal/matching/`
- **6 Dimensions**:
  1. SkillMatch (30%): +10 primary, +6 additional, max 30
  2. BudgetCompatibility (25%): ratio = taskBudget/userMAB
  3. GeoRelevance (15%): Within radius=1.0, 1-2x=0.5, beyond=0
  4. ExperienceFit (15%): Budget < $1000 favors intermediate/beginner
  5. BehaviorIntent (10%): (accept*0.6 + completion*0.3 + reliability*0.1)
  6. SpeedProbability (5%): 1 - (responseTime/300)

### Cold-Start Handling
- Threshold: < 5 completed tasks
- Behavior floor: 0.5 (ensures new users remain competitive)
- 15% cold-start exploration in priority waves

### Orchestrator
- **Location**: `internal/orchestrator/`
- **Purpose**: Wave-based task notification system
- **Waves**: 3 waves x 3 users each = 9 initial candidates
- **Priority**: Uses TURS ranking, cold-start gets 1 slot per wave

## Database Schema

### xo Schema (`pkg/db/schema.sql`)
- `users`, `user_profiles`, `user_behavior_metrics`
- `skills`, `user_skills`
- `tasks`, `task_required_skills`, `task_acceptances`
- `device_tokens`, `task_notifications`

### Gateway Schema (`gateway/schema.sql`)
- `gateway_user_profile` (first/last name, avatar)
- `user_addresses`, `user_payment_methods`
- `user_certificates`, `user_education`, `user_experience`
- `user_onboarding`, `password_reset_tokens`, `email_change_requests`

## File Structure

```
xo/
├── cmd/xo/main.go           # Entry point
├── internal/
│   ├── api/                 # HTTP handlers
│   ├── matching/            # TURS algorithm
│   │   ├── turs.go          # Scoring logic
│   │   ├── types.go         # TaskInput, CandidateInput
│   │   ├── weights.go       # Default weights
│   │   └── turs_test.go     # Unit tests
│   ├── notification/        # Push notification service
│   └── orchestrator/        # Wave-based task distribution
├── pkg/db/
│   ├── schema.sql           # xo database schema
│   ├── seed.sql             # Base seed data
│   └── turs_test_seed.sql   # 20 test users for TURS testing
├── gateway/
│   ├── main.py              # FastAPI app entry
│   ├── config.py            # Settings
│   ├── schema.sql           # Gateway-specific tables
│   └── tests/               # pytest tests (67 passing)
└── docker-compose.yml       # Full stack orchestration
```

## Testing

- **Go tests**: `go test ./...`
- **Gateway tests**: `cd gateway && pytest`
- **Full stack**: `docker compose up`

## Environment Variables

### xo
- `DATABASE_URL`: PostgreSQL connection string
- `LISTEN_ADDR`: Server address (default `:8080`)
- `FCM_PROJECT_ID`: Firebase project for push notifications

### Gateway
- `DATABASE_URL`: PostgreSQL connection string  
- `XO_SERVICE_URL`: Internal xo service URL
- `JWT_SECRET`: JWT signing key
