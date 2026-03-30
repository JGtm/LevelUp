"""Types partagés pour l'analyse d'impact coéquipiers."""

from __future__ import annotations

from dataclasses import dataclass

# Constantes de scoring
SCORE_CLUTCH_FINISHER = 2
SCORE_FIRST_BLOOD = 2
SCORE_LAST_CASUALTY = -2
SCORE_SILENT_HERO = 1.5
SCORE_FALSE_BROTHER = -1.5
SCORE_LAST_GROUP_KILL = -1
SCORE_FIRST_GROUP_DEATH = -1
SCORE_TOP_KILLER = 1.0

# Codes d'outcome
OUTCOME_WIN = 2
OUTCOME_LOSS = 3


@dataclass
class ImpactEvent:
    """Représente un événement d'impact dans un match."""

    match_id: str
    xuid: str
    gamertag: str
    time_ms: int
    event_type: str  # "first_blood", "clutch_finisher", "last_casualty", etc.
