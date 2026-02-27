"""Section Participation au match - Radar unifié 6 axes.

Basé sur PersonalScores et match_stats. Un seul radar : Objectifs, Combat,
Support, Score, Impact, Survie. Réutilisable dans Mes coéquipiers.
"""

from __future__ import annotations

from typing import Any

import streamlit as st

from src.ui.i18n import t
from src.ui.streamlit_modern import fragment_if_available


@fragment_if_available
def render_participation_section(
    db_path: str,
    match_id: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
    *,
    match_row: dict[str, Any] | None = None,
) -> None:
    """Affiche la section Participation au match (radar unifié 6 axes).

    Utilise PersonalScoreAwards et match_stats pour Objectifs, Combat,
    Support, Score, Impact, Survie.

    Args:
        db_path: Chemin vers la base de données.
        match_id: ID du match.
        xuid: XUID du joueur.
        db_key: Clé de cache pour la DB.
        match_row: Ligne match_stats (pair_name, deaths, time_played_seconds)
                   pour Impact et Survie. Optionnel.
    """
    from src.data.repositories import DuckDBRepository
    from src.ui.components.radar_chart import create_participation_profile_radar
    from src.visualization.participation_radar import (
        compute_participation_profile,
        get_radar_axis_lines,
        get_radar_thresholds,
    )

    # Charger les données
    try:
        repo = DuckDBRepository(db_path, xuid)
        if not repo.has_personal_score_awards():
            return

        df = repo.load_personal_score_awards_as_polars(match_id=match_id)

        if df.is_empty():
            return

    except Exception:
        return

    # Convertir match_row en dict si Series
    row_dict = None
    if match_row is not None:
        if hasattr(match_row, "to_dict"):
            row_dict = match_row.to_dict()
        elif isinstance(match_row, dict):
            row_dict = match_row

    # Profil de participation (seuils = meilleur match global si dispo)
    thresholds = get_radar_thresholds(db_path)
    profile = compute_participation_profile(
        df,
        match_row=row_dict,
        name=t("mvc_this_match"),
        color="#636EFA",
        pair_name=row_dict.get("pair_name") if row_dict else None,
        thresholds=thresholds,
    )

    st.subheader(f"🎯 {t('mvp_participation_title')}")

    col_radar, col_legend = st.columns([2, 1])
    with col_radar:
        try:
            fig = create_participation_profile_radar(
                [profile],
                title=t("mvp_participation_title"),
                height=380,
                radial_range=(0, 0.75),  # Échelle réduite pour agrandir le graphe
            )
            if fig is not None:
                st.plotly_chart(fig, width="stretch", config={"staticPlot": True})
            else:
                st.info(t("insufficient_data_chart"))
        except Exception as e:
            st.warning(t("error_chart", error=e))
    with col_legend:
        st.markdown(f"**{t('mvp_axes_label')}**")
        for line in get_radar_axis_lines():
            st.markdown(line)


def render_participation_comparison(
    db_path: str,
    match_ids: list[str],
    xuid: str,
    labels: list[str] | None = None,
    colors: list[str] | None = None,
    match_rows: list[dict[str, Any]] | None = None,
) -> None:
    """Affiche une comparaison de participation entre plusieurs matchs.

    Args:
        db_path: Chemin vers la base de données.
        match_ids: Liste des IDs de matchs à comparer.
        xuid: XUID du joueur.
        labels: Labels pour chaque match (optionnel).
        colors: Couleurs pour chaque match (optionnel).
        match_rows: Lignes match_stats pour chaque match_id (optionnel).
                    Si absent, Impact et Survie utilisent des valeurs par défaut.
    """
    from src.data.repositories import DuckDBRepository
    from src.ui.components.radar_chart import create_participation_profile_radar
    from src.visualization.participation_radar import (
        compute_participation_profile,
        get_radar_axis_lines,
        get_radar_thresholds,
    )

    if not match_ids:
        return

    default_colors = ["#636EFA", "#EF553B", "#00CC96", "#AB63FA", "#FFA15A"]
    if labels is None:
        labels = [f"Match {i + 1}" for i in range(len(match_ids))]
    if colors is None:
        colors = default_colors[: len(match_ids)]

    try:
        repo = DuckDBRepository(db_path, xuid)
        if not repo.has_personal_score_awards():
            return

        thresholds = get_radar_thresholds(db_path)
        profiles = []
        for i, mid in enumerate(match_ids):
            df = repo.load_personal_score_awards_as_polars(match_id=mid)
            if df.is_empty():
                continue

            row = None
            if match_rows and i < len(match_rows):
                r = match_rows[i]
                row = r.to_dict() if hasattr(r, "to_dict") else (r if isinstance(r, dict) else None)

            profile = compute_participation_profile(
                df,
                match_row=row,
                name=labels[i] if i < len(labels) else f"Match {i + 1}",
                color=colors[i] if i < len(colors) else None,
                pair_name=row.get("pair_name") if row else None,
                thresholds=thresholds,
            )
            profiles.append(profile)

        if not profiles:
            return

        st.subheader(f"📊 {t('mvp_comparison_title')}")
        col_radar, col_legend = st.columns([2, 1])
        with col_radar:
            try:
                fig = create_participation_profile_radar(profiles, title="", height=400)
                if fig is not None:
                    st.plotly_chart(fig, width="stretch", config={"staticPlot": True})
                else:
                    st.info(t("insufficient_data_chart"))
            except Exception as e:
                st.warning(t("error_chart", error=e))
        with col_legend:
            st.markdown(f"**{t('mvp_axes_label')}**")
            for line in get_radar_axis_lines():
                st.markdown(line)

    except Exception:
        pass


__all__ = [
    "render_participation_section",
    "render_participation_comparison",
]
