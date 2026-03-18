from datetime import datetime, timezone
from uuid import UUID

import asyncpg
from fastapi import APIRouter, Depends, HTTPException, Query, status

from app.deps import get_current_user, get_db_conn
from app.schemas.envelope import cursor_page, err, ok

router = APIRouter(prefix="/api/v1/notifications", tags=["notifications"])


def _serialise(row: asyncpg.Record) -> dict:
    """Convert a notification row to a JSON-friendly dict."""
    d = dict(row)
    d["id"] = str(d["id"])
    d["user_id"] = str(d["user_id"])
    if d.get("created_at"):
        d["created_at"] = d["created_at"].isoformat()
    if d.get("read_at"):
        d["read_at"] = d["read_at"].isoformat()
    return d


# ---------------------------------------------------------------------------
# GET / -- cursor-paginated notification list
# ---------------------------------------------------------------------------

@router.get("/")
async def list_notifications(
    user: dict = Depends(get_current_user),
    conn: asyncpg.Connection = Depends(get_db_conn),
    cursor: str | None = Query(default=None, description="Opaque cursor (notification UUID)"),
    limit: int = Query(default=20, ge=1, le=100),
):
    """Return in-app notifications for the current user, newest first."""

    if cursor:
        # Fetch created_at of the cursor notification for keyset pagination
        cursor_created = await conn.fetchval(
            "SELECT created_at FROM inapp_notifications WHERE id = $1 AND user_id = $2",
            UUID(cursor),
            UUID(user["id"]),
        )
        if cursor_created is None:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=err("INVALID_CURSOR", "Cursor does not reference a valid notification"),
            )

        rows = await conn.fetch(
            """
            SELECT id, user_id, type, title, body, payload, read_at, created_at
            FROM inapp_notifications
            WHERE user_id = $1
              AND (created_at, id) < ($2, $3)
            ORDER BY created_at DESC, id DESC
            LIMIT $4
            """,
            UUID(user["id"]),
            cursor_created,
            UUID(cursor),
            limit + 1,  # fetch one extra to detect has_more
        )
    else:
        rows = await conn.fetch(
            """
            SELECT id, user_id, type, title, body, payload, read_at, created_at
            FROM inapp_notifications
            WHERE user_id = $1
            ORDER BY created_at DESC, id DESC
            LIMIT $2
            """,
            UUID(user["id"]),
            limit + 1,
        )

    has_more = len(rows) > limit
    items = [_serialise(r) for r in rows[:limit]]
    next_cursor = items[-1]["id"] if has_more and items else None

    return cursor_page(items, next_cursor, has_more)


# ---------------------------------------------------------------------------
# PATCH /{id}/read
# ---------------------------------------------------------------------------

@router.patch("/{notification_id}/read")
async def mark_read(
    notification_id: str,
    user: dict = Depends(get_current_user),
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    result = await conn.fetchrow(
        """
        UPDATE inapp_notifications
        SET read_at = $1
        WHERE id = $2 AND user_id = $3 AND read_at IS NULL
        RETURNING id
        """,
        datetime.now(timezone.utc),
        UUID(notification_id),
        UUID(user["id"]),
    )

    if result is None:
        # Either does not exist, belongs to another user, or was already read
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=err(
                "NOT_FOUND",
                "Notification not found or already read",
            ),
        )

    return ok({"id": notification_id, "read": True})
