from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, Request, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app.config import settings
from app.database import close_pool, init_pool
from app.schemas.envelope import err


# ---------------------------------------------------------------------------
# Lifespan
# ---------------------------------------------------------------------------

@asynccontextmanager
async def lifespan(app: FastAPI):
    await init_pool()
    yield
    await close_pool()


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------

app = FastAPI(
    title="XO Gateway",
    version="0.1.0",
    lifespan=lifespan,
)

# CORS
origins = [o.strip() for o in settings.cors_origins.split(",")]
app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# ---------------------------------------------------------------------------
# Error handlers
# ---------------------------------------------------------------------------

@app.exception_handler(HTTPException)
async def http_exception_handler(request: Request, exc: HTTPException):
    """Normalise HTTPException responses into the envelope format."""
    detail = exc.detail
    # If detail is already an envelope dict, pass it through
    if isinstance(detail, dict) and ("error" in detail or "data" in detail):
        return JSONResponse(status_code=exc.status_code, content=detail)
    return JSONResponse(
        status_code=exc.status_code,
        content=err("HTTP_ERROR", str(detail)),
    )


@app.exception_handler(Exception)
async def generic_exception_handler(request: Request, exc: Exception):
    """Catch-all for unhandled exceptions."""
    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content=err("INTERNAL_ERROR", "An unexpected error occurred"),
    )


# ---------------------------------------------------------------------------
# Routers
# ---------------------------------------------------------------------------

from app.routers import auth, categories, chat, nearby, notifications, oauth, onboarding, profile, reviews, tasks  # noqa: E402

app.include_router(auth.router)
app.include_router(oauth.router)
app.include_router(profile.router)
app.include_router(tasks.router)
app.include_router(categories.router)
app.include_router(notifications.router)
app.include_router(chat.router)
app.include_router(reviews.router)
app.include_router(nearby.router)
app.include_router(onboarding.router)


# ---------------------------------------------------------------------------
# Health check
# ---------------------------------------------------------------------------

@app.get("/health")
async def health():
    return {"status": "ok"}
