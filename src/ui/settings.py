"""Gestion des paramètres utilisateur (persistés).

Objectif:
- Déplacer les "paramètres" hors sidebar (onglet Paramètres)
- Permettre une exécution sur NAS/Docker avec un fichier de config monté

Le chemin est configurable via OPENSPARTAN_SETTINGS_PATH.
"""

from __future__ import annotations

import json
import logging
import os
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

logger = logging.getLogger(__name__)


def get_settings_path() -> str:
    """Retourne le chemin du fichier app_settings.json."""
    override = os.environ.get("OPENSPARTAN_SETTINGS_PATH")
    if override and str(override).strip():
        return str(override).strip()
    repo_root = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
    return os.path.join(repo_root, "app_settings.json")


class AppSettings(BaseModel):
    """Paramètres utilisateur persistés en JSON.

    Pydantic v2 gère automatiquement la coercion des types (str→bool, str→int, etc.)
    et ignore les clés inconnues (extra='ignore') comme ``_comment`` ou
    ``discord_notifications_enabled``.
    """

    model_config = ConfigDict(extra="ignore", str_strip_whitespace=True)

    # Médias
    media_enabled: bool = True
    media_screens_dir: str = ""  # Déprécié: migration vers media_captures_base_dir
    media_videos_dir: str = ""  # Déprécié: migration vers media_captures_base_dir
    media_captures_base_dir: str = ""  # Base unique: base_dir/{gamertag}/ pour captures
    media_tolerance_minutes: int = Field(default=3, ge=0)

    # UX
    refresh_clears_caches: bool = False

    # Source
    prefer_spnkr_db_if_available: bool = True

    # SPNKr (API → DB) : rafraîchissement automatique
    spnkr_refresh_on_start: bool = True
    spnkr_refresh_on_manual_refresh: bool = True
    spnkr_refresh_match_type: Literal["all", "matchmaking", "custom", "local"] = "matchmaking"
    spnkr_refresh_max_matches: int = Field(default=200, ge=1)
    spnkr_refresh_rps: int = Field(default=3, ge=1)
    spnkr_refresh_with_highlight_events: bool = False

    # Backfill après synchronisation
    spnkr_refresh_with_backfill: bool = False  # Backfill complet après sync
    spnkr_refresh_backfill_medals: bool = False
    spnkr_refresh_backfill_events: bool = False
    spnkr_refresh_backfill_skill: bool = False
    spnkr_refresh_backfill_personal_scores: bool = False
    spnkr_refresh_backfill_performance_scores: bool = True  # Par défaut activé
    spnkr_refresh_backfill_aliases: bool = False
    spnkr_refresh_backfill_lusr: bool = True  # LUSR local (sans API) — par défaut activé

    # Fichiers (overrides optionnels)
    aliases_path: str = ""
    profiles_path: str = ""

    # Profil joueur (bannière/rang) — aucun accès réseau implicite
    profile_assets_download_enabled: bool = False
    profile_assets_auto_refresh_hours: int = Field(default=24, ge=0)  # 0 = désactivé

    # Profil joueur (auto depuis API Waypoint via SPNKr) — opt-in
    profile_api_enabled: bool = False
    profile_api_auto_refresh_hours: int = Field(default=6, ge=0)

    profile_banner: str = ""  # URL https://... ou chemin local
    profile_emblem: str = ""  # URL https://... ou chemin local
    profile_backdrop: str = ""  # URL https://... ou chemin local
    profile_nameplate: str = ""  # URL https://... ou chemin local
    profile_service_tag: str = ""  # ex: "SPTA"
    profile_id_badge_text_color: str = ""  # ex: "#FFFFFF"
    profile_rank_label: str = ""  # ex: "Diamond III" / "Héros" / etc.
    profile_rank_subtitle: str = ""  # ex: "CSR 1540" / "Saison 5" / etc.

    # Architecture de données v4
    repository_mode: Literal["duckdb"] = "duckdb"
    enable_duckdb_analytics: bool = False

    # Doppler secrets (opt-in)
    doppler_enabled: bool = False  # Charger les secrets depuis Doppler au démarrage
    doppler_project: str = ""  # Nom du projet Doppler (ex: "levelup")
    doppler_config: str = ""  # Config Doppler (ex: "dev", "prod")

    # Tailscale funnel (opt-in)
    tailscale_funnel_enabled: bool = False  # Exposer l'app via Tailscale au démarrage

    # Internationalisation
    lang: Literal["fr", "en"] = "fr"  # Langue de l'UI
    discord_lang: Literal["fr", "en"] = "fr"  # Langue des messages Discord
    cli_lang: Literal["fr", "en"] = "fr"  # Langue des scripts CLI

    # --- Validators ---

    @model_validator(mode="before")
    @classmethod
    def _strip_none_values(cls, data: Any) -> Any:
        """Supprime les valeurs None pour laisser les défauts Pydantic s'appliquer."""
        if isinstance(data, dict):
            return {k: v for k, v in data.items() if v is not None}
        return data

    @field_validator("spnkr_refresh_match_type", mode="before")
    @classmethod
    def _normalize_match_type(cls, v: Any) -> str:
        """Normalise le type de match en minuscules avec fallback."""
        s = str(v).strip().lower()
        return s if s in {"all", "matchmaking", "custom", "local"} else "matchmaking"

    @field_validator("lang", "discord_lang", "cli_lang", mode="before")
    @classmethod
    def _normalize_lang(cls, v: Any) -> str:
        """Normalise la langue en minuscules avec fallback sur 'fr'."""
        s = str(v).strip().lower()
        return s if s in {"fr", "en"} else "fr"

    @field_validator("repository_mode", mode="before")
    @classmethod
    def _normalize_repo_mode(cls, _v: Any) -> str:
        """Seul mode supporté : duckdb."""
        return "duckdb"

    @field_validator(
        "media_tolerance_minutes",
        "profile_assets_auto_refresh_hours",
        "profile_api_auto_refresh_hours",
        mode="before",
    )
    @classmethod
    def _clamp_non_negative(cls, v: Any, info: Any) -> int:
        """Convertit en int ≥ 0 avec fallback sur le défaut du champ."""
        try:
            return max(0, int(v))
        except (ValueError, TypeError):
            return cls.model_fields[info.field_name].default

    @field_validator("spnkr_refresh_max_matches", "spnkr_refresh_rps", mode="before")
    @classmethod
    def _clamp_positive(cls, v: Any, info: Any) -> int:
        """Convertit en int ≥ 1 avec fallback sur le défaut du champ."""
        try:
            return max(1, int(v))
        except (ValueError, TypeError):
            return cls.model_fields[info.field_name].default

    @model_validator(mode="after")
    def _migrate_legacy_media_dirs(self) -> AppSettings:
        """Migration : si base vide mais anciens champs renseignés, propose le parent commun."""
        if not self.media_captures_base_dir and (self.media_screens_dir or self.media_videos_dir):
            from pathlib import Path

            paths = [Path(p) for p in [self.media_screens_dir, self.media_videos_dir] if p]
            if paths:
                try:
                    common = paths[0].parent
                    for p in paths[1:]:
                        common = Path(os.path.commonpath([str(common), str(p)]))
                    if str(common).strip():
                        self.media_captures_base_dir = str(common)
                except (ValueError, TypeError):
                    pass
        return self


def load_settings() -> AppSettings:
    """Charge les paramètres depuis le fichier JSON."""
    path = get_settings_path()
    if not os.path.exists(path):
        try:
            os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
            with open(path, "w", encoding="utf-8") as f:
                f.write("{}\n")
        except OSError:
            pass
        return AppSettings()
    try:
        with open(path, encoding="utf-8") as f:
            obj = json.load(f) or {}
    except Exception:
        return AppSettings()

    if not isinstance(obj, dict):
        return AppSettings()

    try:
        settings = AppSettings.model_validate(obj)
        logger.info("Settings chargées depuis %s", path)
        return settings
    except Exception:
        return AppSettings()


def save_settings(settings: AppSettings) -> tuple[bool, str]:
    """Sauvegarde les paramètres dans le fichier JSON."""
    logger.info("Settings sauvegardées")
    path = get_settings_path()
    try:
        os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
        with open(path, "w", encoding="utf-8") as f:
            json.dump(settings.model_dump(), f, ensure_ascii=False, indent=2)
        return True, ""
    except Exception as e:
        return False, f"Impossible d'écrire {path}: {e}"
