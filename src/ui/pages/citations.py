"""Page Citations (Commendations & Médailles)."""

from __future__ import annotations

from collections.abc import Callable

import polars as pl
import streamlit as st

from src.ui.commendations import render_h5g_commendations_section
from src.ui.i18n import get_lang, t
from src.ui.medals import load_medal_name_maps, medal_label, render_medals_grid
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.distributions import plot_medals_distribution


def render_citations_page(
    *,
    dff: DataFrameLike,
    df_full: DataFrameLike,
    xuid: str | None,
    db_path: str,
    db_key: tuple[int, int] | None,
    top_medals_fn: Callable[
        [str, str, list[str], int | None, tuple[int, int] | None], list[tuple[int, int]] | None
    ],
) -> None:
    """Rend la page Citations (Commendations H5G + Médailles HI).

    Parameters
    ----------
    dff : DataFrameLike
        DataFrame filtré des matchs.
    df_full : DataFrameLike
        DataFrame complet (non filtré) pour calculer les deltas.
    xuid : str | None
        XUID du joueur principal.
    db_path : str
        Chemin vers la base de données.
    db_key : tuple[int, int] | None
        Clé de cache pour la base de données.
    top_medals_fn : Callable
        Fonction pour récupérer les top médailles (signature: db_path, xuid, match_ids, top_n, db_key).
    """
    # Normaliser en Polars dès l'entrée
    dff = ensure_polars(dff)
    df_full = ensure_polars(df_full)
    # Protection contre les DataFrames vides
    if dff.is_empty():
        st.warning(t("no_matches"))
        return

    xuid_clean = str(xuid or "").strip()

    # Extraire les match_ids pour les filtres et le delta
    filtered_match_ids: list[str] | None = None
    all_match_ids: list[str] | None = None
    is_filtered = len(dff) < len(df_full) if not df_full.is_empty() else False

    if xuid_clean:
        all_match_ids = (
            df_full.select(pl.col("match_id").drop_nulls().cast(pl.String)).to_series().to_list()
        )
        if is_filtered:
            filtered_match_ids = (
                dff.select(pl.col("match_id").drop_nulls().cast(pl.String)).to_series().to_list()
            )

    # 1) Commendations Halo 5 (via CitationEngine + match_citations)
    st.subheader(t("citations_halo5_title"))

    # Compter les citations totales
    total_citations_count = 0
    if xuid_clean and db_path:
        try:
            from src.analysis.citations.engine import CitationEngine

            engine = CitationEngine(db_path, xuid_clean)
            citations_full = engine.aggregate_for_display(match_ids=None)
            total_citations_count = sum(citations_full.values())
        except Exception:
            total_citations_count = 0

    # Afficher les metrics
    cols_metrics = st.columns(3)
    with cols_metrics[0]:
        st.metric(t("cit_obtained"), f"{total_citations_count:,}")
    with cols_metrics[1]:
        st.metric(t("cit_matches_analyzed"), len(dff) if not dff.is_empty() else 0)

    render_h5g_commendations_section(
        db_path=db_path,
        xuid=xuid_clean,
        filtered_match_ids=filtered_match_ids,
        all_match_ids=all_match_ids,
    )
    st.divider()

    # 2) Médailles (Halo Infinite) - Agrégation des médailles pour la grille
    counts_by_medal: dict[int, int] = {}
    if not dff.is_empty() and xuid_clean:
        match_ids = (
            dff.select(pl.col("match_id").drop_nulls().cast(pl.String)).to_series().to_list()
        )
        with st.spinner(t("tm_computing_medals_all")):
            top_all = top_medals_fn(db_path, xuid_clean, match_ids, top_n=None, db_key=db_key)
        try:
            counts_by_medal = {int(nid): int(cnt) for nid, cnt in (top_all or [])}
        except Exception:
            counts_by_medal = {}

    # 2) Médailles (Halo Infinite) - Affiche TOUJOURS toutes les médailles.
    st.subheader(t("citations_medals_title"))

    # Calculer les totaux pour les médailles
    total_medals_distinct = len(counts_by_medal)
    total_medals_count = sum(counts_by_medal.values())

    # Afficher les metrics
    cols_medals = st.columns(3)
    with cols_medals[0]:
        st.metric(t("cit_distinct_medals"), total_medals_distinct)
    with cols_medals[1]:
        st.metric(t("cit_total_medals"), f"{total_medals_count:,}")
    with cols_medals[2]:
        st.metric(t("cit_matches_analyzed"), len(dff) if not dff.is_empty() else 0)

    st.caption(t("citations_medals_caption"))
    if dff.is_empty():
        st.info(t("no_data_filter"))
    else:
        top = sorted(counts_by_medal.items(), key=lambda kv: kv[1], reverse=True)

        if not top:
            st.info(t("citations_no_medals"))
        else:
            _fr_map, _en_map = load_medal_name_maps()
            _medal_map = {
                **{str(k): v for k, v in _en_map.items()},
                **{str(k): v for k, v in _fr_map.items()},
            }
            md = pl.DataFrame(
                {"name_id": [t[0] for t in top], "count": [t[1] for t in top]}
            ).with_columns(
                pl.col("name_id")
                .cast(pl.Utf8)
                .replace_strict(_medal_map, default=None, return_dtype=pl.Utf8)
                .fill_null(
                    pl.lit(t("mv_medal_fallback", n="") + " ") + pl.col("name_id").cast(pl.Utf8)
                )
                .alias("label")
            )
            md_desc = md.sort("count", descending=True)

            # === Graphique de distribution des médailles (Sprint 5.4.9) ===
            st.subheader(t("citations_medals_distribution"))

            # Préparer les données pour le graphique
            try:
                medal_names_dict = {
                    int(nid): medal_label(int(nid), lang=get_lang()) for nid, _ in top
                }
                fig_medals = plot_medals_distribution(
                    top,
                    medal_names_dict,
                    title=None,
                    top_n=25,
                    lang=get_lang(),
                )
                if fig_medals is not None:
                    st.plotly_chart(fig_medals, width="stretch", config=PLOTLY_STATIC_CONFIG)
                else:
                    st.info(t("insufficient_data_chart"))
            except Exception as e:
                st.warning(t("error_chart", error=e))

            st.divider()
            st.subheader(t("citations_medals_grid"))

            # Passer les deltas par médaille si filtré
            deltas = None
            if is_filtered:
                deltas = {int(nid): int(cnt) for nid, cnt in counts_by_medal.items()}
            render_medals_grid(
                md_desc.select("name_id", "count").to_dicts(),
                cols_per_row=8,
                deltas=deltas,
            )
