"""JWT token creation and verification."""

from __future__ import annotations

import hashlib
import secrets
from datetime import datetime, timedelta, timezone

import jwt

from app.config import settings


def create_access_token(user_id: str, role: str) -> tuple[str, int]:
    """Create a signed JWT access token. Returns (token, expires_in_seconds)."""
    expires_delta = timedelta(minutes=settings.jwt_access_token_minutes)
    expire = datetime.now(timezone.utc) + expires_delta
    payload = {
        "sub": user_id,
        "role": role,
        "type": "access",
        "exp": expire,
        "iat": datetime.now(timezone.utc),
    }
    token = jwt.encode(payload, settings.jwt_secret, algorithm=settings.jwt_algorithm)
    return token, int(expires_delta.total_seconds())


def create_refresh_token() -> tuple[str, str]:
    """Create a random refresh token. Returns (raw_token, sha256_hash)."""
    raw = secrets.token_urlsafe(48)
    hashed = hashlib.sha256(raw.encode()).hexdigest()
    return raw, hashed


def decode_access_token(token: str) -> dict:
    """Decode and verify a JWT access token. Raises jwt.PyJWTError on failure."""
    payload = jwt.decode(token, settings.jwt_secret, algorithms=[settings.jwt_algorithm])
    if payload.get("type") != "access":
        raise jwt.InvalidTokenError("Not an access token")
    return payload


def hash_refresh_token(raw: str) -> str:
    """Hash a raw refresh token for DB lookup."""
    return hashlib.sha256(raw.encode()).hexdigest()
