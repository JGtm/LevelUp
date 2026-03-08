"""Extraction du MMR d'équipe et ennemi depuis les données skill Halo Infinite.

Ce module isole la logique d'extraction du MMR depuis la réponse JSON
de l'API skill (PlayerMatchStats), évitant de surcharger _helpers.py.
"""

from __future__ import annotations

from typing import Any

from src.data.sync.transformers._helpers_conversions import XUID_RE, _safe_float, _safe_int


def _extract_mmr_from_skill(  # noqa: C901, PLR0912
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
        Tuple (team_mmr, enemy_mmr) où chaque valeur peut être None, ou None si le joueur n'est pas trouvé.
        Retourne None uniquement si le joueur n'est pas trouvé dans le JSON.
        Sinon, retourne toujours un tuple, même si une seule valeur est disponible.
    """
    value = skill_json.get("Value")
    if not isinstance(value, list):
        return None

    # Trouver notre joueur et extraire TeamMmrs
    my_result = None
    my_team_id = None

    for player in value:
        if not isinstance(player, dict):
            continue

        player_id = player.get("Id")
        player_xuid = None
        if isinstance(player_id, str):
            m = XUID_RE.search(player_id)
            if m:
                player_xuid = m.group(1)

        if player_xuid == xuid:
            my_result = player.get("Result")
            if isinstance(my_result, dict):
                my_team_id = _safe_int(my_result.get("TeamId"))
                break

    # Si le joueur n'est pas trouvé, retourner None
    if not my_result:
        return None

    # Utiliser team_id du paramètre si non trouvé dans Result
    if my_team_id is None:
        my_team_id = team_id

    # Extraire team_mmr depuis TeamMmr du joueur
    team_mmr = _safe_float(my_result.get("TeamMmr"))

    # Extraire enemy_mmr depuis TeamMmrs (recommandé)
    # TeamMmrs contient les MMR de toutes les équipes : {"0": 1200.5, "1": 1150.3}
    enemy_mmr = None
    team_mmrs_raw = my_result.get("TeamMmrs")
    if isinstance(team_mmrs_raw, dict) and my_team_id is not None:
        my_key = str(my_team_id)
        for k, v in team_mmrs_raw.items():
            if k != my_key:
                enemy_mmr = _safe_float(v)
                break

    # Fallback : utiliser TeamMmr d'un adversaire si TeamMmrs n'est pas disponible
    if enemy_mmr is None and my_team_id is not None:
        enemy_team_mmrs = []
        for player in value:
            if not isinstance(player, dict):
                continue
            result = player.get("Result")
            if not isinstance(result, dict):
                continue
            player_team = result.get("TeamId")
            player_team_mmr = _safe_float(result.get("TeamMmr"))
            if (
                player_team is not None
                and player_team != my_team_id
                and player_team_mmr is not None
            ):
                enemy_team_mmrs.append(player_team_mmr)

        if enemy_team_mmrs:
            enemy_mmr = sum(enemy_team_mmrs) / len(enemy_team_mmrs)

    # Retourner les valeurs même si une seule est disponible
    return (team_mmr, enemy_mmr)
