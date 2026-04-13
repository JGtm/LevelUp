"""Configuration centralisée de l'API FastAPI LevelUp.

Lit les paramètres depuis les variables d'environnement et les fichiers
de configuration existants du repo (app_settings.json, db_profiles.json).
"""

from __future__ import annotations

import os
from functools import lru_cache
from pathlib import Path

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


def _repo_root() -> Path:
    """Retourne la racine du repo (parent de apps/)."""
    return Path(__file__).resolve().parents[4]


class ApiSettings(BaseSettings):
    """Paramètres de l'API — chargés depuis l'environnement ou .env.local."""

    model_config = SettingsConfigDict(
        env_file=str(_repo_root() / ".env.local"),
        env_file_encoding="utf-8",
        extra="ignore",
    )

    # --- API ------------------------------------------------------------
    api_host: str = Field(default="127.0.0.1", alias="LEVELUP_API_HOST")
    api_port: int = Field(default=8000, alias="LEVELUP_API_PORT")
    api_reload: bool = Field(default=False, alias="LEVELUP_API_RELOAD")

    # --- Sécurité / Session ---------------------------------------------
    # Clé de signature des cookies httpOnly (à remplacer en prod !)
    session_secret_key: str = Field(
        default="CHANGE_ME_IN_PRODUCTION",
        alias="LEVELUP_SESSION_SECRET",
    )
    session_ttl_seconds: int = Field(default=7 * 24 * 3600, alias="LEVELUP_SESSION_TTL")
    session_storage_dir: str = Field(
        default=str(_repo_root() / "data" / "sessions"),
        alias="LEVELUP_SESSION_DIR",
    )

    # --- CORS -----------------------------------------------------------
    cors_origins: list[str] = Field(
        default=["http://localhost:5173", "http://127.0.0.1:5173"],
        alias="LEVELUP_CORS_ORIGINS",
    )

    # --- DEMO_MODE -------------------------------------------------------
    demo_mode: bool = Field(default=False, alias="LEVELUP_DEMO_MODE")
    demo_fixtures_dir: str = Field(
        default=str(_repo_root() / "tests" / "fixtures" / "ref_player"),
        alias="LEVELUP_DEMO_FIXTURES_DIR",
    )

    # --- Données --------------------------------------------------------
    repo_root: str = Field(default=str(_repo_root()), alias="LEVELUP_REPO_ROOT")
    db_profiles_path: str = Field(
        default=str(_repo_root() / "db_profiles.json"),
        alias="LEVELUP_DB_PROFILES",
    )
    app_settings_path: str = Field(
        default=str(_repo_root() / "app_settings.json"),
        alias="LEVELUP_APP_SETTINGS",
    )

    # --- Observabilité --------------------------------------------------
    log_level: str = Field(default="INFO", alias="LEVELUP_LOG_LEVEL")
    log_json: bool = Field(default=False, alias="LEVELUP_LOG_JSON")

    # --- Versioning -----------------------------------------------------
    app_version: str = Field(default="6.5.0", alias="LEVELUP_APP_VERSION")

    @property
    def is_production(self) -> bool:
        """True si l'environnement n'est pas local/dev."""
        return os.getenv("LEVELUP_ENV", "dev").lower() == "production"


@lru_cache(maxsize=1)
def get_settings() -> ApiSettings:
    """Retourne l'instance singleton des paramètres API (mise en cache)."""
    return ApiSettings()
