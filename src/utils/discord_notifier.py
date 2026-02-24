"""Notifications Discord pour les opérations sync et backfill LevelUp.

Envoie un résumé via webhook Discord à la fin de chaque sync ou backfill :
- Heure de début / fin et durée totale
- Par joueur : matchs synchronisés, complétude des données
- Dernier match synchronisé : carte, mode de jeu, FDA (Kills/Deaths/Assists) et résultat

Configuration (app_settings.json) :
    "discord_notifications_enabled": true,
    "discord_webhook_url": "https://discord.com/api/webhooks/<id>/<token>"

Le module est entièrement failsafe — aucune exception n'est jamais propagée.

Usage :
    from src.utils.discord_notifier import (
        DiscordPlayerResult,
        LastMatchInfo,
        notify_operation_done,
        count_new_matches,
        count_matches_missing_data,
        fetch_last_match_info,
    )

    started_at = datetime.now(timezone.utc)
    # ... opération sync/backfill ...
    finished_at = datetime.now(timezone.utc)

    notify_operation_done(
        operation="sync_delta",
        started_at=started_at,
        finished_at=finished_at,
        players=[
            DiscordPlayerResult(
                gamertag="JGtm",
                xuid="1234567890",
                matches_synced=12,
                missing_data_count=0,
                last_match=fetch_last_match_info("1234567890"),
            ),
        ],
        success=True,
    )
"""

from __future__ import annotations

import json
import logging
import urllib.request
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

_PROJECT_ROOT = Path(__file__).resolve().parents[2]
_SHARED_DB = _PROJECT_ROOT / "data" / "warehouse" / "shared_matches.duckdb"
_APP_SETTINGS = _PROJECT_ROOT / "app_settings.json"


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
    is_ranked: bool = False
    start_time: datetime | None = None
    kills: int = 0
    deaths: int = 0
    assists: int = 0
    outcome: int = 0  # 1=Tie, 2=Win, 3=Loss, 4=Left
    score: int = 0


