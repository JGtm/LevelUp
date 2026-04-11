"""inv134 -- Attribution player_index via b5 >> 4 (acurtis 2026-03-18).

Fire event structure (nibble-shifted bitstream):
  marker 11b : 0b101 0010 0110  (0d 26, same for ALL players)
  event_start = marker_pos + 3
  b1 [+0..+7]   : 0x26 (constant)
  b2 [+8..+15]  : fire_seq
  b3 [+16..+23] : 0x40 (constant)
  b4 [+24..+31] : fire_counter (wraps at 255)
  b5 [+32..+39] : (player_index << 4) | slot
  weapon [+40..+103] : 8 bytes big-endian

player_index = b5 >> 4
slot = b5 & 0x03  (1=primary, 3=secondary)

Usage:
    python scripts/experimental/inv134_b5_pi_attribution.py
    python scripts/experimental/inv134_b5_pi_attribution.py --match <match_id> --debug
"""
from __future__ import annotations

import argparse
import json
import sys
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

import duckdb
from bitstring import Bits

from src.analysis._weapon_data import WEAPON_ID_MAP, WEAPON_INT_TO_NAME
from src.analysis._weapon_scanners import COMMON_WEAPON_SUFFIX
from src.analysis.packet_index import (
    build_packet_estimator,
    detect_pi_from_metadata,
    extract_metadata_payload,
    index_chunk,
)

CACHE_DIR = ROOT / "data" / "cache" / "film_chunks"
MANIFEST_DIR = ROOT / "data" / "cache" / "film_manifests"
SHARED_DB = ROOT / "data" / "warehouse" / "shared_matches_v2.duckdb"

# Universal marker: b1=0x26 is constant for ALL players (acurtis 2026-03-18)
_MARKER = Bits("0b10100100110")  # 11 bits: 101 + 00100110 (0x26)
_WEAPON_BIT_OFFSET = 40  # bits from event_start to weapon start (8B)
_B5_BIT_OFFSET = 32      # bits from event_start to b5

# Kills with gap > this are flagged as uncertain (grenade/melee/pause)
GAP_THRESHOLD_MS = 500


def load_chunks(match_id: str) -> dict[int, tuple[bytes, int, int]]:
    short = match_id[:8]
    manifest_path = MANIFEST_DIR / f"{short}.json"
    chunks_dir = CACHE_DIR / short
    if not manifest_path.exists() or not chunks_dir.exists():
        return {}
    chunks_meta = {
        c["index"]: c
        for c in json.loads(manifest_path.read_text()).get("chunks", [])
    }
    result = {}
    for f in sorted(chunks_dir.glob("chunk_*.bin")):
        idx = int(f.stem.split("_")[1])
        m = chunks_meta.get(idx)
        if m:
            result[idx] = (f.read_bytes(), m["start_ms"], m["duration_ms"])
    return result


