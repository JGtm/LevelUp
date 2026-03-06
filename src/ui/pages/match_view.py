"""Page Match View - Affichage détaillé d'un match.

Ce module a été refactorisé en sous-modules :
- match_view_helpers.py : Utilitaires date/heure, médias, composants UI
- match_view_charts.py : Graphiques Expected vs Actual
- match_view_players.py : Sections Némésis et Roster
"""

from __future__ import annotations

import contextlib
import html
import logging
from collections.abc import Callable
from datetime import datetime
from typing import Any

import polars as pl
import streamlit as st

from src.analysis.performance_config import SCORE_THRESHOLDS
from src.analysis.performance_score import compute_relative_performance_score
from src.app.helpers import normalize_map_label
from src.config import HALO_COLORS, OUTCOME_CODES
from src.ui import (
    AppSettings,
    translate_pair_name,
    translate_playlist_name,
)
from src.ui.formatting import format_date_fr
from src.ui.i18n import get_lang, get_outcome_map, t
from src.ui.medals import load_medal_name_maps, render_medals_grid
from src.ui.pages.match_view_charts import render_expected_vs_actual
from src.ui.pages.match_view_citations import (
    render_match_citations_section as _render_match_citations_section,
)
from src.ui.pages.match_view_encounters import render_encounter_section

# Imports depuis les sous-modules
from src.ui.pages.match_view_helpers import (
    map_thumb_path,
    os_card,
    render_media_section,
)
from src.ui.pages.match_view_participation import render_participation_section
from src.ui.pages.match_view_players import (
    render_kd_timeline_section,
    render_match_impact_section,
    render_match_scoreboard,
    render_nemesis_section,
    render_team_dominance_section,
)
from src.ui.pages.match_view_rank import _build_match_rank_html
from src.visualization._compat import DataFrameLike, ensure_polars

logger = logging.getLogger(__name__)


# =============================================================================
# Fonction principale
# =============================================================================


