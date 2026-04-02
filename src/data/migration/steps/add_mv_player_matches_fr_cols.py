"""Migration : ajout des colonnes de traduction FR dans mv_player_matches.

Recrée la vue mv_player_matches en ajoutant :
- map_name_fr      : nom de carte traduit (depuis v_match_full)
- playlist_name_fr : nom de playlist traduit (depuis v_match_full)
- game_variant_name_fr : nom de variante de jeu traduit (depuis v_match_full)

Ces colonnes permettent à _add_derived_columns de construire map_ui,
playlist_fr et mode_ui directement depuis la DB sans recourir aux
fonctions de fallback qui ne traduisent pas tout.
"""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_mv_player_matches_view

register(
    Migration(
        name="add_mv_player_matches_fr_cols",
        target_db="shared",
        description="Recréation de mv_player_matches avec colonnes map_name_fr, playlist_name_fr, game_variant_name_fr",
        apply_schema=ensure_mv_player_matches_view,
    )
)
