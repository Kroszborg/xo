from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    database_url: str = "postgresql://xo:xo@localhost:5432/xo"
    xo_service_url: str = "http://localhost:8080"
    jwt_secret: str = "dev-secret-change-in-production"
    jwt_algorithm: str = "HS256"
    jwt_access_token_expire_minutes: int = 30
    jwt_refresh_token_expire_days: int = 30
    google_client_id: str = ""
    google_client_secret: str = ""
    google_redirect_uri: str = ""
    facebook_client_id: str = ""
    facebook_client_secret: str = ""
    facebook_redirect_uri: str = ""
    cors_origins: str = "*"
    port: int = 8000

    model_config = {"env_prefix": "", "case_sensitive": False}


settings = Settings()
