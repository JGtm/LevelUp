"""Extraction et transformation des PersonalScores (décomposition du score personnel).

Fonctions publiques :
- extract_personal_score_awards   : JSON match + xuid → [dict]
- categorize_personal_score       : name_id → catégorie str
- transform_personal_score_awards : match_id + xuid + personal_scores → [PersonalScoreAwardRow]

Fonctions privées (internes) :
- _find_personal_scores : trouve la liste PersonalScores dans l'objet joueur
"""

from __future__ import annotations

from typing import Any

from src.data.domain.refdata import PERSONAL_SCORE_POINTS
from src.data.sync.transformers._helpers import (
    _find_player,
    _safe_int,
)


def _find_personal_scores(player_obj: dict[str, Any]) -> list[dict[str, Any]]:
    """Trouve la liste PersonalScores dans l'objet joueur.

    Parcourt récursivement PlayerTeamStats pour trouver PersonalScores[].
    """

    def find_ps(x: Any) -> list[dict[str, Any]] | None:
        if isinstance(x, dict):
            # Vérifier si ce dict contient PersonalScores
            ps = x.get("PersonalScores")
            if isinstance(ps, list) and len(ps) > 0:
                return ps

            # Parcourir récursivement
            for v in x.values():
                r = find_ps(v)
                if r is not None:
                    return r
        elif isinstance(x, list):
            for v in x:
                r = find_ps(v)
                if r is not None:
                    return r
        return None

    return find_ps(player_obj.get("PlayerTeamStats")) or []


def extract_personal_score_awards(
    match_json: dict[str, Any],
    xuid: str,
) -> list[dict[str, Any]]:
    """Extrait les PersonalScores depuis le JSON du match.

    Les PersonalScores sont la décomposition du score personnel en
    différents types d'actions (kills, assists, objectifs, etc.).

    Chemin API: Players[].PlayerTeamStats[].Stats.CoreStats.PersonalScores[]

    Args:
        match_json: JSON brut du match (MatchStats).
        xuid: XUID du joueur.

    Returns:
        Liste de dicts avec name_id, count, total_score.
        Ex: [{"name_id": 1024030246, "count": 7, "total_score": 700}, ...]
    """
    players = match_json.get("Players")
    if not isinstance(players, list):
        return []

    # Trouver le joueur
    me = _find_player(players, xuid)
    if me is None:
        return []

    # Parcourir les structures pour trouver PersonalScores
    personal_scores = _find_personal_scores(me)
    if not personal_scores:
        return []

    result = []
    for ps in personal_scores:
        if not isinstance(ps, dict):
            continue

        name_id = _safe_int(ps.get("NameId"))
        if name_id is None:
            continue

        count = _safe_int(ps.get("Count")) or 0
        total_score = _safe_int(ps.get("TotalPersonalScoreAwarded"))
        if total_score is None:
            total_score = count * PERSONAL_SCORE_POINTS.get(name_id, 0)

        result.append(
            {
                "name_id": name_id,
                "count": count,
                "total_score": total_score,
            }
        )

    return result


def categorize_personal_score(name_id: int) -> str:
    """Détermine la catégorie d'un PersonalScore.

    Args:
        name_id: ID du score (PersonalScoreNameId).

    Returns:
        Catégorie: "kill", "assist", "objective", "vehicle", "penalty", "other".
    """
    from src.data.domain.refdata import (
        ASSIST_SCORES,
        KILL_SCORES,
        OBJECTIVE_SCORES,
        PENALTY_SCORES,
        VEHICLE_DESTRUCTION_SCORES,
    )

    if name_id in KILL_SCORES:
        return "kill"
    if name_id in ASSIST_SCORES:
        return "assist"
    if name_id in OBJECTIVE_SCORES:
        return "objective"
    if name_id in VEHICLE_DESTRUCTION_SCORES:
        return "vehicle"
    if name_id in PENALTY_SCORES:
        return "penalty"
    return "other"


def transform_personal_score_awards(
    match_id: str,
    xuid: str,
    personal_scores: list[dict[str, Any]],
) -> list:
    """Transforme les PersonalScores en PersonalScoreAwardRows.

    Args:
        match_id: ID du match.
        xuid: XUID du joueur.
        personal_scores: Liste de dicts depuis extract_personal_score_awards().

    Returns:
        Liste de PersonalScoreAwardRow.
    """
    from src.data.domain.refdata import get_personal_score_technical_id
    from src.data.sync.models import PersonalScoreAwardRow

    rows = []
    for ps in personal_scores:
        name_id = ps.get("name_id")
        if name_id is None:
            continue

        category = categorize_personal_score(name_id)
        technical_id = get_personal_score_technical_id(name_id)

        rows.append(
            PersonalScoreAwardRow(
                match_id=match_id,
                xuid=xuid,
                award_name=technical_id,
                award_category=category,
                award_count=ps.get("count", 1),
                award_score=ps.get("total_score", 0),
            )
        )

    return rows
