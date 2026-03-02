# Implementation Steps

## Phase 1: Architecture Redesign
- [x] Design new architecture (REST API, dynamic weights, cold-start, notification interface)
- [x] Update architecture.md

## Phase 2: Database Changes
- [ ] Add 'completed' state to tasks CHECK constraint + completed_at column
- [ ] Add new queries: CancelTask, CompleteTask, ListTasks, GetNewUsersWithSkills, UpdateTask, GetTaskAcceptance
- [ ] Regenerate sqlc

## Phase 3: Matching Engine Updates
- [ ] Dynamic weights (online: redistribute geo weight proportionally)
- [ ] Cold-start behavior-intent floor for new users (IsNewUser flag)
- [ ] Update tests

## Phase 4: Notification Interface
- [ ] Create Notifier interface
- [ ] LogNotifier (development)
- [ ] WebhookNotifier (production)

## Phase 5: REST API Layer
- [ ] response.go — JSON response helpers
- [ ] server.go — HTTP server, routes, middleware
- [ ] handler.go — Task CRUD + accept + complete handlers

## Phase 6: Orchestrator Redesign
- [ ] Integrate cold-start exploration slots (15% of wave)
- [ ] Use Notifier interface instead of fmt.Printf
- [ ] Integrate with API (called on task creation)

## Phase 7: Task Completion + EM Update
- [ ] EM update formula with adaptive learning rate
- [ ] Persist EM history + behavior metric updates

## Phase 8: Entry Point
- [ ] Rewrite main.go as HTTP server with graceful shutdown

## Phase 9: Testing
- [ ] Unit tests for matching (dynamic weights, cold-start)
- [ ] Compilation check
- [ ] Run all existing + new tests