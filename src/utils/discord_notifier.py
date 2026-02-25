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


def _to_local(dt: datetime) -> datetime:
    """Convertit un datetime UTC vers l'heure locale du système."""
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt.astimezone()


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
    pair_name: str = "—"
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
                COALESCE(mr.game_variant_name, '—')                  AS game_variant_name,
                COALESCE(mr.pair_name,         '—')                  AS pair_name,
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
            pair_name=str(row[4] or "—"),
            is_ranked=bool(row[5]),
            start_time=row[6],
            kills=int(row[7] or 0),
            deaths=int(row[8] or 0),
            assists=int(row[9] or 0),
            outcome=int(row[10] or 0),
            score=int(row[11] or 0),
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


def _get_discord_lang() -> str:
    """Retourne la langue Discord configurée dans app_settings.json."""
    try:
        with open(_APP_SETTINGS, encoding="utf-8") as fh:
            data = json.load(fh)
        return data.get("discord_lang", "fr")
    except Exception:
        return "fr"


# Traductions des noms de playlists : délégué à src.ui.translations.
# Pas de dict local — source unique de vérité.


def _localize_playlist(name: str, lang: str = "fr") -> str:
    """Retourne la traduction du nom de playlist via le module translations."""
    try:
        from src.ui.translations import translate_playlist_name

        result = translate_playlist_name(name, lang=lang)
        return result if result is not None else name
    except Exception:
        return name


def _clean_game_variant(name: str) -> str:
    """Supprime le préfixe 'Type:' des noms de variantes (ex: 'Slayer:Arena Fiesta' → 'Arena Fiesta')."""
    if ":" in name:
        _, _, after = name.partition(":")
        return after.strip() or name
    return name


def _resolve_mode_label(pair_name: str, game_variant_name: str, lang: str = "fr") -> str:
    """Résout le libellé du mode de jeu dans la langue demandée."""
    if pair_name and pair_name != "—":
        try:
            from src.analysis.mode_categories import normalize_pair_name_to_mode_ui

            mode = normalize_pair_name_to_mode_ui(pair_name, lang=lang)
            if mode:
                return mode
        except Exception:
            pass
    return _clean_game_variant(game_variant_name)


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
    from src.ui.i18n.cli import discord_t

    lang = _get_discord_lang()
    name = f"👤  {player.gamertag}"
    lines: list[str] = []

    # ── Résultat de l'opération ──────────────────────────────────────────────
    if op_type.startswith("sync"):
        if player.matches_synced == 0:
            lines.append(discord_t("discord_up_to_date_sync"))
        else:
            lines.append(discord_t("discord_matches_synced", count=player.matches_synced))
    else:
        if player.matches_synced == 0:
            lines.append(discord_t("discord_no_matches_to_reprocess"))
        else:
            lines.append(discord_t("discord_matches_processed", count=player.matches_synced))

    # ── Complétude des données ───────────────────────────────────────────────
    if player.error:
        lines.append(discord_t("discord_error_field", error=player.error[:80]))
    elif player.missing_data_count == 0:
        lines.append(discord_t("discord_data_complete"))
    else:
        lines.append(discord_t("discord_data_incomplete", count=player.missing_data_count))

    # ── Dernier match ────────────────────────────────────────────────────────
    if player.last_match:
        lm = player.last_match
        _OUTCOME_KEYS: dict[int, str] = {
            1: "discord_outcome_draw",
            2: "discord_outcome_win",
            3: "discord_outcome_loss",
            4: "discord_outcome_quit",
        }
        _OUTCOME_ICONS: dict[int, str] = {1: "⚖️", 2: "🏆", 3: "💀", 4: "🚶"}
        outcome_key = _OUTCOME_KEYS.get(lm.outcome)
        outcome_icon = _OUTCOME_ICONS.get(lm.outcome, "❓")
        outcome_label = discord_t(outcome_key) if outcome_key else "—"
        ranked_tag = f" · 🏅 {discord_t('discord_ranked_tag')}" if lm.is_ranked else ""
        kda = discord_t("discord_kda", k=lm.kills, d=lm.deaths, a=lm.assists)
        _st = _to_local(lm.start_time) if lm.start_time else None
        time_str = _st.strftime("%d/%m %H:%M") if _st else "—"
        variant = _resolve_mode_label(lm.pair_name, lm.game_variant_name, lang=lang)
        playlist = _localize_playlist(lm.playlist_name, lang=lang)
        last_match_label = discord_t("discord_last_match")
        lines.append(
            f"**{last_match_label}** ({time_str}){ranked_tag}\n"
            f"🗺️  **{lm.map_name}**  ·  🎮 {variant}\n"
            f"📋  {playlist}\n"
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
    from src.ui.i18n.cli import discord_t

    _OP_KEYS: dict[str, str] = {
        "sync_delta": "discord_op_sync_delta",
        "sync_full":  "discord_op_sync_full",
        "backfill":   "discord_op_backfill",
    }
    op_label = discord_t(_OP_KEYS.get(operation, operation))
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

    t_start = _to_local(started_at).strftime("%H:%M:%S")
    t_end = _to_local(finished_at).strftime("%H:%M:%S")
    total_new = sum(p.matches_synced for p in players)

    if operation.startswith("sync"):
        if total_new > 0:
            matches_str = discord_t("discord_matches_synced", count=total_new)
        else:
            matches_str = discord_t("discord_no_new_matches")
    else:
        if total_new > 0:
            matches_str = discord_t("discord_matches_processed", count=total_new)
        else:
            matches_str = discord_t("discord_all_up_to_date")

    completed_line = discord_t("discord_completed_in", status=status_icon, op=op_label, duration=duration)
    time_line = discord_t("discord_time_range", t_start=t_start, t_end=t_end)
    players_line = discord_t("discord_player_count", count=len(players)) + "  ·  " + matches_str
    description = f"{completed_line}\n{time_line}\n{players_line}"

    # Champs par joueur (max 25 selon l'API Discord)
    fields: list[dict[str, Any]] = []
    for player in players[:25]:
        fname, fvalue = _format_player_field(player, operation)
        if fvalue:
            fields.append({"name": fname, "value": fvalue, "inline": False})

    embed: dict[str, Any] = {
        "title": discord_t("discord_title", op=op_label),
        "description": description,
        "color": color,
        "fields": fields,
        "footer": {"text": discord_t("discord_footer")},
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
