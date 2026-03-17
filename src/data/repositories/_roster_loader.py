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
Les requêtes de relations (matchs communs, détails ami) sont dans ``_match_relations.py``.
"""

from __future__ import annotations

import logging
from typing import Any

from src.data.repositories._gamertag_resolver import (
    GamertagResolverMixin,
    _clean_gamertag_static,
)

logger = logging.getLogger(__name__)


def _get_my_team_id(conn, match_id: str, my_xuid_str: str) -> int | None:
    """Retourne le team_id du joueur principal pour un match, ou None."""
    try:
        row = conn.execute(
            "SELECT team_id FROM shared.match_participants WHERE match_id = ? AND xuid = ?",
            [match_id, my_xuid_str],
        ).fetchone()
        return int(row[0]) if row and row[0] is not None else None
    except Exception:
        return None


def _load_participants_data(
    conn, match_id: str
) -> tuple[dict[str, int | None], dict[str, str | None]] | None:
    """Charge les participants d'un match depuis shared.match_participants.

    Returns:
        (team_by_xuid, gamertag_by_xuid) ou None si la table est indisponible.
    """

    try:
        rows = conn.execute(
            "SELECT xuid, team_id, gamertag FROM shared.match_participants WHERE match_id = ?",
            [match_id],
        ).fetchall()
    except Exception as e:
        logger.debug("Erreur lecture match_participants: %s", e)
        return None

    if not rows:
        return None

    team_by_xuid: dict[str, int | None] = {}
    gamertag_by_xuid: dict[str, str | None] = {}
    for xuid, team_id, mp_gt in rows:
        xu = str(xuid).strip()
        if not xu:
            continue
        team_by_xuid[xu] = team_id
        gt = _clean_gamertag_static(mp_gt)
        if gt:
            gamertag_by_xuid[xu] = gt

    return team_by_xuid, gamertag_by_xuid


def _assemble_roster(
    team_by_xuid: dict[str, int | None],
    gamertag_by_xuid: dict[str, str | None],
    resolved_gamertags: dict[str, str],
    my_xuid_str: str,
    my_team_id: int,
) -> tuple[list, list]:
    """Construit les listes my_team / enemy_team triées."""
    my_team: list = []
    enemy_team: list = []

    for xuid_str in team_by_xuid:
        is_me = xuid_str == my_xuid_str
        cleaned_gamertag = resolved_gamertags.get(xuid_str) or gamertag_by_xuid.get(xuid_str)
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

    def _sort_key(r: dict) -> tuple[int, str]:
        return (0 if r.get("is_me") else 1, str(r.get("gamertag") or r.get("xuid") or "").lower())

    my_team.sort(key=_sort_key)
    enemy_team.sort(key=_sort_key)
    return my_team, enemy_team


def _scoreboard_row_to_dict(idx: int, row: tuple) -> dict:
    """Convertit une ligne DuckDB du scoreboard en dict métier."""
    return {
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
        "top_weapon_id": int(row[20]) if row[20] is not None else None,
    }


class RosterLoaderMixin(GamertagResolverMixin):
    """Mixin fournissant le chargement des rosters et la résolution des gamertags pour DuckDBRepository."""

    def load_match_rosters(
        self,
        match_id: str,
    ) -> dict[str, Any] | None:
        """Charge les rosters d'un match depuis shared.match_participants.

        Returns:
            None si le match n'existe pas ou si les données sont insuffisantes.
            Sinon un dict avec : my_team_id, my_team, enemy_team.
        """
        conn = self._get_connection()
        if not self.has_shared:
            return None

        my_xuid_str = str(self._xuid).strip()

        try:
            my_team_id = _get_my_team_id(conn, match_id, my_xuid_str)
            if my_team_id is None:
                return None

            participants = _load_participants_data(conn, match_id)
            if participants is None:
                logger.warning("match_participants manquant pour %s", match_id)
                return None

            team_by_xuid, gamertag_by_xuid = participants
            if len(team_by_xuid) < 1:
                return None

            logger.debug(
                "load_match_rosters: %d participants chargés pour %s",
                len(team_by_xuid),
                match_id,
            )
            # v6 : résolution gamertag via v_gamertag_lookup (xuid_aliases ∪ match_participants)
            resolved_gamertags = self.resolve_gamertags_batch(
                list(team_by_xuid.keys()), match_id=match_id
            )
            my_team, enemy_team = _assemble_roster(
                team_by_xuid, gamertag_by_xuid, resolved_gamertags, my_xuid_str, my_team_id
            )
            return {
                "my_team_id": int(my_team_id),
                "my_team_name": None,
                "my_team": my_team,
                "enemy_team": enemy_team,
                "enemy_team_ids": [],
                "enemy_team_names": [],
            }
        except Exception as e:
            logger.warning("Erreur lors du chargement des rosters pour %s: %s", match_id, e)
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
            logger.debug("Erreur load_matches_with_teammate shared: %s", e)
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
            logger.debug("Erreur load_same_team_match_ids shared: %s", e)
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
                    COALESCE(vg.gamertag, p.gamertag, p.xuid) AS gamertag,
                    p.team_id,
                    p.rank,
                    p.score,
                    p.kills,
                    p.deaths,
                    p.assists
                FROM shared.match_participants p
                LEFT JOIN shared.v_gamertag_lookup vg ON vg.xuid = p.xuid
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
            logger.debug("Erreur load_match_players_stats shared: %s", e)
            return []

    def load_match_scoreboard(self, match_id: str) -> list[dict[str, Any]]:
        """Charge le tableau de bord complet de tous les joueurs d'un match.

        Récupère toutes les colonnes de match_participants + le compte de médailles
        Perfect Kill (ID 1512363953) + l'arme la plus utilisée pour les kills
        (top_weapon_id, depuis weapon_kills, hors sentinelles 0/1/2).

        Args:
            match_id: ID du match.

        Returns:
            Liste de dicts triée par (team_id, rank) avec les champs :
            xuid, gamertag, team_id, rank, score, kills, deaths, assists,
            kda, max_killing_spree, headshot_kills, shots_fired, shots_hit,
            accuracy, melee_kills, power_weapon_kills, damage_dealt,
            damage_taken, avg_life_seconds, perfect_kills, top_weapon_id.
            top_weapon_id vaut None si aucun kill d'arme n'est enregistré.
        """
        if not match_id:
            return []

        conn = self._get_connection()

        try:
            rows = conn.execute(
                """
                SELECT
                    p.xuid,
                    COALESCE(vg.gamertag, p.gamertag, p.xuid) AS gamertag,
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
                    COALESCE(pk.perfect_kills, 0) AS perfect_kills,
                    wk.weapon_id AS top_weapon_id
                FROM shared.match_participants p
                LEFT JOIN shared.v_gamertag_lookup vg ON vg.xuid = p.xuid
                LEFT JOIN (
                    SELECT xuid, SUM(count) AS perfect_kills
                    FROM shared.medals_earned
                    WHERE match_id = ? AND medal_name_id = 1512363953
                    GROUP BY xuid
                ) pk ON pk.xuid = p.xuid
                LEFT JOIN (
                    SELECT xuid, effective_weapon_id AS weapon_id
                    FROM (
                        SELECT xuid, effective_weapon_id,
                               ROW_NUMBER() OVER (
                                   PARTITION BY xuid ORDER BY COUNT(*) DESC
                               ) AS rn
                        FROM shared.v_weapon_kills
                        WHERE match_id = ? AND effective_weapon_id NOT IN (0, 1, 2)
                        GROUP BY xuid, effective_weapon_id
                    )
                    WHERE rn = 1
                ) wk ON wk.xuid = p.xuid
                WHERE p.match_id = ?
                ORDER BY p.team_id ASC NULLS LAST, p.rank ASC NULLS LAST
                """,
                [match_id, match_id, match_id],
            ).fetchall()

            result = []
            for idx, row in enumerate(rows):
                result.append(_scoreboard_row_to_dict(idx, row))
            return result
        except Exception as e:
            logger.debug("Erreur load_match_scoreboard shared: %s", e)
            return []
