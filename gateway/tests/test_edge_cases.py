"""Edge case and error path tests."""

from __future__ import annotations

from datetime import datetime, timezone, timedelta
from unittest.mock import AsyncMock
import uuid

import pytest

from app.auth import create_access_token, decode_access_token, create_refresh_token, hash_refresh_token


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


# ─── JWT Token Tests ────────────────────────────────────────────────────

def test_decode_valid_token():
    """Test decoding a valid access token."""
    token, _ = create_access_token("user-123", "task_doer")
    payload = decode_access_token(token)
    assert payload["sub"] == "user-123"
    assert payload["role"] == "task_doer"
    assert payload["type"] == "access"


def test_decode_invalid_token():
    """Test decoding an invalid token raises exception."""
    import jwt
    with pytest.raises(jwt.exceptions.DecodeError):
        decode_access_token("invalid.token.here")


def test_decode_expired_token():
    """Test decoding an expired token raises ExpiredSignatureError."""
    import jwt
    from app.config import settings

    expired_payload = {
        "sub": "user-123",
        "role": "task_doer",
        "type": "access",
        "exp": datetime.now(timezone.utc) - timedelta(hours=1),
        "iat": datetime.now(timezone.utc) - timedelta(hours=2),
    }
    token = jwt.encode(expired_payload, settings.jwt_secret, algorithm=settings.jwt_algorithm)
    with pytest.raises(jwt.exceptions.ExpiredSignatureError):
        decode_access_token(token)


# ─── Auth Endpoint Edge Cases ───────────────────────────────────────────

@pytest.mark.asyncio
async def test_register_with_phone(client):
    """Test registration with optional phone number."""
    ac, mock_conn = client
    user_id = uuid.uuid4()
    mock_conn.fetchval = AsyncMock(return_value=None)
    mock_conn.fetchrow = AsyncMock(return_value={
        "id": user_id,
        "email": "phone@example.com",
        "role": "task_giver",
        "created_at": datetime.now(timezone.utc),
    })
    mock_conn.execute = AsyncMock()

    resp = await ac.post("/api/v1/auth/register", json={
        "email": "phone@example.com",
        "password": "securepass123",
        "role": "giver",
        "phone": "+1234567890",
    })
    assert resp.status_code == 201


@pytest.mark.asyncio
async def test_login_suspended_user(client):
    """Test login fails for suspended users (returns 403)."""
    ac, mock_conn = client
    from argon2 import PasswordHasher
    ph = PasswordHasher()

    mock_conn.fetchrow = AsyncMock(return_value={
        "id": uuid.uuid4(),
        "email": "suspended@example.com",
        "password_hash": ph.hash("password"),
        "role": "task_doer",
        "status": "suspended",  # Not active
    })

    resp = await ac.post("/api/v1/auth/login", json={
        "email": "suspended@example.com",
        "password": "password",
    })
    assert resp.status_code == 403


@pytest.mark.asyncio
async def test_login_wrong_password(client):
    """Test login fails with wrong password."""
    ac, mock_conn = client
    from argon2 import PasswordHasher
    ph = PasswordHasher()

    mock_conn.fetchrow = AsyncMock(return_value={
        "id": uuid.uuid4(),
        "email": "test@example.com",
        "password_hash": ph.hash("correctpassword"),
        "role": "task_doer",
        "status": "active",
    })

    resp = await ac.post("/api/v1/auth/login", json={
        "email": "test@example.com",
        "password": "wrongpassword",
    })
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_refresh_revoked_token(client):
    """Test refresh fails for revoked tokens."""
    ac, mock_conn = client

    # Return a row but with revoked=True
    mock_conn.fetchrow = AsyncMock(return_value={
        "user_id": uuid.uuid4(),
        "expires_at": datetime.now(timezone.utc) + timedelta(days=1),
        "revoked": True,
    })

    resp = await ac.post("/api/v1/auth/refresh", json={
        "refresh_token": "some-token",
    })
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_refresh_expired_token(client):
    """Test refresh fails for expired tokens."""
    ac, mock_conn = client

    mock_conn.fetchrow = AsyncMock(return_value={
        "user_id": uuid.uuid4(),
        "expires_at": datetime.now(timezone.utc) - timedelta(days=1),  # Expired
        "revoked": False,
    })

    resp = await ac.post("/api/v1/auth/refresh", json={
        "refresh_token": "some-token",
    })
    assert resp.status_code == 401


