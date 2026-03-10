"""
inv110 -- Formula A pi collision audit.

Question:
  Does Formula A pi identify a single player timeline in weapon slots, or can one
  (match, pi, slot) aggregate events from multiple real players?

Collision criterion:
  In a single chunk, a single player inventory slot cannot legitimately expose two
  or more distinct unknown weapon prefixes at once. Such co-presence indicates that
  Formula A pi is an alias bucket rather than a one-player identity.

Output:
  - per (match, pi, slot) collision summaries;
  - per-prefix peer counts inside collision chunks;
  - JSON export for reuse in findings/documentation.
"""

from __future__ import annotations

import json
import sys
import zlib
from collections import Counter, defaultdict
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(PROJECT_ROOT))

CHUNKS_DIR = PROJECT_ROOT / "data/investigation/chunks"
OUTPUT_PATH = PROJECT_ROOT / "scripts/experimental/outputs/inv110_formula_a_pi_collisions.json"

FORMULA_A_MARKER = bytes.fromhex("200002")
SUFFIX = bytes.fromhex("42c9679f")
WEAPON_SLOTS = {0x00, 0x81}

KNOWN_PREFIXES = {
    "0a1992bc",
    "0d20c469",
    "1488d0bb",
    "18e1fea0",
    "230447b1",
    "2ac9c2ff",
    "2b1824d5",
    "2fb21c87",
    "30484ea6",
    "3ad55da4",
    "3e070217",
    "48c19d2d",
    "6683257c",
    "6acdc44d",
    "71ab0a2c",
    "767db96d",
    "80977ba5",
    "84bd29ed",
    "8afc0855",
    "9387a8b9",
    "9d6aaed2",
    "b533957e",
    "b619d84a",
    "b6dbead8",
    "c1e1bab0",
    "c24e549e",
    "c30d87c7",
    "c3542946",
    "d7915565",
    "daf193c7",
    "f408190f",
    "f5c335df",
    "fd98554c",
}


def decompress_chunk(path: Path) -> bytes:
    raw = path.read_bytes()
    try:
        return zlib.decompress(raw[2:], -15)
    except Exception:
        return raw


def scan_formula_a_weapon_events(data: bytes) -> list[tuple[int, int, str]]:
    events: list[tuple[int, int, str]] = []
    index = 0
    while index < len(data) - 20:
        if data[index : index + 3] != FORMULA_A_MARKER:
            index += 1
            continue
        pb = data[index + 3]
        pi = pb >> 5
        window = data[index + 4 : index + 80]
        suffix_pos = window.find(SUFFIX)
        if suffix_pos >= 8:
            wid_start = suffix_pos - 4
            prefix = window[wid_start : wid_start + 4].hex()
            slot_byte = window[wid_start - 1] if wid_start >= 1 else -1
            if slot_byte in WEAPON_SLOTS and prefix not in KNOWN_PREFIXES:
                events.append((pi, slot_byte, prefix))
        index += 1
    return events


def build_collision_rows(corpus_dir: Path) -> list[dict]:
    rows: list[dict] = []
    for match_dir in sorted(corpus_dir.iterdir()):
        if not match_dir.is_dir():
            continue
        match_id = match_dir.name
        chunks = sorted(match_dir.glob("chunk_*.bin"))
        for chunk_index, chunk_path in enumerate(chunks):
            chunk_groups: dict[tuple[int, int], set[str]] = defaultdict(set)
            for pi, slot_byte, prefix in scan_formula_a_weapon_events(decompress_chunk(chunk_path)):
                chunk_groups[(pi, slot_byte)].add(prefix)
            for (pi, slot_byte), prefixes in chunk_groups.items():
                if len(prefixes) < 2:
                    continue
                rows.append(
                    {
                        "match_id": match_id,
                        "chunk_index": chunk_index,
                        "pi": pi,
                        "slot": f"0x{slot_byte:02x}",
                        "prefixes": sorted(prefixes),
                        "n_prefixes": len(prefixes),
                    }
                )
    return rows


