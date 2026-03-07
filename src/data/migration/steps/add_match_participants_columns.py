"""Migration : colonnes supplémentaires sur match_participants (shared)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import (
    ensure_match_participants_backfill_bits,
    ensure_match_participants_columns,
)


def _apply(conn):  # noqa: ANN001
    """Applique les deux migrations match_participants."""
    ensure_match_participants_columns(conn)
    ensure_match_participants_backfill_bits(conn)


register(
    Migration(
        name="add_match_participants_columns",
        target_db="shared",
        description="Colonnes stats/MMR/backfill_bits sur match_participants",
        apply_schema=_apply,
    )
)
