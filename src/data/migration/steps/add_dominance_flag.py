"""Migration : colonne dominance_flag sur player_match_enrichment (player)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_dominance_flag_column

register(
    Migration(
        name="add_dominance_flag_column",
        target_db="player",
        description="Colonne dominance_flag sur player_match_enrichment (v5.7)",
        apply_schema=ensure_dominance_flag_column,
    )
)
