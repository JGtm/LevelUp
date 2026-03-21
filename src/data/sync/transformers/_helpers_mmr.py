"""Extraction du MMR d'équipe et ennemi depuis les données skill Halo Infinite.

Ce module isole la logique d'extraction du MMR depuis la réponse JSON
de l'API skill (PlayerMatchStats), évitant de surcharger _helpers.py.
"""

from __future__ import annotations

from typing import Any

from src.data.sync.transformers._helpers_conversions import XUID_RE, _safe_float, _safe_int


def _extract_mmr_from_skill(
    skill_json: dict[str, Any],
    xuid: str,
    team_id: int | None,
) -> tuple[float | None, float | None] | None:
    """Extrait team_mmr et enemy_mmr depuis le JSON skill.

    Args:
        skill_json: JSON de l'API skill (PlayerMatchStats).
        xuid: XUID du joueur.
        team_id: ID de l'équipe du joueur (utilisé comme fallback si non trouvé dans Result).

    Returns:
        Tuple (team_mmr, enemy_mmr) ou None si le joueur n'est pas trouvé.
    """
    value = skill_json.get("Value")
    if not isinstance(value, list):
        return None

    my_result, my_team_id = _find_player_result(value, xuid)

    if not my_result:
        return None

    if my_team_id is None:
        my_team_id = team_id

    team_mmr = _safe_float(my_result.get("TeamMmr"))

    enemy_mmr = _extract_enemy_mmr_from_team_mmrs(my_result, my_team_id)

    if enemy_mmr is None and my_team_id is not None:
        enemy_mmr = _extract_enemy_mmr_from_teammates(value, my_team_id)

    return (team_mmr, enemy_mmr)


def _find_player_result(
    value: list[Any],
    xuid: str,
) -> tuple[dict | None, int | None]:
    """Trouve le Result et team_id du joueur dans la liste skill."""
    for player in value:
        if not isinstance(player, dict):
            continue
        player_id = player.get("Id")
        if isinstance(player_id, str):
            m = XUID_RE.search(player_id)
            if m and m.group(1) == xuid:
                result = player.get("Result")
                if isinstance(result, dict):
                    return result, _safe_int(result.get("TeamId"))
    return None, None


def _extract_enemy_mmr_from_team_mmrs(
    my_result: dict,
    my_team_id: int | None,
) -> float | None:
    """Extrait l'enemy_mmr depuis TeamMmrs du joueur."""
    team_mmrs_raw = my_result.get("TeamMmrs")
    if isinstance(team_mmrs_raw, dict) and my_team_id is not None:
        my_key = str(my_team_id)
        for k, v in team_mmrs_raw.items():
            if k != my_key:
                return _safe_float(v)
    return None


def _extract_enemy_mmr_from_teammates(
    value: list[Any],
    my_team_id: int,
) -> float | None:
    """Fallback : moyenne des TeamMmr des adversaires."""
    enemy_team_mmrs = []
    for player in value:
        if not isinstance(player, dict):
            continue
        result = player.get("Result")
        if not isinstance(result, dict):
            continue
        p_team = result.get("TeamId")
        p_mmr = _safe_float(result.get("TeamMmr"))
        if p_team is not None and p_team != my_team_id and p_mmr is not None:
            enemy_team_mmrs.append(p_mmr)
    return (sum(enemy_team_mmrs) / len(enemy_team_mmrs)) if enemy_team_mmrs else None
