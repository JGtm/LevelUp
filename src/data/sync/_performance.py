"""Mixin — calcul des scores de performance.

Score relatif basé sur l'historique du joueur, écrit dans player_match_enrichment.
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import TYPE_CHECKING

import polars as pl

from src.analysis.performance_config import MIN_MATCHES_FOR_RELATIVE
from src.analysis.performance_score import compute_relative_performance_score

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol
    from src.data.sync.models import MatchStatsRow

logger = logging.getLogger(__name__)

_SHARED_PERF_QUERY = """
SELECT
    mr.match_id, mr.start_time,
    mp.kills, mp.deaths, mp.assists, mp.kda, mp.accuracy,
    mp.time_played_seconds, mp.avg_life_seconds,
    mp.personal_score, mp.damage_dealt,
    mp.rank, mp.team_mmr, mp.enemy_mmr,
    mp.kills_expected, mp.deaths_expected
FROM {registry} mr
JOIN {participants} mp ON mr.match_id = mp.match_id
WHERE mp.xuid = ?
  AND mr.start_time IS NOT NULL
ORDER BY mr.start_time ASC
"""


def _load_matches_for_perf(
    shared_conn: object | None,
    player_conn: object,
    xuid: str,
) -> pl.DataFrame:
    """Charge tous les matchs depuis shared pour le calcul batch de performance.

    Priorité : connexion standalone shared. Fallback : connexion joueur avec
    shared déjà attaché (cas du conflit de handle en post-sync).
    """
    if shared_conn is not None:
        query = _SHARED_PERF_QUERY.format(
            registry="match_registry", participants="match_participants"
        )
        return shared_conn.execute(query, [xuid]).pl()

    logger.debug("batch_perf: shared_conn indisponible, fallback connexion joueur (shared.*)")
    query = _SHARED_PERF_QUERY.format(
        registry="shared.match_registry", participants="shared.match_participants"
    )
    try:
        return player_conn.execute(query, [xuid]).pl()
    except Exception as e:
        logger.warning("batch_perf: fallback connexion joueur échoué: %s", e)
        return pl.DataFrame()


def _compute_perf_updates(
    all_matches_df: pl.DataFrame,
    match_ids: list[str],
    existing_match_ids: set[str],
) -> list[tuple[str, float, str]] | None:
    """Calcule les updates de performance_score pour les matchs sans score.

    Returns:
        Liste de (match_id, score, updated_at) ou None si tous les scores existent.
    """
    null_mask = [mid not in existing_match_ids for mid in match_ids]
    if not any(null_mask):
        return None

    updates: list[tuple[str, float, str]] = []
    now = str(datetime.now(timezone.utc))

    for i in range(len(all_matches_df)):
        if not null_mask[i] or i < MIN_MATCHES_FOR_RELATIVE:
            continue
        row = all_matches_df.row(i, named=True)
        match_dict = {
            "kills": row.get("kills") or 0,
            "deaths": row.get("deaths") or 0,
            "assists": row.get("assists") or 0,
            "kda": row.get("kda"),
            "accuracy": row.get("accuracy"),
            "time_played_seconds": row.get("time_played_seconds") or 600.0,
            "personal_score": row.get("personal_score"),
            "damage_dealt": row.get("damage_dealt"),
            "rank": row.get("rank"),
            "team_mmr": row.get("team_mmr"),
            "enemy_mmr": row.get("enemy_mmr"),
            "kills_expected": row.get("kills_expected"),
            "deaths_expected": row.get("deaths_expected"),
        }
        _window = 50
        _start = max(0, i - _window)
        score = compute_relative_performance_score(
            match_dict, all_matches_df.slice(_start, i - _start)
        )
        if score is not None:
            updates.append((match_ids[i], score, now))

    return updates


class PerformanceMixin:
    """Méthodes de calcul du score de performance."""

    def _load_perf_history_df(
        self: _SyncProtocol, match_id: str, current_start_time_str: str
    ) -> pl.DataFrame | None:
        """Charge les 50 derniers matchs d'historique pour le calcul de performance."""
        shared_conn = self._get_shared_connection()
        if shared_conn is None:
            logger.debug("shared_connection indisponible pour calcul score")
            return None
        return shared_conn.execute(
            """
            SELECT match_id, start_time,
                   kills, deaths, assists, kda, accuracy,
                   time_played_seconds, avg_life_seconds,
                   personal_score, damage_dealt,
                   rank, team_mmr, enemy_mmr,
                   kills_expected, deaths_expected
            FROM (
                SELECT
                    mr.match_id, mr.start_time,
                    mp.kills, mp.deaths, mp.assists, mp.kda, mp.accuracy,
                    mp.time_played_seconds, mp.avg_life_seconds,
                    mp.personal_score, mp.damage_dealt,
                    mp.rank, mp.team_mmr, mp.enemy_mmr,
                    mp.kills_expected, mp.deaths_expected
                FROM match_registry mr
                JOIN match_participants mp ON mr.match_id = mp.match_id
                WHERE mp.xuid = ?
                  AND mr.match_id != ?
                  AND mr.start_time IS NOT NULL
                  AND mr.start_time < CAST(? AS TIMESTAMP)
                ORDER BY mr.start_time DESC
                LIMIT 50
            ) sub
            ORDER BY start_time ASC
            """,
            (self._xuid, match_id, current_start_time_str),
        ).pl()

    def _compute_performance_score(
        self: _SyncProtocol, match_id: str, match_row: MatchStatsRow
    ) -> float | None:
        """Calcule le score de performance relatif pour un match.

        Returns:
            Score relatif (float) ou None si données insuffisantes.
        """
        current_start_time = match_row.start_time
        if current_start_time is None:
            logger.debug("Pas de start_time pour %s, skip calcul score", match_id)
            return None

        if isinstance(current_start_time, datetime):
            current_start_time_str = current_start_time.isoformat()
        else:
            current_start_time_str = str(current_start_time)

        history_df = self._load_perf_history_df(match_id, current_start_time_str)
        if (
            history_df is None
            or history_df.is_empty()
            or len(history_df) < MIN_MATCHES_FOR_RELATIVE
        ):
            logger.debug(
                "Pas assez d'historique pour calculer le score (%s matchs)",
                0 if history_df is None else len(history_df),
            )
            return None

        match_dict = {
            "kills": match_row.kills or 0,
            "deaths": match_row.deaths or 0,
            "assists": match_row.assists or 0,
            "kda": match_row.kda,
            "accuracy": match_row.accuracy,
            "time_played_seconds": match_row.time_played_seconds or 600.0,
            "personal_score": getattr(match_row, "personal_score", None),
            "damage_dealt": getattr(match_row, "damage_dealt", None),
            "rank": getattr(match_row, "rank", None),
            "team_mmr": getattr(match_row, "team_mmr", None),
            "enemy_mmr": getattr(match_row, "enemy_mmr", None),
            "kills_expected": getattr(match_row, "kills_expected", None),
            "deaths_expected": getattr(match_row, "deaths_expected", None),
        }

        return compute_relative_performance_score(match_dict, history_df)

    def _update_performance_score(
        self: _SyncProtocol, match_id: str, match_row: MatchStatsRow
    ) -> None:
        """Orchestre le calcul et le stockage du score de performance.

        Vérifie si le score existe déjà, calcule via ``_compute_performance_score``,
        puis écrit dans player_match_enrichment.
        """
        try:
            conn = self._get_connection()

            existing = conn.execute(
                "SELECT performance_score FROM player_match_enrichment WHERE match_id = ?",
                (match_id,),
            ).fetchone()

            if existing and existing[0] is not None:
                logger.debug("Score de performance déjà présent pour %s", match_id)
                return

            score = self._compute_performance_score(match_id, match_row)

            if score is not None:
                now = datetime.now(timezone.utc)
                conn.execute(
                    """INSERT INTO player_match_enrichment (match_id, performance_score, updated_at)
                    VALUES (?, ?, ?)
                    ON CONFLICT (match_id) DO UPDATE SET
                        performance_score = EXCLUDED.performance_score,
                        updated_at = EXCLUDED.updated_at
                    """,
                    (match_id, score, now),
                )
                logger.debug("Score de performance calculé pour %s: %.1f", match_id, score)
            else:
                logger.debug("Impossible de calculer le score pour %s", match_id)

        except Exception as e:
            # Ne pas bloquer la synchronisation si le calcul échoue
            logger.warning("Erreur calcul score performance pour %s: %s", match_id, e)

    def batch_compute_performance_scores(self: _SyncProtocol, *, force: bool = False) -> int:  # noqa: C901, PLR0912
        """Calcule les performance_score pour tous les matchs où il est NULL.

        Exécuté post-sync pour ne pas bloquer l'insertion des matchs.
        Utilise le calcul vectorisé de compute_relative_performance_score()
        avec un chargement unique de l'historique complet.

        Architecture v5.1 :
            team_mmr et enemy_mmr sont lus directement depuis
            shared.match_participants (mp.enemy_mmr). Corrigé v5.1 :
            remplace l'ancienne sous-requête corrélée.

        Returns:
            Nombre de matchs mis à jour.
        """
        try:
            if not self._xuid:
                logger.warning("xuid manquant pour batch performance scores")
                return 0

            conn = self._get_connection()
            shared_conn = self._get_shared_connection()
            all_matches_df = _load_matches_for_perf(shared_conn, conn, self._xuid)

            if all_matches_df.is_empty():
                return 0

            # 2. Charger les performance_score existants depuis la DB joueur
            try:
                existing_scores_df = conn.execute(
                    """SELECT match_id, performance_score
                       FROM player_match_enrichment
                       WHERE performance_score IS NOT NULL"""
                ).pl()
                existing_match_ids = set(existing_scores_df["match_id"].to_list())
            except Exception:
                # Table peut ne pas exister ou être vide
                existing_match_ids = set()

            # 3-4. Identifier les matchs sans score et calculer
            match_ids = all_matches_df["match_id"].to_list()
            existing_match_ids_set = set() if force else set(existing_match_ids)
            updates = _compute_perf_updates(all_matches_df, match_ids, existing_match_ids_set)

            if updates is None:
                logger.info("Tous les matchs ont déjà un performance_score")
                return 0

            # 5. Batch UPSERT dans player_match_enrichment
            if updates:
                for match_id, score, updated_at in updates:
                    conn.execute(
                        """INSERT INTO player_match_enrichment (match_id, performance_score, updated_at)
                        VALUES (?, ?, ?)
                        ON CONFLICT (match_id) DO UPDATE SET
                            performance_score = EXCLUDED.performance_score,
                            updated_at = EXCLUDED.updated_at
                        """,
                        (match_id, score, updated_at),
                    )
                conn.commit()
                logger.info("Performance scores batch : %s matchs mis à jour", len(updates))

            return len(updates)

        except Exception as e:
            logger.warning("Erreur batch calcul performance scores : %s", e)
            return 0
