"""Accès données pour la page Explorer.

Requêtes DuckDB dédiées à la recherche de matchs et de joueurs.
Aucune dépendance Streamlit — module testable indépendamment.
"""

from __future__ import annotations

import logging
from pathlib import Path

import polars as pl

from src.utils.db import duckdb_read_only

logger = logging.getLogger(__name__)


def _shared_db_path(db_path: str) -> Path:
    """Dérive shared_matches.duckdb depuis le chemin stats.duckdb joueur."""
    return Path(db_path).resolve().parent.parent.parent / "warehouse" / "shared_matches.duckdb"


def load_is_with_friends(db_path: str, match_ids: list[str]) -> dict[str, bool]:
    """Charge le flag escouade/solo pour une liste de match_ids.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        match_ids: Liste des match_id à interroger.

    Returns:
        Mapping match_id → is_with_friends (True = escouade).
    """
    if not match_ids:
        return {}
    ph = ", ".join(["?"] * len(match_ids))
    sql = f"SELECT match_id, is_with_friends FROM player_match_enrichment WHERE match_id IN ({ph})"
    try:
        with duckdb_read_only(db_path) as conn:
            rows = conn.execute(sql, match_ids).fetchall()
        return {str(r[0]): bool(r[1]) for r in rows if r[1] is not None}
    except Exception:
        logger.debug("load_is_with_friends échoué", exc_info=True)
        return {}


def get_all_gamertags(db_path: str) -> list[str]:
    """Retourne tous les gamertags connus depuis shared_matches.duckdb.

    Args:
        db_path: Chemin vers stats.duckdb du joueur (pour dériver shared).

    Returns:
        Liste triée de gamertags uniques.
    """
    shared = _shared_db_path(db_path)
    if not shared.exists():
        logger.warning("shared_matches.duckdb introuvable: %s", shared)
        return []
    try:
        with duckdb_read_only(shared) as conn:
            rows = conn.execute(
                """
                SELECT DISTINCT gamertag FROM (
                    SELECT gamertag FROM xuid_aliases WHERE gamertag IS NOT NULL
                    UNION
                    SELECT gamertag FROM highlight_events WHERE gamertag IS NOT NULL
                )
                ORDER BY gamertag
                """
            ).fetchall()
        result = [str(r[0]) for r in rows if r[0]]
        logger.debug("%d gamertags chargés (xuid_aliases + highlight_events)", len(result))
        return result
    except Exception:
        logger.error("get_all_gamertags échoué", exc_info=True)
        return []


def resolve_gamertag_to_xuid(db_path: str, gamertag: str) -> str | None:
    """Résout un gamertag exact en XUID via shared_matches.duckdb.

    Cherche dans xuid_aliases puis highlight_events en fallback.
    La recherche est insensible à la casse.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        gamertag: Gamertag à rechercher.

    Returns:
        XUID correspondant ou None si introuvable.
    """
    shared = _shared_db_path(db_path)
    if not shared.exists():
        return None
    try:
        with duckdb_read_only(shared) as conn:
            row = conn.execute(
                "SELECT xuid FROM xuid_aliases WHERE LOWER(gamertag) = LOWER(?) LIMIT 1",
                [gamertag],
            ).fetchone()
            if row:
                return str(row[0])
            # Fallback : highlight_events (gamertags absents de xuid_aliases)
            row = conn.execute(
                "SELECT xuid FROM highlight_events WHERE LOWER(gamertag) = LOWER(?) LIMIT 1",
                [gamertag],
            ).fetchone()
        return str(row[0]) if row else None
    except Exception:
        logger.error("resolve_gamertag_to_xuid échoué pour %s", gamertag, exc_info=True)
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
    shared = _shared_db_path(db_path)
    if not shared.exists():
        return pl.DataFrame()

    sql = """
    SELECT
        p.match_id,
        r.start_time,
        p.team_id  AS player_team_id,
        t.team_id  AS target_team_id,
        r.map_name,
        r.playlist_name,
        r.pair_name,
        p.outcome,
        p.kills,
        p.deaths,
        p.assists,
        p.kda
    FROM match_participants p
    INNER JOIN match_participants t
        ON t.match_id = p.match_id AND t.xuid = ?
    INNER JOIN match_registry r
        ON r.match_id = p.match_id
    WHERE p.xuid = ?
    ORDER BY r.start_time DESC
    """
    try:
        with duckdb_read_only(shared) as conn:
            result = conn.execute(sql, [target_xuid, player_xuid]).pl()
        logger.debug(
            "load_common_matches: %d matchs entre %s et %s",
            len(result),
            player_xuid,
            target_xuid,
        )
        return result
    except Exception:
        logger.error("load_common_matches échoué", exc_info=True)
        return pl.DataFrame()
