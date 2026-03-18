import uuid as _uuid

import asyncpg
from fastapi import APIRouter, Depends

from app.deps import get_current_user, get_db_conn
from app.schemas.envelope import ok

router = APIRouter(prefix="/api/v1/onboarding", tags=["onboarding"])


def _serialize_row(row: asyncpg.Record) -> dict:
    """Convert an asyncpg Record to a JSON-friendly dict."""
    d = dict(row)
    for k, v in d.items():
        if isinstance(v, _uuid.UUID):
            d[k] = str(v)
        elif hasattr(v, "isoformat"):
            d[k] = v.isoformat()
    return d


# ---------------------------------------------------------------------------
# GET /skills
# ---------------------------------------------------------------------------

@router.get("/skills")
async def get_skills(
    user: dict = Depends(get_current_user),
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    user_id = _uuid.UUID(user["id"])

    # User's selected skills
    core_rows = await conn.fetch(
        """
        SELECT us.id, us.skill_id, s.name AS skill_name, us.proficiency_level
        FROM user_skills us
        JOIN skills s ON s.id = us.skill_id
        WHERE us.user_id = $1
        ORDER BY s.name
        """,
        user_id,
    )
    core_skill_ids = {r["skill_id"] for r in core_rows}
    core_skills = [_serialize_row(r) for r in core_rows]

    # All other available skills
    all_rows = await conn.fetch("SELECT id, name FROM skills ORDER BY name")
    other_skills = [
        {"id": str(r["id"]), "name": r["name"]}
        for r in all_rows
        if r["id"] not in core_skill_ids
    ]

    return ok({"core_skills": core_skills, "other_skills": other_skills})


# ---------------------------------------------------------------------------
# GET /experience
# ---------------------------------------------------------------------------

@router.get("/experience")
async def get_experience(
    user: dict = Depends(get_current_user),
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    user_id = _uuid.UUID(user["id"])
    rows = await conn.fetch(
        """
        SELECT id, title, company, start_date, end_date, current, description
        FROM user_experience
        WHERE user_id = $1
        ORDER BY current DESC, start_date DESC NULLS LAST
        """,
        user_id,
    )
    return ok([_serialize_row(r) for r in rows])


# ---------------------------------------------------------------------------
# GET /education
# ---------------------------------------------------------------------------

@router.get("/education")
async def get_education(
    user: dict = Depends(get_current_user),
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    user_id = _uuid.UUID(user["id"])
    rows = await conn.fetch(
        """
        SELECT id, institution, degree, field_of_study, start_date, end_date, description
        FROM user_education
        WHERE user_id = $1
        ORDER BY start_date DESC NULLS LAST
        """,
        user_id,
    )
    return ok([_serialize_row(r) for r in rows])


# ---------------------------------------------------------------------------
# GET /certificates
# ---------------------------------------------------------------------------

@router.get("/certificates")
async def get_certificates(
    user: dict = Depends(get_current_user),
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    user_id = _uuid.UUID(user["id"])
    rows = await conn.fetch(
        """
        SELECT id, name, issuing_org, issue_date, expiry_date, credential_id, credential_url
        FROM user_certificates
        WHERE user_id = $1
        ORDER BY issue_date DESC NULLS LAST
        """,
        user_id,
    )
    return ok([_serialize_row(r) for r in rows])


# ---------------------------------------------------------------------------
# GET /languages
# ---------------------------------------------------------------------------

@router.get("/languages")
async def get_languages(
    user: dict = Depends(get_current_user),
    conn: asyncpg.Connection = Depends(get_db_conn),
):
    user_id = _uuid.UUID(user["id"])
    rows = await conn.fetch(
        """
        SELECT id, language, proficiency
        FROM user_languages
        WHERE user_id = $1
        ORDER BY language
        """,
        user_id,
    )
    return ok([_serialize_row(r) for r in rows])
