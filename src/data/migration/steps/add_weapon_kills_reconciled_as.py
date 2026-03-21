"""Migration : colonnes reconciled_as, attribution_path, player_index sur weapon_kills.

Parser v2 : reconciled_as stocke le sentinel API sans écraser weapon_id,
attribution_path trace la source, player_index identifie le joueur dans le film.
"""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

register(
    Migration(
        name="add_weapon_kills_reconciled_as",
        target_db="shared",
        description="Colonnes reconciled_as, attribution_path, player_index sur weapon_kills",
        apply_schema=ensure_weapon_kills_reconciled_as,
    )
)
