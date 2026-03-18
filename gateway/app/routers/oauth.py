import asyncpg
from fastapi import APIRouter, Depends, HTTPException, Query, status
from fastapi.responses import RedirectResponse

from app.deps import get_db_conn
from app.oauth import common, google, facebook
from app.schemas.envelope import err, ok

router = APIRouter(prefix="/api/v1/auth", tags=["oauth"])


# --- Google ---

@router.get("/google")
async def google_redirect():
    url = google.get_google_redirect_url()
    return RedirectResponse(url)


@router.get("/google/callback")
async def google_callback(
    code: str = Query(...),
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    try:
        info = await google.exchange_code(code)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=err("OAUTH_ERROR", f"Google OAuth failed: {e}"),
        )

    user = await common.find_or_create_user(
        conn,
        email=info["email"],
        provider="google",
        provider_user_id=info["provider_user_id"],
        provider_email=info["email"],
        full_name=info.get("name"),
    )
    tokens = await common.issue_tokens(conn, user["user_id"], user["role"])
    tokens["is_new"] = user["is_new"]
    return ok(tokens)


@router.post("/google/token")
async def google_mobile_token(
    body: dict,
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    id_token = body.get("id_token")
    if not id_token:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=err("VALIDATION_ERROR", "id_token is required"),
        )
    try:
        info = await google.validate_id_token(id_token)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=err("OAUTH_ERROR", f"Invalid Google ID token: {e}"),
        )

    user = await common.find_or_create_user(
        conn,
        email=info["email"],
        provider="google",
        provider_user_id=info["provider_user_id"],
        full_name=info.get("name"),
    )
    tokens = await common.issue_tokens(conn, user["user_id"], user["role"])
    tokens["is_new"] = user["is_new"]
    return ok(tokens)


# --- Facebook ---

@router.get("/facebook")
async def facebook_redirect():
    url = facebook.get_facebook_redirect_url()
    return RedirectResponse(url)


@router.get("/facebook/callback")
async def facebook_callback(
    code: str = Query(...),
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    try:
        info = await facebook.exchange_code(code)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=err("OAUTH_ERROR", f"Facebook OAuth failed: {e}"),
        )

    if not info.get("email"):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=err("OAUTH_ERROR", "Facebook account must have an email"),
        )

    user = await common.find_or_create_user(
        conn,
        email=info["email"],
        provider="facebook",
        provider_user_id=info["provider_user_id"],
        provider_email=info["email"],
        full_name=info.get("name"),
    )
    tokens = await common.issue_tokens(conn, user["user_id"], user["role"])
    tokens["is_new"] = user["is_new"]
    return ok(tokens)


@router.post("/facebook/token")
async def facebook_mobile_token(
    body: dict,
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    access_token = body.get("access_token")
    if not access_token:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=err("VALIDATION_ERROR", "access_token is required"),
        )
    try:
        info = await facebook.validate_access_token(access_token)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=err("OAUTH_ERROR", f"Invalid Facebook token: {e}"),
        )

    if not info.get("email"):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=err("OAUTH_ERROR", "Facebook account must have an email"),
        )

    user = await common.find_or_create_user(
        conn,
        email=info["email"],
        provider="facebook",
        provider_user_id=info["provider_user_id"],
        full_name=info.get("name"),
    )
    tokens = await common.issue_tokens(conn, user["user_id"], user["role"])
    tokens["is_new"] = user["is_new"]
    return ok(tokens)
