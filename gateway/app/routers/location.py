"""Location endpoints."""

from __future__ import annotations

from pydantic import BaseModel, Field
from fastapi import APIRouter

from app.deps import CurrentUser, DBConn
from app.schemas.envelope import ok

router = APIRouter(prefix="/api/v1", tags=["location"])


class LocationUpdate(BaseModel):
    lat: float = Field(..., ge=-90, le=90)
    lng: float = Field(..., ge=-180, le=180)
    accuracy_m: float | None = Field(default=None, ge=0)


class AddressCreate(BaseModel):
    label: str | None = None
    line1: str
    line2: str | None = None
    city: str
    state: str | None = None
    postal_code: str | None = None
    country: str = "US"
    is_default: bool = False


@router.put("/location")
async def update_location(body: LocationUpdate, user: CurrentUser, conn: DBConn):
    """Update the user's current/live location."""
    await conn.execute(
        """
        INSERT INTO user_location (user_id, lat, lng, accuracy_m)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (user_id)
        DO UPDATE SET lat = $2, lng = $3, accuracy_m = $4
        """,
        user["id"],
        body.lat,
        body.lng,
        body.accuracy_m,
    )

    # Also update xo's user_profiles fixed_lat/lng
    await conn.execute(
        "UPDATE user_profiles SET fixed_lat = $1, fixed_lng = $2 WHERE user_id = $3",
        body.lat,
        body.lng,
        user["id"],
    )

    return ok(message="Location updated")


@router.get("/location")
async def get_location(user: CurrentUser, conn: DBConn):
    """Get the user's last-known location."""
    row = await conn.fetchrow(
        "SELECT lat, lng, accuracy_m, updated_at FROM user_location WHERE user_id = $1",
        user["id"],
    )
    if row is None:
        return ok(data=None, message="No location recorded")
    return ok(data={
        "lat": float(row["lat"]),
        "lng": float(row["lng"]),
        "accuracy_m": float(row["accuracy_m"]) if row["accuracy_m"] else None,
        "updated_at": row["updated_at"].isoformat(),
    })


@router.get("/addresses")
async def list_addresses(user: CurrentUser, conn: DBConn):
    """List user's saved addresses."""
    rows = await conn.fetch(
        "SELECT * FROM user_addresses WHERE user_id = $1 ORDER BY is_default DESC, created_at",
        user["id"],
    )
    items = []
    for r in rows:
        d = dict(r)
        d["id"] = str(d["id"])
        d["user_id"] = str(d["user_id"])
        d["created_at"] = d["created_at"].isoformat()
        d["updated_at"] = d["updated_at"].isoformat()
        items.append(d)
    return ok(data=items)


@router.post("/addresses", status_code=201)
async def create_address(body: AddressCreate, user: CurrentUser, conn: DBConn):
    """Create a new saved address."""
    import uuid as _uuid

    if body.is_default:
        await conn.execute(
            "UPDATE user_addresses SET is_default = FALSE WHERE user_id = $1",
            user["id"],
        )

    row = await conn.fetchrow(
        """
        INSERT INTO user_addresses (user_id, label, line1, line2, city, state, postal_code, country, is_default)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id
        """,
        user["id"],
        body.label,
        body.line1,
        body.line2,
        body.city,
        body.state,
        body.postal_code,
        body.country,
        body.is_default,
    )
    return ok(data={"id": str(row["id"])}, message="Address created")


@router.delete("/addresses/{address_id}")
async def delete_address(address_id: str, user: CurrentUser, conn: DBConn):
    """Delete a saved address."""
    import uuid as _uuid
    from fastapi import HTTPException

    result = await conn.execute(
        "DELETE FROM user_addresses WHERE id = $1 AND user_id = $2",
        _uuid.UUID(address_id),
        user["id"],
    )
    if result == "DELETE 0":
        raise HTTPException(status_code=404, detail="Address not found")
    return ok(message="Address deleted")
