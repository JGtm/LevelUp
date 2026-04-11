"""
inv43 — Melee events CORRECTED : scan with proper melee structure (not fire event filters)

════════════════════════════════════════════════════════════════════════════════
CONTEXTE — inv42 was WRONG
════════════════════════════════════════════════════════════════════════════════

inv42 applied fire event b1/b3 constraints to melee events:
  - (b1 & 0x1F) == 0x06   → WRONG for melee (acurtis shows b1=0x40)
  - (b3 & 0xFC) == 0x40   → WRONG for melee (acurtis shows b3=0x00)
  - weapon_id at [6:14]   → WRONG offset (acurtis: 11 bytes "Before" + animation + wid)

acurtis (2026-03-06) confirmed melee events:
  Before:  d3 40 13 00 a6 80 26 01 51 00 42
  AnimTyp: d or 5 (0x0d or 0x05 — 1 byte)
  Weapon:  f408190f42c9679f  (= Mk51 Sidekick — weapon HELD during melee)
  After:   0 10 3d a1 xx xx 40 35 f7 c0

Key insight: melee weapon_id = weapon player is HOLDING (not "fist").
  Same weapon IDs as fire events.

Hypothesized melee event structure (30 bytes):
  [0]     = lead byte (0xD3, LSBs=0x03)
  [1]     = 0x40 (NOT pi_byte like fire events)
  [2:11]  = 9 bytes (context / counter / flags)
  [11]    = animation type (0x0d=regular?, 0x05=lunge/backsmack?)
  [12:20] = weapon_id (8 bytes, same IDs as fire events)
  [20:30] = trailing data (includes 0x00 10 3d a1 ...)

Fire event structure comparison (20 bytes):
  [0]    = lead byte (0x0D, LSBs=0x05)
  [1]    = (pi << 5) | 0x06
  [2]    = fire_seq
  [3]    = 0x40-0x43
  [4]    = fire_counter
  [5]    = b5
  [6:14] = weapon_id (8 bytes)

Approach: suffix-first scan (find ALL weapon suffix occurrences, classify by context).

Date : 2026-03-06
"""

from __future__ import annotations

import sys
from collections import Counter, defaultdict
from pathlib import Path

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")

FILM_ROOT = Path(__file__).resolve().parents[2]
CHUNKS_BASE = FILM_ROOT / "data" / "investigation" / "chunks"

COMMON_SUFFIX = bytes.fromhex("42c9679f")
ALT_SUFFIXES = [
    bytes.fromhex("a730e49f"),  # Gravity Hammer alt
    bytes.fromhex("8978aa7a"),  # Energy Sword alt
    bytes.fromhex("e7232c0b"),  # MA5K Avenger
    bytes.fromhex("c8fb11d0"),  # Mythic Sandwich
]
ALL_SUFFIXES = [COMMON_SUFFIX] + ALT_SUFFIXES

WEAPON_IDS: dict[str, str] = {
    "6acdc44d42c9679f": "Bandit Evo",
    "2b1824d542c9679f": "BR75",
    "5eec4e0842c9679f": "BR75 v2",
    "3dc4e30942c9679f": "BR75 v3",
    "230447b142c9679f": "Cindershot",
    "b619d84a42c9679f": "CQS48 Bulldog",
    "84bd29ed42c9679f": "Disruptor",
    "9d6aaed242c9679f": "Fuel Rod SPNKr",
    "2ac9c2ff42c9679f": "Heatwave",
    "71ab0a2c42c9679f": "M41 SPNKr",
    "2fb21c8742c9679f": "M392 Bandit",
    "48c19d2d42c9679f": "MA40 AR",
    "0daf3b0042c9679f": "MA40 AR v2",
    "80977ba542c9679f": "Mangler",
    "7deb133f42c9679f": "Mangler v2",
    "767db96d42c9679f": "MLRS-2 Hydra",
    "5ded6cf242c9679f": "MLRS-2 Hydra v2",
    "f408190f42c9679f": "Mk51 Sidekick",
    "d791556542c9679f": "Mutilator",
    "b533957e42c9679f": "Needler",
    "c354294642c9679f": "Plasma Pistol",
    "30484ea642c9679f": "Pulse Carbine",
    "c30d87c742c9679f": "Ravager",
    "0a1992bc42c9679f": "S7 Sniper",
    "9387a8b942c9679f": "Shock Rifle",
    "0d20c46942c9679f": "Skewer",
    "daf193c742c9679f": "Stalker Rifle",
    "fd98554c42c9679f": "VK78 Commando",
    "3e07021742c9679f": "Vestige Carbine",
    "c24e549e42c9679f": "Sentinel Beam",
    "a0955e9e42c9679f": "Sentinel Beam v2",
    "8afc085542c9679f": "Gravity Hammer",
    "841ac5e5a730e49f": "Gravity Hammer (alt)",
    "1488d0bb42c9679f": "Energy Sword",
    "4ff3937e8978aa7a": "Energy Sword (alt)",
    "67edd20142c9679f": "M392 Bandit v2",
    "b6dbead842c9679f": "M9 Frag Grenade",
    "c1e1bab042c9679f": "Plasma Grenade",
    "ab879f1e42c9679f": "Spike Grenade",
    "e3a0a51842c9679f": "Dynamo Grenade",
    "7a9e791042c9679f": "Plasma Grenade v2",
    "6683257c42c9679f": "Spike Grenade v2",
    "3ad55da442c9679f": "Dynamo Grenade v2",
    "6d32c7dc42c9679f": "Scattershot",
    "f5c335dfe7232c0b": "MA5K Avenger",
}

