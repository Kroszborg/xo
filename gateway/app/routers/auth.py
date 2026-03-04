"""Authentication endpoints: register, login, OTP, refresh, logout."""

from __future__ import annotations

import secrets
from datetime import datetime, timedelta, timezone

from argon2 import PasswordHasher
from argon2.exceptions import VerifyMismatchError
from fastapi import APIRouter, HTTPException, status

from app.auth import (
    create_access_token,
    create_refresh_token,
    hash_refresh_token,
)
from app.config import settings
from app.deps import CurrentUser, DBConn
from app.schemas.auth import (
    LoginRequest,
    RefreshRequest,
    RegisterRequest,
    SendOTPRequest,
    TokenResponse,
    VerifyOTPRequest,
)
from app.schemas.envelope import err, ok

router = APIRouter(prefix="/api/v1/auth", tags=["auth"])
ph = PasswordHasher()

# ─── Role mapping: frontend sends giver/doer, DB stores task_giver/task_doer ─
_ROLE_MAP = {"giver": "task_giver", "doer": "task_doer", "both": "both"}


@router.post("/register", status_code=status.HTTP_201_CREATED)
async def register(body: RegisterRequest, conn: DBConn):
    """Register a new user."""
    db_role = _ROLE_MAP.get(body.role)
    if db_role is None:
        raise HTTPException(status_code=400, detail="Invalid role")

    # Check duplicate email
    existing = await conn.fetchval("SELECT id FROM users WHERE email = $1", body.email)
    if existing:
        raise HTTPException(status_code=409, detail="Email already registered")

    password_hash = ph.hash(body.password)

    row = await conn.fetchrow(
        """
        INSERT INTO users (email, phone, password_hash, role, status)
        VALUES ($1, $2, $3, $4, 'active')
        RETURNING id, email, role, created_at
        """,
        body.email,
        body.phone,
        password_hash,
        db_role,
    )

    # Create gateway profile stub
    await conn.execute(
        "INSERT INTO gateway_user_profile (user_id) VALUES ($1) ON CONFLICT DO NOTHING",
        row["id"],
    )

    # Create xo user_profiles stub so xo queries don't break
    await conn.execute(
        """
        INSERT INTO user_profiles (user_id, experience_level, experience_multiplier, mab, radius_km)
        VALUES ($1, 'beginner', 1.00, 0.00, 10)
        ON CONFLICT DO NOTHING
        """,
        row["id"],
    )

    # Create user behavior metrics stub
    await conn.execute(
        "INSERT INTO user_behavior_metrics (user_id) VALUES ($1) ON CONFLICT DO NOTHING",
        row["id"],
    )

    # Create verification record
    await conn.execute(
        "INSERT INTO user_verification (user_id) VALUES ($1) ON CONFLICT DO NOTHING",
        row["id"],
    )

    # Issue tokens
    access_token, expires_in = create_access_token(str(row["id"]), db_role)
    raw_refresh, refresh_hash = create_refresh_token()
    expires_at = datetime.now(timezone.utc) + timedelta(days=settings.jwt_refresh_token_days)
    await conn.execute(
        "INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
        row["id"],
        refresh_hash,
        expires_at,
    )

    return ok(
        data=TokenResponse(
            access_token=access_token,
            refresh_token=raw_refresh,
            expires_in=expires_in,
        ).model_dump(),
        message="Registration successful",
    )


@router.post("/login")
async def login(body: LoginRequest, conn: DBConn):
    """Authenticate with email + password."""
    row = await conn.fetchrow(
        "SELECT id, email, password_hash, role, status FROM users WHERE email = $1",
        body.email,
    )
    if row is None:
        raise HTTPException(status_code=401, detail="Invalid credentials")
    if row["status"] != "active":
        raise HTTPException(status_code=403, detail="Account is not active")

    try:
        ph.verify(row["password_hash"], body.password)
    except VerifyMismatchError:
        raise HTTPException(status_code=401, detail="Invalid credentials")

    # Rehash if parameters changed
    if ph.check_needs_rehash(row["password_hash"]):
        new_hash = ph.hash(body.password)
        await conn.execute("UPDATE users SET password_hash = $1 WHERE id = $2", new_hash, row["id"])

    access_token, expires_in = create_access_token(str(row["id"]), row["role"])
    raw_refresh, refresh_hash = create_refresh_token()
    expires_at = datetime.now(timezone.utc) + timedelta(days=settings.jwt_refresh_token_days)
    await conn.execute(
        "INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
        row["id"],
        refresh_hash,
        expires_at,
    )

    return ok(
        data=TokenResponse(
            access_token=access_token,
            refresh_token=raw_refresh,
            expires_in=expires_in,
        ).model_dump(),
        message="Login successful",
    )


