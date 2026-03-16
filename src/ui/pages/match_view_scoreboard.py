"""Tableau de scores par équipe pour la page Match View.

Contient les helpers de formatage du scoreboard et la fonction
de rendu ``render_match_scoreboard``.
"""

from __future__ import annotations

import html
import logging
from typing import Any

import streamlit as st

from src.config import TEAM_MAP, get_bot_name
from src.ui.i18n import get_lang, t
from src.ui.pages.match_table_html import gamertag_link
from src.ui.pages.match_view_players_data import load_match_scoreboard
from src.ui.pages.match_view_scoreboard_detail import (
    SCOREBOARD_JS,
    build_team_table_html_with_details,
    preload_match_extra_data,
)
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG  # noqa: F401 — re-export éventuel
from src.utils import parse_xuid_input

logger = logging.getLogger(__name__)


# =============================================================================
# Helpers scoreboard
# =============================================================================


def _get_scoreboard_cols() -> list[tuple[str, str]]:
    """Retourne les colonnes du scoreboard traduites."""
    return [
        (t("col_player"), "gamertag"),
        (t("col_rank"), "rank"),
        (t("col_score"), "score"),
        (t("col_kills"), "kills"),
        (t("col_deaths"), "deaths"),
        (t("col_assists_short"), "assists"),
        (t("col_kda"), "kda"),
        (t("col_weapon_of_destruction"), "top_weapon_id"),
        (t("col_killing_spree"), "max_killing_spree"),
        (t("col_headshots"), "headshot_kills"),
        (t("col_perfect_kills"), "perfect_kills"),
        (t("col_shots_fired_short"), "shots_fired"),
        (t("col_shots_hit"), "shots_hit"),
        (t("col_accuracy"), "accuracy"),
        (t("col_melee"), "melee_kills"),
        (t("col_power_weapon"), "power_weapon_kills"),
        (t("col_dmg_dealt"), "damage_dealt"),
        (t("col_dmg_taken"), "damage_taken"),
        (t("mv_scoreboard_avg_life"), "avg_life_seconds"),
    ]


# Colonnes non comparables (texte / ordinal) : pas de highlight min/max
_SB_SKIP_HIGHLIGHT: set[str] = {"gamertag", "rank", "top_weapon_id"}

# Colonnes inversées : moins = mieux (vert), plus = pire (rouge)
_SB_INVERTED: set[str] = {"deaths", "damage_taken"}


def _sb_numeric_value(key: str, value: Any) -> float | None:
    """Extrait la valeur numérique comparable d'une cellule du scoreboard."""
    if value is None:
        return None
    try:
        v = float(value)
        if key == "accuracy" and v <= 1.0:
            v *= 100.0
        return v
    except (ValueError, TypeError):
        return None


def _compute_scoreboard_extremes(
    players: list[dict[str, Any]],
) -> dict[str, tuple[float, float]]:
    """Calcule le min et max de chaque colonne numérique.

    Returns:
        Dict ``{key: (min_val, max_val)}``.
    """
    extremes: dict[str, tuple[float, float]] = {}
    for _, key in _get_scoreboard_cols():
        if key in _SB_SKIP_HIGHLIGHT:
            continue
        vals = [v for p in players if (v := _sb_numeric_value(key, p.get(key))) is not None]
        if len(vals) < 2:
            continue
        mn, mx = min(vals), max(vals)
        if mn == mx:
            continue
        extremes[key] = (mn, mx)
    return extremes


def _sb_cell_class(
    key: str,
    value: Any,
    extremes: dict[str, tuple[float, float]],
) -> str:
    """Retourne la classe CSS de highlight pour une cellule."""
    if key in _SB_SKIP_HIGHLIGHT or key not in extremes:
        return ""
    v = _sb_numeric_value(key, value)
    if v is None:
        return ""
    mn, mx = extremes[key]
    inverted = key in _SB_INVERTED
    if v == mx:
        return " os-sb-td--worst" if inverted else " os-sb-td--best"
    if v == mn:
        return " os-sb-td--best" if inverted else " os-sb-td--worst"
    return ""


