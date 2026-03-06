"""Fragments SQL partagés — source de vérité unique pour les expressions récurrentes.

Évite la duplication des formules (win_rate, outcomes) entre analytics.py et trends.py.
"""

from __future__ import annotations

from src.data.domain.refdata import Outcome

# Valeurs numériques exposées pour injection dans les f-strings SQL
_WIN = Outcome.WIN.value  # 2
_LOSS = Outcome.LOSS.value  # 3
_DNF = Outcome.DID_NOT_FINISH.value  # 4

# ── Expressions CASE WHEN réutilisables ────────────────────────────────────
IS_WIN = f"CASE WHEN outcome = {_WIN} THEN 1 ELSE 0 END"
IS_LOSS = f"CASE WHEN outcome = {_LOSS} THEN 1 ELSE 0 END"

# ── Formule win_rate canonique ─────────────────────────────────────────────
# Dénominateur = WIN + LOSS uniquement (exclut TIE et DNF).
# Retourne NULL si aucun match décisif (évite division par zéro).
WIN_RATE_EXPR = (
    f"SUM({IS_WIN}) * 1.0 "
    f"/ NULLIF(SUM(CASE WHEN outcome IN ({_WIN}, {_LOSS}) THEN 1 ELSE 0 END), 0)"
)
