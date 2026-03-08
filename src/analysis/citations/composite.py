"""Logique d'agrégation pour l'affichage des citations composites."""

from __future__ import annotations

import json
from typing import Any


def _apply_composite_citations(
    result: dict[str, int],
    mappings: dict[str, dict[str, Any]],
) -> dict[str, int]:
    """Calcule les citations composites et les injecte dans result.

    Une citation composite est masterisée si toutes ses sous-citations
    ont atteint leur seuil cible.

    Args:
        result: Agrégat {citation_name_norm: total} à enrichir.
        mappings: Dictionnaire de mappings (depuis load_mappings).

    Returns:
        result enrichi avec les citations composites.
    """
    for norm_name, mapping in mappings.items():
        if mapping.get("mapping_type") != "composite":
            continue
        children_raw = mapping.get("composite_children")
        if not children_raw:
            continue
        try:
            children = json.loads(children_raw) if isinstance(children_raw, str) else children_raw
        except (json.JSONDecodeError, TypeError):
            continue
        if not isinstance(children, list):
            continue

        mastered_count = 0
        for child in children:
            if child not in mappings:
                continue
            child_count = result.get(child, 0)
            if child_count <= 0:
                continue
            child_tiers = mappings[child].get("tier_targets")
            if not child_tiers:
                mastered_count += 1
                continue
            targets = sorted(
                int(t.strip()) for t in str(child_tiers).split(",") if t.strip().isdigit()
            )
            if targets and child_count >= targets[-1]:
                mastered_count += 1
        if mastered_count > 0:
            result[norm_name] = mastered_count

    return result