def render_match_view(  # noqa: C901, PLR0912, PLR0913, PLR0915
    *,
    row: dict[str, Any],
    match_id: str,
    db_path: str,
    xuid: str,
    waypoint_player: str,
    db_key: tuple[int, int] | None,
    settings: AppSettings,
    df_full: DataFrameLike | None = None,
    # Fonctions injectées
    normalize_mode_label_fn: Callable[[str | None], str],
    format_score_label_fn: Callable[[Any, Any], str],
    score_css_color_fn: Callable[[Any, Any], str],
    format_datetime_fn: Callable[[datetime | None], str],
    load_player_match_result_fn: Callable,
    load_match_medals_fn: Callable,
    load_highlight_events_fn: Callable,
    load_match_gamertags_fn: Callable,
    load_match_rosters_fn: Callable,
    paris_tz,
) -> None:
    """Rend la vue détaillée d'un match.

    Parameters
    ----------
    row : dict[str, Any]
        Données du match (dict issu de iter_rows(named=True) ou to_dicts()).
    match_id : str
        Identifiant du match.
    db_path : str
        Chemin vers la base de données.
    xuid : str
        XUID du joueur principal.
    waypoint_player : str
        Gamertag pour les liens Waypoint.
    db_key : tuple[int, int] | None
        Clé de cache pour la base de données.
    settings : AppSettings
        Paramètres de l'application.
    df_full : DataFrameLike | None
        DataFrame complet pour le calcul du score relatif.
    normalize_mode_label_fn, format_score_label_fn, score_css_color_fn, format_datetime_fn
        Fonctions de formatage injectées.
    load_player_match_result_fn, load_match_medals_fn, load_highlight_events_fn,
    load_match_gamertags_fn, load_match_rosters_fn
        Fonctions de chargement de données injectées.
    paris_tz
        Timezone Paris.
    """
    # Normaliser df_full en Polars
    if df_full is not None:
        df_full = ensure_polars(df_full)

    match_id = str(match_id or "").strip()
    if not match_id:
        st.info(t("mv_match_id_missing"))
        return

    logger.debug("render_match_view match=%s xuid=%s", match_id, xuid)

    last_time = row.get("start_time")
    last_map = row.get("map_name")
    last_playlist = row.get("playlist_name")
    last_pair = row.get("pair_name")
    last_mode = row.get("game_variant_name")
    last_outcome = row.get("outcome")

    last_playlist_fr = (
        translate_playlist_name(str(last_playlist), lang=get_lang()) if last_playlist else None
    )
    last_pair_fr = translate_pair_name(str(last_pair), lang=get_lang()) if last_pair else None

    outcome_map = get_outcome_map()
    try:
        outcome_code = int(last_outcome) if last_outcome is not None else None
    except (TypeError, ValueError):
        outcome_code = None
    outcome_label = outcome_map.get(outcome_code, "?") if outcome_code is not None else "-"

    colors = HALO_COLORS.as_dict()
    if outcome_code == OUTCOME_CODES.WIN:
        outcome_color = colors["green"]
    elif outcome_code == OUTCOME_CODES.LOSS:
        outcome_color = colors["red"]
    elif outcome_code == OUTCOME_CODES.TIE or outcome_code == OUTCOME_CODES.NO_FINISH:
        outcome_color = colors["violet"]
    else:
        outcome_color = colors["slate"]

    last_my_score = row.get("my_team_score")
    last_enemy_score = row.get("enemy_team_score")
    score_label = format_score_label_fn(last_my_score, last_enemy_score)

    wp = str(waypoint_player or "").strip()
    match_url = None
    if wp and match_id and match_id.strip() and match_id.strip() != "-":
        match_url = (
            f"https://www.halowaypoint.com/halo-infinite/players/{wp}/matches/{match_id.strip()}"
        )

    # Détecter le flag had_bot_teammate depuis player_match_enrichment
    # P4: aussi lire performance_score stocké pour éviter le recalcul
    _had_bot_teammate = False
    _stored_perf_score: float | None = None
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as _pme_conn:
            _bot_row = _pme_conn.execute(
                "SELECT had_bot_teammate, performance_score"
                " FROM player_match_enrichment WHERE match_id = ? LIMIT 1",
                [match_id],
            ).fetchone()
        if _bot_row:
            if _bot_row[0]:
                _had_bot_teammate = True
            if _bot_row[1] is not None:
                _stored_perf_score = float(_bot_row[1])
    except Exception:
        logger.debug("match_view: enrichment introuvable match=%s", match_id)

    # Bot + outcome : leniency uniquement sur défaite (rage quit en cours de partie)
    _bot_is_loss = _had_bot_teammate and outcome_code != OUTCOME_CODES.WIN
    _bot_is_win = _had_bot_teammate and outcome_code == OUTCOME_CODES.WIN

    # P4: Utiliser le performance_score déjà stocké en DB (évite le recalcul à l'affichage)
    # Fallback : calcul relatif si non stocké et historique suffisant (≥ 10 matchs)
    perf_score: float | None = _stored_perf_score
    if perf_score is None and df_full is not None and len(df_full) >= 10:
        perf_score = compute_relative_performance_score(
            row, df_full, had_bot_teammate=_had_bot_teammate
        )
    perf_display = f"{perf_score:.0f}" if perf_score is not None else "-"
    perf_color = None
    if perf_score is not None:
        if perf_score >= SCORE_THRESHOLDS["excellent"]:
            perf_color = colors["green"]
        elif perf_score >= SCORE_THRESHOLDS["good"]:
            perf_color = colors["cyan"]
        elif perf_score >= SCORE_THRESHOLDS["average"]:
            perf_color = colors["amber"]
        elif perf_score >= SCORE_THRESHOLDS["below_average"]:
            perf_color = colors.get("orange", "#FF8C00")
        else:
            perf_color = colors["red"]

    # Cartes KPI - Date, Résultat, Performance
    top_cols = st.columns(3)
    with top_cols[0]:
        os_card(t("col_date"), format_date_fr(last_time, lang=get_lang()))
    with top_cols[1]:
        outcome_class = (
            "text-win"
            if outcome_code == OUTCOME_CODES.WIN
            else ("text-loss" if outcome_code == OUTCOME_CODES.LOSS else "text-tie")
        )
        os_card(
            t("mv_results"),
            str(outcome_label),
            f"<span class='{outcome_class} fw-bold'>{html.escape(str(score_label))}</span>",
            accent=str(outcome_color),
            kpi_color=str(outcome_color),
        )
    with top_cols[2]:
        if _bot_is_loss:
            _perf_subtitle = t("mv_bot_teammate_note")  # "Équipe incomplète (+5 pts)"
        elif _bot_is_win:
            _perf_subtitle = t("mv_bot_teammate_win_note")  # "Victoire malgré équipe incomplète"
        else:
            _perf_subtitle = (
                t("mv_relative_history") if perf_score is not None else t("mv_insufficient_history")
            )
        os_card(
            t("mv_performance"),
            perf_display,
            _perf_subtitle,
            accent=perf_color,
            kpi_color=perf_color,
        )

    last_mode_ui = row.get("mode_ui") or normalize_mode_label_fn(
        str(last_pair) if last_pair else None
    )

    # Normaliser les labels pour masquer les UUIDs non résolus
    map_display = normalize_map_label(last_map) if last_map else None
    if not map_display:
        map_display = "-"

    playlist_display = (
        last_playlist_fr
        or (translate_playlist_name(str(last_playlist), lang=get_lang()) if last_playlist else None)
        or "-"
    )
    mode_display = (
        last_mode_ui
        or last_pair_fr
        or (normalize_mode_label_fn(str(last_pair)) if last_pair else None)
        or last_mode
        or "-"
    )

    row_cols = st.columns(3)
    row_cols[0].metric(" ", map_display)
    row_cols[1].metric(" ", playlist_display)
    row_cols[2].metric(" ", mode_display)

    # Miniature de la carte + Rang (LUSR/CSR) côte à côte
    map_id = row.get("map_id")
    thumb = map_thumb_path(row, str(map_id) if map_id else None)

    # Convertir la miniature en data URI pour l'intégrer au HTML
    from src.ui.player_assets import file_to_data_url as _f2du

    thumb_data_url = None
    if thumb:
        with contextlib.suppress(Exception):
            thumb_data_url = _f2du(str(thumb))

    # Construire le HTML du rang (avec contexte bot si applicable)
    rank_html = _build_match_rank_html(
        match_id=match_id,
        db_path=db_path,
        db_key=db_key,
        had_bot_teammate=_had_bot_teammate,
        bot_outcome=outcome_code,
    )

    # Assembler carte + rang dans un seul bloc HTML centré verticalement
    map_img_html = ""
    if thumb_data_url:
        map_img_html = (
            f"<img src='{thumb_data_url}' "
            f"style='width:100%;max-width:480px;height:auto;border-radius:4px;object-fit:cover'>"
        )
    else:
        map_img_html = (
            "<div style='padding:16px;color:#888;font-style:italic'>"
            f"{t('mv_thumbnail_unavailable')}</div>"
        )

    if rank_html:
        st.markdown(
            f"""<div style='display:flex;align-items:center;gap:24px;flex-wrap:wrap;margin-bottom:1.5rem'>
  <div style='flex:1;min-width:250px'>{map_img_html}</div>
  <div style='flex:1;min-width:250px'>{rank_html}</div>
</div>""",
            unsafe_allow_html=True,
        )
    else:
        st.markdown(map_img_html, unsafe_allow_html=True)

    # Chargement des données détaillées (commun à tous les onglets)
    with st.spinner(t("mv_loading")):
        pm = load_player_match_result_fn(db_path, match_id, xuid.strip(), db_key=db_key)
        medals_last = load_match_medals_fn(db_path, match_id, xuid.strip(), db_key=db_key)

    # Enrichir pm avec les valeurs réelles depuis row si elles sont manquantes (DuckDB v4)
    if pm:
        if pm.get("kills", {}).get("count") is None:
            kills_val = row.get("kills")
            if kills_val is not None:
                pm.setdefault("kills", {})["count"] = (
                    float(kills_val) if kills_val == kills_val else None
                )
        if pm.get("deaths", {}).get("count") is None:
            deaths_val = row.get("deaths")
            if deaths_val is not None:
                pm.setdefault("deaths", {})["count"] = (
                    float(deaths_val) if deaths_val == deaths_val else None
                )
        if pm.get("assists", {}).get("count") is None:
            assists_val = row.get("assists")
            if assists_val is not None:
                pm.setdefault("assists", {})["count"] = (
                    float(assists_val) if assists_val == assists_val else None
                )

    # Onglets de navigation (P2 — lazy rendering par section)
    (
        _tab_summary,
        _tab_combat,
        _tab_team,
        _tab_cit,
        _tab_media,
    ) = st.tabs(
        [
            t("mv_tab_summary"),
            t("mv_tab_combat"),
            t("mv_tab_team"),
            t("mv_tab_citations_medals"),
            t("mv_tab_media"),
        ]
    )

    # ── Onglet Résumé ────────────────────────────────────────────────────────
    with _tab_summary:
        if not pm:
            st.info(t("mv_stats_unavailable"))
        else:
            render_expected_vs_actual(row, pm, colors, df_full=df_full, db_path=db_path, xuid=xuid)

        # Section Participation (PersonalScores) - Radar unifié 6 axes
        render_participation_section(
            db_path=db_path,
            match_id=match_id,
            xuid=xuid,
            db_key=db_key,
            match_row=row,
        )

    # ── Onglet Combat ────────────────────────────────────────────────────────
    with _tab_combat:
        # Impact & Timeline (kills/deaths cumulées + badges)
        render_match_impact_section(
            match_id=match_id,
            db_path=db_path,
            xuid=xuid,
            db_key=db_key,
            outcome=outcome_code,
            load_highlight_events_fn=load_highlight_events_fn,
            load_match_gamertags_fn=load_match_gamertags_fn,
        )

        # Dynamique du match (frise de dominance — PvP uniquement)
        render_team_dominance_section(
            match_id=match_id,
            db_path=db_path,
            xuid=xuid,
            db_key=db_key,
            is_firefight=bool(row.get("is_firefight")),
            load_highlight_events_fn=load_highlight_events_fn,
        )

        # Némésis / Souffre-douleur
        render_nemesis_section(
            match_id=match_id,
            db_path=db_path,
            xuid=xuid,
            db_key=db_key,
            colors=colors,
            load_highlight_events_fn=load_highlight_events_fn,
            load_match_gamertags_fn=load_match_gamertags_fn,
        )

        # Évolution K/D de tous les joueurs
        render_kd_timeline_section(
            match_id=match_id,
            db_path=db_path,
            xuid=xuid,
            db_key=db_key,
            load_highlight_events_fn=load_highlight_events_fn,
            load_match_gamertags_fn=load_match_gamertags_fn,
        )

    # ── Onglet Équipe ────────────────────────────────────────────────────────
    with _tab_team:
        # Tableau des scores par équipe
        render_match_scoreboard(
            match_id=match_id,
            db_path=db_path,
            xuid=xuid,
            db_key=db_key,
            load_match_gamertags_fn=load_match_gamertags_fn,
        )

        # Historique des rencontres (v5.4)
        render_encounter_section(
            match_id=match_id,
            self_xuid=xuid,
            db_path=db_path,
        )

    # ── Onglet Citations & Médailles ─────────────────────────────────────────
    with _tab_cit:
        # Citations (progressées dans ce match)
        st.subheader(t("mv_citations"))
        _render_match_citations_section(
            match_id=match_id,
            db_path=db_path,
            xuid=xuid.strip(),
        )

        # Médailles
        st.subheader(t("mv_medals"))
        if not medals_last:
            st.info(t("mv_medals_no_data"))
        else:
            md_df = pl.DataFrame(medals_last)
            _fr_map, _en_map = load_medal_name_maps()
            _medal_map = {
                **{str(k): v for k, v in _en_map.items()},
                **{str(k): v for k, v in _fr_map.items()},
            }
            md_df = md_df.with_columns(
                pl.col("name_id")
                .cast(pl.Utf8)
                .replace_strict(_medal_map, default=None, return_dtype=pl.Utf8)
                .fill_null(
                    pl.lit(t("mv_medal_fallback", n="") + " ") + pl.col("name_id").cast(pl.Utf8)
                )
                .alias("label")
            )
            md_df = md_df.sort(["count", "label"], descending=[True, False])
            render_medals_grid(
                md_df.select(["name_id", "count"]).to_dicts(),
                cols_per_row=8,
                center=True,
                lang=get_lang(),
            )

    # ── Onglet Médias ────────────────────────────────────────────────────────
    with _tab_media:
        render_media_section(
            row=row,
            settings=settings,
            format_datetime_fn=format_datetime_fn,
            paris_tz=paris_tz,
            gamertag=waypoint_player,
            db_path=db_path,
            current_xuid=xuid,
        )

        # Lien Waypoint
        if match_url:
            st.write("")
            st.link_button(t("mv_open_waypoint"), match_url, width="stretch")


