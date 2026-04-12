"""Rendu des résultats pour la page Explorer.

Affiche les résultats de recherche (par filtres ou par joueur)
sous forme de tableaux HTML + vue match détaillée.
"""

from __future__ import annotations

import logging
import math

import polars as pl
import streamlit as st

from src.app._page_context import MatchViewParams
from src.ui.i18n import t
from src.ui.pages.explorer_enrich import enrich_common_matches, enrich_for_table
from src.ui.pages.explorer_logic import split_by_team
from src.ui.pages.match_table_html import render_match_table_html

logger = logging.getLogger(__name__)

_SK_SELECTED_MATCH = "_explorer_selected_match"
_DEFAULT_TABLE_PAGE_SIZE = 100
_TABLE_PAGE_SIZE_OPTIONS = (50, 100, 250)


# ---------------------------------------------------------------------------
# Résultats par filtres (section 4a)
# ---------------------------------------------------------------------------


def render_filter_results(
    filtered: pl.DataFrame,
    selected_match_id: str | None,
    df: pl.DataFrame,
    params: MatchViewParams,
) -> None:
    """Affiche le tableau de résultats filtres et la vue match si sélectionné.

    Si un match précis est sélectionné via le selectbox, affiche directement
    sa vue détaillée sans passer par le tableau récapitulatif.
    """
    if filtered.is_empty():
        st.warning(t("exp_no_results"))
        return

    mid = selected_match_id or st.session_state.get(_SK_SELECTED_MATCH)
    if mid:
        # Match précis sélectionné → vue détaillée uniquement
        show_single_match(df, mid, params)
        return

    # Aucun match précis → tableau de tous les résultats filtrés
    waypoint_player = params.waypoint_player
    df_full = params.df_full
    enriched = enrich_for_table(filtered, waypoint_player, df_full)
    st.subheader(t("exp_results_title", count=len(enriched)))
    _render_paginated_match_table(
        enriched,
        table_key="exp_filters",
        waypoint_player=waypoint_player,
    )


# ---------------------------------------------------------------------------
# Résultats par joueur (section 4b)
# ---------------------------------------------------------------------------


def render_player_results(
    target_xuid: str,
    target_gt: str,
    df: pl.DataFrame,
    params: MatchViewParams,
) -> None:
    """Affiche les résultats de recherche par joueur avec bilan encounter."""
    from src.ui.pages.explorer_data import load_common_matches

    self_xuid = params.xuid
    db_path = params.db_path
    waypoint_player = params.waypoint_player
    df_full = params.df_full

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
        _render_paginated_match_table(
            enrich_common_matches(allies, df_full),
            table_key="exp_allies",
            waypoint_player=waypoint_player,
            header_css_class="os-sb-team--mine",
            hide_empty_cols=True,
        )

    if not enemies.is_empty():
        st.subheader(t("exp_results_enemy", count=len(enemies)))
        _render_paginated_match_table(
            enrich_common_matches(enemies, df_full),
            table_key="exp_enemies",
            waypoint_player=waypoint_player,
            header_css_class="os-sb-team--enemy",
            hide_empty_cols=True,
        )


def _render_paginated_match_table(
    view: pl.DataFrame,
    *,
    table_key: str,
    waypoint_player: str | None,
    header_css_class: str = "",
    hide_empty_cols: bool = False,
) -> None:
    """Rend un tableau Explorer avec pagination légère pour limiter le DOM HTML."""
    total_rows = len(view)
    if total_rows <= 0:
        return

    page_size = _get_table_page_size(total_rows, table_key)
    page_number, start_row, end_row = _compute_table_window(
        total_rows=total_rows,
        page_size=page_size,
        current_page=st.session_state.get(f"{table_key}_page", 1),
    )

    if total_rows > _DEFAULT_TABLE_PAGE_SIZE:
        _render_pagination_controls(
            table_key=table_key,
            total_rows=total_rows,
            page_size=page_size,
            page_number=page_number,
            end_row=end_row,
        )

    html = render_match_table_html(
        view,
        waypoint_player=waypoint_player,
        header_css_class=header_css_class,
        hide_empty_cols=hide_empty_cols,
        max_rows=page_size,
        start_row=start_row,
    )
    st.markdown(html, unsafe_allow_html=True)


