"""Fonctions de cache Streamlit — Requêtes atomiques depuis DuckDB.

Chaque fonction encapsule une lecture unitaire avec @st.cache_data.
Extrait de cache_loaders.py pour respecter la limite de 500 lignes.
"""

from __future__ import annotations

import contextlib
import logging

import streamlit as st

from src.utils.db import is_duckdb_v4_path as _is_duckdb_v4_path
from src.utils.profiles import list_local_dbs

from ._cache_core import _resolve_player_xuid, get_cached_repository_st

logger = logging.getLogger(__name__)


# 8bis.A4 : TTL augmenté de 30s à 300s (le filesystem ne change pas en navigation)
@st.cache_data(show_spinner=False, ttl=300)
def cached_list_local_dbs(_refresh_token: int = 0) -> list[str]:
    """Liste des DB locales (TTL court pour éviter un scan disque trop fréquent)."""
    return list_local_dbs()


@st.cache_data(show_spinner=False)
def cached_same_team_match_ids_with_friend(
    db_path: str,
    self_xuid: str,
    friend_xuid: str,
    db_key: tuple[int, int] | None,
) -> tuple[str, ...]:
    """Retourne les match_id (str) joués dans la même équipe avec un ami (cache).

    Utilise DuckDBRepository pour DuckDB v4, sinon fallback legacy.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(self_xuid).strip())
            match_ids = repo.load_same_team_match_ids(str(friend_xuid).strip())
            return tuple(sorted(match_ids))
        except Exception:
            return ()


@st.cache_data(show_spinner=False)
def cached_query_matches_with_friend(
    db_path: str,
    self_xuid: str,
    friend_xuid: str,
    db_key: tuple[int, int] | None,
):
    """Requête les matchs joués avec un ami (cache).

    Utilise DuckDBRepository pour DuckDB v4, sinon fallback legacy.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(self_xuid).strip())
            match_ids = repo.load_matches_with_teammate(str(friend_xuid).strip())
            return match_ids
        except Exception:
            return []


@st.cache_data(show_spinner=False)
def cached_load_player_match_result(
    db_path: str,
    match_id: str,
    xuid: str,
    db_key: tuple[int, int] | None,
):
    """Charge le résultat d'un match pour un joueur (cache).

    Utilise DuckDBRepository pour .duckdb, sinon fallback legacy.

    Pipeline de lecture v5.1 :
        1. repo.load_match_skill_data(match_id) — charge team_mmr, enemy_mmr,
           kills/deaths/assists expected/stddev depuis shared.match_participants.
        2. Fallback : repo.load_match_mmr_batch() — uniquement team_mmr/enemy_mmr.

    ⚠️ assists expected/stddev : toujours NULL (limitation API Halo Infinite).
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    logger.debug("Match result chargé: %s", match_id)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(xuid).strip())
            # Charger skill data complet (MMR + expected/stddev)
            skill_data = repo.load_match_skill_data(match_id)
            if skill_data:
                skill_data["team_mmrs"] = None  # Non disponible dans DuckDB v4
                return skill_data
            # Fallback: load_match_mmr_batch si load_match_skill_data ne retourne rien
            mmr_data = repo.load_match_mmr_batch([match_id])
            team_mmr = None
            enemy_mmr = None
            if match_id in mmr_data:
                team_mmr, enemy_mmr = mmr_data[match_id]
            return {
                "team_id": None,
                "team_mmr": team_mmr,
                "enemy_mmr": enemy_mmr,
                "team_mmrs": None,
                "kills": {"count": None, "expected": None, "stddev": None},
                "deaths": {"count": None, "expected": None, "stddev": None},
                "assists": {"count": None, "expected": None, "stddev": None},
            }
        except Exception:
            return None


@st.cache_data(show_spinner=False)
def cached_load_match_medals_for_player(
    db_path: str,
    match_id: str,
    xuid: str,
    db_key: tuple[int, int] | None,
):
    """Charge les médailles d'un match pour un joueur (cache).

    Utilise DuckDBRepository pour .duckdb, sinon fallback legacy.
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    logger.debug("Médailles chargées: %s", match_id)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(xuid).strip())
            result = repo.load_match_medals(match_id)
            logger.debug(
                "Médailles chargées: %s (%d médailles)",
                match_id,
                len(result) if result is not None else 0,
            )
            return result
        except Exception:
            return []


