"""Tests for auth endpoints."""

from __future__ import annotations

from datetime import datetime, timezone
from unittest.mock import AsyncMock
import uuid

import pytest

from app.auth import create_access_token, create_refresh_token, hash_refresh_token


@pytest.mark.asyncio
async def test_register_success(client):
    ac, mock_conn = client
    user_id = uuid.uuid4()

    # fetchval for duplicate check -> None (no duplicate)
    mock_conn.fetchval = AsyncMock(return_value=None)

    # fetchrow for INSERT RETURNING
    mock_conn.fetchrow = AsyncMock(return_value={
        "id": user_id,
        "email": "test@example.com",
        "role": "task_doer",
        "created_at": datetime.now(timezone.utc),
    })
    mock_conn.execute = AsyncMock()

    resp = await ac.post("/api/v1/auth/register", json={
        "email": "test@example.com",
        "password": "securepass123",
        "role": "doer",
    })
    assert resp.status_code == 201
    data = resp.json()
    assert data["success"] is True
    assert "access_token" in data["data"]
    assert "refresh_token" in data["data"]


@pytest.mark.asyncio
async def test_register_duplicate_email(client):
    ac, mock_conn = client

    # fetchval returns existing user ID (indicates duplicate)
    mock_conn.fetchval = AsyncMock(return_value=uuid.uuid4())
    mock_conn.fetchrow = AsyncMock(return_value=None)
    mock_conn.execute = AsyncMock()

    resp = await ac.post("/api/v1/auth/register", json={
        "email": "existing@example.com",
        "password": "securepass123",
        "role": "giver",
    })
    assert resp.status_code == 409


@pytest.mark.asyncio
async def test_register_invalid_role(client):
    ac, mock_conn = client
    resp = await ac.post("/api/v1/auth/register", json={
        "email": "test@example.com",
        "password": "securepass123",
        "role": "invalid",
    })
    assert resp.status_code == 422


@pytest.mark.asyncio
async def test_login_invalid_credentials(client):
    ac, mock_conn = client
    # fetchrow returns None => no user found
    mock_conn.fetchrow = AsyncMock(return_value=None)

    resp = await ac.post("/api/v1/auth/login", json={
        "email": "nobody@example.com",
        "password": "wrongpass",
    })
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_login_success(client):
    ac, mock_conn = client
    user_id = uuid.uuid4()
    from argon2 import PasswordHasher
    ph = PasswordHasher()
    pw_hash = ph.hash("correctpassword")

    mock_conn.fetchrow = AsyncMock(return_value={
        "id": user_id,
        "email": "alice@example.com",
        "password_hash": pw_hash,
        "role": "both",
        "status": "active",
    })
    mock_conn.execute = AsyncMock()

    resp = await ac.post("/api/v1/auth/login", json={
        "email": "alice@example.com",
        "password": "correctpassword",
    })
    assert resp.status_code == 200
    data = resp.json()
    assert data["success"] is True
    assert "access_token" in data["data"]


@pytest.mark.asyncio
async def test_refresh_invalid_token(client):
    ac, mock_conn = client
    # fetchrow returns None => no matching token
    mock_conn.fetchrow = AsyncMock(return_value=None)

    resp = await ac.post("/api/v1/auth/refresh", json={
        "refresh_token": "invalid-token",
    })
    assert resp.status_code == 401


def test_create_access_token():
    token, expires_in = create_access_token("user-123", "task_doer")
    assert isinstance(token, str)
    assert expires_in > 0


def test_create_refresh_token():
    raw, hashed = create_refresh_token()
    assert len(raw) > 0
    assert len(hashed) == 64  # SHA-256 hex


def test_hash_refresh_token():
    raw, expected = create_refresh_token()
    assert hash_refresh_token(raw) == expected