def scan_fire_events_b5(chunk_data: bytes, estimate_ts) -> list[dict]:
    """Scan ALL fire events and extract player_index from b5 >> 4.

    Deduplication by byte position proximity (within 2 bytes = same event).
    Does NOT dedup by (fire_counter, weapon) since fire_counter wraps at 255.
    """
    bits = Bits(bytes=chunk_data)
    total_bits = len(bits)
    events: list[dict] = []

    for position in bits.findall(_MARKER, bytealigned=False):
        event_start = position + 3
        weapon_start = event_start + _WEAPON_BIT_OFFSET
        b5_start = event_start + _B5_BIT_OFFSET

        if weapon_start + 64 > total_bits:
            continue

        b5_int = bits[b5_start : b5_start + 8].uint
        player_index = b5_int >> 4
        slot = b5_int & 0x03

        weapon_int = bits[weapon_start : weapon_start + 64].uint
        weapon_bytes = weapon_int.to_bytes(8, byteorder="big")

        is_known = weapon_int in WEAPON_INT_TO_NAME
        has_common_suffix = weapon_bytes[4:] == COMMON_WEAPON_SUFFIX
        if not is_known and not has_common_suffix:
            continue

        fire_seq = (
            bits[event_start + 8 : event_start + 16].uint
            if event_start + 16 <= total_bits else 0
        )
        fire_counter = (
            bits[event_start + 24 : event_start + 32].uint
            if event_start + 32 <= total_bits else 0
        )

        byte_pos = position // 8
        ts = estimate_ts(byte_pos)
        weapon_name = WEAPON_INT_TO_NAME.get(
            weapon_int,
            WEAPON_ID_MAP.get(weapon_bytes, f"?({weapon_bytes[:4].hex()})"),
        )

        events.append({
            "timestamp_ms": ts,
            "player_index": player_index,
            "slot": slot,
            "b5": b5_int,
            "fire_seq": fire_seq,
            "fire_counter": fire_counter,
            "weapon_name": weapon_name,
            "weapon_bytes": weapon_bytes,
            "byte_pos": byte_pos,
        })

    # Dedup by byte position proximity: two hits within 2 bytes = same event
    events.sort(key=lambda x: x["byte_pos"])
    deduped: list[dict] = []
    last_pos = -999
    for ev in events:
        if ev["byte_pos"] - last_pos > 2:
            deduped.append(ev)
            last_pos = ev["byte_pos"]

    return sorted(deduped, key=lambda x: x["timestamp_ms"])


def build_timing(chunks: dict, chunks_sorted: list[int]) -> list[tuple[float, float]]:
    return [(chunks[c][1], chunks[c][1] + chunks[c][2]) for c in chunks_sorted]


