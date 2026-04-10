"""Vue escouade (moi + 2 ou 3 coéquipiers) pour la page Escouade/Squad.

Extraite de teammates_views.py — contient render_trio_view.
Les helpers privés sont dans _teammates_trio_helpers.py.
"""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.analysis.squad_records import (
    compute_player_pm_records,
    compute_squad_records,
    compute_squad_records_per_map,
    get_dominant_pair_name,
)
from src.data.services.teammates_service import TeammatesService
from src.ui import display_name_from_xuid
from src.ui.cache import cached_same_team_match_ids_with_friend
from src.ui.i18n import get_lang, t
from src.ui.pages._teammates_trio_helpers import (
    _detect_trio_session,
    _render_per_minute_stats,
    _render_trio_performance_charts,
)
from src.ui.pages.teammates_charts import render_first_events_chart, render_metric_bar_charts
from src.ui.pages.teammates_intensity import render_squad_intensity_heatmap
from src.ui.pages.teammates_synergy import render_trio_synergy_radar
from src.ui.pages.teammates_weapons import render_weapon_kills_bar_chart
from src.visualization._chart_series import SquadRecordSet
from src.visualization._compat import DataFrameLike, ensure_polars

# ---------------------------------------------------------------------------
# Helpers privés
# ---------------------------------------------------------------------------


def _build_pm_records(
    full_raw: list[tuple[str, pl.DataFrame]],
    dominant_pair: str | None,
    session_dfs: list[tuple[str | None, DataFrameLike | None]],
) -> dict[str, tuple[float | None, float | None, float | None]]:
    """Records stats/min par joueur avec fallback session si historique vide."""
    records = {
        name: compute_player_pm_records(full_df, dominant_pair) for name, full_df in full_raw
    }
    for _n, _fb in session_dfs:
        if not _n or _fb is None:
            continue
        if _n not in records or all(v is None for v in records[_n]):
            records[_n] = compute_player_pm_records(ensure_polars(_fb), None)
    return records


# Métriques records partagées entre compute_squad_records et compute_squad_records_per_map
_RECORD_METRICS = [
    ("kills", False),
    ("deaths", True),
    ("assists", False),
    ("ratio", False),
    ("accuracy", False),
    ("average_life_seconds", False),
    ("performance_score", False),
    ("max_killing_spree", False),
    ("headshot_kills", False),
]
_RECORD_METRICS_PER_MAP = [m for m in _RECORD_METRICS if m[0] != "headshot_kills"]


def _rename_perf_score(records: dict[str, dict[str, float | None]]) -> None:
    """Renomme performance_score → performance dans les records (in-place)."""
    for r in records.values():
        if "performance_score" in r:
            r["performance"] = r.pop("performance_score")


