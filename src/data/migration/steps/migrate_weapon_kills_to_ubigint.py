"""Migration : weapon_kills weapon_name VARCHAR → weapon_id UBIGINT.

Convertit les données existantes en préservant toutes les lignes.
Les weapon_name connus sont mappés vers leur weapon_id uint64,
les noms au format ``?{hex}`` sont parsés, les valeurs non résolues
(UNKNOWN, NON TROUVE, etc.) reçoivent ``NULL``.
"""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_weapon_kills_table

register(
    Migration(
        name="migrate_weapon_kills_to_ubigint",
        target_db="shared",
        description="Conversion weapon_kills weapon_name→weapon_id UBIGINT",
        apply_schema=ensure_weapon_kills_table,
    )
)
