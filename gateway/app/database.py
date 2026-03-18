import asyncpg
from contextlib import asynccontextmanager
from typing import AsyncIterator

from app.config import settings

_pool: asyncpg.Pool | None = None


def _dsn() -> str:
    """Return an asyncpg-compatible DSN.

    The config value may use the ``postgresql://`` scheme (common in
    application configs) or the ``postgres://`` scheme (used by Docker
    Compose).  ``asyncpg`` only accepts ``postgresql://``, so we normalise
    here.
    """
    url = settings.database_url
    if url.startswith("postgres://"):
        url = url.replace("postgres://", "postgresql://", 1)
    return url


async def init_pool() -> None:
    """Create the connection pool.  Call once at application startup."""
    global _pool
    _pool = await asyncpg.create_pool(
        dsn=_dsn(),
        min_size=2,
        max_size=10,
    )


async def close_pool() -> None:
    """Gracefully close the connection pool.  Call at shutdown."""
    global _pool
    if _pool is not None:
        await _pool.close()
        _pool = None


def get_pool() -> asyncpg.Pool:
    """Return the current pool, raising if not yet initialised."""
    if _pool is None:
        raise RuntimeError("Database pool is not initialised")
    return _pool


@asynccontextmanager
async def get_db() -> AsyncIterator[asyncpg.Connection]:
    """Async context manager that acquires a connection from the pool."""
    pool = get_pool()
    conn: asyncpg.Connection = await pool.acquire()
    try:
        yield conn
    finally:
        await pool.release(conn)
