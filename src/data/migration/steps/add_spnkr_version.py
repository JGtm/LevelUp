"""Migration : colonne sync_spnkr_version sur match_registry (shared)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_match_registry_spnkr_version

register(
    Migration(
        name="add_spnkr_version",
        target_db="shared",
        description="Colonne sync_spnkr_version sur match_registry",
        apply_schema=ensure_match_registry_spnkr_version,
    )
)
