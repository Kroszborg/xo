from typing import Optional

from pydantic import BaseModel


class ProfileUpdate(BaseModel):
    full_name: Optional[str] = None
    bio: Optional[str] = None
    phone: Optional[str] = None
    latitude: Optional[float] = None
    longitude: Optional[float] = None
    city: Optional[str] = None
    state: Optional[str] = None
    country: Optional[str] = None
    max_distance_km: Optional[int] = None
    preferred_budget_min: Optional[float] = None
    preferred_budget_max: Optional[float] = None
    is_online: Optional[bool] = None


class OnboardingStepRequest(BaseModel):
    step: int
    data: dict


class ProfileResponse(BaseModel):
    user_id: str
    email: str
    role: str
    full_name: Optional[str] = None
    bio: Optional[str] = None
    phone: Optional[str] = None
    latitude: Optional[float] = None
    longitude: Optional[float] = None
    city: Optional[str] = None
    state: Optional[str] = None
    country: Optional[str] = None
    max_distance_km: Optional[int] = None
    preferred_budget_min: Optional[float] = None
    preferred_budget_max: Optional[float] = None
    is_online: Optional[bool] = None
    onboarding_step: Optional[int] = None
    onboarding_completed: Optional[bool] = None