def summarize_groups(rows: list[dict]) -> list[dict]:
    grouped: dict[tuple[str, int, str], list[dict]] = defaultdict(list)
    for row in rows:
        grouped[(row["match_id"], row["pi"], row["slot"])].append(row)

    summaries: list[dict] = []
    for (match_id, pi, slot), group_rows in sorted(grouped.items()):
        distinct_prefixes = sorted({prefix for row in group_rows for prefix in row["prefixes"]})
        summaries.append(
            {
                "match_id": match_id,
                "pi": pi,
                "slot": slot,
                "collision_chunks": len(group_rows),
                "max_concurrent_prefixes": max(row["n_prefixes"] for row in group_rows),
                "distinct_prefixes": distinct_prefixes,
                "sample_chunks": group_rows[:5],
            }
        )
    return summaries


def summarize_prefix_peers(rows: list[dict]) -> list[dict]:
    peer_map: dict[str, Counter] = defaultdict(Counter)
    chunk_counts: Counter = Counter()
    for row in rows:
        prefixes = row["prefixes"]
        for prefix in prefixes:
            chunk_counts[prefix] += 1
            for peer in prefixes:
                if peer != prefix:
                    peer_map[prefix][peer] += 1

    summaries: list[dict] = []
    for prefix, peers in sorted(
        peer_map.items(), key=lambda item: (-chunk_counts[item[0]], item[0])
    ):
        summaries.append(
            {
                "prefix": prefix,
                "collision_chunks": chunk_counts[prefix],
                "n_distinct_peers": len(peers),
                "top_peers": dict(peers.most_common(5)),
            }
        )
    return summaries


def print_report(rows: list[dict], groups: list[dict], peers: list[dict]) -> None:
    print("\n" + "=" * 80)
    print("inv110 -- Formula A pi collision audit")
    print("=" * 80)
    print(f"collision chunks: {len(rows)}")
    print(f"collision groups: {len(groups)}")

    print("\nTop collision groups")
    print(f"  {'Match':<10} {'pi':>2} {'slot':>4} {'chunks':>6} {'max':>4} {'prefixes':>8}")
    for rec in sorted(
        groups, key=lambda item: (-item["collision_chunks"], -item["max_concurrent_prefixes"])
    )[:12]:
        print(
            f"  {rec['match_id']:<10} {rec['pi']:>2} {rec['slot']:>4} "
            f"{rec['collision_chunks']:>6} {rec['max_concurrent_prefixes']:>4} "
            f"{len(rec['distinct_prefixes']):>8}"
        )
        for sample in rec["sample_chunks"][:3]:
            print(f"    ck{sample['chunk_index']:02d}: {', '.join(sample['prefixes'])}")

    print("\nTop collision prefixes")
    print(f"  {'Prefix':<12} {'chunks':>6} {'peers':>5}  Top peers")
    for rec in peers[:12]:
        peer_str = ", ".join(f"{peer}x{count}" for peer, count in rec["top_peers"].items())
        print(
            f"  {rec['prefix']:<12} {rec['collision_chunks']:>6} "
            f"{rec['n_distinct_peers']:>5}  {peer_str}"
        )


def export_report(rows: list[dict], groups: list[dict], peers: list[dict]) -> None:
    payload = {
        "investigation": "inv110_formula_a_pi_collision_audit",
        "question": "Does Formula A pi alias multiple players inside weapon slots?",
        "collision_criterion": (
            "Same chunk + same match + same Formula A pi + same slot + at least two "
            "distinct unknown prefixes."
        ),
        "summary": {
            "collision_chunks": len(rows),
            "collision_groups": len(groups),
        },
        "groups": groups,
        "prefix_peers": peers,
    }
    OUTPUT_PATH.parent.mkdir(exist_ok=True)
    OUTPUT_PATH.write_text(json.dumps(payload, indent=2))
    print(f"\nResults saved to {OUTPUT_PATH}")


def main() -> None:
    rows = build_collision_rows(CHUNKS_DIR)
    groups = summarize_groups(rows)
    peers = summarize_prefix_peers(rows)
    print_report(rows, groups, peers)
    export_report(rows, groups, peers)


if __name__ == "__main__":
    main()
