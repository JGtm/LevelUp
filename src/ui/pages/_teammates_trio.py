"""Vue escouade (moi + 2 ou 3 coéquipiers) pour la page Escouade/Squad.

Extraite de teammates_views.py — contient render_trio_view.
Les helpers privés sont dans _teammates_trio_helpers.py.
"""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.data.services.teammates_service import TeammatesService
from src.ui import display_name_from_xuid
from src.ui.cache import cached_same_team_match_ids_with_friend
from src.ui.i18n import t
from src.ui.pages._teammates_trio_helpers import (
    _detect_trio_session,
    _render_per_minute_stats,
    _render_trio_performance_charts,
)
from src.ui.pages.teammates_charts import render_metric_bar_charts
from src.ui.pages.teammates_synergy import render_trio_synergy_radar
from src.ui.pages.teammates_weapons import render_weapon_kills_bar_chart
from src.visualization._compat import DataFrameLike, ensure_polars

# ---------------------------------------------------------------------------
# Vue trio publique
# ---------------------------------------------------------------------------


def render_trio_view(  # noqa: PLR0913, PLR0915, C901, PLR0912
    df: DataFrameLike,
    dff: DataFrameLike,
    base: DataFrameLike,
    me_name: str,
    xuid: str,
    db_path: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    picked_xuids: list[str],
    apply_current_filters: bool,
    include_firefight: bool,
    series: list[tuple[str, DataFrameLike]],
    colors_by_name: dict[str, str],
    show_smooth: bool,
    assign_player_colors_fn,
    plot_multi_metric_bars_fn,
    top_medals_fn,
    load_teammate_stats_fn,
    enrich_series_fn,
) -> bool:
    """Affiche la vue escouade (moi + 2 ou 3 coéquipiers). Retourne True si les graphes du bas sont rendus."""
    df = ensure_polars(df)
    dff = ensure_polars(dff)
    base = ensure_polars(base)
    f1_xuid = picked_xuids[0]
    f2_xuid = picked_xuids[1] if len(picked_xuids) >= 2 else None
    f3_xuid = picked_xuids[2] if len(picked_xuids) >= 3 else None

    f1_name = display_name_from_xuid(f1_xuid, db_path=db_path)
    f2_name = display_name_from_xuid(f2_xuid, db_path=db_path) if f2_xuid else None
    f3_name = display_name_from_xuid(f3_xuid, db_path=db_path) if f3_xuid else None

    squad_size = 2 + (1 if f2_xuid else 0) + (1 if f3_xuid else 0)
    logger.info(
        "render_trio_view: escouade %d joueurs — %s",
        squad_size,
        [me_name, f1_name] + ([f2_name] if f2_name else []) + ([f3_name] if f3_name else []),
    )

    friend_names_all = [f1_name] + ([f2_name] if f2_name else []) + ([f3_name] if f3_name else [])
    st.subheader(t("tm_squad_header", names=" + ".join(friend_names_all)))

    squad_ids = set(
        cached_same_team_match_ids_with_friend(db_path, xuid.strip(), f1_xuid, db_key=db_key)
    )
    if f2_xuid:
        ids_c = set(
            cached_same_team_match_ids_with_friend(db_path, xuid.strip(), f2_xuid, db_key=db_key)
        )
        squad_ids = squad_ids & ids_c
    if f3_xuid:
        ids_d = set(
            cached_same_team_match_ids_with_friend(db_path, xuid.strip(), f3_xuid, db_key=db_key)
        )
        squad_ids = squad_ids & ids_d

    base_for_trio = dff if apply_current_filters else df
    # Sauvegarder l'ensemble complet AVANT le filtre UI :
    # le radar de complémentarité utilise l'historique all-time (profil de style de jeu global).
    # Les DFs timeline/graphes utilisent squad_ids filtré.
    radar_squad_ids = squad_ids.copy()
    squad_ids = squad_ids & set(base_for_trio["match_id"].cast(pl.Utf8).to_list())

    logger.debug(
        "render_trio_view: squad_ids=%d (f2=%s f3=%s)",
        len(squad_ids),
        f2_xuid is not None,
        f3_xuid is not None,
    )
    logger.info("render_trio_view: %d matchs d'escouade trouvés", len(squad_ids))
    if not squad_ids:
        st.warning(t("tm_no_trio_matches"))
        return False

    trio_ids_set = {str(x) for x in squad_ids}

    _detect_trio_session(db_path, xuid, db_key, include_firefight, aliases_key, trio_ids_set)

    me_df = base_for_trio.filter(pl.col("match_id").is_in(list(squad_ids)))

    # Charger les stats des coéquipiers depuis leurs propres DBs
    f1_df = ensure_polars(load_teammate_stats_fn(f1_name, trio_ids_set, db_path))
    f2_df: pl.DataFrame | None = (
        ensure_polars(load_teammate_stats_fn(f2_name, trio_ids_set, db_path)) if f2_name else None
    )
    f3_df: pl.DataFrame | None = (
        ensure_polars(load_teammate_stats_fn(f3_name, trio_ids_set, db_path)) if f3_name else None
    )

    # Enrichir avec les performance_score stockés dans player_match_enrichment
    me_df = TeammatesService.enrich_with_performance_score(me_df, me_name, db_path, is_main=True)
    f1_df = TeammatesService.enrich_with_performance_score(f1_df, f1_name, db_path)
    if f2_df is not None and f2_name:
        f2_df = TeammatesService.enrich_with_performance_score(f2_df, f2_name, db_path)
    if f3_df is not None and f3_name:
        f3_df = TeammatesService.enrich_with_performance_score(f3_df, f3_name, db_path)

    # Filtrer les DataFrames pour ne garder que les match_ids présents dans me_df
    filtered_match_ids = me_df["match_id"].cast(pl.Utf8).to_list() if not me_df.is_empty() else []
    if not f1_df.is_empty() and "match_id" in f1_df.columns and filtered_match_ids:
        f1_df = f1_df.filter(pl.col("match_id").cast(pl.Utf8).is_in(filtered_match_ids))
    if (
        f2_df is not None
        and not f2_df.is_empty()
        and "match_id" in f2_df.columns
        and filtered_match_ids
    ):
        f2_df = f2_df.filter(pl.col("match_id").cast(pl.Utf8).is_in(filtered_match_ids))
    if (
        f3_df is not None
        and not f3_df.is_empty()
        and "match_id" in f3_df.columns
        and filtered_match_ids
    ):
        f3_df = f3_df.filter(pl.col("match_id").cast(pl.Utf8).is_in(filtered_match_ids))

    me_df = me_df.sort("start_time")

    _render_per_minute_stats(
        me_df,
        f1_df,
        f2_df,
        me_name,
        f1_name,
        f2_name,
        colors_by_name,
        f3_df=f3_df,
        f3_name=f3_name,
    )

    # Radar de complémentarité escouade — utilise l'historique all-time (radar_squad_ids)
    # pour que les PSA soient disponibles même si les filtres UI excluent les matchs récents.
    radar_me_df = df.filter(pl.col("match_id").cast(pl.Utf8).is_in(list(radar_squad_ids)))
    radar_ids_str = {str(x) for x in radar_squad_ids}
    radar_f1_df = ensure_polars(load_teammate_stats_fn(f1_name, radar_ids_str, db_path))
    radar_f2_df: pl.DataFrame | None = (
        ensure_polars(load_teammate_stats_fn(f2_name, radar_ids_str, db_path)) if f2_name else None
    )
    radar_f3_df: pl.DataFrame | None = (
        ensure_polars(load_teammate_stats_fn(f3_name, radar_ids_str, db_path)) if f3_name else None
    )
    logger.debug(
        "render_trio_view: radar squad_ids=%d (all-time) vs squad_ids=%d (filtrés)",
        len(radar_squad_ids),
        len(squad_ids),
    )
    render_trio_synergy_radar(
        me_df=radar_me_df,
        f1_df=radar_f1_df,
        f2_df=radar_f2_df,
        me_name=me_name,
        f1_name=f1_name,
        f2_name=f2_name,
        colors_by_name=colors_by_name,
        db_path=db_path,
        f3_df=radar_f3_df,
        f3_name=f3_name,
    )

    # Vérification alignment : besoin d'au moins me + f1
    if me_df.is_empty() or f1_df.is_empty():
        st.warning(t("tm_trio_warning"))
        return False

    # Si f2 était attendu mais vide, on le retire silencieusement
    if f2_xuid and (f2_df is None or f2_df.is_empty()):
        logger.warning(
            "render_trio_view: f2 (%s) vide après chargement — retiré de l'escouade", f2_name
        )
        f2_df = None
        f2_xuid = None
        f2_name = None

    # Si f3 était attendu mais vide, on le retire silencieusement
    if f3_xuid and (f3_df is None or f3_df.is_empty()):
        logger.warning(
            "render_trio_view: f3 (%s) vide après chargement — retiré de l'escouade", f3_name
        )
        f3_df = None
        f3_xuid = None
        f3_name = None

    _render_trio_performance_charts(
        me_df,
        f1_df,
        f2_df,
        me_name,
        f1_name,
        f2_name,
        f1_xuid,
        f2_xuid,
        f3_df=f3_df,
        f3_name=f3_name,
        f3_xuid=f3_xuid,
        colors_by_name=colors_by_name,
    )

    # Graphes de barres - reconstruire series avec les DataFrames de l'escouade
    series = [(me_name, me_df)]
    if not f1_df.is_empty():
        series.append((f1_name, f1_df))
    if f2_df is not None and not f2_df.is_empty():
        series.append((f2_name, f2_df))
    if f3_name and f3_df is not None and not f3_df.is_empty():
        series.append((f3_name, f3_df))
    colors_by_name = assign_player_colors_fn([n for n, _ in series])
    series = enrich_series_fn(series, db_path)

    squad_match_ids = me_df["match_id"].cast(pl.Utf8).to_list() if not me_df.is_empty() else []

    # Graphe des armes de kill par joueur (escouade)
    _weapon_player_infos = [
        (me_name, xuid, squad_match_ids),
        (f1_name, f1_xuid, squad_match_ids),
    ]
    if f2_xuid and f2_name:
        _weapon_player_infos.append((f2_name, f2_xuid, squad_match_ids))
    if f3_xuid and f3_name:
        _weapon_player_infos.append((f3_name, f3_xuid, squad_match_ids))
    render_weapon_kills_bar_chart(
        player_infos=_weapon_player_infos,
        colors_by_name=colors_by_name,
        db_path=db_path,
        key_suffix=f"trio_{len(_weapon_player_infos)}",
    )

    render_metric_bar_charts(
        series=series,
        colors_by_name=colors_by_name,
        show_smooth=show_smooth,
        key_suffix=f"trio_{len(series)}",
        plot_fn=plot_multi_metric_bars_fn,
    )

    return True
