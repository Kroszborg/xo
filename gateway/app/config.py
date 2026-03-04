"""Application settings loaded from environment variables."""

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Gateway configuration. All values can be overridden via env vars."""

    # Database
    database_url: str = "postgres://xo:xo@postgres:5432/xo?sslmode=disable"
    db_min_pool_size: int = 2
    db_max_pool_size: int = 10

    # xo service
    xo_service_url: str = "http://xo:8080"

    # JWT
    jwt_secret: str = "change-me-in-production"
    jwt_algorithm: str = "HS256"
    jwt_access_token_minutes: int = 30
    jwt_refresh_token_days: int = 30

    # Server
    listen_host: str = "0.0.0.0"
    listen_port: int = 8000

    # Uploads
    upload_dir: str = "/app/uploads"

    # CORS
    cors_origins: list[str] = ["*"]

    model_config = {"env_prefix": "", "case_sensitive": False}


settings = Settings()
