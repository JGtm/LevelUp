"""Migration : colonnes performance_score et end_time sur match_stats (player)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_match_stats_columns

register(
    Migration(
        name="add_performance_score",
        target_db="player",
        description="Colonnes optionnelles sur match_stats (performance_score, accuracy, etc.)",
        apply_schema=ensure_match_stats_columns,
    )
)
