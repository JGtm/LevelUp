"""Onglet Impact pour la page Coéquipiers.

Extrait de teammates.py (Sprint 16 — refactoring Phase A).
Heatmap des événements clés + tableau de ranking MVP/Boulet.
"""

from __future__ import annotations

import polars as pl
import streamlit as st

from src.ui.i18n import t

from src.analysis.friends_impact import (
    build_impact_matrix,
    get_all_impact_events,
)
from src.data.repositories import DuckDBRepository
from src.utils.paths import get_shared_matches_path_from_player
from src.visualization.friends_impact_heatmap import (
    build_impact_ranking_df,
    count_events_by_player,
    plot_friends_impact_heatmap,
    render_impact_summary_stats,
)


def _ensure_shared_attached(conn, player_db_path: str) -> str | None:
    """Attache shared_matches.duckdb si nécessaire.

    Returns:
        Nom de l'alias de la DB shared, ou None si échec.
    """
    # Chercher si shared est déjà attaché
    try:
        dbs = conn.execute("SELECT database_name, path FROM duckdb_databases()").fetchall()
        for db_name, db_path_val in dbs:
            if db_path_val and "shared_matches.duckdb" in str(db_path_val).lower():
                return db_name
            if db_name and "shared" in db_name.lower():
                # Vérifier que cette DB a bien match_participants
                try:
                    conn.execute(f"SELECT 1 FROM {db_name}.match_participants LIMIT 1")
                    return db_name
                except Exception:
                    continue
    except Exception:
        pass

    # Pas trouvé, attacher
    shared_db = get_shared_matches_path_from_player(player_db_path)
    if not shared_db or not shared_db.exists():
        return None

    try:
        conn.execute(f"ATTACH '{shared_db}' AS shared (READ_ONLY)")
        return "shared"
    except Exception:
        return None


def _load_highlight_events(
    conn,
    match_ids: list[str],
    shared_alias: str,
) -> pl.DataFrame | None:
    """Charge les événements highlight depuis shared_matches.duckdb.

    Args:
        conn: Connexion DuckDB.
        match_ids: Liste des match_id.
        shared_alias: Alias de la DB shared attachée.

    Returns:
        DataFrame Polars des événements, ou None si indisponible.
    """
    # V5 : highlight_events est dans shared_matches.duckdb
    try:
        conn.execute(f"SELECT 1 FROM {shared_alias}.highlight_events LIMIT 1")
    except Exception:
        return None

    events_query = f"""
        SELECT match_id, xuid::TEXT as xuid, gamertag, event_type, time_ms
        FROM {shared_alias}.highlight_events
        WHERE match_id IN ({{}})
    """.format(", ".join(["?" for _ in match_ids]))

    events_result = conn.execute(events_query, match_ids).fetchall()

    if not events_result:
        return pl.DataFrame()

    return pl.DataFrame(
        {
            "match_id": [str(r[0]) for r in events_result],
            "xuid": [str(r[1]) for r in events_result],
            "gamertag": [r[2] or "Unknown" for r in events_result],
            "event_type": [r[3] for r in events_result],
            "time_ms": [int(r[4] or 0) for r in events_result],
        }
    )


def _load_match_outcomes(
    conn,
    match_ids: list[str],
    xuid: str,
    shared_alias: str,
) -> pl.DataFrame:
    """Charge les outcomes des matchs depuis shared.match_participants.

    Args:
        conn: Connexion DuckDB.
        match_ids: Liste des match_id.
        xuid: XUID du joueur principal.
        shared_alias: Alias de la DB shared attachée.

    Returns:
        DataFrame avec match_id et outcome du joueur principal.
    """
    # V5 : match_stats n'existe plus, lire depuis shared.match_participants
    matches_query = f"""
        SELECT match_id, outcome
        FROM {shared_alias}.match_participants
        WHERE match_id IN ({{}}) AND xuid = ?
    """.format(", ".join(["?" for _ in match_ids]))

    matches_result = conn.execute(matches_query, match_ids + [xuid]).fetchall()

    return pl.DataFrame(
        {
            "match_id": [str(r[0]) for r in matches_result],
            "outcome": [int(r[1] or 0) for r in matches_result],
        }
    )


def _render_impact_stats(
    first_bloods: dict,
    clutch_finishers: dict,
    last_casualties: dict,
) -> None:
    """Affiche les métriques résumées d'impact."""
    stats = render_impact_summary_stats(first_bloods, clutch_finishers, last_casualties)
    cols = st.columns(4)
    cols[0].metric("🟢 Premier Sang", stats["total_fb"])
    cols[1].metric("🟡 Finisseur", stats["total_clutch"])
    cols[2].metric("🔴 Boulet", stats["total_casualty"])
    cols[3].metric("📊 Matchs analysés", stats["total_matches"])


