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

from src.ui.i18n import t

from src.data.services.teammates_service import TeammatesService
from src.ui.cache import cached_has_cache_tables
from src.ui.pages.teammates_helpers import render_teammate_cards
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


def render_teammates_page(
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

    # Vérification du cache pour performance
    if not st.session_state.get("_cache_warning_shown"):
        has_cache = cached_has_cache_tables(db_path, db_key)
        if not has_cache:
            st.warning(
                "⚠️ **Performance** : Les tables de cache ne sont pas initialisées. "
                "Le chargement sera plus lent. Exécutez `python scripts/migrate_to_cache.py` "
                "pour accélérer significativement cette page.",
                icon="⚠️",
            )
            st.session_state["_cache_warning_shown"] = True

    apply_current_filters_teammates = st.toggle(
        "Appliquer les filtres actuels (période/sessions + map/playlist)",
        value=True,
        key="apply_current_filters_teammates",
    )
    same_team_only_teammates = st.checkbox(
        "Même équipe", value=True, key="teammates_same_team_only"
    )

    show_smooth_teammates = st.toggle(
        "Afficher les courbes lissées",
        value=bool(st.session_state.get("teammates_show_smooth", True)),
        key="teammates_show_smooth",
        help="Active/désactive les courbes de moyenne lissée sur les graphes de cette section.",
    )

    with perf_section("teammates/build_friends_opts_map"):
        opts_map, default_labels = build_friends_opts_map_fn(
            db_path, xuid.strip(), db_key, aliases_key
        )
    picked_labels = st.multiselect(
        "Coéquipiers",
        options=list(opts_map.keys()),
        default=default_labels,
        key="teammates_picked_labels",
    )
    picked_xuids = [opts_map[lbl] for lbl in picked_labels if lbl in opts_map]

    # Spartan ID cards des coéquipiers sélectionnés
    with perf_section("teammates/render_cards"):
        render_teammate_cards(picked_xuids, settings, db_path=db_path)

    # Tendance de session (matchs affichés) — multi-joueurs
    _req_trend = ["start_time", "kills", "deaths"]
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

        st.subheader("Tendance de session")
        st.caption("F/M 1ère moitié → 2nde moitié des matchs affichés.")
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

    if len(picked_xuids) < 1:
        st.info("Sélectionne au moins un coéquipier.")
    elif len(picked_xuids) == 1:
        render_single_teammate_view(
            df=df,
            dff=dff,
            me_name=me_name,
            xuid=xuid,
            db_path=db_path,
            db_key=db_key,
            picked_xuids=picked_xuids,
            apply_current_filters=apply_current_filters_teammates,
            same_team_only=same_team_only_teammates,
            show_smooth=show_smooth_teammates,
            assign_player_colors_fn=assign_player_colors_fn,
            plot_multi_metric_bars_fn=plot_multi_metric_bars_fn,
            top_medals_fn=top_medals_fn,
            load_teammate_stats_fn=_load_teammate_stats_from_own_db,
            enrich_series_fn=_enrich_series_with_perfect_kills,
        )
    else:
        render_multi_teammate_view(
            df=df,
            dff=dff,
            base=base,
            me_name=me_name,
            xuid=xuid,
            db_path=db_path,
            db_key=db_key,
            aliases_key=aliases_key,
            picked_xuids=picked_xuids,
            picked_session_labels=picked_session_labels,
            apply_current_filters=apply_current_filters_teammates,
            same_team_only=same_team_only_teammates,
            show_smooth=show_smooth_teammates,
            include_firefight=include_firefight,
            waypoint_player=waypoint_player,
            assign_player_colors_fn=assign_player_colors_fn,
            plot_multi_metric_bars_fn=plot_multi_metric_bars_fn,
            top_medals_fn=top_medals_fn,
            load_teammate_stats_fn=_load_teammate_stats_from_own_db,
            enrich_series_fn=_enrich_series_with_perfect_kills,
        )
