"""Package des steps de migration individuels.

L'import de ce module charge tous les steps, qui s'auto-enregistrent
dans le registre via ``register()``.
"""

from src.data.migration.steps import (
    add_asset_translations,
    add_bot_teammate_column,
    add_career_progression_sequence,
    add_dominance_flag,
    add_highlight_events_autoincrement,
    add_match_participants_columns,
    add_medal_definitions,
    add_medals_bigint,
    add_mv_player_matches_view,
    add_performance_indexes,
    add_performance_score,
    add_pve_schema,
    add_skill_rating_table,
    add_spnkr_version,
    add_team_ps_scores,
    add_weapon_kills,
    add_weapon_kills_reconciled_as,
    add_weapon_labels,
    drop_highlight_events_gamertag,
    drop_legacy_translation_tables,
    fix_bot_xuid,
    fix_mv_player_matches_scores,
    migrate_weapon_kills_to_ubigint,
)

__all__ = [
    "add_asset_translations",
    "add_bot_teammate_column",
    "add_career_progression_sequence",
    "add_dominance_flag",
    "add_highlight_events_autoincrement",
    "add_match_participants_columns",
    "add_medal_definitions",
    "add_medals_bigint",
    "add_mv_player_matches_view",
    "add_performance_indexes",
    "add_performance_score",
    "add_pve_schema",
    "add_skill_rating_table",
    "add_spnkr_version",
    "add_team_ps_scores",
    "add_weapon_kills",
    "add_weapon_kills_reconciled_as",
    "add_weapon_labels",
    "drop_highlight_events_gamertag",
    "drop_legacy_translation_tables",
    "fix_bot_xuid",
    "fix_mv_player_matches_scores",
    "migrate_weapon_kills_to_ubigint",
]
