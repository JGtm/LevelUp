"""Requêtes SQL pour charger l'historique complet d'un coéquipier.

Utilisé pour calculer les records historiques indépendamment du filtre
de session ou de la sélection de matchs actuellement affichés.
"""

from __future__ import annotations

import polars as pl


def query_teammate_full_history(conn: object, xuid: str) -> pl.DataFrame:
    """Charge l'historique complet d'un joueur depuis shared.match_participants.

    Contrairement à _query_teammate_shared_stats, aucun filtre match_id n'est
    appliqué — retourne tous les matchs du joueur, avec pair_name pour permettre
    le filtrage par catégorie de mode lors du calcul des records.

    Args:
        conn: Connexion DuckDB active (shared attaché).
        xuid: XUID du joueur.

    Returns:
        DataFrame Polars avec toutes les colonnes nécessaires aux records.
    """
    from src.data.repositories._arrow_bridge import result_to_polars

    query = """
        SELECT
            p.match_id, r.start_time,
            COALESCE(r.pair_name, '') AS pair_name,
            COALESCE(r.map_name, '') AS map_name,
            COALESCE(r.map_name_fr, r.map_name, '') AS map_name_fr,
            COALESCE(r.map_name_fr, r.map_name, '') AS map_ui,
            p.kda AS ratio,
            COALESCE(p.max_killing_spree, 0) AS max_killing_spree,
            COALESCE(p.headshot_kills, 0) AS headshot_kills,
            COALESCE(p.avg_life_seconds, 0) AS average_life_seconds,
            COALESCE(r.duration_seconds, 0) AS time_played_seconds,
            COALESCE(p.kills, 0) AS kills,
            COALESCE(p.deaths, 0) AS deaths,
            COALESCE(p.assists, 0) AS assists,
            CASE WHEN p.shots_fired > 0
                THEN CAST(p.shots_hit AS FLOAT) * 100.0 / CAST(p.shots_fired AS FLOAT)
                ELSE NULL
            END AS accuracy,
            COALESCE(r.is_firefight, FALSE) AS is_firefight
        FROM shared.match_participants p
        JOIN shared.v_match_full r ON p.match_id = r.match_id
        WHERE p.xuid = ?
          AND COALESCE(r.is_firefight, FALSE) = FALSE
        ORDER BY r.start_time DESC
    """
    result = conn.execute(query, [xuid])
    return result_to_polars(result)
