"""Migration : correction des XIDs de bots dans match_participants (shared).

Bug d'origine : recover_from_sqlite.py soustrayait la ) fermante de bid(X.0),
produisant 'bid(0.0' au lieu de 'bid(0.0)'. Cette migration corrige les
506 enregistrements affectés en ajoutant la ) manquante.
"""

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_fix_bot_xuid_shared

register(
    Migration(
        name="fix_bot_xuid",
        target_db="shared",
        description="Corrige bid(X.0 → bid(X.0) dans match_participants",
        apply_schema=ensure_fix_bot_xuid_shared,
    )
)