@st.cache_data(show_spinner=False)
def cached_load_match_rosters(
    db_path: str,
    match_id: str,
    xuid: str,
    db_key: tuple[int, int] | None,
):
    """Charge les rosters d'un match (cache).

    Utilise DuckDBRepository pour DuckDB v4, sinon fallback legacy.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    logger.debug("Roster chargé: %s", match_id)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(xuid).strip())
            return repo.load_match_rosters(match_id)
        except Exception:
            return None


@st.cache_data(show_spinner=False)
def cached_load_highlight_events_for_match(
    db_path: str,
    match_id: str,
    *,
    db_key: tuple[int, int] | None = None,
):
    """Charge les événements highlight d'un match (cache).

    Utilise DuckDBRepository caché pour .duckdb, sinon fallback legacy.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    logger.debug("Events chargés: %s", match_id)
    if _is_duckdb_v4_path(db_path):
        try:
            player_xuid = _resolve_player_xuid(db_path)
            repo = get_cached_repository_st(db_path, player_xuid)
            return repo.load_highlight_events(match_id)
        except Exception:
            return []


@st.cache_data(show_spinner=False)
def cached_load_match_player_gamertags(
    db_path: str,
    match_id: str,
    *,
    db_key: tuple[int, int] | None = None,
):
    """Charge les gamertags des joueurs d'un match (cache).

    Sprint Gamertag Roster Fix : Utilise DuckDBRepository.resolve_gamertags_batch
    pour obtenir des gamertags propres depuis match_participants/xuid_aliases.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository pour résolution centralisée
    logger.debug("Gamertags résolus: %s", match_id)
    if _is_duckdb_v4_path(db_path):
        try:
            # Utiliser le repo caché pour récupérer les XUIDs et résoudre les gamertags
            # Résolution XUID : on utilise un xuid temporaire, resolve_gamertags_batch
            # ne dépend pas du xuid du repo
            repo = get_cached_repository_st(db_path, "")

            # Récupérer tous les XUIDs du match via le repo (highlight_events)
            events = repo.load_highlight_events(match_id)
            xuids = list({str(e["xuid"]) for e in events if e.get("xuid")})
            if not xuids:
                return {}

            return {
                xuid: gt
                for xuid, gt in repo.resolve_gamertags_batch(xuids, match_id=match_id).items()
                if gt
            }
        except Exception:
            return {}


@st.cache_data(show_spinner=False)
def cached_load_top_medals(
    db_path: str,
    xuid: str,
    match_ids: tuple[str, ...],
    top_n: int | None,
    db_key: tuple[int, int] | None,
):
    """Charge les top médailles (cache).

    Utilise DuckDBRepository pour les bases .duckdb, sinon fallback legacy.
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(xuid).strip())
            return repo.load_top_medals(
                list(match_ids),
                top_n=(int(top_n) if top_n is not None else None),
            )
        except Exception:
            return []


def top_medals_smart(
    db_path: str,
    xuid: str,
    match_ids: list[str],
    *,
    top_n: int | None,
    db_key: tuple[int, int] | None,
):
    """Charge les top médailles avec gestion intelligente du cache.

    Évite de stocker d'immenses tuples en cache pour les grandes listes.
    Utilise DuckDBRepository pour les bases .duckdb.
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    logger.debug(
        "Top medals: %d match_ids, route=%s",
        len(match_ids),
        "direct" if len(match_ids) > 1500 else "cache",
    )
    if _is_duckdb_v4_path(db_path):
        if len(match_ids) > 1500:
            try:
                repo = get_cached_repository_st(db_path, str(xuid).strip())
                return repo.load_top_medals(match_ids, top_n=top_n)
            except Exception:
                return []
        return cached_load_top_medals(db_path, xuid, tuple(match_ids), top_n, db_key=db_key)


def clear_app_caches() -> None:
    """Vide les caches Streamlit (utile si DB/alias/csv changent en dehors de l'app).

    Invalide aussi le cache repository (v5.1) pour forcer une reconnexion
    avec les données fraîches.
    """
    with contextlib.suppress(Exception):
        st.cache_data.clear()
    with contextlib.suppress(Exception):
        st.cache_resource.clear()


@st.cache_data(show_spinner=False)
def cached_list_other_xuids(
    db_path: str, self_xuid: str, db_key: tuple[int, int] | None = None, limit: int = 500
) -> list[str]:
    """Version cachée de list_other_player_xuids.

    DuckDB v4 utilise xuid_aliases. En v5, shared.match_participants.
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(self_xuid).strip())
            return repo.list_other_player_xuids(limit=limit)
        except Exception:
            return []


