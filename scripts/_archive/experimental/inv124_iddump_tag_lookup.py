"""
inv124 -- InfiniteIDDump tag path lookup for known and unknown wids.

Context (inv123):
  Binary approaches to resolving Formula-A-exclusive wids to base weapon wids
  are exhausted. External data is required.

Method:
  Use InfiniteIDDump (Krevil) to map wid prefixes to game tag paths.
  Byte-swap convention:
    LE wid prefix (4B) -> reverse bytes -> 4-byte uppercase BE hex -> lookup in dump

  Byte-swap example:
    MA40 AR  LE=48c19d2d -> reversed bytes [2d 9d c1 48] -> BE=2D9DC148
    Found: "2D9DC148 : objects\\weapons\\rifle\\assault_rifle\\assault_rifle_mp.weapon"

  The launch dump (v6.10020.17500.0) covers 18/33 known wids.
  Unknown wids (6d32c7dc etc.) are post-Season-3 and absent from the launch dump.

Usage:
  1. Download dump file(s) locally:
       curl -o globals-rx-new.txt https://raw.githubusercontent.com/Krevil/InfiniteIDDump/main/6.10020.17500.0/any/globals/globals-rx-new.txt

  2. Run with the local file path:
       python inv124_iddump_tag_lookup.py globals-rx-new.txt

  3. Or run without arguments to use cached results from the last search.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(PROJECT_ROOT))

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

EXCLUSIVE_WIDS: dict[str, str] = {
    "6d32c7dc42c9679f": "UNK_PRIMARY_0x00",  # pragma: allowlist secret
    "f55c4bd242c9679f": "UNK_PRIMARY_0x00_2",  # pragma: allowlist secret
    "0131ea1042c9679f": "UNK_SECONDARY_0x81",  # pragma: allowlist secret
    "d48d9b8442c9679f": "UNK_GRENADE_0x01",  # pragma: allowlist secret
    "67fed82c42c9679f": "UNK_SPIKE",  # pragma: allowlist secret
    "0af3952e42c9679f": "UNK_PAIR_WID",  # pragma: allowlist secret
}

ALL_WIDS = {**KNOWN_WIDS, **EXCLUSIVE_WIDS}


def le_prefix_to_be(le_hex: str) -> str:
    """Convert LE 4-byte hex prefix to BE uppercase hex for IDDump lookup."""
    return bytes.fromhex(le_hex)[::-1].hex().upper()


def load_dump_index(dump_path: Path) -> dict[str, str]:
    """Load IDDump file into a BE_hex -> tag_path dict."""
    index: dict[str, str] = {}
    pattern = re.compile(r"^([0-9A-Fa-f]{8})\s*:\s*(.+)$")
    with dump_path.open("rb") as f:
        for line in f:
            s = line.decode("utf-8", errors="replace").rstrip()
            m = pattern.match(s)
            if m:
                index[m.group(1).upper()] = m.group(2).strip()
    return index


def lookup_wids(dump_index: dict[str, str]) -> None:
    print("\n" + "=" * 80)
    print(f"inv124 -- IDDump tag lookup ({len(dump_index):,} entries loaded)")
    print("=" * 80)

    print("\n  Known wids:")
    found = 0
    for wid_hex, label in KNOWN_WIDS.items():
        le_prefix = wid_hex[:8]
        be_key = le_prefix_to_be(le_prefix)
        tag_path = dump_index.get(be_key)
        status = f"-> {tag_path}" if tag_path else "[NOT FOUND]"
        print(f"  {le_prefix}  [{label:<22}]  BE={be_key}  {status}")
        if tag_path:
            found += 1

    print(f"\n  {found}/{len(KNOWN_WIDS)} known wids found in dump")

    print("\n  Formula-A-exclusive wids (unknown):")
    for wid_hex, label in EXCLUSIVE_WIDS.items():
        le_prefix = wid_hex[:8]
        be_key = le_prefix_to_be(le_prefix)
        tag_path = dump_index.get(be_key)
        status = f"-> {tag_path}" if tag_path else "[NOT FOUND - post-Season-3?]"
        print(f"  {le_prefix}  [{label:<22}]  BE={be_key}  {status}")


def main() -> None:
    if len(sys.argv) < 2:
        print("Usage: python inv124_iddump_tag_lookup.py <path_to_dump_file>")
        print("")
        print("Download launch dump (v6.10020.17500.0):")
        print(
            "  curl -o globals-rx-new.txt https://raw.githubusercontent.com/Krevil/InfiniteIDDump/main/6.10020.17500.0/any/globals/globals-rx-new.txt"
        )
        print("")
        print("Known results for launch dump:")
        print("  18/33 known wids found (18 in globals-rx-new.txt, 0 in common-rtx-new.txt)")
        print("  0/6 exclusive wids found (all post-Season-3)")
        return

    dump_path = Path(sys.argv[1])
    if not dump_path.exists():
        print(f"File not found: {dump_path}")
        sys.exit(1)

    print(f"inv124 -- Loading dump: {dump_path.name} ({dump_path.stat().st_size / 1e6:.1f} MB)")
    dump_index = load_dump_index(dump_path)
    print(f"  Loaded {len(dump_index):,} entries")

    lookup_wids(dump_index)


if __name__ == "__main__":
    main()
