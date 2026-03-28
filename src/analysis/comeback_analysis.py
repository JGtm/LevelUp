"""Détection des badges narrative de comeback (Remontada / Débandade / Contre-Remontada).

Algorithme basé sur les kill-events de ``highlight_events`` (event_type='kill')
et leur timestamp (``time_ms``). Le module est 100 % pur (pas d'accès DB) :
toutes les données sont reçues en paramètre sous forme de DataFrames Polars.

Algorithme — max-deficit sur tout le match
------------------------------------------
On reconstruit la timeline des frags par équipe (kills cumulés triés par ``time_ms``)
et on cherche l'écart **maximal** atteint à un quelconque instant du match.
Aucun "checkpoint" fixe n'est nécessaire : c'est le pire moment qui qualifie.

Concepts
--------
- **Remontada** : on était derrière de ≥ COMEBACK_DEFICIT_THRESHOLD frags d'équipe
  à un moment quelconque, mais on a gagné.
- **Débandade** : on était en avance de ≥ COMEBACK_DEFICIT_THRESHOLD frags d'équipe
  à un moment quelconque, mais on a perdu.
- **Contre-Remontada** (miroir de Remontada) : on était en avance de ≥ COMEBACK_DEFICIT_THRESHOLD
  frags à un moment, l'adversaire a failli revenir (a au moins égalisé), mais on a tenu et gagné.

Les seuils sont définis dans :mod:`src.analysis._medal_verdicts`.
"""

from __future__ import annotations

import logging

import polars as pl

from src.analysis._medal_verdicts import (
    COMEBACK_DEFICIT_THRESHOLD,
    DominanceFlag,
)

logger = logging.getLogger(__name__)

_FLAG_NONE = int(DominanceFlag.NONE)


def _build_kill_differential_series(
    events: pl.DataFrame,
    participants: pl.DataFrame,
    my_xuid: str,
) -> list[int] | None:
    """Construit la série du différentiel kills (enemy - my_team) tri chronologique.

    Args:
        events: DataFrame avec colonnes ``event_type``, ``time_ms``, ``xuid``.
        participants: DataFrame avec colonnes ``xuid``, ``team_id``.
        my_xuid: XUID du joueur analysé (pour identifier son équipe).

    Returns:
        Liste du différentiel cumulé après chaque kill (positif = ennemi devant),
        ou ``None`` si les données sont insuffisantes.
    """
    if events.is_empty() or participants.is_empty():
        return None

    my_rows = participants.filter(pl.col("xuid") == my_xuid)
    if my_rows.is_empty():
        return None
    my_team_id = my_rows["team_id"][0]

    kill_events = events.filter(pl.col("event_type") == "kill").sort("time_ms")
    if kill_events.is_empty():
        return None

    xuid_to_team = dict(
        zip(participants["xuid"].to_list(), participants["team_id"].to_list(), strict=False)
    )

    my_count = 0
    enemy_count = 0
    diffs: list[int] = []

    for killer_xuid in kill_events["xuid"].to_list():
        team = xuid_to_team.get(killer_xuid)
        if team == my_team_id:
            my_count += 1
        else:
            enemy_count += 1
        diffs.append(enemy_count - my_count)

    return diffs


def detect_comeback_badge(
    events: pl.DataFrame,
    participants: pl.DataFrame,
    my_xuid: str,
    outcome: int,
) -> int:
    """Classifie le badge narrative d'un match pour un joueur.

    Args:
        events: DataFrame ``highlight_events`` du match (event_type, time_ms, xuid).
        participants: DataFrame ``match_participants`` du match (xuid, team_id).
        my_xuid: XUID du joueur analysé.
        outcome: Résultat du joueur (2=victoire, 3=défaite, 1=égalité, 4=DNF).

    Returns:
        Valeur entière de :class:`DominanceFlag` (0-5).
    """
    if outcome not in (2, 3):  # Victoire ou défaite uniquement
        return _FLAG_NONE

    diffs = _build_kill_differential_series(events, participants, my_xuid)
    if not diffs:
        return _FLAG_NONE

    # Différentiel ennemi - nous : positif = ennemi devant, négatif = nous devant
    max_deficit = max(diffs)  # pire moment où l'ennemi était devant
    max_lead = -min(diffs)  # meilleur moment où nous étions devant

    won = outcome == 2

    # Remontada : on était très derrière à un moment mais on a gagné
    if won and max_deficit >= COMEBACK_DEFICIT_THRESHOLD:
        return int(DominanceFlag.REMONTADA)

    # Débandade : on avait une grosse avance à un moment mais on a perdu
    if not won and max_lead >= COMEBACK_DEFICIT_THRESHOLD:
        return int(DominanceFlag.DEBANDADE)

    # Contre-Remontada (miroir de Remontada) : on avait 25+ frags d'avance,
    # l'adversaire a failli revenir (a au moins égalisé/dépassé), mais on a tenu.
    if won and max_lead >= COMEBACK_DEFICIT_THRESHOLD and max_deficit >= 1:
        return int(DominanceFlag.CONTRE_REMONTADA)

    return _FLAG_NONE
