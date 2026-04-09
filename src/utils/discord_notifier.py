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
from datetime import datetime, timezone
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
    map_id: str = ""
    playlist_id: str = ""
    pair_id: str = ""
    game_variant_id: str = ""
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
        cfg = _load_app_settings()
        if not cfg.get("discord_notify_sync", True):
            logger.debug("[Discord] Notif sync désactivée par discord_notify_sync=false")
            return

        webhook_url = _get_webhook_url()
        if not webhook_url:
            return

        if not players:
            logger.debug("[Discord] Aucun joueur à notifier, skip")
            return

        if skip_idle:
            all_players = players
            players = [p for p in players if p.matches_synced > 0 or p.error]
            if not players:
                logger.info("[Discord] Tous les joueurs à jour — envoi embed allégé")
                payload = build_embed_payload(
                    operation=operation,
                    started_at=started_at,
                    finished_at=finished_at,
                    players=all_players,
                    success=success,
                )
                ok = send_discord_notification(payload, webhook_url)
                if ok:
                    logger.info("[Discord] Notification 'déjà à jour' envoyée")
                else:
                    logger.warning("[Discord] Notification non reçue par Discord (voir logs)")
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


# =============================================================================
# Notification de déploiement — nouvelle version majeure
# =============================================================================

_README = Path(__file__).resolve().parents[2] / "README.md"
_MAX_DISCORD_BODY = 1900  # Discord limite les embeds à 2048 chars, marge de sécurité


def _is_major_minor_change(old: str, new: str) -> bool:
    """Retourne True si X ou Y changent dans vX.Y.Z.

    Retourne False si old est vide (premier démarrage) pour éviter le spam.
    """
    if not old or not old.strip():
        return False

    def _parse(v: str) -> tuple[int, int]:
        parts = v.lstrip("v").split(".")
        return (int(parts[0]), int(parts[1])) if len(parts) >= 2 else (0, 0)

    try:
        return _parse(old) != _parse(new)
    except (ValueError, IndexError):
        return old.strip() != new.strip()


def _extract_whats_new(version: str) -> str:
    """Extrait la section What's New du README pour la version donnée.

    Cherche le bloc ``**vX.Y`` correspondant et retourne les bullet points
    jusqu'au prochain bloc ``**v``.

    Retourne une chaîne vide si la section n'est pas trouvée.
    """
    if not _README.exists():
        return ""
    try:
        content = _README.read_text(encoding="utf-8")
    except Exception:
        return ""

    # Construire le préfixe à chercher : "**v6.4" à partir de "6.4.0"
    parts = version.lstrip("v").split(".")
    if len(parts) < 2:
        return ""
    version_prefix = f"**v{parts[0]}.{parts[1]}"

    lines = content.splitlines()
    in_section = False
    collected: list[str] = []

    for line in lines:
        if not in_section:
            if line.strip().startswith(version_prefix):
                in_section = True
                collected.append(line.strip())
            continue

        # Fin de section : nouvelle entrée **vX.Y ou heading de niveau 2
        stripped = line.strip()
        if stripped.startswith("**v") and stripped != collected[0]:
            break
        if stripped.startswith("## "):
            break
        collected.append(stripped)

    return "\n".join(collected).strip()


def notify_new_version(current_version: str) -> bool:
    """Envoie une notification Discord lors du déploiement d'une nouvelle version majeure.

    Conditions requises (toutes doivent être vraies) :
    - discord_notifications_enabled = True et webhook valide
    - discord_notify_new_version = True
    - Variable d'env LEVELUP_NOTIFY_VERSIONS=1 (opt-in explicite, prod uniquement)
    - La version courante diffère de last_notified_version sur major/minor

    Retourne True si la notification a été envoyée avec succès.
    Fonction entièrement failsafe.
    """
    import os

    try:
        # Guard env var : opt-in explicite (prod uniquement)
        if os.environ.get("LEVELUP_NOTIFY_VERSIONS", "").strip() not in ("1", "true", "True"):
            logger.debug("[Discord] LEVELUP_NOTIFY_VERSIONS non défini — notif version ignorée")
            return False

        cfg = _load_app_settings()
        if not cfg.get("discord_notify_new_version", True):
            logger.debug("[Discord] Notif version désactivée par discord_notify_new_version=false")
            return False

        webhook_url = _get_webhook_url()
        if not webhook_url:
            return False

        whats_new = _extract_whats_new(current_version)
        description = (
            whats_new[:_MAX_DISCORD_BODY] if whats_new else (f"Version {current_version} déployée.")
        )

        payload: dict[str, Any] = {
            "embeds": [
                {
                    "title": f"🚀 LevelUp v{current_version} — Nouvelle version déployée",
                    "description": description,
                    "color": 0x5865F2,  # Blurple Discord
                    "footer": {"text": f"LevelUp v{current_version}"},
                    "timestamp": datetime.now(tz=timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
                }
            ]
        }

        ok = send_discord_notification(payload, webhook_url)
        if ok:
            logger.info("[Discord] Notification nouvelle version v%s envoyée", current_version)
        else:
            logger.warning("[Discord] Notification version non reçue par Discord")
        return ok

    except Exception as exc:
        logger.warning("[Discord] Erreur notif version : %s", exc)
        return False