def _compute_all_records(  # noqa: PLR0913
    df: DataFrameLike,
    me_name: str,
    f1_name: str,
    f2_name: str | None,
    f3_name: str | None,
    db_path: str,
    me_df: pl.DataFrame,
    f1_df: pl.DataFrame,
    f2_df: pl.DataFrame | None,
    f3_df: pl.DataFrame | None,
) -> tuple[SquadRecordSet, dict[str, tuple[float | None, float | None, float | None]]]:
    """Calcule les records historiques (SquadRecordSet + pm_records)."""
    _me_full = ensure_polars(df).filter(
        pl.col("is_firefight").cast(pl.Boolean).not_()
        if "is_firefight" in ensure_polars(df).columns
        else pl.lit(True)
    )
    if "map_ui" not in _me_full.columns and "map_name_fr" in _me_full.columns:
        _me_full = _me_full.with_columns(
            pl.coalesce(
                [pl.col("map_name_fr").cast(pl.Utf8), pl.col("map_name").cast(pl.Utf8)]
            ).alias("map_ui")
        )
    _f1_full = TeammatesService.load_all_teammate_stats(f1_name, db_path).df
    _f2_full = (
        TeammatesService.load_all_teammate_stats(f2_name, db_path).df if f2_name else pl.DataFrame()
    )
    _f3_full = (
        TeammatesService.load_all_teammate_stats(f3_name, db_path).df if f3_name else pl.DataFrame()
    )

    _full_raw: list[tuple[str, pl.DataFrame]] = [(me_name, _me_full), (f1_name, _f1_full)]
    if f2_name and not _f2_full.is_empty():
        _full_raw.append((f2_name, _f2_full))
    if f3_name and not _f3_full.is_empty():
        _full_raw.append((f3_name, _f3_full))

    _dominant_pair = get_dominant_pair_name([d for _, d in _full_raw])

    _squad_records = compute_squad_records(_full_raw, _RECORD_METRICS, _dominant_pair)
    _rename_perf_score(_squad_records)

    _squad_records_per_map = compute_squad_records_per_map(
        _full_raw, _RECORD_METRICS_PER_MAP, _dominant_pair
    )
    _rename_perf_score(_squad_records_per_map)

    record_set = SquadRecordSet(records=_squad_records, per_map=_squad_records_per_map)
    pm_records = _build_pm_records(
        _full_raw,
        _dominant_pair,
        [(me_name, me_df), (f1_name, f1_df), (f2_name, f2_df), (f3_name, f3_df)],
    )
    return record_set, pm_records


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
    from src.ui.pages.teammates_legend import render_player_legend_panel

    render_player_legend_panel(colors_by_name)

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

    # ── Records depuis l'historique COMPLET (pas filtré à l'escouade) ────────
    # Conditionnés par le setting show_records (page Paramètres).
    _show_records = bool(getattr(st.session_state.get("app_settings"), "show_records", False))

    _squad_record_set: SquadRecordSet | None = None
    _pm_records: dict[str, tuple[float | None, float | None, float | None]] | None = None

    if _show_records:
        _squad_record_set, _pm_records = _compute_all_records(
            df,
            me_name,
            f1_name,
            f2_name,
            f3_name,
            db_path,
            me_df,
            f1_df,
            f2_df,
            f3_df,
        )

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
        pm_records=_pm_records,
    )

    # Radar de complémentarité escouade — filtré par la session/période courante
    radar_ids_str = {str(x) for x in squad_ids}
    radar_me_df = base_for_trio.filter(pl.col("match_id").cast(pl.Utf8).is_in(list(squad_ids)))
    radar_f1_df = ensure_polars(load_teammate_stats_fn(f1_name, radar_ids_str, db_path))
    radar_f2_df: pl.DataFrame | None = (
        ensure_polars(load_teammate_stats_fn(f2_name, radar_ids_str, db_path)) if f2_name else None
    )
    radar_f3_df: pl.DataFrame | None = (
        ensure_polars(load_teammate_stats_fn(f3_name, radar_ids_str, db_path)) if f3_name else None
    )
    logger.debug("render_trio_view: radar squad_ids=%d (filtrés par session)", len(squad_ids))
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

    # Heatmap d'intensité par joueur — toggle segmented_control
    _intensity_xuid_name: dict[str, str] = {xuid: me_name}
    if f1_xuid:
        _intensity_xuid_name[f1_xuid] = f1_name
    if f2_xuid and f2_name:
        _intensity_xuid_name[f2_xuid] = f2_name
    if f3_xuid and f3_name:
        _intensity_xuid_name[f3_xuid] = f3_name
    _intensity_match_ids = (
        me_df.sort("start_time")["match_id"].cast(pl.Utf8).to_list()
        if "start_time" in me_df.columns
        else me_df["match_id"].cast(pl.Utf8).to_list()
    )
    render_squad_intensity_heatmap(
        db_path=db_path,
        xuid_name_map=_intensity_xuid_name,
        match_ids_ordered=_intensity_match_ids,
        lang=get_lang(),
    )

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
        squad_records=_squad_record_set,
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

    # Records killing spree et HS+PK — conditionnés par show_records
    _spree_record_set: SquadRecordSet | None = None
    _hspk_records: dict | None = None
    if _show_records and _squad_record_set is not None:
        _spree_record_set = SquadRecordSet(
            records={
                n: {"max_killing_spree": r.get("max_killing_spree")}
                for n, r in _squad_record_set.records.items()
            },
            per_map=_squad_record_set.per_map,
        )
        # Records HS+PK : utilise headshot_kills de l'historique complet comme proxy
        # (perfect_kills absent du full history — enrichissement non disponible hors session)
        _hspk_records = {
            n: {"hs_pk_total": r.get("headshot_kills")}
            for n, r in _squad_record_set.records.items()
        }
    render_metric_bar_charts(
        series=series,
        colors_by_name=colors_by_name,
        show_smooth=show_smooth,
        key_suffix=f"trio_{len(series)}",
        plot_fn=plot_multi_metric_bars_fn,
        squad_records=_spree_record_set,
        hspk_records=_hspk_records,
    )

    # Tendance premier frag / première mort
    # Le joueur principal (me_name) a le xuid déjà disponible en paramètre ;
    # les coéquipiers l'ont dans leur colonne xuid (via _query_teammate_shared_stats).
    xuids_by_name: dict[str, str] = {me_name: xuid}
    for _name, _df in series[1:]:
        if "xuid" in _df.columns and not _df.is_empty():
            _unique = _df["xuid"].drop_nulls().unique()
            if len(_unique) > 0:
                xuids_by_name[_name] = str(_unique[0])
    if xuids_by_name and squad_match_ids:
        render_first_events_chart(
            db_path=db_path,
            xuids_by_name=xuids_by_name,
            match_ids=squad_match_ids,
            colors_by_name=colors_by_name,
            key_suffix=f"trio_{len(series)}",
        )

    return True
