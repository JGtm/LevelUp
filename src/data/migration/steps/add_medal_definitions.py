"""Migration : table medal_definitions (metadata.duckdb).

Stocke les labels FR/EN et descriptions de chaque médaille Halo Infinite
(medal_name_id BIGINT). Source de population : scripts/populate_medal_metadata.py.
"""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_medal_definitions_table

register(
    Migration(
        name="add_medal_definitions",
        target_db="metadata",
        description="Table medal_definitions (medal_name_id, noms, descriptions) dans metadata.duckdb",
        apply_schema=ensure_medal_definitions_table,
    )
)
