# Architecture

## 1. Overview

xo is a focused microservice responsible for **task lifecycle management**, **TURS-based candidate scoring**, and **priority-flow orchestration**. It exposes a REST API consumed by other services in the platform.

The system operates in two phases per task:

1. **Priority Phase** (10 minutes) — curated wave-based notifications to top-ranked candidates
2. **Active Phase** (24 hours) — open marketplace listing with TTL

Core design principles:

- Fast acceptance over perfect matching
- Deterministic scoring with dynamic weight adjustment (online vs offline)
- Atomic task assignment via row-level locking
- Cold-start exploration quota ensuring new users get visibility
- Self-correcting pricing via Experience Multiplier (EM)
- Clean separation: data → scoring → orchestration → API

PostgreSQL is the sole coordination and persistence layer. No Redis or message broker.

---

## 2. High-Level Architecture

```
External Services
    ↓ REST (JSON)
┌─────────────────────────┐
│   API Layer (net/http)   │
│   /api/v1/tasks/*        │
├─────────────────────────┤
│   Application Services   │
│   ├── Orchestrator       │
│   ├── Matching (TURS)    │
│   └── Notifier           │
├─────────────────────────┤
│   Data Layer (sqlc)      │
│   PostgreSQL             │
└─────────────────────────┘
```

---

## 3. REST API

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health/readiness check |
| POST | /api/v1/tasks | Create task → triggers priority flow |
| GET | /api/v1/tasks | List tasks (filterable by state, category) |
| GET | /api/v1/tasks/{id} | Get task by ID |
| PUT | /api/v1/tasks/{id} | Update task (active state only) |
| DELETE | /api/v1/tasks/{id} | Cancel task |
| POST | /api/v1/tasks/{id}/accept | Task-doer accepts task |
| POST | /api/v1/tasks/{id}/complete | Mark completed → EM update |

---

## 4. Task Lifecycle

```
POST /tasks → state=priority
  → TURS scoring + candidate ranking
  → Wave notifications (15/wave, 60s apart, 10 min window)
    → If accepted: state=accepted, stop waves
    → If not accepted after 10 min: state=active (24h TTL)
      → If accepted from pool: state=accepted
      → After 24h: state=expired
  → POST /complete → state=completed → EM update + metric refresh
```

Cancellation allowed from: priority, active.
Update allowed from: active only.

---

## 5. Core Domains

### 5.1 Matching (TURS)

TURS scores predict likelihood of fast, successful task acceptance.

**Offline tasks** (standard weights):
```
TURS = 0.30 * SkillMatch + 0.25 * BudgetCompat + 0.15 * GeoRelevance
     + 0.15 * ExperienceFit + 0.10 * BehaviorIntent + 0.05 * SpeedProb
```

**Online tasks** (geo removed, proportionally redistributed):
```
TURS = 0.3529 * SkillMatch + 0.2941 * BudgetCompat + 0.0 * GeoRelevance
     + 0.1765 * ExperienceFit + 0.1176 * BehaviorIntent + 0.0588 * SpeedProb
```

Properties:
- Stateless, deterministic scoring
- Dynamic weights based on task.IsOnline
- Pure functions, no side effects
- Cold-start baseline for new users (boosted behavior intent)

### 5.2 Cold-Start Strategy

Users with < 5 completed tasks are classified as "new users".

- Separate DB query fetches new users matching task skills (relaxed activity filter)
- New users get a behavior-intent floor of 0.5 to offset zero acceptance-rate penalty
- 15% of each notification wave is reserved for exploration (new users)
- If insufficient new users exist, slots are filled by regular candidates
- This prevents the chicken-and-egg problem where new users never get priority notifications

### 5.3 Orchestration

Orchestrator responsibilities:
- Trigger TURS scoring when task enters priority
- Build mixed waves (veterans + exploration slots)
- Execute wave-based notifications (15 users per wave, 60s intervals)
- Stop waves immediately on acceptance
- Transition task to active after 10 minutes if not accepted

### 5.4 Notification

Interface-based design for testability and flexibility:
```go
type Notifier interface {
    Notify(ctx context.Context, taskID uuid.UUID, userIDs []uuid.UUID, waveNumber int) error
}
```
- LogNotifier: development (prints to stdout)
- WebhookNotifier: production (POSTs to configured URL)

### 5.5 EM Update (Task Completion)

On task completion:
1. Fetch acceptance record (accepted_budget) and task (shown_budget)
2. Compute alpha: 0.20 (first 5 accepts), 0.10 (next 5), 0.05 (steady-state)
3. EM_new = clamp(EM_old * (1-alpha) + EM_old * (accepted/shown) * alpha, 0.5, 2.0)
4. Persist: update profile, insert EM history, increment completed tasks

---

## 6. Package Layout

