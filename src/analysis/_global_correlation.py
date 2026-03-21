"""Corrélation globale fire events → kills (claim-and-remove sans bucketing pi).

Dans le protocole film Halo Infinite, tous les fire events partagent le même
marker universel (0b10100100110). Le player_index est extrait directement de
b5 >> 4 (inv134) lors du scan — pas de bucketing per-player_index nécessaire.

Ce module fournit correlate_kills_global() qui opère sur un pool unique
d'events triés par temps, avec filtre player_index == killer_pi (claim-and-remove).
"""

from __future__ import annotations

import logging
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

logger = logging.getLogger(__name__)

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
    timeline_ns: dict[int, dict[int, bytes]] | None = None,
    log_collector: MatchLogCollector | None = None,
) -> list[KillAttribution]:
    """Corrélation claim-and-remove par joueur via player_index.

    Chaque kill ne consomme que les fire events dont player_index correspond
    au pi du tueur (b5>>4, extrait lors du scan — inv134).
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
            killer_pi = xuid_to_pi.get(kill["xuid"])
            candidates = [
                (i, ev)
                for i, ev in enumerate(available)
                if (t_ms - KILL_WINDOW_MS) <= ev["timestamp_ms"] <= t_ms
                and (killer_pi is None or ev.get("player_index") == killer_pi)
            ]
            if candidates:
                best_idx, best_ev = max(candidates, key=lambda x: x[1]["timestamp_ms"])
                available.pop(best_idx)
                attr = _attribution_from_event(
                    kill,
                    best_ev,
                    match_id,
                    compute_confidence,
                    timeline_ns=timeline_ns,
                    timing=timing,
                    chunks_sorted=chunks_sorted,
                )
            else:
                pi = xuid_to_pi.get(kill["xuid"])
                attr = _fallback_formula_a(
                    kill,
                    pi,
                    timeline,
                    swap_pis,
                    timing,
                    chunks_sorted,
                    match_id,
                    timeline_ns=timeline_ns,
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


def _attribution_from_event(  # noqa: PLR0913
    kill: dict,
    ev: dict,
    match_id: str,
    compute_confidence_fn,
    *,
    timeline_ns: dict | None = None,
    timing: list | None = None,
    chunks_sorted: list | None = None,
) -> KillAttribution:
    """Construit un KillAttribution depuis un fire event claimé."""
    wid_int = WEAPON_BYTES_TO_INT.get(ev["weapon_bytes"])
    if wid_int is None and timeline_ns and timing is not None and chunks_sorted is not None:
        pi = ev.get("player_index")
        if pi is not None:
            from src.analysis.weapon_parser import find_chunk_at_time  # noqa: PLC0415

            chunk_idx = find_chunk_at_time(chunks_sorted, timing, int(ev["timestamp_ms"]))
            ns_wb = timeline_ns.get(chunk_idx, {}).get(pi)
            if ns_wb:
                wid_int = WEAPON_BYTES_TO_INT.get(ns_wb)
                if wid_int is not None:
                    logger.debug(
                        "match=%s NS lookup hit pi=%s chunk=%s → weapon_id=%s",
                        match_id,
                        pi,
                        chunk_idx,
                        wid_int,
                    )
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
