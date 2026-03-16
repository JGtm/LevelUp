"""Chargement des statistiques d'historique des rencontres depuis shared_matches.duckdb.

Fournit load_encounter_stats() — fonction libre sans état, ouvrant
shared_matches.duckdb en lecture seule pour calculer, pour chaque XUIDs
cible, les métriques d'interactions historiques avec le joueur principal.
"""

from __future__ import annotations

import logging
from pathlib import Path

import duckdb
import polars as pl

from src.utils.db import duckdb_read_only

logger = logging.getLogger(__name__)


def _has_view(conn: duckdb.DuckDBPyConnection, view_name: str) -> bool:
    """Vérifie si une vue existe dans la connexion courante."""
    try:
        result = conn.execute(
            "SELECT view_name FROM duckdb_views() WHERE view_name = ?",
            [view_name],
        ).fetchone()
        return result is not None
    except Exception:
        return False


def _get_shared_db_path(db_path: str) -> Path:
    """Dérive le chemin shared_matches.duckdb depuis le chemin stats.duckdb du joueur.

    Args:
        db_path: Chemin vers data/players/{gamertag}/stats.duckdb.

    Returns:
        Chemin résolu vers data/warehouse/shared_matches.duckdb.
    """
    return Path(db_path).resolve().parent.parent.parent / "warehouse" / "shared_matches.duckdb"


def _build_encounter_sql(n_targets: int, *, has_gamertag_lookup: bool = True) -> str:
    """Construit la requête SQL d'historique des rencontres.

    Génère 3n+6 placeholders pour les paramètres DuckDB.

    Args:
        n_targets: Nombre de XUIDs cibles (len(target_xuids)).
        has_gamertag_lookup: True si la vue v_gamertag_lookup est disponible (v6+).

    Returns:
        Chaîne SQL complète avec placeholders positionnels.
    """
    ph = ", ".join(["?"] * n_targets)
    if has_gamertag_lookup:
        gamertag_expr = "MAX(COALESCE(vg.gamertag, p.gamertag)) AS gamertag"
        gamertag_join = "LEFT JOIN  v_gamertag_lookup vg ON vg.xuid = p.xuid"
    else:
        gamertag_expr = "MAX(p.gamertag) AS gamertag"
        gamertag_join = ""
    return f"""
    WITH my_matches AS (
        SELECT match_id, team_id, outcome
        FROM match_participants
        WHERE xuid = ?
    ),
    encounters AS (
        SELECT
            p.xuid,
            {gamertag_expr},
            COUNT(*)                                AS total_encounters,
            SUM(CASE WHEN p.team_id = m.team_id   THEN 1 ELSE 0 END) AS ally_count,
            SUM(CASE WHEN p.team_id != m.team_id  THEN 1 ELSE 0 END) AS enemy_count,
            AVG(CASE
                WHEN p.team_id = m.team_id AND m.outcome = 2         THEN 1.0
                WHEN p.team_id = m.team_id AND m.outcome IN (3, 4)   THEN 0.0
                ELSE NULL
            END) AS winrate_as_ally,
            AVG(CASE
                WHEN p.team_id != m.team_id AND m.outcome = 2        THEN 1.0
                WHEN p.team_id != m.team_id AND m.outcome IN (3, 4)  THEN 0.0
                ELSE NULL
            END) AS winrate_vs_enemy,
            MAX(r.start_time) AS last_seen
        FROM match_participants p
        INNER JOIN my_matches m  ON m.match_id = p.match_id
        INNER JOIN match_registry r ON r.match_id = p.match_id
        {gamertag_join}
        WHERE p.xuid IN ({ph})
        GROUP BY p.xuid
    ),
    kvp_agg AS (
        SELECT
            CASE WHEN k.killer_xuid = ? THEN k.victim_xuid ELSE k.killer_xuid END AS opp,
            SUM(CASE WHEN k.killer_xuid = ? THEN k.kill_count ELSE 0 END)         AS kills_dealt,
            SUM(CASE WHEN k.victim_xuid = ? THEN k.kill_count ELSE 0 END)         AS deaths_suffered
        FROM killer_victim_pairs k
        INNER JOIN my_matches m ON m.match_id = k.match_id
        WHERE (k.killer_xuid = ? AND k.victim_xuid IN ({ph}))
           OR (k.killer_xuid IN ({ph}) AND k.victim_xuid = ?)
        GROUP BY 1
    )
    SELECT
        e.xuid,
        e.gamertag,
        e.total_encounters,
        e.ally_count,
        e.enemy_count,
        e.winrate_as_ally,
        e.winrate_vs_enemy,
        COALESCE(kvp.kills_dealt, 0)     AS kills_dealt,
        COALESCE(kvp.deaths_suffered, 0) AS deaths_suffered,
        e.last_seen
    FROM encounters e
    LEFT JOIN kvp_agg kvp ON kvp.opp = e.xuid
    """


