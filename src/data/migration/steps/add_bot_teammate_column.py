"""Migration : colonne had_bot_teammate sur player_match_enrichment (player)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_bot_teammate_column

register(
    Migration(
        name="add_bot_teammate_column",
        target_db="player",
        description="Colonne had_bot_teammate sur player_match_enrichment",
        apply_schema=ensure_bot_teammate_column,
    )
)
