import asyncpg
from fastapi import APIRouter, Depends, HTTPException, status

from app.deps import get_current_user, get_db_conn
from app.schemas.envelope import err, ok
from app.schemas.profile import ProfileUpdate

router = APIRouter(prefix="/api/v1/profile", tags=["profile"])


# ---------------------------------------------------------------------------
# GET / -- current user's profile
# ---------------------------------------------------------------------------

@router.get("/")
async def get_profile(
    user: dict = Depends(get_current_user),
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    row = await conn.fetchrow(
        """
        SELECT
            u.id        AS user_id,
            u.email,
            u.role,
            p.full_name,
            p.bio,
            p.phone,
            p.latitude,
            p.longitude,
            p.city,
            p.state,
            p.country,
            p.max_distance_km,
            p.preferred_budget_min,
            p.preferred_budget_max,
            p.is_online,
            p.onboarding_step,
            p.onboarding_completed
        FROM users u
        LEFT JOIN user_profiles p ON p.user_id = u.id
        WHERE u.id = $1
        """,
        user["id"],
    )

    if row is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=err("PROFILE_NOT_FOUND", "Profile not found"),
        )

    data = dict(row)
    # Convert UUID and Decimal types to JSON-friendly values
    data["user_id"] = str(data["user_id"])
    for key in ("latitude", "longitude", "preferred_budget_min", "preferred_budget_max"):
        if data.get(key) is not None:
            data[key] = float(data[key])

    return ok(data)


# ---------------------------------------------------------------------------
# PATCH / -- update profile fields
# ---------------------------------------------------------------------------

@router.patch("/")
async def update_profile(
    body: ProfileUpdate,
    user: dict = Depends(get_current_user),
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    updates = body.model_dump(exclude_unset=True)
    if not updates:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=err("NO_FIELDS", "No fields to update"),
        )

    # Build dynamic SET clause
    set_parts: list[str] = []
    values: list = []
    idx = 1
    for field, value in updates.items():
        idx += 1  # $1 is reserved for user_id
        set_parts.append(f"{field} = ${idx}")
        values.append(value)

    query = f"""
        UPDATE user_profiles
        SET {', '.join(set_parts)}
        WHERE user_id = $1
        RETURNING *
    """

    row = await conn.fetchrow(query, user["id"], *values)
    if row is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=err("PROFILE_NOT_FOUND", "Profile not found"),
        )

    result = dict(row)
    result["user_id"] = str(result.get("user_id", ""))
    result["id"] = str(result.get("id", ""))
    for key in ("latitude", "longitude", "preferred_budget_min", "preferred_budget_max"):
        if result.get(key) is not None:
            result[key] = float(result[key])

    return ok(result)
