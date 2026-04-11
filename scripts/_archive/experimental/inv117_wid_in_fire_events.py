"""
inv117 -- Full-wid presence in fire/kill events (non-Formula-A).

Question:
  Does the wid 6d32c7dc42c9679f (and other unknown primary weapons) appear
  in binary contexts OUTSIDE Formula A (i.e., in fire or kill events)?

  Formula A events are teammate weapon snapshots. Kill/fire events (Formula C
  and POV fire events) use the weapon's wid to record what fired the shot.
  If the two systems use the SAME wid (skin wid), then 6d32c7dc is the
  "effective" weapon id the game uses for kills.
  If kill events use the BASE weapon wid, then 6d32c7dc is skin-only.

Method:
  For each match, scan all decompressed chunks for the full 8-byte sequence
  of each top unknown. Record:
    - byte offset of each occurrence
    - the 8 bytes immediately BEFORE the wid (context prefix)
    - the 4 bytes immediately AFTER the wid (context suffix)
  Then classify each occurrence:
    - Formula A context: preceded by [20 00 02][pb] ... (marker within 80 bytes back)
    - Formula C context: preceded by [0D] or [0E] (fire event markers)
    - Unknown context: everything else

Output:
  For each wid: total occurrences, split by context type, sample non-Formula-A hits.
"""

from __future__ import annotations

import json
import sys
import zlib
from collections import Counter
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(PROJECT_ROOT))

CHUNKS_DIR = PROJECT_ROOT / "data/investigation/chunks"

# Wids to search — full 8-byte sequences
TARGET_WIDS: dict[str, str] = {
    "6d32c7dc42c9679f": "UNK_PRIMARY_0x00 (top)",  # pragma: allowlist secret
    "f55c4bd2 42c9679f".replace(" ", ""): "UNK_PRIMARY_0x00 #2",  # pragma: allowlist secret
    "0131ea10 42c9679f".replace(" ", ""): "UNK_SECONDARY_0x81 (top)",  # pragma: allowlist secret
    "d48d9b84 42c9679f".replace(" ", ""): "UNK_GRENADE_0x01 (top)",  # pragma: allowlist secret
    "67fed82c 42c9679f".replace(" ", ""): "UNK_SPIKE_SKIN #1",  # pragma: allowlist secret
    # Known reference weapons for comparison
    "48c19d2d42c9679f": "MA40 AR (known)",  # pragma: allowlist secret
    "2b1824d542c9679f": "BR75 (known)",  # pragma: allowlist secret
    "f408190f42c9679f": "Mk51 Sidekick (known)",  # pragma: allowlist secret
}

FORMULA_A_MARKER = bytes.fromhex("200002")

# Fire event markers from inv1-22 (POV fire events)
FIRE_MARKER_0D = bytes([0x0D])
FIRE_MARKER_0E = bytes([0x0E])


def decompress_chunk(path: Path) -> bytes:
    raw = path.read_bytes()
    try:
        return zlib.decompress(raw[2:], -15)
    except Exception:
        return raw


def classify_context(data: bytes, pos: int) -> str:
    """Classify the context of a wid occurrence at byte position pos."""
    # Check backward for Formula A marker within 80 bytes
    look_back = max(0, pos - 80)
    window_back = data[look_back:pos]
    if FORMULA_A_MARKER in window_back:
        return "FORMULA_A"
    # Check byte immediately before the 4-byte prefix (pos - 4 - 1)
    pre4 = pos - 4  # start of the 4-byte prefix
    if pre4 > 0:
        byte_before = data[pre4 - 1]
        if byte_before in (0x0D, 0x0E, 0x4D, 0x4E):
            return "FIRE_EVENT"
    return "OTHER"


