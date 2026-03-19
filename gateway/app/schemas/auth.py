"""Auth schemas: request/response models for authentication endpoints."""

from __future__ import annotations

from pydantic import BaseModel, EmailStr, Field


class RegisterRequest(BaseModel):
    email: EmailStr
    phone: str | None = None
    password: str = Field(..., min_length=8, max_length=128)
    role: str = Field(..., pattern=r"^(giver|doer|both)$")


class LoginRequest(BaseModel):
    email: EmailStr
    password: str


class TokenResponse(BaseModel):
    access_token: str
    refresh_token: str
    token_type: str = "bearer"
    expires_in: int


class RefreshRequest(BaseModel):
    refresh_token: str


class SendOTPRequest(BaseModel):
    channel: str = Field(..., pattern=r"^(sms|email)$")
    purpose: str = Field(default="verify_phone", pattern=r"^(login|verify_phone|verify_email|reset_password)$")


class VerifyOTPRequest(BaseModel):
    code: str = Field(..., min_length=6, max_length=6)
    purpose: str = Field(default="verify_phone", pattern=r"^(login|verify_phone|verify_email|reset_password)$")
