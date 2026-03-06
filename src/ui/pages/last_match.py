"""Page Dernier match.

Affiche la dernière partie selon la sélection/filtres actuels.
"""

from __future__ import annotations

import logging
from collections.abc import Callable
from typing import TYPE_CHECKING

import streamlit as st

from src.ui.i18n import t
from src.visualization._compat import DataFrameLike, ensure_polars

logger = logging.getLogger(__name__)

if TYPE_CHECKING:
    from zoneinfo import ZoneInfo

    from src.ui.settings import AppSettings


def render_last_match_page(  # noqa: PLR0913
    dff: DataFrameLike,
    db_path: str,
    xuid: str,
    waypoint_player: str,
    db_key: tuple[int, int] | None,
    settings: AppSettings,
    df_full: DataFrameLike | None,
    render_match_view_fn: Callable,
    normalize_mode_label_fn: Callable[[str | None], str | None],
    format_score_label_fn: Callable,
    score_css_color_fn: Callable,
    format_datetime_fn: Callable,
    load_player_match_result_fn: Callable,
    load_match_medals_fn: Callable,
    load_highlight_events_fn: Callable,
    load_match_gamertags_fn: Callable,
    load_match_rosters_fn: Callable,
    paris_tz: ZoneInfo,
) -> None:
    """Rend la page Dernier match.

    Affiche la dernière partie selon la sélection/filtres actuels.

    Args:
        dff: DataFrame filtré des matchs.
        db_path: Chemin vers la base de données.
        xuid: XUID du joueur.
        waypoint_player: Nom du joueur Waypoint.
        db_key: Clé de cache de la DB.
        settings: Paramètres de l'application.
        df_full: DataFrame complet pour le calcul du score relatif.
        render_match_view_fn: Fonction de rendu du match.
        normalize_mode_label_fn: Fonction de normalisation du label de mode.
        format_score_label_fn: Fonction de formatage du score.
        score_css_color_fn: Fonction de couleur CSS du score.
        format_datetime_fn: Fonction de formatage date/heure.
        load_player_match_result_fn: Fonction de chargement du résultat joueur.
        load_match_medals_fn: Fonction de chargement des médailles.
        load_highlight_events_fn: Fonction de chargement des événements.
        load_match_gamertags_fn: Fonction de chargement des gamertags.
        load_match_rosters_fn: Fonction de chargement des rosters.
        paris_tz: Timezone Paris.
    """
    st.caption(t("last_match_caption"))

    dff = ensure_polars(dff)
    if dff.is_empty():
        st.info(t("no_data_filter"))
        return

    last_row = dff.sort("start_time").row(-1, named=True)
    last_match_id = str(last_row.get("match_id", "")).strip()
    logger.debug("Dernier match auto: %s", last_match_id)

    render_match_view_fn(
        row=last_row,
        match_id=last_match_id,
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
