"""Calcul de la cadence de kills par tranche de temps dans un match.

Module d'analyse pure — pas d'accès DB ni Streamlit.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any


@dataclass
class CadenceBucket:
    """Nombre de kills par équipe sur une tranche de temps."""

    t_start_s: float
    t_end_s: float
    my_kills: int = 0
    enemy_kills: int = 0

    @property
    def t_center_s(self) -> float:
        return (self.t_start_s + self.t_end_s) / 2.0

    @property
    def total(self) -> int:
        return self.my_kills + self.enemy_kills


def compute_cadence_buckets(
    events: list[dict[str, Any]],
    xuid_to_team: dict[str, int],
    my_team_id: int,
    duration_s: float,
    bucket_s: int = 30,
) -> list[CadenceBucket]:
    """Découpe les kills du match en tranches de *bucket_s* secondes.

    Args:
        events: Liste de dicts avec clés ``event_type``, ``time_ms``, ``xuid``.
        xuid_to_team: Mapping xuid → team_id.
        my_team_id: ID de l'équipe du joueur courant.
        duration_s: Durée totale du match en secondes.
        bucket_s: Largeur de chaque tranche en secondes.

    Returns:
        Liste ordonnée de CadenceBucket.
    """
    if duration_s <= 0 or bucket_s <= 0:
        return []

    n_buckets = max(1, int(duration_s / bucket_s) + (1 if duration_s % bucket_s else 0))
    buckets = [
        CadenceBucket(t_start_s=i * bucket_s, t_end_s=min((i + 1) * bucket_s, duration_s))
        for i in range(n_buckets)
    ]

    for ev in events:
        if str(ev.get("event_type", "")).lower() != "kill":
            continue
        time_ms = ev.get("time_ms")
        if time_ms is None:
            continue

        time_s = int(time_ms) / 1000.0
        idx = min(int(time_s / bucket_s), n_buckets - 1)
        if idx < 0:
            idx = 0

        killer_xuid = str(ev.get("xuid", "")).strip()
        killer_team = xuid_to_team.get(killer_xuid)

        if killer_team == my_team_id:
            buckets[idx].my_kills += 1
        elif killer_team is not None:
            buckets[idx].enemy_kills += 1

    return buckets


def compute_cadence_moving_avg(
    buckets: list[CadenceBucket],
    window: int = 3,
) -> list[float]:
    """Moyenne glissante du total de kills sur *window* tranches.

    Returns:
        Liste de flottants de même longueur que *buckets*.
    """
    if not buckets:
        return []

    totals = [b.total for b in buckets]
    result: list[float] = []
    for i in range(len(totals)):
        start = max(0, i - window + 1)
        result.append(sum(totals[start : i + 1]) / (i - start + 1))
    return result
