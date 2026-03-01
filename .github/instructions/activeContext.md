# Active Context

## Current Focus

All known gaps from initial session have been implemented.

---

## Project State Summary

**Module:** `xo` (Go 1.25.0)  
**Dependencies:** `github.com/google/uuid v1.6.0`, `github.com/lib/pq v1.10.9`  
**DB Layer:** PostgreSQL via sqlc v1.30.0  

### Completed Components

| Component | Location | Status |
|---|---|---|
| Database schema | `pkg/db/schema.sql` | Complete (10 tables, triggers, indexes) |
| SQL queries | `pkg/db/queries.sql` | Complete (CRUD, hard filter, metrics) |
| sqlc generated code | `pkg/db/db/` | Complete (models, queries, db interface) |
| TURS types | `internal/matching/types.go` | Complete (TaskInput, CandidateInput w/ Skills, ScoreBreakdown w/ GeoRelevance) |
| TURS service interface | `internal/matching/service.go` | Complete (ScoreCandidate, RankCandidates) |
| TURS weights | `internal/matching/weights.go` | Complete (6 weight factors) |
| TURS scoring engine | `internal/matching/turs.go` | Complete (all 6 components including skillMatch + geoRelevance) |
| Orchestrator | `internal/orchestrator/orchestrator.go` | Complete (wave-based priority flow, AcceptTask) |
| Entry point | `cmd/xo/main.go` | Complete (graceful shutdown, env-based config) |
| Unit tests | `internal/matching/turs_test.go` | Complete (16 tests, all passing) |

### Previously Known Gaps — Now Resolved

1. **`skillMatch()`** — Real skill intersection logic (primary +10, each additional +6, cap 30, normalised 0–1)
2. **GeoRelevance** — Haversine-based scoring for offline tasks; neutral 0.5 for online; `ScoreBreakdown.GeoRelevance` field added
3. **Orchestrator** — Wave scheduling (15/wave, 60s intervals, 10min window), stop-on-accept, move-to-active fallback, transactional AcceptTask
4. **Entry point** — `cmd/xo/main.go` with graceful signal handling
5. **Tests** — 16 unit tests covering all TURS components

---

## Architecture Decisions

- PostgreSQL as single source of truth (no Redis/event bus yet)
- sqlc for type-safe query generation
- Stateless TURS scoring engine (pure functions, versionable weights)
- Atomic task acceptance via `SELECT FOR UPDATE` + `UNIQUE(task_id)` constraint
- Wave-based priority broadcasting (15 users/wave, 60s intervals, 10min window)
- lib/pq as the postgres database/sql driver

---

## Pending Phase 4 Enhancements

1. Exploration injection (10-15% randomness)
2. Crash-safe timeout recovery
3. Event-driven architecture
4. Redis candidate caching
5. Proper geo scoring for online tasks (timezone/language)
6. Structured logging
7. Metrics instrumentation
8. Load testing
9. Batch skill fetching in orchestrator (currently N+1 per candidate)

---

## Last Updated

Phase 3 implementation complete — all MVP gaps closed.

