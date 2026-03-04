"""Config and FAQ endpoints."""

from __future__ import annotations

from fastapi import APIRouter

from app.deps import DBConn
from app.schemas.envelope import ok

router = APIRouter(prefix="/api/v1", tags=["config"])


@router.get("/categories")
async def list_categories(conn: DBConn):
    """List active task categories (public, no auth)."""
    rows = await conn.fetch(
        "SELECT id, name, icon_url, sort_order FROM categories WHERE active = TRUE ORDER BY sort_order",
    )
    items = [{"id": str(r["id"]), "name": r["name"], "icon_url": r["icon_url"]} for r in rows]
    return ok(data=items)


@router.get("/faqs")
async def list_faqs(conn: DBConn):
    """List active FAQs (public, no auth)."""
    rows = await conn.fetch(
        "SELECT id, question, answer, category FROM faqs WHERE active = TRUE ORDER BY sort_order",
    )
    items = [
        {"id": str(r["id"]), "question": r["question"], "answer": r["answer"], "category": r["category"]}
        for r in rows
    ]
    return ok(data=items)
