"""Migration : correction des scores d'équipe dans mv_player_matches.

Recrée la vue mv_player_matches avec la logique de scores corrigée :
- Seuil de corruption réduit de 500 à 100
- Suppression du fallback ps_score (remplacé par NULL pour les scores corrompus)
- Le transformer lit désormais les stats mode-spécifiques en priorité
  (CaptureTheFlagStats.FlagCaptures, ZonesStats.StrongholdScoringTicks)
"""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_mv_player_matches_view

register(
    Migration(
        name="fix_mv_player_matches_scores",
        target_db="shared",
        description=(
            "Recréation de mv_player_matches : seuil corruption >100, NULL au lieu de ps_score"
        ),
        apply_schema=ensure_mv_player_matches_view,
    )
)