def _render_ranking_table(
    scores: dict,
    first_bloods: dict,
    clutch_finishers: dict,
    last_casualties: dict,
) -> None:
    """Affiche le tableau de classement MVP/Boulet."""
    fb_counts = count_events_by_player(first_bloods)
    clutch_counts = count_events_by_player(clutch_finishers)
    casualty_counts = count_events_by_player(last_casualties)

    ranking_df = build_impact_ranking_df(
        scores,
        first_blood_counts=fb_counts,
        clutch_counts=clutch_counts,
        casualty_counts=casualty_counts,
    )

    if not ranking_df.is_empty():
        # Sprint 19 : renommer les colonnes en Polars sans conversion Pandas
        display_df = ranking_df.rename(
            dict(
                zip(
                    ranking_df.columns,
                    [
                        "Rang",
                        "Joueur",
                        "Score",
                        "⚡ Premier Sang",
                        "🎯 Finisseur",
                        "💀 Boulet",
                        "Badge",
                    ],
                    strict=False,
                )
            )
        )
        st.dataframe(display_df, width="stretch", hide_index=True)

        mvp = ranking_df[0, "gamertag"] if len(ranking_df) > 0 else None
        boulet = (
            ranking_df[-1, "gamertag"]
            if len(ranking_df) > 1 and ranking_df[-1, "score"] < 0
            else None
        )

        summary_cols = st.columns(2)
        if mvp:
            summary_cols[0].success(f"**🏆 Brute de la Soirée :** {mvp}")
        if boulet:
            summary_cols[1].error(f"**🍌 Maillon Faible :** {boulet}")


def render_impact_taquinerie(
    db_path: str,
    xuid: str,
    match_ids: list[str],
    friend_xuids: list[str],
    db_key: tuple[int, int] | None = None,
) -> None:
    """Affiche l'onglet Impact (Sprint 12).

    Args:
        db_path: Chemin vers la DB principale.
        xuid: XUID du joueur principal.
        match_ids: Liste des match_id à analyser.
        friend_xuids: Liste des XUIDs des coéquipiers sélectionnés.
        db_key: Clé de cache (optionnel).
    """
    st.subheader(t("tm_impact_header"))

    if len(friend_xuids) < 2:
        st.info(t("tm_impact_select_two"))
        return

    if not match_ids:
        st.warning(t("tm_impact_no_matches"))
        return

    st.caption("⚡ Premier sang | 🎯 Finisseur | 💀 Boulet | 🐌 Plus lent | 🪦 Première victime")

    try:
        repo = DuckDBRepository(db_path, xuid.strip())
        conn = repo._get_connection()

        # Attacher shared_matches.duckdb
        shared_alias = _ensure_shared_attached(conn, db_path)
        if not shared_alias:
            st.warning(
                "Impossible d'accéder à shared_matches.duckdb. "
                "Cette fonctionnalité nécessite l'architecture v5."
            )
            return

        # Charger les événements depuis shared_matches
        events_df = _load_highlight_events(conn, match_ids, shared_alias)
        if events_df is None:
            st.info(
                "Les données d'événements (highlight_events) ne sont pas disponibles. "
                "Cette fonctionnalité nécessite une synchronisation avec les détails de matchs."
            )
            return
        if events_df.is_empty():
            st.info(t("tm_impact_no_events_matches"))
            return

        # Charger les outcomes depuis shared_matches
        matches_df = _load_match_outcomes(conn, match_ids, xuid, shared_alias)

        # Inclure le joueur principal + tous les amis sélectionnés
        all_friend_xuids = {str(x) for x in friend_xuids}
        all_friend_xuids.add(str(xuid).strip())

        # Calculer les événements d'impact
        (
            first_bloods,
            clutch_finishers,
            last_casualties,
            last_group_kills,
            first_group_deaths,
            scores,
        ) = get_all_impact_events(events_df, matches_df, friend_xuids=all_friend_xuids)

        if not scores:
            st.info(t("tm_impact_no_events_players"))
            return

        gamertags = list(scores.keys())
        sorted_match_ids = sorted(
            {
                m
                for m in match_ids
                if m
                in set(
                    list(first_bloods.keys())
                    + list(clutch_finishers.keys())
                    + list(last_casualties.keys())
                    + list(last_group_kills.keys())
                    + list(first_group_deaths.keys())
                )
            }
        )

        # Créer un dict des outcomes pour la heatmap
        match_outcomes = {}
        if not matches_df.is_empty():
            for row in matches_df.iter_rows(named=True):
                match_outcomes[str(row["match_id"])] = int(row["outcome"])

        # Construire la matrice d'impact
        impact_matrix = build_impact_matrix(
            first_bloods,
            clutch_finishers,
            last_casualties,
            last_group_kills,
            first_group_deaths,
            match_ids=sorted_match_ids,
            gamertags=gamertags,
            match_outcomes=match_outcomes,
        )

        # Métriques résumées (masquées à la demande de l'utilisateur)
        # _render_impact_stats(first_bloods, clutch_finishers, last_casualties)

        # Heatmap
        st.subheader(t("tm_impact_heatmap"))
        fig = plot_friends_impact_heatmap(
            impact_matrix,
            title=None,
            max_matches=len(sorted_match_ids),
        )
        st.plotly_chart(fig, width="stretch", config={"staticPlot": True})

        # Tableau de ranking
        st.subheader(t("tm_impact_ranking"))
        _render_ranking_table(scores, first_bloods, clutch_finishers, last_casualties)

    except Exception as e:
        st.warning(t("error_chart", error=e))
