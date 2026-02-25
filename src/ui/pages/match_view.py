"""Page Match View - Affichage détaillé d'un match.

Ce module a été refactorisé en sous-modules :
- match_view_helpers.py : Utilitaires date/heure, médias, composants UI
- match_view_charts.py : Graphiques Expected vs Actual
- match_view_players.py : Sections Némésis et Roster
"""

from __future__ import annotations

import html
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
from src.ui.medals import load_medal_name_maps, render_medals_grid
from src.ui.pages.match_view_charts import render_expected_vs_actual

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
from src.visualization._compat import DataFrameLike, ensure_polars

# =============================================================================
# Section Rang LUSR / CSR
# =============================================================================


def _render_match_rank_tab(*, match_id: str, db_path: str) -> None:
    """Affiche l'onglet 🏅 Rang pour un match (LUSR ou CSR).

    Charge depuis ``match_skill_rank`` (stats.duckdb du joueur) et affiche :
    - Image du rang (120px-HINF-CSR_*.png)
    - Label du tier (ex: « Or III », « Onyx »)
    - Barre de progression dans le sous-tier avec delta coloré

    Args:
        match_id: Identifiant du match.
        db_path: Chemin vers stats.duckdb du joueur.
    """
    try:
        import duckdb
        from pathlib import Path

        from src.analysis.skill_rating_config import (
            UNRANKED_COLOR,
            UNRANKED_LABEL,
            get_rank_image_path,
            get_sub_tier_start,
            get_tier_for_rating,
            get_tier_size,
        )
        from src.data.sync.migrations import ensure_match_skill_rank_table
    except ImportError:
        st.info("Modules LUSR non disponibles.")
        return

    try:
        conn = duckdb.connect(str(db_path), read_only=True)
        ensure_match_skill_rank_table(conn)
        row_rank = conn.execute(
            "SELECT rating_type, rating_value, rating_deviation, tier_label, "
            "       sub_tier, tier, tier_fr, rating_delta, playlist_group "
            "FROM match_skill_rank WHERE match_id = ?",
            [match_id],
        ).fetchone()
        conn.close()
    except Exception:
        st.info("Aucun rating disponible pour ce match.")
        return

    if row_rank is None:
        st.info(
            "Aucun rating LUSR/CSR calculé pour ce match. "
            "Lancez `--lusr` (non classé) ou `--csr` (classé) pour calculer."
        )
        return

    (
        rating_type, rating_value, rating_deviation,
        tier_label, sub_tier, tier_name, tier_fr, rating_delta, playlist_group,
    ) = row_rank

    # ── Affichage centré ──
    col_img, col_info = st.columns([1, 2])

    with col_img:
        # Image du rang
        img_path_rel = get_rank_image_path(rating_value) if rating_value else None
        if img_path_rel:
            img_full = Path(__file__).resolve().parents[3] / img_path_rel
            if img_full.exists():
                st.image(str(img_full), width=120)
            else:
                st.markdown(f"<span style='color:{UNRANKED_COLOR}'>◆</span>", unsafe_allow_html=True)

    with col_info:
        # Label du tier
        tier_display = tier_label or UNRANKED_LABEL
        rating_type_badge = "CSR" if rating_type == "CSR" else "LUSR"
        st.markdown(
            f"<h3 style='margin:0'>{tier_display}</h3>"
            f"<small style='color:#888'>{rating_type_badge} • {rating_value:.0f}</small>",
            unsafe_allow_html=True,
        )

        # Delta
        if rating_delta is not None:
            delta_color = "#50C878" if rating_delta >= 0 else "#FF4444"
            delta_sign = "+" if rating_delta >= 0 else ""
            st.markdown(
                f"<span style='color:{delta_color};font-size:1.1em'>"
                f"{delta_sign}{rating_delta:.0f} pts</span>",
                unsafe_allow_html=True,
            )

        # Barre de progression dans le sous-tier
        if rating_value is not None:
            tier_obj, sub = get_tier_for_rating(rating_value)
            if tier_obj and tier_obj.sub_tiers > 1:
                tier_size = get_tier_size(rating_value)
                sub_start = get_sub_tier_start(rating_value)
                progress_val = min(1.0, max(0.0, (rating_value - sub_start) / tier_size))
                st.progress(progress_val)
                st.caption(
                    f"{rating_value - sub_start:.0f} / {tier_size:.0f} pts "
                    f"dans {tier_fr or tier_name or ''} {sub}"
                )

        if playlist_group:
            st.caption(f"Groupe : {playlist_group.capitalize()}")


# =============================================================================
# Section Citations (progressées dans un match)
# =============================================================================


