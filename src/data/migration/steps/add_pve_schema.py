"""Migration : schéma PvE pve_match_stats (shared_pve)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_pve_schema

register(
    Migration(
        name="add_pve_schema",
        target_db="shared_pve",
        description="Table pve_match_stats (Firefight)",
        apply_schema=ensure_pve_schema,
    )
)
