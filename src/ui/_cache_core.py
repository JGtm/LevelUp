"""Infrastructure partagée du cache Streamlit — Repository, constantes, résolution XUID.

Extrait de cache_loaders.py pour respecter la limite de 500 lignes.
"""

from __future__ import annotations

import logging
import os
from typing import TYPE_CHECKING

import streamlit as st

from src.ui.formatting import PARIS_TZ_NAME  # noqa: F401

if TYPE_CHECKING:
    from src.data.repositories.duckdb_repo import DuckDBRepository

logger = logging.getLogger(__name__)


class SharedDBUnavailableError(RuntimeError):
    """Levée quand shared_matches.duckdb est verrouillé par un autre processus.

    Non cachée par @st.cache_data → le prochain appel retente l'ATTACH.
    """


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

    logger.info("Création d'un repository mis en cache pour %s", db_path)
    repo = DuckDBRepository(db_path, xuid, read_only=True)
    # Warm-up : forcer la connexion + ATTACH immédiatement
    repo._get_connection()
    # Ne PAS cacher un repo sans shared si le fichier existe : raise pour
    # empêcher @st.cache_resource de stocker un repo cassé. Le prochain
    # appel retentera l'ATTACH.
    if not repo.has_shared and repo._shared_db_path.exists():
        repo.close()
        raise SharedDBUnavailableError(
            "shared_matches.duckdb est verrouillé par un autre processus "
            "et ne peut pas être attaché. Fermez toute connexion DuckDB "
            "externe (extension VS Code, CLI) et rechargez la page."
        )
    return repo


# ─── Constantes de projection par page (Sprint 19 — tâche 19.3) ────────────
# Colonnes effectivement utilisées par les pages principales.
# Permet de réduire la mémoire en ne chargeant que le nécessaire.
# Exclus du set commun (non utilisés dans le hot-path) :
#   game_variant_id — jamais affiché directement
#   team_id         — lu depuis shared.match_participants (vue detail)

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
    "rank",
    "personal_score",
    "my_team_score",
    "enemy_team_score",
    "team_mmr",
    "enemy_mmr",
    "performance_score",  # pré-calculé all-time depuis player_match_enrichment
]

# Colonnes calculées ajoutées par _enrich_matches_df (post-chargement)
COLUMNS_COMPUTED: list[str] = [
    "date",
    "kills_per_min",
    "deaths_per_min",
    "assists_per_min",
]


def db_cache_key(db_path: str) -> tuple[int, int, int, int, int] | None:
    """Retourne une signature stable des DBs pour invalider les caches.

    Surveille à la fois *stats.duckdb* (player) ET *shared_matches.duckdb*
    (matchs partagés v5) : les nouveaux matchs sont écrits dans shared, pas
    dans stats. Sans ce second composant, le cache @st.cache_data ne voit
    pas les matchs ajoutés après la dernière lecture.

    Le 5e composant, ``wal_sentinel``, prend la valeur ``mtime_ns`` du WAL
    de shared s'il existe, sinon 0. Cela ferme la race condition entre
    l'écriture WAL (mtime principal inchangé) et le checkpoint DuckDB :
    dès que le WAL est créé par un sync en cours, le cache est invalidé
    sans attendre le checkpoint.

    Returns:
        (mtime_ns_player, size_player, mtime_ns_shared, size_shared,
        wal_sentinel) ou None.
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
    wal_sentinel = 0
    try:
        from src.utils.paths import get_shared_matches_path_from_player

        shared_path = get_shared_matches_path_from_player(db_path)
        if shared_path and shared_path.exists():
            st_shared = os.stat(shared_path)
            mtime_shared = int(getattr(st_shared, "st_mtime_ns", int(st_shared.st_mtime * 1e9)))
            size_shared = int(st_shared.st_size)
            # WAL sentinel : si le WAL existe, on inclut son mtime dans la clé.
            # Cela force un cache miss dès qu'un sync commence à écrire,
            # même avant le checkpoint DuckDB (qui met à jour le fichier principal).
            wal_path = shared_path.with_suffix(shared_path.suffix + ".wal")
            if wal_path.exists():
                st_wal = os.stat(wal_path)
                wal_sentinel = int(getattr(st_wal, "st_mtime_ns", int(st_wal.st_mtime * 1e9)))
                logger.debug("WAL shared détecté — cache invalidé (wal_mtime_ns=%d)", wal_sentinel)
    except Exception:
        pass

    logger.debug(
        "db_cache_key(%s): mtime_player=%d size_player=%d mtime_shared=%d size_shared=%d wal=%d",
        os.path.basename(db_path),
        mtime_player,
        size_player,
        mtime_shared,
        size_shared,
        wal_sentinel,
    )
    return mtime_player, size_player, mtime_shared, size_shared, wal_sentinel


def _resolve_player_xuid(db_path: str) -> str:
    """Résout le XUID du joueur depuis sa DB.

    Stratégie de fallback :
    1. sync_meta (key='xuid') — source canonique v5
    2. shared.xuid_aliases via gamertag — fallback v5.1

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

        # Stratégie 2 : xuid_aliases via shared_matches.duckdb (v5.1)
        try:
            from pathlib import Path

            from src.utils.paths import get_shared_matches_path_from_player

            gamertag = Path(db_path).parent.name
            shared_path = get_shared_matches_path_from_player(db_path)
            if shared_path and shared_path.exists():
                with duckdb_read_only(shared_path) as shared_con:
                    from src.utils.xuid import lookup_xuid_for_gamertag

                    resolved = lookup_xuid_for_gamertag(shared_con, gamertag)
                    if resolved:
                        return resolved
        except Exception:
            pass

    except Exception as e:
        logger.debug("_resolve_player_xuid: résolution XUID échouée pour %s: %s", db_path, e)

    return ""
