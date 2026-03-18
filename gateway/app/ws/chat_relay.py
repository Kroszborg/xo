import httpx
from app.config import settings


async def moderate_message(content: str) -> dict:
    """Send message to xo's internal moderation endpoint."""
    async with httpx.AsyncClient() as client:
        resp = await client.post(
            f"{settings.xo_service_url}/internal/chat/moderate",
            json={"content": content},
            timeout=15.0,
        )
        if resp.status_code != 200:
            return {"data": {"status": "clean", "sanitized": content, "flags": {}}}
        return resp.json()
