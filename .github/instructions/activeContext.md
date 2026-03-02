# Active Context

## Current Work
FCM push notification integration complete. xo now supports direct FCM delivery.

## Key Decisions
- Go 1.25 net/http ServeMux for routing (no third-party router)
- Online tasks: GeoRelevance weight = 0, redistributed proportionally to remaining 5 dimensions
- Cold-start: users with < 5 completed tasks get exploration slots (15% per wave) + behavior-intent floor of 0.5
- Notification: interface-based (LogNotifier for dev, WebhookNotifier for webhook, FCMNotifier for push)
- FCM: integrated directly into xo via FCM HTTP v1 API + golang.org/x/oauth2/google ADC
- FCM auth: GOOGLE_APPLICATION_CREDENTIALS + FCM_PROJECT_ID env vars
- Device tokens: stored in device_tokens table with user_id + token + platform (android/ios)
- Stale token cleanup: automatic on FCM NOT_FOUND/UNREGISTERED errors
- EM update happens on POST /tasks/{id}/complete using adaptive alpha (0.20 → 0.10 → 0.05)
- Task states: priority → active → accepted → completed (also: expired, cancelled)

## Files Being Modified
- pkg/db/schema.sql — add 'completed' state, completed_at column
- pkg/db/queries.sql — new queries for CRUD, cold-start candidates, task completion
- internal/matching/weights.go — dynamic weight computation
- internal/matching/turs.go — cold-start behavior baseline
- internal/matching/types.go — IsNewUser flag on CandidateInput
- internal/api/ — new package (server.go, handler.go, response.go)
- internal/notification/ — new package (notifier.go)
- internal/orchestrator/orchestrator.go — cold-start slots, notifier integration
- cmd/xo/main.go — HTTP server with graceful shutdown