# =============================================================================

# Réexporter les fonctions helpers pour rétrocompatibilité
from src.ui.pages.match_view_charts import (
    render_expected_vs_actual as _render_expected_vs_actual,
)
from src.ui.pages.match_view_helpers import (
    index_media_dir as _index_media_dir,
)
from src.ui.pages.match_view_helpers import (
    map_thumb_path as _map_thumb_path,
)
from src.ui.pages.match_view_helpers import (
    match_time_window as _match_time_window,
)
from src.ui.pages.match_view_helpers import (
    os_card as _os_card,
)
from src.ui.pages.match_view_helpers import (
    paris_epoch_seconds_local as _paris_epoch_seconds_local,
)
from src.ui.pages.match_view_helpers import (
    render_media_section as _render_media_section,
)
from src.ui.pages.match_view_helpers import (
    safe_dt as _safe_dt,
)
from src.ui.pages.match_view_helpers import (
    to_paris_naive_local as _to_paris_naive_local,
)
from src.ui.pages.match_view_players import (
    render_match_impact_section as _render_match_impact_section,
)
from src.ui.pages.match_view_players import (
    render_match_scoreboard as _render_match_scoreboard,
)
from src.ui.pages.match_view_players import (
    render_nemesis_section as _render_nemesis_section,
)
from src.ui.pages.match_view_players import (
    render_team_dominance_section as _render_team_dominance_section,
)

__all__ = [
    "render_match_view",
    # Helpers (rétrocompatibilité)
    "_to_paris_naive_local",
    "_safe_dt",
    "_match_time_window",
    "_paris_epoch_seconds_local",
    "_index_media_dir",
    "_render_media_section",
    "_os_card",
    "_map_thumb_path",
    "_render_expected_vs_actual",
    "_render_match_impact_section",
    "_render_nemesis_section",
    "_render_team_dominance_section",
    "_render_match_scoreboard",
]
