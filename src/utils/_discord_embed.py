"""Construction des messages Discord (Rich Embeds).

Formate les données de sync/backfill en payload JSON pour l'API webhook Discord.
"""

from __future__ import annotations

import json
import logging
from datetime import datetime, timezone
from pathlib import Path
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from src.utils.discord_notifier import DiscordPlayerResult

logger = logging.getLogger(__name__)

_APP_SETTINGS = Path(__file__).resolve().parents[2] / "app_settings.json"


def _to_local(dt: datetime) -> datetime:
    """Convertit un datetime UTC vers l'heure locale du système."""
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt.astimezone()


def _get_discord_lang() -> str:
    """Retourne la langue Discord configurée dans app_settings.json."""
    try:
        with open(_APP_SETTINGS, encoding="utf-8") as fh:
            data = json.load(fh)
        return data.get("discord_lang", "fr")
    except Exception:
        return "fr"


def _localize_playlist(name: str, lang: str = "fr") -> str:
    """Retourne la traduction du nom de playlist via le module translations."""
    try:
        from src.ui.translations import translate_playlist_name

        result = translate_playlist_name(name, lang=lang)
        return result if result is not None else name
    except Exception:
        return name


def _clean_game_variant(name: str) -> str:
    """Supprime le préfixe 'Type:' des noms de variantes."""
    if ":" in name:
        _, _, after = name.partition(":")
        return after.strip() or name
    return name


def _resolve_mode_label(
    pair_name: str,
    game_variant_name: str,
    lang: str = "fr",
) -> str:
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


# Constantes de mapping outcome → clé i18n / icône
_OUTCOME_KEYS: dict[int, str] = {
    1: "discord_outcome_draw",
    2: "discord_outcome_win",
    3: "discord_outcome_loss",
    4: "discord_outcome_quit",
}
_OUTCOME_ICONS: dict[int, str] = {1: "⚖️", 2: "🏆", 3: "💀", 4: "🚶"}


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

    # ── Résultat de l'opération ──────────────────────────────────────────
    if op_type.startswith("sync"):
        if player.matches_synced == 0:
            lines.append(discord_t("discord_up_to_date_sync"))
        else:
            lines.append(discord_t("discord_matches_synced", count=player.matches_synced))
    elif player.matches_synced == 0:
        lines.append(discord_t("discord_no_matches_to_reprocess"))
    else:
        lines.append(discord_t("discord_matches_processed", count=player.matches_synced))

    # ── Complétude des données ───────────────────────────────────────────
    if player.error:
        lines.append(discord_t("discord_error_field", error=player.error[:80]))
    elif player.missing_data_count == 0:
        lines.append(discord_t("discord_data_complete"))
    else:
        lines.append(discord_t("discord_data_incomplete", count=player.missing_data_count))
    _counts = getattr(player, "backfill_counts", {}) or {}
    _BACKFILL_I18N_KEYS = {
        "medals_inserted": "discord_bf_medals",
        "events_inserted": "discord_bf_events",
        "lusr_computed": "discord_bf_lusr",
        "csr_fetched": "discord_bf_csr",
        "sessions_updated": "discord_bf_sessions",
        "citations_computed": "discord_bf_citations",
        "killer_victim_pairs_inserted": "discord_bf_kvp",
        "personal_scores_inserted": "discord_bf_personal_scores",
        "performance_scores_inserted": "discord_bf_perf_scores",
        "aliases_inserted": "discord_bf_aliases",
        "pve_stats_inserted": "discord_bf_pve",
    }
    for _key, _i18n_key in _BACKFILL_I18N_KEYS.items():
        _n = _counts.get(_key, 0)
        if _n > 0:
            lines.append(discord_t(_i18n_key, count=_n))

    # ── Dernier match ────────────────────────────────────────────────────
    if player.last_match:
        _append_last_match_lines(lines, player.last_match, lang)

    return name[:256], "\n".join(lines)[:1024]


def _append_last_match_lines(
    lines: list[str],
    lm: object,
    lang: str,
) -> None:
    """Ajoute les lignes du dernier match au champ Discord."""
    from src.ui.i18n.cli import discord_t

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

    if lm.squad_friends:
        lines.append(discord_t("discord_squad_match", n=lm.participants_count))
        lines.append(
            discord_t("discord_squad_friends", friends=", ".join(lm.squad_friends)),
        )


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
        "sync_full": "discord_op_sync_full",
        "backfill": "discord_op_backfill",
    }
    op_label = discord_t(_OP_KEYS.get(operation, operation))
    duration = _format_duration(started_at, finished_at)

    color, status_icon = _resolve_status_color(players, success)

    t_start = _to_local(started_at).strftime("%H:%M:%S")
    t_end = _to_local(finished_at).strftime("%H:%M:%S")
    total_new = sum(p.matches_synced for p in players)
    matches_str = _format_matches_summary(operation, total_new)

    completed_line = discord_t(
        "discord_completed_in",
        status=status_icon,
        op=op_label,
        duration=duration,
    )
    time_line = discord_t("discord_time_range", t_start=t_start, t_end=t_end)
    players_line = discord_t("discord_player_count", count=len(players)) + "  ·  " + matches_str
    description = f"{completed_line}\n{time_line}\n{players_line}"

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
        "timestamp": (
            finished_at.replace(tzinfo=timezone.utc).isoformat()
            if finished_at.tzinfo is None
            else finished_at.isoformat()
        ),
    }

    return {"embeds": [embed]}


def _resolve_status_color(
    players: list[DiscordPlayerResult],
    success: bool,
) -> tuple[int, str]:
    """Détermine la couleur de la barre latérale et l'icône de statut."""
    has_errors = any(p.error for p in players)
    has_missing = any(p.missing_data_count > 0 for p in players)
    if not success or has_errors:
        return 0xED4245, "❌"  # Rouge Discord
    if has_missing:
        return 0xFEE75C, "⚠️"  # Jaune Discord
    return 0x57F287, "✅"  # Vert Discord


def _format_matches_summary(operation: str, total_new: int) -> str:
    """Construit le résumé des matchs pour la description de l'embed."""
    from src.ui.i18n.cli import discord_t

    if operation.startswith("sync"):
        if total_new > 0:
            return discord_t("discord_matches_synced", count=total_new)
        return discord_t("discord_no_new_matches")
    if total_new > 0:
        return discord_t("discord_matches_processed", count=total_new)
    return discord_t("discord_all_up_to_date")
