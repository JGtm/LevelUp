"""Page Mes coéquipiers.

Analyse des statistiques avec les coéquipiers fréquents.

Ce module est le point d'entrée de la page. Les fonctions de rendu lourdes
sont déléguées aux sous-modules :
- teammates_views.py     : vues single / multi / trio
- teammates_synergy.py   : radars de complémentarité
- teammates_impact.py    : onglet Impact
- teammates_charts.py    : graphes de comparaison
- teammates_helpers.py   : helpers de rendu (cartes, tableaux)
"""

from __future__ import annotations

import polars as pl
import streamlit as st

from src.app._filters_friends import build_teammates_opts_map
from src.config import CORE_STAT_COLUMNS
from src.data.services.teammates_service import TeammatesService
from src.ui.cache import cached_has_cache_tables
from src.ui.i18n import t
from src.ui.pages.teammates_views import (
    render_multi_teammate_view,
    render_single_teammate_view,
)
from src.ui.perf import perf_section
from src.visualization._compat import DataFrameLike, ensure_polars

# =============================================================================
# Helpers légers — chargement et enrichissement
# =============================================================================


def _load_teammate_stats_from_own_db(
    teammate_gamertag: str,
    match_ids: set[str],
    reference_db_path: str,
) -> DataFrameLike:
    """Charge les stats d'un coéquipier depuis sa propre DB si disponible.

    Délègue au TeammatesService (Sprint 14 — isolation backend/frontend).

    Args:
        teammate_gamertag: Gamertag du coéquipier.
        match_ids: Set des match_id à filtrer.
        reference_db_path: Chemin vers la DB de référence.

    Returns:
        DataFrame des stats du coéquipier (filtré sur match_ids), ou vide si DB non trouvée.
    """
    result = TeammatesService.load_teammate_stats(teammate_gamertag, match_ids, reference_db_path)
    return result.df


def _enrich_series_with_perfect_kills(
    series: list[tuple[str, DataFrameLike]],
    db_path: str,
) -> list[tuple[str, DataFrameLike]]:
    """Ajoute la colonne perfect_kills à chaque DataFrame de la série.

    Délègue au TeammatesService (Sprint 14 — isolation backend/frontend).
    """
    result = TeammatesService.enrich_series_with_perfect_kills(series, db_path)
    return result.series


# =============================================================================
# Point d'entrée de la page
# =============================================================================


