"""Mapping canonique des outcomes de match (FR/EN).

Données statiques — aucune dépendance vers la couche UI.
"""

from typing import Final

_OUTCOME_LABELS: Final[dict[str, dict[int, str]]] = {
    "fr": {1: "Égalité", 2: "Victoire", 3: "Défaite", 4: "Non terminé"},
    "en": {1: "Tie", 2: "Win", 3: "Loss", 4: "Did Not Finish"},
}


def get_outcome_map(lang: str = "fr") -> dict[int, str]:
    """Retourne {code: label} pour les outcomes. Aucune dépendance vers ui."""
    return _OUTCOME_LABELS.get(lang, _OUTCOME_LABELS["fr"])