MATCH_SHORTS = ["d9329229", "00162144", "63d6f727"]


def section(title: str) -> None:
    print(f"\n{'=' * 76}\n  {title}\n{'=' * 76}")


def _nibble_shift(data: bytes) -> bytes:
    out = bytearray(len(data) - 1)
    for i in range(len(out)):
        out[i] = ((data[i] << 4) & 0xFF) | (data[i + 1] >> 4)
    return bytes(out)


def _wid_name(wid_bytes: bytes) -> str:
    return WEAPON_IDS.get(wid_bytes.hex(), f"?{wid_bytes[:4].hex()}[{wid_bytes[4:].hex()}]")


def _has_any_suffix(wid_bytes: bytes) -> bool:
    return any(wid_bytes[4:] == s for s in ALL_SUFFIXES)


def load_chunks(match_short: str) -> list[tuple[int, bytes]]:
    d = CHUNKS_BASE / match_short
    if not d.exists():
        return []
    result = []
    for p in sorted(d.glob("chunk_*.bin")):
        try:
            idx = int(p.stem.split("_")[1])
            result.append((idx, p.read_bytes()))
        except (ValueError, IndexError):
            pass
    return result


# ═══════════════════════════════════════════════════════════════════════════
# PHASE 1 : Suffix-first scan
#   Find ALL occurrences of weapon suffix (42c9679f etc.) in nibble-shifted
#   layer. For each hit, look backwards to determine event type (fire vs melee).
# ═══════════════════════════════════════════════════════════════════════════


def phase1_suffix_first(match_short: str) -> None:
    section(f"PHASE 1 — Suffix-first scan ({match_short})")

    chunks = load_chunks(match_short)
    if not chunks:
        print(f"  No chunks for {match_short}")
        return

    print(f"  {len(chunks)} chunks loaded")

    for layer_name, do_shift in [("nibble-shifted", True), ("raw", False)]:
        # Find all suffix occurrences
        fire_count = 0
        melee_count = 0
        other_count = 0
        classifications: Counter[str] = Counter()

        for ck_idx, chunk_data in chunks:
            view = _nibble_shift(chunk_data) if do_shift else chunk_data
            vlen = len(view)

            for suffix in ALL_SUFFIXES:
                pos = 0
                while pos < vlen:
                    idx = view.find(suffix, pos)
                    if idx == -1:
                        break
                    # suffix starts at idx, so weapon_id starts at idx-4
                    wid_start = idx - 4
                    if wid_start < 0:
                        pos = idx + 1
                        continue

                    wid = bytes(view[wid_start : wid_start + 8])

                    # Only count if it's a known weapon ID
                    if wid.hex() not in WEAPON_IDS:
                        pos = idx + 1
                        continue

                    # Classify: look back for fire event lead byte or melee lead byte
                    # Fire: weapon_id at offset [6:14] → lead byte at wid_start - 6
                    # Melee: weapon_id at offset [12:20] → lead byte at wid_start - 12
                    classification = "unknown"

                    # Check fire event pattern (lead at -6)
                    fire_lead = wid_start - 6
                    if fire_lead >= 0:
                        b0_fire = view[fire_lead]
                        b1_fire = view[fire_lead + 1]
                        b3_fire = view[fire_lead + 3]
                        if (
                            (b0_fire & 0x07) == 0x05
                            and (b1_fire & 0x1F) == 0x06
                            and (b3_fire & 0xFC) == 0x40
                        ):
                            classification = "fire"
                            fire_count += 1

                    # Check melee event pattern (lead at -12)
                    melee_lead = wid_start - 12
                    if classification == "unknown" and melee_lead >= 0:
                        b0_m = view[melee_lead]
                        b1_m = view[melee_lead + 1]
                        if (b0_m & 0x07) == 0x03 and b1_m == 0x40:
                            classification = "melee"
                            melee_count += 1

                    if classification == "unknown":
                        other_count += 1

                    classifications[classification] += 1
                    pos = idx + 1

        print(f"\n  [{layer_name}] Total suffix hits (known weapon IDs):")
        print(f"    Fire events:  {fire_count}")
        print(f"    Melee events: {melee_count}")
        print(f"    Other/unknown: {other_count}")
        for cls, cnt in classifications.most_common():
            print(f"    → {cls}: {cnt}")


