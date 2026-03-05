"""
Mixin pour le chargement des rosters et la composition d'équipes.

Regroupe les méthodes d'analyse des compositions d'équipes
extraites de DuckDBRepository :
- load_match_rosters
- load_matches_with_teammate
- load_same_team_match_ids
- has_match_participants
- load_match_players_stats
- load_match_scoreboard

La résolution XUID → Gamertag est dans ``_gamertag_resolver.py``.
"""

from __future__ import annotations

import contextlib
import logging
from typing import TYPE_CHECKING, Any

from src.data.repositories._gamertag_resolver import (
    GamertagResolverMixin,
    _clean_gamertag_static,
)

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)


class RosterLoaderMixin(GamertagResolverMixin):
    """Mixin fournissant le chargement des rosters et la résolution des gamertags pour DuckDBRepository."""

    def load_match_rosters(  # noqa: C901, PLR0912, PLR0915
        self,
        match_id: str,
    ) -> dict[str, Any] | None:
        """Charge les rosters d'un match depuis killer_victim_pairs ou highlight_events.

        Utilise killer_victim_pairs si disponible (source fiable), sinon
        analyse les patterns de kills dans highlight_events.

        Returns:
            None si le match n'existe pas ou si les données sont insuffisantes.
            Sinon un dict avec la structure:
            {
                "my_team_id": int,
                "my_team": [{"xuid": str, "gamertag": str|None, "team_id": int|None, "is_me": bool}],
                "enemy_team": [...],
            }
        """
        conn = self._get_connection()

        try:
            # V5.1 : shared.match_participants est la source canonique du team_id
            my_xuid_str = str(self._xuid).strip()
            match_info = None

            if self._has_shared_table("match_participants"):
                with contextlib.suppress(Exception):
                    match_info = conn.execute(
                        "SELECT team_id FROM shared.match_participants WHERE match_id = ? AND xuid = ?",
                        [match_id, my_xuid_str],
                    ).fetchone()

            # NOTE v5.1 : match_stats supprimée, pas de fallback legacy
            if not match_info:
                return None

            my_team_id = match_info[0]
            if my_team_id is None:
                return None

            # Alias local pour la fonction de nettoyage module-level
            _clean_gamertag = _clean_gamertag_static

            # ======================================================================
            # MÉTHODE 0 (PRIORITAIRE) : Utiliser match_participants.team_id
            # C'est la source la plus fiable car elle vient directement de l'API
            # ======================================================================
            team_by_xuid: dict[str, int | None] = {}
            gamertag_by_xuid: dict[str, str | None] = {}
            mp_success = False

            if self._has_shared_table("match_participants"):
                try:
                    # Requête de base : match_participants seul (toujours présent en v5.1)
                    mp_result = conn.execute(
                        """
                        SELECT xuid, team_id, gamertag
                        FROM shared.match_participants
                        WHERE match_id = ?
                        """,
                        [match_id],
                    ).fetchall()

                    if mp_result and len(mp_result) > 0:
                        for xuid, team_id, mp_gt in mp_result:
                            xu = str(xuid).strip()
                            if not xu:
                                continue

                            # Stocker le team_id (source fiable)
                            team_by_xuid[xu] = team_id

                            # Gamertag depuis match_participants
                            gt = _clean_gamertag(mp_gt)
                            if gt:
                                gamertag_by_xuid[xu] = gt

                        # Enrichir les gamertags depuis xuid_aliases (optionnel)
                        if self._has_shared_table("xuid_aliases"):
                            try:
                                alias_result = conn.execute(
                                    """
                                    SELECT xuid, gamertag
                                    FROM shared.xuid_aliases
                                    WHERE xuid IN (SELECT xuid FROM shared.match_participants WHERE match_id = ?)
                                    """,
                                    [match_id],
                                ).fetchall()
                                for xuid, alias_gt in alias_result:
                                    xu = str(xuid).strip()
                                    gt = _clean_gamertag(alias_gt)
                                    if gt and (
                                        xu not in gamertag_by_xuid
                                        or len(gt) > len(gamertag_by_xuid.get(xu) or "")
                                    ):
                                        gamertag_by_xuid[xu] = gt
                            except Exception:
                                pass

                        # Enrichir les gamertags depuis highlight_events (optionnel)
                        if self._has_shared_table("highlight_events"):
                            try:
                                he_result = conn.execute(
                                    """
                                    SELECT xuid, gamertag
                                    FROM (
                                        SELECT xuid, gamertag,
                                               ROW_NUMBER() OVER (
                                                   PARTITION BY xuid
                                                   ORDER BY LENGTH(COALESCE(gamertag, '')) DESC
                                               ) as rn
                                        FROM shared.highlight_events
                                        WHERE match_id = ? AND xuid IS NOT NULL AND xuid != ''
                                    ) sub
                                    WHERE rn = 1
                                    """,
                                    [match_id],
                                ).fetchall()
                                for xuid, he_gt in he_result:
                                    xu = str(xuid).strip()
                                    gt = _clean_gamertag(he_gt)
                                    if gt and (
                                        xu not in gamertag_by_xuid
                                        or len(gt) > len(gamertag_by_xuid.get(xu) or "")
                                    ):
                                        gamertag_by_xuid[xu] = gt
                            except Exception:
                                pass

                        mp_success = len(team_by_xuid) >= 1
                        logger.debug(
                            f"MÉTHODE 0: {len(team_by_xuid)} participants chargés depuis match_participants"
                        )

                except Exception as e:
                    logger.debug(f"Erreur lecture match_participants: {e}")

            # ======================================================================
            # V5.1 : match_participants.team_id est la source canonique.
            # Pas de fallback killer_victim_pairs/highlight_events (obsolète).
            # ======================================================================
            if not mp_success:
                # En v5.1, match_participants devrait toujours être disponible
                logger.warning(f"match_participants manquant pour {match_id}")
                return None

            all_xuids: set[str] = set(team_by_xuid.keys())

            # ======================================================================
            # Construire les listes d'équipes
            # ======================================================================
            # Sprint Gamertag Roster Fix : Utiliser resolve_gamertags_batch pour
            # obtenir des gamertags propres depuis match_participants/xuid_aliases
            resolved_gamertags = self.resolve_gamertags_batch(list(all_xuids), match_id=match_id)

            my_team = []
            enemy_team = []

            for xuid_str in all_xuids:
                is_me = xuid_str == my_xuid_str
                # Priorité : gamertag résolu > gamertag extrait > XUID
                cleaned_gamertag = resolved_gamertags.get(xuid_str) or gamertag_by_xuid.get(
                    xuid_str
                )
                display_name = cleaned_gamertag if cleaned_gamertag else xuid_str
                player_team_id = team_by_xuid.get(xuid_str, None if not is_me else my_team_id)

                player_data = {
                    "xuid": xuid_str,
                    "gamertag": cleaned_gamertag,
                    "team_id": player_team_id,
                    "is_me": is_me,
                    "is_bot": False,
                    "display_name": display_name,
                }

                if player_team_id == my_team_id or is_me:
                    my_team.append(player_data)
                else:
                    enemy_team.append(player_data)

            # Trier: moi en premier, puis alphabétique
            def _sort_key(r: dict[str, Any]) -> tuple[int, str]:
                me_rank = 0 if r.get("is_me") else 1
                name = str(r.get("gamertag") or r.get("xuid") or "").strip().lower()
                return (me_rank, name)

            my_team.sort(key=_sort_key)
            enemy_team.sort(key=_sort_key)

            return {
                "my_team_id": int(my_team_id),
                "my_team_name": None,
                "my_team": my_team,
                "enemy_team": enemy_team,
                "enemy_team_ids": [],
                "enemy_team_names": [],
            }
        except Exception as e:
            logger.warning(f"Erreur lors du chargement des rosters pour {match_id}: {e}")
            return None

    def load_matches_with_teammate(
        self,
        teammate_xuid: str,
    ) -> list[str]:
        """Retourne les match_id joués avec un coéquipier depuis shared.

        Args:
            teammate_xuid: XUID du coéquipier.

        Returns:
            Liste des match_id où les deux joueurs apparaissent.
        """
        conn = self._get_connection()

        try:
            result = conn.execute(
                """
                SELECT DISTINCT me.match_id
                FROM shared.match_participants me
                INNER JOIN shared.match_participants tm
                    ON me.match_id = tm.match_id
                WHERE me.xuid = ? AND tm.xuid = ?
                ORDER BY me.match_id DESC
                """,
                [self._xuid, teammate_xuid],
            )
            return [row[0] for row in result.fetchall()]
        except Exception as e:
            logger.debug(f"Erreur load_matches_with_teammate shared: {e}")
            return []

    def load_same_team_match_ids(
        self,
        teammate_xuid: str,
    ) -> list[str]:
        """Retourne les match_id où les deux joueurs étaient dans la même équipe.

        Utilise shared.match_participants (team_id fiable).

        Args:
            teammate_xuid: XUID du coéquipier.

        Returns:
            Liste des match_id où les deux joueurs étaient dans la même équipe.
        """
        conn = self._get_connection()

        try:
            result = conn.execute(
                """
                SELECT DISTINCT me.match_id
                FROM shared.match_participants me
                INNER JOIN shared.match_participants tm
                    ON me.match_id = tm.match_id
                    AND me.team_id = tm.team_id
                WHERE me.xuid = ? AND tm.xuid = ?
                ORDER BY me.match_id DESC
                """,
                [self._xuid, teammate_xuid],
            )
            return [row[0] for row in result.fetchall()]
        except Exception as e:
            logger.debug(f"Erreur load_same_team_match_ids shared: {e}")
            return []

    def has_match_participants(self) -> bool:
        """Vérifie si des données match_participants existent dans shared."""
        conn = self._get_connection()
        try:
            count = conn.execute(
                "SELECT COUNT(*) FROM shared.match_participants WHERE xuid = ?",
                [self._xuid],
            ).fetchone()[0]
            return count > 0
        except Exception:
            return False

    def load_match_players_stats(self, match_id: str) -> list[dict[str, Any]]:
        """Charge les statistiques officielles de tous les joueurs d'un match.

        Utilise shared.match_participants (roster complet, toutes colonnes).

        Args:
            match_id: ID du match.

        Returns:
            Liste de dicts avec: xuid, gamertag, kills, deaths, assists, team_id, rank, score
        """
        if not match_id:
            return []

        conn = self._get_connection()

        try:
            rows = conn.execute(
                """
                SELECT
                    p.xuid,
                    COALESCE(p.gamertag, a.gamertag, p.xuid) AS gamertag,
                    p.team_id,
                    p.rank,
                    p.score,
                    p.kills,
                    p.deaths,
                    p.assists
                FROM shared.match_participants p
                LEFT JOIN shared.xuid_aliases a ON a.xuid = p.xuid
                WHERE p.match_id = ?
                ORDER BY p.rank ASC NULLS LAST
                """,
                [match_id],
            ).fetchall()

            result = []
            for idx, row in enumerate(rows):
                result.append(
                    {
                        "xuid": str(row[0] or "").strip(),
                        "gamertag": str(row[1] or row[0] or "").strip(),
                        "team_id": int(row[2]) if row[2] is not None else None,
                        "rank": int(row[3]) if row[3] is not None else idx + 1,
                        "score": int(row[4]) if row[4] is not None else None,
                        "kills": int(row[5]) if row[5] is not None else 0,
                        "deaths": int(row[6]) if row[6] is not None else 0,
                        "assists": int(row[7]) if row[7] is not None else 0,
                    }
                )
            return result
        except Exception as e:
            logger.debug(f"Erreur load_match_players_stats shared: {e}")
            return []

    def load_match_scoreboard(self, match_id: str) -> list[dict[str, Any]]:
        """Charge le tableau de bord complet de tous les joueurs d'un match.

        Récupère toutes les colonnes de match_participants + le compte de médailles
        Perfect Kill (ID 1512363953) par joueur.

        Args:
            match_id: ID du match.

        Returns:
            Liste de dicts triée par (team_id, rank) avec les champs :
            xuid, gamertag, team_id, rank, score, kills, deaths, assists,
            kda, max_killing_spree, headshot_kills, shots_fired, shots_hit,
            accuracy, melee_kills, power_weapon_kills, damage_dealt,
            damage_taken, avg_life_seconds, perfect_kills.
        """
        if not match_id:
            return []

        conn = self._get_connection()

        try:
            rows = conn.execute(
                """
                SELECT
                    p.xuid,
                    COALESCE(p.gamertag, a.gamertag, p.xuid) AS gamertag,
                    p.team_id,
                    p.rank,
                    p.score,
                    p.kills,
                    p.deaths,
                    p.assists,
                    p.kda,
                    p.max_killing_spree,
                    p.headshot_kills,
                    p.shots_fired,
                    p.shots_hit,
                    p.accuracy,
                    p.melee_kills,
                    p.power_weapon_kills,
                    p.damage_dealt,
                    p.damage_taken,
                    p.avg_life_seconds,
                    COALESCE(pk.perfect_kills, 0) AS perfect_kills
                FROM shared.match_participants p
                LEFT JOIN shared.xuid_aliases a ON a.xuid = p.xuid
                LEFT JOIN (
                    SELECT xuid, SUM(count) AS perfect_kills
                    FROM shared.medals_earned
                    WHERE match_id = ? AND medal_name_id = 1512363953
                    GROUP BY xuid
                ) pk ON pk.xuid = p.xuid
                WHERE p.match_id = ?
                ORDER BY p.team_id ASC NULLS LAST, p.rank ASC NULLS LAST
                """,
                [match_id, match_id],
            ).fetchall()

            result = []
            for idx, row in enumerate(rows):
                result.append(
                    {
                        "xuid": str(row[0] or "").strip(),
                        "gamertag": str(row[1] or row[0] or "").strip(),
                        "team_id": int(row[2]) if row[2] is not None else None,
                        "rank": int(row[3]) if row[3] is not None else idx + 1,
                        "score": int(row[4]) if row[4] is not None else None,
                        "kills": int(row[5]) if row[5] is not None else None,
                        "deaths": int(row[6]) if row[6] is not None else None,
                        "assists": int(row[7]) if row[7] is not None else None,
                        "kda": float(row[8]) if row[8] is not None else None,
                        "max_killing_spree": int(row[9]) if row[9] is not None else None,
                        "headshot_kills": int(row[10]) if row[10] is not None else None,
                        "shots_fired": int(row[11]) if row[11] is not None else None,
                        "shots_hit": int(row[12]) if row[12] is not None else None,
                        "accuracy": float(row[13]) if row[13] is not None else None,
                        "melee_kills": int(row[14]) if row[14] is not None else None,
                        "power_weapon_kills": int(row[15]) if row[15] is not None else None,
                        "damage_dealt": float(row[16]) if row[16] is not None else None,
                        "damage_taken": float(row[17]) if row[17] is not None else None,
                        "avg_life_seconds": float(row[18]) if row[18] is not None else None,
                        "perfect_kills": int(row[19]) if row[19] is not None else 0,
                    }
                )
            return result
        except Exception as e:
            logger.debug(f"Erreur load_match_scoreboard shared: {e}")
            return []