def load_encounter_stats(
    self_xuid: str,
    target_xuids: list[str],
    db_path: str,
) -> pl.DataFrame:
    """Charge les métriques d'historique des rencontres pour une liste de joueurs.

    Pour chaque xuid dans target_xuids, calcule depuis shared_matches.duckdb :
    - Nombre total de rencontres communes (tous matchs confondus)
    - Répartition allié / ennemi
    - Win rate quand allié, win rate quand ennemi (perspectives du joueur principal)
    - Kills infligés / morts subies dans les duels directs (killer_victim_pairs)
    - Date de dernière rencontre

    Args:
        self_xuid: XUID du joueur principal.
        target_xuids: XUIDs des adversaires/alliés à analyser.
        db_path: Chemin du stats.duckdb joueur (pour dériver shared_db_path).

    Returns:
        DataFrame Polars avec colonnes : xuid, gamertag, total_encounters,
        ally_count, enemy_count, winrate_as_ally, winrate_vs_enemy,
        kills_dealt, deaths_suffered, last_seen. Vide si erreur ou no data.
    """
    if not target_xuids or not self_xuid:
        return pl.DataFrame()

    shared_path = _get_shared_db_path(db_path)
    if not shared_path.exists():
        logger.warning("shared_matches.duckdb introuvable : %s", shared_path)
        return pl.DataFrame()

    n = len(target_xuids)
    logger.debug("load_encounter_stats: %d cible(s) pour xuid=%s", n, self_xuid)
    try:
        with duckdb_read_only(shared_path) as _probe:
            has_lookup = _has_view(_probe, "v_gamertag_lookup")
    except Exception:
        has_lookup = False
    sql = _build_encounter_sql(n, has_gamertag_lookup=has_lookup)
    # Ordre des paramètres : voir _build_encounter_sql docstring — 3n+6 total
    params: list[str] = (
        [self_xuid]  # my_matches WHERE xuid = ?
        + target_xuids  # encounters WHERE p.xuid IN (...)
        + [self_xuid] * 3  # kvp CASE + SUM kills + SUM deaths
        + [self_xuid]  # kvp WHERE killer = ?
        + target_xuids  # kvp WHERE victim IN (...)
        + target_xuids  # kvp WHERE killer IN (...)
        + [self_xuid]  # kvp WHERE victim = ?
    )

    columns = [
        "xuid",
        "gamertag",
        "total_encounters",
        "ally_count",
        "enemy_count",
        "winrate_as_ally",
        "winrate_vs_enemy",
        "kills_dealt",
        "deaths_suffered",
        "last_seen",
    ]

    try:
        with duckdb_read_only(shared_path) as conn:
            rows = conn.execute(sql, params).fetchall()
        logger.debug("load_encounter_stats: %d ligne(s) retournée(s)", len(rows))
        if not rows:
            return pl.DataFrame()
        return pl.DataFrame(rows, schema=columns, orient="row")
    except Exception:
        logger.warning("load_encounter_stats échec pour xuid=%s", self_xuid, exc_info=True)
        return pl.DataFrame()