# ═══════════════════════════════════════════════════════════════════════════
# PHASE 2 : Pattern-first melee scan (d3 40 ... weapon suffix nearby)
#   Instead of using fire event b1/b3 constraints, use melee-specific pattern:
#   b0=0xD3, b1=0x40 — then scan ahead for weapon suffix.
# ═══════════════════════════════════════════════════════════════════════════


def phase2_pattern_first(match_short: str) -> None:
    section(f"PHASE 2 — Pattern-first melee d3+40 ({match_short})")

    chunks = load_chunks(match_short)
    if not chunks:
        print(f"  No chunks for {match_short}")
        return

    for layer_name, do_shift in [("nibble-shifted", True), ("raw", False)]:
        events: list[dict] = []

        for ck_idx, chunk_data in chunks:
            view = _nibble_shift(chunk_data) if do_shift else chunk_data
            vlen = len(view)

            for i in range(vlen - 30):
                b0 = view[i]
                if (b0 & 0x07) != 0x03:
                    continue
                b1 = view[i + 1]
                if b1 != 0x40:
                    continue

                # Potential melee event. Look for weapon suffix at various offsets.
                # acurtis shows weapon_id at offset 12 from lead byte.
                # Test offsets 10-18 to be sure.
                for wid_offset in range(10, 20):
                    wid_end = i + wid_offset + 8
                    if wid_end > vlen:
                        break
                    wid = bytes(view[i + wid_offset : wid_end])
                    if _has_any_suffix(wid) and wid.hex() in WEAPON_IDS:
                        events.append(
                            {
                                "chunk": ck_idx,
                                "pos": i,
                                "b0": b0,
                                "b1": b1,
                                "wid_offset": wid_offset,
                                "weapon": _wid_name(wid),
                                "wid_hex": wid.hex(),
                                "before": bytes(view[i : i + wid_offset]).hex(" "),
                                "after_wid": bytes(view[wid_end : wid_end + 10]).hex(" ")
                                if wid_end + 10 <= vlen
                                else "N/A",
                                "anim_byte": f"0x{view[i + wid_offset - 1]:02x}"
                                if wid_offset > 0
                                else "N/A",
                            }
                        )

        print(f"\n  [{layer_name}] Melee events (b0&0x07==0x03, b1==0x40, known wid nearby):")
        print(f"    Total: {len(events)}")

        if events:
            # Group by wid_offset to confirm structure
            by_offset: Counter[int] = Counter()
            for ev in events:
                by_offset[ev["wid_offset"]] += 1
            print("    Weapon ID offset distribution:")
            for off, cnt in by_offset.most_common():
                print(f"      offset {off}: {cnt} events")

            # Show first 15 events
            print(f"\n    First {min(15, len(events))} events:")
            for ev in events[:15]:
                print(
                    f"      ck={ev['chunk']:2d}  pos={ev['pos']:8d}"
                    f"  b0=0x{ev['b0']:02x}  wid_off={ev['wid_offset']}"
                    f"  anim={ev['anim_byte']}"
                    f"  weapon={ev['weapon']}"
                )
                print(f"        before: {ev['before']}")
                print(f"        after:  {ev['after_wid']}")

            # Group by weapon
            by_weapon: Counter[str] = Counter()
            for ev in events:
                by_weapon[ev["weapon"]] += 1
            print("\n    Weapons held during melee:")
            for w, cnt in by_weapon.most_common():
                print(f"      {w}: {cnt}")

            # Group by animation byte value
            by_anim: Counter[str] = Counter()
            for ev in events:
                by_anim[ev["anim_byte"]] += 1
            print("\n    Animation type distribution:")
            for a, cnt in by_anim.most_common():
                print(f"      {a}: {cnt}")

            # Group by b0 value
            by_b0: Counter[int] = Counter()
            for ev in events:
                by_b0[ev["b0"]] += 1
            print("\n    Lead byte (b0) distribution:")
            for b0v, cnt in by_b0.most_common():
                print(f"      0x{b0v:02x} (bin={b0v:08b}): {cnt}")


