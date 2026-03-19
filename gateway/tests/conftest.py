"""Tests for the gateway API endpoints."""

from __future__ import annotations

import asyncio
import os
import sys

import pytest
import pytest_asyncio
from httpx import ASGITransport, AsyncClient
from unittest.mock import AsyncMock, MagicMock, patch

# Ensure app is importable
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))


@pytest.fixture(scope="session")
def event_loop():
    """Create an event loop for the session."""
    loop = asyncio.new_event_loop()
    yield loop
    loop.close()


class MockConn:
    """Mock asyncpg connection that supports configurable responses."""

    def __init__(self):
        self.fetchrow = AsyncMock(return_value=None)
        self.fetchval = AsyncMock(return_value=None)
        self.fetch = AsyncMock(return_value=[])
        self.execute = AsyncMock(return_value="UPDATE 1")


@pytest_asyncio.fixture
async def client():
    """Create a test client using dependency overrides for DB connection."""
    mock_conn = MockConn()

    # Patch init_pool and close_pool to be no-ops
    with patch("app.database.init_pool", AsyncMock(return_value=MagicMock())), \
         patch("app.database.close_pool", AsyncMock()):

        from app.main import app
        from app.deps import get_db

        # Override the get_db dependency to yield our mock connection
        async def override_get_db():
            yield mock_conn

        app.dependency_overrides[get_db] = override_get_db

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as ac:
            yield ac, mock_conn

        # Clean up overrides
        app.dependency_overrides.clear()
