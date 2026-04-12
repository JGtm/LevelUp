"""Migration : catalogue metadata partagé du battle pass."""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations_metadata import (
    ensure_battlepass_item_definitions_table,
    ensure_battlepass_item_translations_table,
    ensure_battlepass_track_definitions_table,
    ensure_battlepass_track_translations_table,
)


def _ensure_battlepass_metadata_schema(conn) -> None:
    """Crée les tables metadata du battle pass en une seule migration."""
    ensure_battlepass_track_definitions_table(conn)
    ensure_battlepass_track_translations_table(conn)
    ensure_battlepass_item_definitions_table(conn)
    ensure_battlepass_item_translations_table(conn)


register(
    Migration(
        name="add_battlepass_metadata",
        target_db="metadata",
        description="Tables metadata partagées pour reward tracks et items battle pass",
        apply_schema=_ensure_battlepass_metadata_schema,
    )
)
