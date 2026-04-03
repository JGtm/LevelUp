"""Extraction des médailles (par joueur et pour tous les joueurs d'un match).

Fonctions publiques :
- extract_medals      : JSON match + xuid → [MedalEarnedRow] (un seul joueur)
- extract_all_medals  : JSON match → [SharedMedalEarnedRow] (tous les joueurs)
"""

from __future__ import annotations

import json
from typing import Any

from src.data.sync.models import (
    MedalEarnedRow,
    SharedMedalEarnedRow,
)
from src.data.sync.transformers._helpers import (
    XUID_RE,
    _find_player,
    _safe_int,
)


def extract_medals(  # noqa: C901
    match_json: dict[str, Any],
    xuid: str,
) -> list[MedalEarnedRow]:
    """Extrait les médailles d'un joueur depuis le JSON du match.

    Args:
        match_json: JSON brut du match depuis l'API SPNKr.
        xuid: XUID du joueur.

    Returns:
        Liste de MedalEarnedRow avec les médailles obtenues.
    """
    match_id = match_json.get("MatchId")
    if not isinstance(match_id, str):
        return []

    players = match_json.get("Players")
    if not isinstance(players, list):
        return []

    # Trouver le joueur
    me = _find_player(players, xuid)
    if me is None:
        return []

    # Extraire les médailles depuis PlayerTeamStats[].Stats.CoreStats.Medals[]
    pts = me.get("PlayerTeamStats")
    if not isinstance(pts, list):
        return []

    medals_dict: dict[int, int] = {}  # medal_name_id -> total_count

    for team_stats in pts:
        if not isinstance(team_stats, dict):
            continue

        stats = team_stats.get("Stats")
        if not isinstance(stats, dict):
            continue

        core_stats = stats.get("CoreStats")
        if not isinstance(core_stats, dict):
            continue

        medals = core_stats.get("Medals")
        if not isinstance(medals, list):
            continue

        for medal in medals:
            if not isinstance(medal, dict):
                continue

            name_id = _safe_int(medal.get("NameId"))
            count = _safe_int(medal.get("Count"))

            if name_id is not None and count is not None and count > 0:
                medals_dict[name_id] = medals_dict.get(name_id, 0) + count

    # Convertir en MedalEarnedRow
    return [
        MedalEarnedRow(match_id=match_id, medal_name_id=name_id, count=count)
        for name_id, count in medals_dict.items()
    ]


def extract_all_medals(  # noqa: C901, PLR0912
    match_json: dict[str, Any],
) -> list[SharedMedalEarnedRow]:
    """Extrait les médailles de TOUS les joueurs d'un match.

    Contrairement à extract_medals() qui n'extrait que pour un seul xuid,
    cette fonction parcourt TOUS les Players[] et retourne les médailles
    de chacun avec leur xuid.

    Utilisée par :
    - La migration v5 (shared_matches_v2.duckdb)
    - Le sync engine v5 (insertion dans shared)

    Args:
        match_json: JSON brut du match depuis l'API SPNKr.

    Returns:
        Liste de SharedMedalEarnedRow avec xuid pour chaque joueur.
    """
    match_id = match_json.get("MatchId")
    if not isinstance(match_id, str):
        return []

    players = match_json.get("Players")
    if not isinstance(players, list):
        return []

    all_medals: list[SharedMedalEarnedRow] = []

    for player in players:
        if not isinstance(player, dict):
            continue

        # Extraire le XUID du joueur
        pid = player.get("PlayerId")
        xuid: str | None = None
        if isinstance(pid, str):
            m = XUID_RE.search(pid)
            if m:
                xuid = m.group(1)
        elif isinstance(pid, dict):
            xuid_val = pid.get("Xuid") or pid.get("xuid")
            if xuid_val is not None:
                xuid = str(xuid_val)
            else:
                try:
                    s = json.dumps(pid)
                    m = XUID_RE.search(s)
                    if m:
                        xuid = m.group(1)
                except (TypeError, ValueError):
                    pass

        if not xuid:
            continue

        # Extraire les médailles depuis PlayerTeamStats[].Stats.CoreStats.Medals[]
        pts = player.get("PlayerTeamStats")
        if not isinstance(pts, list):
            continue

        medals_dict: dict[int, int] = {}  # medal_name_id -> total_count

        for team_stats in pts:
            if not isinstance(team_stats, dict):
                continue
            stats = team_stats.get("Stats")
            if not isinstance(stats, dict):
                continue
            core_stats = stats.get("CoreStats")
            if not isinstance(core_stats, dict):
                continue
            medals = core_stats.get("Medals")
            if not isinstance(medals, list):
                continue

            for medal in medals:
                if not isinstance(medal, dict):
                    continue
                name_id = _safe_int(medal.get("NameId"))
                count = _safe_int(medal.get("Count"))
                if name_id is not None and count is not None and count > 0:
                    medals_dict[name_id] = medals_dict.get(name_id, 0) + count

        for name_id, count in medals_dict.items():
            all_medals.append(
                SharedMedalEarnedRow(
                    match_id=match_id,
                    xuid=xuid,
                    medal_name_id=name_id,
                    count=count,
                )
            )

    return all_medals
