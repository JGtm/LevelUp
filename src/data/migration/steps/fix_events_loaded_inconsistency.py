"""Migration : corrige les matchs avec events_loaded=TRUE mais aucun highlight_event.

Root cause : la migration add_highlight_events_autoincrement (2026-03-07) a recréé
la table highlight_events avec autoincrement — les events antérieurs ont été perdus
mais le flag events_loaded est resté TRUE dans match_registry.

Correction : remet events_loaded=FALSE pour tout match dont aucune ligne n'existe
dans highlight_events. Le delta sync (via pending_events_ids dans _load_existing_match_ids)
retentera automatiquement l'API film pour les matchs récents.
"""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import fix_events_loaded_inconsistency

register(
    Migration(
        name="fix_events_loaded_inconsistency",
        target_db="shared",
        description=(
            "Remet events_loaded=FALSE pour les matchs sans highlight_events "
            "dans la table (incohérence après migration autoincrement)."
        ),
        apply_schema=fix_events_loaded_inconsistency,
    )
)
