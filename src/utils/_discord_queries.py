"""Requêtes DuckDB pour les notifications Discord.

Fonctions de récupération des données de matchs et de complétude
depuis shared_matches.duckdb.
"""

from __future__ import annotations

import json
import logging
from datetime import datetime
from pathlib import Path

logger = logging.getLogger(__name__)

_PROJECT_ROOT = Path(__file__).resolve().parents[2]

from src.utils.paths import get_shared_matches_path  # noqa: E402

_SHARED_DB = get_shared_matches_path()


def _load_friends_defaults() -> dict[str, list[str]]:
    """Charge .streamlit/friends_defaults.json → {xuid: [gamertag, ...]}."""
    p = _PROJECT_ROOT / ".streamlit" / "friends_defaults.json"
    if not p.exists():
        return {}
    try:
        with open(p, encoding="utf-8") as fh:
            data = json.load(fh)
        if isinstance(data, dict):
            return {str(k): [str(g).lstrip("~") for g in v if g] for k, v in data.items()}
    except Exception:
        pass
    return {}


def _fetch_squad_info(
    conn: object,
    match_id: str,
    xuid: str,
) -> tuple[int, list[str]]:
    """Récupère le nombre de participants et les amis présents dans un match.

    Args:
        conn: Connexion DuckDB (read-only).
        match_id: Identifiant du match.
        xuid: XUID du joueur courant.

    Returns:
        Tuple (participants_count, squad_friends).
    """
    participants_count = 0
    squad_friends: list[str] = []
    try:
        cnt = conn.execute(
            "SELECT COUNT(DISTINCT xuid) FROM match_participants WHERE match_id = ?",
            [match_id],
        ).fetchone()
        participants_count = int(cnt[0]) if cnt else 0

        friends_map = _load_friends_defaults()
        my_friends_gt = {g.casefold() for g in friends_map.get(str(xuid), [])}
        if my_friends_gt:
            rows = conn.execute(
                """
                SELECT DISTINCT vg.gamertag
                FROM match_participants mp
                JOIN v_gamertag_lookup vg ON mp.xuid = vg.xuid
                WHERE mp.match_id = ?
                  AND mp.xuid != ?
                  AND mp.team_id = (
                      SELECT team_id FROM match_participants
                      WHERE match_id = ? AND xuid = ?
                      LIMIT 1
                  )
                """,
                [match_id, xuid, match_id, xuid],
            ).fetchall()
            for (gt,) in rows:
                if gt and gt.casefold() in my_friends_gt:
                    squad_friends.append(str(gt))
    except Exception:
        pass
    return participants_count, squad_friends


def fetch_last_match_info(xuid: str) -> object | None:
    """Récupère les informations du dernier match depuis shared_matches.duckdb.

    Args:
        xuid: Xbox User ID du joueur.

    Returns:
        LastMatchInfo rempli (avec squad_friends et participants_count), ou None.
    """
    if not xuid or not _SHARED_DB.exists():
        return None
    try:
        from src.utils.db import duckdb_read_only
        from src.utils.discord_notifier import LastMatchInfo

        with duckdb_read_only(_SHARED_DB) as conn:
            row = conn.execute(
                """
                SELECT
                    mr.match_id,
                    COALESCE(mr.map_name,          mr.map_id,       '—') AS map_name,
                    COALESCE(mr.playlist_name,     '—')                  AS playlist_name,
                    COALESCE(mr.game_variant_name, '—')                  AS game_variant_name,
                    COALESCE(mr.pair_name,         '—')                  AS pair_name,
                    COALESCE(mr.is_ranked,         FALSE)                AS is_ranked,
                    mr.start_time,
                    COALESCE(mp.kills,   0) AS kills,
                    COALESCE(mp.deaths,  0) AS deaths,
                    COALESCE(mp.assists, 0) AS assists,
                    COALESCE(mp.outcome, 0) AS outcome,
                    COALESCE(mp.score,   0) AS score
                FROM match_registry mr
                JOIN match_participants mp ON mr.match_id = mp.match_id
                WHERE mp.xuid = ?
                ORDER BY mr.start_time DESC
                LIMIT 1
                """,
                [xuid],
            ).fetchone()
            if not row:
                return None

            match_id = str(row[0] or "")
            participants_count, squad_friends = (
                _fetch_squad_info(conn, match_id, xuid) if match_id else (0, [])
            )

            return LastMatchInfo(
                match_id=match_id,
                map_name=str(row[1] or "—"),
                playlist_name=str(row[2] or "—"),
                game_variant_name=str(row[3] or "—"),
                pair_name=str(row[4] or "—"),
                is_ranked=bool(row[5]),
                start_time=row[6],
                kills=int(row[7] or 0),
                deaths=int(row[8] or 0),
                assists=int(row[9] or 0),
                outcome=int(row[10] or 0),
                score=int(row[11] or 0),
                participants_count=participants_count,
                squad_friends=squad_friends,
            )
    except Exception as exc:
        logger.debug("[Discord] fetch_last_match_info(%s): %s", xuid, exc)
        return None


