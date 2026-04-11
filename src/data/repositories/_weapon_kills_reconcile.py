"""Mixin – opérations d'écriture et de réconciliation weapon_kills.

Extrait de _weapon_kills_repo.py (H9 post-v7 housekeeping).
Contient : insert, mark, get_missing, load pour attribution (static methods).
Les loaders UI (lecture seule) restent dans _weapon_kills_repo.py.
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

import duckdb

from src.utils.xuid import lookup_xuid_for_gamertag

if TYPE_CHECKING:
    from src.analysis._kill_attribution import KillAttribution

logger = logging.getLogger(__name__)

# Bits weapon_kills — copies locales pour éviter import circulaire
_WEAPON_KILLS_BIT = 1 << 21
_WEAPON_KILLS_NO_FILM_BIT = 1 << 22


class WeaponKillsReconcileMixin:
    """Méthodes d'écriture et de réconciliation pour la table ``weapon_kills``."""

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
    def insert_weapon_kill_rows_v2(
        conn: duckdb.DuckDBPyConnection,
        match_id: str,
        attributions: list[KillAttribution],
    ) -> int:
        """Insertion batch v2 avec reconciled_as, attribution_path, player_index.

        Idempotent : DELETE + INSERT pour match_id.
        Quality gate : n'écrase pas si existing_good > new_good.
        """
        if not attributions:
            return 0

        new_good = sum(1 for a in attributions if a.weapon_id is not None)
        existing_good = conn.execute(
            "SELECT COUNT(*) FROM weapon_kills WHERE match_id = ? AND weapon_id IS NOT NULL",
            (match_id,),
        ).fetchone()[0]
        if existing_good > 0 and new_good <= existing_good:
            logger.debug(
                "insert_v2 %s : skip (new_good=%d <= existing_good=%d)",
                match_id[:8],
                new_good,
                existing_good,
            )
            return 0

        if existing_good > 0:
            logger.info(
                "insert_v2 %s : replacing %d existing with %d new (better quality)",
                match_id[:8],
                existing_good,
                new_good,
            )
        conn.execute("DELETE FROM weapon_kills WHERE match_id = ?", (match_id,))
        conn.executemany(
            "INSERT INTO weapon_kills "
            "(match_id, xuid, time_ms, weapon_id, delta_ms, confidence, "
            " swap_detected, delayed_damage, reconciled_as, attribution_path, "
            " player_index) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
            [
                (
                    match_id,
                    a.xuid,
                    a.time_ms,
                    a.weapon_id,
                    a.delta_ms,
                    a.confidence,
                    a.swap_detected,
                    a.delayed_damage,
                    a.reconciled_as,
                    a.attribution_path,
                    a.player_index,
                )
                for a in attributions
            ],
        )
        # Quality threshold logging (Task H)
        total = len(attributions)
        null_wid = sum(1 for a in attributions if a.weapon_id is None)
        formula_a_null = sum(
            1
            for a in attributions
            if a.weapon_id is None and getattr(a, "attribution_path", "") == "formula_a"
        )
        null_ratio = null_wid / total if total else 0.0
        if null_ratio > 0.5:
            logger.warning(
                "insert_v2 %s : %d/%d weapon_id=NULL (%.0f%%) — "
                "%d via formula_a ; données partielles",
                match_id[:8],
                null_wid,
                total,
                null_ratio * 100,
                formula_a_null,
            )
        else:
            logger.debug(
                "insert_v2 %s : %d lignes (%d joueurs, %d NULL wid)",
                match_id[:8],
                total,
                len({a.xuid for a in attributions}),
                null_wid,
            )
        return total

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
        """Résout un gamertag vers son xuid via v_gamertag_lookup."""
        return lookup_xuid_for_gamertag(conn, gamertag)

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
            "SELECT he.time_ms, NULL AS gamertag, he.xuid "
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
        """Charge kills + médailles de TOUS les joueurs en un seul LEFT JOIN DuckDB.

        Remplace N appels à load_player_kills_for_match pour un traitement batch.

        Returns:
            {xuid: [{"time_ms", "gamertag", "xuid", "medals_nearby",
                     "is_melee", "is_grenade"}, ...]}
        """
        from src.analysis.weapon_parser import GRENADE_MEDALS, MELEE_MEDALS

        try:
            rows = conn.execute(
                """
                SELECT
                    k.time_ms,
                    k.xuid,
                    list(m.medal_name) FILTER (WHERE m.medal_name IS NOT NULL) AS medals_nearby
                FROM highlight_events k
                LEFT JOIN (
                    SELECT
                        xuid,
                        time_ms,
                        json_extract_string(raw_json, '$.medal_name') AS medal_name
                    FROM highlight_events
                    WHERE match_id = ? AND event_type = 'medal'
                      AND json_extract_string(raw_json, '$.is_medal') = 'true'
                ) m ON m.xuid = k.xuid AND ABS(m.time_ms - k.time_ms) <= 500
                WHERE k.match_id = ? AND k.event_type = 'kill'
                GROUP BY k.xuid, k.time_ms
                ORDER BY k.xuid, k.time_ms
                """,
                (match_id, match_id),
            ).fetchall()
        except Exception as exc:
            logger.debug("load_all_kills_for_match %s : %s", match_id[:8], exc)
            return {}

        result: dict[str, list[dict]] = {}
        for time_ms, xuid, medals_raw in rows:
            nearby: list[str] = medals_raw if medals_raw else []
            result.setdefault(xuid, []).append(
                {
                    "time_ms": time_ms,
                    "gamertag": None,
                    "xuid": xuid,
                    "medals_nearby": nearby,
                    "is_melee": any(m in MELEE_MEDALS for m in nearby),
                    "is_grenade": any(m in GRENADE_MEDALS for m in nearby),
                }
            )
        logger.debug(
            "load_all_kills_for_match %s : %d kills pour %d joueurs",
            match_id[:8],
            sum(len(v) for v in result.values()),
            len(result),
        )
        return result
