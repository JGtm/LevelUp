"""
inv106 — Formula A event byte structure analysis.

Goal: Find structural bytes in the Formula A event window that encode weapon
slot, type, or category — allowing classification of unknown wids.

Formula A structure (known):
  [20 00 02] [pb: pi<<5 | ?] [??? bytes] [wid: 8B] [42c9679f]

The window between pb and wid varies in length (suf_pos ranges from 8 to ~30+).
Questions:
  1. Is there a fixed-offset byte that encodes weapon SLOT (primary/secondary/equipment)?
  2. Are there structural bytes that differ between KNOWN weapon categories
     (rifle vs launcher vs grenade) that would let us classify unknowns?
  3. What is the exact event structure — are there always fixed bytes at offsets
     relative to the wid?

Method:
  - For KNOWN weapons: extract full event bytes from Formula A marker to end of wid
  - Group events by weapon name
  - For each offset relative to wid start (e.g. wid_start - 1, wid_start - 2, ...),
    compute the distribution of byte values
  - Find offsets where value is CONSISTENT within a weapon group but DIFFERS
    between groups → these are discriminating structural bytes
  - Then apply those byte patterns to UNKNOWN wids to classify them

Focus match: d9329229 (where we have ground truth pi->player).
"""

from __future__ import annotations

import sys
import zlib
from collections import Counter, defaultdict
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(PROJECT_ROOT))

CHUNKS_DIR = PROJECT_ROOT / "data/investigation/chunks"

KNOWN_WIDS: dict[str, str] = {
    "6acdc44d42c9679f": "Bandit Evo",  # pragma: allowlist secret
    "2b1824d542c9679f": "BR75",  # pragma: allowlist secret
    "230447b142c9679f": "Cindershot",  # pragma: allowlist secret
    "b619d84a42c9679f": "Bulldog",  # pragma: allowlist secret
    "84bd29ed42c9679f": "Disruptor",  # pragma: allowlist secret
    "9d6aaed242c9679f": "Fuel Rod SPNKr",  # pragma: allowlist secret
    "2ac9c2ff42c9679f": "Heatwave",  # pragma: allowlist secret
    "71ab0a2c42c9679f": "M41 SPNKr",  # pragma: allowlist secret
    "2fb21c8742c9679f": "M392 Bandit",  # pragma: allowlist secret
    "48c19d2d42c9679f": "MA40 AR",  # pragma: allowlist secret
    "80977ba542c9679f": "Mangler",  # pragma: allowlist secret
    "767db96d42c9679f": "MLRS-2 Hydra",  # pragma: allowlist secret
    "f408190f42c9679f": "Mk51 Sidekick",  # pragma: allowlist secret
    "d791556542c9679f": "Mutilator",  # pragma: allowlist secret
    "b533957e42c9679f": "Needler",  # pragma: allowlist secret
    "c354294642c9679f": "Plasma Pistol",  # pragma: allowlist secret
    "30484ea642c9679f": "Pulse Carbine",  # pragma: allowlist secret
    "c30d87c742c9679f": "Ravager",  # pragma: allowlist secret
    "0a1992bc42c9679f": "S7 Sniper",  # pragma: allowlist secret
    "9387a8b942c9679f": "Shock Rifle",  # pragma: allowlist secret
    "0d20c46942c9679f": "Skewer",  # pragma: allowlist secret
    "daf193c742c9679f": "Stalker Rifle",  # pragma: allowlist secret
    "fd98554c42c9679f": "VK78 Commando",  # pragma: allowlist secret
    "3e07021742c9679f": "Vestige",  # pragma: allowlist secret
    "c24e549e42c9679f": "Sentinel Beam",  # pragma: allowlist secret
    "8afc085542c9679f": "Gravity Hammer",  # pragma: allowlist secret
    "1488d0bb42c9679f": "Energy Sword",  # pragma: allowlist secret
    "b6dbead842c9679f": "M9 Frag",  # pragma: allowlist secret
    "c1e1bab042c9679f": "Plasma Grenade",  # pragma: allowlist secret
    "6683257c42c9679f": "Spike Grenade",  # pragma: allowlist secret
    "3ad55da442c9679f": "Dynamo (hand)",  # pragma: allowlist secret
    "18e1fea042c9679f": "Dynamo (projectile)",  # pragma: allowlist secret
    "f5c335dfe7232c0b": "MA5K Avenger",  # pragma: allowlist secret
}
KNOWN_PREFIXES = {h[:8] for h in KNOWN_WIDS}

FORMULA_A_MARKER = bytes.fromhex("200002")
SUFFIX = bytes.fromhex("42c9679f")