def _render_match_citations_section(
    *,
    match_id: str,
    db_path: str,
    xuid: str,
) -> None:
    """Affiche les citations qui ont progressé dans ce match, avec compteur delta."""
    import os as _os

    from src.analysis.citations.engine import CitationEngine
    from src.ui.commendations import (
        _compute_mastery_display,
        _img_data_uri,
        _img_src,
        _load_citations_from_db,
        _parse_tier_targets,
    )

    citations_db = _load_citations_from_db()
    if not citations_db:
        st.caption("Référentiel Citations indisponible.")
        return

    try:
        engine = CitationEngine(db_path, xuid)
        delta_map = engine.aggregate_for_display(match_ids=[match_id])
    except Exception:
        st.caption("Citations indisponibles pour ce match.")
        return

    active_norms = {norm for norm, val in delta_map.items() if val > 0}
    if not active_norms:
        st.info("Aucune citation n'a progressé dans ce match.")
        return

    try:
        full_map = engine.aggregate_for_display(match_ids=None)
    except Exception:
        full_map = {}

    items = []
    for cit in citations_db:
        norm = cit["citation_name_norm"]
        if norm not in active_norms:
            continue
        tiers = _parse_tier_targets(cit.get("tier_targets"))
        current_full = full_map.get(norm, 0)
        delta = delta_map.get(norm, 0)
        _, _, is_master, _ = _compute_mastery_display(current_full, tiers)
        # Si on est maître ACTUELLEMENT, vérifier si on le was AVANT ce match.
        # Si on ne l'était pas avant (count_before < seuil maître), c'est que
        # ce match a fait atteindre le niveau maître → on l'affiche quand même.
        if is_master:
            count_before = current_full - delta
            _, _, was_master_before, _ = _compute_mastery_display(count_before, tiers)
            if was_master_before:
                continue  # Déjà maître avant ce match → on ne l'affiche pas
        items.append(
            {
                "name": cit["citation_name_display"],
                "norm": norm,
                "description": cit.get("description") or "",
                "image_path": cit.get("image_path"),
                "tiers": tiers,
                "current_full": current_full,
                "delta": delta,
            }
        )

    if not items:
        st.info("Aucune citation n'a progressé dans ce match.")
        return

    # Grille centrée : padding symétrique si moins de 8 éléments
    n_cols = min(len(items), 8)
    if n_cols < 8:
        _pad = max(1, (8 - n_cols) // 2)
        _all_cols = st.columns([_pad] + [1] * n_cols + [_pad])
        display_cols = _all_cols[1 : 1 + n_cols]
    else:
        display_cols = st.columns(8)

    for i, item in enumerate(items):
        col = display_cols[i % n_cols]
        name = item["name"]
        tiers = item["tiers"]
        current_full = item["current_full"]
        delta = item["delta"]

        level_label, counter_label, is_master, progress_ratio = _compute_mastery_display(
            current_full, tiers
        )

        img = _img_src(item["image_path"])
        data_uri = None
        if img:
            try:
                mtime = _os.path.getmtime(img)
            except OSError:
                mtime = None
            data_uri = _img_data_uri(img, mtime)

        desc = item["description"]
        tip = html.escape(desc) if desc else html.escape(name)

        with col:
            st.markdown("<div class='os-citation-top-gap'></div>", unsafe_allow_html=True)

            if data_uri:
                ring_class = (
                    "os-citation-ring os-citation-ring--master" if is_master else "os-citation-ring"
                )
                ring_color = "#d6b35a" if is_master else "#41d6ff"
                st.markdown(
                    "<div class='"
                    + ring_class
                    + "' title='"
                    + tip
                    + "' style=\"--p:"
                    + str(float(progress_ratio))
                    + ";--ring-color:"
                    + ring_color
                    + ";--img:url('"
                    + data_uri
                    + "')\"></div>",
                    unsafe_allow_html=True,
                )
            else:
                st.markdown(
                    "<div class='os-medal-missing' title='" + tip + "'>?</div>",
                    unsafe_allow_html=True,
                )

            st.markdown(
                "<div class='os-citation-name' title='" + tip + "'>" + html.escape(name) + "</div>",
                unsafe_allow_html=True,
            )
            level_class = (
                "os-citation-level os-citation-level--master" if is_master else "os-citation-level"
            )
            st.markdown(
                f"<div class='{level_class}'>{html.escape(level_label)}</div>",
                unsafe_allow_html=True,
            )

            delta_html = ""
            if delta > 0:
                delta_html = f" <span style='color: #4CAF50; font-weight: bold;'>+{delta}</span>"
            st.markdown(
                "<div class='os-citation-counter'>"
                + html.escape(counter_label)
                + delta_html
                + "</div>",
                unsafe_allow_html=True,
            )


# =============================================================================
# Fonction principale
# =============================================================================


def render_match_view(
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
        st.info("MatchId manquant.")
        return

    last_time = row.get("start_time")
    last_map = row.get("map_name")
    last_playlist = row.get("playlist_name")
    last_pair = row.get("pair_name")
    last_mode = row.get("game_variant_name")
    last_outcome = row.get("outcome")

    last_playlist_fr = translate_playlist_name(str(last_playlist)) if last_playlist else None
    last_pair_fr = translate_pair_name(str(last_pair)) if last_pair else None

    outcome_map = {2: "Victoire", 3: "Défaite", 1: "Égalité", 4: "Non terminé"}
    try:
        outcome_code = int(last_outcome) if last_outcome == last_outcome else None
    except Exception:
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
    _ = score_css_color_fn(last_my_score, last_enemy_score)  # noqa: F841

    wp = str(waypoint_player or "").strip()
    match_url = None
    if wp and match_id and match_id.strip() and match_id.strip() != "-":
        match_url = (
            f"https://www.halowaypoint.com/halo-infinite/players/{wp}/matches/{match_id.strip()}"
        )

    # Calcul du score de performance RELATIF
    perf_score = None
    if df_full is not None and len(df_full) >= 10:
        perf_score = compute_relative_performance_score(row, df_full)
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
        os_card("Date", format_date_fr(last_time))
    with top_cols[1]:
        outcome_class = (
            "text-win"
            if "victoire" in str(outcome_label).lower()
            else ("text-loss" if "défaite" in str(outcome_label).lower() else "text-tie")
        )
        os_card(
            "Résultats",
            str(outcome_label),
            f"<span class='{outcome_class} fw-bold'>{html.escape(str(score_label))}</span>",
            accent=str(outcome_color),
            kpi_color=str(outcome_color),
        )
    with top_cols[2]:
        os_card(
            "Performance",
            perf_display,
            "Relatif à ton historique" if perf_score is not None else "Historique insuffisant",
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
        or (translate_playlist_name(str(last_playlist)) if last_playlist else None)
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

    # Miniature de la carte + onglet Rang (LUSR/CSR)
    map_id = row.get("map_id")
    thumb = map_thumb_path(row, str(map_id) if map_id else None)

    tab_map, tab_rank = st.tabs(["🗺️ Carte", "🏅 Rang"])

    with tab_map:
        if thumb:
            import contextlib

            c = st.columns([1, 2, 1])
            with c[1], contextlib.suppress(Exception):
                st.image(thumb, width=400)
        else:
            st.info("Miniature de carte indisponible.")

    with tab_rank:
        _render_match_rank_tab(match_id=match_id, db_path=db_path)

    # Stats détaillées
    with st.spinner("Lecture des stats détaillées (attendu vs réel, médailles)…"):
        pm = load_player_match_result_fn(db_path, match_id, xuid.strip(), db_key=db_key)
        medals_last = load_match_medals_fn(db_path, match_id, xuid.strip(), db_key=db_key)

    if not pm:
        st.info(
            "Stats détaillées indisponibles pour ce match (PlayerMatchStats manquant ou format inattendu)."
        )
    else:
        # Enrichir pm avec les valeurs réelles depuis row si elles sont manquantes (DuckDB v4)
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

        render_expected_vs_actual(row, pm, colors, df_full=df_full, db_path=db_path, xuid=xuid)

    # Section Participation (PersonalScores) - Radar unifié 6 axes
    render_participation_section(
        db_path=db_path,
        match_id=match_id,
        xuid=xuid,
        db_key=db_key,
        match_row=row,
    )

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

    # Tableau des scores par équipe
    render_match_scoreboard(
        match_id=match_id,
        db_path=db_path,
        xuid=xuid,
        db_key=db_key,
        load_match_gamertags_fn=load_match_gamertags_fn,
    )

    # Citations (progressées dans ce match)
    st.subheader("Citations")
    _render_match_citations_section(
        match_id=match_id,
        db_path=db_path,
        xuid=xuid.strip(),
    )

    # Médailles
    st.subheader("Médailles")
    if not medals_last:
        st.info("Médailles indisponibles pour ce match (ou aucune médaille).")
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
            .fill_null(pl.lit("Médaille #") + pl.col("name_id").cast(pl.Utf8))
            .alias("label")
        )
        md_df = md_df.sort(["count", "label"], descending=[True, False])
        render_medals_grid(
            md_df.select(["name_id", "count"]).to_dicts(), cols_per_row=8, center=True
        )

    # Médias
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
        st.link_button("Ouvrir sur HaloWaypoint", match_url, width="stretch")


# =============================================================================
# Exports publics (rétrocompatibilité)
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
