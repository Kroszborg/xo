"""Tests for xo service proxy (tasks and devices)."""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock, patch
import uuid

import pytest
import httpx

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


# ─── List Tasks ─────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_list_tasks_success(client):
    """Test listing tasks via xo proxy."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    # Mock httpx response
    mock_response = MagicMock()
    mock_response.content = b'{"data":[{"id":"task-1"}]}'
    mock_response.status_code = 200
    mock_response.headers = {"content-type": "application/json"}

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(return_value=mock_response)
        mock_get_client.return_value = mock_client

        resp = await ac.get("/api/v1/tasks", headers=_auth_header())

    assert resp.status_code == 200
    assert b"task-1" in resp.content


@pytest.mark.asyncio
async def test_list_tasks_with_pagination(client):
    """Test page/limit translation to limit/offset."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_response = MagicMock()
    mock_response.content = b'{"data":[]}'
    mock_response.status_code = 200
    mock_response.headers = {"content-type": "application/json"}

    captured_params = {}

    async def capture_request(*args, **kwargs):
        captured_params.update(kwargs.get("params", {}))
        return mock_response

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(side_effect=capture_request)
        mock_get_client.return_value = mock_client

        resp = await ac.get("/api/v1/tasks?page=3&limit=10", headers=_auth_header())

    assert resp.status_code == 200
    # page=3, limit=10 -> offset=(3-1)*10=20
    assert captured_params.get("offset") == "20"
    assert captured_params.get("limit") == "10"


@pytest.mark.asyncio
async def test_list_tasks_xo_unavailable(client):
    """Test handling when xo service is unavailable."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(side_effect=httpx.RequestError("Connection refused"))
        mock_get_client.return_value = mock_client

        resp = await ac.get("/api/v1/tasks", headers=_auth_header())

    assert resp.status_code == 502
    data = resp.json()
    assert "xo service unavailable" in data["detail"]


@pytest.mark.asyncio
async def test_list_tasks_no_auth(client):
    """Test that listing tasks requires authentication."""
    ac, mock_conn = client

    resp = await ac.get("/api/v1/tasks")
    assert resp.status_code == 401


# ─── Create Task ────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_create_task_success(client):
    """Test creating a task via xo proxy."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    task_id = str(uuid.uuid4())
    mock_response = MagicMock()
    mock_response.content = f'{{"data":{{"id":"{task_id}"}}}}'.encode()
    mock_response.status_code = 201
    mock_response.headers = {"content-type": "application/json"}

    captured_headers = {}
    captured_content = b""

    async def capture_request(*args, **kwargs):
        nonlocal captured_headers, captured_content
        captured_headers = kwargs.get("headers", {})
        captured_content = kwargs.get("content", b"")
        return mock_response

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(side_effect=capture_request)
        mock_get_client.return_value = mock_client

        resp = await ac.post(
            "/api/v1/tasks",
            json={"category_id": "cat-1", "budget": 100},
            headers=_auth_header(),
        )

    assert resp.status_code == 201
    # Verify user context headers were forwarded
    assert captured_headers["X-User-ID"] == str(_USER_ID)
    assert captured_headers["X-User-Role"] == "both"


# ─── Get Task ───────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_get_task_success(client):
    """Test getting a single task."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    task_id = str(uuid.uuid4())
    mock_response = MagicMock()
    mock_response.content = f'{{"data":{{"id":"{task_id}","state":"active"}}}}'.encode()
    mock_response.status_code = 200
    mock_response.headers = {"content-type": "application/json"}

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(return_value=mock_response)
        mock_get_client.return_value = mock_client

        resp = await ac.get(f"/api/v1/tasks/{task_id}", headers=_auth_header())

    assert resp.status_code == 200
    assert task_id.encode() in resp.content


@pytest.mark.asyncio
async def test_get_task_not_found(client):
    """Test 404 response from xo is passed through."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_response = MagicMock()
    mock_response.content = b'{"error":"not_found"}'
    mock_response.status_code = 404
    mock_response.headers = {"content-type": "application/json"}

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(return_value=mock_response)
        mock_get_client.return_value = mock_client

        resp = await ac.get(f"/api/v1/tasks/{uuid.uuid4()}", headers=_auth_header())

    assert resp.status_code == 404


# ─── Update Task ────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_update_task_success(client):
    """Test updating a task via PATCH."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    task_id = str(uuid.uuid4())
    mock_response = MagicMock()
    mock_response.content = f'{{"data":{{"id":"{task_id}","budget":"200"}}}}'.encode()
    mock_response.status_code = 200
    mock_response.headers = {"content-type": "application/json"}

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(return_value=mock_response)
        mock_get_client.return_value = mock_client

        resp = await ac.patch(
            f"/api/v1/tasks/{task_id}",
            json={"budget": 200},
            headers=_auth_header(),
        )

    assert resp.status_code == 200


# ─── Cancel Task ────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_cancel_task_success(client):
    """Test cancelling a task via DELETE."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    task_id = str(uuid.uuid4())
    mock_response = MagicMock()
    mock_response.content = b'{"success":true}'
    mock_response.status_code = 200
    mock_response.headers = {"content-type": "application/json"}

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(return_value=mock_response)
        mock_get_client.return_value = mock_client

        resp = await ac.delete(f"/api/v1/tasks/{task_id}", headers=_auth_header())

    assert resp.status_code == 200


# ─── Accept Task ────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_accept_task_success(client):
    """Test accepting a task."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    task_id = str(uuid.uuid4())
    mock_response = MagicMock()
    mock_response.content = b'{"success":true}'
    mock_response.status_code = 200
    mock_response.headers = {"content-type": "application/json"}

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(return_value=mock_response)
        mock_get_client.return_value = mock_client

        resp = await ac.post(f"/api/v1/tasks/{task_id}/accept", headers=_auth_header())

    assert resp.status_code == 200


# ─── Complete Task ──────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_complete_task_success(client):
    """Test completing a task."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    task_id = str(uuid.uuid4())
    mock_response = MagicMock()
    mock_response.content = b'{"success":true}'
    mock_response.status_code = 200
    mock_response.headers = {"content-type": "application/json"}

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(return_value=mock_response)
        mock_get_client.return_value = mock_client

        resp = await ac.post(f"/api/v1/tasks/{task_id}/complete", headers=_auth_header())

    assert resp.status_code == 200


# ─── Device Tokens ──────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_register_device_success(client):
    """Test registering a device token."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_response = MagicMock()
    mock_response.content = b'{"success":true}'
    mock_response.status_code = 200
    mock_response.headers = {"content-type": "application/json"}

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(return_value=mock_response)
        mock_get_client.return_value = mock_client

        resp = await ac.put(
            "/api/v1/devices",
            json={"token": "fcm-token-123", "platform": "android"},
            headers=_auth_header(),
        )

    assert resp.status_code == 200


@pytest.mark.asyncio
async def test_unregister_device_success(client):
    """Test unregistering a device token."""
    ac, mock_conn = client
    from app.main import app
    _override_current_user(app)

    mock_response = MagicMock()
    mock_response.content = b'{"success":true}'
    mock_response.status_code = 200
    mock_response.headers = {"content-type": "application/json"}

    with patch("app.routers.tasks._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_client.request = AsyncMock(return_value=mock_response)
        mock_get_client.return_value = mock_client

        resp = await ac.delete("/api/v1/devices", headers=_auth_header())

    assert resp.status_code == 200
