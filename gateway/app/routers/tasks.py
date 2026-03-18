import httpx
from fastapi import APIRouter, Depends, HTTPException, Request, status

from app.config import settings
from app.deps import get_client_type, get_current_user, get_db_conn
from app.schemas.envelope import err, ok

router = APIRouter(prefix="/api/v1/tasks", tags=["tasks"])

# Shared httpx client -- created once, reused across requests.
_http_client: httpx.AsyncClient | None = None


def _client() -> httpx.AsyncClient:
    global _http_client
    if _http_client is None or _http_client.is_closed:
        _http_client = httpx.AsyncClient(
            base_url=settings.xo_service_url,
            timeout=30.0,
        )
    return _http_client


async def _proxy(
    method: str,
    path: str,
    user: dict,
    *,
    json_body: dict | None = None,
    params: dict | None = None,
) -> dict:
    """Forward a request to the xo service and return the JSON response."""
    headers = {
        "X-User-ID": user["id"],
        "X-User-Role": user["role"],
        "Content-Type": "application/json",
    }
    try:
        resp = await _client().request(
            method,
            path,
            headers=headers,
            json=json_body,
            params=params,
        )
    except httpx.RequestError as exc:
        raise HTTPException(
            status_code=status.HTTP_502_BAD_GATEWAY,
            detail=err("XO_UNREACHABLE", f"Could not reach xo service: {exc}"),
        )

    # Pass through the xo response
    try:
        data = resp.json()
    except Exception:
        data = {"raw": resp.text}

    if resp.status_code >= 400:
        raise HTTPException(status_code=resp.status_code, detail=data)

    return data


# ---------------------------------------------------------------------------
# POST / -- create task
# ---------------------------------------------------------------------------

@router.post("/", status_code=status.HTTP_201_CREATED)
async def create_task(
    request: Request,
    user: dict = Depends(get_current_user),
    client_type: str = Depends(get_client_type),
    conn=Depends(get_db_conn),
):
    body = await request.json()

    # Enforce offline-task rule: if is_online is false and client is web, reject.
    is_online = body.get("is_online", False)
    if not is_online and client_type == "web":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail=err(
                "OFFLINE_TASK_WEB",
                "Offline tasks cannot be created from the web client. Use the mobile app.",
            ),
        )

    body["client_type"] = client_type
    return await _proxy("POST", "/tasks", user, json_body=body)


# ---------------------------------------------------------------------------
# GET / -- list tasks
# ---------------------------------------------------------------------------

@router.get("/")
async def list_tasks(
    request: Request,
    user: dict = Depends(get_current_user),
):
    return await _proxy("GET", "/tasks", user, params=dict(request.query_params))


# ---------------------------------------------------------------------------
# GET /{id}
# ---------------------------------------------------------------------------

@router.get("/{task_id}")
async def get_task(
    task_id: str,
    user: dict = Depends(get_current_user),
):
    return await _proxy("GET", f"/tasks/{task_id}", user)


# ---------------------------------------------------------------------------
# PUT /{id}
# ---------------------------------------------------------------------------

@router.put("/{task_id}")
async def update_task(
    task_id: str,
    request: Request,
    user: dict = Depends(get_current_user),
):
    body = await request.json()
    return await _proxy("PUT", f"/tasks/{task_id}", user, json_body=body)


# ---------------------------------------------------------------------------
# DELETE /{id}
# ---------------------------------------------------------------------------

@router.delete("/{task_id}")
async def delete_task(
    task_id: str,
    user: dict = Depends(get_current_user),
):
    return await _proxy("DELETE", f"/tasks/{task_id}", user)


# ---------------------------------------------------------------------------
# POST /{id}/accept
# ---------------------------------------------------------------------------

@router.post("/{task_id}/accept")
async def accept_task(
    task_id: str,
    user: dict = Depends(get_current_user),
):
    return await _proxy("POST", f"/tasks/{task_id}/accept", user)


# ---------------------------------------------------------------------------
# POST /{id}/complete
# ---------------------------------------------------------------------------

@router.post("/{task_id}/complete")
async def complete_task(
    task_id: str,
    user: dict = Depends(get_current_user),
):
    return await _proxy("POST", f"/tasks/{task_id}/complete", user)