def scan_match(match_dir: Path, wid_bytes: bytes) -> list[dict]:
    """Scan all chunks of a match for a given wid, return hit records."""
    hits = []
    for ck_idx, chunk_path in enumerate(sorted(match_dir.glob("chunk_*.bin"))):
        data = decompress_chunk(chunk_path)
        pos = 0
        while True:
            idx = data.find(wid_bytes, pos)
            if idx == -1:
                break
            context = classify_context(data, idx)
            pre_bytes = data[max(0, idx - 8) : idx].hex()
            post_bytes = data[idx + 8 : idx + 12].hex()
            hits.append(
                {
                    "chunk": ck_idx,
                    "offset": idx,
                    "context": context,
                    "pre_bytes": pre_bytes,
                    "post_bytes": post_bytes,
                }
            )
            pos = idx + 8
    return hits


def build_full_report(corpus_dir: Path) -> dict:
    """Scan entire corpus for each target wid."""
    results = {}
    for wid_hex, label in TARGET_WIDS.items():
        wid_bytes = bytes.fromhex(wid_hex)
        all_hits: list[dict] = []
        match_counts: dict[str, Counter] = {}
        for match_dir in sorted(corpus_dir.iterdir()):
            if not match_dir.is_dir():
                continue
            hits = scan_match(match_dir, wid_bytes)
            if hits:
                ctx_counter = Counter(h["context"] for h in hits)
                match_counts[match_dir.name] = dict(ctx_counter)
                all_hits.extend({**h, "match_id": match_dir.name} for h in hits)
        total = len(all_hits)
        by_context = Counter(h["context"] for h in all_hits)
        non_formula_a = [h for h in all_hits if h["context"] != "FORMULA_A"]
        results[wid_hex] = {
            "label": label,
            "total_hits": total,
            "by_context": dict(by_context),
            "match_counts": match_counts,
            "non_formula_a_sample": non_formula_a[:20],
        }
    return results


def print_report(report: dict) -> None:
    print("\n" + "=" * 80)
    print("inv117 -- Full-wid presence in fire/kill events")
    print("=" * 80)

    for wid_hex, rec in report.items():
        print(f"\n  {wid_hex}  [{rec['label']}]")  # pragma: allowlist secret
        print(f"    Total hits: {rec['total_hits']}")
        for ctx, cnt in sorted(rec["by_context"].items()):
            print(f"      {ctx:<15}  {cnt}")
        non_fa = rec["non_formula_a_sample"]
        if non_fa:
            print(f"    Non-Formula-A hits ({len(non_fa)} shown):")
            for h in non_fa[:5]:
                print(
                    f"      match={h['match_id']}  ck={h['chunk']:>3}  "
                    f"offset={h['offset']:>6}  "
                    f"pre=[{h['pre_bytes']}]  post=[{h['post_bytes']}]"
                )
        else:
            print("    No non-Formula-A hits.")

    # Summary comparison: known vs unknown
    print("\n\n=== CONTEXT DISTRIBUTION COMPARISON ===\n")
    print(
        f"  {'Wid prefix':<12}  {'Label':<32}  {'FORMULA_A':>10}  {'FIRE_EVENT':>10}  {'OTHER':>10}"
    )
    print(f"  {'-' * 12}  {'-' * 32}  {'-' * 10}  {'-' * 10}  {'-' * 10}")
    for wid_hex, rec in report.items():
        prefix = wid_hex[:8]
        ctx = rec["by_context"]
        print(
            f"  {prefix}  {rec['label']:<32}  "
            f"{ctx.get('FORMULA_A', 0):>10}  "
            f"{ctx.get('FIRE_EVENT', 0):>10}  "
            f"{ctx.get('OTHER', 0):>10}"
        )


def main() -> None:
    print("inv117 -- Full-wid presence in fire/kill events")
    print("Scanning corpus for target wids...")

    report = build_full_report(CHUNKS_DIR)

    print_report(report)

    out_path = PROJECT_ROOT / "scripts/experimental/outputs/inv117_wid_contexts.json"
    out_path.parent.mkdir(exist_ok=True)
    # Drop large non_formula_a_sample from JSON
    export = {
        k: {kk: vv for kk, vv in v.items() if kk != "non_formula_a_sample"}
        for k, v in report.items()
    }
    out_path.write_text(json.dumps(export, indent=2))
    print(f"\nResults saved to {out_path}")


if __name__ == "__main__":
    main()
