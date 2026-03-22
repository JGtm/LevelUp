"""Migration : vue mv_player_matches dans shared_matches_v2.duckdb.

Cette vue est le point d'entrée principal pour charger les matchs d'un joueur
dans l'architecture v6. Sans elle, l'app affiche "Aucun match trouvé" sur une
fresh install même après une sync réussie.
"""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_mv_player_matches_view

register(
    Migration(
        name="add_mv_player_matches_view",
        target_db="shared",
        description="Vue mv_player_matches pour le chargement des matchs joueur (v6)",
        apply_schema=ensure_mv_player_matches_view,
    )
)
