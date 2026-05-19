from functools import lru_cache

from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    model_config = SettingsConfigDict(env_prefix="MEUNAS_", env_file=".env", extra="ignore")

    env: str = "development"
    db_url: str = "sqlite+aiosqlite:///../data/meu-nas.db"
    secret_key: str = Field(
        min_length=32,
        default="troque-esta-chave-em-producao-com-32-caracteres",
    )
    access_token_minutes: int = 15
    refresh_token_days: int = 7
    admin_username: str = "admin"
    admin_password: str = Field(min_length=8, default="admin12345")
    cors_origins: list[str] = ["http://localhost:5173", "http://127.0.0.1:5173"]

    @field_validator("cors_origins", mode="before")
    @classmethod
    def parse_cors_origins(cls, value: str | list[str]) -> list[str]:
        if isinstance(value, str):
            return [origin.strip() for origin in value.split(",") if origin.strip()]
        return value


@lru_cache
def get_settings() -> Settings:
    """Return cached application settings."""
    return Settings()
