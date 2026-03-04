"""FastAPI dependencies: DB connection, current user, etc."""

from __future__ import annotations

from typing import Annotated

import asyncpg
import jwt
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from app.auth import decode_access_token
from app.database import get_pool

security = HTTPBearer(auto_error=False)


async def get_db() -> asyncpg.Connection:
    """Yield a connection from the pool."""
    pool = get_pool()
    async with pool.acquire() as conn:
        yield conn


async def get_current_user(
    credentials: Annotated[HTTPAuthorizationCredentials | None, Depends(security)],
    conn: Annotated[asyncpg.Connection, Depends(get_db)],
) -> dict:
    """Extract and validate the current user from the Authorization header."""
    if credentials is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Missing authorization header",
        )
    try:
        payload = decode_access_token(credentials.credentials)
    except jwt.PyJWTError as e:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Invalid token: {e}",
        )

    user_id = payload.get("sub")
    if not user_id:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token missing subject",
        )

    user = await conn.fetchrow("SELECT id, email, phone, role, status FROM users WHERE id = $1", user_id)
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User not found",
        )
    if user["status"] != "active":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Account is not active",
        )

    return dict(user)


# Type alias for convenience
CurrentUser = Annotated[dict, Depends(get_current_user)]
DBConn = Annotated[asyncpg.Connection, Depends(get_db)]
