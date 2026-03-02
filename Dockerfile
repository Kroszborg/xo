# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download modules first so this layer is cached independently of source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /xo ./cmd/xo

# ── Runtime stage ─────────────────────────────────────────────────────────────
# alpine:3.17+ ships ca-certificates in the base image; no extra apk step needed.
FROM alpine:3.21

WORKDIR /app

COPY --from=builder /xo /app/xo

# Copy the DB schema so it can be applied via an init container / entrypoint
# script if needed outside of a migration tool.
COPY pkg/db/schema.sql /app/schema.sql

# ── Environment ──────────────────────────────────────────────────────────────
ENV DATABASE_URL=""
ENV LISTEN_ADDR=":8080"
ENV NOTIFICATION_WEBHOOK_URL=""

# FCM push notifications (set FCM_PROJECT_ID + mount service account JSON).
ENV FCM_PROJECT_ID=""
ENV GOOGLE_APPLICATION_CREDENTIALS=""

EXPOSE 8080

ENTRYPOINT ["/app/xo"]
