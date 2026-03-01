# Architecture

## 1. Overview

This platform is a gig-based concierge system designed to optimize for fast and successful task acceptance using a behavior-driven scoring algorithm (TURS – Task User Relevancy Score find at instructions/turs.md).

The system operates in two phases:

1. Priority Phase (10 minutes curated broadcast)
2. Active Phase (24 hours open marketplace)

The architecture is designed around:

- Acceptance speed over talent perfection
- Deterministic scoring
- Atomic assignment
- Self-correcting pricing via Experience Multiplier (EM)
- Clean separation between data, scoring, and orchestration layers

The current implementation uses PostgreSQL as the sole coordination layer. There is no Redis or event bus at this stage.

---

## 2. High-Level Architecture

Client (Mobile/Web)
    ↓
API Layer
    ↓
Application Services
    ├── Database (PostgreSQL via sqlc)
    ├── Matching Engine (TURS)
    └── Orchestrator (Priority Flow)
    ↓
PostgreSQL

All state is persisted in PostgreSQL. Concurrency control relies on row-level locking and transactional integrity.

---

## 3. Core Domains

### 3.1 Users Domain

Responsible for:

- Identity
- Skill mapping
- Experience tier
- Experience Multiplier (EM)
- Minimum Acceptable Budget (MAB)
- Behavior metrics (acceptance rate, reliability, response speed)

Properties:

- EM is bounded between 0.5 and 2.0
- EM updates are logged in experience_multiplier_history
- Behavior metrics are aggregated and stored

---

### 3.2 Tasks Domain

Task lifecycle:

draft  
→ priority (10 minutes curated broadcast)  
→ active (24h open listing)  
→ accepted  
→ expired  
→ cancelled  

Atomic acceptance is enforced by:

- UNIQUE(task_id) constraint in task_acceptances
- SELECT FOR UPDATE during acceptance transaction

---

### 3.3 Matching Domain (TURS)

TURS formula:

TURS =
0.30 SkillMatch  
+ 0.25 BudgetCompatibility  
+ 0.15 GeoRelevance  
+ 0.15 ExperienceFit  
+ 0.10 BehaviorIntent  
+ 0.05 SpeedProbability  

Properties:

- Stateless scoring
- Deterministic ranking
- Versionable weights
- Pure function design
- Produces score breakdown and final score

---

### 3.4 Orchestration Domain

Orchestrator responsibilities:

- Trigger matching when task enters priority
- Execute wave-based broadcasts (15 users per wave)
- Stop waves if task is accepted
- Move task to active after 10 minutes if not accepted

Current implementation:

- Goroutine-based scheduling
- Database as single source of truth
- No distributed coordination
- Print statements as notification placeholders

---

## 4. Current Execution Flow

### Task Creation

1. Task inserted with state = priority
2. Orchestrator.StartPriority(taskID) invoked

### Priority Flow

1. Fetch task
2. Fetch required skills
3. Hard filter via SQL
4. Convert DB models to DTOs
5. Score candidates using TURS
6. Rank candidates
7. Send waves every 60 seconds
8. If accepted → stop waves
9. After 10 minutes → move to active

### Acceptance Flow

1. Begin transaction
2. SELECT task FOR UPDATE
3. Insert acceptance
4. Update task state to accepted
5. Commit

Guarantees single winner.

---

## 5. Current Architectural Characteristics

Strengths:

- Clean domain separation
- Proper data modeling
- Atomic correctness
- Scoring isolated and testable
- No distributed complexity

Limitations:

- Not horizontally scalable
- Goroutine timeouts are not crash-safe
- No exploration injection yet
- No event-driven architecture