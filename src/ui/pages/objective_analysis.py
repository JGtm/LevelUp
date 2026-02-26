"""Page d'analyse des objectifs.

Sprint 7: Page dédiée à l'analyse de la participation aux objectifs
et à la valorisation des joueurs support.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

import streamlit as st

from src.analysis.objective_participation import (
    compute_award_frequency_polars,
    compute_objective_summary_by_match_polars,
)
from src.ui.i18n import t
from src.ui.streamlit_modern import fragment_if_available
from src.visualization.objective_charts import (
    plot_assist_breakdown_pie,
    plot_objective_breakdown_bars,
    plot_objective_ratio_gauge,
    plot_objective_trend_over_time,
    plot_objective_vs_kills_scatter,
)

# Import conditionnel de Polars
try:
    import polars as pl

    POLARS_AVAILABLE = True
except ImportError:
    POLARS_AVAILABLE = False
    pl = None

if TYPE_CHECKING:
    from src.data.repositories.duckdb_repo import DuckDBRepository


def _format_score(value: float | None) -> str:
    """Formate un score pour l'affichage."""
    if value is None:
        return "—"
    return f"{value:,.0f}"


def _format_ratio(value: float | None) -> str:
    """Formate un ratio en pourcentage."""
    if value is None:
        return "—"
    return f"{value * 100:.1f}%"


