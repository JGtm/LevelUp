"""
inv107 — Slot-byte + pre_wid_len structural family classification.

Key discovery from inv106:
  The Formula A event structure contains a variable-length data block before
  each weapon ID (wid). The LAST BYTE of this block encodes the player's
  inventory slot, and the LENGTH of the block encodes the weapon data type.

Structure:  [20 00 02] [pb: pi<<5|?] [N bytes variable data] [8-byte wid]
  where N bytes = pre_wid_len, and data[-1] = slot_byte.

Confirmed anchor weapons and their structural signatures:
  - Spike Grenade:  slot=0x80, pre_wid_len=16  (grenade slot 1)
  - Dynamo (hand):  slot=0x01, pre_wid_len=20  (grenade slot 2, Dynamo-type)
  - Skewer:         slot=0x27, pre_wid_len=18  (power weapon slot)

Classification logic:
  - Same (slot_byte, pre_wid_len) as a known weapon → same structural family
  - Different slot_byte from all known grenades → primary/secondary weapon

Key conclusions:
  - 67fed82c, 91eb16de, 60f1d512 → slot=0x80, len=16 → SPIKE GRENADE SKIN
  - d48d9b84, 92f99df4 → slot=0x01, len=16 → GRENADE TYPE 2 (non-Dynamo)
  - 6d32c7dc, f55c4bd2, 82a3f54a → slot=0x00, len=16 → PRIMARY WEAPON (non-grenade)
  - f9514800, edff0e96, b1eb695e → slot=0x01/0x00, len=15 → ALTERNATE WEAPON STRUCTURE

Grenade anchors: slot 0x80 = grenade type 1 (Spike), slot 0x01/0x81 = grenade type 2.
Any unknown NOT matching grenade slots is a weapon, not a grenade.
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

# Confirmed structural families from known weapon anchors
KNOWN_FAMILIES: dict[tuple[int, int], str] = {
    (0x80, 16): "SPIKE_GRENADE_FAMILY",
    (0x01, 20): "DYNAMO_FAMILY",
    (0x27, 18): "SKEWER_FAMILY",
}

# Grenade slots (both grenade slot 1 and slot 2)
GRENADE_SLOTS = {0x80, 0x01}

FORMULA_A_MARKER = bytes.fromhex("200002")
SUFFIX = bytes.fromhex("42c9679f")


# ---------------------------------------------------------------------------
# Parsing
# ---------------------------------------------------------------------------


def decompress_chunk(path: Path) -> bytes:
    raw = path.read_bytes()
    try:
        return zlib.decompress(raw[2:], -15)
    except Exception:
        return raw


def extract_formula_a_signatures(data: bytes) -> list[dict]:
    """
    Extract (pi, wid_hex, slot_byte, pre_wid_len) from Formula A events.
    """
    results = []
    i = 0
    while i < len(data) - 20:
        if data[i : i + 3] == FORMULA_A_MARKER:
            pb = data[i + 3]
            pi = pb >> 5
            window = data[i + 4 : i + 80]
            suf_pos = window.find(SUFFIX)
            if suf_pos >= 8:
                wid_start = suf_pos - 4
                wid_full = window[wid_start : wid_start + 8].hex()
                slot_byte = window[wid_start - 1] if wid_start >= 1 else -1
                pre_wid_len = wid_start
                results.append(
                    {
                        "pi": pi,
                        "wid_hex": wid_full,
                        "prefix": wid_full[:8],
                        "slot_byte": slot_byte,
                        "pre_wid_len": pre_wid_len,
                    }
                )
        i += 1
    return results


# ---------------------------------------------------------------------------
# Corpus analysis
# ---------------------------------------------------------------------------


def build_prefix_signatures(corpus_dir: Path) -> dict[str, dict]:
    """
    For each prefix: collect (slot_byte, pre_wid_len) distribution + match/pi info.
    Returns {prefix: {
        'n_events': int,
        'slot_counter': Counter,
        'len_counter': Counter,
        'n_matches': int,
        'n_pis': set of (match_id, pi),
        'is_known': bool,
        'known_label': str,
    }}
    """
    prefix_data: dict[str, dict] = defaultdict(
        lambda: {
            "n_events": 0,
            "slot_counter": Counter(),
            "len_counter": Counter(),
            "match_set": set(),
            "pi_set": set(),
        }
    )

    for match_dir in sorted(corpus_dir.iterdir()):
        if not match_dir.is_dir():
            continue
        match_id = match_dir.name
        for chunk_path in sorted(match_dir.glob("chunk_*.bin")):
            data = decompress_chunk(chunk_path)
            for ev in extract_formula_a_signatures(data):
                rec = prefix_data[ev["prefix"]]
                rec["n_events"] += 1
                rec["slot_counter"][ev["slot_byte"]] += 1
                rec["len_counter"][ev["pre_wid_len"]] += 1
                rec["match_set"].add(match_id)
                rec["pi_set"].add((match_id, ev["pi"]))

    # Finalize: compute dominant values and family
    results = {}
    for prefix, rec in prefix_data.items():
        dom_slot, _ = rec["slot_counter"].most_common(1)[0]
        dom_len, _ = rec["len_counter"].most_common(1)[0]
        family_key = (dom_slot, dom_len)
        is_known = prefix in KNOWN_PREFIXES
        known_label = KNOWN_WIDS.get(prefix + "42c9679f", "")
        if not known_label:
            known_label = KNOWN_WIDS.get(prefix + "e7232c0b", "")

        # Determine structural classification
        if is_known:
            classification = f"KNOWN: {known_label}"
        elif family_key in KNOWN_FAMILIES:
            classification = KNOWN_FAMILIES[family_key]
        elif dom_slot in GRENADE_SLOTS:
            classification = f"GRENADE_VARIANT (slot=0x{dom_slot:02x}, len={dom_len})"
        elif dom_slot in {0x00, 0x81}:
            classification = f"WEAPON_VARIANT (slot=0x{dom_slot:02x}, len={dom_len})"
        elif dom_slot == 0x27:
            classification = "POWER_WEAPON_VARIANT"
        else:
            classification = f"UNKNOWN_FAMILY (slot=0x{dom_slot:02x}, len={dom_len})"

        results[prefix] = {
            "n_events": rec["n_events"],
            "n_matches": len(rec["match_set"]),
            "n_distinct_pi": len(rec["pi_set"]),
            "dom_slot_byte": dom_slot,
            "dom_pre_wid_len": dom_len,
            "slot_distribution": {f"0x{k:02x}": v for k, v in rec["slot_counter"].items()},
            "len_distribution": dict(rec["len_counter"]),
            "is_known": is_known,
            "known_label": known_label,
            "classification": classification,
        }

    return results


# ---------------------------------------------------------------------------
# Report
# ---------------------------------------------------------------------------


def print_classification_report(signatures: dict) -> None:
    print("\n" + "=" * 80)
    print("inv107 — Structural family classification by slot_byte + pre_wid_len")
    print("=" * 80)

    # Group by classification
    by_class: dict[str, list] = defaultdict(list)
    for prefix, rec in signatures.items():
        by_class[rec["classification"]].append((prefix, rec))

    # Sort each group by n_events desc
    for cls in sorted(by_class):
        items = sorted(by_class[cls], key=lambda x: -x[1]["n_events"])
        known_flag = " [CONFIRMED]" if any(r["is_known"] for _, r in items) else ""
        print(f"\n--- {cls}{known_flag} ({len(items)} prefixes) ---")
        print(
            f"  {'Prefix':<12}  {'n':>5}  {'matches':>7}  {'pis':>4}  "
            f"{'slot':>6}  {'len':>4}  slot_dist"
        )
        for prefix, rec in items[:15]:
            slot_dist = ", ".join(f"{k}:{v}" for k, v in sorted(rec["slot_distribution"].items()))
            known_str = f" [{rec['known_label']}]" if rec["is_known"] else ""
            print(
                f"  {prefix}  {rec['n_events']:>5}  {rec['n_matches']:>7}  "
                f"{rec['n_distinct_pi']:>4}  "
                f"0x{rec['dom_slot_byte']:02x}  {rec['dom_pre_wid_len']:>4}  "
                f"{slot_dist}{known_str}"
            )
        if len(items) > 15:
            print(f"  ... +{len(items) - 15} more")

    # Summary table
    print("\n\n=== SUMMARY: Structural families ===\n")
    print(f"  {'slot':>6}  {'len':>4}  {'known anchor':<24}  n_prefixes  total_events")
    family_summary: dict[tuple, dict] = defaultdict(
        lambda: {"known": "", "n_prefixes": 0, "total_events": 0}
    )
    for _prefix, rec in signatures.items():
        key = (rec["dom_slot_byte"], rec["dom_pre_wid_len"])
        if rec["is_known"] and rec["known_label"]:
            family_summary[key]["known"] = rec["known_label"]
        family_summary[key]["n_prefixes"] += 1
        family_summary[key]["total_events"] += rec["n_events"]

    for (sb, plen), fs in sorted(family_summary.items()):
        print(
            f"  0x{sb:02x}  {plen:>4}  {fs['known']:<24}  "
            f"{fs['n_prefixes']:>10}  {fs['total_events']:>12}"
        )

    # Specific focus: classify the top 10 unknowns
    print("\n\n=== TOP UNKNOWN CLASSIFICATIONS ===\n")
    unknowns = [(p, r) for p, r in signatures.items() if not r["is_known"]]
    unknowns.sort(key=lambda x: -x[1]["n_events"])
    print(f"  {'Prefix':<12}  {'n':>5}  {'Classification':<40}  {'Confidence'}")
    for prefix, rec in unknowns[:15]:
        confidence = _assess_confidence(rec, signatures)
        print(f"  {prefix}  {rec['n_events']:>5}  {rec['classification']:<40}  {confidence}")


def _assess_confidence(rec: dict, all_sigs: dict) -> str:
    """Assess classification confidence based on slot+len consistency."""
    slot_total = sum(rec["slot_distribution"].values())
    dom_slot_count = rec["slot_distribution"].get(f"0x{rec['dom_slot_byte']:02x}", 0)
    slot_purity = dom_slot_count / slot_total if slot_total else 0

    len_total = sum(rec["len_distribution"].values())
    dom_len_count = rec["len_distribution"].get(rec["dom_pre_wid_len"], 0)
    len_purity = dom_len_count / len_total if len_total else 0

    if slot_purity >= 0.95 and len_purity >= 0.95 and rec["n_events"] >= 10:
        return "HIGH"
    elif slot_purity >= 0.80 and len_purity >= 0.90 and rec["n_events"] >= 5:
        return "MEDIUM"
    else:
        return "LOW"


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main() -> None:
    print("inv107 — Slot-byte + pre_wid_len structural family classification")
    print("Building corpus signatures...")

    signatures = build_prefix_signatures(CHUNKS_DIR)
    print(f"  {len(signatures)} prefixes found")

    print_classification_report(signatures)

    # Save
    out = {
        p: {k: v for k, v in rec.items() if k not in ("slot_distribution",)}
        for p, rec in signatures.items()
    }
    out_path = PROJECT_ROOT / "scripts/experimental/outputs/inv107_slot_family_classification.json"
    out_path.parent.mkdir(exist_ok=True)
    out_path.write_text(json.dumps(out, indent=2))
    print(f"\nResults saved to {out_path}")


if __name__ == "__main__":
    main()
