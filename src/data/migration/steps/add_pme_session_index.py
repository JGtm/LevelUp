"""Migration : index session sur player_match_enrichment + type VARCHAR sur mv_session_stats."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_mv_session_stats_varchar, ensure_pme_session_index

register(
    Migration(
        name="add_pme_session_index",
        target_db="player",
        description="Index idx_pme_session sur player_match_enrichment(session_id)",
        apply_schema=ensure_pme_session_index,
    )
)

register(
    Migration(
        name="fix_mv_session_stats_varchar",
        target_db="player",
        description="Migre mv_session_stats.session_id de INTEGER vers VARCHAR",
        apply_schema=ensure_mv_session_stats_varchar,
    )
)
