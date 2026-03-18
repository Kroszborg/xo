import asyncpg
import httpx
from fastapi import APIRouter, Depends, HTTPException, status

from app.config import settings
from app.deps import get_current_user, get_db_conn
from app.schemas.envelope import err, ok

router = APIRouter(prefix="/api/v1", tags=["reviews"])


async def _proxy_to_xo(method: str, path: str, user: dict, json_body: dict | None = None):
    """Forward request to xo service."""
    async with httpx.AsyncClient() as client:
        resp = await client.request(
            method,
            f"{settings.xo_service_url}{path}",
            json=json_body,
            headers={"X-User-ID": user["id"], "X-User-Role": user["role"]},
            timeout=10.0,
        )
    return resp.json(), resp.status_code


@router.post("/tasks/{task_id}/reviews")
async def create_review(
    task_id: str,
    body: dict,
    user: dict = Depends(get_current_user),
):
    data, status_code = await _proxy_to_xo(
        "POST", f"/api/v1/tasks/{task_id}/reviews", user, body
    )
    if status_code >= 400:
        raise HTTPException(status_code=status_code, detail=data)
    return data


@router.get("/tasks/{task_id}/reviews")
async def get_task_reviews(
    task_id: str,
    user: dict = Depends(get_current_user),
):
    data, status_code = await _proxy_to_xo(
        "GET", f"/api/v1/tasks/{task_id}/reviews", user
    )
    if status_code >= 400:
        raise HTTPException(status_code=status_code, detail=data)
    return data


@router.get("/users/{user_id}/reviews")
async def get_user_reviews(
    user_id: str,
    user: dict = Depends(get_current_user),
):
    data, status_code = await _proxy_to_xo(
        "GET", f"/api/v1/users/{user_id}/reviews", user
    )
    if status_code >= 400:
        raise HTTPException(status_code=status_code, detail=data)
    return data


@router.post("/tasks/{task_id}/dispute")
async def create_dispute(
    task_id: str,
    body: dict,
    user: dict = Depends(get_current_user),
):
    data, status_code = await _proxy_to_xo(
        "POST", f"/api/v1/tasks/{task_id}/dispute", user, body
    )
    if status_code >= 400:
        raise HTTPException(status_code=status_code, detail=data)
    return data