def count_new_matches(xuid: str, gamertag: str, since: datetime) -> int:
    """Compte les matchs ajoutés dans shared_matches depuis ``since``.

    Utilise first_sync_at pour identifier les matchs synchronisés lors de cette
    session de sync (uniquement pertinent pour sync_delta / sync_full).

    Args:
        xuid:     XUID du joueur.
        gamertag: Gamertag (utilisé pour les logs uniquement).
        since:    Timestamp de début de la sync.

    Returns:
        Nombre de nouveaux matchs, 0 en cas d'erreur.
    """
    if not xuid or not _SHARED_DB.exists():
        return 0
    try:
        from src.utils.db import duckdb_read_only

        since_naive = since.replace(tzinfo=None) if since.tzinfo is not None else since

        with duckdb_read_only(_SHARED_DB) as conn:
            result = conn.execute(
                """
                SELECT COUNT(*)
                FROM match_registry mr
                JOIN match_participants mp ON mr.match_id = mp.match_id
                WHERE mp.xuid = ?
                  AND mr.first_sync_at >= ?
                """,
                [xuid, since_naive],
            ).fetchone()
            return int(result[0]) if result else 0
    except Exception as exc:
        logger.debug("[Discord] count_new_matches(%s): %s", gamertag, exc)
        return 0


def count_matches_missing_data(xuid: str) -> int:
    """Compte les matchs avec données incomplètes pour ce joueur.

    Un match est considéré incomplet si ses médailles ET ses événements highlight
    n'ont été chargés ni par le sync ni par le backfill.

    Args:
        xuid: XUID du joueur.

    Returns:
        Nombre de matchs incomplets, 0 en cas d'erreur.
    """
    if not xuid or not _SHARED_DB.exists():
        return 0
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(_SHARED_DB) as conn:
            result = conn.execute(
                """
                SELECT COUNT(*)
                FROM match_registry mr
                JOIN match_participants mp ON mr.match_id = mp.match_id
                WHERE mp.xuid = ?
                  AND (
                      -- Double-guard : boolean ET bitmask doivent tous les deux indiquer
                      -- "absent" pour qu'un match soit compté comme incomplet.
                      -- Raison : données historiques où medals_loaded/events_loaded=FALSE
                      -- mais backfill_completed a été posé par un sync antérieur.
                      -- Source de vérité préférée : boolean (plus fiable depuis v5.4).
                      (COALESCE(mr.medals_loaded, FALSE) = FALSE
                       AND (COALESCE(mr.backfill_completed, 0) & 1) = 0)
                      OR
                      (COALESCE(mr.events_loaded, FALSE) = FALSE
                       AND (COALESCE(mr.backfill_completed, 0) & 2) = 0)
                  )
                """,
                [xuid],
            ).fetchone()
            return int(result[0]) if result else 0
    except Exception as exc:
        logger.debug("[Discord] count_matches_missing_data(%s): %s", xuid, exc)
        return 0
