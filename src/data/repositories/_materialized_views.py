"""
Mixin pour les vues matérialisées DuckDB.

Regroupe les méthodes de création, rafraîchissement et lecture
des vues matérialisées extraites de DuckDBRepository :
- _ensure_mv_tables
- refresh_materialized_views
- get_map_stats
- get_mode_category_stats
- get_global_stats
- get_session_stats
- has_materialized_views
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

import duckdb

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)

# SQL d'agrégation partagé entre rebuild complet et incrémental.
# Paramètres positionnels : [xuid, ...session_ids (optionnel)].
_SESSION_STATS_INSERT_SQL = """
INSERT INTO mv_session_stats
SELECT
    pme.session_id,
    COUNT(*) AS match_count,
    MIN(mr.start_time) AS start_time,
    MAX(mr.start_time) AS end_time,
    SUM(mp.kills) AS total_kills,
    SUM(mp.deaths) AS total_deaths,
    SUM(mp.assists) AS total_assists,
    CASE WHEN SUM(mp.deaths) > 0
         THEN CAST(SUM(mp.kills) AS DOUBLE) / SUM(mp.deaths)
         ELSE SUM(mp.kills) END AS kd_ratio,
    CASE WHEN COUNT(*) > 0
         THEN SUM(CASE WHEN mp.outcome = 2 THEN 1.0 ELSE 0.0 END) / COUNT(*)
         ELSE 0 END AS win_rate,
    AVG(mp.accuracy) AS avg_accuracy,
    AVG(mp.avg_life_seconds) AS avg_life_seconds,
    CURRENT_TIMESTAMP AS updated_at
