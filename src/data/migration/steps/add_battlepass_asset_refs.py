"""Migration : références d'assets battle pass dans metadata.duckdb."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations_metadata import ensure_battlepass_asset_refs_table

register(
    Migration(
        name="add_battlepass_asset_refs",
        target_db="metadata",
        description="Table battlepass_asset_refs pour référencer les visuels battle pass mis en cache",
        apply_schema=ensure_battlepass_asset_refs_table,
    )
)