@st.cache_data(show_spinner=False)
def cached_list_top_teammates(
    db_path: str, self_xuid: str, db_key: tuple[int, int] | None = None, limit: int = 20
) -> list[tuple[str, int]]:
    """Version cachée de list_top_teammates.

    Utilise DuckDBRepository pour .duckdb, sinon TeammatesAggregate (cache DB),
    sinon fallback sur la requête JSON lente (list_top_teammates).
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(self_xuid).strip())
            return repo.list_top_teammates(limit=limit)
        except Exception:
            return []


# =============================================================================
# Nouvelles fonctions utilisant les tables de cache optimisées
# =============================================================================


@st.cache_data(show_spinner=False)
def cached_has_cache_tables(db_path: str, db_key: tuple[int, int] | None = None) -> bool:
    """Vérifie si les tables de cache existent.

    DuckDB v4 considéré comme ayant toujours les tables de cache.
    """
    _ = db_key
    # DuckDB v4 : toujours considéré comme ayant le cache
    if _is_duckdb_v4_path(db_path):
        return True


@st.cache_data(show_spinner=False, ttl=300)
def cached_get_match_skill_rank(
    db_path: str,
    match_id: str,
    db_key: tuple[int, int] | None = None,
) -> tuple | None:
    """Retourne le rating LUSR/CSR d'un match depuis match_skill_rank (read-only, mis en cache).

    N'exécute aucun DDL : si la table n'existe pas, retourne None silencieusement.
    Le ``rating_delta`` est calculé dynamiquement via ``LAG`` sur les ``rating_value``
    stockés pour garantir la cohérence avec les valeurs affichés (corrige l'incohérence
    du delta stocké lors d'un recalcul avec seed différent).

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        match_id: Identifiant du match.
        db_key: Clé d'invalidation (mtime, size) de la DB.

    Returns:
        Tuple (rating_type, rating_value, rating_deviation, tier_label,
               sub_tier, tier, tier_fr, rating_delta, playlist_group)
        ou None si absent.
    """
    _ = db_key
    from src.data.repositories.duckdb_repo import DuckDBRepository

    try:
        with DuckDBRepository(str(db_path), "", read_only=True) as repo:
            row = (
                repo._get_connection()
                .execute(
                    """
                WITH cte AS (
                    SELECT
                        msr.match_id,
                        msr.rating_type, msr.rating_value, msr.rating_deviation,
                        msr.tier_label, msr.sub_tier, msr.tier, msr.tier_fr,
                        msr.playlist_group,
                        msr.rating_value - LAG(msr.rating_value) OVER (
                            PARTITION BY msr.playlist_group
                            ORDER BY COALESCE(msr.start_time, msr.updated_at)
                        ) AS computed_delta
                    FROM match_skill_rank msr
                )
                SELECT rating_type, rating_value, rating_deviation, tier_label,
                       sub_tier, tier, tier_fr, computed_delta, playlist_group
                FROM cte
                WHERE match_id = ?
                """,
                    [match_id],
                )
                .fetchone()
            )
        return row
    except Exception:
        return None


@st.cache_data(show_spinner=False)
def cached_get_cache_stats(db_path: str, xuid: str, db_key: tuple[int, int] | None = None) -> dict:
    """Retourne les stats du cache DB pour un joueur.

    DuckDB v4 retourne des stats depuis le repository.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(xuid).strip())
            storage = repo.get_storage_info()
            return {
                "has_cache": True,
                "match_count": storage.get("total_matches", 0),
                "sessions_count": storage.get("sessions_count", 0),
            }
        except Exception:
            return {"has_cache": True}
