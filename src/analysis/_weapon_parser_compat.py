"""Fonctions de compatibilité v1 du weapon parser.

À supprimer après migration complète vers le pipeline v2.
Conservé pour les tests existants et les scripts expérimentaux.
"""

from __future__ import annotations

from collections import Counter
from collections.abc import Callable

from src.analysis._weapon_data import (
    GRENADE_WEAPON_ID,
    MELEE_WEAPON_ID,
    WEAPON_BYTES_TO_INT,
    WEAPON_TIMING_BY_ID,
)
from src.analysis._weapon_scanners import (
    FRAME_MARKER,
    estimate_ts_frames,
    scan_fire_events_bitstring,
)
from src.analysis.weapon_parser import (
    KILL_WINDOW_MS,
    compute_confidence,
)

_DEFAULT_TIMING = (650, 2000)


def _special_kill(kill: dict, weapon_id: int) -> dict:
    """Construit un résultat pour un kill spécial (melee/grenade)."""
    return {
        **kill,
        "weapon_id": weapon_id,
        "matched_fire_event": None,
        "delta_ms": None,
        "confidence": "none",
        "swap_detected": False,
        "delayed_damage": False,
    }


def correlate_kills_to_weapons(
    kills: list[dict],
    fire_events_all: list[dict],
) -> list[dict]:
    """Compat v1 : corrélation POV simple, sans claim-and-remove."""
    results = []
    for kill in kills:
        kill_t = kill["time_ms"]
        if kill.get("is_melee"):
            results.append(_special_kill(kill, MELEE_WEAPON_ID))
            continue
        if kill.get("is_grenade"):
            results.append(_special_kill(kill, GRENADE_WEAPON_ID))
            continue

        candidates = [
            ev
            for ev in fire_events_all
            if (kill_t - KILL_WINDOW_MS) <= ev["timestamp_ms"] <= kill_t
        ]
        best = max(candidates, key=lambda e: e["timestamp_ms"]) if candidates else None

        if best is None:
            wid_int = None
        else:
            wid_int = WEAPON_BYTES_TO_INT.get(best["weapon_bytes"])
            if wid_int is None:
                wid_int = int.from_bytes(best["weapon_bytes"], byteorder="big")
        delta = int(kill_t - best["timestamp_ms"]) if best else None
        conf = compute_confidence(wid_int, delta)
        swap_ms, travel_max = (
            WEAPON_TIMING_BY_ID.get(wid_int, _DEFAULT_TIMING) if wid_int else _DEFAULT_TIMING
        )
        if conf == "medium" and best is not None:
            post_swap = [
                ev
                for ev in fire_events_all
                if kill_t < ev["timestamp_ms"] <= kill_t + swap_ms
                and ev["weapon_bytes"] != best["weapon_bytes"]
            ]
            if post_swap:
                w2_ev = min(post_swap, key=lambda e: e["timestamp_ms"])
                w2_int = WEAPON_BYTES_TO_INT.get(w2_ev["weapon_bytes"])
                if w2_int is None:
                    w2_int = int.from_bytes(w2_ev["weapon_bytes"], byteorder="big")
                wid_int = w2_int
                conf = "high"

        swap_detected = best is not None and delta is not None and delta >= swap_ms
        delayed = best is not None and delta is not None and delta > travel_max
        results.append(
            {
                **kill,
                "weapon_id": wid_int,
                "fire_seq": best["fire_seq"] if best else None,
                "matched_fire_event": best,
                "delta_ms": delta,
                "confidence": conf,
                "swap_detected": swap_detected,
                "delayed_damage": delayed,
            }
        )
    return results


def count_kills_by_film_weapon(correlated: list[dict]) -> Counter:
    """Agrège les kills corrélés en Counter {weapon_id: n_kills}."""
    counts: Counter = Counter()
    for r in correlated:
        wid = r.get("weapon_id")
        if wid is not None:
            counts[wid] += 1
    return counts


count_kills_by_api_weapon = count_kills_by_film_weapon


def find_frame_positions(data: bytes) -> list[int]:
    """Trouve toutes les positions des frame markers A0 7B 42."""
    positions: list[int] = []
    pos = 0
    while True:
        idx = data.find(FRAME_MARKER, pos)
        if idx == -1:
            break
        positions.append(idx)
        pos = idx + 1
    return positions


def build_frame_estimator(
    chunk_data: bytes,
    chunk_start_ms: int,
    chunk_duration_ms: int,
) -> Callable[[int], float]:
    """Retourne une fonction estimate_ts(byte_pos) → float."""
    return estimate_ts_frames(chunk_data, chunk_start_ms, chunk_duration_ms)


def scan_all_players(
    chunk_data: bytes,
    chunk_start_ms: int,
    chunk_duration_ms: int,
    n_players: int = 8,
) -> dict[int, list[dict]]:
    """Scanne les fire events pour tous les player_index (diagnostic)."""
    ts_fn = estimate_ts_frames(chunk_data, chunk_start_ms, chunk_duration_ms)
    return {idx: scan_fire_events_bitstring(chunk_data, idx, ts_fn) for idx in range(n_players)}
