"""Migration : supprimer la colonne gamertag de highlight_events (shared)."""

import logging

from src.data.migration.registry import Migration, register

logger = logging.getLogger(__name__)


def _apply(conn) -> None:  # noqa: ANN001
    """Supprime la colonne gamertag de highlight_events (idempotente)."""
    try:
        col = conn.execute(
            "SELECT column_name FROM information_schema.columns "
            "WHERE table_name = 'highlight_events' AND column_name = 'gamertag'"
        ).fetchone()
        if col:
            conn.execute("ALTER TABLE highlight_events DROP COLUMN gamertag")
            logger.info("✅ highlight_events.gamertag supprimé")
        else:
            logger.debug("highlight_events.gamertag déjà absent — skip")
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
