"""Migration : marqueur initial_sync_completed_at dans sync_meta (player DB).

Ce marqueur est la source de vérité persistante pour ``setup_state="ready"``
dans le bootstrap V7.  Il distingue les profils ayant effectivement terminé
une synchronisation de ceux qui viennent d'être créés.

Backfill : pour les profils existants sans marqueur mais ayant ``last_sync_at``
dans sync_meta, on considère la sync initiale comme accomplie et on recopie la
valeur de ``last_sync_at`` comme horodatage du marqueur.
"""

from __future__ import annotations

import duckdb

from src.data.migration.registry import Migration, register


def ensure_initial_sync_completed_at(conn: duckdb.DuckDBPyConnection) -> None:
    """Vérifie que la table sync_meta peut accueillir le marqueur (idem-potent).

    La table ``sync_meta`` est créée bien en amont (migration ``add_spnkr_version``).
    Cette migration ne crée pas de colonne supplémentaire — le marqueur est une
    *ligne* clé/valeur dans sync_meta, pas une colonne dédiée.

    On s'assure simplement que la table existe et possède bien les colonnes
    ``key`` / ``value`` / ``updated_at`` attendues.
    """
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS sync_meta (
            key        VARCHAR PRIMARY KEY,
            value      VARCHAR,
            updated_at TIMESTAMP
        )
        """
    )


def _backfill_initial_sync_marker(conn: duckdb.DuckDBPyConnection) -> None:
    """Backfill : recopie ``last_sync_at`` vers ``initial_sync_completed_at``.

    Uniquement si :
    - ``last_sync_at`` existe dans sync_meta avec une valeur non-NULL
    - ``initial_sync_completed_at`` est absent
    """
    conn.execute(
        """
        INSERT OR IGNORE INTO sync_meta (key, value, updated_at)
        SELECT
            'initial_sync_completed_at',
            value,
            CURRENT_TIMESTAMP
        FROM sync_meta
        WHERE key = 'last_sync_at'
          AND value IS NOT NULL
          AND NOT EXISTS (
              SELECT 1 FROM sync_meta WHERE key = 'initial_sync_completed_at'
          )
        """
    )


register(
    Migration(
        name="add_initial_sync_completed_at",
        target_db="player",
        description="Marqueur initial_sync_completed_at dans sync_meta (backfill depuis last_sync_at)",
        apply_schema=ensure_initial_sync_completed_at,
        apply_backfill=_backfill_initial_sync_marker,
    )
)