FROM player_match_enrichment pme
JOIN shared.match_participants mp ON mp.match_id = pme.match_id AND mp.xuid = ?
JOIN shared.match_registry mr ON mr.match_id = pme.match_id
WHERE pme.session_id IS NOT NULL
{extra_filter}
GROUP BY pme.session_id
"""


class MaterializedViewsMixin:
    """Mixin fournissant la gestion des vues matérialisées pour DuckDBRepository."""

    def _ensure_mv_tables(self) -> None:
        """Crée les tables de vues matérialisées si elles n'existent pas."""
        conn = self._get_connection()
        for ddl in (
            """CREATE TABLE IF NOT EXISTS mv_map_stats (
                map_id VARCHAR PRIMARY KEY, map_name VARCHAR,
                matches_played INTEGER, wins INTEGER, losses INTEGER, ties INTEGER,
                avg_kills DOUBLE, avg_deaths DOUBLE, avg_assists DOUBLE,
                avg_accuracy DOUBLE, avg_kda DOUBLE, win_rate DOUBLE,
                updated_at TIMESTAMP)""",
            """CREATE TABLE IF NOT EXISTS mv_mode_category_stats (
                mode_category VARCHAR PRIMARY KEY, matches_played INTEGER,
                avg_kills DOUBLE, avg_deaths DOUBLE, avg_assists DOUBLE,
                avg_kda DOUBLE, avg_accuracy DOUBLE, win_rate DOUBLE,
                updated_at TIMESTAMP)""",
            """CREATE TABLE IF NOT EXISTS mv_session_stats (
                session_id VARCHAR PRIMARY KEY, match_count INTEGER,
                start_time TIMESTAMP, end_time TIMESTAMP,
                total_kills INTEGER, total_deaths INTEGER, total_assists INTEGER,
                kd_ratio DOUBLE, win_rate DOUBLE,
                avg_accuracy DOUBLE, avg_life_seconds DOUBLE, updated_at TIMESTAMP)""",
            """CREATE TABLE IF NOT EXISTS mv_global_stats (
                stat_key VARCHAR PRIMARY KEY, stat_value DOUBLE, updated_at TIMESTAMP)""",
        ):
            conn.execute(ddl)

    # Expression CASE de catégorisation des modes (dupliquée dans _refresh_map_stats/mode_stats)
    _MODE_CATEGORY_EXPR = """
        COALESCE(
            CASE
                WHEN mr.pair_name LIKE '%Slayer%' OR mr.pair_name LIKE '%Tuerie%' THEN 'Slayer'
                WHEN mr.pair_name LIKE '%CTF%' OR mr.pair_name LIKE '%Flag%' OR mr.pair_name LIKE '%Drapeau%' THEN 'CTF'
                WHEN mr.pair_name LIKE '%Stronghold%' OR mr.pair_name LIKE '%Forteresse%' THEN 'Strongholds'
                WHEN mr.pair_name LIKE '%Oddball%' OR mr.pair_name LIKE '%Balle%' THEN 'Oddball'
                WHEN mr.pair_name LIKE '%Total%Control%' OR mr.pair_name LIKE '%Contrôle%' THEN 'Total Control'
                WHEN mr.pair_name LIKE '%Attrition%' THEN 'Attrition'
                WHEN mr.pair_name LIKE '%KOTH%' OR mr.pair_name LIKE '%King%' OR mr.pair_name LIKE '%Roi%' THEN 'King of the Hill'
                WHEN mr.pair_name LIKE '%Extraction%' THEN 'Extraction'
                WHEN mr.pair_name LIKE '%Firefight%' OR mr.pair_name LIKE '%Sentry%' THEN 'Firefight'
                WHEN mr.pair_name LIKE '%FFA%' OR mr.pair_name LIKE '%Free%For%All%' THEN 'FFA'
                ELSE 'Autre'
            END,
            'Autre'
        )
    """.strip()

    def refresh_materialized_views(self, *, new_ids: list[str] | None = None) -> dict[str, int]:
        """Rafraîchit les vues matérialisées après sync.

        Quand ``new_ids`` est fourni (liste de match_id insérés lors de cette
        sync), effectue un rebuild **partiel** pour ``mv_map_stats`` et
        ``mv_mode_category_stats`` : seules les lignes affectées par les
        nouveaux matchs sont supprimées puis réinsérées.  Les vues globales
        (``mv_global_stats``, ``mv_session_stats``) sont toujours reconstruites
        en totalité car elles agrègent l'ensemble des matchs du joueur.

        Args:
            new_ids: Match IDs insérés lors de cette synchronisation, ou
                ``None`` pour un rebuild complet.

        Returns:
            Dict avec le nombre de lignes insérées par table.
        """
        # Forcer une connexion en écriture
        if self._read_only:
            if self._connection is not None:
                self._connection.close()
            self._connection = duckdb.connect(
                str(self._player_db_path),
                read_only=False,
            )
            self._connection.execute(f"SET memory_limit = '{self._memory_limit}'")
            self._read_only = False
            self._attached_dbs.clear()

        conn = self._get_connection()

        # Créer les tables si nécessaire
        self._ensure_mv_tables()

        results = {}

        # ─── mv_map_stats ───
        results["mv_map_stats"] = self._refresh_mv_map_stats(conn, new_ids)

        # ─── mv_mode_category_stats ───
        results["mv_mode_category_stats"] = self._refresh_mv_mode_category_stats(conn, new_ids)

        # ─── mv_global_stats (toujours rebuild complet) ───
        results["mv_global_stats"] = self._rebuild_global_stats(conn)

        # ─── mv_session_stats (rebuild complet ou incrémental) ───
        results["mv_session_stats"] = self._refresh_session_stats(conn, new_ids)

        logger.info("Vues matérialisées rafraîchies: %s", results)
        return results

    def _rebuild_global_stats(self, conn: duckdb.DuckDBPyConnection) -> int:
        """Reconstruit mv_global_stats depuis zéro (compteurs globaux du joueur)."""
        conn.execute("DELETE FROM mv_global_stats")
        global_stats = conn.execute(
            """
            SELECT
                COUNT(*) as total_matches,
                SUM(mp.kills) as total_kills,
                SUM(mp.deaths) as total_deaths,
                SUM(mp.assists) as total_assists,
                SUM(CASE WHEN mp.outcome = 2 THEN 1 ELSE 0 END) as wins,
                SUM(CASE WHEN mp.outcome = 3 THEN 1 ELSE 0 END) as losses,
                AVG(mp.kda) as avg_kda,
                AVG(mp.accuracy) as avg_accuracy,
                SUM(mp.time_played_seconds) / 3600.0 as total_hours,
                AVG(mp.avg_life_seconds) as avg_life_seconds
            FROM shared.match_participants mp
            WHERE mp.xuid = ?
            """,
            [self._xuid],
        ).fetchone()
        if global_stats:
            stats_data = [
                ("total_matches", global_stats[0]),
                ("total_kills", global_stats[1]),
                ("total_deaths", global_stats[2]),
                ("total_assists", global_stats[3]),
                ("wins", global_stats[4]),
                ("losses", global_stats[5]),
                ("avg_kda", global_stats[6]),
                ("avg_accuracy", global_stats[7]),
                ("total_hours", global_stats[8]),
                ("avg_life_seconds", global_stats[9]),
            ]
            conn.executemany(
                "INSERT INTO mv_global_stats (stat_key, stat_value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
                stats_data,
            )
        return conn.execute("SELECT COUNT(*) FROM mv_global_stats").fetchone()[0]

    def _refresh_session_stats(
        self, conn: duckdb.DuckDBPyConnection, new_ids: list[str] | None
    ) -> int:
        """Rafraîchit mv_session_stats.

        - ``new_ids`` fourni (sync delta) : recalcule les 3 sessions les plus
          récentes (par ``start_time DESC``).
        - ``new_ids`` est None (backfill/full) : rebuild complet.

        Retourne le nombre total de lignes dans mv_session_stats.
        """
        try:
            has_sessions = (
                conn.execute(
                    "SELECT COUNT(*) FROM information_schema.columns "
                    "WHERE table_name = 'player_match_enrichment' AND column_name = 'session_id'"
                ).fetchone()[0]
                > 0
            )
            if not has_sessions:
                return 0

            if new_ids is not None:
                self._partial_refresh_session_stats(conn)
            else:
                conn.execute("DELETE FROM mv_session_stats")
                conn.execute(
                    _SESSION_STATS_INSERT_SQL.format(extra_filter=""),
                    [self._xuid],
                )

            return conn.execute("SELECT COUNT(*) FROM mv_session_stats").fetchone()[0]
        except Exception:
            return 0

    def _partial_refresh_session_stats(self, conn: duckdb.DuckDBPyConnection) -> None:
        """Supprime et recalcule les 3 dernières sessions (rebuild incrémental)."""
        rows = conn.execute(
            "SELECT session_id FROM mv_session_stats ORDER BY start_time DESC LIMIT 3"
        ).fetchall()
        recent_ids = [r[0] for r in rows]
        if not recent_ids:
            return
        placeholders = ", ".join("?" * len(recent_ids))
        conn.execute(
            f"DELETE FROM mv_session_stats WHERE session_id IN ({placeholders})",
            recent_ids,
        )
        conn.execute(
            _SESSION_STATS_INSERT_SQL.format(
                extra_filter=f"AND pme.session_id IN ({placeholders})"
            ),
            [self._xuid, *recent_ids],
        )

    def _refresh_mv_map_stats(
        self,
        conn: duckdb.DuckDBPyConnection,
        new_ids: list[str] | None,
    ) -> int:
        """Rebuild mv_map_stats — partiel si new_ids, complet sinon."""
        _INSERT_SQL = """
            INSERT INTO mv_map_stats
            SELECT
                mr.map_id,
                mr.map_name,
                COUNT(*) as matches_played,
                SUM(CASE WHEN mp.outcome = 2 THEN 1 ELSE 0 END) as wins,
                SUM(CASE WHEN mp.outcome = 3 THEN 1 ELSE 0 END) as losses,
                SUM(CASE WHEN mp.outcome = 1 THEN 1 ELSE 0 END) as ties,
                AVG(CAST(mp.kills AS DOUBLE)) as avg_kills,
                AVG(CAST(mp.deaths AS DOUBLE)) as avg_deaths,
                AVG(CAST(mp.assists AS DOUBLE)) as avg_assists,
                AVG(mp.accuracy) as avg_accuracy,
                AVG(mp.kda) as avg_kda,
                CASE WHEN COUNT(*) > 0
                     THEN SUM(CASE WHEN mp.outcome = 2 THEN 1.0 ELSE 0.0 END) / COUNT(*)
                     ELSE 0 END as win_rate,
                CURRENT_TIMESTAMP as updated_at
            FROM shared.match_participants mp
            JOIN shared.match_registry mr ON mr.match_id = mp.match_id
            WHERE mp.xuid = ?
              AND mr.map_id IS NOT NULL
        """
        if new_ids:
            placeholders = ", ".join("?" * len(new_ids))
            touched = [
                r[0]
                for r in conn.execute(
                    f"""
                    SELECT DISTINCT mr.map_id
                    FROM shared.match_registry mr
                    JOIN shared.match_participants mp ON mp.match_id = mr.match_id
                    WHERE mr.match_id IN ({placeholders})
                      AND mp.xuid = ?
                      AND mr.map_id IS NOT NULL
                    """,
                    new_ids + [self._xuid],
                ).fetchall()
            ]
            if not touched:
                logger.debug(
                    "mv_map_stats: 0 carte touchée par %d nouveaux matchs — skip", len(new_ids)
                )
                return conn.execute("SELECT COUNT(*) FROM mv_map_stats").fetchone()[0]
            logger.debug(
                "mv_map_stats: rebuild partiel pour %d carte(s): %s", len(touched), touched
            )
            map_ph = ", ".join("?" * len(touched))
            conn.execute(f"DELETE FROM mv_map_stats WHERE map_id IN ({map_ph})", touched)
            conn.execute(
                _INSERT_SQL
                + f"  AND mr.map_id IN ({map_ph})\n            GROUP BY mr.map_id, mr.map_name",
                [self._xuid] + touched,
            )
        else:
            logger.debug("mv_map_stats: rebuild complet (new_ids=None)")
            conn.execute("DELETE FROM mv_map_stats")
            conn.execute(
                _INSERT_SQL + "            GROUP BY mr.map_id, mr.map_name",
                [self._xuid],
            )
        return conn.execute("SELECT COUNT(*) FROM mv_map_stats").fetchone()[0]

    def _refresh_mv_mode_category_stats(
        self,
        conn: duckdb.DuckDBPyConnection,
        new_ids: list[str] | None,
    ) -> int:
        """Rebuild mv_mode_category_stats — partiel si new_ids, complet sinon."""
        mode_expr = self._MODE_CATEGORY_EXPR
        _INNER_SQL = f"""
            SELECT
                {mode_expr} as mode_category,
                COUNT(*) as matches_played,
                AVG(CAST(mp.kills AS DOUBLE)) as avg_kills,
                AVG(CAST(mp.deaths AS DOUBLE)) as avg_deaths,
                AVG(CAST(mp.assists AS DOUBLE)) as avg_assists,
                AVG(mp.kda) as avg_kda,
                AVG(mp.accuracy) as avg_accuracy,
                CASE WHEN COUNT(*) > 0
                     THEN SUM(CASE WHEN mp.outcome = 2 THEN 1.0 ELSE 0.0 END) / COUNT(*)
                     ELSE 0 END as win_rate
            FROM shared.match_participants mp
            JOIN shared.match_registry mr ON mr.match_id = mp.match_id
            WHERE mp.xuid = ?
            GROUP BY 1
        """
        if new_ids:
            placeholders = ", ".join("?" * len(new_ids))
            touched = [
                r[0]
                for r in conn.execute(
                    f"""
                    SELECT DISTINCT {mode_expr} as mode_category
                    FROM shared.match_registry mr
                    JOIN shared.match_participants mp ON mp.match_id = mr.match_id
                    WHERE mr.match_id IN ({placeholders}) AND mp.xuid = ?
                    """,
                    new_ids + [self._xuid],
                ).fetchall()
            ]
            if not touched:
                logger.debug(
                    "mv_mode_category_stats: 0 catégorie touchée par %d nouveaux matchs — skip",
                    len(new_ids),
                )
                return conn.execute("SELECT COUNT(*) FROM mv_mode_category_stats").fetchone()[0]
            logger.debug("mv_mode_category_stats: rebuild partiel pour catégorie(s): %s", touched)
            cat_ph = ", ".join("?" * len(touched))
            conn.execute(
                f"DELETE FROM mv_mode_category_stats WHERE mode_category IN ({cat_ph})",
                touched,
            )
            conn.execute(
                f"""
                INSERT INTO mv_mode_category_stats
                SELECT sub.mode_category, sub.matches_played, sub.avg_kills,
                       sub.avg_deaths, sub.avg_assists, sub.avg_kda, sub.avg_accuracy,
                       sub.win_rate, CURRENT_TIMESTAMP
                FROM ({_INNER_SQL}) sub
                WHERE sub.mode_category IN ({cat_ph})
                """,
                [self._xuid] + touched,
            )
        else:
            logger.debug("mv_mode_category_stats: rebuild complet (new_ids=None)")
            conn.execute("DELETE FROM mv_mode_category_stats")
            conn.execute(
                f"""
                INSERT INTO mv_mode_category_stats
                SELECT sub.mode_category, sub.matches_played, sub.avg_kills,
                       sub.avg_deaths, sub.avg_assists, sub.avg_kda, sub.avg_accuracy,
                       sub.win_rate, CURRENT_TIMESTAMP
                FROM ({_INNER_SQL}) sub
                """,
                [self._xuid],
            )
        return conn.execute("SELECT COUNT(*) FROM mv_mode_category_stats").fetchone()[0]

    def get_map_stats(self, min_matches: int = 1) -> list[dict]:
        """Récupère les stats par carte depuis la vue matérialisée.

        Args:
            min_matches: Nombre minimum de matchs pour inclure une carte.

        Returns:
            Liste de dicts avec les stats par carte.
        """
        conn = self._get_connection()

        try:
            result = conn.execute(
                """
                SELECT
                    map_id, map_name, matches_played, wins, losses, ties,
                    avg_kills, avg_deaths, avg_assists, avg_accuracy,
                    avg_kda, win_rate, updated_at
                FROM mv_map_stats
                WHERE matches_played >= ?
                ORDER BY matches_played DESC
                """,
                [min_matches],
            )
            columns = [desc[0] for desc in result.description]
            return [dict(zip(columns, row, strict=False)) for row in result.fetchall()]
        except Exception:
            return []

    def get_mode_category_stats(self) -> list[dict]:
        """Récupère les stats par catégorie de mode depuis la vue matérialisée.

        Returns:
            Liste de dicts avec les stats par catégorie.
        """
        conn = self._get_connection()

        try:
            result = conn.execute(
                """
                SELECT
                    mode_category, matches_played, avg_kills, avg_deaths,
                    avg_assists, avg_kda, avg_accuracy, win_rate, updated_at
                FROM mv_mode_category_stats
                ORDER BY matches_played DESC
                """
            )
            columns = [desc[0] for desc in result.description]
            return [dict(zip(columns, row, strict=False)) for row in result.fetchall()]
        except Exception:
            return []

    def get_global_stats(self) -> dict[str, float]:
        """Récupère les stats globales depuis la vue matérialisée.

        Returns:
            Dict avec stat_key -> stat_value.
        """
        conn = self._get_connection()

        try:
            result = conn.execute("SELECT stat_key, stat_value FROM mv_global_stats")
            return {row[0]: row[1] for row in result.fetchall()}
        except Exception:
            return {}

    def get_session_stats(self, limit: int = 50) -> list[dict]:
        """Récupère les stats par session depuis la vue matérialisée.

        Args:
            limit: Nombre maximum de sessions à retourner.

        Returns:
            Liste de dicts avec les stats par session.
        """
        conn = self._get_connection()

        try:
            result = conn.execute(
                """
                SELECT
                    session_id, match_count, start_time, end_time,
                    total_kills, total_deaths, total_assists,
                    kd_ratio, win_rate, avg_accuracy, avg_life_seconds, updated_at
                FROM mv_session_stats
                ORDER BY start_time DESC
                LIMIT ?
                """,
                [limit],
            )
            columns = [desc[0] for desc in result.description]
            return [dict(zip(columns, row, strict=False)) for row in result.fetchall()]
        except Exception:
            return []

    def has_materialized_views(self) -> bool:
        """Vérifie si les vues matérialisées sont disponibles et remplies.

        Returns:
            True si au moins mv_global_stats contient des données.
        """
        conn = self._get_connection()

        try:
            count = conn.execute("SELECT COUNT(*) FROM mv_global_stats").fetchone()[0]
            return count > 0
        except Exception:
            return False