def _fmt_scoreboard_cell(key: str, value: Any) -> str:
    """Formate une valeur pour l'affichage dans le scoreboard."""
    if key == "top_weapon_id":
        from src.analysis._weapon_data import resolve_weapon_display

        if value is None:
            return "-"
        name = resolve_weapon_display(int(value))
        if name is None or name.startswith("weapon_"):
            return "-"
        return name
    if value is None:
        return "—"
    if key == "gamertag":
        s = str(value).strip()
        if not s or s.isdigit() or s.lower().startswith("xuid(") or s == "?":
            return "—"
        return s
    if key == "kda":
        return f"{float(value):.2f}"
    if key == "accuracy":
        v = float(value)
        if v <= 1.0:
            v *= 100.0
        return f"{v:.1f}\u202f%"
    if key == "avg_life_seconds":
        secs = int(float(value))
        return f"{secs // 60}:{secs % 60:02d}"
    if key in ("damage_dealt", "damage_taken"):
        return str(int(round(float(value))))
    return str(value)


# =============================================================================
# Résolution gamertags du scoreboard
# =============================================================================


def _resolve_scoreboard_gamertags(
    players: list[dict[str, Any]],
    gt_map: dict[str, str],
) -> None:
    """Résout les gamertags manquants dans la liste de joueurs (in-place)."""
    for p in players:
        xu = str(
            parse_xuid_input(str(p.get("xuid") or "").strip()) or str(p.get("xuid") or "").strip()
        ).strip()

        # Priorité 1 : bots
        if xu.lower().startswith("bid("):
            bot_name = get_bot_name(xu)
            if isinstance(bot_name, str) and bot_name.strip():
                p["gamertag"] = bot_name.strip()
                continue

        # Priorité 2 : gt_map (source fraîche DuckDB v5)
        if xu and isinstance(gt_map, dict):
            mapped = gt_map.get(xu)
            if isinstance(mapped, str) and mapped.strip():
                p["gamertag"] = mapped.strip()
                continue

        # Priorité 3 : gamertag déjà présent
        gt = str(p.get("gamertag") or "").strip()
        if gt and gt != "?" and not gt.isdigit() and not gt.lower().startswith("xuid("):
            continue

        p["gamertag"] = None


# =============================================================================
# Calcul MVP / LVP
# =============================================================================


def _compute_mvp_lvp(
    players: list[dict[str, Any]],
    extremes: dict[str, tuple[float, float]],
) -> tuple[str, str]:
    """Identifie les xuids MVP et LVP (bots exclus).

    Returns:
        Tuple (mvp_xuid, lvp_xuid). Chaînes vides si pas de distinction.
    """
    # Filtrer les bots pour ne garder que les humains
    human_indices = [
        i for i, p in enumerate(players) if get_bot_name(str(p.get("xuid", ""))) is None
    ]
    if not human_indices:
        return "", ""

    player_best: dict[int, int] = {}
    player_worst: dict[int, int] = {}
    for i in human_indices:
        p = players[i]
        best = worst = 0
        for _, key in _get_scoreboard_cols():
            cls = _sb_cell_class(key, p.get(key), extremes)
            if "best" in cls:
                best += 1
            elif "worst" in cls:
                worst += 1
        player_best[i] = best
        player_worst[i] = worst

    mvp_idx = (
        max(
            player_best,
            key=lambda i: (
                player_best[i],
                -player_worst.get(i, 0),
                -(players[i].get("rank") or 999),
            ),
        )
        if player_best
        else -1
    )
    lvp_idx = (
        max(
            player_worst,
            key=lambda i: (
                player_worst[i],
                -player_best.get(i, 0),
                (players[i].get("rank") or 0),
            ),
        )
        if player_worst
        else -1
    )
    if player_best.get(mvp_idx, 0) < 2:
        mvp_idx = -1
    if player_worst.get(lvp_idx, 0) < 2:
        lvp_idx = -1

    mvp_xuid = str(players[mvp_idx].get("xuid", "")).strip() if mvp_idx >= 0 else ""
    lvp_xuid = str(players[lvp_idx].get("xuid", "")).strip() if lvp_idx >= 0 else ""
    return mvp_xuid, lvp_xuid


