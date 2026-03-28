"""Mixin pour les données de rencontres carrière et top matchs.

Regroupe :
- Rencontres joueurs (top encountered, antagonistes)
- Top matchs meilleurs/pires (via shared.mv_player_matches)
"""

from __future__ import annotations

import logging
from datetime import datetime

from src.analysis._medal_verdicts import DominanceFlag
from src.data.domain.refdata import Outcome
from src.data.repositories._roster_loader import _SQL_NOT_GHOST

logger = logging.getLogger(__name__)

# Filtre ghost pré-résolu pour alias 'p' (réutilisé dans _TOP_ENCOUNTERED_SQL).
_GHOST_FILTER_P = _SQL_NOT_GHOST.format(p="p")

# Durée minimale d'un match "valide" (en secondes).
MIN_MATCH_DURATION_SECONDS: int = 180

_TOP_ENCOUNTERED_SQL = (
    """
WITH my_matches AS (
    SELECT mp.match_id, mp.team_id, mp.outcome
    FROM shared.match_participants mp
    JOIN shared.match_registry r2 ON r2.match_id = mp.match_id
    WHERE mp.xuid = ?
      AND (? IS NULL OR r2.start_time >= ?)
),
encounters AS (
    SELECT
        p.xuid,
        MAX(COALESCE(vg.gamertag, p.gamertag)) AS gamertag,
        COUNT(*)                                AS total_encounters,
        SUM(CASE WHEN p.team_id = m.team_id   THEN 1 ELSE 0 END) AS ally_count,
        SUM(CASE WHEN p.team_id != m.team_id  THEN 1 ELSE 0 END) AS enemy_count,
        AVG(CASE
            WHEN p.team_id = m.team_id AND m.outcome = 2         THEN 1.0
            WHEN p.team_id = m.team_id AND m.outcome IN (3, 4)   THEN 0.0
            ELSE NULL
        END) AS winrate_as_ally,
        AVG(CASE
            WHEN p.team_id != m.team_id AND m.outcome = 2        THEN 1.0
            WHEN p.team_id != m.team_id AND m.outcome IN (3, 4)  THEN 0.0
            ELSE NULL
        END) AS winrate_vs_enemy,
        MAX(r.start_time) AS last_seen
    FROM shared.match_participants p
    INNER JOIN my_matches m  ON m.match_id = p.match_id
    LEFT JOIN  shared.v_gamertag_lookup vg ON vg.xuid = p.xuid
    LEFT JOIN  shared.match_registry r ON r.match_id = p.match_id
    WHERE p.xuid != ?
      AND """
    + _GHOST_FILTER_P
    + """
      AND NOT LOWER(CAST(p.xuid AS VARCHAR)) LIKE 'bid(%'
    GROUP BY p.xuid
),
kvp_agg AS (
    SELECT
        CASE WHEN k.killer_xuid = ? THEN k.victim_xuid
             ELSE k.killer_xuid END AS opp,
        SUM(CASE WHEN k.killer_xuid = ? THEN k.kill_count ELSE 0 END) AS kills_dealt,
        SUM(CASE WHEN k.victim_xuid = ? THEN k.kill_count ELSE 0 END) AS deaths_suffered
    FROM {kv_table} k
    WHERE k.killer_xuid = ? OR k.victim_xuid = ?
    GROUP BY 1
)
SELECT
    e.xuid, e.gamertag, e.total_encounters, e.ally_count, e.enemy_count,
    e.winrate_as_ally, e.winrate_vs_enemy,
    COALESCE(kvp.kills_dealt, 0)     AS kills_dealt,
    COALESCE(kvp.deaths_suffered, 0) AS deaths_suffered,
    e.last_seen
FROM encounters e
LEFT JOIN kvp_agg kvp ON kvp.opp = e.xuid
ORDER BY e.total_encounters DESC
LIMIT ?
"""
)  # noqa: S608

