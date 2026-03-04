# ── Build stage ──────────────────────────────────────────────────────────────
FROM python:3.12-slim AS builder

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM python:3.12-slim

WORKDIR /app

# Copy installed packages from builder
COPY --from=builder /install /usr/local

# Copy application code
COPY app/ /app/app/

# ── Environment ──────────────────────────────────────────────────────────────
ENV DATABASE_URL="postgres://xo:xo@postgres:5432/xo?sslmode=disable"
ENV XO_SERVICE_URL="http://xo:8080"
ENV JWT_SECRET="change-me-in-production"
ENV JWT_ALGORITHM="HS256"
ENV JWT_ACCESS_TOKEN_MINUTES="30"
ENV JWT_REFRESH_TOKEN_DAYS="30"
ENV LISTEN_HOST="0.0.0.0"
ENV LISTEN_PORT="8000"
ENV UPLOAD_DIR="/app/uploads"

EXPOSE 8000

CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
