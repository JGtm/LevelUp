"""Migration : table weapon_kills (shared_matches.duckdb).

Stocke le nombre de kills par arme, joueur et match,
alimentée par le backfill depuis les films SPNKr.
"""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_weapon_kills_table

register(
    Migration(
        name="add_weapon_kills",
        target_db="shared",
        description="Table weapon_kills (match_id, xuid, weapon_id, kills) + index",
        apply_schema=ensure_weapon_kills_table,
    )
)
