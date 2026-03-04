"""Profile and onboarding endpoints."""

from __future__ import annotations

from fastapi import APIRouter, HTTPException, UploadFile, File, status
import uuid
import os

from app.config import settings
from app.deps import CurrentUser, DBConn
from app.schemas.envelope import ok
from app.schemas.profile import (
    OnboardingBioData,
    OnboardingCertificatesData,
    OnboardingEducationData,
    OnboardingExperienceData,
    OnboardingLanguagesData,
    OnboardingRoleData,
    OnboardingSkillsData,
    ProfileResponse,
    ProfileUpdate,
)

router = APIRouter(prefix="/api/v1", tags=["profile"])

# ─── Role mapping ────────────────────────────────────────────────────────────
_ROLE_MAP = {"giver": "task_giver", "doer": "task_doer", "both": "both"}


# ─── Profile CRUD ─────────────────────────────────────────────────────────────

@router.get("/profile")
async def get_profile(user: CurrentUser, conn: DBConn):
    """Get the current user's profile."""
    row = await conn.fetchrow(
        """
        SELECT gp.*, u.email, u.phone, u.role
        FROM gateway_user_profile gp
        JOIN users u ON u.id = gp.user_id
        WHERE gp.user_id = $1
        """,
        user["id"],
    )
    if row is None:
        # Auto-create if missing
        await conn.execute(
            "INSERT INTO gateway_user_profile (user_id) VALUES ($1) ON CONFLICT DO NOTHING",
            user["id"],
        )
        row = await conn.fetchrow(
            """
            SELECT gp.*, u.email, u.phone, u.role
            FROM gateway_user_profile gp
            JOIN users u ON u.id = gp.user_id
            WHERE gp.user_id = $1
            """,
            user["id"],
        )

    return ok(data={
        "user_id": str(row["user_id"]),
        "email": row["email"],
        "phone": row["phone"],
        "role": row["role"],
        "display_name": row["display_name"],
        "first_name": row["first_name"],
        "last_name": row["last_name"],
        "bio": row["bio"],
        "avatar_url": row["avatar_url"],
        "date_of_birth": str(row["date_of_birth"]) if row["date_of_birth"] else None,
        "gender": row["gender"],
        "onboarding_step": row["onboarding_step"],
        "onboarding_done": row["onboarding_done"],
    })


@router.patch("/profile")
async def update_profile(body: ProfileUpdate, user: CurrentUser, conn: DBConn):
    """Update profile fields."""
    updates = body.model_dump(exclude_unset=True)
    if not updates:
        return ok(message="No changes")

    set_clauses = []
    params = []
    idx = 1
    for key, val in updates.items():
        set_clauses.append(f"{key} = ${idx}")
        params.append(val)
        idx += 1
    params.append(user["id"])

    await conn.execute(
        f"UPDATE gateway_user_profile SET {', '.join(set_clauses)} WHERE user_id = ${idx}",
        *params,
    )
    return ok(message="Profile updated")


@router.post("/profile/avatar")
async def upload_avatar(user: CurrentUser, conn: DBConn, file: UploadFile = File(...)):
    """Upload a profile avatar."""
    ext = os.path.splitext(file.filename or "avatar.jpg")[1]
    file_id = str(uuid.uuid4())
    filename = f"{file_id}{ext}"
    upload_path = os.path.join(settings.upload_dir, "avatars")
    os.makedirs(upload_path, exist_ok=True)
    file_path = os.path.join(upload_path, filename)

    content = await file.read()
    with open(file_path, "wb") as f:
        f.write(content)

    avatar_url = f"/uploads/avatars/{filename}"
    await conn.execute(
        "UPDATE gateway_user_profile SET avatar_url = $1 WHERE user_id = $2",
        avatar_url,
        user["id"],
    )

    # Record in file_uploads
    await conn.execute(
        """
        INSERT INTO file_uploads (user_id, file_type, file_name, file_path, mime_type, size_bytes)
        VALUES ($1, 'avatar', $2, $3, $4, $5)
        """,
        user["id"],
        file.filename,
        file_path,
        file.content_type,
        len(content),
    )

    return ok(data={"avatar_url": avatar_url}, message="Avatar uploaded")


# ─── Onboarding Steps ─────────────────────────────────────────────────────────

@router.get("/onboarding/status")
async def onboarding_status(user: CurrentUser, conn: DBConn):
    """Get current onboarding progress."""
    row = await conn.fetchrow(
        "SELECT onboarding_step, onboarding_done FROM gateway_user_profile WHERE user_id = $1",
        user["id"],
    )
    if row is None:
        return ok(data={"current_step": 0, "completed": False})
    return ok(data={"current_step": row["onboarding_step"], "completed": row["onboarding_done"]})


async def _advance_step(conn: DBConn, user_id, target_step: int):
    """Advance onboarding step if user is at target_step - 1."""
    current = await conn.fetchval(
        "SELECT onboarding_step FROM gateway_user_profile WHERE user_id = $1",
        user_id,
    )
    if current is None:
        current = 0
    # Allow re-submission of current step or advancing to next
    if target_step <= current + 1:
        new_step = max(current, target_step)
        done = new_step >= 7
        await conn.execute(
            "UPDATE gateway_user_profile SET onboarding_step = $1, onboarding_done = $2 WHERE user_id = $3",
            new_step,
            done,
            user_id,
        )


