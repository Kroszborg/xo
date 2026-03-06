# Development Steps

## How to Run the Project

### Prerequisites
- Docker & Docker Compose
- Go 1.25+
- Python 3.12+ (for gateway development)

### Start Full Stack
```bash
cd /home/hakaitech/sandbox/src/xo
docker compose up -d
```

### Access Services
- Gateway: http://localhost:8000
- xo API: http://localhost:8080
- PostgreSQL: localhost:5432 (xo/xo)

### Fresh Start (with new seed data)
```bash
docker compose down -v
docker compose up -d
```

## Testing

### Go Tests
```bash
# All tests
go test ./...

# TURS tests with verbose output
go test -v -run "TestTURS_" ./internal/matching/...

# Specific test
go test -v -run "TestTURS_IntegrationScenario" ./internal/matching/...
```

### Gateway Tests
```bash
cd gateway
source .venv/bin/activate    # if using venv
pytest                        # all tests
pytest -v                     # verbose
pytest --tb=short             # short tracebacks
```

## Database Operations

### Connect to PostgreSQL
```bash
docker compose exec postgres psql -U xo -d xo
```

### View TURS Test Users
```sql
SELECT u.email, up.experience_level, up.mab, up.radius_km,
       ubm.acceptance_rate, ubm.completion_rate, ubm.total_tasks_completed
FROM users u 
JOIN user_profiles up ON u.id = up.user_id
JOIN user_behavior_metrics ubm ON u.id = ubm.user_id
ORDER BY up.experience_level DESC, up.mab DESC;
```

### View Test Task
```sql
SELECT t.id, t.budget, t.complexity_level, t.lat, t.lng, t.radius_km,
       array_agg(s.name) as required_skills
FROM tasks t
LEFT JOIN task_required_skills trs ON t.id = trs.task_id
LEFT JOIN skills s ON trs.skill_id = s.id
WHERE t.id = 'c1000000-0000-0000-0000-000000000100'
GROUP BY t.id;
```

## Adding Test Data

### Add New Test Users

1. Edit `pkg/db/turs_test_seed.sql`
2. Rebuild: `docker compose down -v && docker compose up -d`
3. Verify: Check postgres logs for INSERT statements

### Add New Skills

```sql
INSERT INTO skills (id, name) VALUES
    ('a1000000-0000-0000-0000-000000000011', 'new_skill')
ON CONFLICT DO NOTHING;
```

## TURS Algorithm Reference

### Weights (offline task)
- SkillMatch: 30%
- BudgetCompatibility: 25%
- GeoRelevance: 15%
- ExperienceFit: 15%
- BehaviorIntent: 10%
- SpeedProbability: 5%

### For online tasks, GeoRelevance is redistributed proportionally

### Budget Ratio Scoring
```
ratio = taskBudget / userMAB
>= 1.2: 1.0 (task pays more than expected)
>= 1.0: 0.72
>= 0.8: 0.4
>= 0.6: 0.2
< 0.6: 0 (task pays much less than expected)
```

### Experience Fit (budget < $1000)
```
beginner: 0.8
intermediate: 1.0 (best for small tasks)
pro: 0.3
elite: 0
```