# ─── Profile Update Tests ───────────────────────────────────────────────

@pytest.mark.asyncio
async def test_update_profile(client):
    """Test profile update."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_conn.execute = AsyncMock()

    resp = await ac.patch("/api/v1/profile", json={
        "display_name": "New Name",
        "first_name": "John",
        "last_name": "Doe",
        "bio": "Updated bio",
        "gender": "male",
    }, headers=_auth_header())
    assert resp.status_code == 200


@pytest.mark.asyncio
async def test_update_profile_invalid_gender(client):
    """Test profile update with invalid gender."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    resp = await ac.patch("/api/v1/profile", json={
        "gender": "invalid",
    }, headers=_auth_header())
    assert resp.status_code == 422


# ─── Location Tests ─────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_update_location_invalid_lat(client):
    """Test location update with invalid latitude."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    resp = await ac.put("/api/v1/location", json={
        "lat": 91.0,  # Invalid: > 90
        "lng": -74.006,
    }, headers=_auth_header())
    assert resp.status_code == 422


@pytest.mark.asyncio
async def test_update_location_invalid_lng(client):
    """Test location update with invalid longitude."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    resp = await ac.put("/api/v1/location", json={
        "lat": 40.7128,
        "lng": 181.0,  # Invalid: > 180
    }, headers=_auth_header())
    assert resp.status_code == 422


@pytest.mark.asyncio
async def test_get_location_exists(client):
    """Test getting location when one exists."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    from decimal import Decimal
    mock_conn.fetchrow = AsyncMock(return_value={
        "lat": Decimal("40.7128"),
        "lng": Decimal("-74.006"),
        "accuracy_m": Decimal("10.5"),
        "updated_at": datetime.now(timezone.utc),
    })

    resp = await ac.get("/api/v1/location", headers=_auth_header())
    assert resp.status_code == 200
    data = resp.json()
    assert float(data["data"]["lat"]) == pytest.approx(40.7128, rel=1e-4)


# ─── Payment Method Tests ───────────────────────────────────────────────

@pytest.mark.asyncio
async def test_create_payment_method_invalid_type(client):
    """Test creating payment method with invalid type."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    resp = await ac.post("/api/v1/payment-methods", json={
        "method_type": "bitcoin",  # Not in allowed list
        "account_ref": "1234",
    }, headers=_auth_header())
    assert resp.status_code == 422


@pytest.mark.asyncio
async def test_update_payment_method(client):
    """Test updating a payment method."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    method_id = uuid.uuid4()
    mock_conn.fetchval = AsyncMock(return_value=method_id)  # Verify ownership
    mock_conn.execute = AsyncMock()

    resp = await ac.patch(f"/api/v1/payment-methods/{method_id}", json={
        "is_default": True,
    }, headers=_auth_header())
    assert resp.status_code == 200


@pytest.mark.asyncio
async def test_update_payment_method_not_found(client):
    """Test updating a non-existent payment method."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    method_id = uuid.uuid4()
    mock_conn.fetchval = AsyncMock(return_value=None)  # Not found

    resp = await ac.patch(f"/api/v1/payment-methods/{method_id}", json={
        "is_default": True,
    }, headers=_auth_header())
    assert resp.status_code == 404


@pytest.mark.asyncio
async def test_delete_payment_method(client):
    """Test deleting a payment method."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    method_id = uuid.uuid4()
    mock_conn.fetchval = AsyncMock(return_value=method_id)
    mock_conn.execute = AsyncMock()

    resp = await ac.delete(f"/api/v1/payment-methods/{method_id}", headers=_auth_header())
    assert resp.status_code == 200


@pytest.mark.asyncio
async def test_delete_payment_method_not_found(client):
    """Test deleting a non-existent payment method."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    method_id = uuid.uuid4()
    mock_conn.execute = AsyncMock(return_value="DELETE 0")

    resp = await ac.delete(f"/api/v1/payment-methods/{method_id}", headers=_auth_header())
    assert resp.status_code == 404