# Weapon categories for grouping known weapons
WEAPON_CATEGORIES: dict[str, str] = {
    "Bandit Evo": "rifle",
    "BR75": "rifle",
    "M392 Bandit": "rifle",
    "MA40 AR": "rifle",
    "VK78 Commando": "rifle",
    "MA5K Avenger": "rifle",
    "Bulldog": "shotgun",
    "Heatwave": "shotgun",
    "Mangler": "pistol",
    "Mk51 Sidekick": "pistol",
    "Plasma Pistol": "pistol",
    "Pulse Carbine": "pistol",
    "Disruptor": "pistol",
    "Cindershot": "launcher",
    "Fuel Rod SPNKr": "launcher",
    "M41 SPNKr": "launcher",
    "MLRS-2 Hydra": "launcher",
    "Ravager": "launcher",
    "Skewer": "launcher",
    "Stalker Rifle": "sniper",
    "S7 Sniper": "sniper",
    "Shock Rifle": "sniper",
    "Mutilator": "sniper",
    "Gravity Hammer": "melee",
    "Energy Sword": "melee",
    "Vestige": "melee",
    "Sentinel Beam": "special",
    "Needler": "special",
    "M9 Frag": "grenade",
    "Plasma Grenade": "grenade",
    "Spike Grenade": "grenade",
    "Dynamo (hand)": "grenade",
    "Dynamo (projectile)": "grenade",
}


# ---------------------------------------------------------------------------
# Parsing
# ---------------------------------------------------------------------------


def decompress_chunk(path: Path) -> bytes:
    raw = path.read_bytes()
    try:
        return zlib.decompress(raw[2:], -15)
    except Exception:
        return raw


def extract_formula_a_events(data: bytes) -> list[dict]:
    """
    Extract full Formula A event byte context.
    Returns list of dicts with all bytes in the event window.
    """
    events = []
    i = 0
    while i < len(data) - 20:
        if data[i : i + 3] == FORMULA_A_MARKER:
            pb = data[i + 3]
            pi = pb >> 5
            window = data[i + 4 : i + 60]  # generous window
            suf_pos = window.find(SUFFIX)
            if suf_pos >= 8:
                # wid = 8 bytes ending at (suf_pos + 4): [4 var bytes][42c9679f suffix]
                wid_end_in_window = suf_pos + 4
                wid_start_in_window = wid_end_in_window - 8  # = suf_pos - 4
                wid_full = window[wid_start_in_window:wid_end_in_window].hex()

                # Bytes before wid in the window
                pre_wid = window[:wid_start_in_window]
                post_wid = window[wid_end_in_window : wid_end_in_window + 4]

                events.append(
                    {
                        "offset": i,
                        "pi": pi,
                        "pb": pb,
                        "wid_hex": wid_full,
                        "prefix": wid_full[:8],
                        "suf_pos": suf_pos,
                        "pre_wid_bytes": bytes(pre_wid),
                        "post_wid_bytes": bytes(post_wid),
                        "pre_wid_len": wid_start_in_window,
                    }
                )
        i += 1
    return events


# ---------------------------------------------------------------------------
# Structural analysis
# ---------------------------------------------------------------------------


def analyze_byte_patterns(
    events: list[dict],
) -> dict[str, dict]:
    """
    For each known weapon: compute byte value distributions at each relative
    offset before the wid (pre_wid[-1], pre_wid[-2], etc.)

    Returns {weapon_name: {offset_from_wid: Counter(byte_val -> count)}}
    """
    weapon_events: dict[str, list[dict]] = defaultdict(list)

    for ev in events:
        prefix = ev["prefix"]
        if prefix in KNOWN_PREFIXES:
            wid_full = ev["wid_hex"]
            name = KNOWN_WIDS.get(wid_full, f"known({prefix})")
            weapon_events[name].append(ev)

    patterns: dict[str, dict] = {}
    for name, evs in weapon_events.items():
        offset_dist: dict[int, Counter] = defaultdict(Counter)
        pre_lengths = Counter(ev["pre_wid_len"] for ev in evs)

        for ev in evs:
            pre = ev["pre_wid_bytes"]
            # Bytes at fixed offsets from wid start (-1 = byte just before wid)
            for off in range(1, min(len(pre) + 1, 9)):
                offset_dist[-off][pre[-off]] += 1

        patterns[name] = {
            "n_events": len(evs),
            "pre_len_dist": dict(pre_lengths.most_common(5)),
            "byte_dist": {k: dict(v.most_common(5)) for k, v in offset_dist.items()},
        }

    return patterns