# ═══════════════════════════════════════════════════════════════════════════
# PHASE 3 : Relaxed lead byte scan for melee
#   Like phase 2 but with relaxed lead byte: any (b0 & 0x07) == 0x03
#   and relaxed b1: any value (not just 0x40)
#   Goal: discover if non-POV melee events exist with different b1 values.
# ═══════════════════════════════════════════════════════════════════════════


def phase3_relaxed_melee(match_short: str) -> None:
    section(f"PHASE 3 — Relaxed melee (b0&0x07==0x03, any b1) ({match_short})")

    chunks = load_chunks(match_short)
    if not chunks:
        print(f"  No chunks for {match_short}")
        return

    # Only nibble-shifted (the productive layer for fire events)
    events: list[dict] = []
    b1_dist: Counter[int] = Counter()

    for ck_idx, chunk_data in chunks:
        view = _nibble_shift(chunk_data)
        vlen = len(view)

        for i in range(vlen - 30):
            b0 = view[i]
            if (b0 & 0x07) != 0x03:
                continue

            b1 = view[i + 1]
            b1_dist[b1] += 1

            # Check for weapon suffix at offset 12 (primary) and nearby
            found_wid = False
            for wid_offset in [12, 11, 13, 14, 10, 15]:
                wid_end = i + wid_offset + 8
                if wid_end > vlen:
                    break
                wid = bytes(view[i + wid_offset : wid_end])
                if _has_any_suffix(wid) and wid.hex() in WEAPON_IDS:
                    events.append(
                        {
                            "chunk": ck_idx,
                            "pos": i,
                            "b0": b0,
                            "b1": b1,
                            "wid_offset": wid_offset,
                            "weapon": _wid_name(wid),
                            "anim_byte": f"0x{view[i + wid_offset - 1]:02x}",
                        }
                    )
                    found_wid = True
                    break  # take first match

    print("  [nibble-shifted] Total melee positions scanned: many")
    print(f"  Events with known weapon nearby: {len(events)}")

    if events:
        # Distribution of b1 among events with valid weapon IDs
        b1_events: Counter[int] = Counter()
        for ev in events:
            b1_events[ev["b1"]] += 1
        print("\n  b1 distribution (events WITH valid weapon):")
        for b1v, cnt in b1_events.most_common(15):
            print(f"    b1=0x{b1v:02x}: {cnt}  (pi_if_fire={b1v >> 5})")

        # Check if non-0x40 b1 values might encode player_index
        non_40 = [ev for ev in events if ev["b1"] != 0x40]
        print(f"\n  Events with b1 != 0x40: {len(non_40)}")
        if non_40:
            print("  First 10 non-0x40 events:")
            for ev in non_40[:10]:
                print(
                    f"    ck={ev['chunk']:2d} pos={ev['pos']:8d}"
                    f" b0=0x{ev['b0']:02x} b1=0x{ev['b1']:02x}"
                    f" weapon={ev['weapon']} anim={ev['anim_byte']}"
                )


# ═══════════════════════════════════════════════════════════════════════════
# PHASE 4 : Melee anatomy — full hex dump of all melee events
#   Dump 40 bytes around each confirmed melee event for manual analysis.
# ═══════════════════════════════════════════════════════════════════════════


