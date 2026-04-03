"""Migration : film_match_start_ms sur match_registry (shared)."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_match_registry_film_start

register(
    Migration(
        name="add_film_match_start",
        target_db="shared",
        description=(
            "Ajoute film_match_start_ms (INTEGER, nullable) à match_registry — "
            "timestamp filmshell (ms) de la fin du countdown, détecté via analyse "
            "des chunks REPLICATION_DATA (scripts/_exp_spawn_download.py)."
        ),
        apply_schema=ensure_match_registry_film_start,
    )
)
