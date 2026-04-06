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
from src.data.services.teammates_service import TeammatesService
from src.ui.cache import cached_has_cache_tables
from src.ui.i18n import t
from src.ui.pages.teammates_views import render_multi_teammate_view
from src.ui.perf import perf_section
from src.visualization._compat import DataFrameLike, ensure_polars


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


def _select_picked_teammates(
    opts_map: dict[str, str],
    default_labels: list[str],
) -> list[str]:
    """Retourne les XUID sélectionnés dans le multiselect coéquipiers."""
    persisted = st.session_state.get("teammates_picked_labels")
    valid_persisted = (
        [label for label in persisted if label in opts_map] if isinstance(persisted, list) else []
    )
    effective_default = valid_persisted if valid_persisted else default_labels
    if "teammates_picked_labels" not in st.session_state:
        st.session_state["teammates_picked_labels"] = effective_default

    picked_labels = st.multiselect(
        label=t("tm_select_teammates"),
        options=list(opts_map.keys()),
        key="teammates_picked_labels",
        max_selections=3,
    )
    return [opts_map[label] for label in picked_labels if label in opts_map]


def _render_teammate_views(
    *,
    df: pl.DataFrame,
    dff: pl.DataFrame,
    base: pl.DataFrame,
    ctx: dict[str, object],
    callbacks: dict[str, object],
) -> None:
    """Affiche la vue single ou multi selon le nombre de coéquipiers sélectionnés."""
    filters = {
        "apply_current_filters": True,
        "same_team_only": True,
        "show_smooth": True,
    }
    picked_xuids = ctx["picked_xuids"]

    if len(picked_xuids) < 1:
        st.info(t("tm_select_teammate"))
    else:
        render_multi_teammate_view(
            df=df,
            dff=dff,
            base=base,
            ctx=ctx,
            filters=filters,
            callbacks=callbacks,
        )


def _should_abort_teammates_page(
    dff: pl.DataFrame,
    picked_session_labels: list[str] | None,
    db_path: str,
    db_key: tuple[int, int] | None,
) -> bool:
    """Retourne True si la page ne peut pas afficher de contenu utile."""
    if dff.is_empty():
        st.warning(t("no_matches"))
        return True
    if (
        picked_session_labels is not None
        and st.session_state.get("picked_solo_session_label", "(toutes)") != "(toutes)"
    ):
        st.info(t("tm_solo_session_info"))
        return True
    if st.session_state.get("_cache_warning_shown"):
        return False
    has_cache = cached_has_cache_tables(db_path, db_key)
    if has_cache:
        return False
    st.warning(t("tm_loading_slow"), icon="⚠️")
    st.session_state["_cache_warning_shown"] = True
    return False


def _build_teammates_callbacks(
    assign_player_colors_fn,
    plot_multi_metric_bars_fn,
    top_medals_fn,
) -> dict[str, object]:
    """Construit les callbacks de rendu pour les vues coéquipiers."""
    return {
        "assign_player_colors_fn": assign_player_colors_fn,
        "plot_multi_metric_bars_fn": plot_multi_metric_bars_fn,
        "top_medals_fn": top_medals_fn,
        "load_teammate_stats_fn": _load_teammate_stats_from_own_db,
        "enrich_series_fn": _enrich_series_with_perfect_kills,
    }


# =============================================================================
# Point d'entrée de la page
# =============================================================================


def render_teammates_page(  # noqa: PLR0912, PLR0913, PLR0915
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
    base_s_ui: pl.DataFrame | None,
    include_firefight: bool,
    waypoint_player: str,
    build_friends_opts_map_fn,
    assign_player_colors_fn,
    plot_multi_metric_bars_fn,
    top_medals_fn,
) -> None:
    """Affiche la page Mes coéquipiers."""
    df = ensure_polars(df)
    dff = ensure_polars(dff)
    base = ensure_polars(base)

    if _should_abort_teammates_page(dff, picked_session_labels, db_path, db_key):
        return

    with perf_section("teammates/build_friends_opts_map"):
        opts_map, default_labels = build_teammates_opts_map(
            db_path, xuid.strip(), db_key, aliases_key
        )
    picked_xuids = _select_picked_teammates(opts_map, default_labels)

    _render_squad_header_if_needed(
        scope={
            "df": df,
            "dff": dff,
            "picked_session_labels": picked_session_labels,
            "base_s_ui": base_s_ui,
        },
        player_ctx={
            "me_name": me_name,
            "db_path": db_path,
            "picked_xuids": picked_xuids,
        },
    )

    ctx = {
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
    callbacks = _build_teammates_callbacks(
        assign_player_colors_fn,
        plot_multi_metric_bars_fn,
        top_medals_fn,
    )
    _render_teammate_views(df=df, dff=dff, base=base, ctx=ctx, callbacks=callbacks)


def _get_squad_header_df(
    df: pl.DataFrame,
    dff: pl.DataFrame,
    picked_session_labels: list[str] | None,
    base_s_ui: pl.DataFrame | None,
) -> pl.DataFrame:
    """Retourne le DataFrame du joueur principal pour l'en-tête escouade.

    Quand une session est active, utilise `df` filtré par les match_id de la session
    uniquement (sans les filtres mode/playlist/carte), pour correspondre au scope
    de la page Session. Utilise `base_s_ui` — source de vérité de `apply_filters` —
    pour garantir des match_ids identiques sans recalcul indépendant.
    """
    if not picked_session_labels or base_s_ui is None or base_s_ui.is_empty():
        return dff
    session_ids = (
        base_s_ui.filter(pl.col("session_label").is_in(picked_session_labels))["match_id"]
        .cast(pl.Utf8)
        .to_list()
    )
    if not session_ids:
        return dff
    result = df.filter(pl.col("match_id").cast(pl.Utf8).is_in(session_ids))
    return result if not result.is_empty() else dff


def _render_squad_header_if_needed(
    *,
    scope: dict[str, object],
    player_ctx: dict[str, object],
) -> None:
    """Affiche l'en-tête escouade sur le même scope que la page Session."""
    dff = scope["dff"]
    if len(dff) < 2:
        return

    from src.ui import display_name_from_xuid
    from src.ui.components.performance import render_squad_session_header
    from src.ui.i18n import get_lang

    header_df = _get_squad_header_df(
        scope["df"],
        dff,
        scope["picked_session_labels"],
        scope["base_s_ui"],
    )
    players_data: list[tuple[str, object]] = [(player_ctx["me_name"], header_df)]
    picked_xuids = player_ctx["picked_xuids"]
    if picked_xuids and "match_id" in header_df.columns:
        match_ids = set(header_df["match_id"].cast(pl.Utf8).to_list())
        for friend_xuid in picked_xuids:
            friend_name = display_name_from_xuid(friend_xuid, db_path=player_ctx["db_path"])
            try:
                friend_df = ensure_polars(
                    _load_teammate_stats_from_own_db(
                        friend_name,
                        match_ids,
                        player_ctx["db_path"],
                    )
                )
                if not friend_df.is_empty():
                    players_data.append((friend_name, friend_df))
            except Exception:
                pass

    render_squad_session_header(players_data, lang=get_lang())
