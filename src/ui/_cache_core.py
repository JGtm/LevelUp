"""Infrastructure partagée du cache Streamlit — Repository, constantes, résolution XUID.

Extrait de cache_loaders.py pour respecter la limite de 500 lignes.
"""

from __future__ import annotations

import logging
import os
from typing import TYPE_CHECKING

import streamlit as st

if TYPE_CHECKING:
    from src.data.repositories.duckdb_repo import DuckDBRepository

logger = logging.getLogger(__name__)

# Timezone Paris pour les conversions
PARIS_TZ_NAME = "Europe/Paris"


# ─── Cache Repository (v5.1 perf) ──────────────────────────────────────────
# Connexion persistante via @st.cache_resource pour éviter les reconnexions
# coûteuses (ATTACH × 3 = 50-100ms par instanciation).


@st.cache_resource(ttl=3600)
def get_cached_repository_st(
    db_path: str,
    xuid: str,
) -> DuckDBRepository:
    """Retourne un DuckDBRepository mis en cache avec connexion persistante.

    Le repository est réutilisé entre les pages Streamlit pour éviter
    les reconnexions coûteuses (3× ATTACH = 50-100ms).

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        xuid: XUID du joueur.

    Returns:
        Instance DuckDBRepository mise en cache.
    """
    from src.data.repositories.duckdb_repo import DuckDBRepository

    logger.info(f"Création d'un repository mis en cache pour {db_path}")
    repo = DuckDBRepository(db_path, xuid, read_only=True)
    # Warm-up : forcer la connexion + ATTACH immédiatement
    repo._get_connection()
    return repo


# ─── Constantes de projection par page (Sprint 19 — tâche 19.3) ────────────
# Colonnes effectivement utilisées par les pages principales.
# Permet de réduire la mémoire en ne chargeant que le nécessaire.
# Exclus du set commun (non utilisés dans le hot-path) :
#   game_variant_id — jamais affiché directement
#   team_id         — lu depuis shared.match_participants (vue detail)
#   rank            — lu depuis shared.match_participants (vue detail)

COLUMNS_COMMON: list[str] = [
    "match_id",
    "start_time",
    "map_id",
    "map_name",
    "playlist_id",
    "playlist_name",
    "pair_id",
    "pair_name",
    "game_variant_name",  # utilisé dans match_view.py (mode affiché)
    "outcome",
    "kda",
    "ratio",  # calculé dans load_matches_as_polars (kills/deaths)
    "kills",
    "deaths",
    "assists",
    "accuracy",
    "average_life_seconds",
    "time_played_seconds",
    "max_killing_spree",
    "headshot_kills",
    "personal_score",
    "my_team_score",
    "enemy_team_score",
    "team_mmr",
    "enemy_mmr",
]

# Colonnes calculées ajoutées par _enrich_matches_df (post-chargement)
COLUMNS_COMPUTED: list[str] = [
    "date",
    "kills_per_min",
    "deaths_per_min",
    "assists_per_min",
]


def db_cache_key(db_path: str) -> tuple[int, int, int, int] | None:
    """Retourne une signature stable des DBs pour invalider les caches.

    Surveille à la fois *stats.duckdb* (player) ET *shared_matches.duckdb*
    (matchs partagés v5) : les nouveaux matchs sont écrits dans shared, pas
    dans stats. Sans ce second composant, le cache @st.cache_data ne voit
    pas les matchs ajoutés après la dernière lecture.

    Returns:
        (mtime_ns_player, size_player, mtime_ns_shared, size_shared) ou None.
    """
    try:
        st_ = os.stat(db_path)
    except OSError:
        return None

    mtime_player = int(getattr(st_, "st_mtime_ns", int(st_.st_mtime * 1e9)))
    size_player = int(st_.st_size)

    # Chemin shared_matches.duckdb déduit du chemin joueur
    mtime_shared = 0
    size_shared = 0
    try:
        from src.utils.paths import get_shared_matches_path_from_player

        shared_path = get_shared_matches_path_from_player(db_path)
        if shared_path and shared_path.exists():
            st_shared = os.stat(shared_path)
            mtime_shared = int(getattr(st_shared, "st_mtime_ns", int(st_shared.st_mtime * 1e9)))
            size_shared = int(st_shared.st_size)
    except Exception:
        pass

    return mtime_player, size_player, mtime_shared, size_shared


def _resolve_player_xuid(db_path: str) -> str:
    """Résout le XUID du joueur depuis sa DB.

    Stratégie de fallback :
    1. sync_meta (key='xuid') — source canonique v5
    2. player_match_stats.xuid — source legacy v3/v4 (toujours présente)
    3. shared.xuid_aliases via gamertag — dernier recours

    Returns:
        XUID en string, ou "" si introuvable.
    """
    from src.utils.db import duckdb_read_only

    try:
        with duckdb_read_only(db_path) as conn:
            # Stratégie 1 : sync_meta (source canonique v5)
            try:
                result = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid'").fetchone()
                if result and result[0] and str(result[0]).strip():
                    return str(result[0]).strip()
            except Exception:
                pass

            # Stratégie 2 : player_match_stats.xuid (legacy v3/v4)
            try:
                result = conn.execute(
                    "SELECT DISTINCT xuid FROM player_match_stats WHERE xuid IS NOT NULL LIMIT 1"
                ).fetchone()
                if result and result[0] and str(result[0]).strip():
                    return str(result[0]).strip()
            except Exception:
                pass

        # Stratégie 3 : xuid_aliases via shared_matches.duckdb (v5.1)
        try:
            from pathlib import Path

            from src.utils.paths import get_shared_matches_path_from_player

            gamertag = Path(db_path).parent.name
            shared_path = get_shared_matches_path_from_player(db_path)
            if shared_path and shared_path.exists():
                with duckdb_read_only(shared_path) as shared_con:
                    result = shared_con.execute(
                        "SELECT xuid FROM xuid_aliases WHERE gamertag = ? LIMIT 1", [gamertag]
                    ).fetchone()
                    if result and result[0] and str(result[0]).strip():
                        return str(result[0]).strip()
        except Exception:
            pass

    except Exception:
        pass

    return ""
