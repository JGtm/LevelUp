"""Accès données pour la page Explorer.

Requêtes dédiées à la recherche de matchs et de joueurs.
Aucune dépendance Streamlit — module testable indépendamment.
"""

from __future__ import annotations

import logging

import polars as pl

from src.ui._cache_core import get_cached_repository_st

logger = logging.getLogger(__name__)


def load_is_with_friends(db_path: str, xuid: str, match_ids: list[str]) -> dict[str, bool]:
    """Charge le flag escouade/solo pour une liste de match_ids.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        xuid: XUID du joueur.
        match_ids: Liste des match_id à interroger.

    Returns:
        Mapping match_id → is_with_friends (True = escouade).
    """
    if not match_ids:
        return {}
    return get_cached_repository_st(db_path, xuid).load_is_with_friends_batch(match_ids)


def get_all_gamertags(db_path: str, xuid: str) -> list[str]:
    """Retourne tous les gamertags connus depuis shared_matches.duckdb.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        xuid: XUID du joueur (pour initialiser le repo).

    Returns:
        Liste triée de gamertags uniques.
    """
    try:
        return get_cached_repository_st(db_path, xuid).get_all_gamertags()
    except FileNotFoundError:
        logger.warning("get_all_gamertags: DB introuvable %s", db_path)
        return []


def resolve_gamertag_to_xuid(db_path: str, xuid: str, gamertag: str) -> str | None:
    """Résout un gamertag exact en XUID via shared_matches.duckdb.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        xuid: XUID du joueur (pour initialiser le repo).
        gamertag: Gamertag à rechercher (insensible à la casse).

    Returns:
        XUID correspondant ou None si introuvable.
    """
    try:
        return get_cached_repository_st(db_path, xuid).resolve_xuid_from_gamertag(gamertag)
    except FileNotFoundError:
        logger.debug("resolve_gamertag_to_xuid: DB introuvable %s", db_path)
        return None


def load_common_matches(
    db_path: str,
    player_xuid: str,
    target_xuid: str,
) -> pl.DataFrame:
    """Charge les matchs communs entre deux joueurs.

    Retourne un DataFrame avec les colonnes match_id, start_time,
    player_team_id, target_team_id, map_name, playlist_name, pair_name,
    outcome (du joueur principal), kills/deaths du joueur principal.

    Args:
        db_path: Chemin vers stats.duckdb joueur.
        player_xuid: XUID du joueur principal.
        target_xuid: XUID du joueur recherché.

    Returns:
        DataFrame Polars des matchs communs. Vide si erreur.
    """
    try:
        return get_cached_repository_st(db_path, player_xuid).load_common_matches_df(target_xuid)
    except FileNotFoundError:
        logger.debug("load_common_matches: DB introuvable %s", db_path)
        return pl.DataFrame()
