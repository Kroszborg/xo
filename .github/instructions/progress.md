# Progress

## Completed
- [x] Architecture redesign document written
- [x] Memory files updated (steps.md, activeContext.md)
- [x] DB schema updated: added 'completed' state + completed_at column
- [x] DB queries added: CancelTask, CompleteTask, UpdateTask, ListTasks, GetNewUserCandidates, GetTaskAcceptance
- [x] sqlc regenerated (v1.30.0)
- [x] Dynamic weights: Weights.ForTask() redistributes geo weight proportionally for online tasks
- [x] Cold-start: IsNewUser flag on CandidateInput, behavior-intent floor of 0.5 for new users
- [x] Notification interface: Notifier with LogNotifier + WebhookNotifier implementations
- [x] REST API: server.go (routes + middleware), handler.go (CRUD + accept + complete), response.go
- [x] Orchestrator redesigned: cold-start exploration slots (15% per wave), Notifier integration, CompleteTask + EM update
- [x] main.go: HTTP server with graceful shutdown, configurable via LISTEN_ADDR/DATABASE_URL/NOTIFICATION_WEBHOOK_URL
- [x] Tests: 24/24 passing (16 existing + 8 new covering dynamic weights, cold-start, EM)
- [x] go vet: clean
- [x] go build ./...: clean
- [x] Dockerfile updated: multi-stage build, LISTEN_ADDR/NOTIFICATION_WEBHOOK_URL env, EXPOSE 8080
- [x] docker-compose.yml updated: ports 8080:8080, new env vars, removed TASK_ID
- [x] Push notification strategy documented in architecture.md (FCM + @react-native-firebase)
- [x] FCM integration: device_tokens table + queries + sqlc regenerated
- [x] FCMNotifier: FCM HTTP v1 API with OAuth2, auto token refresh, stale token cleanup
- [x] Device token API: PUT/DELETE /api/v1/devices, GET /api/v1/devices/{user_id}
- [x] main.go: FCM_PROJECT_ID + GOOGLE_APPLICATION_CREDENTIALS env var support
- [x] Docker: FCM env vars + secrets volume mount in docker-compose
- [x] Dependencies: golang.org/x/oauth2 added to go.mod
- [x] Tests: 29/29 passing (24 matching + 5 notification)

## Pending
- None
