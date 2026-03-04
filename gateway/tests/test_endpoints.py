"""Tests for dashboard, location, verification, and config endpoints."""

from __future__ import annotations

from unittest.mock import AsyncMock
import uuid

import pytest

from app.auth import create_access_token

_USER_ID = uuid.UUID("b1000000-0000-0000-0000-000000000001")
_USER = {"id": _USER_ID, "email": "alice@example.com", "phone": "+1555", "role": "both", "status": "active"}


def _auth_header():
    token, _ = create_access_token(str(_USER_ID), "both")
    return {"Authorization": f"Bearer {token}"}


def _override_current_user(app):
    from app.deps import get_current_user
    async def override():
        return dict(_USER)
    app.dependency_overrides[get_current_user] = override


# ─── Dashboard ──────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_dashboard(client):
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_conn.fetchrow = AsyncMock(side_effect=[
        # gateway_user_profile
        {"onboarding_step": 5, "onboarding_done": False},
        # task_stats
        {"active_tasks": 2, "completed_tasks": 3, "accepted_tasks": 1, "open_tasks": 4, "total_tasks": 10},
        # doer_stats
        {"tasks_accepted": 5, "total_earned": 1500.00},
        # behavior_metrics
        {"acceptance_rate": 0.85, "completion_rate": 0.90, "reliability_score": 95.0, "total_tasks_completed": 8},
        # verification
        {"email_verified": True, "phone_verified": False, "id_verified": False},
    ])

    resp = await ac.get("/api/v1/dashboard", headers=_auth_header())
    assert resp.status_code == 200
    data = resp.json()
    assert data["success"] is True
    assert data["data"]["onboarding"]["step"] == 5
    assert data["data"]["tasks_as_giver"]["active"] == 2
    assert data["data"]["tasks_as_doer"]["total_earned"] == 1500.0
    assert data["data"]["behavior"]["reliability_score"] == 95.0
    assert data["data"]["verification"]["email_verified"] is True


# ─── Location ───────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_update_location(client):
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_conn.execute = AsyncMock()

    resp = await ac.put("/api/v1/location", json={
        "lat": 40.7128,
        "lng": -74.006,
        "accuracy_m": 10.5,
    }, headers=_auth_header())
    assert resp.status_code == 200
    data = resp.json()
    assert data["success"] is True


@pytest.mark.asyncio
async def test_get_location_none(client):
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_conn.fetchrow = AsyncMock(return_value=None)

    resp = await ac.get("/api/v1/location", headers=_auth_header())
    assert resp.status_code == 200
    data = resp.json()
    assert data["data"] is None


# ─── Verification ───────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_verification_status(client):
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_conn.fetchrow = AsyncMock(return_value={
        "email_verified": True,
        "phone_verified": False,
        "id_verified": False,
        "verified_at": None,
    })

    resp = await ac.get("/api/v1/verification/status", headers=_auth_header())
    assert resp.status_code == 200
    data = resp.json()
    assert data["success"] is True
    assert data["data"]["email_verified"] is True


# ─── Payment Methods ────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_list_payment_methods_empty(client):
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_conn.fetch = AsyncMock(return_value=[])

    resp = await ac.get("/api/v1/payment-methods", headers=_auth_header())
    assert resp.status_code == 200
    data = resp.json()
    assert data["success"] is True
    assert data["data"] == []


@pytest.mark.asyncio
async def test_create_payment_method(client):
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    method_id = uuid.uuid4()
    mock_conn.fetchrow = AsyncMock(return_value={"id": method_id})

    resp = await ac.post("/api/v1/payment-methods", json={
        "method_type": "bank_account",
        "provider": "Chase",
        "account_ref": "****1234",
    }, headers=_auth_header())
    assert resp.status_code == 201
    data = resp.json()
    assert data["success"] is True
    assert data["data"]["id"] == str(method_id)


# ─── Categories (public, no auth) ──────────────────────────────────────

@pytest.mark.asyncio
async def test_list_categories(client):
    ac, mock_conn = client
    cat_id = uuid.uuid4()
    mock_conn.fetch = AsyncMock(return_value=[
        {"id": cat_id, "name": "Web Dev", "icon_url": "/icons/web.png"},
    ])

    resp = await ac.get("/api/v1/categories")
    assert resp.status_code == 200
    data = resp.json()
    assert data["success"] is True
    assert len(data["data"]) == 1
    assert data["data"][0]["name"] == "Web Dev"


# ─── FAQs (public, no auth) ────────────────────────────────────────────

@pytest.mark.asyncio
async def test_list_faqs(client):
    ac, mock_conn = client
    faq_id = uuid.uuid4()
    mock_conn.fetch = AsyncMock(return_value=[
        {"id": faq_id, "question": "How to start?", "answer": "Sign up first", "category": "getting_started"},
    ])

    resp = await ac.get("/api/v1/faqs")
    assert resp.status_code == 200
    data = resp.json()
    assert data["success"] is True
    assert data["data"][0]["question"] == "How to start?"


# ─── Addresses ──────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_list_addresses(client):
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    from datetime import datetime, timezone
    addr_id = uuid.uuid4()
    mock_conn.fetch = AsyncMock(return_value=[
        {
            "id": addr_id, "user_id": _USER_ID,
            "label": "Home", "line1": "123 Main St", "line2": None,
            "city": "NYC", "state": "NY", "postal_code": "10001", "country": "US",
            "is_default": True,
            "created_at": datetime.now(timezone.utc),
            "updated_at": datetime.now(timezone.utc),
        },
    ])

    resp = await ac.get("/api/v1/addresses", headers=_auth_header())
    assert resp.status_code == 200
    data = resp.json()
    assert data["success"] is True
    assert len(data["data"]) == 1
    assert data["data"][0]["city"] == "NYC"


@pytest.mark.asyncio
async def test_create_address(client):
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    addr_id = uuid.uuid4()
    mock_conn.fetchrow = AsyncMock(return_value={"id": addr_id})
    mock_conn.execute = AsyncMock()

    resp = await ac.post("/api/v1/addresses", json={
        "line1": "456 Elm St",
        "city": "Boston",
        "country": "US",
    }, headers=_auth_header())
    assert resp.status_code == 201
    data = resp.json()
    assert data["data"]["id"] == str(addr_id)
