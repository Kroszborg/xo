"""xo service proxy: forwards task-domain requests to the xo Go service."""

from __future__ import annotations

from fastapi import APIRouter, HTTPException, Request, Response
import httpx

from app.config import settings
from app.deps import CurrentUser

router = APIRouter(prefix="/api/v1", tags=["tasks"])

# Lazily initialized httpx async client
_client: httpx.AsyncClient | None = None


def _get_client() -> httpx.AsyncClient:
    global _client
    if _client is None:
        _client = httpx.AsyncClient(
            base_url=settings.xo_service_url,
            timeout=30.0,
        )
    return _client


async def _proxy(method: str, path: str, request: Request, user: dict | None = None) -> Response:
    """Forward a request to xo and return its response."""
    client = _get_client()

    # Build headers, forward auth context as X-User-ID
    headers = {}
    if user:
        headers["X-User-ID"] = str(user["id"])
        headers["X-User-Role"] = user.get("role", "")

    # Read body for non-GET requests
    body = None
    if method.upper() not in ("GET", "HEAD", "DELETE"):
        body = await request.body()

    # Forward query params
    url = f"/api/v1{path}"
    params = dict(request.query_params)

    # Translate page/limit to limit/offset for xo
    if "page" in params and "limit" in params:
        try:
            page = int(params.pop("page"))
            limit = int(params.get("limit", "20"))
            params["offset"] = str((page - 1) * limit)
        except ValueError:
            pass

    try:
        resp = await client.request(
            method=method,
            url=url,
            params=params,
            headers=headers,
            content=body,
        )
    except httpx.RequestError as e:
        raise HTTPException(status_code=502, detail=f"xo service unavailable: {e}")

    return Response(
        content=resp.content,
        status_code=resp.status_code,
        headers=dict(resp.headers),
        media_type=resp.headers.get("content-type", "application/json"),
    )


# ─── Task CRUD ────────────────────────────────────────────────────────────────

@router.post("/tasks")
async def create_task(request: Request, user: CurrentUser):
    """Create a new task (proxied to xo)."""
    return await _proxy("POST", "/tasks", request, user)


@router.get("/tasks")
async def list_tasks(request: Request, user: CurrentUser):
    """List tasks (proxied to xo)."""
    return await _proxy("GET", "/tasks", request, user)


@router.get("/tasks/{task_id}")
async def get_task(task_id: str, request: Request, user: CurrentUser):
    """Get a single task (proxied to xo)."""
    return await _proxy("GET", f"/tasks/{task_id}", request, user)


@router.patch("/tasks/{task_id}")
async def update_task(task_id: str, request: Request, user: CurrentUser):
    """Update a task (proxied to xo)."""
    return await _proxy("PATCH", f"/tasks/{task_id}", request, user)


@router.delete("/tasks/{task_id}")
async def cancel_task(task_id: str, request: Request, user: CurrentUser):
    """Cancel a task (proxied to xo)."""
    return await _proxy("DELETE", f"/tasks/{task_id}", request, user)


@router.post("/tasks/{task_id}/accept")
async def accept_task(task_id: str, request: Request, user: CurrentUser):
    """Accept a task (proxied to xo)."""
    return await _proxy("POST", f"/tasks/{task_id}/accept", request, user)


@router.post("/tasks/{task_id}/complete")
async def complete_task(task_id: str, request: Request, user: CurrentUser):
    """Complete a task (proxied to xo)."""
    return await _proxy("POST", f"/tasks/{task_id}/complete", request, user)


# ─── Device Tokens ────────────────────────────────────────────────────────────

@router.put("/devices")
async def register_device(request: Request, user: CurrentUser):
    """Register a device token (proxied to xo)."""
    return await _proxy("PUT", "/devices", request, user)


@router.delete("/devices")
async def unregister_device(request: Request, user: CurrentUser):
    """Unregister a device token (proxied to xo)."""
    return await _proxy("DELETE", "/devices", request, user)
