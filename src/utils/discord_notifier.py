"""Notifications Discord pour les opérations sync et backfill LevelUp.

Envoie un résumé via webhook Discord à la fin de chaque sync ou backfill.

Configuration (app_settings.json) :
    "discord_notifications_enabled": true,
    "discord_webhook_url": "https://discord.com/api/webhooks/<id>/<token>"

Le module est entièrement failsafe — aucune exception n'est jamais propagée.

Usage :
    from src.utils.discord_notifier import (
        DiscordPlayerResult,
        LastMatchInfo,
        notify_operation_done,
    )
"""

from __future__ import annotations

import json
import logging
import urllib.request
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Any

# ── Re-exports pour compatibilité ascendante ─────────────────────────────────
from src.utils._discord_embed import (  # noqa: F401
    _format_duration,
    _format_player_field,
    _to_local,
    build_embed_payload,
)
from src.utils._discord_queries import (  # noqa: F401
    count_matches_missing_data,
    count_new_matches,
    fetch_last_match_info,
)

logger = logging.getLogger(__name__)

_APP_SETTINGS = Path(__file__).resolve().parents[2] / "app_settings.json"


# =============================================================================
# Dataclasses publiques
# =============================================================================


@dataclass
class LastMatchInfo:
    """Informations sur le dernier match du joueur."""

    match_id: str = ""
    map_name: str = "—"
    playlist_name: str = "—"
    game_variant_name: str = "—"
    pair_name: str = "—"
    is_ranked: bool = False
    start_time: datetime | None = None
    kills: int = 0
    deaths: int = 0
    assists: int = 0
    outcome: int = 0  # 1=Tie, 2=Win, 3=Loss, 4=Left
    score: int = 0
    participants_count: int = 0
    squad_friends: list[str] = field(default_factory=list)


@dataclass
class DiscordPlayerResult:
    """Résultat d'une opération sync/backfill pour un joueur."""

    gamertag: str
    xuid: str | None = None
    matches_synced: int = 0
    missing_data_count: int = 0
    last_match: LastMatchInfo | None = None
    error: str | None = None
    backfill_counts: dict = field(default_factory=dict)


# =============================================================================
# Lecture de la configuration
# =============================================================================


def _load_app_settings() -> dict[str, Any]:
    """Charge app_settings.json → dict. Retourne {} en cas d'échec."""
    if not _APP_SETTINGS.exists():
        return {}
    try:
        with open(_APP_SETTINGS, encoding="utf-8") as fh:
            return json.load(fh)
    except Exception:
        return {}


def _get_webhook_url() -> str | None:
    """Retourne l'URL du webhook Discord si les notifications sont activées.

    Priorité :
    1. Variable d'environnement DISCORD_WEBHOOK_URL (.env.local / .env)
    2. Clé discord_webhook_url dans app_settings.json (rétrocompatibilité)

    Retourne None si désactivé ou URL invalide.
    """
    cfg = _load_app_settings()
    if not cfg.get("discord_notifications_enabled", False):
        return None

    try:
        from src.ui.profile_api_tokens import _load_dotenv_if_present

        _load_dotenv_if_present()
    except Exception:
        pass

    import os

    url = os.environ.get("DISCORD_WEBHOOK_URL", "").strip()

    if not url:
        url = str(cfg.get("discord_webhook_url") or "").strip()

    if url.startswith("https://discord.com/api/webhooks/"):
        return url

    logger.warning(
        "[Discord] Notifications activées mais aucun webhook valide trouvé. "
        "Vérifiez DISCORD_WEBHOOK_URL dans .env.local ou discord_webhook_url "
        "dans app_settings.json."
    )
    return None


# =============================================================================
# Envoi HTTP (stdlib uniquement)
# =============================================================================


def send_discord_notification(
    payload: dict[str, Any],
    webhook_url: str,
) -> bool:
    """Envoie le payload JSON au webhook Discord via urllib.

    Returns:
        True si Discord répond 200 ou 204, False sinon.
    """
    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(
        webhook_url,
        data=data,
        headers={
            "Content-Type": "application/json; charset=utf-8",
            "User-Agent": "LevelUp-HaloBot/1.0 (+https://github.com/levelup)",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status in (200, 204)
    except Exception as exc:
        logger.warning("[Discord] Envoi échoué : %s", exc)
        return False


# =============================================================================
# Point d'entrée public principal
# =============================================================================


def notify_operation_done(  # noqa: PLR0913
    operation: str,
    started_at: datetime,
    finished_at: datetime,
    players: list[DiscordPlayerResult],
    success: bool = True,
    *,
    disabled: bool = False,
    skip_idle: bool = False,
) -> None:
    """Envoie la notification Discord de fin d'opération.

    Fonction entièrement failsafe : ne lève jamais d'exception.
    """
    if disabled:
        return

    try:
        webhook_url = _get_webhook_url()
        if not webhook_url:
            return

        if not players:
            logger.debug("[Discord] Aucun joueur à notifier, skip")
            return

        if skip_idle:
            players = [p for p in players if p.matches_synced > 0 or p.error]
            if not players:
                logger.info("[Discord] Tous les joueurs à jour, notification omise")
                return

        payload = build_embed_payload(
            operation=operation,
            started_at=started_at,
            finished_at=finished_at,
            players=players,
            success=success,
        )

        ok = send_discord_notification(payload, webhook_url)
        if ok:
            logger.info("[Discord] Notification envoyée avec succès")
        else:
            logger.warning("[Discord] Notification non reçue par Discord (voir logs)")

    except Exception as exc:
        logger.warning("[Discord] Erreur inattendue : %s", exc)
