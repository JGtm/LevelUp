"""Migration : COALESCE secondaire sur playlist_name_fr dans v_match_full.

Certaines playlists ont plusieurs UUIDs dans match_registry (ex: deux versions
de "Quick Play"), dont certains sans traduction FR dans asset_translations.
La vue v_match_full ne trouvait qu'une traduction pour l'UUID avec correspondance
directe, laissant playlist_name_fr=NULL pour les autres.

Fix : la vue v_match_full utilise maintenant un COALESCE secondaire qui cherche
la traduction FR par nom EN quand le JOIN par playlist_id échoue.
Cette migration recréée v_match_full + mv_player_matches pour propager le fix.
"""

import duckdb

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_mv_player_matches_view, ensure_resolution_views


def _apply(conn: duckdb.DuckDBPyConnection) -> None:
    """Recrée v_match_full (avec fallback FR) puis mv_player_matches."""
    ensure_resolution_views(conn)
    ensure_mv_player_matches_view(conn)


register(
    Migration(
        name="add_playlist_fr_name_fallback",
        target_db="shared",
        description="v_match_full: COALESCE secondaire playlist_name_fr par nom EN + recréation mv_player_matches",
        apply_schema=_apply,
    )
)
