"""Requête SQL — premiers événements de match (frag / mort) par joueur.

Module privé du service Coéquipiers : ne pas importer directement depuis l'UI.
"""

from __future__ import annotations

import logging

import polars as pl

logger = logging.getLogger(__name__)


def query_first_events(
    conn: object,
    xuids: list[str],
    match_ids: list[str],
) -> pl.DataFrame:
    """Charge le premier frag et la première mort de chaque joueur par match.

    Requête shared.highlight_events filtrée sur les xuids et match_ids fournis.
    Résultat trié par start_time ascendant pour faciliter le calcul de tendance.

    Args:
        conn: Connexion DuckDB active (schéma shared disponible).
        xuids: XUIDs des joueurs à inclure.
        match_ids: Identifiants des matchs à interroger.

    Returns:
        DataFrame avec colonnes : match_id, xuid, start_time,
        first_kill_s (float, secondes), first_death_s (float, secondes).
        Valeurs NULL si aucun event du type trouvé pour ce joueur/match.
    """
    from src.data.repositories._arrow_bridge import result_to_polars

    if not xuids or not match_ids:
        return pl.DataFrame()

    xu_ph = ", ".join(["?" for _ in xuids])
    mi_ph = ", ".join(["?" for _ in match_ids])
    # countdown_s = durée du compte à rebours pré-match (0 si données absentes).
    # first_kill_s / first_death_s sont soustraits du countdown pour obtenir
    # le temps écoulé depuis le vrai début du gameplay, pas depuis le début
    # de la timeline Halo (qui inclut le lobby/countdown).
    query = f"""
        SELECT
            e.match_id,
            e.xuid,
            ANY_VALUE(r.start_time) AS start_time,
            GREATEST(
                MIN(CASE WHEN LOWER(e.event_type) = 'kill'  THEN e.time_ms END) / 1000.0
                    - GREATEST(
                        COALESCE(ANY_VALUE(r.duration_seconds), 0)
                        - COALESCE(ANY_VALUE(r.playable_duration_seconds),
                                   ANY_VALUE(r.duration_seconds), 0),
                        0
                    ),
                0
            ) AS first_kill_s,
            GREATEST(
                MIN(CASE WHEN LOWER(e.event_type) = 'death' THEN e.time_ms END) / 1000.0
                    - GREATEST(
                        COALESCE(ANY_VALUE(r.duration_seconds), 0)
                        - COALESCE(ANY_VALUE(r.playable_duration_seconds),
                                   ANY_VALUE(r.duration_seconds), 0),
                        0
                    ),
                0
            ) AS first_death_s
        FROM shared.highlight_events e
        JOIN shared.match_registry r ON e.match_id = r.match_id
        WHERE e.xuid IN ({xu_ph})
          AND e.match_id IN ({mi_ph})
        GROUP BY e.match_id, e.xuid
        ORDER BY start_time ASC
    """
    try:
        result = conn.execute(query, [*xuids, *match_ids])  # type: ignore[union-attr]
        return result_to_polars(result)
    except Exception:
        logger.debug("query_first_events: erreur requête highlight_events", exc_info=True)
        return pl.DataFrame()
