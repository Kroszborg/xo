from datetime import datetime, timedelta, timezone

import asyncpg
from fastapi import APIRouter, Depends, HTTPException, status

from app.auth import (
    create_access_token,
    create_refresh_token,
    decode_token,
    hash_password,
    hash_token,
    verify_password,
)
from app.config import settings
from app.deps import get_db_conn
from app.schemas.auth import LoginRequest, RefreshRequest, RegisterRequest
from app.schemas.envelope import err, ok

router = APIRouter(prefix="/api/v1/auth", tags=["auth"])


# ---------------------------------------------------------------------------
# POST /register
# ---------------------------------------------------------------------------

@router.post("/register", status_code=status.HTTP_201_CREATED)
async def register(
    body: RegisterRequest,
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    # Check for existing user
    existing = await conn.fetchval(
        "SELECT id FROM users WHERE email = $1", body.email
    )
    if existing:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail=err("EMAIL_EXISTS", "A user with this email already exists"),
        )

    pw_hash = hash_password(body.password)

    async with conn.transaction():
        # Create user
        user_id = await conn.fetchval(
            """
            INSERT INTO users (email, password_hash, role)
            VALUES ($1, $2, $3)
            RETURNING id
            """,
            body.email,
            pw_hash,
            body.role,
        )
        user_id_str = str(user_id)

        # Create profile row
        await conn.execute(
            """
            INSERT INTO user_profiles (user_id)
            VALUES ($1)
            """,
            user_id,
        )

        # Create behavior metrics row
        await conn.execute(
            """
            INSERT INTO user_behavior_metrics (user_id)
            VALUES ($1)
            """,
            user_id,
        )

        # Issue tokens
        access = create_access_token(user_id_str, body.role)
        refresh = create_refresh_token(user_id_str)

        # Persist refresh token hash
        await conn.execute(
            """
            INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
            VALUES ($1, $2, $3)
            """,
            user_id,
            hash_token(refresh),
            datetime.now(timezone.utc)
            + timedelta(days=settings.jwt_refresh_token_expire_days),
        )

    return ok(
        {
            "access_token": access,
            "refresh_token": refresh,
            "token_type": "bearer",
            "user_id": user_id_str,
            "role": body.role,
        }
    )


# ---------------------------------------------------------------------------
# POST /login
# ---------------------------------------------------------------------------

@router.post("/login")
async def login(
    body: LoginRequest,
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    row = await conn.fetchrow(
        "SELECT id, password_hash, role, is_active FROM users WHERE email = $1",
        body.email,
    )
    if row is None or row["password_hash"] is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=err("INVALID_CREDENTIALS", "Invalid email or password"),
        )

    if not verify_password(body.password, row["password_hash"]):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=err("INVALID_CREDENTIALS", "Invalid email or password"),
        )

    if not row["is_active"]:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail=err("ACCOUNT_DISABLED", "This account has been disabled"),
        )

    user_id_str = str(row["id"])
    role = row["role"]

    access = create_access_token(user_id_str, role)
    refresh = create_refresh_token(user_id_str)

    # Store refresh token
    await conn.execute(
        """
        INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
        VALUES ($1, $2, $3)
        """,
        row["id"],
        hash_token(refresh),
        datetime.now(timezone.utc)
        + timedelta(days=settings.jwt_refresh_token_expire_days),
    )

    return ok(
        {
            "access_token": access,
            "refresh_token": refresh,
            "token_type": "bearer",
            "user_id": user_id_str,
            "role": role,
        }
    )


# ---------------------------------------------------------------------------
# POST /refresh
# ---------------------------------------------------------------------------

@router.post("/refresh")
async def refresh(
    body: RefreshRequest,
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    try:
        payload = decode_token(body.refresh_token)
    except Exception:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=err("INVALID_TOKEN", "Refresh token is invalid or expired"),
        )

    if payload.get("type") != "refresh":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=err("INVALID_TOKEN", "Token is not a refresh token"),
        )

    token_h = hash_token(body.refresh_token)

    # Verify the token exists and is not revoked
    rt_row = await conn.fetchrow(
        """
        SELECT id, user_id, revoked FROM refresh_tokens
        WHERE token_hash = $1
        """,
        token_h,
    )

    if rt_row is None or rt_row["revoked"]:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=err("INVALID_TOKEN", "Refresh token has been revoked"),
        )

    user_id = rt_row["user_id"]

    # Look up current role
    role = await conn.fetchval("SELECT role FROM users WHERE id = $1", user_id)
    if role is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=err("USER_NOT_FOUND", "User no longer exists"),
        )

    user_id_str = str(user_id)

    # Rotate: revoke old, issue new
    async with conn.transaction():
        await conn.execute(
            "UPDATE refresh_tokens SET revoked = TRUE WHERE id = $1",
            rt_row["id"],
        )

        new_access = create_access_token(user_id_str, role)
        new_refresh = create_refresh_token(user_id_str)

        await conn.execute(
            """
            INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
            VALUES ($1, $2, $3)
            """,
            user_id,
            hash_token(new_refresh),
            datetime.now(timezone.utc)
            + timedelta(days=settings.jwt_refresh_token_expire_days),
        )

    return ok(
        {
            "access_token": new_access,
            "refresh_token": new_refresh,
            "token_type": "bearer",
        }
    )


# ---------------------------------------------------------------------------
# POST /logout
# ---------------------------------------------------------------------------

@router.post("/logout", status_code=status.HTTP_200_OK)
async def logout(
    body: RefreshRequest,
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    token_h = hash_token(body.refresh_token)
    result = await conn.execute(
        "UPDATE refresh_tokens SET revoked = TRUE WHERE token_hash = $1 AND revoked = FALSE",
        token_h,
    )
    # Always return success to avoid leaking token existence
    return ok({"message": "Logged out successfully"})
