"""Mixin pour les données de carrière et progression de rang.

Regroupe les méthodes d'accès à :
- career_progression (rang XP, snapshots)
- match_skill_rank (LUSR/CSR par playlist)
- match_registry + match_participants (dates de matchs autour d'un sync)
"""

from __future__ import annotations

import logging
from datetime import datetime

logger = logging.getLogger(__name__)


class CareerMixin:
    """Mixin fournissant les données de carrière pour DuckDBRepository."""

    # ------------------------------------------------------------------ #
    # Progression de rang (career_progression)                            #
    # ------------------------------------------------------------------ #

    def load_career_data(self) -> dict | None:
        """Charge le dernier snapshot de rang depuis career_progression.

        Returns:
            Dict avec rank, rank_name, rank_tier, current_xp, etc. ou None.
        """
        conn = self._get_connection()
        try:
            result = conn.execute(
                """
                SELECT rank, rank_name, rank_tier, current_xp,
                       xp_for_next_rank, xp_total, is_max_rank,
                       adornment_path, recorded_at
                FROM career_progression
                WHERE xuid = ?
                ORDER BY recorded_at DESC
                LIMIT 1
                """,
                [str(self._xuid)],
            ).fetchone()
            if result:
                return {
                    "rank": result[0],
                    "rank_name": result[1],
                    "rank_tier": result[2],
                    "current_xp": result[3],
                    "xp_for_next_rank": result[4],
                    "xp_total": result[5],
                    "is_max_rank": bool(result[6]),
                    "adornment_path": result[7],
                    "recorded_at": result[8],
                }
        except Exception:
            logger.debug("load_career_data: introuvable pour xuid=%s", self._xuid, exc_info=True)
        return None

    def load_career_history(self, limit: int = 50) -> list[dict]:
        """Charge l'historique de snapshots XP ordonnés par date croissante.

        Args:
            limit: Nombre maximum de snapshots à retourner.

        Returns:
            Liste de dicts ordonnés par recorded_at ASC.
        """
        conn = self._get_connection()
        try:
            rows = conn.execute(
                """
                SELECT rank, rank_name, rank_tier, current_xp,
                       xp_for_next_rank, xp_total, is_max_rank, recorded_at
                FROM career_progression
                WHERE xuid = ?
                ORDER BY recorded_at ASC
                LIMIT ?
                """,
                [str(self._xuid), limit],
            ).fetchall()
            return [
                {
                    "rank": r[0],
                    "rank_name": r[1],
                    "rank_tier": r[2],
                    "current_xp": r[3],
                    "xp_for_next_rank": r[4],
                    "xp_total": r[5],
                    "is_max_rank": bool(r[6]),
                    "recorded_at": r[7],
                }
                for r in rows
            ]
        except Exception:
            logger.debug("load_career_history: introuvable pour xuid=%s", self._xuid, exc_info=True)
            return []

    # ------------------------------------------------------------------ #
    # Matchs autour du premier sync                                       #
    # ------------------------------------------------------------------ #

    def load_pre_sync_match_dates(self, first_sync_at: datetime) -> list[datetime]:
        """Charge les dates de matchs antérieurs au premier sync du joueur.

        Lit depuis shared.match_registry + shared.match_participants.

        Args:
            first_sync_at: Date du premier sync — filtre < cette date.

        Returns:
            Liste de datetime ou [] si shared non disponible.
        """
        if not self._has_shared_table("match_participants"):
            return []
        conn = self._get_connection()
        try:
            rows = conn.execute(
                """
                SELECT mr.start_time
                FROM shared.match_registry mr
                JOIN shared.match_participants mp ON mr.match_id = mp.match_id
                WHERE mp.xuid = ?
                  AND mr.start_time < ?
                ORDER BY mr.start_time ASC
                """,
                [str(self._xuid), first_sync_at],
            ).fetchall()
            return [r[0] for r in rows if r[0] is not None]
        except Exception:
            logger.debug("load_pre_sync_match_dates: erreur xuid=%s", self._xuid, exc_info=True)
            return []

    # ------------------------------------------------------------------ #
    # LUSR / CSR (match_skill_rank)                                       #
    # ------------------------------------------------------------------ #

    def load_lusr_snapshot(self) -> list[dict]:
        """Charge le dernier rating LUSR/CSR par playlist_group.

        Utilise une fenêtre ROW_NUMBER pour le dernier snapshot par groupe.

        Returns:
            Liste de dicts (rating_type, rating_value, tier_label, etc.).
        """
        conn = self._get_connection()
        try:
            rows = conn.execute(
                """
                WITH history AS (
                    SELECT
                        msr.match_id, msr.rating_type, msr.rating_value,
                        msr.tier_label, msr.sub_tier, msr.tier, msr.tier_fr,
                        msr.playlist_group,
                        COALESCE(msr.start_time, msr.updated_at) AS sort_time,
                        msr.rating_value - LAG(msr.rating_value) OVER (
                            PARTITION BY msr.playlist_group
                            ORDER BY COALESCE(msr.start_time, msr.updated_at)
                        ) AS rating_delta,
                        ROW_NUMBER() OVER (
                            PARTITION BY msr.playlist_group
                            ORDER BY COALESCE(msr.start_time, msr.updated_at) DESC
                        ) AS rn
                    FROM match_skill_rank msr
                )
                SELECT rating_type, rating_value, tier_label, sub_tier,
                       tier, tier_fr, rating_delta, playlist_group
                FROM history
                WHERE rn = 1
                ORDER BY playlist_group
                """
            ).fetchall()
            return [
                {
                    "rating_type": r[0],
                    "rating_value": r[1],
                    "tier_label": r[2],
                    "sub_tier": r[3] or 0,
                    "tier": r[4],
                    "tier_fr": r[5],
                    "rating_delta": r[6],
                    "playlist_group": r[7],
                }
                for r in rows
            ]
        except Exception:
            logger.debug("load_lusr_snapshot: introuvable", exc_info=True)
            return []

    def load_lusr_history(self, playlist_group: str | None = None) -> list[dict]:
        """Charge l'historique LUSR/CSR pour le graphe d'évolution.

        Args:
            playlist_group: Si fourni, filtre sur ce groupe de playlist.

        Returns:
            Liste de dicts ordonnés par start_time ASC.
        """
        conn = self._get_connection()
        pg_filter = "AND msr.playlist_group = ?" if playlist_group else ""
        params: list = [playlist_group] if playlist_group else []
        try:
            rows = conn.execute(
                f"""
                SELECT msr.match_id, msr.rating_value, msr.rating_deviation,
                       msr.rating_type, msr.playlist_group, msr.tier_label,
                       COALESCE(msr.start_time, msr.created_at) AS start_time
                FROM match_skill_rank msr
                WHERE 1=1 {pg_filter}
                ORDER BY COALESCE(msr.start_time, msr.created_at) ASC
                """,  # noqa: S608
                params,
            ).fetchall()
            return [
                {
                    "match_id": r[0],
                    "rating_value": r[1],
                    "rating_deviation": r[2],
                    "rating_type": r[3],
                    "playlist_group": r[4],
                    "tier_label": r[5],
                    "start_time": r[6],
                }
                for r in rows
            ]
        except Exception:
            logger.debug("load_lusr_history: erreur (group=%s)", playlist_group, exc_info=True)
            return []

    # ------------------------------------------------------------------ #
    # Enrichissement de match (player_match_enrichment)                   #
    # ------------------------------------------------------------------ #

    def load_is_with_friends_batch(self, match_ids: list[str]) -> dict[str, bool]:
        """Charge le flag escouade/solo pour une liste de match_ids.

        Args:
            match_ids: Liste des match_id à interroger.

        Returns:
            Mapping match_id → is_with_friends (True = escouade).
        """
        if not match_ids:
            return {}
        ph = ", ".join(["?"] * len(match_ids))
        conn = self._get_connection()
        try:
            rows = conn.execute(
                f"SELECT match_id, is_with_friends FROM player_match_enrichment"  # noqa: S608
                f" WHERE match_id IN ({ph})",
                match_ids,
            ).fetchall()
            return {str(r[0]): bool(r[1]) for r in rows if r[1] is not None}
        except Exception:
            logger.debug("load_is_with_friends_batch: erreur", exc_info=True)
            return {}

    def load_post_sync_match_count(self, first_sync_at: datetime) -> int:
        """Compte les matchs du joueur postérieurs au premier sync.

        Args:
            first_sync_at: Date du premier sync — filtre >= cette date.

        Returns:
            Nombre de matchs après le sync, 0 si shared indisponible.
        """
        if not self._has_shared_table("match_participants"):
            return 0
        conn = self._get_connection()
        try:
            row = conn.execute(
                """
                SELECT COUNT(*)
                FROM shared.match_registry mr
                JOIN shared.match_participants mp ON mr.match_id = mp.match_id
                WHERE mp.xuid = ?
                  AND mr.start_time >= ?
                """,
                (str(self._xuid), first_sync_at),
            ).fetchone()
            return int(row[0]) if row else 0
        except Exception:
            logger.debug("load_post_sync_match_count: erreur", exc_info=True)
            return 0

    def load_friends_xuids_csv(self, match_id: str) -> str | None:
        """Lit friends_xuids depuis player_match_enrichment pour un match.

        Args:
            match_id: Identifiant du match.

        Returns:
            Chaîne CSV des xuids amis, ou None si absent.
        """
        conn = self._get_connection()
        try:
            row = conn.execute(
                "SELECT friends_xuids FROM player_match_enrichment WHERE match_id = ? LIMIT 1",
                [match_id],
            ).fetchone()
            return str(row[0]) if row and row[0] else None
        except Exception:
            logger.debug("load_friends_xuids_csv: erreur match=%s", match_id, exc_info=True)
            return None

    def load_skill_ratings_batch(self, match_ids: list[str]) -> dict[str, tuple[float, str]]:
        """Charge les ratings LUSR/CSR pour une liste de match_ids.

        Args:
            match_ids: Liste des match_id à interroger.

        Returns:
            Mapping match_id → (rating_value, rating_type). {} si vide ou erreur.
        """
        if not match_ids:
            return {}
        ph = ", ".join(["?"] * len(match_ids))
        conn = self._get_connection()
        try:
            rows = conn.execute(
                f"SELECT match_id, rating_value, rating_type FROM match_skill_rank"  # noqa: S608
                f" WHERE match_id IN ({ph})",
                match_ids,
            ).fetchall()
            return {str(r[0]): (float(r[1]), str(r[2])) for r in rows}
        except Exception:
            logger.debug("load_skill_ratings_batch: erreur", exc_info=True)
            return {}
