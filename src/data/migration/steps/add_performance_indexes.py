"""Migration : index de performance sur shared et player (v5.1)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import (
    ensure_performance_indexes,
    ensure_player_performance_indexes,
)

register(
    Migration(
        name="add_shared_performance_indexes",
        target_db="shared",
        description="Index de performance sur match_participants/match_registry/medals_earned",
        apply_schema=ensure_performance_indexes,
    )
)

register(
    Migration(
        name="add_player_performance_indexes",
        target_db="player",
        description="Index de performance sur match_stats/personal_score_awards",
        apply_schema=ensure_player_performance_indexes,
    )
)
