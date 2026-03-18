"""Smoke test: verify the FastAPI app can be imported and the health endpoint responds."""

from unittest.mock import AsyncMock, patch

import pytest
from fastapi.testclient import TestClient


@pytest.fixture()
def client():
    """Create a TestClient with the DB pool init/close mocked out."""
    with (
        patch("app.database.init_pool", new_callable=AsyncMock),
        patch("app.database.close_pool", new_callable=AsyncMock),
    ):
        from app.main import app

        with TestClient(app) as c:
            yield c


def test_health_returns_ok(client):
    resp = client.get("/health")
    assert resp.status_code == 200
    assert resp.json() == {"status": "ok"}


def test_app_includes_auth_router(client):
    """Verify the auth router is mounted by checking the OpenAPI schema."""
    resp = client.get("/openapi.json")
    assert resp.status_code == 200
    paths = resp.json()["paths"]
    assert "/api/v1/auth/register" in paths
    assert "/api/v1/auth/login" in paths


def test_app_includes_profile_router(client):
    resp = client.get("/openapi.json")
    assert resp.status_code == 200
    paths = resp.json()["paths"]
    assert "/api/v1/profile/" in paths


def test_app_includes_tasks_router(client):
    resp = client.get("/openapi.json")
    assert resp.status_code == 200
    paths = resp.json()["paths"]
    assert "/api/v1/tasks/" in paths


def test_app_includes_categories_router(client):
    resp = client.get("/openapi.json")
    assert resp.status_code == 200
    paths = resp.json()["paths"]
    assert "/api/v1/categories/" in paths


def test_app_includes_notifications_router(client):
    resp = client.get("/openapi.json")
    assert resp.status_code == 200
    paths = resp.json()["paths"]
    assert "/api/v1/notifications/" in paths
