"""Verification and payment method endpoints."""

from __future__ import annotations

from fastapi import APIRouter, HTTPException, UploadFile, File
from pydantic import BaseModel, Field
import os
import uuid

from app.config import settings
from app.deps import CurrentUser, DBConn
from app.schemas.envelope import ok

router = APIRouter(prefix="/api/v1", tags=["verification", "payments"])


# ─── Schemas ──────────────────────────────────────────────────────────────────

class PaymentMethodCreate(BaseModel):
    method_type: str = Field(..., pattern=r"^(bank_account|mobile_money|paypal|stripe)$")
    provider: str | None = None
    account_ref: str


class PaymentMethodUpdate(BaseModel):
    is_default: bool | None = None
    provider: str | None = None


# ─── Verification ─────────────────────────────────────────────────────────────

@router.get("/verification/status")
async def get_verification_status(user: CurrentUser, conn: DBConn):
    """Get verification status for the current user."""
    row = await conn.fetchrow(
        "SELECT email_verified, phone_verified, id_verified, verified_at FROM user_verification WHERE user_id = $1",
        user["id"],
    )
    if row is None:
        return ok(data={"email_verified": False, "phone_verified": False, "id_verified": False})
    return ok(data=dict(row))


@router.post("/verification/id-document")
async def upload_id_document(user: CurrentUser, conn: DBConn, file: UploadFile = File(...)):
    """Upload an ID document for verification."""
    ext = os.path.splitext(file.filename or "document.jpg")[1]
    file_id = str(uuid.uuid4())
    filename = f"{file_id}{ext}"
    upload_path = os.path.join(settings.upload_dir, "id_documents")
    os.makedirs(upload_path, exist_ok=True)
    file_path = os.path.join(upload_path, filename)

    content = await file.read()
    with open(file_path, "wb") as f:
        f.write(content)

    doc_url = f"/uploads/id_documents/{filename}"
    await conn.execute(
        """
        INSERT INTO user_verification (user_id, id_document_url)
        VALUES ($1, $2)
        ON CONFLICT (user_id)
        DO UPDATE SET id_document_url = $2
        """,
        user["id"],
        doc_url,
    )

    await conn.execute(
        """
        INSERT INTO file_uploads (user_id, file_type, file_name, file_path, mime_type, size_bytes)
        VALUES ($1, 'id_document', $2, $3, $4, $5)
        """,
        user["id"],
        file.filename,
        file_path,
        file.content_type,
        len(content),
    )

    return ok(data={"document_url": doc_url}, message="ID document uploaded")


# ─── Payment Methods ──────────────────────────────────────────────────────────

@router.get("/payment-methods")
async def list_payment_methods(user: CurrentUser, conn: DBConn):
    """List user's payment methods."""
    rows = await conn.fetch(
        """
        SELECT id, method_type, provider, account_ref, is_default, created_at
        FROM user_payment_methods
        WHERE user_id = $1
        ORDER BY created_at
        """,
        user["id"],
    )
    items = [dict(r) for r in rows]
    for item in items:
        item["id"] = str(item["id"])
        item["created_at"] = item["created_at"].isoformat()
    return ok(data=items)


@router.post("/payment-methods", status_code=201)
async def create_payment_method(body: PaymentMethodCreate, user: CurrentUser, conn: DBConn):
    """Add a new payment method."""
    row = await conn.fetchrow(
        """
        INSERT INTO user_payment_methods (user_id, method_type, provider, account_ref)
        VALUES ($1, $2, $3, $4)
        RETURNING id
        """,
        user["id"],
        body.method_type,
        body.provider,
        body.account_ref,
    )
    return ok(data={"id": str(row["id"])}, message="Payment method added")


@router.patch("/payment-methods/{method_id}")
async def update_payment_method(method_id: str, body: PaymentMethodUpdate, user: CurrentUser, conn: DBConn):
    """Update a payment method."""
    existing = await conn.fetchval(
        "SELECT id FROM user_payment_methods WHERE id = $1 AND user_id = $2",
        uuid.UUID(method_id),
        user["id"],
    )
    if existing is None:
        raise HTTPException(status_code=404, detail="Payment method not found")

    updates = body.model_dump(exclude_unset=True)
    if not updates:
        return ok(message="No changes")

    # If setting as default, unset others first
    if updates.get("is_default"):
        await conn.execute(
            "UPDATE user_payment_methods SET is_default = FALSE WHERE user_id = $1",
            user["id"],
        )

    set_clauses = []
    params = []
    idx = 1
    for key, val in updates.items():
        set_clauses.append(f"{key} = ${idx}")
        params.append(val)
        idx += 1
    params.append(uuid.UUID(method_id))

    await conn.execute(
        f"UPDATE user_payment_methods SET {', '.join(set_clauses)} WHERE id = ${idx}",
        *params,
    )
    return ok(message="Payment method updated")


@router.delete("/payment-methods/{method_id}")
async def delete_payment_method(method_id: str, user: CurrentUser, conn: DBConn):
    """Delete a payment method."""
    result = await conn.execute(
        "DELETE FROM user_payment_methods WHERE id = $1 AND user_id = $2",
        uuid.UUID(method_id),
        user["id"],
    )
    if result == "DELETE 0":
        raise HTTPException(status_code=404, detail="Payment method not found")
    return ok(message="Payment method deleted")
