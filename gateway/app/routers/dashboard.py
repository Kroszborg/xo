"""Dashboard endpoints: aggregate stats for the current user."""

from __future__ import annotations

from fastapi import APIRouter

from app.deps import CurrentUser, DBConn
from app.schemas.envelope import ok

router = APIRouter(prefix="/api/v1/dashboard", tags=["dashboard"])


@router.get("")
async def get_dashboard(user: CurrentUser, conn: DBConn):
    """Get dashboard overview for the current user."""
    user_id = user["id"]

    # Onboarding progress
    profile = await conn.fetchrow(
        "SELECT onboarding_step, onboarding_done FROM gateway_user_profile WHERE user_id = $1",
        user_id,
    )

    # Task stats from xo tables
    task_stats = await conn.fetchrow(
        """
        SELECT
            COUNT(*) FILTER (WHERE state = 'active') AS active_tasks,
            COUNT(*) FILTER (WHERE state = 'completed') AS completed_tasks,
            COUNT(*) FILTER (WHERE state = 'accepted') AS accepted_tasks,
            COUNT(*) FILTER (WHERE state IN ('draft', 'priority', 'active', 'accepted')) AS open_tasks,
            COUNT(*) AS total_tasks
        FROM tasks
        WHERE task_giver_id = $1
        """,
        user_id,
    )

    # Tasks accepted as doer
    doer_stats = await conn.fetchrow(
        """
        SELECT
            COUNT(*) AS tasks_accepted,
            COALESCE(SUM(accepted_budget), 0) AS total_earned
        FROM task_acceptances
        WHERE user_id = $1
        """,
        user_id,
    )

    # Behavior metrics
    behavior = await conn.fetchrow(
        """
        SELECT acceptance_rate, completion_rate, reliability_score, total_tasks_completed
        FROM user_behavior_metrics
        WHERE user_id = $1
        """,
        user_id,
    )

    # Verification status
    verification = await conn.fetchrow(
        "SELECT email_verified, phone_verified, id_verified FROM user_verification WHERE user_id = $1",
        user_id,
    )

    return ok(data={
        "onboarding": {
            "step": profile["onboarding_step"] if profile else 0,
            "done": profile["onboarding_done"] if profile else False,
        },
        "tasks_as_giver": {
            "active": task_stats["active_tasks"] if task_stats else 0,
            "completed": task_stats["completed_tasks"] if task_stats else 0,
            "accepted": task_stats["accepted_tasks"] if task_stats else 0,
            "open": task_stats["open_tasks"] if task_stats else 0,
            "total": task_stats["total_tasks"] if task_stats else 0,
        },
        "tasks_as_doer": {
            "accepted": doer_stats["tasks_accepted"] if doer_stats else 0,
            "total_earned": float(doer_stats["total_earned"]) if doer_stats else 0.0,
        },
        "behavior": {
            "acceptance_rate": float(behavior["acceptance_rate"]) if behavior else 0.0,
            "completion_rate": float(behavior["completion_rate"]) if behavior else 0.0,
            "reliability_score": float(behavior["reliability_score"]) if behavior else 0.0,
            "total_tasks_completed": behavior["total_tasks_completed"] if behavior else 0,
        },
        "verification": {
            "email_verified": verification["email_verified"] if verification else False,
            "phone_verified": verification["phone_verified"] if verification else False,
            "id_verified": verification["id_verified"] if verification else False,
        },
    })
