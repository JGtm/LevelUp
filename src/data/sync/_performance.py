"""Mixin — calcul des scores de performance.

Score relatif basé sur l'historique du joueur, écrit dans player_match_enrichment.
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import TYPE_CHECKING

from src.analysis.performance_config import MIN_MATCHES_FOR_RELATIVE
from src.analysis.performance_score import compute_relative_performance_score

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol
    from src.data.sync.models import MatchStatsRow

logger = logging.getLogger(__name__)


class PerformanceMixin:
    """Méthodes de calcul du score de performance."""

    def _compute_and_update_performance_score(
        self: _SyncProtocol, match_id: str, match_row: MatchStatsRow
    ) -> None:
        """Calcule et met à jour le score de performance pour un match.

        Architecture v5.1 :
            Lit team_mmr et enemy_mmr directement depuis mp.enemy_mmr dans
            shared.match_participants (corrigé : remplace l'ancienne sous-requête
            corrélée qui calculait la moyenne de l'équipe adverse).
            Écrit dans player_match_enrichment (player DB).

        Args:
            match_id: ID du match
            match_row: Données du match inséré
        """
        try:
            conn = self._get_connection()

            # Vérifier si le score existe déjà dans player_match_enrichment
            existing = conn.execute(
                "SELECT performance_score FROM player_match_enrichment WHERE match_id = ?",
                (match_id,),
            ).fetchone()

            if existing and existing[0] is not None:
                # Score déjà calculé, skip
                logger.debug("Score de performance déjà présent pour %s", match_id)
                return

            # Charger l'historique (tous les matchs AVANT celui-ci, triés par date)
            current_start_time = match_row.start_time
            if current_start_time is None:
                logger.debug("Pas de start_time pour %s, skip calcul score", match_id)
                return

            # Convertir datetime en format compatible avec DuckDB
            if isinstance(current_start_time, datetime):
                current_start_time_str = current_start_time.isoformat()
            else:
                current_start_time_str = str(current_start_time)

            # V5 finale : lire depuis shared.match_participants + match_registry
            shared_conn = self._get_shared_connection()
            if shared_conn is None:
                logger.debug("shared_connection indisponible pour calcul score")
                return

            # v5 : lire depuis shared avec xuid du joueur
            history_df = shared_conn.execute(
                """
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
                ORDER BY mr.start_time ASC
                """,
                (self._xuid, match_id, current_start_time_str),
            ).pl()

            if history_df.is_empty() or len(history_df) < MIN_MATCHES_FOR_RELATIVE:
                logger.debug(
                    "Pas assez d'historique pour calculer le score (%s matchs)",
                    len(history_df),
                )
                return

            # Convertir match_row en dict pour le calcul v5
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

            # Calculer le score
            score = compute_relative_performance_score(match_dict, history_df)

            if score is not None:
                # V5 finale : écrire dans player_match_enrichment au lieu de match_stats
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

    def batch_compute_performance_scores(self: _SyncProtocol) -> int:  # noqa: C901, PLR0912
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
            # V5 finale : lire depuis shared.match_participants + match_registry
            shared_conn = self._get_shared_connection()
            if shared_conn is None or not self._xuid:
                logger.warning("shared_connection ou xuid manquant pour batch performance scores")
                return 0

            conn = self._get_connection()

            # 1. Charger TOUS les matchs triés par date depuis shared (SANS le JOIN cross-DB)
            all_matches_df = shared_conn.execute(
                """
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
                  AND mr.start_time IS NOT NULL
                ORDER BY mr.start_time ASC
                """,
                [self._xuid],
            ).pl()

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

            # 3. Identifier les matchs sans score
            match_ids = all_matches_df["match_id"].to_list()
            null_mask = [mid not in existing_match_ids for mid in match_ids]

            if not any(null_mask):
                logger.info("Tous les matchs ont déjà un performance_score")
                return 0

            # 4. Calculer le score pour chaque match NULL
            #    en utilisant l'historique des matchs précédents
            updates: list[tuple[str, float, str]] = []
            now = datetime.now(timezone.utc)

            for i in range(len(all_matches_df)):
                if not null_mask[i]:
                    continue

                # Pas assez d'historique ?
                if i < MIN_MATCHES_FOR_RELATIVE:
                    continue

                # Historique = tous les matchs AVANT l'index i
                history_df = all_matches_df.slice(0, i)

                # Match courant en dict
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

                score = compute_relative_performance_score(match_dict, history_df)
                if score is not None:
                    # V5 finale : UPDATE player_match_enrichment
                    updates.append((match_ids[i], score, str(now)))

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
