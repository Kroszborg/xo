import httpx
from fastapi import APIRouter, Depends, HTTPException, Request, status

from app.config import settings
from app.deps import get_current_user
from app.schemas.envelope import err

router = APIRouter(prefix="/api/v1/nearby", tags=["nearby"])

_http_client: httpx.AsyncClient | None = None


def _client() -> httpx.AsyncClient:
    global _http_client
    if _http_client is None or _http_client.is_closed:
        _http_client = httpx.AsyncClient(
            base_url=settings.xo_service_url,
            timeout=30.0,
        )
    return _http_client


@router.get("/users")
async def nearby_users(
    request: Request,
    user: dict = Depends(get_current_user),
):
    """Proxy to xo service nearby users endpoint."""
    headers = {
        "X-User-ID": user["id"],
        "X-User-Role": user["role"],
    }
    try:
        resp = await _client().request(
            "GET",
            "/api/v1/nearby/users",
            headers=headers,
            params=dict(request.query_params),
        )
    except httpx.RequestError as exc:
        raise HTTPException(
            status_code=status.HTTP_502_BAD_GATEWAY,
            detail=err("XO_UNREACHABLE", f"Could not reach xo service: {exc}"),
        )

    try:
        data = resp.json()
    except Exception:
        data = {"raw": resp.text}

    if resp.status_code >= 400:
        raise HTTPException(status_code=resp.status_code, detail=data)

    return data
