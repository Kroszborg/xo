import asyncpg
from fastapi import APIRouter, Depends

from app.deps import get_db_conn
from app.schemas.envelope import ok

router = APIRouter(prefix="/api/v1/categories", tags=["categories"])


@router.get("/")
async def list_categories(
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    """List all active task categories."""
    rows = await conn.fetch(
        """
        SELECT id, name, description, icon_url
        FROM task_categories
        WHERE active = TRUE
        ORDER BY name
        """
    )
    items = [
        {
            "id": str(r["id"]),
            "name": r["name"],
            "description": r["description"],
            "icon_url": r["icon_url"],
        }
        for r in rows
    ]
    return ok(items)
