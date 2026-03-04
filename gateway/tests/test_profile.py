"""Tests for profile and onboarding endpoints."""

from __future__ import annotations

from unittest.mock import AsyncMock
import uuid

import pytest

from app.auth import create_access_token


_USER_ID = uuid.UUID("b1000000-0000-0000-0000-000000000001")
_USER = {"id": _USER_ID, "email": "alice@example.com", "phone": "+1555", "role": "both", "status": "active"}


def _auth_header(user_id: str = "b1000000-0000-0000-0000-000000000001", role: str = "both"):
    token, _ = create_access_token(user_id, role)
    return {"Authorization": f"Bearer {token}"}


def _override_current_user(app):
    """Override get_current_user to return a fixed user dict."""
    from app.deps import get_current_user

    async def override():
        return dict(_USER)

    app.dependency_overrides[get_current_user] = override


@pytest.mark.asyncio
async def test_get_profile_unauthenticated(client):
    ac, _ = client
    # Remove current_user override if set, so auth is required
    from app.main import app
    from app.deps import get_current_user
    app.dependency_overrides.pop(get_current_user, None)

    resp = await ac.get("/api/v1/profile")
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_get_profile_success(client):
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    # get_profile: SELECT from gateway_user_profile JOIN users
    mock_conn.fetchrow = AsyncMock(return_value={
        "user_id": _USER_ID, "email": "alice@example.com", "phone": "+1555", "role": "both",
        "display_name": "Alice", "first_name": "Alice", "last_name": "Smith",
        "bio": "Hello", "avatar_url": None, "date_of_birth": None, "gender": None,
        "onboarding_step": 3, "onboarding_done": False,
    })

    resp = await ac.get("/api/v1/profile", headers=_auth_header())
    assert resp.status_code == 200
    data = resp.json()
    assert data["success"] is True
    assert data["data"]["display_name"] == "Alice"


@pytest.mark.asyncio
async def test_onboarding_step_1(client):
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_conn.execute = AsyncMock()
    mock_conn.fetchval = AsyncMock(return_value=0)

    resp = await ac.post(
        "/api/v1/onboarding/step/1",
        json={"role": "doer"},
        headers=_auth_header(),
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["success"] is True


@pytest.mark.asyncio
async def test_onboarding_status(client):
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_conn.fetchrow = AsyncMock(return_value={"onboarding_step": 4, "onboarding_done": False})

    resp = await ac.get("/api/v1/onboarding/status", headers=_auth_header())
    assert resp.status_code == 200
    data = resp.json()
    assert data["data"]["current_step"] == 4
