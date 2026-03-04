"""Package transformers : API JSON → lignes DuckDB.

Ce package regroupe les fonctions de transformation pour convertir
les réponses JSON de l'API SPNKr en rows prêtes pour DuckDB.

Architecture des modules internes :
- _helpers.py        : Constantes (XUID_RE, logger) + fonctions utilitaires pures
- _match.py          : transform_match_stats, extract_participants, extract_match_registry_data
- _skill.py          : transform_skill_stats, transform_all_skill_stats
- _events.py         : transform_highlight_events, extract_aliases, build_players_lookup
- _medals.py         : extract_medals, extract_all_medals
- _personal_scores.py: extract_personal_score_awards, transform_personal_score_awards
- _pve.py            : extract_pve_stats + helpers PvE

Zéro breaking change : toutes les fonctions publiques de l'ancien transformers.py
sont ré-exportées ici.
"""

from __future__ import annotations

# --- _events.py : événements, aliases, killer→victim ---
from src.data.sync.transformers._events import (
    build_players_lookup,
    extract_aliases,
    extract_killer_victim_pairs_from_highlight_events,
    transform_highlight_events,
)

# --- _helpers.py : constantes, utilitaires publics et privés (compatibilité) ---
from src.data.sync.transformers._helpers import (
    XUID_RE,
    _determine_mode_category,
    _extract_asset_id,
    _extract_kda,
    _extract_life_time_stats,
    _extract_mmr_from_skill,
    _extract_player_outcome_team,
    _extract_player_rank,
    _extract_player_score,
    _extract_player_stats,
    _extract_public_name,
    _extract_spree_headshots,
    _extract_team_scores,
    _extract_team_scores_by_id,
    _extract_xuid,
    _find_core_stats_dict,
    _find_player,
    _is_firefight_match,
    _is_ranked_playlist,
    _is_uuid,
    _normalize_gamertag,
    _parse_duration_to_seconds,
    _parse_iso_utc,
    _safe_float,
    _safe_int,
    _safe_str,
    compute_teammates_signature,
    extract_game_variant_category,
    logger,
)

# --- _match.py : matchs, participants, registry ---
from src.data.sync.transformers._match import (
    create_metadata_resolver,
    extract_match_registry_data,
    extract_participants,
    extract_xuids_from_match,
    transform_match_stats,
)

# --- _medals.py : médailles ---
from src.data.sync.transformers._medals import (
    extract_all_medals,
    extract_medals,
)

# --- _personal_scores.py : scores personnels ---
from src.data.sync.transformers._personal_scores import (
    _find_personal_scores,
    categorize_personal_score,
    extract_personal_score_awards,
    transform_personal_score_awards,
)

# --- _pve.py : PvE / Firefight ---
from src.data.sync.transformers._pve import (
    _extract_enemy_kills_by_type,
    _find_pve_stats_dict,
    extract_pve_stats,
)

# --- _skill.py : skill rating ---
from src.data.sync.transformers._skill import (
    transform_all_skill_stats,
    transform_skill_stats,
)

__all__ = [
    # Constantes
    "XUID_RE",
    "logger",
    # _helpers publics (y compris fonctions privées exposées pour compatibilité tests)
    "_extract_mmr_from_skill",
    "compute_teammates_signature",
    "extract_game_variant_category",
    # _match
    "create_metadata_resolver",
    "extract_match_registry_data",
    "extract_participants",
    "extract_xuids_from_match",
    "transform_match_stats",
    # _skill
    "transform_all_skill_stats",
    "transform_skill_stats",
    # _events
    "build_players_lookup",
    "extract_aliases",
    "extract_killer_victim_pairs_from_highlight_events",
    "transform_highlight_events",
    # _medals
    "extract_all_medals",
    "extract_medals",
    # _personal_scores
    "categorize_personal_score",
    "extract_personal_score_awards",
    "transform_personal_score_awards",
    # _pve
    "extract_pve_stats",
]
