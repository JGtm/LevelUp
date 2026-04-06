"""Migration : playable_duration_seconds + real_start_time sur match_registry (shared)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_match_registry_playable_duration

register(
    Migration(
        name="add_playable_duration",
        target_db="shared",
        description=(
            "Ajoute playable_duration_seconds (durée gameplay réelle) et "
            "real_start_time (heure UTC du début effectif) à match_registry"
        ),
        apply_schema=ensure_match_registry_playable_duration,
    )
)
