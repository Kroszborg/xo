"""Profile and onboarding schemas."""

from __future__ import annotations

from datetime import date

from pydantic import BaseModel, Field


# ─── Profile ─────────────────────────────────────────────────────────────────

class ProfileUpdate(BaseModel):
    display_name: str | None = None
    first_name: str | None = None
    last_name: str | None = None
    bio: str | None = None
    date_of_birth: date | None = None
    gender: str | None = Field(default=None, pattern=r"^(male|female|other|prefer_not_to_say)$")


class ProfileResponse(BaseModel):
    user_id: str
    display_name: str | None = None
    first_name: str | None = None
    last_name: str | None = None
    bio: str | None = None
    avatar_url: str | None = None
    date_of_birth: date | None = None
    gender: str | None = None
    onboarding_step: int = 0
    onboarding_done: bool = False


# ─── Onboarding sub-models ───────────────────────────────────────────────────

class OnboardingRoleData(BaseModel):
    """Step 1: role selection."""
    role: str = Field(..., pattern=r"^(giver|doer|both)$")


class SkillItem(BaseModel):
    skill_name: str


class OnboardingSkillsData(BaseModel):
    """Step 2: skills selection."""
    core_skills: list[SkillItem] = []
    other_skills: list[SkillItem] = []


class ExperienceItem(BaseModel):
    title: str
    company: str | None = None
    start_date: date | None = None
    end_date: date | None = None
    current: bool = False
    description: str | None = None


class OnboardingExperienceData(BaseModel):
    """Step 3: experience."""
    items: list[ExperienceItem] = []


class EducationItem(BaseModel):
    institution: str
    degree: str | None = None
    field_of_study: str | None = None
    start_date: date | None = None
    end_date: date | None = None
    description: str | None = None


class OnboardingEducationData(BaseModel):
    """Step 4: education."""
    items: list[EducationItem] = []


class CertificateItem(BaseModel):
    name: str
    issuing_org: str | None = None
    issue_date: date | None = None
    expiry_date: date | None = None
    credential_id: str | None = None
    credential_url: str | None = None


class OnboardingCertificatesData(BaseModel):
    """Step 5: certificates."""
    items: list[CertificateItem] = []


class LanguageItem(BaseModel):
    language: str
    proficiency: str = Field(default="conversational", pattern=r"^(basic|conversational|fluent|native)$")


class OnboardingLanguagesData(BaseModel):
    """Step 6: languages."""
    items: list[LanguageItem] = []


class OnboardingBioData(BaseModel):
    """Step 7: bio/about."""
    bio: str = Field(..., max_length=2000)
    display_name: str | None = None
