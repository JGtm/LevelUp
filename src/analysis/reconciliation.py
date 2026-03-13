"""Réconciliation post-traitement API découplée.

Applique les agrégats API (kills par arme) pour résoudre les attributions
à faible confidence. N'écrase JAMAIS weapon_id — écrit dans reconciled_as.
"""

from __future__ import annotations

import logging
from collections import Counter
from typing import TYPE_CHECKING

from src.analysis._kill_attribution import KillAttribution
from src.analysis._weapon_data import (
    GRENADE_WEAPON_ID,
    MELEE_WEAPON_ID,
    VEHICLE_WEAPON_ID,
    WEAPON_INT_TO_NAME,
)

if TYPE_CHECKING:
    from src.analysis._parser_logging import MatchLogCollector

logger = logging.getLogger(__name__)

# IDs sentinels qui ne doivent pas être réconciliés
_SENTINEL_IDS = frozenset({MELEE_WEAPON_ID, GRENADE_WEAPON_ID, VEHICLE_WEAPON_ID})


def reconcile_api_aggregates(
    attributions: list[KillAttribution],
    api_weapon_counts: dict[int, int],
    *,
    log_collector: MatchLogCollector | None = None,
) -> list[KillAttribution]:
    """Compare les attributions film vs agrégats API.

    Pour les kills à confidence "low" ou "none" (hors melee/grenade) :
    - Si un surplus API existe pour une arme, assigne la valeur dans reconciled_as
    - weapon_id n'est JAMAIS modifié

    Args:
        attributions: Résultats de correlate_kills().
        api_weapon_counts: {weapon_id_api: n_kills} depuis l'API stats.
        log_collector: Logging structuré (optionnel).

    Returns:
        Liste enrichie (reconciled_as rempli si applicable).
    """
    # Compter les attributions film high/medium par arme
    film_counts = _count_confident_attributions(attributions)

    # Calculer le surplus/déficit API - film par arme
    surplus = _compute_surplus(api_weapon_counts, film_counts)

    if not surplus:
        return attributions

    # Trier les low/none par temps pour attribution séquentielle
    results = list(attributions)
    reconcilable = [
        (i, a)
        for i, a in enumerate(results)
        if a.confidence in ("low", "none")
        and a.weapon_id not in _SENTINEL_IDS
        and a.attribution_path != "none"  # pas melee/grenade
    ]

    reconciled_count = 0
    for idx, attr in reconcilable:
        # Chercher l'arme API avec le plus grand surplus
        best_wid = _find_best_surplus(surplus)
        if best_wid is None:
            remaining = len(reconcilable) - reconciled_count
            if log_collector and remaining > 0:
                log_collector.warn(
                    "reconciliation_surplus_exhausted",
                    remaining_unreconciled=remaining,
                )
            break

        before = {"weapon_id": attr.weapon_id, "confidence": attr.confidence}
        results[idx] = _with_reconciled(attr, best_wid)
        reconciled_count += 1  # noqa: SIM113 — not an index, tracks conditional reconciliations
        surplus[best_wid] -= 1
        if surplus[best_wid] <= 0:
            del surplus[best_wid]

        if log_collector:
            after = {"reconciled_as": best_wid, "name": WEAPON_INT_TO_NAME.get(best_wid)}
            log_collector.reconciliation_decision(
                "assign_sentinel",
                attr.time_ms,
                attr.xuid,
                before,
                after,
            )

    return results


def _with_reconciled(attr: KillAttribution, best_wid: int) -> KillAttribution:
    """Crée une copie de l'attribution avec ``reconciled_as`` rempli."""
    return KillAttribution(
        match_id=attr.match_id,
        xuid=attr.xuid,
        time_ms=attr.time_ms,
        weapon_id=attr.weapon_id,
        reconciled_as=best_wid,
        delta_ms=attr.delta_ms,
        confidence=attr.confidence,
        attribution_path=attr.attribution_path,
        swap_detected=attr.swap_detected,
        delayed_damage=attr.delayed_damage,
        player_index=attr.player_index,
        source_chunk_idx=attr.source_chunk_idx,
    )


def _count_confident_attributions(attributions: list[KillAttribution]) -> Counter:
    """Compte les attributions high/medium par weapon_id."""
    counts: Counter = Counter()
    for a in attributions:
        if a.confidence in ("high", "medium") and a.weapon_id is not None:
            counts[a.weapon_id] += 1
    return counts


def _compute_surplus(
    api_counts: dict[int, int],
    film_counts: Counter,
) -> dict[int, int]:
    """Calcule le surplus API par arme (API - film confident)."""
    surplus: dict[int, int] = {}
    for wid, api_n in api_counts.items():
        if wid in _SENTINEL_IDS:
            continue
        film_n = film_counts.get(wid, 0)
        diff = api_n - film_n
        if diff > 0:
            surplus[wid] = diff
    return surplus


def _find_best_surplus(surplus: dict[int, int]) -> int | None:
    """Retourne le weapon_id avec le plus grand surplus."""
    if not surplus:
        return None
    return max(surplus, key=surplus.get)  # type: ignore[arg-type]


def assign_sentinels(
    attributions: list[KillAttribution],
    sentinel_map: dict[str, int],
    *,
    log_collector: MatchLogCollector | None = None,
) -> list[KillAttribution]:
    """Assigne des sentinels (melee/grenade/vehicle) via reconciled_as.

    Pour les attributions dont le xuid+time_ms correspond à une entrée
    dans sentinel_map, écrit reconciled_as sans toucher weapon_id.
    """
    key_to_sentinel = dict(sentinel_map)
    results = list(attributions)
    assigned_count = 0
    for i, attr in enumerate(results):
        key = f"{attr.xuid}_{attr.time_ms}"
        sentinel = key_to_sentinel.get(key)
        if sentinel is not None and attr.reconciled_as is None:
            assigned_count += 1
            results[i] = KillAttribution(
                match_id=attr.match_id,
                xuid=attr.xuid,
                time_ms=attr.time_ms,
                weapon_id=attr.weapon_id,
                reconciled_as=sentinel,
                delta_ms=attr.delta_ms,
                confidence=attr.confidence,
                attribution_path=attr.attribution_path,
                swap_detected=attr.swap_detected,
                delayed_damage=attr.delayed_damage,
                player_index=attr.player_index,
                source_chunk_idx=attr.source_chunk_idx,
            )
    if log_collector and assigned_count > 0:
        log_collector.record_step("assign_sentinels", assigned=assigned_count)
    return results
