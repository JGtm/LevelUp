"""Rendu des résultats pour la page Explorer.

Affiche les résultats de recherche (par filtres ou par joueur)
sous forme de tableaux HTML + vue match détaillée.
"""

from __future__ import annotations

import logging
from collections.abc import Callable
from typing import Any

import polars as pl
import streamlit as st

from src.ui.i18n import t
from src.ui.pages.explorer_enrich import enrich_common_matches, enrich_for_table
from src.ui.pages.explorer_logic import split_by_team
from src.ui.pages.match_table_html import render_match_table_html

logger = logging.getLogger(__name__)

_SK_SELECTED_MATCH = "_explorer_selected_match"


# ---------------------------------------------------------------------------
# Résultats par filtres (section 4a)
# ---------------------------------------------------------------------------


def render_filter_results(  # noqa: PLR0913
    filtered: pl.DataFrame,
    selected_match_id: str | None,
    waypoint_player: str,
    df: pl.DataFrame,
    db_path: str,
    xuid: str,
    db_key: Any,
    settings: Any,
    df_full: Any,
    render_match_view_fn: Callable[..., Any],
    normalize_mode_label_fn: Callable[..., Any],
    format_score_label_fn: Callable[..., Any],
    score_css_color_fn: Callable[..., Any],
    format_datetime_fn: Callable[..., Any],
    load_player_match_result_fn: Callable[..., Any],
    load_match_medals_fn: Callable[..., Any],
    load_highlight_events_fn: Callable[..., Any],
    load_match_gamertags_fn: Callable[..., Any],
    load_match_rosters_fn: Callable[..., Any],
    paris_tz: Any,
) -> None:
    """Affiche le tableau de résultats filtres et la vue match si sélectionné."""
    if filtered.is_empty():
        st.warning(t("exp_no_results"))
        return

    enriched = enrich_for_table(filtered, waypoint_player, df_full)
    st.subheader(t("exp_results_title", count=len(enriched)))

    html = render_match_table_html(enriched, waypoint_player=waypoint_player)
    st.markdown(html, unsafe_allow_html=True)

    # Si un match est sélectionné via le selectbox, afficher sa vue
    mid = selected_match_id or st.session_state.get(_SK_SELECTED_MATCH)
    if mid:
        show_single_match(
            df,
            mid,
            db_path,
            xuid,
            waypoint_player,
            db_key,
            settings,
            df_full,
            render_match_view_fn,
            normalize_mode_label_fn,
            format_score_label_fn,
            score_css_color_fn,
            format_datetime_fn,
            load_player_match_result_fn,
            load_match_medals_fn,
            load_highlight_events_fn,
            load_match_gamertags_fn,
            load_match_rosters_fn,
            paris_tz,
        )


# ---------------------------------------------------------------------------
# Résultats par joueur (section 4b)
# ---------------------------------------------------------------------------


def render_player_results(  # noqa: PLR0913
    db_path: str,
    self_xuid: str,
    target_xuid: str,
    target_gt: str,
    waypoint_player: str,
    df: pl.DataFrame,
    db_key: Any,
    settings: Any,
    df_full: Any,
    render_match_view_fn: Callable[..., Any],
    normalize_mode_label_fn: Callable[..., Any],
    format_score_label_fn: Callable[..., Any],
    score_css_color_fn: Callable[..., Any],
    format_datetime_fn: Callable[..., Any],
    load_player_match_result_fn: Callable[..., Any],
    load_match_medals_fn: Callable[..., Any],
    load_highlight_events_fn: Callable[..., Any],
    load_match_gamertags_fn: Callable[..., Any],
    load_match_rosters_fn: Callable[..., Any],
    paris_tz: Any,
) -> None:
    """Affiche les résultats de recherche par joueur avec bilan encounter."""
    from src.ui.pages.explorer_data import load_common_matches

    common = load_common_matches(db_path, self_xuid, target_xuid)
    if common.is_empty():
        logger.info("Aucun match commun entre %s et %s", self_xuid, target_xuid)
        st.warning(t("exp_no_results"))
        return
    logger.debug("%d matchs communs trouvés avec %s", len(common), target_gt)

    # Bilan encounter (réutilise la logique existante)
    _render_encounter_summary(db_path, self_xuid, target_xuid, target_gt)

    # Séparer alliés / adversaires
    allies, enemies = split_by_team(common)

    if not allies.is_empty():
        st.subheader(t("exp_results_ally", count=len(allies)))
        html_ally = render_match_table_html(
            enrich_common_matches(allies, df_full),
            waypoint_player=waypoint_player,
            header_css_class="os-sb-team--mine",
        )
        st.markdown(html_ally, unsafe_allow_html=True)

    if not enemies.is_empty():
        st.subheader(t("exp_results_enemy", count=len(enemies)))
        html_enemy = render_match_table_html(
            enrich_common_matches(enemies, df_full),
            waypoint_player=waypoint_player,
            header_css_class="os-sb-team--enemy",
        )
        st.markdown(html_enemy, unsafe_allow_html=True)


