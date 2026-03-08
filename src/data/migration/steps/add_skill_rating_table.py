"""Migration : table match_skill_rank + skill_history (player)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import (
    ensure_match_skill_rank_table,
    ensure_skill_history_table,
)


def _apply(conn):  # noqa: ANN001
    """Crée les tables skill_history et match_skill_rank."""
    ensure_skill_history_table(conn)
    ensure_match_skill_rank_table(conn)


register(
    Migration(
        name="add_skill_rating_table",
        target_db="player",
        description="Tables match_skill_rank et skill_history (LUSR/CSR)",
        apply_schema=_apply,
    )
)