@router.post("/refresh")
async def refresh(body: RefreshRequest, conn: DBConn):
    """Exchange a refresh token for a new access + refresh token pair."""
    token_hash = hash_refresh_token(body.refresh_token)
    row = await conn.fetchrow(
        """
        SELECT rt.id, rt.user_id, rt.expires_at, rt.revoked, u.role, u.status
        FROM refresh_tokens rt
        JOIN users u ON u.id = rt.user_id
        WHERE rt.token_hash = $1
        """,
        token_hash,
    )
    if row is None:
        raise HTTPException(status_code=401, detail="Invalid refresh token")
    if row["revoked"]:
        raise HTTPException(status_code=401, detail="Refresh token revoked")
    if row["expires_at"].replace(tzinfo=timezone.utc) < datetime.now(timezone.utc):
        raise HTTPException(status_code=401, detail="Refresh token expired")
    if row["status"] != "active":
        raise HTTPException(status_code=403, detail="Account is not active")

    # Revoke old token
    await conn.execute("UPDATE refresh_tokens SET revoked = TRUE WHERE id = $1", row["id"])

    # Issue new pair
    access_token, expires_in = create_access_token(str(row["user_id"]), row["role"])
    raw_refresh, new_hash = create_refresh_token()
    expires_at = datetime.now(timezone.utc) + timedelta(days=settings.jwt_refresh_token_days)
    await conn.execute(
        "INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
        row["user_id"],
        new_hash,
        expires_at,
    )

    return ok(
        data=TokenResponse(
            access_token=access_token,
            refresh_token=raw_refresh,
            expires_in=expires_in,
        ).model_dump(),
        message="Token refreshed",
    )


@router.post("/send-otp")
async def send_otp(body: SendOTPRequest, user: CurrentUser, conn: DBConn):
    """Send a one-time password to the user's phone or email."""
    code = f"{secrets.randbelow(900000) + 100000}"  # 6-digit code
    expires_at = datetime.now(timezone.utc) + timedelta(minutes=10)

    await conn.execute(
        """
        INSERT INTO otp_codes (user_id, code, channel, purpose, expires_at)
        VALUES ($1, $2, $3, $4, $5)
        """,
        user["id"],
        code,
        body.channel,
        body.purpose,
        expires_at,
    )

    # STUB: log OTP instead of sending via SMS/email provider
    print(f"[OTP STUB] user={user['id']} channel={body.channel} code={code}")

    return ok(message="OTP sent successfully")


@router.post("/verify-otp")
async def verify_otp(body: VerifyOTPRequest, user: CurrentUser, conn: DBConn):
    """Verify a one-time password."""
    row = await conn.fetchrow(
        """
        SELECT id, code, expires_at, verified, attempts
        FROM otp_codes
        WHERE user_id = $1 AND purpose = $2 AND verified = FALSE
        ORDER BY created_at DESC
        LIMIT 1
        """,
        user["id"],
        body.purpose,
    )
    if row is None:
        raise HTTPException(status_code=400, detail="No pending OTP found")

    # Increment attempts
    attempts = row["attempts"] + 1
    await conn.execute("UPDATE otp_codes SET attempts = $1 WHERE id = $2", attempts, row["id"])

    if attempts > 5:
        raise HTTPException(status_code=429, detail="Too many attempts")

    if row["expires_at"].replace(tzinfo=timezone.utc) < datetime.now(timezone.utc):
        raise HTTPException(status_code=400, detail="OTP expired")

    if row["code"] != body.code:
        raise HTTPException(status_code=400, detail="Invalid OTP")

    # Mark verified
    await conn.execute("UPDATE otp_codes SET verified = TRUE WHERE id = $1", row["id"])

    # Update verification status
    if body.purpose == "verify_phone":
        await conn.execute(
            "UPDATE user_verification SET phone_verified = TRUE WHERE user_id = $1",
            user["id"],
        )
    elif body.purpose == "verify_email":
        await conn.execute(
            "UPDATE user_verification SET email_verified = TRUE WHERE user_id = $1",
            user["id"],
        )

    return ok(message="OTP verified successfully")


@router.post("/logout")
async def logout(user: CurrentUser, conn: DBConn):
    """Revoke all refresh tokens for the current user."""
    await conn.execute(
        "UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = $1 AND revoked = FALSE",
        user["id"],
    )
    return ok(message="Logged out successfully")
