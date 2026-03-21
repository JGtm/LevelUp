"""Migration : supprimer la colonne gamertag de highlight_events (shared)."""

import logging

from src.data.migration.registry import Migration, register

logger = logging.getLogger(__name__)


def _apply(conn) -> None:  # noqa: ANN001
    """Supprime la colonne gamertag de highlight_events via recréation de table (idempotente).

    DuckDB ne supporte pas ALTER TABLE DROP COLUMN quand des index existent.
    On recrée la table sans la colonne en préservant les données et les index.
    """
    col = conn.execute(
        "SELECT column_name FROM information_schema.columns "
        "WHERE table_name = 'highlight_events' AND column_name = 'gamertag'"
    ).fetchone()
    if not col:
        logger.debug("highlight_events.gamertag déjà absent — skip")
        return

    try:
        # 1) Sauvegarder sans la colonne gamertag
        conn.execute("DROP TABLE IF EXISTS highlight_events_no_gt_backup")
        conn.execute("""
            CREATE TABLE highlight_events_no_gt_backup AS
            SELECT id, match_id, event_type, time_ms, xuid, type_hint, raw_json
            FROM highlight_events
        """)

        # 2) Obtenir le max id pour la séquence
        max_id = conn.execute(
            "SELECT COALESCE(MAX(id), 0) FROM highlight_events_no_gt_backup"
        ).fetchone()[0]

        # 3) Supprimer l'ancienne table (CASCADE pour les index/séquences dépendantes)
        conn.execute("DROP TABLE highlight_events CASCADE")
        conn.execute("DROP SEQUENCE IF EXISTS highlight_events_id_seq CASCADE")

        # 4) Recréer la séquence et la table sans gamertag
        conn.execute(f"CREATE SEQUENCE highlight_events_id_seq START WITH {max_id + 1}")
        conn.execute("""
            CREATE TABLE highlight_events (
                id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
                match_id VARCHAR NOT NULL,
                event_type VARCHAR NOT NULL,
                time_ms INTEGER,
                xuid VARCHAR,
                type_hint INTEGER,
                raw_json VARCHAR
            )
        """)

        # 5) Restaurer les données
        conn.execute("""
            INSERT INTO highlight_events (id, match_id, event_type, time_ms, xuid, type_hint, raw_json)
            SELECT id, match_id, event_type, time_ms, xuid, type_hint, raw_json
            FROM highlight_events_no_gt_backup
        """)
        conn.execute("DROP TABLE highlight_events_no_gt_backup")

        # 6) Recréer l'index
        conn.execute("CREATE INDEX IF NOT EXISTS idx_highlight_match ON highlight_events(match_id)")

        logger.info("✅ highlight_events.gamertag supprimé (table recréée)")
    except Exception as exc:
        logger.warning("drop_highlight_events_gamertag: %s", exc)


register(
    Migration(
        name="drop_highlight_events_gamertag",
        target_db="shared",
        description="Supprime la colonne gamertag de highlight_events (remplacée par v_gamertag_lookup)",
        apply_schema=_apply,
    )
)
