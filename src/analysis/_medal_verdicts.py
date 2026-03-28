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
# Basés sur une analyse corpus de 931 matchs PvP (highlight_events, equipe entière).
# Ces constantes permettent d'ajuster la sensibilité sans toucher au code.

# Écart maximal de kills (équipe) à n'importe quel moment du match pour qualifier.
# Seuil 25 → ~0.5 % des matchs (5 remontadas / 931). Fallback si win_score inconnu.
# Utilisé à la fois pour Remontada (on était derrière de X+) et
# Contre-Remontada (on était devant de X+, les deux sont symétriques).
COMEBACK_DEFICIT_THRESHOLD: int = 25

# Seuil de win_score au-delà duquel le mode est considéré objectif (non-Slayer).
# Les modes objectif (CTF, Strongholds, Oddball, etc.) ont des scores en centaines
# ou milliers ; les modes Slayer ont un win_score ≤ 100 (Arena=50, BTB=100).
COMEBACK_MAX_SLAYER_WIN_SCORE: int = 100

# Plancher minimal du seuil pour éviter le bruit sur les modes très courts
# (ex: Escalation Slayer win_score=11 → threshold=5).
COMEBACK_MIN_THRESHOLD: int = 5


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
