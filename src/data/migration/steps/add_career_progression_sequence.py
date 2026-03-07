"""Migration : career_progression id → nextval(séquence) + colonne spartan_id (player)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import (
    add_spartan_id_to_career_progression,
    ensure_career_progression_autoincrement,
)


def _apply(conn):  # noqa: ANN001
    """Applique la séquence auto-increment et ajoute spartan_id."""
    ensure_career_progression_autoincrement(conn)
    add_spartan_id_to_career_progression(conn)


register(
    Migration(
        name="add_career_progression_sequence",
        target_db="player",
        description="career_progression : séquence auto-increment + spartan_id",
        apply_schema=_apply,
    )
)
