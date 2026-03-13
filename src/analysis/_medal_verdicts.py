"""Constantes et enum pour les verdicts de domination (médaille Steaktacular).

La médaille "À table" (Steaktacular) est décernée à l'équipe gagnante
lorsqu'elle écrase l'adversaire. Ce module fournit l'enum ``DominanceFlag``
qui qualifie le degré de domination d'un match.
"""

from __future__ import annotations

from enum import IntEnum

MEDAL_STEAKTACULAR_ID: int = 1169390319  # ID médaille "À table" / "Steaktacular"


class DominanceFlag(IntEnum):
    """Qualificateur de domination d'un match.

    Stocké dans ``player_match_enrichment.dominance_flag`` (TINYINT).
    """

    NONE = 0
    """Match normal — pas de Steaktacular."""

    DOMINATION = 1
    """Domination totale — notre équipe a obtenu Steaktacular."""

    HUMILIATION = 2
    """Humiliation totale — l'équipe ennemie a obtenu Steaktacular."""
