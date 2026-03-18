import asyncpg
from app.auth import create_access_token, create_refresh_token, hash_token
from app.config import settings
from datetime import datetime, timedelta, timezone


async def find_or_create_user(
    conn: asyncpg.Connection,
    email: str,
    provider: str,
    provider_user_id: str,
    provider_email: str | None = None,
    full_name: str | None = None,
) -> dict:
    """Find existing user by OAuth provider or create a new one.

    Returns dict with user_id, role, is_new.
    """
    # Check if provider link exists
    row = await conn.fetchrow(
        """SELECT user_id FROM user_auth_providers
           WHERE provider = $1 AND provider_user_id = $2""",
        provider, provider_user_id,
    )
    if row:
        user = await conn.fetchrow(
            "SELECT id, role FROM users WHERE id = $1", row["user_id"]
        )
        return {"user_id": str(user["id"]), "role": user["role"], "is_new": False}

    # Check if user exists by email
    existing = await conn.fetchrow(
        "SELECT id, role FROM users WHERE email = $1", email
    )

    async with conn.transaction():
        if existing:
            user_id = existing["id"]
            role = existing["role"]
            is_new = False
        else:
            # Create new user (no password — OAuth only)
            user_id = await conn.fetchval(
                """INSERT INTO users (email, password_hash, role)
                   VALUES ($1, NULL, 'task_doer')
                   RETURNING id""",
                email,
            )
            role = "task_doer"
            is_new = True

            # Create profile
            await conn.execute(
                "INSERT INTO user_profiles (user_id, full_name) VALUES ($1, $2)",
                user_id, full_name,
            )
            # Create behavior metrics
            await conn.execute(
                "INSERT INTO user_behavior_metrics (user_id) VALUES ($1)",
                user_id,
            )

        # Link provider
        await conn.execute(
            """INSERT INTO user_auth_providers
               (user_id, provider, provider_user_id, provider_email)
               VALUES ($1, $2, $3, $4)
               ON CONFLICT (provider, provider_user_id) DO NOTHING""",
            user_id, provider, provider_user_id, provider_email,
        )

    return {"user_id": str(user_id), "role": role, "is_new": is_new}


async def issue_tokens(conn: asyncpg.Connection, user_id: str, role: str) -> dict:
    """Issue access + refresh tokens and persist the refresh hash."""
    import uuid as uuid_mod
    access = create_access_token(user_id, role)
    refresh = create_refresh_token(user_id)

    await conn.execute(
        """INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
           VALUES ($1, $2, $3)""",
        uuid_mod.UUID(user_id),
        hash_token(refresh),
        datetime.now(timezone.utc) + timedelta(days=settings.jwt_refresh_token_expire_days),
    )

    return {
        "access_token": access,
        "refresh_token": refresh,
        "token_type": "bearer",
        "user_id": user_id,
        "role": role,
    }
