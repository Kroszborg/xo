"""Async database connection pool using asyncpg."""

import asyncpg

_pool: asyncpg.Pool | None = None


def _normalize_dsn(dsn: str) -> str:
    """Convert postgres:// to postgresql:// for asyncpg compatibility."""
    if dsn.startswith("postgres://"):
        return "postgresql://" + dsn[len("postgres://"):]
    return dsn


async def init_pool(dsn: str, min_size: int = 2, max_size: int = 10) -> asyncpg.Pool:
    """Create the global connection pool."""
    global _pool
    _pool = await asyncpg.create_pool(
        _normalize_dsn(dsn),
        min_size=min_size,
        max_size=max_size,
    )
    return _pool


async def close_pool() -> None:
    """Close the global connection pool."""
    global _pool
    if _pool is not None:
        await _pool.close()
        _pool = None


def get_pool() -> asyncpg.Pool:
    """Return the current pool. Raises if not initialized."""
    if _pool is None:
        raise RuntimeError("Database pool not initialized. Call init_pool() first.")
    return _pool