@router.post("/onboarding/step/1")
async def onboarding_step_1(body: OnboardingRoleData, user: CurrentUser, conn: DBConn):
    """Step 1: Role selection."""
    db_role = _ROLE_MAP.get(body.role)
    if db_role is None:
        raise HTTPException(status_code=400, detail="Invalid role")
    await conn.execute("UPDATE users SET role = $1 WHERE id = $2", db_role, user["id"])
    await _advance_step(conn, user["id"], 1)
    return ok(message="Role set")


@router.post("/onboarding/step/2")
async def onboarding_step_2(body: OnboardingSkillsData, user: CurrentUser, conn: DBConn):
    """Step 2: Skills selection."""
    # Clear existing
    await conn.execute("DELETE FROM user_core_skills WHERE user_id = $1", user["id"])
    await conn.execute("DELETE FROM user_other_skills WHERE user_id = $1", user["id"])

    for s in body.core_skills:
        await conn.execute(
            "INSERT INTO user_core_skills (user_id, skill_name) VALUES ($1, $2) ON CONFLICT DO NOTHING",
            user["id"],
            s.skill_name,
        )
    for s in body.other_skills:
        await conn.execute(
            "INSERT INTO user_other_skills (user_id, skill_name) VALUES ($1, $2) ON CONFLICT DO NOTHING",
            user["id"],
            s.skill_name,
        )
    await _advance_step(conn, user["id"], 2)
    return ok(message="Skills saved")


@router.post("/onboarding/step/3")
async def onboarding_step_3(body: OnboardingExperienceData, user: CurrentUser, conn: DBConn):
    """Step 3: Experience."""
    await conn.execute("DELETE FROM user_experience WHERE user_id = $1", user["id"])
    for item in body.items:
        await conn.execute(
            """
            INSERT INTO user_experience (user_id, title, company, start_date, end_date, current, description)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            """,
            user["id"],
            item.title,
            item.company,
            item.start_date,
            item.end_date,
            item.current,
            item.description,
        )
    await _advance_step(conn, user["id"], 3)
    return ok(message="Experience saved")


@router.post("/onboarding/step/4")
async def onboarding_step_4(body: OnboardingEducationData, user: CurrentUser, conn: DBConn):
    """Step 4: Education."""
    await conn.execute("DELETE FROM user_education WHERE user_id = $1", user["id"])
    for item in body.items:
        await conn.execute(
            """
            INSERT INTO user_education (user_id, institution, degree, field_of_study, start_date, end_date, description)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            """,
            user["id"],
            item.institution,
            item.degree,
            item.field_of_study,
            item.start_date,
            item.end_date,
            item.description,
        )
    await _advance_step(conn, user["id"], 4)
    return ok(message="Education saved")


@router.post("/onboarding/step/5")
async def onboarding_step_5(body: OnboardingCertificatesData, user: CurrentUser, conn: DBConn):
    """Step 5: Certificates."""
    await conn.execute("DELETE FROM user_certificates WHERE user_id = $1", user["id"])
    for item in body.items:
        await conn.execute(
            """
            INSERT INTO user_certificates (user_id, name, issuing_org, issue_date, expiry_date, credential_id, credential_url)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            """,
            user["id"],
            item.name,
            item.issuing_org,
            item.issue_date,
            item.expiry_date,
            item.credential_id,
            item.credential_url,
        )
    await _advance_step(conn, user["id"], 5)
    return ok(message="Certificates saved")


@router.post("/onboarding/step/6")
async def onboarding_step_6(body: OnboardingLanguagesData, user: CurrentUser, conn: DBConn):
    """Step 6: Languages."""
    await conn.execute("DELETE FROM user_languages WHERE user_id = $1", user["id"])
    for item in body.items:
        await conn.execute(
            """
            INSERT INTO user_languages (user_id, language, proficiency)
            VALUES ($1, $2, $3) ON CONFLICT (user_id, language) DO UPDATE SET proficiency = $3
            """,
            user["id"],
            item.language,
            item.proficiency,
        )
    await _advance_step(conn, user["id"], 6)
    return ok(message="Languages saved")


@router.post("/onboarding/step/7")
async def onboarding_step_7(body: OnboardingBioData, user: CurrentUser, conn: DBConn):
    """Step 7: Bio/about (final step)."""
    updates = {"bio": body.bio}
    if body.display_name:
        updates["display_name"] = body.display_name

    set_clauses = []
    params = []
    idx = 1
    for key, val in updates.items():
        set_clauses.append(f"{key} = ${idx}")
        params.append(val)
        idx += 1
    params.append(user["id"])

    await conn.execute(
        f"UPDATE gateway_user_profile SET {', '.join(set_clauses)} WHERE user_id = ${idx}",
        *params,
    )
    await _advance_step(conn, user["id"], 7)
    return ok(message="Onboarding complete")
