"""Mixin – requêtes weapon_kills (shared_matches.duckdb).

Schéma v5.6 per-kill : (match_id, xuid, time_ms, weapon_name, delta_ms,
confidence, swap_detected, delayed_damage).
"""

from __future__ import annotations

import logging

import duckdb
import polars as pl

from src.data.repositories._arrow_bridge import result_to_polars

logger = logging.getLogger(__name__)

# Bit WEAPON_KILLS (1 << 21) — copie locale pour éviter import circulaire
_WEAPON_KILLS_BIT = 1 << 21


class WeaponKillsMixin:
    """Méthodes d'accès à la table ``weapon_kills`` (shared)."""

    # ── Match-level ──────────────────────────────────────────────────────

    def load_weapon_kills_for_match(self, match_id: str) -> pl.DataFrame:
        """Charge les kills agrégés par arme pour tous les joueurs d'un match.

        Returns:
            DataFrame (xuid, weapon_name, kills) trié par kills DESC.
        """
        try:
            conn = self._get_connection()
            result = conn.execute(
                "SELECT xuid, weapon_name, COUNT(*)::INTEGER AS kills "
                "FROM shared.weapon_kills "
                "WHERE match_id = ? "
                "GROUP BY xuid, weapon_name "
                "ORDER BY kills DESC",
                (match_id,),
            )
            return result_to_polars(result)
        except Exception as exc:
            logger.debug("weapon_kills match %s : %s", match_id, exc)
            return pl.DataFrame(schema={"xuid": pl.Utf8, "weapon_name": pl.Utf8, "kills": pl.Int32})

    def load_top_weapon_per_player(self, match_id: str) -> dict[str, tuple[str, int]]:
        """Retourne l'arme avec le plus de kills par joueur.

        Returns:
            ``{xuid: (weapon_name, kills)}`` — une entrée par joueur.
        """
        try:
            conn = self._get_connection()
            rows = conn.execute(
                "SELECT xuid, weapon_name, kills FROM ("
                "  SELECT xuid, weapon_name, COUNT(*)::INTEGER AS kills,"
                "    ROW_NUMBER() OVER (PARTITION BY xuid ORDER BY COUNT(*) DESC) AS rn"
                "  FROM shared.weapon_kills"
                "  WHERE match_id = ?"
                "  GROUP BY xuid, weapon_name"
                ") sub WHERE rn = 1",
                (match_id,),
            ).fetchall()
            return {r[0]: (r[1], r[2]) for r in rows}
        except Exception as exc:
            logger.debug("top_weapon match %s : %s", match_id, exc)
            return {}

    # ── Player-level ─────────────────────────────────────────────────────

    def load_weapon_kills_for_player(
        self, xuid: str, match_ids: list[str] | None = None
    ) -> pl.DataFrame:
        """Charge les kills par arme d'un joueur sur un ensemble de matchs.

        Returns:
            DataFrame (match_id, weapon_name, kills).
        """
        try:
            conn = self._get_connection()
            if match_ids:
                placeholders = ", ".join("?" for _ in match_ids)
                result = conn.execute(
                    f"SELECT match_id, weapon_name, COUNT(*)::INTEGER AS kills "
                    f"FROM shared.weapon_kills "
                    f"WHERE xuid = ? AND match_id IN ({placeholders}) "
                    f"GROUP BY match_id, weapon_name "
                    f"ORDER BY match_id, kills DESC",
                    [xuid, *match_ids],
                )
            else:
                result = conn.execute(
                    "SELECT match_id, weapon_name, COUNT(*)::INTEGER AS kills "
                    "FROM shared.weapon_kills "
                    "WHERE xuid = ? "
                    "GROUP BY match_id, weapon_name "
                    "ORDER BY match_id, kills DESC",
                    (xuid,),
                )
            return result_to_polars(result)
        except Exception as exc:
            logger.debug("weapon_kills player %s : %s", xuid, exc)
            return pl.DataFrame(
                schema={"match_id": pl.Utf8, "weapon_name": pl.Utf8, "kills": pl.Int32}
            )

    def load_weapon_kills_aggregated(
        self, xuid: str, match_ids: list[str] | None = None
    ) -> pl.DataFrame:
        """Agrège les kills par arme d'un joueur (total sur plusieurs matchs).

        Returns:
            DataFrame (weapon_name, total_kills) trié par total_kills DESC.
        """
        try:
            conn = self._get_connection()
            if match_ids:
                placeholders = ", ".join("?" for _ in match_ids)
                result = conn.execute(
                    f"SELECT weapon_name, COUNT(*)::INTEGER AS total_kills "
                    f"FROM shared.weapon_kills "
                    f"WHERE xuid = ? AND match_id IN ({placeholders}) "
                    f"GROUP BY weapon_name ORDER BY total_kills DESC",
                    [xuid, *match_ids],
                )
            else:
                result = conn.execute(
                    "SELECT weapon_name, COUNT(*)::INTEGER AS total_kills "
                    "FROM shared.weapon_kills "
                    "WHERE xuid = ? "
                    "GROUP BY weapon_name ORDER BY total_kills DESC",
                    (xuid,),
                )
            return result_to_polars(result)
        except Exception as exc:
            logger.debug("weapon_kills_agg player %s : %s", xuid, exc)
            return pl.DataFrame(schema={"weapon_name": pl.Utf8, "total_kills": pl.Int32})

    # ── Write operations (accept explicit connection) ────────────────────

    @staticmethod
    def insert_weapon_kill_rows(
        conn: duckdb.DuckDBPyConnection,
        match_id: str,
        xuid: str,
        kill_rows: list[dict],
    ) -> int:
        """Insère les lignes per-kill (schéma v5.6).

        Chaque dict doit contenir : time_ms, weapon_name, delta_ms (ou None),
        confidence, swap_detected, delayed_damage.
        Retourne le nombre de lignes insérées.
        """
        if not kill_rows:
            return 0
        conn.execute(
            "DELETE FROM weapon_kills WHERE match_id = ? AND xuid = ?",
            (match_id, xuid),
        )
        conn.executemany(
            "INSERT INTO weapon_kills "
            "(match_id, xuid, time_ms, weapon_name, delta_ms, confidence, "
            " swap_detected, delayed_damage) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
            [
                (
                    match_id,
                    xuid,
                    r["time_ms"],
                    r["weapon_name"],
                    r.get("delta_ms"),
                    r.get("confidence", "none"),
                    bool(r.get("swap_detected", False)),
                    bool(r.get("delayed_damage", False)),
                )
                for r in kill_rows
            ],
        )
        logger.debug(
            "insert_weapon_kill_rows %s %s : %d lignes", match_id[:8], xuid[:8], len(kill_rows)
        )
        return len(kill_rows)

    @staticmethod
    def mark_weapon_backfill_done(
        conn: duckdb.DuckDBPyConnection,
        match_id: str,
    ) -> None:
        """Pose le bit WEAPON_KILLS sur match_registry.backfill_completed."""
        conn.execute(
            "UPDATE match_registry "
            "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
            "WHERE match_id = ?",
            (_WEAPON_KILLS_BIT, match_id),
        )
        logger.debug(
            "mark_weapon_backfill_done %s : bit 0x%x posé", match_id[:8], _WEAPON_KILLS_BIT
        )

    @staticmethod
    def get_matches_missing_weapons(
        conn: duckdb.DuckDBPyConnection,
        xuid: str,
        limit: int = 5,
        *,
        force: bool = False,
    ) -> list[str]:
        """Retourne les match_ids récents sans WEAPON_KILLS (non-Firefight)."""
        bit_filter = (
            "" if force else f"AND (COALESCE(r.backfill_completed, 0) & {_WEAPON_KILLS_BIT}) = 0"
        )
        rows = conn.execute(
            f"""
            SELECT DISTINCT mp.match_id
            FROM match_participants mp
            JOIN match_registry r ON r.match_id = mp.match_id
            WHERE mp.xuid = ?
              AND r.is_firefight = FALSE
              {bit_filter}
            ORDER BY r.start_time DESC
            LIMIT ?
            """,
            (xuid, limit),
        ).fetchall()
        return [r[0] for r in rows]

    @staticmethod
    def get_xuid_by_gamertag(
        conn: duckdb.DuckDBPyConnection,
        gamertag: str,
    ) -> str | None:
        """Résout un gamertag vers son xuid."""
        row = conn.execute(
            "SELECT xuid FROM xuid_aliases WHERE gamertag ILIKE ? LIMIT 1",
            (gamertag,),
        ).fetchone()
        return str(row[0]) if row else None

    @staticmethod
    def load_player_kills_for_match(
        conn: duckdb.DuckDBPyConnection,
        match_id: str,
        xuid: str,
    ) -> list[dict]:
        """Charge les kills du joueur + médailles proches depuis highlight_events.

        Returns:
            Liste de dicts avec time_ms, gamertag, xuid,
            medals_nearby, is_melee, is_grenade.
        """
        from src.analysis.weapon_parser import GRENADE_MEDALS, MELEE_MEDALS

        kill_rows = conn.execute(
            "SELECT he.time_ms, he.gamertag, he.xuid "
            "FROM highlight_events he "
            "WHERE he.match_id = ? AND he.xuid = ? "
            "AND he.event_type = 'kill' "
            "ORDER BY he.time_ms",
            (match_id, xuid),
        ).fetchall()

        medal_rows = conn.execute(
            "SELECT he.time_ms, "
            "json_extract_string(he.raw_json, '$.medal_name') AS medal_name "
            "FROM highlight_events he "
            "WHERE he.match_id = ? AND he.xuid = ? "
            "AND he.event_type = 'medal' "
            "AND json_extract_string(he.raw_json, '$.is_medal') = 'true' "
            "ORDER BY he.time_ms",
            (match_id, xuid),
        ).fetchall()

        medals_by_time = [(t, name) for t, name in medal_rows if name]

        kills = []
        for time_ms, gt, xuid_val in kill_rows:
            nearby = [name for (mt, name) in medals_by_time if abs(mt - time_ms) <= 500]
            kills.append(
                {
                    "time_ms": time_ms,
                    "gamertag": gt,
                    "xuid": xuid_val,
                    "medals_nearby": nearby,
                    "is_melee": any(m in MELEE_MEDALS for m in nearby),
                    "is_grenade": any(m in GRENADE_MEDALS for m in nearby),
                }
            )
        return kills
