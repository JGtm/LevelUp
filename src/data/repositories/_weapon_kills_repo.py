"""Mixin – requêtes weapon_kills (shared_matches.duckdb).

Schéma v5.7 per-kill : (match_id, xuid, time_ms, weapon_id, delta_ms,
confidence, swap_detected, delayed_damage).
weapon_id est un UBIGINT (uint64 filmshell), NULL si non résolu.
"""

from __future__ import annotations

import logging

import duckdb
import polars as pl

from src.data.repositories._arrow_bridge import result_to_polars

logger = logging.getLogger(__name__)

# Bits weapon_kills — copies locales pour éviter import circulaire
_WEAPON_KILLS_BIT = 1 << 21
_WEAPON_KILLS_NO_FILM_BIT = 1 << 22


class WeaponKillsMixin:
    """Méthodes d'accès à la table ``weapon_kills`` (shared)."""

    # ── Match-level ──────────────────────────────────────────────────────

    def load_weapon_kills_for_match(self, match_id: str) -> pl.DataFrame:
        """Charge les kills agrégés par arme pour tous les joueurs d'un match.

        Returns:
            DataFrame (xuid, weapon_id, kills) trié par kills DESC.
        """
        try:
            conn = self._get_connection()
            result = conn.execute(
                "SELECT xuid, weapon_id, COUNT(*)::INTEGER AS kills "
                "FROM shared.weapon_kills "
                "WHERE match_id = ? AND weapon_id IS NOT NULL "
                "GROUP BY xuid, weapon_id "
                "ORDER BY kills DESC",
                (match_id,),
            )
            return result_to_polars(result)
        except Exception as exc:
            logger.debug("weapon_kills match %s : %s", match_id, exc)
            return pl.DataFrame(schema={"xuid": pl.Utf8, "weapon_id": pl.UInt64, "kills": pl.Int32})

    def load_top_weapon_per_player(self, match_id: str) -> dict[str, tuple[int, int]]:
        """Retourne l'arme avec le plus de kills par joueur.

        Returns:
            ``{xuid: (weapon_id, kills)}`` — une entrée par joueur.
        """
        try:
            conn = self._get_connection()
            rows = conn.execute(
                "SELECT xuid, weapon_id, kills FROM ("
                "  SELECT xuid, weapon_id, COUNT(*)::INTEGER AS kills,"
                "    ROW_NUMBER() OVER (PARTITION BY xuid ORDER BY COUNT(*) DESC) AS rn"
                "  FROM shared.weapon_kills"
                "  WHERE match_id = ? AND weapon_id IS NOT NULL"
                "  GROUP BY xuid, weapon_id"
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
            DataFrame (match_id, weapon_id, kills).
        """
        try:
            conn = self._get_connection()
            if match_ids:
                placeholders = ", ".join("?" for _ in match_ids)
                result = conn.execute(
                    f"SELECT match_id, weapon_id, COUNT(*)::INTEGER AS kills "
                    f"FROM shared.weapon_kills "
                    f"WHERE xuid = ? AND match_id IN ({placeholders}) "
                    f"AND weapon_id IS NOT NULL "
                    f"GROUP BY match_id, weapon_id "
                    f"ORDER BY match_id, kills DESC",
                    [xuid, *match_ids],
                )
            else:
                result = conn.execute(
                    "SELECT match_id, weapon_id, COUNT(*)::INTEGER AS kills "
                    "FROM shared.weapon_kills "
                    "WHERE xuid = ? AND weapon_id IS NOT NULL "
                    "GROUP BY match_id, weapon_id "
                    "ORDER BY match_id, kills DESC",
                    (xuid,),
                )
            return result_to_polars(result)
        except Exception as exc:
            logger.debug("weapon_kills player %s : %s", xuid, exc)
            return pl.DataFrame(
                schema={"match_id": pl.Utf8, "weapon_id": pl.UInt64, "kills": pl.Int32}
            )

    def load_weapon_kills_aggregated(
        self, xuid: str, match_ids: list[str] | None = None
    ) -> pl.DataFrame:
        """Agrège les kills par arme d'un joueur (total sur plusieurs matchs).

        Returns:
            DataFrame (weapon_id, total_kills) trié par total_kills DESC.
        """
        try:
            conn = self._get_connection()
            if match_ids:
                placeholders = ", ".join("?" for _ in match_ids)
                result = conn.execute(
                    f"SELECT weapon_id, COUNT(*)::INTEGER AS total_kills "
                    f"FROM shared.weapon_kills "
                    f"WHERE xuid = ? AND match_id IN ({placeholders}) "
                    f"AND weapon_id IS NOT NULL "
                    f"GROUP BY weapon_id ORDER BY total_kills DESC",
                    [xuid, *match_ids],
                )
            else:
                result = conn.execute(
                    "SELECT weapon_id, COUNT(*)::INTEGER AS total_kills "
                    "FROM shared.weapon_kills "
                    "WHERE xuid = ? AND weapon_id IS NOT NULL "
                    "GROUP BY weapon_id ORDER BY total_kills DESC",
                    (xuid,),
                )
            return result_to_polars(result)
        except Exception as exc:
            logger.debug("weapon_kills_agg player %s : %s", xuid, exc)
            return pl.DataFrame(schema={"weapon_id": pl.UInt64, "total_kills": pl.Int32})

    # ── Write operations (accept explicit connection) ────────────────────

    @staticmethod
    def insert_weapon_kill_rows(
        conn: duckdb.DuckDBPyConnection,
        match_id: str,
        xuid: str,
        kill_rows: list[dict],
    ) -> int:
        """Insère les lignes per-kill (schéma v5.7).

        Chaque dict doit contenir : time_ms, weapon_id (int|None), delta_ms,
        confidence, swap_detected, delayed_damage.
        Ne remplace pas des données existantes de meilleure qualité.
        Retourne le nombre de lignes insérées.
        """
        if not kill_rows:
            return 0
        new_good = sum(1 for r in kill_rows if r.get("weapon_id") is not None)
        if new_good < len(kill_rows):
            existing_good = conn.execute(
                "SELECT COUNT(*) FROM weapon_kills WHERE match_id = ? AND xuid = ?"
                " AND weapon_id IS NOT NULL",
                (match_id, xuid),
            ).fetchone()[0]
            if new_good <= existing_good:
                logger.debug(
                    "insert_weapon_kill_rows %s %s : skip (new_good=%d <= existing_good=%d)",
                    match_id[:8],
                    xuid[:8],
                    new_good,
                    existing_good,
                )
                return 0
        conn.execute(
            "DELETE FROM weapon_kills WHERE match_id = ? AND xuid = ?",
            (match_id, xuid),
        )
        conn.executemany(
            "INSERT INTO weapon_kills "
            "(match_id, xuid, time_ms, weapon_id, delta_ms, confidence, "
            " swap_detected, delayed_damage) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
            [
                (
                    match_id,
                    xuid,
                    r["time_ms"],
                    r.get("weapon_id"),
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
    def mark_weapon_no_film(
        conn: duckdb.DuckDBPyConnection,
        match_id: str,
    ) -> None:
        """Pose le bit WEAPON_KILLS_NO_FILM (film 404/expiré).

        N'affecte PAS WEAPON_KILLS : le match pourra être re-tenté via
        --force-no-film si les films redeviennent disponibles sur Halo servers.
        """
        conn.execute(
            "UPDATE match_registry "
            "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
            "WHERE match_id = ?",
            (_WEAPON_KILLS_NO_FILM_BIT, match_id),
        )
        logger.debug(
            "mark_weapon_no_film %s : bit 0x%x posé", match_id[:8], _WEAPON_KILLS_NO_FILM_BIT
        )

    @staticmethod
    def get_matches_missing_weapons(
        conn: duckdb.DuckDBPyConnection,
        xuid: str,
        limit: int = 5,
        *,
        force: bool = False,
    ) -> list[str]:
        """Retourne les match_ids récents où le joueur n'a pas de bonnes données d'armes.

        Inclut :
        - les matchs sans WEAPON_KILLS_BIT (jamais traités)
        - les matchs avec WEAPON_KILLS_BIT mais où le joueur n'a que du UNKNOWN
          (traité depuis le POV d'un coéquipier, T1 ayant échoué)
        Le flag ``force`` ignore toutes les conditions et retourne tous les matchs.
        """
        if force:
            rows = conn.execute(
                """
                SELECT DISTINCT mp.match_id
                FROM match_participants mp
                JOIN match_registry r ON r.match_id = mp.match_id
                WHERE mp.xuid = ?
                  AND r.is_firefight = FALSE
                ORDER BY r.start_time DESC
                LIMIT ?
                """,
                (xuid, limit),
            ).fetchall()
        else:
            rows = conn.execute(
                f"""
                SELECT DISTINCT mp.match_id
                FROM match_participants mp
                JOIN match_registry r ON r.match_id = mp.match_id
                WHERE mp.xuid = ?
                  AND r.is_firefight = FALSE
                  AND (COALESCE(r.backfill_completed, 0) & {_WEAPON_KILLS_BIT}) = 0
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

    @staticmethod
    def load_all_kills_for_match(
        conn: duckdb.DuckDBPyConnection,
        match_id: str,
    ) -> dict[str, list[dict]]:
        """Charge kills + médailles de TOUS les joueurs en 2 requêtes.

        Remplace N appels à load_player_kills_for_match pour un traitement batch.

        Returns:
            {xuid: [{"time_ms", "gamertag", "xuid", "medals_nearby",
                     "is_melee", "is_grenade"}, ...]}
        """
        from src.analysis.weapon_parser import GRENADE_MEDALS, MELEE_MEDALS

        try:
            kill_rows = conn.execute(
                "SELECT he.time_ms, he.gamertag, he.xuid "
                "FROM highlight_events he "
                "WHERE he.match_id = ? AND he.event_type = 'kill' "
                "ORDER BY he.xuid, he.time_ms",
                (match_id,),
            ).fetchall()

            medal_rows = conn.execute(
                "SELECT he.xuid, he.time_ms, "
                "json_extract_string(he.raw_json, '$.medal_name') AS medal_name "
                "FROM highlight_events he "
                "WHERE he.match_id = ? AND he.event_type = 'medal' "
                "AND json_extract_string(he.raw_json, '$.is_medal') = 'true' "
                "ORDER BY he.xuid, he.time_ms",
                (match_id,),
            ).fetchall()
        except Exception as exc:
            logger.debug("load_all_kills_for_match %s : %s", match_id[:8], exc)
            return {}

        medals_by_xuid: dict[str, list[tuple[int, str]]] = {}
        for xuid, t, name in medal_rows:
            if name:
                medals_by_xuid.setdefault(xuid, []).append((t, name))

        result: dict[str, list[dict]] = {}
        for time_ms, gt, xuid in kill_rows:
            nearby = [
                name for (mt, name) in medals_by_xuid.get(xuid, []) if abs(mt - time_ms) <= 500
            ]
            result.setdefault(xuid, []).append(
                {
                    "time_ms": time_ms,
                    "gamertag": gt,
                    "xuid": xuid,
                    "medals_nearby": nearby,
                    "is_melee": any(m in MELEE_MEDALS for m in nearby),
                    "is_grenade": any(m in GRENADE_MEDALS for m in nearby),
                }
            )
        logger.debug(
            "load_all_kills_for_match %s : %d kills pour %d joueurs",
            match_id[:8],
            len(kill_rows),
            len(result),
        )
        return result
