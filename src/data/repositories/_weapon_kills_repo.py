"""Mixin – requêtes weapon_kills (shared_matches_v2.duckdb).

Schéma v5.7+ per-kill : (match_id, xuid, time_ms, weapon_id, delta_ms,
confidence, swap_detected, delayed_damage, reconciled_as, attribution_path,
player_index).
weapon_id est un UBIGINT (uint64 film), NULL si non résolu.
reconciled_as stocke le sentinel API sans écraser weapon_id (v2).

Loaders UI (lecture seule) — opérations d'écriture/réconciliation
dans _weapon_kills_reconcile.py.
"""

from __future__ import annotations

import logging

import polars as pl

from src.data.repositories._arrow_bridge import result_to_polars
from src.data.repositories._weapon_kills_reconcile import (
    _WEAPON_KILLS_BIT,  # noqa: F401 — re-export pour les callers existants
    _WEAPON_KILLS_NO_FILM_BIT,  # noqa: F401 — re-export pour les callers existants
    WeaponKillsReconcileMixin,
)

logger = logging.getLogger(__name__)


class WeaponKillsMixin(WeaponKillsReconcileMixin):
    """Loaders UI de la table ``weapon_kills`` + opérations de réconciliation héritées.

    Les méthodes d'écriture/reconciliation sont physiquement dans
    WeaponKillsReconcileMixin (_weapon_kills_reconcile.py).
    """

    # ── Match-level ──────────────────────────────────────────────────────

    def load_weapon_kills_for_match(self, match_id: str) -> pl.DataFrame:
        """Charge les kills agrégés par arme pour tous les joueurs d'un match.

        Returns:
            DataFrame (xuid, weapon_id, kills) trié par kills DESC.
        """
        try:
            conn = self._get_connection()
            result = conn.execute(
                "SELECT xuid, effective_weapon_id AS weapon_id, "
                "COUNT(*)::INTEGER AS kills "
                "FROM shared.v_weapon_kills "
                "WHERE match_id = ? AND effective_weapon_id IS NOT NULL "
                "GROUP BY xuid, effective_weapon_id "
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
                "  SELECT xuid, effective_weapon_id AS weapon_id, "
                "    COUNT(*)::INTEGER AS kills,"
                "    ROW_NUMBER() OVER (PARTITION BY xuid ORDER BY COUNT(*) DESC) AS rn"
                "  FROM shared.v_weapon_kills"
                "  WHERE match_id = ? AND effective_weapon_id IS NOT NULL"
                "  GROUP BY xuid, effective_weapon_id"
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
                    f"SELECT match_id, effective_weapon_id AS weapon_id, "
                    f"COUNT(*)::INTEGER AS kills "
                    f"FROM shared.v_weapon_kills "
                    f"WHERE xuid = ? AND match_id IN ({placeholders}) "
                    f"AND effective_weapon_id IS NOT NULL "
                    f"GROUP BY match_id, effective_weapon_id "
                    f"ORDER BY match_id, kills DESC",
                    [xuid, *match_ids],
                )
            else:
                result = conn.execute(
                    "SELECT match_id, effective_weapon_id AS weapon_id, "
                    "COUNT(*)::INTEGER AS kills "
                    "FROM shared.v_weapon_kills "
                    "WHERE xuid = ? AND effective_weapon_id IS NOT NULL "
                    "GROUP BY match_id, effective_weapon_id "
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
                    f"SELECT effective_weapon_id AS weapon_id, "
                    f"COUNT(*)::INTEGER AS total_kills "
                    f"FROM shared.v_weapon_kills "
                    f"WHERE xuid = ? AND match_id IN ({placeholders}) "
                    f"AND effective_weapon_id IS NOT NULL "
                    f"GROUP BY effective_weapon_id ORDER BY total_kills DESC",
                    [xuid, *match_ids],
                )
            else:
                result = conn.execute(
                    "SELECT effective_weapon_id AS weapon_id, "
                    "COUNT(*)::INTEGER AS total_kills "
                    "FROM shared.v_weapon_kills "
                    "WHERE xuid = ? AND effective_weapon_id IS NOT NULL "
                    "GROUP BY effective_weapon_id ORDER BY total_kills DESC",
                    (xuid,),
                )
            return result_to_polars(result)
        except Exception as exc:
            logger.debug("weapon_kills_agg player %s : %s", xuid, exc)
            return pl.DataFrame(schema={"weapon_id": pl.UInt64, "total_kills": pl.Int32})

    def load_total_kills_for_player(self, xuid: str, match_ids: list[str]) -> int:
        """Retourne le total de kills API (match_participants) pour un joueur sur des matchs."""
        try:
            conn = self._get_connection()
            xuid_str = str(xuid).strip()
            placeholders = ", ".join("?" * len(match_ids))
            row = conn.execute(
                f"SELECT COALESCE(SUM(kills), 0) "
                f"FROM shared.match_participants "
                f"WHERE xuid = ? AND match_id IN ({placeholders})",
                [xuid_str, *match_ids],
            ).fetchone()
            return int(row[0]) if row else 0
        except Exception as exc:
            logger.debug("load_total_kills_for_player xuid=%s : %s", xuid, exc)
            return 0

    def load_grenade_melee_kills(
        self, xuid: str, match_ids: list[str] | None = None
    ) -> tuple[int, int]:
        """Retourne (grenade_kills, melee_kills) depuis shared.match_participants.

        Agrège sur un ensemble de matchs si match_ids est fourni, sinon sur tous
        les matchs du joueur. Utilisé par l'UI pour enrichir les tableaux d'armes.

        Returns:
            Tuple (grenade_kills, melee_kills).
        """
        try:
            conn = self._get_connection()
            xuid_str = str(xuid).strip()
            if match_ids:
                placeholders = ", ".join("?" * len(match_ids))
                row = conn.execute(
                    f"SELECT COALESCE(SUM(grenade_kills), 0), "
                    f"COALESCE(SUM(melee_kills), 0) "
                    f"FROM shared.match_participants "
                    f"WHERE xuid = ? AND match_id IN ({placeholders})",
                    [xuid_str, *match_ids],
                ).fetchone()
            else:
                row = conn.execute(
                    "SELECT COALESCE(SUM(grenade_kills), 0), "
                    "COALESCE(SUM(melee_kills), 0) "
                    "FROM shared.match_participants WHERE xuid = ?",
                    [xuid_str],
                ).fetchone()
            if row:
                return int(row[0]), int(row[1])
        except Exception as exc:
            logger.debug("load_grenade_melee_kills xuid=%s : %s", xuid, exc)
        return 0, 0