```
cmd/xo/main.go                   - HTTP server entry point
internal/
  api/
    server.go                     - Server struct, routes, middleware
    handler.go                    - Task HTTP handlers
    response.go                   - JSON response helpers
  matching/
    service.go                    - TURSService interface
    turs.go                       - Scoring implementation
    turs_test.go                  - Tests
    types.go                      - TaskInput, CandidateInput, ScoreBreakdown
    weights.go                    - Weights + dynamic adjustment
  orchestrator/
    orchestrator.go               - Priority flow + wave management
  notification/
    notifier.go                   - Notifier interface + implementations
pkg/
  db/
    schema.sql                    - DDL
    queries.sql                   - SQL queries
    sqlc.yaml                     - sqlc config
    seed.sql                      - Test data
    db/                           - Generated Go code (sqlc)
```

---

## 7. Key Guarantees

- **Atomic acceptance**: SELECT FOR UPDATE + UNIQUE constraint on task_acceptances(task_id)
- **Single winner**: Transaction isolation prevents double-accept
- **Crash recovery**: Background context used for state transitions after timeout
- **Deterministic scoring**: Same inputs always produce same TURS output
- **Exploration fairness**: New users guaranteed wave slots via quota system

---

## 8. Push Notification Strategy (React Native)

The recommended stack for delivering real-time task notifications to a React Native
app cross-compiled for Android and iOS:

### 8.1 Transport: Firebase Cloud Messaging (FCM)

FCM is the single transport layer for both platforms:
- **Android**: native FCM (GCM replacement, built into Google Play Services)
- **iOS**: FCM wraps APNs — you upload your APNs auth key to Firebase once,
  and FCM handles the APNs relay transparently

This gives xo a single API to target (FCM HTTP v1) regardless of client OS.

### 8.2 Architecture (Implemented)

FCM is integrated directly into xo via `FCMNotifier`, eliminating the need for
a separate sidecar. The notifier implements the `Notifier` interface alongside
`LogNotifier` (dev) and `WebhookNotifier` (webhook fallback).

```
xo (FCMNotifier)
    → device_tokens table (user_id → FCM token lookup)
    → FCM HTTP v1 API (OAuth2 via GOOGLE_APPLICATION_CREDENTIALS)
        ├── Android: FCM direct (priority: high, channel: task_alerts)
        └── iOS: FCM → APNs relay (sound: default, content-available: 1)
```

The `FCMNotifier` is responsible for:
1. Looking up device tokens from `device_tokens` table for target user IDs
2. Building platform-specific payloads (android vs ios overrides)
3. Dispatching to FCM HTTP v1 with bounded concurrency (10 parallel sends)
4. Auto-cleaning stale tokens on FCM NOT_FOUND/UNREGISTERED errors
5. OAuth2 token auto-refresh via `golang.org/x/oauth2/google` ADC

### 8.3 React Native Client

| Library | Purpose |
|---------|---------|
| `@react-native-firebase/app` | Firebase core |
| `@react-native-firebase/messaging` | FCM registration, foreground/background handlers |
| `@notifee/react-native` | Rich local notifications (heads-up, actions, channels) |

Flow:
1. App registers with FCM on launch → receives device token
2. Token sent to xo via `PUT /api/v1/devices` (user_id + token + platform)
3. On background message: `messaging().setBackgroundMessageHandler` triggers OS notification via Notifee
4. On foreground message: `messaging().onMessage` shows in-app alert via Notifee
5. User taps notification → deep-links to task detail screen
6. On logout: `DELETE /api/v1/devices` removes the device token

### 8.4 Payload Design

```json
{
  "message": {
    "token": "<device_token>",
    "data": {
      "type": "task_notification",
      "task_id": "uuid",
      "wave_number": "1",
      "budget": "1200.00"
    },
    "notification": {
      "title": "New task available",
      "body": "A ₹1,200 task matching your skills is available"
    },
    "android": {
      "priority": "high",
      "notification": { "channel_id": "task_alerts" }
    },
    "apns": {
      "payload": {
        "aps": {
          "alert": { "title": "New task available", "body": "..." },
          "sound": "default",
          "content-available": 1
        }
      }
    }
  }
}
```

Use **data messages** (not notification-only) so the app always processes the
payload in both foreground and background, giving full control over display.

### 8.5 Why FCM Over Alternatives

| Criteria | FCM | Expo Push | OneSignal |
|----------|-----|-----------|-----------|
| Free tier | Unlimited | 500 req/s | 10k devices |
| iOS support | APNs relay built-in | APNs relay | APNs relay |
| Payload control | Full (data msgs) | Limited | Moderate |
| Vendor lock-in | Google (token-level) | Expo infra | OneSignal |
| Self-host option | No (but thin wrapper) | No | No |
| Latency | ~100-300ms P95 | +hop via Expo | +hop via OS |

FCM is the most direct path with full payload control and zero per-message cost.

### 8.6 Environment

| Variable | Used By | Purpose |
|----------|---------|---------|
| `FCM_PROJECT_ID` | xo | Firebase project ID (enables FCMNotifier) |
| `GOOGLE_APPLICATION_CREDENTIALS` | xo | Path to Firebase service account JSON |
| `NOTIFICATION_WEBHOOK_URL` | xo | Webhook fallback (used when FCM_PROJECT_ID empty) |

### 8.7 API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `PUT` | `/api/v1/devices` | Register/update device token |
| `DELETE` | `/api/v1/devices` | Remove device token (logout) |
| `GET` | `/api/v1/devices/{user_id}` | List user's device tokens |