# Active Context

## Current Session Focus

Testing the TURS (Task-User Relevance Score) algorithm with realistic seed data.

## Recent Changes

1. **Created TURS test seed** (`pkg/db/turs_test_seed.sql`):
   - 20 users with diverse profiles
   - 4 additional skills (machine_learning, project_management, ui_ux_design, devops)
   - 1 test task (web_development, $250, medium complexity, 10km radius)

2. **Updated docker-compose.yml**:
   - Added `turs_test_seed.sql` mount as `04-turs-test-seed.sql`

3. **Created integration tests** (`internal/matching/turs_integration_test.go`):
   - `TestTURS_IntegrationScenario`: Full ranking test with 8 candidates
   - `TestTURS_BudgetCompatibilityLevels`: Budget ratio scoring
   - `TestTURS_ExperienceFitForSmallBudgets`: Experience penalties for small tasks
   - `TestTURS_ColdStartBehaviorFloor`: New user floor verified
   - `TestTURS_GeoRelevanceDistances`: Haversine distance scoring
   - `TestTURS_SpeedProbabilityScoring`: Response time scoring

## Key Findings

### TURS Algorithm Behavior

1. **Budget < $1000 tasks penalize elite/pro**:
   - Elite: experienceFit = 0
   - Pro: experienceFit = 0.3
   - Intermediate: experienceFit = 1.0 (best)
   - Beginner: experienceFit = 0.8

2. **Budget compatibility thresholds**:
   - ratio >= 1.2: 1.0
   - ratio >= 1.0: 0.72
   - ratio >= 0.8: 0.4
   - ratio >= 0.6: 0.2
   - ratio < 0.6: 0

3. **Cold-start protection**:
   - Users with < 5 completed tasks get IsNewUser=true
   - Behavior intent floor: 0.5 (prevents 0 score)

## Running Tests

```bash
# Go TURS tests
go test -v -run "TestTURS_" ./internal/matching/...

# All Go tests
go test ./...

# Gateway tests
cd gateway && pytest

# Full stack
docker compose up
```

## Database State

After `docker compose down -v && docker compose up`:
- 23 users total (3 original + 20 TURS test)
- 10 skills
- 1 test task with web_development requirement

## Files Modified This Session

- `pkg/db/turs_test_seed.sql` (NEW)
- `internal/matching/turs_integration_test.go` (NEW)
- `docker-compose.yml` (added seed mount)
- `instructions/` (NEW - memory files)
