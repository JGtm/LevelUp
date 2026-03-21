"""
Mixin pour les relations match/joueur : matchs communs, détails matchs partagés.

Méthodes extraites de RosterLoaderMixin (v6) pour garder chaque module sous 500L.
"""

from __future__ import annotations

import logging

import polars as pl

from src.data.repositories._gamertag_resolver import GamertagResolverMixin

logger = logging.getLogger(__name__)


class MatchRelationsMixin(GamertagResolverMixin):
    """Mixin fournissant les requêtes de relations entre joueurs sur les matchs partagés."""

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
            pair_name, map_name, my_team_id, my_outcome, friend_team_id, friend_outcome, same_team.
            Retourne un DataFrame vide avec le schéma correct si aucun résultat.
        """
        _empty_schema: dict[str, pl.PolarsDataType] = {
            "match_id": pl.Utf8,
            "start_time": pl.Datetime,
            "playlist_name": pl.Utf8,
            "pair_name": pl.Utf8,
            "map_name": pl.Utf8,
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
                    mr.map_name,
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
            logger.warning(
                "load_friend_match_details: erreur friend=%s", friend_xuid, exc_info=True
            )
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
        conn = self._get_connection()
        if not self.has_shared:
            return pl.DataFrame()
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
            logger.warning("load_common_matches_df: erreur target=%s", target_xuid, exc_info=True)
            return pl.DataFrame()