@dataclass
class DiscordPlayerResult:
    """Résultat d'une opération sync/backfill pour un joueur."""

    gamertag: str
    xuid: str | None = None
    matches_synced: int = 0  # Nouveaux matchs (sync) ou matchs traités (backfill)
    missing_data_count: int = 0  # Matchs avec données incomplètes après opération
    last_match: LastMatchInfo | None = None
    error: str | None = None


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

    Priorité de lecture de l'URL :
    1. Variable d'environnement `DISCORD_WEBHOOK_URL` (chargée depuis .env.local / .env)
    2. Clé `discord_webhook_url` dans app_settings.json (rétrocompatibilité)

    Retourne None si :
    - discord_notifications_enabled est False ou absent dans app_settings.json
    - Aucune URL trouvée ou URL invalide
    """
    cfg = _load_app_settings()
    if not cfg.get("discord_notifications_enabled", False):
        return None

    # 1. Charger .env.local / .env (même mécanisme que profile_api_tokens)
    try:
        from src.ui.profile_api_tokens import _load_dotenv_if_present

        _load_dotenv_if_present()
    except Exception:
        pass

    # 2. Variable d'environnement (source prioritaire)
    import os

    url = os.environ.get("DISCORD_WEBHOOK_URL", "").strip()

    # 3. Fallback : clé dans app_settings.json (rétrocompatibilité)
    if not url:
        url = str(cfg.get("discord_webhook_url") or "").strip()

    if url.startswith("https://discord.com/api/webhooks/"):
        return url
    return None


# =============================================================================
# Requêtes DuckDB
# =============================================================================


def fetch_last_match_info(xuid: str) -> LastMatchInfo | None:
    """Récupère les informations du dernier match depuis shared_matches.duckdb.

    Args:
        xuid: Xbox User ID du joueur.

    Returns:
        LastMatchInfo rempli, ou None si aucune donnée / erreur.
    """
    if not xuid or not _SHARED_DB.exists():
        return None
    try:
        import duckdb

        conn = duckdb.connect(str(_SHARED_DB), read_only=True)
        row = conn.execute(
            """
            SELECT
                mr.match_id,
                COALESCE(mr.map_name,          mr.map_id,       '—') AS map_name,
                COALESCE(mr.playlist_name,     '—')                  AS playlist_name,
                COALESCE(mr.game_variant_name, mr.pair_name,    '—') AS game_variant_name,
                COALESCE(mr.is_ranked,         FALSE)                AS is_ranked,
                mr.start_time,
                COALESCE(mp.kills,   0) AS kills,
                COALESCE(mp.deaths,  0) AS deaths,
                COALESCE(mp.assists, 0) AS assists,
                COALESCE(mp.outcome, 0) AS outcome,
                COALESCE(mp.score,   0) AS score
            FROM match_registry mr
            JOIN match_participants mp ON mr.match_id = mp.match_id
            WHERE mp.xuid = ?
            ORDER BY mr.start_time DESC
            LIMIT 1
            """,
            [xuid],
        ).fetchone()
        conn.close()
        if not row:
            return None
        return LastMatchInfo(
            match_id=str(row[0] or ""),
            map_name=str(row[1] or "—"),
            playlist_name=str(row[2] or "—"),
            game_variant_name=str(row[3] or "—"),
            is_ranked=bool(row[4]),
            start_time=row[5],
            kills=int(row[6] or 0),
            deaths=int(row[7] or 0),
            assists=int(row[8] or 0),
            outcome=int(row[9] or 0),
            score=int(row[10] or 0),
        )
    except Exception as exc:
        logger.debug(f"[Discord] fetch_last_match_info({xuid}): {exc}")
        return None


def count_new_matches(xuid: str, gamertag: str, since: datetime) -> int:
    """Compte les matchs ajoutés dans shared_matches depuis `since` pour ce joueur.

    Utilise first_sync_at pour identifier les matchs synchronisés lors de cette
    session de sync (uniquement pertinent pour sync_delta / sync_full).

    Args:
        xuid:     XUID du joueur.
        gamertag: Gamertag (utilisé pour les logs uniquement).
        since:    Timestamp de début de la sync.

    Returns:
        Nombre de nouveaux matchs, 0 en cas d'erreur.
    """
    if not xuid or not _SHARED_DB.exists():
        return 0
    try:
        import duckdb

        conn = duckdb.connect(str(_SHARED_DB), read_only=True)
        result = conn.execute(
            """
            SELECT COUNT(*)
            FROM match_registry mr
            JOIN match_participants mp ON mr.match_id = mp.match_id
            WHERE mp.xuid = ?
              AND mr.first_sync_at >= ?
            """,
            [xuid, since],
        ).fetchone()
        conn.close()
        return int(result[0]) if result else 0
    except Exception as exc:
        logger.debug(f"[Discord] count_new_matches({gamertag}): {exc}")
        return 0


def count_matches_missing_data(xuid: str) -> int:
    """Compte les matchs avec données incomplètes pour ce joueur.

    Un match est considéré incomplet si ses médailles ET ses événements highlight
    n'ont été chargés ni par le sync (medals_loaded / events_loaded) ni par le
    backfill (backfill_completed bitmask : bit 0 = medals, bit 1 = events).

    Args:
        xuid: XUID du joueur.

    Returns:
        Nombre de matchs incomplets, 0 en cas d'erreur.
    """
    if not xuid or not _SHARED_DB.exists():
        return 0
    try:
        import duckdb

        conn = duckdb.connect(str(_SHARED_DB), read_only=True)
        result = conn.execute(
            """
            SELECT COUNT(*)
            FROM match_registry mr
            JOIN match_participants mp ON mr.match_id = mp.match_id
            WHERE mp.xuid = ?
              AND (
                  -- médailles absentes : ni sync ni backfill (bit 0) ne les a chargées
                  (COALESCE(mr.medals_loaded, FALSE) = FALSE
                   AND (COALESCE(mr.backfill_completed, 0) & 1) = 0)
                  OR
                  -- events absents : ni sync ni backfill (bit 1) ne les a chargés
                  (COALESCE(mr.events_loaded, FALSE) = FALSE
                   AND (COALESCE(mr.backfill_completed, 0) & 2) = 0)
              )
            """,
            [xuid],
        ).fetchone()
        conn.close()
        return int(result[0]) if result else 0
    except Exception as exc:
        logger.debug(f"[Discord] count_matches_missing_data({xuid}): {exc}")
        return 0


# =============================================================================
# Construction du message Discord (Rich Embeds)
# =============================================================================

_OUTCOME_LABELS: dict[int, tuple[str, str]] = {
    1: ("⚖️", "Égalité"),
    2: ("🏆", "Victoire"),
    3: ("💀", "Défaite"),
    4: ("🚶", "Abandon"),
}

_OP_LABELS: dict[str, str] = {
    "sync_delta": "Sync delta",
    "sync_full": "Sync complète",
    "backfill": "Backfill",
}


def _format_duration(started_at: datetime, finished_at: datetime) -> str:
    """Formate la durée entre deux datetimes en chaîne lisible."""
    total_secs = max(0, int((finished_at - started_at).total_seconds()))
    if total_secs < 60:
        return f"{total_secs}s"
    mins, secs = divmod(total_secs, 60)
    if mins < 60:
        return f"{mins}m{secs:02d}s"
    hours, mins = divmod(mins, 60)
    return f"{hours}h{mins:02d}m{secs:02d}s"


def _format_player_field(
    player: DiscordPlayerResult,
    op_type: str,
) -> tuple[str, str]:
    """Construit le nom et la valeur du champ Discord Embed pour un joueur.

    Returns:
        Tuple (name, value) limités aux contraintes Discord (<= 256 / 1024 chars).
    """
    name = f"👤  {player.gamertag}"
    lines: list[str] = []

    # ── Résultat de l'opération ──────────────────────────────────────────────
    if op_type.startswith("sync"):
        lines.append(f"**+{player.matches_synced}** match(s) synchronisé(s)")
    else:
        lines.append(f"**{player.matches_synced}** match(s) traité(s)")

    # ── Complétude des données ───────────────────────────────────────────────
    if player.error:
        lines.append(f"⛔  Erreur : {player.error[:80]}")
    elif player.missing_data_count == 0:
        lines.append("✅  Données complètes")
    else:
        lines.append(f"⚠️   **{player.missing_data_count}** match(s) avec données incomplètes")

    # ── Dernier match ────────────────────────────────────────────────────────
    if player.last_match:
        lm = player.last_match
        outcome_icon, outcome_label = _OUTCOME_LABELS.get(lm.outcome, ("❓", "—"))
        ranked_tag = " · 🏅 Classé" if lm.is_ranked else ""
        kda = f"{lm.kills}K / {lm.deaths}D / {lm.assists}A"
        time_str = lm.start_time.strftime("%d/%m %H:%M") if lm.start_time else "—"
        lines.append(
            f"**Dernier match** ({time_str}){ranked_tag}\n"
            f"🗺️  **{lm.map_name}**  ·  🎮 {lm.game_variant_name}\n"
            f"📋  {lm.playlist_name}\n"
            f"📊  {kda}  ·  {outcome_icon} {outcome_label}"
        )

    return name[:256], "\n".join(lines)[:1024]


def build_embed_payload(
    operation: str,
    started_at: datetime,
    finished_at: datetime,
    players: list[DiscordPlayerResult],
    success: bool,
) -> dict[str, Any]:
    """Construit le payload JSON complet pour l'API Discord webhook.

    Args:
        operation:   "sync_delta", "sync_full" ou "backfill".
        started_at:  Heure de début.
        finished_at: Heure de fin.
        players:     Un DiscordPlayerResult par joueur.
        success:     False si au moins une erreur bloquante s'est produite.

    Returns:
        Dict prêt à sérialiser en JSON pour l'API Discord.
    """
    op_label = _OP_LABELS.get(operation, operation)
    duration = _format_duration(started_at, finished_at)

    has_errors = any(p.error for p in players)
    has_missing = any(p.missing_data_count > 0 for p in players)

    # Couleur de la barre latérale de l'embed
    if not success or has_errors:
        color = 0xED4245  # Rouge Discord
        status_icon = "❌"
    elif has_missing:
        color = 0xFEE75C  # Jaune Discord
        status_icon = "⚠️"
    else:
        color = 0x57F287  # Vert Discord
        status_icon = "✅"

    t_start = started_at.strftime("%H:%M:%S")
    t_end = finished_at.strftime("%H:%M:%S")
    total_new = sum(p.matches_synced for p in players)

    description = (
        f"**{status_icon}  {op_label}** terminée en **{duration}**\n"
        f"🕐  `{t_start}`  →  `{t_end}`\n"
        f"👥  {len(players)} joueur(s)  ·  {total_new} match(s) traité(s)"
    )

    # Champs par joueur (max 25 selon l'API Discord)
    fields: list[dict[str, Any]] = []
    for player in players[:25]:
        fname, fvalue = _format_player_field(player, operation)
        if fvalue:
            fields.append({"name": fname, "value": fvalue, "inline": False})

    embed: dict[str, Any] = {
        "title": f"🎮  LevelUp — {op_label}",
        "description": description,
        "color": color,
        "fields": fields,
        "footer": {"text": "LevelUp · Halo Infinite Stats"},
        "timestamp": finished_at.replace(tzinfo=timezone.utc).isoformat()
        if finished_at.tzinfo is None
        else finished_at.isoformat(),
    }

    return {"embeds": [embed]}


# =============================================================================
# Envoi HTTP (stdlib uniquement, aucune dépendance externe)
# =============================================================================


def send_discord_notification(
    payload: dict[str, Any],
    webhook_url: str,
) -> bool:
    """Envoie le payload JSON au webhook Discord via urllib.

    Args:
        payload:     Dict à sérialiser en JSON.
        webhook_url: URL complète du webhook Discord.

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
        logger.warning(f"[Discord] Envoi échoué : {exc}")
        return False


# =============================================================================
# Point d'entrée public principal
# =============================================================================


def notify_operation_done(
    operation: str,
    started_at: datetime,
    finished_at: datetime,
    players: list[DiscordPlayerResult],
    success: bool = True,
    *,
    disabled: bool = False,
) -> None:
    """Envoie la notification Discord de fin d'opération.

    Fonction entièrement failsafe : ne lève jamais d'exception.
    Toute erreur est loguée en WARNING et ignorée.

    Args:
        operation:   "sync_delta", "sync_full" ou "backfill".
        started_at:  Heure de début de l'opération (with ou without timezone).
        finished_at: Heure de fin.
        players:     Liste de DiscordPlayerResult, un par joueur traité.
        success:     False si au moins une erreur globale s'est produite.
        disabled:    Si True, court-circuitage immédiat (flag --no-discord).
    """
    if disabled:
        return

    try:
        webhook_url = _get_webhook_url()
        if not webhook_url:
            # Non configuré ou désactivé → silencieux
            return

        if not players:
            logger.debug("[Discord] Aucun joueur à notifier, skip")
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
        logger.warning(f"[Discord] Erreur inattendue lors de la notification : {exc}")
