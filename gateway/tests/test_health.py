"""Tests for health check and basic gateway functionality."""

import pytest


@pytest.mark.asyncio
async def test_healthz(client):
    ac, _ = client
    resp = await ac.get("/healthz")
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "ok"
    assert data["service"] == "gateway"


@pytest.mark.asyncio
async def test_openapi_docs(client):
    ac, _ = client
    resp = await ac.get("/openapi.json")
    assert resp.status_code == 200
    data = resp.json()
    assert data["info"]["title"] == "xo API Gateway"
