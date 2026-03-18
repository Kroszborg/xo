from typing import AsyncIterator

import asyncpg
import jwt
from fastapi import Depends, Header, HTTPException, Request, status

from app.database import get_pool

# ---------------------------------------------------------------------------
# Database dependency
# ---------------------------------------------------------------------------

ALLOWED_CLIENT_TYPES = {"web", "mobile_android", "mobile_ios"}


async def get_db_conn() -> AsyncIterator[asyncpg.Connection]:
    """Yield an asyncpg connection, released automatically after the request."""
    pool = get_pool()
    conn: asyncpg.Connection = await pool.acquire()
    try:
        yield conn
    finally:
        await pool.release(conn)


# ---------------------------------------------------------------------------
# Auth dependency
# ---------------------------------------------------------------------------


async def get_current_user(request: Request) -> dict:
    """Extract and validate the JWT from the Authorization header.

    Returns a dict with ``id`` and ``role`` keys.
    """
    auth_header: str | None = request.headers.get("Authorization")
    if not auth_header or not auth_header.startswith("Bearer "):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Missing or invalid Authorization header",
        )

    token = auth_header.removeprefix("Bearer ").strip()

    try:
        from app.auth import decode_token

        payload = decode_token(token)
    except jwt.ExpiredSignatureError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token has expired",
        )
    except jwt.PyJWTError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token",
        )

    if payload.get("type") != "access":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token type",
        )

    return {"id": payload["sub"], "role": payload.get("role", "")}


# ---------------------------------------------------------------------------
# Client-type dependency
# ---------------------------------------------------------------------------


async def get_client_type(
    x_client_type: str = Header(default="web", alias="X-Client-Type"),
) -> str:
    """Read the ``X-Client-Type`` header and validate it."""
    value = x_client_type.lower()
    if value not in ALLOWED_CLIENT_TYPES:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Invalid client type. Allowed: {', '.join(sorted(ALLOWED_CLIENT_TYPES))}",
        )
    return value