# =============================================================================
# Fonction principale
# =============================================================================


def render_match_scoreboard(
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    load_match_gamertags_fn: Any,
) -> None:
    """Rend les tableaux de scores par équipe pour un match.

    Chaque ligne joueur est cliquable (▶/▼) pour révéler les stats détaillées,
    médailles et armes inline dans le tableau — sans section séparée.

    Args:
        match_id: Identifiant du match.
        db_path: Chemin vers la base de données.
        xuid: XUID du joueur principal.
        db_key: Clé de cache.
        load_match_gamertags_fn: Fonction de résolution gamertags injectée.
    """
    st.subheader(t("mv_scoreboard"))

    players = load_match_scoreboard(db_path, match_id.strip())
    if not players:
        st.info(t("mv_scoreboard_no_data"))
        return

    me_xu = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    gt_map: dict[str, str] = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key) or {}
    logger.debug("Résolution gamertags scoreboard match=%s: %d entrées", match_id, len(gt_map))

    _resolve_scoreboard_gamertags(players, gt_map)

    # Équipe du joueur principal
    my_team_id: Any = None
    for p in players:
        p_xu = str(
            parse_xuid_input(str(p.get("xuid") or "").strip()) or str(p.get("xuid") or "").strip()
        ).strip()
        if me_xu and p_xu and p_xu == me_xu:
            my_team_id = p.get("team_id")
            break

    # Grouper par team_id
    teams: dict[Any, list[dict[str, Any]]] = {}
    unknown_team: list[dict[str, Any]] = []
    for p in players:
        tid = p.get("team_id")
        if tid is None:
            unknown_team.append(p)
        else:
            teams.setdefault(tid, []).append(p)
    if unknown_team:
        teams[None] = unknown_team

    n_real_teams = len([t_ for t_ in teams if t_ is not None])
    extremes = _compute_scoreboard_extremes(players)
    mvp_xuid, lvp_xuid = _compute_mvp_lvp(players, extremes)

    # Pré-chargement des données extra (médailles, armes, enrichissement)
    lang = get_lang()
    mid_key = (match_id.strip().replace("-", ""))[:12]
    extra_data = preload_match_extra_data(
        main_db_path=db_path,
        match_id=match_id.strip(),
        players=players,
        lang=lang,
    )

    sb_cols = _get_scoreboard_cols()

    # Rendu combiné : JS unique + tableaux par équipe avec détails inline
    html_parts: list[str] = [SCOREBOARD_JS]
    for t_idx, (tid, team_players) in enumerate(teams.items()):
        team_html = build_team_table_html_with_details(
            team_players=team_players,
            tid=tid,
            my_team_id=my_team_id,
            me_xu=me_xu,
            mvp_xuid=mvp_xuid,
            lvp_xuid=lvp_xuid,
            extremes=extremes,
            sb_cols=sb_cols,
            extra_data=extra_data,
            match_id_key=mid_key,
            team_index=t_idx,
            lang=lang,
            fmt_cell_fn=_fmt_scoreboard_cell,
            cell_class_fn=_sb_cell_class,
            gamertag_link_fn=gamertag_link,
        )
        html_parts.append(team_html)

    st.markdown("\n".join(html_parts), unsafe_allow_html=True)

    if n_real_teams > 1:
        st.caption(t("mv_scoreboard_rank_note"))
