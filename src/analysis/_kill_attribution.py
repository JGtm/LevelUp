"""Dataclass centrale pour le résultat d'attribution d'un kill à une arme.

Séparée du parser pour éviter les imports circulaires
(utilisée par parser, service, reconciliation, repository).
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(slots=True)
class KillAttribution:
    """Résultat d'attribution d'un kill à une arme."""

    match_id: str
    xuid: str
    time_ms: int
    weapon_id: int | None
    reconciled_as: int | None
    delta_ms: int | None
    confidence: str
    attribution_path: str
    swap_detected: bool
    delayed_damage: bool
    player_index: int | None
    source_chunk_idx: int | None

    @property
    def effective_weapon_id(self) -> int | None:
        """Arme effective = COALESCE(reconciled_as, weapon_id)."""
        return self.reconciled_as if self.reconciled_as is not None else self.weapon_id