def run(match_id: str, *, debug: bool = False) -> None:
    conn = duckdb.connect(str(SHARED_DB), read_only=True)

    mr = conn.execute(
        "SELECT start_time, playlist_name FROM match_registry WHERE match_id=?",
        [match_id],
    ).fetchone()
    if not mr:
        print(f"Match {match_id} not found.")
        return

    print(f"\nMatch : {match_id}")
    print(f"Date  : {mr[0]}  |  {mr[1]}\n")

    players = conn.execute(
        """SELECT mp.xuid::TEXT, xa.gamertag, mp.kills
           FROM match_participants mp
           LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid::TEXT
           WHERE mp.match_id=? AND mp.xuid NOT LIKE 'bid(%'""",
        [match_id],
    ).fetchall()
    xuid_to_gt = {r[0]: r[1] or r[0][:16] for r in players}
    xuid_int_to_str = {int(r[0]): r[0] for r in players}
    xuid_kills_count = {r[0]: r[2] for r in players}

    chunks = load_chunks(match_id)
    if not chunks:
        print("No cached chunks.")
        return
    chunks_sorted = sorted(chunks)
    print(f"Chunks : {len(chunks)}\n")

    # Detect player_index -> xuid via PLAYER_METADATA
    pi_to_xuid_int: dict[int, int] = {}
    for idx in chunks_sorted:
        data, _, _ = chunks[idx]
        pkts = index_chunk(data)
        payload = extract_metadata_payload(data, pkts)
        if payload:
            pi_to_xuid_int.update(detect_pi_from_metadata(payload, xuid_int_to_str))

    xuid_to_pi: dict[str, int] = {str(v): k for k, v in pi_to_xuid_int.items()}
    for xs in xuid_to_gt:
        if xs not in xuid_to_pi:
            xuid_to_pi[xs] = 0  # POV or unresolved

    print("pi -> player mapping (PLAYER_METADATA):")
    for xs, pi in sorted(xuid_to_pi.items(), key=lambda x: x[1]):
        print(f"  pi={pi}  {xuid_to_gt.get(xs, xs)}")
    print()

    # Scan all fire events (b5 >> 4 = player_index)
    all_events: list[dict] = []
    for idx in chunks_sorted:
        data, start_ms, _ = chunks[idx]
        pkts = index_chunk(data)
        ts_fn = build_packet_estimator(pkts, start_ms)
        all_events.extend(scan_fire_events_b5(data, ts_fn))
    all_events.sort(key=lambda e: e["timestamp_ms"])

    events_by_pi: dict[int, list[dict]] = defaultdict(list)
    for ev in all_events:
        events_by_pi[ev["player_index"]].append(ev)

    print(f"Fire events total : {len(all_events)}")
    print("Distribution by pi (b5>>4):")
    for pi_val in sorted(events_by_pi):
        print(f"  pi={pi_val} : {len(events_by_pi[pi_val])} events")
    print()

    kills_db = conn.execute(
        "SELECT killer_xuid, time_ms FROM killer_victim_pairs "
        "WHERE match_id=? AND time_ms IS NOT NULL ORDER BY time_ms",
        [match_id],
    ).fetchall()
    kills_by_xuid: dict[str, list[float]] = defaultdict(list)
    for k, t in kills_db:
        kills_by_xuid[k].append(float(t))

    print(f"{'Player':<22} {'pi':>3}  {'k':>3}  {'conf':>6}  Weapons (b5>>4 | ? = gap>500ms)")
    print("-" * 90)

    for xuid, gt in sorted(xuid_to_gt.items(), key=lambda x: xuid_to_pi.get(x[0], 99)):
        pi = xuid_to_pi.get(xuid, 0)
        k_total = xuid_kills_count.get(xuid, 0)
        if k_total == 0:
            continue

        my_kill_ts = kills_by_xuid.get(xuid, [])
        pool = list(events_by_pi.get(pi, []))

        weapon_counts: dict[str, int] = defaultdict(int)
        n_confident = 0
        gaps: list[float] = []

        for t_ms in my_kill_ts:
            candidates = [e for e in pool if e["timestamp_ms"] <= t_ms]
            if candidates:
                ev = max(candidates, key=lambda e: e["timestamp_ms"])
                gap = t_ms - ev["timestamp_ms"]
                gaps.append(gap)
                label = ev["weapon_name"] if gap < GAP_THRESHOLD_MS else ev["weapon_name"] + "?"
                weapon_counts[label] += 1
                if gap < GAP_THRESHOLD_MS:
                    n_confident += 1
                pool = [e for e in pool if e is not ev]
                if debug:
                    flag = "" if gap < GAP_THRESHOLD_MS else " [UNCERTAIN]"
                    print(
                        f"  [DBG] {gt} kill@{t_ms:.0f}ms <- ev@{ev['timestamp_ms']:.0f}ms"
                        f" gap={gap:.0f}ms {ev['weapon_name']}{flag}"
                    )
            else:
                weapon_counts["(no_event)"] += 1
                if debug:
                    print(f"  [DBG] {gt} kill@{t_ms:.0f}ms no fire event found")

        avg_gap = f"{sum(gaps)/len(gaps):.0f}ms" if gaps else "--"
        wlist = "  ".join(
            f"{w}x{c}" for w, c in sorted(weapon_counts.items(), key=lambda x: -x[1])
        )
        print(
            f"  {gt:<22} pi={pi}  {k_total:>3}k  {n_confident:>2}/{k_total:<3} conf"
            f"  [avg:{avg_gap}]  {wlist}"
        )

    print()
    print("NOTE: '?' suffix = gap > 500ms (grenade/melee/pause likely, weapon uncertain)")
    print("      (no_event) = no fire event before this kill in loaded chunks")
    print("      conf = kills with gap < 500ms (high confidence attribution)")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--match", default=None)
    parser.add_argument("--debug", action="store_true")
    args = parser.parse_args()

    conn = duckdb.connect(str(SHARED_DB), read_only=True)
    if args.match:
        match_id = args.match
    else:
        cached = sorted(CACHE_DIR.iterdir(), key=lambda p: p.stat().st_mtime, reverse=True)
        if not cached:
            print("No cached chunks.")
            return
        short = cached[0].name
        row = conn.execute(
            "SELECT match_id FROM match_registry WHERE match_id LIKE ? ORDER BY start_time DESC LIMIT 1",
            [f"{short}%"],
        ).fetchone()
        match_id = row[0] if row else short

    run(match_id, debug=args.debug)


if __name__ == "__main__":
    main()
