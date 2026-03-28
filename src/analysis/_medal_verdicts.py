"""Constantes et enum pour les verdicts de domination et de narrative match.

Deux concepts coexistent dans ``player_match_enrichment.dominance_flag`` :

- **Domination** (Steaktacular) : valeurs 1-2, posées par ``dominance_backfill``.
- **Narrative comeback** : valeurs 3-5, posées par ``comeback_backfill``.

Les valeurs sont mutuellement exclusives par construction (une domination exclut
un comeback, et vice-versa).
"""

from __future__ import annotations

from enum import IntEnum

MEDAL_STEAKTACULAR_ID: int = 1169390319  # ID médaille "À table" / "Steaktacular"

# ── Seuils pour la détection des badges narrative comeback ────────────────────
# Nombre minimal de kill-events d'écart au checkpoint pour qualifier le badge.
# Ces constantes permettent d'ajuster la sensibilité sans toucher au code.

# Déficit minimal de kills (highlight) au checkpoint pour qu'une remontée qualifie.
COMEBACK_DEFICIT_THRESHOLD: int = 3

# Avance minimale de l'adversaire au checkpoint pour qualifier un Contre-Remontada.
COMEBACK_COUNTER_GAP: int = 2

# Fraction du temps de match utilisée comme checkpoint Remontada/Contre-Remontada.
COMEBACK_EARLY_CUTOFF: float = 0.60

# Fraction du temps de match utilisée comme checkpoint Débandade.
COMEBACK_COLLAPSE_CUTOFF: float = 0.60


class DominanceFlag(IntEnum):
    """Qualificateur de domination / narrative d'un match.

    Stocké dans ``player_match_enrichment.dominance_flag`` (TINYINT).

    Valeurs 0-2 : Steaktacular (posées par ``dominance_backfill``).
    Valeurs 3-5 : Badges narrative (posées par ``comeback_backfill``).
    """

    NONE = 0
    """Match normal — aucun badge notable."""

    DOMINATION = 1
    """Domination totale — notre équipe a obtenu Steaktacular."""

    HUMILIATION = 2
    """Humiliation totale — l'équipe ennemie a obtenu Steaktacular."""

    REMONTADA = 3
    """Remontada — on était mené au checkpoint, on a renversé la situation."""

    DEBANDADE = 4
    """Débandade — on était en tête au checkpoint, on a perdu."""

    CONTRE_REMONTADA = 5
    """Contre-Remontada — l'adversaire revenait au checkpoint, on a tenu."""