def phase4_anatomy(match_short: str) -> None:
    section(f"PHASE 4 — Melee anatomy full dump ({match_short})")

    chunks = load_chunks(match_short)
    if not chunks:
        print(f"  No chunks for {match_short}")
        return

    events: list[dict] = []

    for ck_idx, chunk_data in chunks:
        view = _nibble_shift(chunk_data)
        vlen = len(view)

        for i in range(vlen - 30):
            b0 = view[i]
            if (b0 & 0x07) != 0x03:
                continue
            b1 = view[i + 1]
            if b1 != 0x40:
                continue

            # Look for weapon at offset 12
            wid_end = i + 20
            if wid_end > vlen:
                continue
            wid = bytes(view[i + 12 : wid_end])
            if not (_has_any_suffix(wid) and wid.hex() in WEAPON_IDS):
                continue

            # Dump 40 bytes from lead byte
            end_dump = min(i + 40, vlen)
            raw_dump = bytes(view[i:end_dump])

            events.append(
                {
                    "chunk": ck_idx,
                    "pos": i,
                    "weapon": _wid_name(wid),
                    "dump": raw_dump,
                }
            )

    print(f"  Melee events found: {len(events)}")
    print()

    # Deduplicate close positions (within same chunk, within 5 bytes)
    deduped: list[dict] = []
    for ev in events:
        if (
            not deduped
            or ev["chunk"] != deduped[-1]["chunk"]
            or abs(ev["pos"] - deduped[-1]["pos"]) > 5
        ):
            deduped.append(ev)
    print(f"  After dedup (±5 bytes): {len(deduped)}")

    for idx_ev, ev in enumerate(deduped[:30]):
        d = ev["dump"]
        print(
            f"\n  Event #{idx_ev + 1}: chunk={ev['chunk']}  pos={ev['pos']}  weapon={ev['weapon']}"
        )
        # Show bytes with markers
        hex_str = d.hex(" ")
        print(f"    Full: {hex_str}")
        # Annotate structure
        print(f"    [0]    lead:  0x{d[0]:02x} (LSBs={d[0] & 0x07:03b})")
        print(f"    [1]    b1:    0x{d[1]:02x}")
        print(f"    [2:11] body:  {d[2:11].hex(' ')}")
        print(f"    [11]   anim:  0x{d[11]:02x}")
        print(f"    [12:20] wid:  {d[12:20].hex(' ')}  ({ev['weapon']})")
        if len(d) >= 30:
            print(f"    [20:30] after: {d[20:30].hex(' ')}")
        if len(d) >= 40:
            print(f"    [30:40] trail: {d[30:40].hex(' ')}")


# ═══════════════════════════════════════════════════════════════════════════
# PHASE 5 : Fire event burst / hit-miss bits (acurtis feedback)
#   "2nd bit following weapon ID is 0 for mid-burst, 1 for final shot"
#   "3rd bit following weapon ID is hit/miss (0=hit, 1=miss)"
#   Fire event: wid at [6:14], so burst bit = bit 1 of byte[14],
#   hit/miss bit = bit 2 of byte[14]? Or individual bits in byte[14]?
# ═══════════════════════════════════════════════════════════════════════════


