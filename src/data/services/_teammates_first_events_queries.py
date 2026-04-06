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
    # countdown_s = durée du compte à rebours pré-match.
    # Priorité : film_match_start_ms / 1000.0 (calibré filmshell, exact)
    # Fallback  : (duration - playable_duration) en secondes (estimation API)
    query = f"""
        SELECT
            e.match_id,
            e.xuid,
            ANY_VALUE(r.start_time) AS start_time,
            MIN(CASE WHEN LOWER(e.event_type) = 'kill'  THEN e.time_ms END) / 1000.0
                AS first_kill_s_raw,
            MIN(CASE WHEN LOWER(e.event_type) = 'death' THEN e.time_ms END) / 1000.0
                AS first_death_s_raw,
            COALESCE(
                ANY_VALUE(r.film_match_start_ms) / 1000.0,
                GREATEST(
                    COALESCE(ANY_VALUE(r.duration_seconds), 0)
                    - COALESCE(ANY_VALUE(r.playable_duration_seconds),
                                ANY_VALUE(r.duration_seconds), 0),
                    0
                )
            ) AS countdown_s
        FROM shared.highlight_events e
        JOIN shared.match_registry r ON e.match_id = r.match_id
        WHERE e.xuid IN ({xu_ph})
          AND e.match_id IN ({mi_ph})
        GROUP BY e.match_id, e.xuid
        ORDER BY start_time ASC
    """
    try:
        result = conn.execute(query, [*xuids, *match_ids])  # type: ignore[union-attr]
        df = result_to_polars(result)
    except Exception:
        logger.debug("query_first_events: erreur requête highlight_events", exc_info=True)
        return pl.DataFrame()

    if df.is_empty():
        return df

    # Soustraire countdown en Python pour préserver NULL (GREATEST en SQL convertirait
    # NULL → 0 dans DuckDB, ce qui fausserait les histogrammes).
    # first_kill_s_raw / first_death_s_raw sont NULL si aucun event du type pour ce match.
    def _apply_countdown(col: pl.Expr, countdown: pl.Expr) -> pl.Expr:
        """Soustrait countdown_s de col en préservant NULL. Clamp à 0."""
        return (
            pl.when(col.is_not_null()).then((col - countdown).clip(lower_bound=0.0)).otherwise(None)
        )

    df = df.with_columns(
        _apply_countdown(pl.col("first_kill_s_raw"), pl.col("countdown_s")).alias("first_kill_s"),
        _apply_countdown(pl.col("first_death_s_raw"), pl.col("countdown_s")).alias("first_death_s"),
    ).drop(["first_kill_s_raw", "first_death_s_raw", "countdown_s"])

    return df
