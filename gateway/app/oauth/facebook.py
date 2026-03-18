import httpx
from app.config import settings


def get_facebook_redirect_url() -> str:
    params = {
        "client_id": settings.facebook_client_id,
        "redirect_uri": settings.facebook_redirect_uri,
        "scope": "email,public_profile",
        "response_type": "code",
    }
    qs = "&".join(f"{k}={v}" for k, v in params.items())
    return f"https://www.facebook.com/v19.0/dialog/oauth?{qs}"


async def exchange_code(code: str) -> dict:
    async with httpx.AsyncClient() as client:
        token_resp = await client.get(
            "https://graph.facebook.com/v19.0/oauth/access_token",
            params={
                "client_id": settings.facebook_client_id,
                "client_secret": settings.facebook_client_secret,
                "redirect_uri": settings.facebook_redirect_uri,
                "code": code,
            },
        )
        token_resp.raise_for_status()
        tokens = token_resp.json()

        me_resp = await client.get(
            "https://graph.facebook.com/v19.0/me",
            params={
                "fields": "id,name,email,picture",
                "access_token": tokens["access_token"],
            },
        )
        me_resp.raise_for_status()
        me = me_resp.json()

    return {
        "provider_user_id": me["id"],
        "email": me.get("email"),
        "name": me.get("name"),
        "access_token": tokens.get("access_token"),
    }


async def validate_access_token(access_token: str) -> dict:
    """Validate a Facebook access token (for mobile clients)."""
    async with httpx.AsyncClient() as client:
        resp = await client.get(
            "https://graph.facebook.com/v19.0/me",
            params={
                "fields": "id,name,email",
                "access_token": access_token,
            },
        )
        resp.raise_for_status()
        me = resp.json()

    return {
        "provider_user_id": me["id"],
        "email": me.get("email"),
        "name": me.get("name"),
    }