@fragment_if_available
def render_objective_analysis_page(
    repo: DuckDBRepository,
    xuid: str,
    *,
    match_ids: list[str] | None = None,
) -> None:
    """Affiche la page d'analyse des objectifs.

    Cette page permet de :
    - Voir la contribution du joueur aux objectifs vs kills
    - Analyser la répartition du score par catégorie
    - Comparer avec les autres joueurs rencontrés
    - Identifier les forces du joueur (support vs slayer)

    Args:
        repo: Repository DuckDB pour charger les données.
        xuid: XUID du joueur principal.
        match_ids: Liste optionnelle de match_ids à analyser (sinon tous).
    """
    st.title(t("obj_analysis_title"))
    st.caption(
        "Analysez votre contribution aux objectifs de jeu et découvrez votre profil de joueur."
    )

    if not POLARS_AVAILABLE:
        st.error(t("obj_polars_missing"))
        return

    # ══════════════════════════════════════════════════════════════════════════
    # Chargement des données
    # ══════════════════════════════════════════════════════════════════════════
    with st.spinner(t("obj_loading")):
        # Charger les personal_score_awards
        try:
            if match_ids:
                # Paramétrage SQL sécurisé avec placeholders
                placeholders = ", ".join("?" * len(match_ids))
                awards_query = f"""
                    SELECT * FROM personal_score_awards
                    WHERE match_id IN ({placeholders})
                """
                match_query = f"""
                    SELECT * FROM match_stats
                    WHERE match_id IN ({placeholders})
                    ORDER BY start_time ASC
                """
                awards_df = repo.query_df(awards_query, match_ids)
                match_stats_df = repo.query_df(match_query, match_ids)
            else:
                awards_query = "SELECT * FROM personal_score_awards"
                match_query = "SELECT * FROM match_stats ORDER BY start_time ASC"
                awards_df = repo.query_df(awards_query)
                match_stats_df = repo.query_df(match_query)
        except Exception as e:
            st.error(t("error_loading", error=e))
            st.info(
                "💡 Les tables `personal_score_awards` peuvent ne pas exister. "
                "Lancez une synchronisation pour les créer."
            )
            return

    if awards_df.is_empty():
        st.warning(
            "⚠️ Aucune donnée de score personnel disponible. "
            "Synchronisez vos matchs pour obtenir ces données."
        )
        return

    # ══════════════════════════════════════════════════════════════════════════
    # Filtrage par joueur
    # ══════════════════════════════════════════════════════════════════════════
    my_awards_df = (
        awards_df.filter(pl.col("xuid") == xuid) if "xuid" in awards_df.columns else awards_df
    )

    if my_awards_df.is_empty():
        st.warning(t("obj_no_player_data", xuid=xuid))
        return

    # ══════════════════════════════════════════════════════════════════════════
    # Section 1: Vue d'ensemble
    # ══════════════════════════════════════════════════════════════════════════
    st.markdown("---")
    st.markdown(f"## 🎯 {t('obj_overview_title')}")

    # Calculer les métriques globales
    total_objective = (
        my_awards_df.filter(pl.col("score_category").is_in(["objective", "mode"]))
        .select(pl.col("points").sum())
        .item()
    ) or 0

    total_kill = (
        my_awards_df.filter(pl.col("score_category") == "kill")
        .select(pl.col("points").sum())
        .item()
    ) or 0

    total_assist = (
        my_awards_df.filter(pl.col("score_category") == "assist")
        .select(pl.col("points").sum())
        .item()
    ) or 0

    total_all = total_objective + total_kill + total_assist
    objective_ratio = total_objective / total_all if total_all > 0 else 0

    # Métriques en colonnes
    col1, col2, col3, col4 = st.columns(4)

    with col1:
        st.metric(
            label=t("obj_score_label"),
            value=_format_score(total_objective),
            help="Points gagnés sur les objectifs de jeu",
        )

    with col2:
        st.metric(
            label=t("obj_frag_score_label"),
            value=_format_score(total_kill),
            help="Points gagnés avec les éliminations",
        )

    with col3:
        st.metric(
            label=t("obj_assist_score_label"),
            value=_format_score(total_assist),
            help="Points gagnés avec les assistances",
        )

    with col4:
        st.metric(
            label=t("obj_ratio_label"),
            value=_format_ratio(objective_ratio),
            help="Part des objectifs dans le score total",
        )

    # Indicateur de profil
    if objective_ratio >= 0.4:
        profile = "🛡️ Joueur Support/Objectif"
        profile_desc = "Vous contribuez fortement aux objectifs de l'équipe."
    elif objective_ratio >= 0.2:
        profile = "⚔️ Joueur Polyvalent"
        profile_desc = "Bon équilibre entre kills et objectifs."
    else:
        profile = "🎯 Joueur Slayer"
        profile_desc = "Vous excellez dans les éliminations."

    st.info(f"**{t('obj_profile_label')}:** {profile}\n\n{profile_desc}")

    # ══════════════════════════════════════════════════════════════════════════
    # Section 2: Graphiques principaux
    # ══════════════════════════════════════════════════════════════════════════
    st.markdown("---")
    st.markdown(f"## {t('obj_analysis_detailed')}")

    tab_scatter, tab_breakdown, tab_trend = st.tabs(
        [
            t("obj_tab_scatter"),
            t("obj_tab_breakdown"),
            t("obj_tab_trend"),
        ]
    )

    with tab_scatter:
        st.markdown(f"### {t('obj_correlation_title')}")
        st.caption(
            "Chaque point représente un match. "
            "Les points au-dessus de la tendance indiquent une meilleure contribution aux objectifs."
        )
        try:
            fig_scatter = plot_objective_vs_kills_scatter(
                my_awards_df,
                match_stats_df,
                title=None,
            )
            if fig_scatter is not None:
                st.plotly_chart(fig_scatter, width="stretch", config={"displayModeBar": False})
            else:
                st.info(t("insufficient_data_chart"))
        except Exception as e:
            st.warning(t("error_chart", error=e))

    with tab_breakdown:
        col_bars, col_gauge = st.columns([2, 1])

        with col_bars:
            st.markdown(f"### {t('obj_breakdown_title')}")
            try:
                fig_bars = plot_objective_breakdown_bars(
                    my_awards_df,
                    xuid=xuid,
                    title=None,
                )
                if fig_bars is not None:
                    st.plotly_chart(fig_bars, width="stretch", config={"staticPlot": True})
                else:
                    st.info(t("insufficient_data_chart"))
            except Exception as e:
                st.warning(t("error_chart", error=e))

        with col_gauge:
            st.markdown(f"### {t('obj_ratio_label')}")
            try:
                fig_gauge = plot_objective_ratio_gauge(
                    objective_ratio,
                    title="% du score sur objectifs",
                )
                if fig_gauge is not None:
                    st.plotly_chart(fig_gauge, width="stretch", config={"staticPlot": True})
                else:
                    st.info(t("insufficient_data_chart"))
            except Exception as e:
                st.warning(t("error_chart", error=e))

    with tab_trend:
        st.markdown(f"### {t('obj_trend_title')}")

        # Calculer le résumé par match
        summary_df = compute_objective_summary_by_match_polars(my_awards_df, xuid)

        if not summary_df.is_empty():
            # Joindre avec match_stats pour avoir start_time
            summary_with_time = (
                match_stats_df.select(["match_id", "start_time"])
                .join(summary_df, on="match_id", how="inner")
                .sort("start_time")
            )

            try:
                fig_trend = plot_objective_trend_over_time(
                    summary_with_time,
                    title=None,
                )
                if fig_trend is not None:
                    st.plotly_chart(fig_trend, width="stretch", config={"displayModeBar": False})
                else:
                    st.info(t("insufficient_data_chart"))
            except Exception as e:
                st.warning(t("error_chart", error=e))
        else:
            st.info(t("insufficient_data_chart"))

    # ══════════════════════════════════════════════════════════════════════════
    # Section 3: Analyse des assistances
    # ══════════════════════════════════════════════════════════════════════════
    st.markdown("---")
    st.markdown("## 🤝 Analyse des Assistances")
    st.caption(t("obj_assists_caption"))

    # Calculer la décomposition des assistances globale
    assist_awards = my_awards_df.filter(pl.col("score_category") == "assist")

    if not assist_awards.is_empty():
        # Agréger par type d'award
        assist_by_type = (
            assist_awards.group_by("award_name")
            .agg(
                [
                    pl.col("points").sum().alias("total_points"),
                    pl.count().alias("count"),
                ]
            )
            .sort("total_points", descending=True)
        )

        col_pie, col_table = st.columns([1, 1])

        with col_pie:
            # Créer un breakdown simplifié pour le pie chart
            kill_assists = assist_awards.filter(
                pl.col("award_name").str.contains("(?i)kill")
            ).height
            mark_assists = assist_awards.filter(
                pl.col("award_name").str.contains("(?i)mark|spot|tag")
            ).height
            emp_assists = assist_awards.filter(
                pl.col("award_name").str.contains("(?i)emp|disable")
            ).height
            other_assists = assist_awards.height - kill_assists - mark_assists - emp_assists

            breakdown = {
                "kill_assists": kill_assists,
                "mark_assists": mark_assists,
                "emp_assists": emp_assists,
                "other_assists": max(0, other_assists),
            }

            fig_pie = plot_assist_breakdown_pie(
                breakdown,
                title="Types d'Assistances",
            )
            st.plotly_chart(fig_pie, width="stretch", config={"staticPlot": True})

        with col_table:
            st.markdown(f"### {t('obj_assist_detail')}")
            if not assist_by_type.is_empty():
                # Convertir en pandas pour affichage
                assist_table = assist_by_type.to_pandas()
                assist_table.columns = ["Type d'assistance", "Points", "Nombre"]
                st.dataframe(
                    assist_table,
                    width="stretch",
                    hide_index=True,
                )
    else:
        st.info(t("obj_no_assists"))

    # ══════════════════════════════════════════════════════════════════════════
    # Section 4: Awards les plus fréquents
    # ══════════════════════════════════════════════════════════════════════════
    st.markdown("---")
    st.markdown(f"## {t('obj_awards_frequent')}")

    col_obj_awards, col_all_awards = st.columns(2)

    with col_obj_awards:
        st.markdown("### Objectifs & Mode")
        obj_freq = compute_award_frequency_polars(
            my_awards_df.filter(pl.col("score_category").is_in(["objective", "mode"])),
            top_n=10,
        )
        if not obj_freq.is_empty():
            freq_table = obj_freq.to_pandas()
            freq_table.columns = ["Award", "Points", "Occurrences"]
            st.dataframe(freq_table, width="stretch", hide_index=True)
        else:
            st.info(t("obj_no_awards"))

    with col_all_awards:
        st.markdown("### Tous les Awards")
        all_freq = compute_award_frequency_polars(my_awards_df, top_n=10)
        if not all_freq.is_empty():
            all_table = all_freq.to_pandas()
            all_table.columns = ["Award", "Points", "Occurrences"]
            st.dataframe(all_table, width="stretch", hide_index=True)
        else:
            st.info(t("obj_no_awards_generic"))

    # ══════════════════════════════════════════════════════════════════════════
    # Section 5: Comparaison avec les autres joueurs
    # ══════════════════════════════════════════════════════════════════════════
    st.markdown("---")
    st.markdown("## 👥 Comparaison avec les Adversaires")
    st.caption(t("obj_top_opponents_caption"))

    # Note: Cette fonctionnalité nécessite d'avoir les awards de tous les joueurs
    # Pour l'instant, on affiche un placeholder
    with st.expander(t("obj_comparison_coming_soon"), expanded=False):
        st.info(
            "Cette fonctionnalité nécessite d'avoir synchronisé les données de tous les joueurs "
            "d'un match. Elle sera disponible dans une prochaine version."
        )

        # Placeholder pour le graphique
        # rankings = rank_players_by_objective_contribution_polars(awards_df, top_n=10)
        # fig_rankings = plot_top_players_objective_bars(rankings)
        # st.plotly_chart(fig_rankings, width="stretch")

    # ══════════════════════════════════════════════════════════════════════════
    # Section 6: Conseils personnalisés
    # ══════════════════════════════════════════════════════════════════════════
    st.markdown("---")
    st.markdown(f"## {t('obj_tips')}")

    if objective_ratio < 0.15:
        st.warning(
            "🎯 **Pensez aux objectifs !**\n\n"
            "Votre ratio objectifs est faible. Dans les modes objectifs (CTF, Strongholds, etc.), "
            "contribuer aux objectifs rapporte plus de points à l'équipe."
        )
    elif objective_ratio > 0.5:
        st.success(
            "🛡️ **Excellent joueur d'objectif !**\n\n"
            "Vous êtes un pilier pour votre équipe sur les objectifs. "
            "Continuez à jouer le jeu d'équipe !"
        )

    # Conseil basé sur les assistances
    if total_assist > total_kill * 0.3:
        st.info(
            "🤝 **Grand fournisseur d'assists !**\n\n"
            "Vous contribuez beaucoup aux éliminations de vos coéquipiers. "
            "Pensez à utiliser le ping et les EMP pour maximiser cet impact."
        )


def render_objective_analysis_page_from_session_state() -> None:
    """Version de la page utilisant le session_state Streamlit.

    Utilisée quand la page est appelée depuis le menu principal.
    """
    # Récupérer les informations depuis session_state
    db_path = st.session_state.get("db_path")
    xuid = st.session_state.get("player_xuid")

    if not db_path or not xuid:
        st.error(t("obj_no_player_selected"))
        return

    # Créer le repository
    from src.data.repositories.duckdb_repo import DuckDBRepository

    try:
        repo = DuckDBRepository(db_path, xuid)
        render_objective_analysis_page(repo, xuid)
    except Exception as e:
        st.error(t("error_loading", error=e))
