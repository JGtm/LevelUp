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

# Pourcentage de kills-écart par rapport au win_score pour qualifier un badge.
# Référence : Arena Slayer (50 kills) × 40 % = 20 kills d'écart.
COMEBACK_DEFICIT_PCT: float = 0.40

# Fallback quand le win_score est inconnu (= Arena Slayer × 40 %).
COMEBACK_DEFICIT_FALLBACK: int = 20

# Seuil de win_score au-delà duquel le mode est considéré objectif (non-Slayer).
# Les modes objectif (CTF, Strongholds, Oddball, etc.) ont des scores en centaines
# ou milliers ; les modes Slayer ont un win_score ≤ 100 (Arena=50, BTB=100).
COMEBACK_MAX_SLAYER_WIN_SCORE: int = 100

# ── Scores cibles (win_score) des modes Slayer connus ────────────────────────
SLAYER_WIN_SCORES: dict[str, int] = {
    "arena_slayer": 50,  # Arena Slayer (toutes variantes : Super Fiesta, Tactical…)
    "btb_slayer": 100,  # BTB Slayer (toutes variantes : Heavies, Fiesta…)
    "escalation_slayer": 11,  # Escalation Slayer (Arena, BTB, Event)
}

# ── Scores max observés par mode (référence, non utilisé par l'algorithme) ───
# Données issues de l'analyse de ~1 600 matchs PvP (mars 2026).
# Utile pour un futur support des badges sur les modes objectifs.
MODE_MAX_SCORES: dict[str, int] = {
    # Slayer
    "arena_slayer": 50,
    "btb_slayer": 100,
    "escalation_slayer": 11,
    # Objectif — captures / rounds
    "ctf_arena": 3,
    "ctf_neutral_flag": 5,
    "btb_ctf": 3,
    "one_flag_ctf": 3,
    "total_control": 3,
    "stockpile": 5,
    "koth": 3,  # KOTH : rounds gagnés (0-5 observé)
    "attrition": 3,  # Rounds gagnés
    "assault_neutral_bomb": 3,
    "assault_one_bomb": 3,
    "extraction": 7,
    # Objectif — scoring continu (ticks / secondes)
    "strongholds": 200,  # Scoring ticks (132-200 observé)
    "oddball": 200,  # Secondes (160-287 observé, 200 en Ranked)
    "sentry_defense": 400,  # Estimation (165-319 observé hors corruption)
}


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
