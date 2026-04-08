"""Schéma attendu des bases DuckDB LevelUp pour le healthcheck.

Chaque set définit les tables, vues et colonnes critiques attendues par DB.
"""

from __future__ import annotations

# ─────────────────────────────────────────────────────────────────────────────
# shared_matches.duckdb
# ─────────────────────────────────────────────────────────────────────────────

SHARED_EXPECTED_TABLES = {
    "match_registry",
    "match_participants",
    "highlight_events",
    "medals_earned",
    "xuid_aliases",
    "weapon_kills",
    "killer_victim_pairs",
    "schema_migrations",
}

SHARED_EXPECTED_VIEWS = {
    "v_gamertag_lookup",
    "v_match_full",
    "v_killer_victim_full",
    "v_weapon_kills",
}

SHARED_CRITICAL_COLUMNS = {
    "match_participants": {"mmr"},
    "weapon_kills": {"reconciled_as", "attribution_path"},
}

# ─────────────────────────────────────────────────────────────────────────────
# metadata.duckdb
# ─────────────────────────────────────────────────────────────────────────────

METADATA_EXPECTED_TABLES = {
    "weapon_labels",
    "asset_translations",
    "schema_migrations",
}

# ─────────────────────────────────────────────────────────────────────────────
# stats.duckdb (par joueur)
# ─────────────────────────────────────────────────────────────────────────────

PLAYER_EXPECTED_TABLES = {
    "player_match_enrichment",
    "personal_score_awards",
    "match_citations",
    "career_progression",
    "sessions",
    "sync_meta",
    "schema_migrations",
}

PLAYER_CRITICAL_COLUMNS = {
    "player_match_enrichment": {"performance_score", "session_id"},
}

# ─────────────────────────────────────────────────────────────────────────────
# shared_pve.duckdb
# ─────────────────────────────────────────────────────────────────────────────

PVE_EXPECTED_TABLES = {
    "pve_match_stats",
    "schema_migrations",
}