def render_teammates_page(  # noqa: C901, PLR0912, PLR0913, PLR0915
    df: DataFrameLike,
    dff: DataFrameLike,
    base: DataFrameLike,
    me_name: str,
    xuid: str,
    db_path: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    settings: object,
    picked_session_labels: list[str] | None,
    include_firefight: bool,
    waypoint_player: str,
    build_friends_opts_map_fn,
    assign_player_colors_fn,
    plot_multi_metric_bars_fn,
    top_medals_fn,
) -> None:
    """Affiche la page Mes coéquipiers.

    Args:
        df: DataFrame complet des matchs.
        dff: DataFrame filtré des matchs.
        base: DataFrame de base (après filtres Firefight).
        me_name: Nom affiché du joueur.
        xuid: XUID du joueur.
        db_path: Chemin vers la base de données.
        db_key: Clé de cache de la DB.
        aliases_key: Clé de cache des alias.
        settings: Paramètres de l'application.
        picked_session_labels: Labels des sessions sélectionnées.
        include_firefight: Inclure Firefight dans les stats.
        waypoint_player: Nom Waypoint du joueur.
        build_friends_opts_map_fn: Fonction pour construire la map des coéquipiers.
        assign_player_colors_fn: Fonction pour assigner les couleurs aux joueurs.
        plot_multi_metric_bars_fn: Fonction pour tracer les barres multi-métriques.
        top_medals_fn: Fonction pour récupérer les top médailles.
    """
    df = ensure_polars(df)
    dff = ensure_polars(dff)
    base = ensure_polars(base)

    # Protection contre les DataFrames vides
    if dff.is_empty():
        st.warning(t("no_matches"))
        return

    # Session solo — aucun coéquipier
    if (
        picked_session_labels is not None
        and st.session_state.get("picked_solo_session_label", "(toutes)") != "(toutes)"
    ):
        st.info(t("tm_solo_session_info"))
        return

    # Vérification du cache pour performance
    if not st.session_state.get("_cache_warning_shown"):
        has_cache = cached_has_cache_tables(db_path, db_key)
        if not has_cache:
            st.warning(t("tm_loading_slow"), icon="⚠️")
            st.session_state["_cache_warning_shown"] = True

    with perf_section("teammates/build_friends_opts_map"):
        opts_map, default_labels = build_teammates_opts_map(
            db_path, xuid.strip(), db_key, aliases_key
        )
    # Préférer la sélection persistée dans FilterPreferences (v5.3)
    _persisted = st.session_state.get("teammates_picked_labels")
    _valid_persisted = (
        [lbl for lbl in _persisted if lbl in opts_map] if isinstance(_persisted, list) else []
    )
    _effective_default = _valid_persisted if _valid_persisted else default_labels
    if "teammates_picked_labels" not in st.session_state:
        st.session_state["teammates_picked_labels"] = _effective_default

    picked_labels = st.multiselect(
        label=t("tm_select_teammates"),
        options=list(opts_map.keys()),
        key="teammates_picked_labels",
        max_selections=3,
    )
    picked_xuids = [opts_map[lbl] for lbl in picked_labels if lbl in opts_map]

    # Tendance de session (matchs affichés) — multi-joueurs
    _req_trend = CORE_STAT_COLUMNS
    if len(dff) >= 4 and all(c in dff.columns for c in _req_trend):
        from src.analysis.cumulative import compute_session_trend_polars
        from src.ui import display_name_from_xuid

        players_trend: list[tuple[str, dict]] = []

        # Joueur principal
        pl_dff = dff.sort("start_time").select(_req_trend)
        players_trend.append((me_name, compute_session_trend_polars(pl_dff)))

        # Coéquipiers sélectionnés
        if picked_xuids and "match_id" in dff.columns:
            _match_ids = set(dff["match_id"].cast(pl.Utf8).to_list())
            for _friend_xuid in picked_xuids:
                _friend_name = display_name_from_xuid(_friend_xuid, db_path=db_path)
                try:
                    _friend_df = ensure_polars(
                        _load_teammate_stats_from_own_db(_friend_name, _match_ids, db_path)
                    )
                    if (
                        not _friend_df.is_empty()
                        and len(_friend_df) >= 4
                        and all(c in _friend_df.columns for c in _req_trend)
                    ):
                        _friend_pl = _friend_df.sort("start_time").select(_req_trend)
                        players_trend.append(
                            (_friend_name, compute_session_trend_polars(_friend_pl))
                        )
                except Exception:
                    pass

        st.subheader(t("tm_session_trend"))
        st.caption(t("tm_kd_half_caption"))
        _trend_cols = st.columns(len(players_trend))
        for _col, (_pname, _td) in zip(_trend_cols, players_trend, strict=False):
            with _col:
                _first = _td.get("first_half_kd")
                _second = _td.get("second_half_kd")
                _pct = _td.get("kd_change_pct", 0) or 0
                _tr = _td.get("trend", "stable")
                if _first is None or _second is None:
                    st.metric(_pname, "N/A")
                else:
                    _trend_icon = (
                        "▲" if _tr == "improving" else ("▼" if _tr == "declining" else "◆")
                    )
                    st.metric(
                        label=f"{_trend_icon} {_pname}",
                        value=f"{_second:.2f} F/M",
                        delta=f"{_pct:+.1f}%",
                        delta_color="normal" if _tr != "stable" else "off",
                    )
                    st.caption(f"{_first:.2f} → {_second:.2f}")

    _ctx = {
        "me_name": me_name,
        "xuid": xuid,
        "db_path": db_path,
        "db_key": db_key,
        "picked_xuids": picked_xuids,
        "aliases_key": aliases_key,
        "picked_session_labels": picked_session_labels,
        "include_firefight": include_firefight,
        "waypoint_player": waypoint_player,
    }
    _filters = {
        "apply_current_filters": True,
        "same_team_only": True,
        "show_smooth": True,
    }
    _callbacks = {
        "assign_player_colors_fn": assign_player_colors_fn,
        "plot_multi_metric_bars_fn": plot_multi_metric_bars_fn,
        "top_medals_fn": top_medals_fn,
        "load_teammate_stats_fn": _load_teammate_stats_from_own_db,
        "enrich_series_fn": _enrich_series_with_perfect_kills,
    }

    if len(picked_xuids) < 1:
        st.info(t("tm_select_teammate"))
    elif len(picked_xuids) == 1:
        render_single_teammate_view(
            df=df,
            dff=dff,
            ctx=_ctx,
            filters=_filters,
            callbacks=_callbacks,
        )
    else:
        render_multi_teammate_view(
            df=df,
            dff=dff,
            base=base,
            ctx=_ctx,
            filters=_filters,
            callbacks=_callbacks,
        )
