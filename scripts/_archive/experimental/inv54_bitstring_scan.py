#!/usr/bin/env python3
"""
Investigation #54 — Scan bitstring (méthode acurtis)

Compare la méthode bitstring d'acurtis (recherche bit-level exhaustive)
avec notre double-pass nibble-shift pour identifier des fire events
ou weapon IDs manqués.

Approche acurtis :
  _MARKER = Bits("0b101 0010 0110")  # 11 bits : 3 LSBs lead + byte1=0x26
  _WEAPON_OFFSET = 40 bits (= 5 bytes après le marker anchor)
  Validation : weapon_id (64 bits) ∈ Weapon enum

Notre approche actuelle :
  Double pass (raw + nibble-shift), (b0 & 0x07)==0x05, b1==player_byte,
  (b3 & 0xFC)==0x40, weapon_id ∈ WEAPON_ID_MAP ou suffix 42c9679f

Objectif :
  1. Trouver des events manqués par notre filtre b3
  2. Identifier de nouveaux weapon IDs
  3. Comparer le nombre d'events trouvés par chaque méthode
"""

from __future__ import annotations

import sys
from collections import Counter, defaultdict
from pathlib import Path

from bitstring import Bits

PROJECT_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(PROJECT_ROOT))

from scripts.experimental.weapon_extraction import (
    WEAPON_ID_MAP,
    _build_frame_estimator,
    _scan_fire_events_raw,
    find_frame_positions,
)

# ─── Enum-style set des weapon IDs connus (entiers uint64) ───────────────────

WEAPON_IDS_INT: set[int] = set()
WEAPON_INT_TO_NAME: dict[int, str] = {}
for wid_bytes, name in WEAPON_ID_MAP.items():
    val = int.from_bytes(wid_bytes, byteorder="big")
    WEAPON_IDS_INT.add(val)
    WEAPON_INT_TO_NAME[val] = name

# ─── Marqueur acurtis ───────────────────────────────────────────────────────

# 3 bits LSBs du lead byte (101 = 0x05) + 8 bits de byte[1] (0x26 = pi=1 POV)
_MARKER_POV = Bits("0b101 0010 0110")  # 11 bits, pi=1
_WEAPON_OFFSET = 40  # bits après event_start (= 5 bytes)


def _build_marker(player_index: int) -> Bits:
    """Construit le marqueur 11 bits pour un player_index donné."""
    byte1 = (player_index << 5) | 0x06
    # 3 bits (101) + 8 bits (byte1)
    return Bits(bin=f"0b101{byte1:08b}")