_ANTAGONISTS_SQL = """
WITH period_matches AS (
    SELECT match_id FROM shared.match_registry
    WHERE (? IS NULL OR start_time >= ?)
),
kills_dealt AS (
    SELECT victim_xuid AS opponent_xuid, SUM(kill_count) AS times_killed
    FROM {kv_table}
    WHERE killer_xuid = ? AND match_id IN (SELECT match_id FROM period_matches)
    GROUP BY victim_xuid
),
kills_suffered AS (
    SELECT killer_xuid AS opponent_xuid, SUM(kill_count) AS times_killed_by
    FROM {kv_table}
    WHERE victim_xuid = ? AND match_id IN (SELECT match_id FROM period_matches)
    GROUP BY killer_xuid
),
matches_vs AS (
    SELECT DISTINCT kvp.match_id,
           CASE WHEN kvp.killer_xuid = ? THEN kvp.victim_xuid
                ELSE kvp.killer_xuid END AS opponent_xuid
    FROM {kv_table} kvp
    WHERE (kvp.killer_xuid = ? OR kvp.victim_xuid = ?)
      AND kvp.match_id IN (SELECT match_id FROM period_matches)
),
match_counts AS (
    SELECT opponent_xuid, COUNT(*) AS matches_against
    FROM matches_vs GROUP BY opponent_xuid
),
combined AS (
    SELECT COALESCE(kd.opponent_xuid, ks.opponent_xuid) AS opponent_xuid,
           COALESCE(kd.times_killed, 0) AS times_killed,
           COALESCE(ks.times_killed_by, 0) AS times_killed_by
    FROM kills_dealt kd
    FULL OUTER JOIN kills_suffered ks ON kd.opponent_xuid = ks.opponent_xuid
)
SELECT c.opponent_xuid, COALESCE(vg.gamertag, '') AS opponent_gamertag,
       c.times_killed, c.times_killed_by,
       COALESCE(mc.matches_against, 0) AS matches_against,
       c.times_killed - c.times_killed_by AS net_kills
FROM combined c
LEFT JOIN shared.v_gamertag_lookup vg ON vg.xuid = c.opponent_xuid
LEFT JOIN match_counts mc ON mc.opponent_xuid = c.opponent_xuid
WHERE c.opponent_xuid != ?
ORDER BY {order_col} DESC
LIMIT ?
"""  # noqa: S608

# Priorité badge par onglet : les matchs badgés remontent en tête.
# best=True  → Contre-Remontada(5) > Remontada(3) > Domination(1)
# best=False → Débandade(4) > Humiliation(2)
_BADGE_PRIORITY_EXPR: dict[bool, str] = {
    True: (
        f"CASE dominance_flag"
        f" WHEN {int(DominanceFlag.CONTRE_REMONTADA)} THEN 3"
        f" WHEN {int(DominanceFlag.REMONTADA)} THEN 2"
        f" WHEN {int(DominanceFlag.DOMINATION)} THEN 1"
        f" ELSE 0 END"
    ),
    False: (
        f"CASE dominance_flag"
        f" WHEN {int(DominanceFlag.DEBANDADE)} THEN 2"
        f" WHEN {int(DominanceFlag.HUMILIATION)} THEN 1"
        f" ELSE 0 END"
    ),
}

# Filtre BTB optionnel (injecté dans {btb_filter}).
# True  → exclut les matchs dont mode_category = 'BTB'
# False → pas de filtre supplémentaire
_BTB_FILTER_SQL: dict[bool, str] = {
    True: "AND mv.match_id NOT IN ("
    "SELECT match_id FROM shared.match_registry WHERE mode_category = 'BTB')",
    False: "",
}

_TOP_MATCHES_SQL = """
WITH enriched AS (
    SELECT
        mv.match_id,
        mv.start_time,
        mv.map_name,
        mv.playlist_name,
        mv.game_variant_name,
        mv.outcome,
        mv.kills,
        mv.deaths,
        mv.assists,
        mv.kda,
        mv.time_played_seconds,
        mv.my_team_score,
        mv.enemy_team_score,
        mv.my_team_ps_score,
        mv.enemy_team_ps_score,
        COALESCE(pme.dominance_flag, 0) AS dominance_flag,
        COALESCE(pme.had_bot_teammate, FALSE) AS had_bot_teammate
    FROM shared.mv_player_matches mv
    LEFT JOIN player_match_enrichment pme
        ON pme.match_id = mv.match_id
    WHERE mv.xuid = ?
      AND mv.outcome IN (?, ?)
      AND COALESCE(mv.time_played_seconds, 0) >= ?
      AND COALESCE(pme.had_bot_teammate, FALSE) = FALSE
      AND COALESCE(mv.is_firefight, FALSE) = FALSE
      AND mv.my_team_score IS NOT NULL
      AND mv.enemy_team_score IS NOT NULL
      {btb_filter}
)
SELECT * FROM enriched
WHERE outcome = ?
ORDER BY
    {badge_priority} DESC,
    time_played_seconds ASC,
    ABS(COALESCE(my_team_score, 0) - COALESCE(enemy_team_score, 0))
      / NULLIF(GREATEST(COALESCE(my_team_score, 0), COALESCE(enemy_team_score, 0)), 0) DESC
LIMIT 10
"""  # noqa: S608


