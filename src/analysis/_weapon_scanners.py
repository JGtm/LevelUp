"""Fonctions de scan bas-niveau pour le weapon parser.

Scanneurs Section 1 (Formula A) et Section 2 (Fire Events).
Extraits de weapon_parser.py pour respecter la limite de 500L.
"""

from __future__ import annotations

from collections.abc import Callable

from bitstring import Bits

from src.analysis._weapon_data import (
    WEAPON_ID_MAP,
    WEAPON_IDS_INT,
    WEAPON_INT_TO_NAME,
)

# Pattern Section 1 (état snapshot) — Formula A : [20 00 02 pb ... wid:8B]
FORMULA_A_PATTERN = bytes.fromhex("200002")
COMMON_WEAPON_SUFFIX = bytes.fromhex("42c9679f")
FRAME_MARKER = bytes([0xA0, 0x7B, 0x42])
_WEAPON_BIT_OFFSET = 40

_ALL_FORMULA_A_SUFFIXES: frozenset[bytes] = frozenset(k[4:] for k in WEAPON_ID_MAP if len(k) == 8)


# ── Estimation temporelle ──


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


def estimate_ts_frames(
    chunk_data: bytes,
    chunk_start_ms: int,
    chunk_duration_ms: int,
) -> Callable[[int], float]:
    """Retourne une closure estimate_ts(byte_pos) → float."""
    frame_positions = find_frame_positions(chunk_data)
    n_frames = len(frame_positions)
    frame_duration_ms = chunk_duration_ms / n_frames if n_frames > 0 else 16.67

    def estimate_ts(byte_pos: int) -> float:
        lo, hi = 0, n_frames - 1
        frame_idx = 0
        while lo <= hi:
            mid = (lo + hi) // 2
            if frame_positions[mid] <= byte_pos:
                frame_idx = mid
                lo = mid + 1
            else:
                hi = mid - 1
        return chunk_start_ms + frame_idx * frame_duration_ms

    return estimate_ts


# ── Section 1 — Formula A (état snapshot arme) ──


def scan_formula_a(data: bytes) -> list[tuple[int, int, bytes]]:
    """Scanne les mises à jour d'état arme Section 1 (Formula A).

    Returns:
        Liste de (offset, player_index, weapon_id_bytes).
    """
    results: list[tuple[int, int, bytes]] = []
    pos = 0
    while True:
        pos = data.find(FORMULA_A_PATTERN, pos)
        if pos == -1 or pos + 4 > len(data):
            break
        pb = data[pos + 3]
        pi = pb >> 5
        end = min(pos + 68, len(data))
        best_sx = -1
        for suffix in _ALL_FORMULA_A_SUFFIXES:
            sx_c = data.find(suffix, pos + 4, end)
            if sx_c < 4:
                continue
            ws_c = sx_c - 4
            if ws_c <= pos + 3:
                continue
            if suffix != COMMON_WEAPON_SUFFIX and data[ws_c : ws_c + 8] not in WEAPON_ID_MAP:
                continue
            if best_sx == -1 or sx_c < best_sx:
                best_sx = sx_c
        if best_sx >= 4:
            ws = best_sx - 4
            if ws > pos + 3:
                results.append((pos, pi, data[ws : ws + 8]))
        pos += 4
    return results


def scan_formula_a_ns(data: bytes) -> list[tuple[int, int, bytes]]:
    """Scanne Section 1 dans la couche nibble-shiftée pour les TYPE IDs.

    Returns:
        Liste de (ns_pos, player_index, weapon_bytes).
    """
    ns = bytes((data[i] << 4 | data[i + 1] >> 4) & 0xFF for i in range(len(data) - 1))
    results: list[tuple[int, int, bytes]] = []
    for wb in WEAPON_ID_MAP:
        if len(wb) != 8:
            continue
        pos = 0
        while True:
            p = ns.find(wb, pos)
            if p == -1:
                break
            if p >= 5 and ns[p - 5] != 0x26:
                pi = ns[p - 1] >> 5
                results.append((p, pi, wb))
            pos = p + 1
    results.sort(key=lambda x: x[0])
    return results


# ── Section 2 — Fire Events (bitstring scan) ──


def _build_marker(player_index: int) -> Bits:
    """Construit le marqueur 11 bits pour un player_index donné."""
    byte1 = (player_index << 5) | 0x06
    return Bits(bin=f"0b101{byte1:08b}")


def scan_fire_events_bitstring(
    chunk_data: bytes,
    player_index: int,
    estimate_ts: Callable,
) -> list[dict]:
    """Scanne les fire events via recherche exhaustive bitstring."""
    bits = Bits(bytes=chunk_data)
    marker = _build_marker(player_index)
    total_bits = len(bits)
    events: list[dict] = []

    for position in bits.findall(marker, bytealigned=False):
        event_start = position + 3
        weapon_start = event_start + _WEAPON_BIT_OFFSET
        if weapon_start + 64 > total_bits:
            continue

        weapon_int = bits[weapon_start : weapon_start + 64].uint
        weapon_bytes = weapon_int.to_bytes(8, byteorder="big")
        if weapon_int not in WEAPON_IDS_INT and weapon_bytes[4:] != COMMON_WEAPON_SUFFIX:
            continue

        byte_pos = position // 8
        fire_seq = (
            bits[event_start + 8 : event_start + 16].uint if event_start + 16 <= total_bits else 0
        )
        fire_counter = (
            bits[event_start + 24 : event_start + 32].uint if event_start + 32 <= total_bits else 0
        )
        weapon_name = WEAPON_INT_TO_NAME.get(
            weapon_int,
            WEAPON_ID_MAP.get(weapon_bytes, f"INCONNU ({weapon_bytes.hex()})"),
        )
        post_start = weapon_start + 64
        if post_start + 32 <= total_bits:
            post_bytes = bits[post_start : post_start + 32].bytes
        else:
            post_bytes = b"\x00" * 4
        burst_end = bool(post_bytes[1] & 0x01) if len(post_bytes) > 1 else False
        hit_likely = bool((post_bytes[2] & 0x01) == 0) if len(post_bytes) > 2 else None

        events.append(
            {
                "timestamp_ms": estimate_ts(byte_pos),
                "weapon_name": weapon_name,
                "weapon_bytes": weapon_bytes,
                "fire_seq": fire_seq,
                "fire_counter": fire_counter,
                "bit_offset": position % 8,
                "byte_pos": byte_pos,
                "post_bytes": post_bytes,
                "burst_end": burst_end,
                "hit_likely": hit_likely,
            }
        )

    deduped: dict[tuple[int, bytes], dict] = {}
    for ev in sorted(events, key=lambda x: x["timestamp_ms"]):
        key = (ev["fire_counter"], ev["weapon_bytes"])
        if key not in deduped:
            deduped[key] = ev
    return sorted(deduped.values(), key=lambda x: x["timestamp_ms"])
