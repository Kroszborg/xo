from typing import Any, Optional


def ok(data: Any) -> dict:
    return {"data": data}


def err(code: str, message: str, details: list | None = None) -> dict:
    error = {"code": code, "message": message}
    if details:
        error["details"] = details
    return {"error": error}


def cursor_page(items: list, next_cursor: str | None, has_more: bool) -> dict:
    return {
        "data": items,
        "cursor": {"next": next_cursor, "has_more": has_more},
    }
