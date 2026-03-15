"""Corrélation globale fire events → kills (claim-and-remove sans bucketing pi).

Dans le protocole film Halo Infinite, tous les fire events partagent le même
marker fixe (0x26) indépendamment du joueur — le bucketing per-pi via
scan_fire_events_bitstring est donc non fiable.

Ce module fournit correlate_kills_global() qui opère sur un pool unique
d'events, toutes têtes triées par temps, claim-and-remove global bijection.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

from src.analysis._kill_attribution import KillAttribution
from src.analysis._weapon_data import (
    GRENADE_WEAPON_ID,
    MELEE_WEAPON_ID,
    WEAPON_BYTES_TO_INT,
    WEAPON_TIMING_BY_ID,
)

if TYPE_CHECKING:
    from src.analysis._parser_logging import MatchLogCollector

_DEFAULT_TIMING = (650, 2000)


def correlate_kills_global(  # noqa: PLR0913
    kills: list[dict],
    fire_events_all: list[dict],
    xuid_to_pi: dict[str, int | None],
    timeline: dict[int, dict[int, bytes]],
    swap_pis: dict[int, set[int]],
    timing: list[tuple[int, int]],
    chunks_sorted: list[int],
    match_id: str = "",
    *,
    log_collector: MatchLogCollector | None = None,
) -> list[KillAttribution]:
    """Corrélation globale tous joueurs — claim-and-remove sur pool unique.

    Remplace correlate_kills() quand le marker de fire event est fixe (0x26)
    et que le bucketing per-pi ne discrimine pas les joueurs.

    Tri kills par temps → claim-and-remove : chaque fire event ne sert qu'une
    fois, la priorité va au kill le plus précoce.

    Args:
        kills: Tous les kills de tous les joueurs du match (avec "xuid").
        fire_events_all: Pool global de tous les fire events (tous pi fusionnés).
        xuid_to_pi: {xuid: pi} pour le fallback Formula A.
        timeline/swap_pis/timing/chunks_sorted: état Section 1 pour Formula A.
        match_id: Pour le logging.
        log_collector: Collecteur de logs structurés (optionnel).
    """
    from src.analysis.weapon_parser import (
        KILL_WINDOW_MS,
        _fallback_formula_a,  # noqa: PLC2701
        compute_confidence,
    )

    available = sorted(fire_events_all, key=lambda e: e["timestamp_ms"])
    kills_sorted = sorted(kills, key=lambda k: k["time_ms"])
    results: list[KillAttribution] = []

    for kill in kills_sorted:
        candidates: list[tuple[int, dict]] = []
        if kill.get("is_melee"):
            attr = _make_sentinel_global(kill, match_id, MELEE_WEAPON_ID)
        elif kill.get("is_grenade"):
            attr = _make_sentinel_global(kill, match_id, GRENADE_WEAPON_ID)
        else:
            t_ms = kill["time_ms"]
            candidates = [
                (i, ev)
                for i, ev in enumerate(available)
                if (t_ms - KILL_WINDOW_MS) <= ev["timestamp_ms"] <= t_ms
            ]
            if candidates:
                best_idx, best_ev = max(candidates, key=lambda x: x[1]["timestamp_ms"])
                available.pop(best_idx)
                attr = _attribution_from_event(kill, best_ev, match_id, compute_confidence)
            else:
                pi = xuid_to_pi.get(kill["xuid"])
                attr = _fallback_formula_a(
                    kill, pi, timeline, swap_pis, timing, chunks_sorted, match_id
                )

        if log_collector:
            log_collector.kill_decision(
                kill,
                attr,
                {
                    "candidates_count": len(candidates),
                    "fallback_used": attr.attribution_path == "formula_a",
                },
            )
        results.append(attr)

    return results


def _make_sentinel_global(kill: dict, match_id: str, weapon_id: int) -> KillAttribution:
    """Sentinel melee/grenade (réplique de _make_sentinel sans import circulaire)."""
    return KillAttribution(
        match_id=match_id,
        xuid=kill["xuid"],
        time_ms=kill["time_ms"],
        weapon_id=weapon_id,
        reconciled_as=None,
        delta_ms=None,
        confidence="none",
        attribution_path="none",
        swap_detected=False,
        delayed_damage=False,
        player_index=None,
        source_chunk_idx=None,
    )


def _attribution_from_event(
    kill: dict,
    ev: dict,
    match_id: str,
    compute_confidence_fn,
) -> KillAttribution:
    """Construit un KillAttribution depuis un fire event claimé."""
    wid_int = WEAPON_BYTES_TO_INT.get(ev["weapon_bytes"])
    if wid_int is None:
        wid_int = int.from_bytes(ev["weapon_bytes"], byteorder="big")
    delta = int(kill["time_ms"] - ev["timestamp_ms"])
    conf = compute_confidence_fn(wid_int, delta)
    swap_ms, travel_max = WEAPON_TIMING_BY_ID.get(wid_int, _DEFAULT_TIMING)
    return KillAttribution(
        match_id=match_id,
        xuid=kill["xuid"],
        time_ms=kill["time_ms"],
        weapon_id=wid_int,
        reconciled_as=None,
        delta_ms=delta,
        confidence=conf,
        attribution_path="fire_event",
        swap_detected=delta >= swap_ms,
        delayed_damage=delta > travel_max,
        player_index=ev.get("player_index"),
        source_chunk_idx=ev.get("chunk_idx"),
    )