def _render_encounter_summary(
    db_path: str,
    self_xuid: str,
    target_xuid: str,
    target_gt: str,
) -> None:
    """Affiche le bilan encounter avec badges au-dessus des tableaux."""
    from src.data.repositories._encounter_loader import load_encounter_stats
    from src.ui.pages.match_view_encounters import _badge_html
    from src.ui.pages.match_view_encounters_logic import (
        EncounterStats,
        compute_encounter_badges,
    )

    encounter_df = load_encounter_stats(self_xuid, [target_xuid], db_path)
    if encounter_df.is_empty():
        logger.debug("Pas de stats encounter pour %s vs %s", self_xuid, target_xuid)
        return

    row = encounter_df.row(0, named=True)
    stats = EncounterStats(
        xuid=target_xuid,
        gamertag=target_gt,
        total_encounters=int(row.get("total_encounters") or 0),
        ally_count=int(row.get("ally_count") or 0),
        enemy_count=int(row.get("enemy_count") or 0),
        winrate_as_ally=_safe_float(row.get("winrate_as_ally")),
        winrate_vs_enemy=_safe_float(row.get("winrate_vs_enemy")),
        kills_dealt=int(row.get("kills_dealt") or 0),
        deaths_suffered=int(row.get("deaths_suffered") or 0),
    )
    badges = compute_encounter_badges(stats)
    badges_html = " ".join(_badge_html(b) for b in badges) if badges else ""

    parts = [
        f"<strong>{t('exp_player_summary', gamertag=target_gt)}</strong>",
        f" — {stats.total_encounters} matchs",
        f" (A:{stats.ally_count} | E:{stats.enemy_count})",
    ]
    if badges_html:
        parts.append(f" {badges_html}")

    st.markdown("".join(parts), unsafe_allow_html=True)


def _safe_float(v: object) -> float | None:
    """Convertit en float ou None."""
    return float(v) if v is not None else None  # type: ignore[arg-type]


# ---------------------------------------------------------------------------
# Affichage d'un match unique
# ---------------------------------------------------------------------------


def show_single_match(  # noqa: PLR0913
    df: pl.DataFrame,
    match_id: str,
    db_path: str,
    xuid: str,
    waypoint_player: str,
    db_key: Any,
    settings: Any,
    df_full: Any,
    render_match_view_fn: Callable[..., Any],
    normalize_mode_label_fn: Callable[..., Any],
    format_score_label_fn: Callable[..., Any],
    score_css_color_fn: Callable[..., Any],
    format_datetime_fn: Callable[..., Any],
    load_player_match_result_fn: Callable[..., Any],
    load_match_medals_fn: Callable[..., Any],
    load_highlight_events_fn: Callable[..., Any],
    load_match_gamertags_fn: Callable[..., Any],
    load_match_rosters_fn: Callable[..., Any],
    paris_tz: Any,
) -> None:
    """Affiche la vue détaillée d'un match unique."""
    st.divider()
    rows = df.filter(pl.col("match_id").cast(pl.Utf8) == match_id)
    if rows.is_empty():
        logger.warning("Match %s introuvable dans le DataFrame (%d lignes)", match_id, len(df))
        st.warning(t("last_match_not_found"))
        return

    row = rows.sort("start_time").row(-1, named=True)
    render_match_view_fn(
        row=row,
        match_id=match_id,
        db_path=db_path,
        xuid=xuid,
        waypoint_player=waypoint_player,
        db_key=db_key,
        settings=settings,
        df_full=df_full,
        normalize_mode_label_fn=normalize_mode_label_fn,
        format_score_label_fn=format_score_label_fn,
        score_css_color_fn=score_css_color_fn,
        format_datetime_fn=format_datetime_fn,
        load_player_match_result_fn=load_player_match_result_fn,
        load_match_medals_fn=load_match_medals_fn,
        load_highlight_events_fn=load_highlight_events_fn,
        load_match_gamertags_fn=load_match_gamertags_fn,
        load_match_rosters_fn=load_match_rosters_fn,
        paris_tz=paris_tz,
    )
