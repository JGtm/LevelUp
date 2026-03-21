"""Migration : table weapon_labels (metadata.duckdb).

Stocke les labels EN/FR de chaque arme filmshell (weapon_id UBIGINT)
ainsi que les 3 sentinelles (0=Grenade, 1=Melee, 2=Vehicle).
Source de vérité : WEAPON_INT_TO_NAME + WEAPON_NAME_FR dans _weapon_data.py.
"""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_weapon_labels

register(
    Migration(
        name="add_weapon_labels",
        target_db="metadata",
        description="Table weapon_labels (weapon_id, name_en, name_fr) dans metadata.duckdb",
        apply_schema=ensure_weapon_labels,
    )
)
