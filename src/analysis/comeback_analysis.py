"""Détection des badges narrative de comeback (Remontada / Débandade / Contre-Remontada).

Algorithme basé sur les kill-events de ``highlight_events`` (event_type='kill')
et leur timestamp (``time_ms``).  Le module est 100 % pur (pas d'accès DB) :
toutes les données sont reçues en paramètre sous forme de DataFrames Polars.

Concepts
--------
- **Remontada** : on était mené de ≥ COMEBACK_DEFICIT_THRESHOLD kills au checkpoint,
  mais on a gagné.
- **Débandade** : on était en tête de ≥ COMEBACK_COLLAPSE_CUTOFF kills au checkpoint,
  mais on a perdu.
- **Contre-Remontada** : l'adversaire était en tête de ≥ COMEBACK_COUNTER_GAP kills
  au checkpoint ET remontait (était derrière avant ce checkpoint), mais on a tenu et gagné.

Les seuils sont définis dans :mod:`src.analysis._medal_verdicts`.
"""

from __future__ import annotations

import logging

import polars as pl

from src.analysis._medal_verdicts import (
    COMEBACK_COUNTER_GAP,
    COMEBACK_DEFICIT_THRESHOLD,
    COMEBACK_EARLY_CUTOFF,
    DominanceFlag,
)

logger = logging.getLogger(__name__)

# Résultat neutre en cas de données insuffisantes
_FLAG_NONE = int(DominanceFlag.NONE)


def build_score_snapshot(
    events: pl.DataFrame,
    participants: pl.DataFrame,
    my_xuid: str,
    duration_ms: int,
    cutoff: float,
) -> tuple[int, int] | None:
    """Compte les kill-events par camp jusqu'au checkpoint.

    Args:
        events: DataFrame avec colonnes ``event_type``, ``time_ms``, ``xuid``.
        participants: DataFrame avec colonnes ``xuid``, ``team_id``.
        my_xuid: XUID du joueur analysé (pour déterminer son équipe).
        duration_ms: Durée totale du match en millisecondes.
        cutoff: Fraction [0-1] du match utilisée comme checkpoint.

    Returns:
        Tuple ``(my_team_kills, enemy_kills)`` au checkpoint, ou ``None``
        si les données sont insuffisantes.
    """
    if events.is_empty() or participants.is_empty() or duration_ms <= 0:
        return None

    # Déterminer l'équipe du joueur analysé
    my_rows = participants.filter(pl.col("xuid") == my_xuid)
    if my_rows.is_empty():
        return None
    my_team_id = my_rows["team_id"][0]

    # Joindre les kill-events avec l'équipe de chaque killer
    kill_events = events.filter(pl.col("event_type") == "kill")
    if kill_events.is_empty():
        return None

    threshold_ms = int(duration_ms * cutoff)
    early_kills = kill_events.filter(pl.col("time_ms") <= threshold_ms)
    if early_kills.is_empty():
        return None

    xuid_to_team = dict(
        zip(participants["xuid"].to_list(), participants["team_id"].to_list(), strict=False)
    )
    my_kills = sum(1 for x in early_kills["xuid"].to_list() if xuid_to_team.get(x) == my_team_id)
    enemy_kills = sum(1 for x in early_kills["xuid"].to_list() if xuid_to_team.get(x) != my_team_id)
    return my_kills, enemy_kills


def detect_comeback_badge(
    events: pl.DataFrame,
    participants: pl.DataFrame,
    my_xuid: str,
    duration_ms: int,
    outcome: int,
) -> int:
    """Classifie le badge narrative d'un match pour un joueur.

    Args:
        events: DataFrame ``highlight_events`` du match (event_type, time_ms, xuid).
        participants: DataFrame ``match_participants`` du match (xuid, team_id).
        my_xuid: XUID du joueur analysé.
        duration_ms: Durée du match en millisecondes.
        outcome: Résultat du joueur (2=victoire, 3=défaite, 1=égalité, 4=DNF).

    Returns:
        Valeur entière de :class:`DominanceFlag` (0-5).
    """
    if outcome not in (2, 3):  # Victoire ou défaite uniquement
        return _FLAG_NONE

    snap = build_score_snapshot(events, participants, my_xuid, duration_ms, COMEBACK_EARLY_CUTOFF)
    if snap is None:
        return _FLAG_NONE

    my_kills, enemy_kills = snap
    diff = enemy_kills - my_kills  # positif = on était derrière

    won = outcome == 2

    if won and diff >= COMEBACK_DEFICIT_THRESHOLD:
        return int(DominanceFlag.REMONTADA)

    if not won and -diff >= COMEBACK_DEFICIT_THRESHOLD:
        return int(DominanceFlag.DEBANDADE)

    if won and diff >= COMEBACK_COUNTER_GAP:
        # L'adversaire était devant au checkpoint mais on a tenu
        return int(DominanceFlag.CONTRE_REMONTADA)

    return _FLAG_NONE