def scan_bitstring(
    chunk_data: bytes,
    player_index: int = 1,
    chunk_start_ms: int = 0,
    chunk_duration_ms: int = 20000,
) -> list[dict]:
    """
    Scan bitstring pur (méthode acurtis).

    Cherche le marqueur 11 bits à TOUTES les positions, valide par weapon_id ∈ enum.
    Pas de filtre b3, pas de nibble-shift manuelle.
    """
    bits = Bits(bytes=chunk_data)
    marker = _build_marker(player_index)
    total_bits = len(bits)

    # Frame positions pour estimation timestamp
    frame_positions = find_frame_positions(chunk_data)
    n_frames = len(frame_positions)
    frame_duration_ms = chunk_duration_ms / n_frames if n_frames > 0 else 16.67

    def estimate_ts(bit_pos: int) -> float:
        """Estime le timestamp à partir d'une position en bits."""
        byte_pos = bit_pos // 8
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

    events: list[dict] = []
    seen_positions: set[int] = set()

    for position in bits.findall(marker, bytealigned=False):
        event_start = position + 3  # Ancre sur byte1 (0x26)
        weapon_start = event_start + _WEAPON_OFFSET

        if weapon_start + 64 > total_bits:
            continue

        weapon_bits = bits[weapon_start : weapon_start + 64]
        weapon_int = weapon_bits.uint

        if weapon_int not in WEAPON_IDS_INT:
            continue

        # Éviter les doublons proches (même event trouvé à ±1 bit)
        if any(abs(position - p) < 8 for p in seen_positions):
            continue
        seen_positions.add(position)

        # Extraire les bytes contextuels
        event_byte_pos = position // 8
        bit_offset = position % 8

        # Extraire byte[3] (à event_start + 16 bits = position + 19 bits)
        b3_start = event_start + 16
        b3 = bits[b3_start : b3_start + 8].uint if b3_start + 8 <= total_bits else -1

        # fire_counter = byte[4] (à event_start + 24 bits)
        fc_start = event_start + 24
        fire_counter = bits[fc_start : fc_start + 8].uint if fc_start + 8 <= total_bits else -1

        # fire_seq = byte[2] (à event_start + 8 bits)
        fs_start = event_start + 8
        fire_seq = bits[fs_start : fs_start + 8].uint if fs_start + 8 <= total_bits else -1

        weapon_name = WEAPON_INT_TO_NAME.get(weapon_int, f"UNKNOWN-{weapon_int:016x}")
        weapon_bytes = weapon_int.to_bytes(8, byteorder="big")

        events.append(
            {
                "bit_position": position,
                "bit_offset": bit_offset,
                "byte_pos": event_byte_pos,
                "timestamp_ms": estimate_ts(position),
                "weapon_name": weapon_name,
                "weapon_int": weapon_int,
                "weapon_bytes": weapon_bytes,
                "fire_seq": fire_seq,
                "fire_counter": fire_counter,
                "b3": b3,
                "b3_masked": b3 & 0xFC if b3 >= 0 else -1,
            }
        )

    # Dédupliquer par (fire_counter, weapon_int)
    deduped: dict[tuple[int, int], dict] = {}
    for ev in sorted(events, key=lambda x: x["timestamp_ms"]):
        key = (ev["fire_counter"], ev["weapon_int"])
        if key not in deduped:
            deduped[key] = ev
    return sorted(deduped.values(), key=lambda x: x["timestamp_ms"])


def scan_legacy(
    chunk_data: bytes,
    player_index: int = 1,
    chunk_start_ms: int = 0,
    chunk_duration_ms: int = 20000,
) -> list[dict]:
    """Notre scan existant (double-pass raw+nibble)."""
    player_byte1 = (player_index << 5) | 0x06
    estimate_ts = _build_frame_estimator(chunk_data, chunk_start_ms, chunk_duration_ms)
    return _scan_fire_events_raw(chunk_data, player_byte1, estimate_ts)


