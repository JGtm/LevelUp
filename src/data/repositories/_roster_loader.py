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

import polars as pl

from src.data.repositories._gamertag_resolver import (
    GamertagResolverMixin,
    _clean_gamertag_static,
)

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)


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

                        # Enrichir les gamertags depuis v_gamertag_lookup (v6 — remplace
                        # highlight_events.gamertag supprimée par migration drop_highlight_events_gamertag)
                        if self._has_shared_view("v_gamertag_lookup"):
                            try:
                                missing_xuids = [
                                    xu for xu in team_by_xuid if xu not in gamertag_by_xuid
                                ]
                                if missing_xuids:
                                    placeholders = ", ".join(["?" for _ in missing_xuids])
                                    vg_result = conn.execute(
                                        f"SELECT xuid, gamertag FROM shared.v_gamertag_lookup"
                                        f" WHERE xuid IN ({placeholders}) AND gamertag IS NOT NULL",
                                        missing_xuids,
                                    ).fetchall()
                                    for xuid, vg_gt in vg_result:
                                        xu = str(xuid).strip()
                                        gt = _clean_gamertag(vg_gt)
                                        if gt and (
                                            xu not in gamertag_by_xuid
                                            or len(gt) > len(gamertag_by_xuid.get(xu) or "")
                                        ):
                                            gamertag_by_xuid[xu] = gt
                            except Exception:
                                pass

                        mp_success = len(team_by_xuid) >= 1
                        logger.debug(
                            "MÉTHODE 0: %d participants chargés depuis match_participants",
                            len(team_by_xuid),
                        )

                except Exception as e:
                    logger.debug("Erreur lecture match_participants: %s", e)

            # ======================================================================
            # V5.1 : match_participants.team_id est la source canonique.
            # Pas de fallback killer_victim_pairs/highlight_events (obsolète).
            # ======================================================================
            if not mp_success:
                # En v5.1, match_participants devrait toujours être disponible
                logger.warning("match_participants manquant pour %s", match_id)
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
                    COALESCE(pk.perfect_kills, 0) AS perfect_kills,
                    wk.weapon_id AS top_weapon_id
                FROM shared.match_participants p
                LEFT JOIN shared.xuid_aliases a ON a.xuid = p.xuid
                LEFT JOIN (
                    SELECT xuid, SUM(count) AS perfect_kills
                    FROM shared.medals_earned
                    WHERE match_id = ? AND medal_name_id = 1512363953
                    GROUP BY xuid
                ) pk ON pk.xuid = p.xuid
                LEFT JOIN (
                    SELECT xuid, weapon_id
                    FROM (
                        SELECT xuid, weapon_id,
                               ROW_NUMBER() OVER (
                                   PARTITION BY xuid ORDER BY COUNT(*) DESC
                               ) AS rn
                        FROM shared.weapon_kills
                        WHERE match_id = ? AND weapon_id NOT IN (0, 1, 2)
                        GROUP BY xuid, weapon_id
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

    def load_friend_match_details(
        self,
        friend_xuid: str,
        match_ids: list[str],
    ) -> pl.DataFrame:
        """Charge les détails de matchs partagés entre ce joueur et un ami.

        Args:
            friend_xuid: XUID de l'ami.
            match_ids: Liste des IDs de matchs communs.

        Returns:
            DataFrame Polars avec colonnes match_id, start_time, playlist_name,
            pair_name, my_team_id, my_outcome, friend_team_id, friend_outcome, same_team.
            Retourne un DataFrame vide avec le schéma correct si aucun résultat.
        """
        _empty_schema: dict[str, pl.PolarsDataType] = {
            "match_id": pl.Utf8,
            "start_time": pl.Datetime,
            "playlist_name": pl.Utf8,
            "pair_name": pl.Utf8,
            "my_team_id": pl.Int64,
            "my_outcome": pl.Utf8,
            "friend_team_id": pl.Int64,
            "friend_outcome": pl.Utf8,
            "same_team": pl.Boolean,
        }
        if not match_ids:
            return pl.DataFrame(schema=_empty_schema)

        conn = self._get_connection()
        placeholders = ", ".join(["?"] * len(match_ids))
        try:
            result = conn.execute(
                f"""
                SELECT
                    mr.match_id,
                    mr.start_time,
                    mr.playlist_name,
                    mr.pair_name,
                    me.team_id  AS my_team_id,
                    CAST(me.outcome AS VARCHAR) AS my_outcome,
                    fr.team_id  AS friend_team_id,
                    CAST(fr.outcome AS VARCHAR) AS friend_outcome,
                    (me.team_id = fr.team_id) AS same_team
                FROM shared.match_registry mr
                LEFT JOIN shared.match_participants me
                    ON mr.match_id = me.match_id AND me.xuid = ?
                LEFT JOIN shared.match_participants fr
                    ON mr.match_id = fr.match_id AND fr.xuid = ?
                WHERE mr.match_id IN ({placeholders})
                ORDER BY mr.start_time ASC
                """,  # noqa: S608
                [str(self._xuid), friend_xuid, *match_ids],
            )
            dfr = result.pl()
            return dfr if not dfr.is_empty() else pl.DataFrame(schema=_empty_schema)
        except Exception:
            logger.debug("load_friend_match_details: erreur friend=%s", friend_xuid, exc_info=True)
            return pl.DataFrame(schema=_empty_schema)

    def load_common_matches_df(self, target_xuid: str) -> pl.DataFrame:
        """Charge les matchs communs entre le joueur courant et target_xuid.

        Args:
            target_xuid: XUID du joueur recherché.

        Returns:
            DataFrame Polars avec match_id, start_time, player_team_id,
            target_team_id, map_name, playlist_name, pair_name, outcome,
            kills, deaths, assists, kda. Vide si shared indisponible.
        """
        if not self._has_shared_table("match_participants"):
            return pl.DataFrame()
        conn = self._get_connection()
        try:
            result = conn.execute(
                """
                SELECT
                    p.match_id,
                    r.start_time,
                    p.team_id  AS player_team_id,
                    t.team_id  AS target_team_id,
                    r.map_name,
                    r.playlist_name,
                    r.pair_name,
                    p.outcome,
                    p.kills,
                    p.deaths,
                    p.assists,
                    p.kda
                FROM shared.match_participants p
                INNER JOIN shared.match_participants t
                    ON t.match_id = p.match_id AND t.xuid = ?
                INNER JOIN shared.match_registry r
                    ON r.match_id = p.match_id
                WHERE p.xuid = ?
                ORDER BY r.start_time DESC
                """,
                [target_xuid, str(self._xuid)],
            )
            return result.pl()
        except Exception:
            logger.debug("load_common_matches_df: erreur target=%s", target_xuid, exc_info=True)
            return pl.DataFrame()
