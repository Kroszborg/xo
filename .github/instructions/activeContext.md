# Active Context

## Current Focus

Session initialized. No active task in progress.

---

## Project State Summary

**Module:** `xo` (Go 1.25.0)  
**Dependency:** `github.com/google/uuid v1.6.0`  
**DB Layer:** PostgreSQL via sqlc v1.30.0  

### Completed Components

| Component | Location | Status |
|---|---|---|
| Database schema | `pkg/db/schema.sql` | Complete (10 tables, triggers, indexes) |
| SQL queries | `pkg/db/queries.sql` | Complete (CRUD, hard filter, metrics) |
| sqlc generated code | `pkg/db/db/` | Complete (models, queries, db interface) |
| TURS types | `internal/matching/types.go` | Complete (TaskInput, CandidateInput, ScoreBreakdown) |
| TURS service interface | `internal/matching/service.go` | Complete (ScoreCandidate, RankCandidates) |
| TURS weights | `internal/matching/weights.go` | Complete (6 weight factors) |
| TURS scoring engine | `internal/matching/turs.go` | Mostly complete (see gaps below) |

### Known Gaps in Current Code

1. **`skillMatch()` is stubbed** — always returns 1.0 (no actual skill comparison logic)
2. **GeoRelevance not implemented** — weight exists (0.15) but no scoring function and `ScoreBreakdown` is missing the field
3. **No orchestrator code in workspace** — architecture doc describes it but no `cmd/` or orchestration package exists
4. **No `main.go` or entry point** — project has no runnable binary yet
5. **No tests** — no `_test.go` files anywhere

### Database Tables

- `users`, `user_profiles`, `skills`, `user_skills`
- `user_behavior_metrics`, `experience_multiplier_history`
- `tasks`, `task_required_skills`, `task_acceptances`
- `task_notifications`, `task_state_transitions`

---

## Architecture Decisions

- PostgreSQL as single source of truth (no Redis/event bus yet)
- sqlc for type-safe query generation
- Stateless TURS scoring engine (pure functions, versionable weights)
- Atomic task acceptance via `SELECT FOR UPDATE` + `UNIQUE(task_id)` constraint
- Wave-based priority broadcasting (15 users/wave, 60s intervals, 10min window)

---

## Pending Phase 4 Enhancements

1. Exploration injection (10-15% randomness)
2. Crash-safe timeout recovery
3. Event-driven architecture
4. Redis candidate caching
5. Proper geo scoring (Haversine distance)
6. Structured logging
7. Metrics instrumentation
8. Load testing

---

## Last Updated

Session start — full codebase exploration complete.
