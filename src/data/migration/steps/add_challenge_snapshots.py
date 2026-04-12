"""Migration : historique des défis joueur dans stats.duckdb."""

from src.data.migration.registry import Migration, register
from src.data.sync.challenge_migrations import ensure_challenge_snapshots_table

register(
    Migration(
        name="add_challenge_snapshots",
        target_db="player",
        description="Table challenge_snapshots pour historiser les états de défis joueur",
        apply_schema=ensure_challenge_snapshots_table,
    )
)
