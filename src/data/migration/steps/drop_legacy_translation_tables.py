"""Migration : supprimer les tables de traduction legacy de metadata.duckdb.

``mode_translations`` et ``playlist_translations`` sont des tables de la v4
qui stockaient les traductions FR de façon redondante avec les colonnes ``name_fr``
des nouvelles tables ``game_variants`` et ``playlists`` (v6).

- ``mode_translations``    : 313 lignes (name_en, name_fr, category)
- ``playlist_translations``: 14 lignes  (uuid, name_en, name_fr)

Ces tables sont SUPPRIMÉES car :
1. MetadataResolver n'interroge plus ``playlist_translations`` (colonne ``uuid``
   non compatible avec le schéma v6 ``asset_id``).
2. ``mode_translations`` n'était pas interrogée par MetadataResolver.
3. Les traductions sont désormais dans ``game_variants.mode_name_fr`` et
   ``playlists.name_fr`` (peuplées par ``populate_metadata_from_discovery``).
"""

from __future__ import annotations

import duckdb

from src.data.migration.registry import Migration, register


def _drop_legacy_translation_tables(conn: duckdb.DuckDBPyConnection) -> None:
    """Supprime mode_translations et playlist_translations (idempotent)."""
    conn.execute("DROP TABLE IF EXISTS mode_translations")
    conn.execute("DROP TABLE IF EXISTS playlist_translations")


register(
    Migration(
        name="drop_legacy_translation_tables",
        target_db="metadata",
        description="Supprimer mode_translations + playlist_translations (legacy v4)",
        apply_schema=_drop_legacy_translation_tables,
    )
)