def _render_pagination_controls(
    *,
    table_key: str,
    total_rows: int,
    page_size: int,
    page_number: int,
    end_row: int,
) -> None:
    """Affiche les contrôles de pagination d'un tableau Explorer."""
    max_page = max(1, math.ceil(total_rows / page_size))
    info_col, size_col, page_col = st.columns([3.2, 1.2, 1.2])
    with info_col:
        st.caption(
            t(
                "exp_table_showing_rows",
                start=((page_number - 1) * page_size) + 1,
                end=end_row,
                total=total_rows,
            )
        )
    with size_col:
        st.selectbox(
            t("exp_table_rows_per_page"),
            options=_page_size_options(total_rows),
            index=_page_size_options(total_rows).index(page_size),
            key=f"{table_key}_page_size",
        )
    with page_col:
        st.number_input(
            t("exp_table_page"),
            min_value=1,
            max_value=max_page,
            value=page_number,
            step=1,
            key=f"{table_key}_page",
        )


def _get_table_page_size(total_rows: int, table_key: str) -> int:
    """Retourne la taille de page courante pour un tableau Explorer."""
    options = _page_size_options(total_rows)
    stored = st.session_state.get(f"{table_key}_page_size")
    if stored in options:
        return int(stored)
    default_size = _DEFAULT_TABLE_PAGE_SIZE if total_rows > _DEFAULT_TABLE_PAGE_SIZE else total_rows
    if default_size not in options:
        default_size = options[-1]
    st.session_state[f"{table_key}_page_size"] = default_size
    return int(default_size)


def _page_size_options(total_rows: int) -> list[int]:
    """Retourne les tailles de page proposées pour un tableau Explorer."""
    options = [size for size in _TABLE_PAGE_SIZE_OPTIONS if size < total_rows]
    if total_rows not in options:
        options.append(total_rows)
    return options or [total_rows]


def _compute_table_window(
    *, total_rows: int, page_size: int, current_page: object
) -> tuple[int, int, int]:
    """Calcule la fenêtre à rendre pour une pagination 1-based."""
    normalized_size = max(1, min(int(page_size), max(1, total_rows)))
    max_page = max(1, math.ceil(total_rows / normalized_size))
    try:
        normalized_page = int(current_page)
    except (TypeError, ValueError):
        normalized_page = 1
    normalized_page = max(1, min(normalized_page, max_page))
    start_row = (normalized_page - 1) * normalized_size
    end_row = min(start_row + normalized_size, total_rows)
    return normalized_page, start_row, end_row


def _render_encounter_summary(
    db_path: str,
    self_xuid: str,
    target_xuid: str,
    target_gt: str,
) -> None:
    """Affiche le bilan encounter avec badges au-dessus des tableaux."""
    from src.data.repositories._encounter_loader import load_encounter_stats
    from src.ui.pages.match_view_encounters import badge_html as _badge_html
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


def show_single_match(
    df: pl.DataFrame,
    match_id: str,
    params: MatchViewParams,
) -> None:
    """Affiche la vue détaillée d'un match unique."""
    st.divider()
    rows = df.filter(pl.col("match_id").cast(pl.Utf8) == match_id)
    if rows.is_empty():
        logger.warning("Match %s introuvable dans le DataFrame (%d lignes)", match_id, len(df))
        st.warning(t("last_match_not_found"))
        return

    row = rows.sort("start_time").row(-1, named=True)
    params.render_match_view_fn(
        row=row,
        match_id=match_id,
        params=params,
    )
