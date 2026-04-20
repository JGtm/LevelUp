"""Migration : historique des battle pass joueur dans stats.duckdb."""

from src.data.migration.registry import Migration, register
from src.data.sync.battlepass_migrations import ensure_battlepass_snapshots_table

register(
    Migration(
        name="add_battlepass_snapshots",
        target_db="player",
        description="Table battlepass_snapshots pour historiser la progression battle pass d'un joueur",
        apply_schema=ensure_battlepass_snapshots_table,
    )
)