def main() -> None:
    chunks_root = PROJECT_ROOT / "data" / "investigation" / "chunks"
    matches = sorted(d.name for d in chunks_root.iterdir() if d.is_dir())

    print("=" * 80)
    print("INV #54 — BITSTRING SCAN vs LEGACY SCAN")
    print("=" * 80)

    grand_total_bs = 0
    grand_total_lg = 0
    all_bs_only: list[dict] = []  # Events trouvés par bitstring mais PAS legacy
    all_lg_only: list[dict] = []  # Events trouvés par legacy mais PAS bitstring
    bs_weapons: Counter[str] = Counter()
    lg_weapons: Counter[str] = Counter()
    bs_b3_values: Counter[int] = Counter()
    bs_bit_offsets: Counter[int] = Counter()

    for match_id in matches:
        match_dir = chunks_root / match_id
        chunk_files = sorted(match_dir.glob("chunk_*.bin"))

        print(f"\n{'─' * 60}")
        print(f"Match: {match_id}  ({len(chunk_files)} chunks)")
        print(f"{'─' * 60}")

        match_bs = 0
        match_lg = 0
        match_bs_only = 0
        match_lg_only = 0

        for ck_path in chunk_files:
            ck_idx = int(ck_path.stem.split("_")[1])
            data = ck_path.read_bytes()

            # Estimer start/duration (20s par chunk, départ = index * 20s)
            start_ms = ck_idx * 20000
            duration_ms = 20000

            # Scan bitstring
            bs_events = scan_bitstring(data, 1, start_ms, duration_ms)
            # Scan legacy
            lg_events = scan_legacy(data, 1, start_ms, duration_ms)

            # Clé de comparaison : (fire_counter, weapon_bytes)
            bs_keys = {(e["fire_counter"], e["weapon_bytes"]) for e in bs_events}
            lg_keys = {(e["fire_counter"], e.get("weapon_bytes", b"")) for e in lg_events}

            bs_only = bs_keys - lg_keys
            lg_only = lg_keys - bs_keys

            match_bs += len(bs_events)
            match_lg += len(lg_events)
            match_bs_only += len(bs_only)
            match_lg_only += len(lg_only)

            for ev in bs_events:
                bs_weapons[ev["weapon_name"]] += 1
                bs_b3_values[ev["b3"]] += 1
                bs_bit_offsets[ev["bit_offset"]] += 1

            for ev in lg_events:
                lg_weapons[ev["weapon_name"]] += 1

            # Détail des events uniquement bitstring
            for ev in bs_events:
                key = (ev["fire_counter"], ev["weapon_bytes"])
                if key in bs_only:
                    all_bs_only.append({**ev, "match": match_id, "chunk": ck_idx})

            for ev in lg_events:
                key = (ev["fire_counter"], ev.get("weapon_bytes", b""))
                if key in lg_only:
                    all_lg_only.append({**ev, "match": match_id, "chunk": ck_idx})

            if bs_only or lg_only:
                print(
                    f"  chunk_{ck_idx:02d}: BS={len(bs_events):4d}  LG={len(lg_events):4d}  "
                    f"BS_ONLY={len(bs_only)}  LG_ONLY={len(lg_only)}"
                )

        print(
            f"  TOTAL:     BS={match_bs:4d}  LG={match_lg:4d}  "
            f"BS_ONLY={match_bs_only}  LG_ONLY={match_lg_only}"
        )
        grand_total_bs += match_bs
        grand_total_lg += match_lg

    # ─── Scan élargi : chercher des weapon IDs inconnus ──────────────────────
    print(f"\n{'=' * 80}")
    print("SCAN ÉLARGI — Weapon IDs au-delà de WEAPON_ID_MAP")
    print(f"{'=' * 80}")

    unknown_wids: Counter[int] = Counter()
    unknown_contexts: dict[int, list[dict]] = defaultdict(list)

    for match_id in matches:
        match_dir = chunks_root / match_id
        for ck_path in sorted(match_dir.glob("chunk_*.bin")):
            ck_idx = int(ck_path.stem.split("_")[1])
            data = ck_path.read_bytes()
            bits = Bits(bytes=data)
            marker = _build_marker(1)  # POV

            for position in bits.findall(marker, bytealigned=False):
                event_start = position + 3
                weapon_start = event_start + _WEAPON_OFFSET

                if weapon_start + 64 > len(bits):
                    continue

                weapon_int = bits[weapon_start : weapon_start + 64].uint

                # Skip si déjà connu
                if weapon_int in WEAPON_IDS_INT:
                    continue

                # Skip valeurs triviales (0, max)
                if weapon_int == 0 or weapon_int == (2**64 - 1):
                    continue

                # Vérifier le suffixe 42c9679f (4 derniers bytes)
                wid_bytes = weapon_int.to_bytes(8, byteorder="big")
                has_common_suffix = wid_bytes[4:] == bytes.fromhex("42c9679f")

                # Vérifier b3 pour filtrer le bruit
                b3_start = event_start + 16
                b3 = bits[b3_start : b3_start + 8].uint if b3_start + 8 <= len(bits) else -1

                # Garder les candidats avec suffixe commun OU b3 masqué = 0x40
                if has_common_suffix or (b3 >= 0 and (b3 & 0xFC) == 0x40):
                    unknown_wids[weapon_int] += 1
                    if len(unknown_contexts[weapon_int]) < 3:
                        bit_offset = position % 8
                        fc_start = event_start + 24
                        fire_counter = (
                            bits[fc_start : fc_start + 8].uint if fc_start + 8 <= len(bits) else -1
                        )
                        unknown_contexts[weapon_int].append(
                            {
                                "match": match_id,
                                "chunk": ck_idx,
                                "bit_offset": bit_offset,
                                "b3": b3,
                                "b3_masked": b3 & 0xFC if b3 >= 0 else -1,
                                "fire_counter": fire_counter,
                                "has_suffix": has_common_suffix,
                            }
                        )

    # ─── Rapport final ───────────────────────────────────────────────────────
    print(f"\n{'=' * 80}")
    print("RAPPORT FINAL")
    print(f"{'=' * 80}")

    print(f"\n  Total events bitstring : {grand_total_bs}")
    print(f"  Total events legacy    : {grand_total_lg}")
    print(f"  Delta                  : {grand_total_bs - grand_total_lg:+d}")

    print(f"\n  Events BS-only (trouvés par bitstring, manqués par legacy) : {len(all_bs_only)}")
    if all_bs_only:
        print("\n  Détails BS-only :")
        for ev in all_bs_only[:30]:
            print(
                f"    match={ev['match']} ck={ev['chunk']:02d} "
                f"bit_off={ev['bit_offset']} b3=0x{ev['b3']:02x} "
                f"fc={ev['fire_counter']} {ev['weapon_name']}"
            )
        if len(all_bs_only) > 30:
            print(f"    ... et {len(all_bs_only) - 30} de plus")

    print(f"\n  Events LG-only (trouvés par legacy, manqués par bitstring) : {len(all_lg_only)}")
    if all_lg_only:
        print("\n  Détails LG-only :")
        for ev in all_lg_only[:30]:
            print(f"    match={ev['match']} ck={ev['chunk']:02d} " f"{ev['weapon_name']}")

    print("\n  Bit offset distribution (bitstring) :")
    for off, cnt in sorted(bs_bit_offsets.items()):
        bar = "█" * (cnt // 20)
        print(f"    offset {off}: {cnt:5d}  {bar}")

    print("\n  b3 distribution (bitstring, top 15) :")
    for b3_val, cnt in bs_b3_values.most_common(15):
        masked = f"(masked=0x{b3_val & 0xFC:02x})" if b3_val >= 0 else ""
        print(f"    0x{b3_val:02x} {masked}: {cnt:5d}")

    print("\n  Weapons par méthode :")
    all_weapons = sorted(set(bs_weapons) | set(lg_weapons))
    print(f"    {'Weapon':<30} {'Bitstring':>10} {'Legacy':>10} {'Delta':>10}")
    for w in all_weapons:
        bs_c = bs_weapons.get(w, 0)
        lg_c = lg_weapons.get(w, 0)
        delta = bs_c - lg_c
        flag = " <<<" if delta != 0 else ""
        print(f"    {w:<30} {bs_c:>10} {lg_c:>10} {delta:>+10}{flag}")

    if unknown_wids:
        print(f"\n  WEAPON IDs INCONNUS (candidats, {len(unknown_wids)} uniques) :")
        print(f"    {'Hex ID':<20} {'Count':>6} {'Suffix?':>8} {'b3 vals':>20}")
        for wid_int, cnt in unknown_wids.most_common(30):
            wid_hex = f"{wid_int:016x}"
            ctxs = unknown_contexts[wid_int]
            has_suf = "✓" if ctxs[0]["has_suffix"] else "✗"
            b3s = ", ".join(f"0x{c['b3']:02x}" for c in ctxs[:3])
            print(f"    {wid_hex}  {cnt:>6} {has_suf:>8}  [{b3s}]")
    else:
        print("\n  Aucun weapon ID inconnu trouvé.")


if __name__ == "__main__":
    main()
