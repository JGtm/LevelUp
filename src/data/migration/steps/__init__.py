"""Package des steps de migration individuels.

L'import de ce module charge tous les steps, qui s'auto-enregistrent
dans le registre via ``register()``.
"""

from src.data.migration.steps import (
    add_bot_teammate_column,
    add_career_progression_sequence,
    add_dominance_flag,
    add_highlight_events_autoincrement,
    add_match_participants_columns,
    add_medals_bigint,
    add_performance_indexes,
    add_performance_score,
    add_pve_schema,
    add_skill_rating_table,
    add_spnkr_version,
    add_weapon_kills,
    add_weapon_kills_reconciled_as,
    drop_highlight_events_gamertag,
    drop_legacy_translation_tables,
    fix_bot_xuid,
    migrate_weapon_kills_to_ubigint,
)

__all__ = [
    "add_bot_teammate_column",
    "add_career_progression_sequence",
    "add_dominance_flag",
    "add_highlight_events_autoincrement",
    "add_match_participants_columns",
    "add_medals_bigint",
    "add_performance_indexes",
    "add_performance_score",
    "add_pve_schema",
    "add_skill_rating_table",
    "add_spnkr_version",
    "add_weapon_kills",
    "add_weapon_kills_reconciled_as",
    "drop_highlight_events_gamertag",
    "drop_legacy_translation_tables",
    "fix_bot_xuid",
    "migrate_weapon_kills_to_ubigint",
]
