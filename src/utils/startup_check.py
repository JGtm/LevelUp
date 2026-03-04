"""Vérifications de configuration au démarrage de l'app.

Ce module valide que tous les paramètres critiques de app_settings.json
sont correctement définis avant que l'app ne démarre.

Usage dans streamlit_app.py :
    from src.utils.startup_check import check_app_settings
    warnings, errors = check_app_settings(settings)
"""

from __future__ import annotations

import json
import os
from pathlib import Path
from typing import TYPE_CHECKING

from src.utils.secrets import get_secret

if TYPE_CHECKING:
    from src.ui.settings import AppSettings

_SUPPORTED_LANGS = {"fr", "en"}


def _raw_app_settings() -> dict:
    """Lit le fichier app_settings.json brut (pour les clés absentes de AppSettings)."""
    from src.ui.settings import get_settings_path

    path = get_settings_path()
    try:
        with open(path, encoding="utf-8") as f:
            return json.load(f) or {}
    except Exception:
        return {}


def _check_discord_config(raw: dict, warnings: list[str]) -> None:
    """Vérifie la configuration Discord."""
    discord_enabled = raw.get("discord_notifications_enabled", False)
    doppler_active = raw.get("doppler_enabled", False)
    if discord_enabled and not doppler_active:
        discord_url = (
            str(raw.get("discord_webhook_url") or "").strip()
            or str(get_secret("DISCORD_WEBHOOK_URL") or "").strip()
        )
        if not discord_url:
            warnings.append(
                "🔔 `discord_notifications_enabled` est activé mais "
                "`discord_webhook_url` n'est pas défini (ni dans app_settings.json "
                "ni dans la variable d'environnement `DISCORD_WEBHOOK_URL`). "
                "Les notifications Discord seront désactivées silencieusement."
            )


def _check_media_config(settings: AppSettings, warnings: list[str]) -> None:  # type: ignore[name-defined]
    """Vérifie la configuration des médias."""
    if not settings.media_enabled:
        return
    capture_dir = settings.media_captures_base_dir
    if capture_dir and not Path(capture_dir).exists():
        warnings.append(
            f"📁 `media_captures_base_dir` pointe vers un dossier inexistant : "
            f"`{capture_dir}`. "
            "La galerie média sera vide."
        )
    elif not capture_dir and not settings.media_screens_dir and not settings.media_videos_dir:
        warnings.append(
            "📁 `media_enabled` est activé mais aucun dossier de captures "
            "n'est configuré (`media_captures_base_dir` vide). "
            "La galerie média sera vide."
        )


def _check_players_config(errors: list[str]) -> None:
    """Vérifie qu'au moins un joueur est initialisé."""
    try:
        from src.utils.paths import PLAYERS_DIR

        if not PLAYERS_DIR.exists() or not any(
            (PLAYERS_DIR / d / "stats.duckdb").exists()
            for d in os.listdir(PLAYERS_DIR)
            if (PLAYERS_DIR / d).is_dir()
        ):
            errors.append(
                "👤 Aucun joueur trouvé dans `data/players/`. "
                "Lance `python scripts/sync.py --gamertag <ton_gamertag>` "
                "pour initialiser ta base de données."
            )
    except Exception:
        pass


def check_app_settings(settings: AppSettings) -> tuple[list[str], list[str]]:
    """Valide la configuration de l'app au démarrage.

    Returns:
        Tuple (warnings, errors) : listes de messages.
        - warnings : problèmes non bloquants (fonctionnalité dégradée possible)
        - errors : problèmes bloquants (fonctionnalité indisponible)
    """
    warnings: list[str] = []
    errors: list[str] = []

    raw = _raw_app_settings()

    # ── repository_mode ────────────────────────────────────────────────────────
    if settings.repository_mode not in {"duckdb"}:
        errors.append(
            f"⚙️ `repository_mode` invalide : `{settings.repository_mode}`. "
            "Seule la valeur `duckdb` est supportée."
        )

    # ── Langue ────────────────────────────────────────────────────────────────
    if settings.lang not in _SUPPORTED_LANGS:
        warnings.append(
            f"🌐 `lang` inconnu : `{settings.lang}`. "
            f"Langues supportées : {', '.join(sorted(_SUPPORTED_LANGS))}. "
            "L'interface utilisera le français par défaut."
        )

    _check_discord_config(raw, warnings)
    _check_media_config(settings, warnings)
    _check_players_config(errors)

    return warnings, errors
