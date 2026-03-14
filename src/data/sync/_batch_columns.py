"""Constantes de colonnes et plan de cast pour les insertions batch DuckDB.

Centralise :
- CAST_PLAN : mapping colonne → type DuckDB attendu
- Listes de colonnes par table (*_COLUMNS)
- CRITICAL_TABLES pour l'audit
"""

from __future__ import annotations

# =============================================================================
# Plan de cast — Sprint 15.3
# =============================================================================

#: Mapping colonne → type DuckDB attendu, appliqué à l'ingestion
#: pour garantir la cohérence des types dans toutes les tables.
CAST_PLAN: dict[str, dict[str, str]] = {
    "match_stats": {
        "match_id": "VARCHAR",
        "start_time": "TIMESTAMP",
        "end_time": "TIMESTAMP",
        "playlist_id": "VARCHAR",
        "playlist_name": "VARCHAR",
        "map_id": "VARCHAR",
        "map_name": "VARCHAR",
        "pair_id": "VARCHAR",
        "pair_name": "VARCHAR",
        "game_variant_id": "VARCHAR",
        "game_variant_name": "VARCHAR",
        "outcome": "TINYINT",
        "team_id": "TINYINT",
        "rank": "SMALLINT",
        "kills": "SMALLINT",
        "deaths": "SMALLINT",
        "assists": "SMALLINT",
        "kda": "FLOAT",
        "accuracy": "FLOAT",
        "headshot_kills": "SMALLINT",
        "max_killing_spree": "SMALLINT",
        "time_played_seconds": "INTEGER",
        "avg_life_seconds": "FLOAT",
        "my_team_score": "SMALLINT",
        "enemy_team_score": "SMALLINT",
        "team_mmr": "FLOAT",
        "enemy_mmr": "FLOAT",
        "damage_dealt": "FLOAT",
        "damage_taken": "FLOAT",
        "shots_fired": "INTEGER",
        "shots_hit": "INTEGER",
        "grenade_kills": "SMALLINT",
        "melee_kills": "SMALLINT",
        "power_weapon_kills": "SMALLINT",
        "score": "INTEGER",
        "personal_score": "INTEGER",
        "mode_category": "VARCHAR",
        "is_ranked": "BOOLEAN",
        "is_firefight": "BOOLEAN",
        "left_early": "BOOLEAN",
        "session_id": "VARCHAR",
        "session_label": "VARCHAR",
        "performance_score": "FLOAT",
        "teammates_signature": "VARCHAR",
        "known_teammates_count": "SMALLINT",
        "is_with_friends": "BOOLEAN",
        "friends_xuids": "VARCHAR",
        "created_at": "TIMESTAMP",
        "updated_at": "TIMESTAMP",
    },
    "medals_earned": {
        "match_id": "VARCHAR",
        "medal_name_id": "BIGINT",
        "count": "INTEGER",
    },
    "highlight_events": {
        "id": "INTEGER",
        "match_id": "VARCHAR",
        "event_type": "VARCHAR",
        "time_ms": "INTEGER",
        "xuid": "VARCHAR",
        "gamertag": "VARCHAR",
        "type_hint": "INTEGER",
        "raw_json": "VARCHAR",
    },
    "player_match_stats": {
        "match_id": "VARCHAR",
        "xuid": "VARCHAR",
        "team_id": "TINYINT",
        "team_mmr": "FLOAT",
        "enemy_mmr": "FLOAT",
        "kills_expected": "FLOAT",
        "kills_stddev": "FLOAT",
        "deaths_expected": "FLOAT",
        "deaths_stddev": "FLOAT",
        "assists_expected": "FLOAT",
        "assists_stddev": "FLOAT",
        "created_at": "TIMESTAMP",
    },
    "personal_score_awards": {
        "match_id": "VARCHAR",
        "xuid": "VARCHAR",
        "award_name": "VARCHAR",
        "award_category": "VARCHAR",
        "award_count": "INTEGER",
        "award_score": "INTEGER",
        "created_at": "TIMESTAMP",
    },
    "match_participants": {
        "match_id": "VARCHAR",
        "xuid": "VARCHAR",
        "team_id": "INTEGER",
        "outcome": "INTEGER",
        "gamertag": "VARCHAR",
        "rank": "SMALLINT",
        "score": "INTEGER",
        "kills": "SMALLINT",
        "deaths": "SMALLINT",
        "assists": "SMALLINT",
        "shots_fired": "INTEGER",
        "shots_hit": "INTEGER",
        "damage_dealt": "FLOAT",
        "damage_taken": "FLOAT",
        "avg_life_seconds": "FLOAT",
    },
    "xuid_aliases": {
        "xuid": "VARCHAR",
        "gamertag": "VARCHAR",
        "last_seen": "TIMESTAMP",
        "source": "VARCHAR",
        "updated_at": "TIMESTAMP",
    },
    "killer_victim_pairs": {
        "id": "INTEGER",
        "match_id": "VARCHAR",
        "killer_xuid": "VARCHAR",
        "killer_gamertag": "VARCHAR",
        "victim_xuid": "VARCHAR",
        "victim_gamertag": "VARCHAR",
        "kill_count": "INTEGER",
        "time_ms": "INTEGER",
        "is_validated": "BOOLEAN",
        "created_at": "TIMESTAMP",
    },
    "match_registry": {
        "match_id": "VARCHAR",
        "start_time": "TIMESTAMP",
        "end_time": "TIMESTAMP",
        "playlist_id": "VARCHAR",
        "playlist_name": "VARCHAR",
        "map_id": "VARCHAR",
        "map_name": "VARCHAR",
        "pair_id": "VARCHAR",
        "pair_name": "VARCHAR",
        "game_variant_id": "VARCHAR",
        "game_variant_name": "VARCHAR",
        "mode_category": "VARCHAR",
        "is_ranked": "BOOLEAN",
        "is_firefight": "BOOLEAN",
        "duration_seconds": "INTEGER",
        "team_0_score": "SMALLINT",
        "team_1_score": "SMALLINT",
        "backfill_completed": "INTEGER",
        "participants_loaded": "BOOLEAN",
        "events_loaded": "BOOLEAN",
        "medals_loaded": "BOOLEAN",
        "first_sync_by": "VARCHAR",
        "first_sync_at": "TIMESTAMP",
        "last_updated_at": "TIMESTAMP",
        "player_count": "SMALLINT",
        "created_at": "TIMESTAMP",
        "updated_at": "TIMESTAMP",
    },
}