def find_discriminating_offsets(patterns: dict[str, dict]) -> dict[int, dict]:
    """
    Find byte offsets where value is consistent (>= 90%) within each weapon
    but varies between weapons.
    Returns {offset: {weapon_name: dominant_byte}}.
    """
    discriminating = {}

    for offset in range(-1, -9, -1):
        weapon_dominant: dict[str, int | None] = {}
        for name, pat in patterns.items():
            byte_dist = pat["byte_dist"].get(offset, {})
            if not byte_dist:
                continue
            total = sum(byte_dist.values())
            dom_byte, dom_count = max(byte_dist.items(), key=lambda x: x[1])
            if dom_count / total >= 0.80:  # consistent within weapon
                weapon_dominant[name] = dom_byte
            else:
                weapon_dominant[name] = None

        # Check if this offset discriminates between categories
        resolved = {k: v for k, v in weapon_dominant.items() if v is not None}
        if len(resolved) >= 3:
            # Map byte value -> weapon names
            byte_to_weapons: dict[int, list] = defaultdict(list)
            for name, byte_val in resolved.items():
                byte_to_weapons[byte_val].append(name)

            if len(byte_to_weapons) >= 2:  # at least 2 distinct values
                discriminating[offset] = {
                    "byte_to_weapons": dict(byte_to_weapons),
                    "weapon_dominant": resolved,
                }

    return discriminating


def classify_unknowns_by_bytes(
    events: list[dict],
    discriminating: dict[int, dict],
    patterns: dict[str, dict],
) -> dict[str, dict]:
    """
    For each unknown prefix, use discriminating byte offsets to classify.
    Returns {prefix: {offset: dominant_byte, matched_weapons, classification}}.
    """
    unknown_events: dict[str, list[dict]] = defaultdict(list)
    for ev in events:
        if ev["prefix"] not in KNOWN_PREFIXES:
            unknown_events[ev["prefix"]].append(ev)

    results = {}
    for prefix, evs in unknown_events.items():
        byte_votes: dict[str, Counter] = {}  # category -> vote count

        for offset, disc in discriminating.items():
            b2w = disc["byte_to_weapons"]

            # Get dominant byte at this offset for the unknown
            offset_dist: Counter = Counter()
            for ev in evs:
                pre = ev["pre_wid_bytes"]
                idx = offset  # negative
                if len(pre) >= abs(idx):
                    offset_dist[pre[idx]] += 1

            if not offset_dist:
                continue

            total = sum(offset_dist.values())
            dom_byte, dom_count = offset_dist.most_common(1)[0]

            if dom_count / total >= 0.70:
                # Map to category
                matched = b2w.get(dom_byte, [])
                for wname in matched:
                    cat = WEAPON_CATEGORIES.get(wname, "other")
                    if cat not in byte_votes:
                        byte_votes[cat] = Counter()
                    byte_votes[cat][wname] += dom_count

        results[prefix] = {
            "n_events": len(evs),
            "byte_votes": {k: dict(v) for k, v in byte_votes.items()},
        }

    return results


# ---------------------------------------------------------------------------
# Report
# ---------------------------------------------------------------------------


