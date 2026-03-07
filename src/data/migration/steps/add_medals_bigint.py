"""Migration : medals_earned INTEGER → BIGINT (shared)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_medals_earned_bigint

register(
    Migration(
        name="add_medals_bigint",
        target_db="shared",
        description="medals_earned.medal_name_id INTEGER → BIGINT",
        apply_schema=ensure_medals_earned_bigint,
    )
)