class EncounterCareerMixin:
    """Mixin fournissant les données de rencontres et top matchs pour DuckDBRepository."""

    def load_top_encountered(
        self,
        limit: int = 10,
        exclude_xuids: set[str] | None = None,
        *,
        since: datetime | None = None,
    ) -> list[dict]:
        """Charge les joueurs les plus croisés depuis shared_matches.

        Args:
            limit: Nombre de joueurs à retourner.
            exclude_xuids: Ensemble de xuids à exclure des résultats.
            since: Date de début de période (None = tout l'historique).

        Returns:
            Liste de dicts avec xuid, gamertag, total_encounters, etc.
        """
        conn = self._get_connection()
        if not self.has_shared:
            return []
        kv_table = "shared.v_killer_victim_full"
        xuid = str(self._xuid)
        extra = len(exclude_xuids) if exclude_xuids else 0
        sql_limit = limit + extra
        try:
            rows = conn.execute(
                _TOP_ENCOUNTERED_SQL.format(kv_table=kv_table),
                [xuid, since, since, xuid, xuid, xuid, xuid, xuid, xuid, sql_limit],
            ).fetchall()
        except Exception:
            logger.warning("load_top_encountered: erreur", exc_info=True)
            return []
        cols = (
            "xuid",
            "gamertag",
            "total_encounters",
            "ally_count",
            "enemy_count",
            "winrate_as_ally",
            "winrate_vs_enemy",
            "kills_dealt",
            "deaths_suffered",
            "last_seen",
        )
        results = [dict(zip(cols, r, strict=False)) for r in rows]
        if exclude_xuids:
            lower_excl = {e.casefold() for e in exclude_xuids}
            results = [
                r
                for r in results
                if r["xuid"] not in exclude_xuids
                and (r.get("gamertag") or "").casefold() not in lower_excl
            ]
        return results[:limit]

    def load_antagonists(
        self,
        *,
        mode: str,
        limit: int = 10,
        exclude_xuids: set[str] | None = None,
        since: datetime | None = None,
    ) -> list[dict]:
        """Charge les némésis ou souffre-douleurs depuis shared killer_victim_pairs.

        Args:
            mode: "nemesis" (ceux qui tuent le plus) ou "victim" (ceux tués le plus).
            limit: Nombre de résultats.
            exclude_xuids: Ensemble de xuids à exclure.
            since: Date de début de période.

        Returns:
            Liste de dicts avec opponent_xuid, opponent_gamertag, times_killed, etc.
        """
        conn = self._get_connection()
        kv_table = "shared.v_killer_victim_full"
        xuid = str(self._xuid)
        order_col = "times_killed_by" if mode == "nemesis" else "times_killed"
        extra = len(exclude_xuids) if exclude_xuids else 0
        try:
            rows = conn.execute(
                _ANTAGONISTS_SQL.format(order_col=order_col, kv_table=kv_table),
                [since, since, xuid, xuid, xuid, xuid, xuid, xuid, limit + extra],
            ).fetchall()
        except Exception:
            logger.warning("load_antagonists: erreur (mode=%s)", mode, exc_info=True)
            return []
        cols = (
            "opponent_xuid",
            "opponent_gamertag",
            "times_killed",
            "times_killed_by",
            "matches_against",
            "net_kills",
        )
        results = [dict(zip(cols, r, strict=False)) for r in rows]
        if exclude_xuids:
            lower_excl = {e.casefold() for e in exclude_xuids}
            results = [
                r
                for r in results
                if r["opponent_xuid"] not in exclude_xuids
                and (r.get("opponent_gamertag") or "").casefold() not in lower_excl
            ]
        return results[:limit]

    def load_top_match_list(self, *, best: bool, exclude_btb: bool = False) -> list[dict]:
        """Charge le Top 10 meilleurs ou pires matchs.

        Utilise shared.mv_player_matches + player_match_enrichment (locale).

        Args:
            best: True pour les meilleures perf (victoires dominantes),
                  False pour les pires (défaites humiliantes).
            exclude_btb: Si True, exclut les matchs BTB (mode_category='BTB').

        Returns:
            Liste de dicts avec les colonnes du match.
        """
        if not self.has_shared:
            return []
        conn = self._get_connection()
        target_outcome = int(Outcome.WIN) if best else int(Outcome.LOSS)
        sql = _TOP_MATCHES_SQL.format(
            badge_priority=_BADGE_PRIORITY_EXPR[best],
            btb_filter=_BTB_FILTER_SQL[exclude_btb],
        )
        try:
            result = conn.execute(
                sql,
                [
                    str(self._xuid),
                    int(Outcome.WIN),
                    int(Outcome.LOSS),
                    MIN_MATCH_DURATION_SECONDS,
                    target_outcome,
                ],
            )
            columns = [desc[0] for desc in result.description]
            rows = result.fetchall()
            return [dict(zip(columns, row, strict=True)) for row in rows]
        except Exception:
            logger.warning("load_top_match_list: erreur (best=%s)", best, exc_info=True)
            return []