def phase5_fire_bits(match_short: str) -> None:
    section(f"PHASE 5 — Fire event burst/hit-miss bits ({match_short})")

    chunks = load_chunks(match_short)
    if not chunks:
        print(f"  No chunks for {match_short}")
        return

    # Scan fire events and analyze bytes after weapon_id
    events_by_weapon: dict[str, list[dict]] = defaultdict(list)

    for ck_idx, chunk_data in chunks:
        view = _nibble_shift(chunk_data)
        vlen = len(view)
        seen: set[tuple[str, int]] = set()

        for i in range(vlen - 20):
            b0 = view[i]
            if (b0 & 0x07) != 0x05:
                continue
            b1 = view[i + 1]
            if (b1 & 0x1F) != 0x06:
                continue
            if (view[i + 3] & 0xFC) != 0x40:
                continue

            # Valid fire event
            wid_start = i + 6
            wid_end = i + 14
            if wid_end + 4 > vlen:
                continue
            wid = bytes(view[wid_start:wid_end])
            if not (_has_any_suffix(wid) and wid.hex() in WEAPON_IDS):
                continue

            pi = (b1 >> 5) & 0x07
            fc = view[i + 4]
            key = (wid.hex(), fc)
            if key in seen:
                continue
            seen.add(key)

            # Bytes after weapon_id
            after_wid = bytes(view[wid_end : wid_end + 4])
            # "2nd bit following weapon ID" = bit 1 of byte[14]
            # "3rd bit following weapon ID" = bit 2 of byte[14]
            byte14 = view[wid_end]
            burst_bit = (byte14 >> 1) & 1  # bit 1
            hit_miss_bit = (byte14 >> 2) & 1  # bit 2

            # Alternative interpretation: bits are individual bytes/nibbles
            # "2nd bit" could mean the 2nd BIT of the first byte after wid
            # "3rd bit" could mean the 3rd BIT

            wname = _wid_name(wid)
            events_by_weapon[wname].append(
                {
                    "chunk": ck_idx,
                    "fc": fc,
                    "byte14": byte14,
                    "burst_bit": burst_bit,
                    "hit_miss_bit": hit_miss_bit,
                    "after_hex": after_wid.hex(" "),
                    "after_bits": f"{byte14:08b}",
                }
            )

    # Analyze burst patterns for BR (3-round burst → expect 0-0-1)
    for weapon_name in ["BR75", "BR75 v2", "BR75 v3", "Shock Rifle"]:
        evts = events_by_weapon.get(weapon_name, [])
        if not evts:
            continue
        print(f"\n  {weapon_name}: {len(evts)} fire events")

        # Show burst_bit sequences (consecutive fire_counter values)
        evts_sorted = sorted(evts, key=lambda e: (e["chunk"], e["fc"]))
        burst_seqs: list[str] = []
        for ev in evts_sorted[:30]:
            burst_seqs.append(
                f"    fc={ev['fc']:3d}  byte14=0x{ev['byte14']:02x}"
                f"  bits={ev['after_bits']}"
                f"  burst_bit={ev['burst_bit']}"
                f"  hit/miss={ev['hit_miss_bit']}"
                f"  after={ev['after_hex']}"
            )
        for s in burst_seqs:
            print(s)

        # Check bit 1 pattern: for BR, should see 0,0,1 repeating
        bit1_pattern = [ev["burst_bit"] for ev in evts_sorted]
        runs_of_001 = sum(
            1 for j in range(0, len(bit1_pattern) - 2, 3) if bit1_pattern[j : j + 3] == [0, 0, 1]
        )
        total_triplets = (len(bit1_pattern)) // 3
        print(f"    0-0-1 burst pattern (bit 1): {runs_of_001}/{total_triplets} triplets")

        # Try alternative: maybe "2nd bit" means bit 6 (second from MSB)?
        # Or the 2nd byte after wid?
        # Let's also check byte 15 (2nd byte after wid)
        if len(evts_sorted) > 0 and len(evts_sorted[0]["after_hex"]) > 5:
            print("\n    Alt interpretation: 2nd/3rd BYTE after weapon ID:")
            for ev in evts_sorted[:15]:
                after_bytes = bytes.fromhex(ev["after_hex"].replace(" ", ""))
                if len(after_bytes) >= 3:
                    print(
                        f"    fc={ev['fc']:3d}"
                        f"  byte[14]=0x{after_bytes[0]:02x}({after_bytes[0]:08b})"
                        f"  byte[15]=0x{after_bytes[1]:02x}({after_bytes[1]:08b})"
                        f"  byte[16]=0x{after_bytes[2]:02x}({after_bytes[2]:08b})"
                    )

    # Summary of all weapons
    print("\n  All weapons — burst_bit and hit/miss distribution:")
    for wname, evts in sorted(events_by_weapon.items()):
        burst_0 = sum(1 for e in evts if e["burst_bit"] == 0)
        burst_1 = sum(1 for e in evts if e["burst_bit"] == 1)
        hit_0 = sum(1 for e in evts if e["hit_miss_bit"] == 0)
        hit_1 = sum(1 for e in evts if e["hit_miss_bit"] == 1)
        print(
            f"    {wname:25s}  events={len(evts):4d}"
            f"  burst(0/1)={burst_0}/{burst_1}"
            f"  hit/miss(0/1)={hit_0}/{hit_1}"
        )


# ═══════════════════════════════════════════════════════════════════════════
# MAIN
# ═══════════════════════════════════════════════════════════════════════════


def main() -> None:
    print("inv43 — Melee events CORRECTED + Fire event burst/hit-miss bits")
    print(f"Matches: {', '.join(MATCH_SHORTS)}")

    # Phase 1: suffix-first (all matches)
    for m in MATCH_SHORTS:
        phase1_suffix_first(m)

    # Phase 2: pattern-first melee (all matches)
    for m in MATCH_SHORTS:
        phase2_pattern_first(m)

    # Phase 3: relaxed melee (first match for speed)
    phase3_relaxed_melee(MATCH_SHORTS[0])

    # Phase 4: anatomy dump (first match)
    phase4_anatomy(MATCH_SHORTS[0])

    # Phase 5: fire event bits (first match)
    phase5_fire_bits(MATCH_SHORTS[0])

    print("\n" + "=" * 76)
    print("  inv43 DONE")
    print("=" * 76)


if __name__ == "__main__":
    main()
