"""FastAPI application entry point for the API Gateway."""

from contextlib import asynccontextmanager
import logging
import sys
import uuid

from fastapi import FastAPI, Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.gzip import GZipMiddleware
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware

from app.config import settings
from app.database import close_pool, init_pool
from app.schemas.envelope import err

# Configure logging to show INFO level
logging.basicConfig(level=logging.INFO, format="%(message)s", stream=sys.stdout)
logger = logging.getLogger("gateway")


# ─── Lifespan: DB pool ───────────────────────────────────────────────────────

def print_routes(app: FastAPI) -> None:
    """Print all registered routes on startup."""
    logger.info("[gateway] Registered endpoints:")
    routes = []
    for route in app.routes:
        if hasattr(route, "methods") and hasattr(route, "path"):
            for method in route.methods:
                if method != "HEAD":  # Skip implicit HEAD methods
                    routes.append((method, route.path))
    # Sort by path, then method
    routes.sort(key=lambda x: (x[1], x[0]))
    for method, path in routes:
        logger.info(f"  {method:<7} {path}")
    sys.stdout.flush()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup: create DB pool, print routes. Shutdown: close DB pool."""
    pool = await init_pool(
        settings.database_url,
        min_size=settings.db_min_pool_size,
        max_size=settings.db_max_pool_size,
    )
    logger.info(f"[gateway] DB pool created (min={settings.db_min_pool_size}, max={settings.db_max_pool_size})")
    print_routes(app)
    yield
    await close_pool()
    logger.info("[gateway] DB pool closed")


app = FastAPI(
    title="xo API Gateway",
    version="0.1.0",
    lifespan=lifespan,
)


# ─── Middleware ───────────────────────────────────────────────────────────────

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# GZip
app.add_middleware(GZipMiddleware, minimum_size=1000)


class RequestIDMiddleware(BaseHTTPMiddleware):
    """Attach a unique request ID to every request/response."""

    async def dispatch(self, request: Request, call_next):
        request_id = request.headers.get("X-Request-ID", str(uuid.uuid4()))
        request.state.request_id = request_id
        response = await call_next(request)
        response.headers["X-Request-ID"] = request_id
        return response


app.add_middleware(RequestIDMiddleware)


# ─── Exception handlers ──────────────────────────────────────────────────────

@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError):
    """Return validation errors in the standard envelope."""
    details = []
    for e in exc.errors():
        loc = " -> ".join(str(l) for l in e["loc"])
        details.append(f"{loc}: {e['msg']}")
    return JSONResponse(
        status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
        content=err(error="validation_error", message="; ".join(details)),
    )


@app.exception_handler(Exception)
async def general_exception_handler(request: Request, exc: Exception):
    """Catch-all: wrap unexpected errors in the envelope."""
    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content=err(error="internal_error", message=str(exc)),
    )


# ─── Routers ─────────────────────────────────────────────────────────────────

from app.routers import auth, config, dashboard, location, profile, tasks, verification  # noqa: E402

app.include_router(auth.router)
app.include_router(profile.router)
app.include_router(verification.router)
app.include_router(location.router)
app.include_router(config.router)
app.include_router(tasks.router)
app.include_router(dashboard.router)


# ─── Health check ─────────────────────────────────────────────────────────────

@app.get("/healthz", tags=["health"])
async def healthz():
    """Health check endpoint."""
    return {"status": "ok", "service": "gateway"}
