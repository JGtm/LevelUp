"""
Composants UI pour les analytics DuckDB.
(UI components for DuckDB analytics)

HOW IT WORKS:
Ce module fournit des widgets Streamlit pré-configurés pour afficher
les analytics calculées via DuckDB sur les fichiers Parquet.

Ces composants sont opt-in et ne s'affichent que si :
1. enable_duckdb_analytics est True dans les settings
2. Les données Parquet existent pour le joueur

Usage:
    from src.ui.components.duckdb_analytics import (
        render_global_stats_card,
        render_kda_trend_chart,
        render_performance_by_map,
    )

    # Dans une page Streamlit
    render_global_stats_card(db_path, xuid, db_key)
"""

from __future__ import annotations

from typing import TYPE_CHECKING

import streamlit as st

if TYPE_CHECKING:
    pass


def _is_analytics_enabled() -> bool:
    """Vérifie si les analytics DuckDB sont activées."""
    try:
        settings = st.session_state.get("app_settings")
        if settings and hasattr(settings, "enable_duckdb_analytics"):
            return bool(settings.enable_duckdb_analytics)
    except Exception:
        pass
    return False


def render_global_stats_card(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
) -> bool:
    """
    Affiche une carte avec les stats globales via DuckDB.
    (Display a card with global stats via DuckDB)

    Returns:
        True si le composant a été affiché, False sinon.
    """
    if not _is_analytics_enabled():
        return False

    try:
        from src.ui.cache import cached_get_global_stats_duckdb

        stats = cached_get_global_stats_duckdb(db_path, xuid, db_key=db_key)

        if not stats:
            return False

        st.markdown("### 📊 Stats globales (DuckDB)")

        col1, col2, col3, col4 = st.columns(4)

        col1.metric(
            "Matchs",
            f"{stats['total_matches']:,}",
        )
        col2.metric(
            "Win Rate",
            f"{stats['win_rate']:.1f}%",
        )
        col3.metric(
            "KDA moyen",
            f"{stats['avg_kda']:.2f}",
        )
        col4.metric(
            "Précision",
            f"{stats['avg_accuracy']:.1f}%",
        )

        col5, col6, col7, col8 = st.columns(4)

        col5.metric(
            "Frags",
            f"{stats['total_kills']:,}",
        )
        col6.metric(
            "Deaths",
            f"{stats['total_deaths']:,}",
        )
        col7.metric(
            "Assists",
            f"{stats['total_assists']:,}",
        )
        col8.metric(
            "Temps joué",
            f"{stats['total_time_hours']:.0f}h",
        )

        return True

    except Exception as e:
        st.warning(f"Erreur analytics DuckDB: {e}")
        return False


def render_kda_trend_chart(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
    window_size: int = 20,
    last_n: int = 200,
) -> bool:
    """
    Affiche un graphique d'évolution du KDA via DuckDB.
    (Display KDA trend chart via DuckDB)

    Returns:
        True si le composant a été affiché, False sinon.
    """
    if not _is_analytics_enabled():
        return False

    try:
        import polars as pl

        from src.ui.cache import cached_get_kda_trend_duckdb

        data = cached_get_kda_trend_duckdb(
            db_path,
            xuid,
            window_size=window_size,
            last_n=last_n,
            db_key=db_key,
        )

        if not data:
            return False

        df = pl.DataFrame(data)

        if df.is_empty():
            return False

        st.markdown(f"### 📈 Évolution KDA (moyenne mobile {window_size} matchs)")

        # Graphique avec Streamlit natif (supporte Polars)
        chart_df = df.select(
            pl.col("match_number").alias("Match"),
            pl.col("rolling_kda").alias("KDA"),
        )

        st.line_chart(chart_df.to_pandas().set_index("Match"), width="stretch")

        # Stats récentes vs anciennes
        if len(df) >= window_size * 2:
            recent = df.head(window_size).select(pl.col("kda").mean()).item()
            older = df.tail(window_size).select(pl.col("kda").mean()).item()
            delta = recent - older

            col1, col2, col3 = st.columns(3)
            col1.metric(
                f"KDA (derniers {window_size})",
                f"{recent:.2f}",
                f"{delta:+.2f}" if abs(delta) > 0.01 else None,
            )
            col2.metric(
                f"KDA (plus anciens {window_size})",
                f"{older:.2f}",
            )
            col3.metric(
                "Tendance",
                "📈 En hausse" if delta > 0.1 else "📉 En baisse" if delta < -0.1 else "➡️ Stable",
            )

        return True

    except Exception as e:
        st.warning(f"Erreur graphique KDA: {e}")
        return False


def render_performance_by_map(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
    min_matches: int = 3,
    top_n: int = 10,
) -> bool:
    """
    Affiche un tableau des performances par carte via DuckDB.
    (Display performance by map table via DuckDB)

    Returns:
        True si le composant a été affiché, False sinon.
    """
    if not _is_analytics_enabled():
        return False

    try:
        import polars as pl

        from src.ui.cache import cached_get_performance_by_map_duckdb

        data = cached_get_performance_by_map_duckdb(
            db_path,
            xuid,
            min_matches=min_matches,
            db_key=db_key,
        )

        if not data:
            return False

        df = pl.DataFrame(data)

        if df.is_empty():
            return False

        # Trier par win_rate et limiter
        df = df.sort("win_rate", descending=True).head(top_n)

        st.markdown("### 🗺️ Performances par carte (DuckDB)")

        # Formater pour l'affichage
        display_df = df.select(
            pl.col("map_name").alias("Carte"),
            pl.col("total_matches").alias("Matchs"),
            pl.col("wins").alias("V"),
            pl.col("losses").alias("D"),
            (pl.col("win_rate").round(1).cast(pl.Utf8) + "%").alias("Win %"),
            pl.col("avg_kda").round(2).cast(pl.Utf8).alias("KDA"),
            pl.col("kd_ratio").round(2).cast(pl.Utf8).alias("K/D"),
        )

        st.dataframe(
            display_df,
            hide_index=True,
            width="stretch",
        )

        return True

    except Exception as e:
        st.warning(f"Erreur performances par carte: {e}")
        return False


def render_analytics_section(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
) -> bool:
    """
    Affiche une section complète d'analytics DuckDB.
    (Display complete DuckDB analytics section)

    Combine les différents composants dans une section cohérente.

    Returns:
        True si au moins un composant a été affiché, False sinon.
    """
    if not _is_analytics_enabled():
        return False

    displayed = False

    with st.expander("📊 Analytics avancées (DuckDB)", expanded=False):
        st.caption(
            "Ces statistiques sont calculées via DuckDB sur les fichiers Parquet. "
            "Performance 10-20x supérieure au système legacy."
        )

        if render_global_stats_card(db_path, xuid, db_key):
            displayed = True

        st.divider()

        if render_kda_trend_chart(db_path, xuid, db_key, last_n=100):
            displayed = True

        st.divider()

        if render_performance_by_map(db_path, xuid, db_key):
            displayed = True

        if not displayed:
            st.info(
                "Aucune donnée Parquet disponible. "
                "Lancez la migration dans Paramètres → Architecture de données."
            )

    return displayed