#: Tables critiques pour l'audit de typage (Sprint 15.4).
CRITICAL_TABLES = ["match_registry", "match_participants", "highlight_events"]


# =============================================================================
# Colonnes par table
# =============================================================================

MEDAL_COLUMNS = ["match_id", "medal_name_id", "count"]

HIGHLIGHT_EVENT_COLUMNS = [
    "match_id",
    "event_type",
    "time_ms",
    "xuid",
    "gamertag",
    "type_hint",
    "raw_json",
]

SKILL_COLUMNS = [
    "match_id",
    "xuid",
    "team_id",
    "team_mmr",
    "enemy_mmr",
    "kills_expected",
    "kills_stddev",
    "deaths_expected",
    "deaths_stddev",
    "assists_expected",
    "assists_stddev",
    "created_at",
]

PERSONAL_SCORE_COLUMNS = [
    "match_id",
    "xuid",
    "award_name",
    "award_category",
    "award_count",
    "award_score",
    "created_at",
]

PARTICIPANT_COLUMNS = [
    "match_id",
    "xuid",
    "team_id",
    "outcome",
    "gamertag",
    "rank",
    "score",
    "kills",
    "deaths",
    "assists",
    "shots_fired",
    "shots_hit",
    "damage_dealt",
    "damage_taken",
    "avg_life_seconds",
    "headshot_kills",
    "max_killing_spree",
    "kda",
    "accuracy",
    "time_played_seconds",
    "grenade_kills",
    "melee_kills",
    "power_weapon_kills",
    "personal_score",
]

# Colonnes MMR/Skill — écrites UNIQUEMENT par le pipeline skill (sync engine
# ou backfill --skill), JAMAIS par l'upsert participants.
#
# Architecture v5.1 :
#   - batch_upsert_participants() utilise ON CONFLICT DO UPDATE SET
#     qui ne met à jour que PARTICIPANT_COLUMNS (stats de jeu), préservant
#     les colonnes MMR déjà peuplées.
#   - INSERT OR REPLACE (batch_upsert_rows) est INTERDIT pour match_participants
#     car il détruit toutes les colonnes non mentionnées (y compris MMR).
#   - Les colonnes MMR sont écrites par des UPDATE ciblés avec COALESCE dans
#     engine.py et orchestrator.py.
#
# assists_expected / assists_stddev : toujours NULL (limitation API Halo).
PARTICIPANT_MMR_COLUMNS = [
    "team_mmr",
    "enemy_mmr",
    "kills_expected",
    "kills_stddev",
    "deaths_expected",
    "deaths_stddev",
    "assists_expected",
    "assists_stddev",
]

#: Toutes les colonnes (INSERT initiale uniquement, pas les upserts).
PARTICIPANT_ALL_COLUMNS = PARTICIPANT_COLUMNS + PARTICIPANT_MMR_COLUMNS

ALIAS_COLUMNS = ["xuid", "gamertag", "last_seen", "source", "updated_at"]

#: v5 Shared Matches — colonnes spécifiques.
SHARED_MEDAL_COLUMNS = ["match_id", "xuid", "medal_name_id", "count"]