# ─── Address Tests ──────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_create_address_full(client):
    """Test creating an address with all fields."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    addr_id = uuid.uuid4()
    mock_conn.fetchrow = AsyncMock(return_value={"id": addr_id})
    mock_conn.execute = AsyncMock()

    resp = await ac.post("/api/v1/addresses", json={
        "label": "Work",
        "line1": "123 Business St",
        "line2": "Suite 500",
        "city": "Boston",
        "state": "MA",
        "postal_code": "02101",
        "country": "US",
        "is_default": True,
    }, headers=_auth_header())
    assert resp.status_code == 201


@pytest.mark.asyncio
async def test_delete_address(client):
    """Test deleting an address."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    addr_id = uuid.uuid4()
    mock_conn.fetchval = AsyncMock(return_value=addr_id)
    mock_conn.execute = AsyncMock()

    resp = await ac.delete(f"/api/v1/addresses/{addr_id}", headers=_auth_header())
    assert resp.status_code == 200


@pytest.mark.asyncio
async def test_delete_address_not_found(client):
    """Test deleting a non-existent address."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    addr_id = uuid.uuid4()
    mock_conn.execute = AsyncMock(return_value="DELETE 0")

    resp = await ac.delete(f"/api/v1/addresses/{addr_id}", headers=_auth_header())
    assert resp.status_code == 404


# ─── Onboarding Tests ───────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_onboarding_step_2_skills(client):
    """Test onboarding step 2: skills."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_conn.fetchrow = AsyncMock(return_value={"onboarding_step": 1})  # At step 1
    mock_conn.fetchval = AsyncMock(return_value=1)  # Current step for _advance_step
    mock_conn.execute = AsyncMock()

    resp = await ac.post("/api/v1/onboarding/step/2", json={
        "core_skills": [{"skill_name": "Python"}, {"skill_name": "JavaScript"}],
        "other_skills": [{"skill_name": "Docker"}, {"skill_name": "AWS"}],
    }, headers=_auth_header())
    assert resp.status_code == 200


@pytest.mark.asyncio
async def test_onboarding_step_out_of_order(client):
    """Test that skipping onboarding steps is allowed (lazy progression)."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    # User at step 1, trying to submit step 5 — implementation allows this
    mock_conn.fetchrow = AsyncMock(return_value={"onboarding_step": 1})
    mock_conn.fetchval = AsyncMock(return_value=1)
    mock_conn.execute = AsyncMock()

    resp = await ac.post("/api/v1/onboarding/step/5", json={
        "certificates": [],
    }, headers=_auth_header())
    # The _advance_step logic allows target_step <= current + 1, but doesn't reject
    # further steps — it just won't advance. Returns 200 regardless.
    assert resp.status_code == 200


@pytest.mark.asyncio
async def test_onboarding_invalid_step(client):
    """Test invalid onboarding step number (step 8+ not defined)."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    # Step 8 doesn't exist, should return 404 (no route)
    resp = await ac.post("/api/v1/onboarding/step/8", json={}, headers=_auth_header())
    assert resp.status_code == 404


# ─── Dashboard Tests ────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_dashboard_missing_data(client):
    """Test dashboard handles missing behavior metrics gracefully."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_conn.fetchrow = AsyncMock(side_effect=[
        {"onboarding_step": 3, "onboarding_done": False},  # profile
        None,  # task_stats - missing
        None,  # doer_stats - missing
        None,  # behavior_metrics - missing
        {"email_verified": False, "phone_verified": False, "id_verified": False},  # verification
    ])

    resp = await ac.get("/api/v1/dashboard", headers=_auth_header())
    assert resp.status_code == 200
    data = resp.json()
    # Should return zero defaults for missing data
    assert data["data"]["tasks_as_giver"]["active"] == 0
