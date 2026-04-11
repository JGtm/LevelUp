"""Migrations de schéma DuckDB centralisées — façade.

Ce module re-exporte toutes les fonctions de migration depuis les sous-modules
par domaine créés lors du housekeeping pré-v7 (H3) :

- ``_migrations_utils.py`` : utilitaires partagés (get_table_columns, etc.)
- ``migrations_player.py`` : évolutions stats.duckdb (joueur)
- ``migrations_shared.py`` : évolutions shared_matches_v2.duckdb
- ``migrations_metadata.py`` : évolutions metadata.duckdb + shared_pve.duckdb

Les imports existants (``from src.data.sync.migrations import X``) continuent
de fonctionner sans modification grâce aux re-exports ci-dessous.
"""

from __future__ import annotations

import logging

from src.data.sync._migrations_utils import (  # noqa: F401
    _add_column_if_missing,
    _create_index_safe,
    column_exists,
    get_table_columns,
    table_exists,
)
from src.data.sync.migrations_metadata import (  # noqa: F401
    ensure_asset_translations_table,
    ensure_medal_definitions_table,
    ensure_medal_translations_table,
    ensure_pve_schema,
    ensure_weapon_labels,
)
from src.data.sync.migrations_player import (  # noqa: F401
    add_spartan_id_to_career_progression,
    ensure_career_progression_autoincrement,
    ensure_end_time_column,
    ensure_match_skill_rank_table,
    ensure_match_stats_columns,
    ensure_mv_player_matches_view,
    ensure_mv_session_stats_varchar,
    ensure_performance_indexes,
    ensure_performance_score_column,
    ensure_player_performance_indexes,
    ensure_pme_session_index,
    ensure_skill_history_table,
)
from src.data.sync.migrations_shared import (  # noqa: F401
    _create_v_gamertag_lookup,
    _create_v_match_full,
    ensure_backfill_completed_column,
    ensure_bot_teammate_column,
    ensure_dominance_flag_column,
    ensure_fix_bot_xuid_shared,
    ensure_highlight_events_autoincrement,
    ensure_match_participants_backfill_bits,
    ensure_match_participants_columns,
    ensure_match_registry_film_start,
    ensure_match_registry_playable_duration,
    ensure_match_registry_spnkr_version,
    ensure_medals_earned_bigint,
    ensure_resolution_views,
    ensure_team_ps_scores,
    ensure_weapon_kills_reconciled_as,
    ensure_weapon_kills_table,
    fix_events_loaded_inconsistency,
)

logger = logging.getLogger(__name__)


# ─────────────────────────────────────────────────────────────────────────────
# Backfill bitmask — constantes partagées (conservées ici pour rétrocompatibilité)
# ─────────────────────────────────────────────────────────────────────────────

BACKFILL_FLAGS: dict[str, int] = {
    "medals": 1 << 0,  # 1
    "events": 1 << 1,  # 2
    "skill": 1 << 2,  # 4
    "personal_scores": 1 << 3,  # 8
    # performance_scores supprimé : granularité joueur×match, pas par match.
    # Jamais relu pour décider de recalculer (détection via IS NULL dans player_match_enrichment).
    "accuracy": 1 << 5,  # 32
    "shots": 1 << 6,  # 64
    "enemy_mmr": 1 << 7,  # 128
    "assets": 1 << 8,  # 256
    "participants": 1 << 9,  # 512
    "participants_scores": 1 << 10,  # 1024
    "participants_kda": 1 << 11,  # 2048
    "participants_shots": 1 << 12,  # 4096
    "participants_damage": 1 << 13,  # 8192
    "aliases": 1 << 14,  # 16384
    "participants_avg_life": 1 << 15,  # 32768 - Ajouté pour éviter détection infinie
    # ── Weapon kills (v5.5) — OBSOLÈTE ──
    # lusr/csr supprimés : collisionnaient avec MatchBits.EVENTS (1<<16) et
    # MatchBits.ASSETS (1<<17) de constants.py. Jamais écrits en production.
    # Supprimés pour éliminer le risque de corruption silencieuse (cf. plan E.5).
    # Ce bit (18 = 262144) n'est jamais posé en production.
    # Source de vérité : MatchBits.WEAPON_KILLS = 1 << 21 dans constants.py.
    # Conservé pour rétrocompatibilité des tests uniquement.
    "weapon_kills": 1 << 18,  # 262144 — OBSOLÈTE, voir MatchBits.WEAPON_KILLS (1<<21)
    # ── Bits 19-22 : définis dans src.data.sync.constants.MatchBits ──
    # KILLER_VICTIM_LOADED = 1 << 19  (524288)
    # PVE_STATS            = 1 << 20  (1048576)
    # WEAPON_KILLS         = 1 << 21  (2097152) ← source de vérité weapon_kills
    # WEAPON_KILLS_NO_FILM = 1 << 22  (4194304)
}


def compute_backfill_mask(*types: str) -> int:
    """Calcule le masque de bits pour les types demandés.

    >>> compute_backfill_mask("medals", "events")
    3
    """
    mask = 0
    for t in types:
        mask |= BACKFILL_FLAGS.get(t, 0)
    return mask
