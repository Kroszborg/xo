"""Standard API response envelope."""

from __future__ import annotations

from typing import Any, Generic, TypeVar

from pydantic import BaseModel

T = TypeVar("T")


class Envelope(BaseModel, Generic[T]):
    """Uniform JSON envelope for all API responses."""

    success: bool = True
    data: T | None = None
    error: str | None = None
    message: str | None = None


class PaginatedData(BaseModel, Generic[T]):
    """Wrapper for paginated list responses."""

    items: list[T]
    total: int
    page: int
    limit: int
    pages: int


def ok(data: Any = None, message: str | None = None) -> dict:
    """Return a success envelope dict."""
    return {"success": True, "data": data, "error": None, "message": message}


def err(error: str, message: str | None = None) -> dict:
    """Return an error envelope dict."""
    return {"success": False, "data": None, "error": error, "message": message}


def paginated(items: list, total: int, page: int, limit: int) -> dict:
    """Return a paginated success envelope dict."""
    pages = (total + limit - 1) // limit if limit > 0 else 0
    return ok(data={"items": items, "total": total, "page": page, "limit": limit, "pages": pages})