def print_structure_report(events: list[dict], match_id: str) -> None:
    patterns = analyze_byte_patterns(events)
    discriminating = find_discriminating_offsets(patterns)

    print(f"\n{'=' * 70}")
    print(f"Match {match_id}: Formula A event byte structure")
    print(f"  Total events: {len(events)}")
    known_count = sum(1 for e in events if e["prefix"] in KNOWN_PREFIXES)
    print(f"  Known wid events: {known_count} | Unknown: {len(events) - known_count}")
    print(f"{'=' * 70}")

    # Pre-wid length distribution
    print("\n--- Pre-wid byte length distribution per weapon ---")
    print(f"  {'Weapon':<24}  {'n':>4}  {'Lengths (most common)'}")
    for name in sorted(patterns, key=lambda x: -patterns[x]["n_events"]):
        pat = patterns[name]
        lens = ", ".join(f"{ln}x{c}" for ln, c in sorted(pat["pre_len_dist"].items()))
        print(f"  {name:<24}  {pat['n_events']:>4}  {lens}")

    # Discriminating offsets
    if discriminating:
        print(f"\n--- Discriminating byte offsets ({len(discriminating)} found) ---")
        for offset in sorted(discriminating):
            disc = discriminating[offset]
            print(f"\n  Offset {offset} from wid:")
            for byte_val, weapons in sorted(disc["byte_to_weapons"].items()):
                cats = {WEAPON_CATEGORIES.get(w, "?") for w in weapons}
                print(f"    0x{byte_val:02x} -> {', '.join(weapons[:5])} ({'/'.join(cats)})")
    else:
        print("\n  No strongly discriminating byte offsets found.")

    # Unknown classification
    print("\n--- Unknown prefix classification by byte pattern ---")
    unknown_cls = classify_unknowns_by_bytes(events, discriminating, patterns)
    for prefix, rec in sorted(unknown_cls.items(), key=lambda x: -x[1]["n_events"])[:20]:
        votes_str = "; ".join(
            f"{cat}: {sum(v.values())} votes ({', '.join(list(v.keys())[:2])})"
            for cat, v in rec["byte_votes"].items()
        )
        if not votes_str:
            votes_str = "no match"
        print(f"  {prefix}  n={rec['n_events']:>3}  {votes_str}")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main() -> None:
    print("inv106 — Formula A event byte structure analysis")
    print("=" * 70)

    CORPUS = [
        "d9329229",
        "00162144",
        "63d6f727",
        "000d5950",
        "1bd7303b",
        "ebfb64f2",
        "f3bc46ab",
        "73284037",
    ]

    # Focus analysis on d9329229 with detailed output
    primary = "d9329229"
    match_dir = CHUNKS_DIR / primary
    all_events = []
    for chunk_path in sorted(match_dir.glob("chunk_*.bin")):
        data = decompress_chunk(chunk_path)
        all_events.extend(extract_formula_a_events(data))

    print_structure_report(all_events, primary)

    # Cross-corpus: collect events for unknown wids only
    print("\n\n=== Cross-corpus byte pattern for top unknowns ===")
    corpus_events: list[dict] = list(all_events)  # already have d9329229

    for match_id in CORPUS[1:]:
        match_dir = CHUNKS_DIR / match_id
        if not match_dir.exists():
            continue
        for chunk_path in sorted(match_dir.glob("chunk_*.bin")):
            data = decompress_chunk(chunk_path)
            corpus_events.extend(extract_formula_a_events(data))

    print(f"\n  Total corpus events: {len(corpus_events)}")
    patterns = analyze_byte_patterns(corpus_events)
    discriminating = find_discriminating_offsets(patterns)
    print(f"  Discriminating offsets found: {list(discriminating.keys())}")

    unknown_cls = classify_unknowns_by_bytes(corpus_events, discriminating, patterns)
    focus_prefixes = ["6d32c7dc", "d48d9b84", "da43eff0", "92f99df4", "94c3a67a"]
    print(f"\n  {'Prefix':<12}  {'n':>4}  Category votes")
    for prefix in focus_prefixes:
        if prefix not in unknown_cls:
            continue
        rec = unknown_cls[prefix]
        votes_str = "; ".join(
            f"{cat}: {sum(v.values())} ({', '.join(list(v.keys())[:2])})"
            for cat, v in rec["byte_votes"].items()
        )
        if not votes_str:
            votes_str = "no match"
        print(f"  {prefix}  {rec['n_events']:>4}  {votes_str}")

    # Also show the pre-wid byte sequences for 6d32c7dc vs Spike Grenade
    print("\n\n=== Byte-by-byte comparison: 6d32c7dc vs Spike Grenade ===")
    _compare_wid_bytes(corpus_events, "6d32c7dc", "6683257c")


def _compare_wid_bytes(events: list[dict], prefix_a: str, prefix_b: str) -> None:
    """Print aligned byte distributions at each offset for two prefixes."""
    evs_a = [e for e in events if e["prefix"] == prefix_a]
    evs_b = [e for e in events if e["prefix"] == prefix_b]

    name_b = next(
        (n for wid, n in KNOWN_WIDS.items() if wid.startswith(prefix_b)), f"known({prefix_b})"
    )
    print(f"  Comparing {prefix_a} (n={len(evs_a)}) vs {name_b}/{prefix_b} (n={len(evs_b)})")
    print(f"  {'Offset':>8}  {'--- ' + prefix_a + ' ---':^30}  {'--- ' + name_b + ' ---':^30}")

    for offset in range(-1, -9, -1):
        dist_a: Counter = Counter()
        dist_b: Counter = Counter()

        for ev in evs_a:
            pre = ev["pre_wid_bytes"]
            if len(pre) >= abs(offset):
                dist_a[pre[offset]] += 1
        for ev in evs_b:
            pre = ev["pre_wid_bytes"]
            if len(pre) >= abs(offset):
                dist_b[pre[offset]] += 1

        def fmt_dist(d: Counter) -> str:
            if not d:
                return "(empty)"
            total = sum(d.values())
            top3 = d.most_common(3)
            parts = [f"0x{b:02x}×{c}" for b, c in top3]
            if len(d) > 1:
                dom_b, dom_c = top3[0]
                parts[0] = f"0x{dom_b:02x}×{dom_c}({dom_c / total:.0%})"
            return " ".join(parts)

        print(f"  {offset:>8}  {fmt_dist(dist_a):^30}  {fmt_dist(dist_b):^30}")


if __name__ == "__main__":
    main()
