"""Migration : tables asset_translations + medal_translations (metadata.duckdb).

Introduit deux tables pivot multi-langue (v6.3) :
- asset_translations : noms localisés pour maps, playlists, pairs, game_variants
  (peuplée par scripts/populate_asset_translations.py via Accept-Language header)
- medal_translations : noms + descriptions localisés pour les médailles
  (peuplée par scripts/populate_medal_metadata.py via champ translations de l'API)
"""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import (
    ensure_asset_translations_table,
    ensure_medal_translations_table,
)


def _apply(conn):  # type: ignore[no-untyped-def]
    ensure_asset_translations_table(conn)
    ensure_medal_translations_table(conn)


register(
    Migration(
        name="add_asset_translations",
        target_db="metadata",
        description=(
            "Tables asset_translations + medal_translations (pivot multi-langue) dans metadata.duckdb"
        ),
        apply_schema=_apply,
    )
)
