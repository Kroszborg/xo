"""Tests for the response envelope helpers."""

from app.schemas.envelope import err, ok, paginated


def test_ok_envelope():
    result = ok(data={"key": "value"}, message="success")
    assert result["success"] is True
    assert result["data"]["key"] == "value"
    assert result["error"] is None
    assert result["message"] == "success"


def test_err_envelope():
    result = err(error="bad_request", message="Something went wrong")
    assert result["success"] is False
    assert result["data"] is None
    assert result["error"] == "bad_request"
    assert result["message"] == "Something went wrong"


def test_paginated_envelope():
    items = [{"id": 1}, {"id": 2}]
    result = paginated(items=items, total=10, page=1, limit=2)
    assert result["success"] is True
    assert result["data"]["items"] == items
    assert result["data"]["total"] == 10
    assert result["data"]["pages"] == 5


def test_paginated_zero_limit():
    result = paginated(items=[], total=0, page=1, limit=0)
    assert result["data"]["pages"] == 0
