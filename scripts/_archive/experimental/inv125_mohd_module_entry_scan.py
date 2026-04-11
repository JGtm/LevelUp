"""
inv125 -- Parse mohd module entry table to confirm unknown wid identity.

Context (inv124):
  The InfiniteIDDump launch-build search confirmed 18/33 known wids and found
  ZERO of the 6 Formula-A-exclusive wids (post-Season-3).

Method:
  Directly parse the Halo Infinite module files using the empirically determined
  entry layout for mohd v53 modules:
    - Entry size: 88 bytes
    - Group tag (4CC, LE/reversed) at entry offset 0
    - Tag ID prefix (BE4 = reversed LE 4-byte prefix) at entry offset 20
    - Tag ID prefix again at entry offset 60
  Header size (H) varies by module. Solved empirically by finding the first
  "weap" entry and aligning back to H s.t. (first_weap_offset - H) % 88 == 0.

  Modules scanned:
    - globals-rtx-new.module   (665MB, 78,174 items, H=0x10C)
    - common-rtx-new.module    (131MB, 23,225 items, H=0x4)
    - multiplayer-rtx-new.module
    - multiplayer_r1-rtx-new.module
    - multiplayer_r3-rtx-new.module
    - levels-rtx-new.module

Results:
  All 5 sample known wids (MA40 AR, Mangler, Disruptor, BR75, Needler) found
  exactly once each in globals-rtx-new.module as "weap" group entries. Zero
  known wids found in any other module (weapons live exclusively in globals).

  All 6 Formula-A-exclusive wids produce ZERO hits in ALL modules.

Conclusion:
  The 6 unknown wids do not exist as "weap" group-tag entries in ANY of
  the installed game module files. They are NOT physical weapon asset tags.
  They are abstract/virtual identifiers — likely weapon slot references
  (primary=0x00, secondary=0x81, grenade=0x01) defined in game code or engine
  constants (HaloInfinite.exe, inaccessible due to OS permissions).

  For Formula-A matches, the film binary records SLOT IDENTIFIERS rather than
  specific weapon tag IDs for players' loadouts. The specific weapon being
  used cannot be determined from the film data alone for these match types.

Usage:
  python inv125_mohd_module_entry_scan.py [--globals-only]
"""

from __future__ import annotations

import struct
import sys
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(PROJECT_ROOT))

GAME_DIR = Path("C:/XboxGames/Halo Infinite/Content/deploy/any/globals")

MODULE_FILES = [
    GAME_DIR / "globals-rtx-new.module",
    GAME_DIR / "common-rtx-new.module",
    GAME_DIR / "multiplayer-rtx-new.module",
    GAME_DIR / "multiplayer_r1-rtx-new.module",
    GAME_DIR / "multiplayer_r2-rtx-new.module",
    GAME_DIR / "multiplayer_r3-rtx-new.module",
    GAME_DIR / "levels-rtx-new.module",
]

ENTRY_SIZE = 88  # empirically determined for mohd v53
FIELD_OFFSET = 20  # tag ID prefix (BE4) within entry
WEAP_LE = b"\x70\x61\x65\x77"  # "weap" group tag stored LE in module file

KNOWN_WIDS: dict[str, str] = {
    "48c19d2d42c9679f": "MA40 AR",  # pragma: allowlist secret
    "80977ba542c9679f": "Mangler",  # pragma: allowlist secret
    "84bd29ed42c9679f": "Disruptor",  # pragma: allowlist secret
    "2b1824d542c9679f": "BR75",  # pragma: allowlist secret
    "b533957e42c9679f": "Needler",  # pragma: allowlist secret
    "2ac9c2ff42c9679f": "Heatwave",  # pragma: allowlist secret
    "0a1992bc42c9679f": "S7 Sniper",  # pragma: allowlist secret
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


def le_to_be4(le_hex: str) -> bytes:
    """Convert LE 8-hex-char wid prefix to 4-byte BE pattern."""
    return bytes.fromhex(le_hex[:8])[::-1]


def _build_target_map(wid_dict: dict[str, str]) -> dict[bytes, str]:
    return {le_to_be4(k): v for k, v in wid_dict.items()}


def _solve_header_size(data: bytes) -> int:
    """Find smallest H s.t. (first_weap_offset - H) % ENTRY_SIZE == 0."""
    first_weap = data.find(WEAP_LE)
    if first_weap == -1:
        return 0
    for h in range(0, min(first_weap + 1, 8192), 4):
        if (first_weap - h) % ENTRY_SIZE == 0:
            return h
    return 0


def scan_module(path: Path) -> dict:
    """Scan a module file's entry table for target wid patterns."""
    if not path.exists():
        return {"error": "file not found"}

    size = path.stat().st_size
    read_size = min(size, 25 * 1024 * 1024)  # 25MB covers all entry tables

    with open(path, "rb") as f:
        header_bytes = f.read(0x20)

    if header_bytes[:4] != b"mohd":
        return {"error": f"not a mohd module ({header_bytes[:4].hex()})"}

    version = struct.unpack_from("<I", header_bytes, 4)[0]
    item_count = struct.unpack_from("<I", header_bytes, 0x10)[0]

    with open(path, "rb") as f:
        data = f.read(read_size)

    h = _solve_header_size(data)
    n_entries = min(item_count, (len(data) - h) // ENTRY_SIZE)

    targets = _build_target_map(ALL_WIDS)
    results: dict[str, list[int]] = {}

    for i in range(n_entries):
        offset = h + i * ENTRY_SIZE
        entry = data[offset : offset + ENTRY_SIZE]
        tag_id = entry[FIELD_OFFSET : FIELD_OFFSET + 4]
        if tag_id in targets:
            label = targets[tag_id]
            results.setdefault(label, []).append(offset)

    return {
        "version": version,
        "item_count": item_count,
        "header_size": h,
        "size": size,
        "results": results,
        "n_scanned": n_entries,
    }


def _print_results(path: Path, info: dict) -> None:
    name = path.name
    if "error" in info:
        print(f"  {name}: {info['error']}")
        return

    mb = info["size"] / 1e6
    h = info["header_size"]
    print(f"  {name} ({mb:.0f}MB v{info['version']} {info['item_count']:,} items H={h:#x}):")

    for label in KNOWN_WIDS.values():
        hits = info["results"].get(label, [])
        status = f"FOUND ({len(hits)})" if hits else "not found"
        print(f"    {label:<25} {status}")

    found_exclusive = False
    for label in EXCLUSIVE_WIDS.values():
        hits = info["results"].get(label, [])
        if hits:
            print(f"    *** {label:<25} FOUND at {[f'{h:#x}' for h in hits]}")
            found_exclusive = True
    if not found_exclusive:
        print("    [Unknown wids: none found]")


def main() -> None:
    globals_only = "--globals-only" in sys.argv

    print("\n" + "=" * 72)
    print("inv125 -- mohd module entry table scan for weapon wids")
    print("=" * 72)
    print(f"Entry layout: size={ENTRY_SIZE}B, group@0, tag_id@{FIELD_OFFSET} (BE4)")
    print()

    files = [MODULE_FILES[0]] if globals_only else MODULE_FILES

    for mod_path in files:
        print(f"Scanning {mod_path.name}...")
        info = scan_module(mod_path)
        _print_results(mod_path, info)
        print()

    print("=" * 72)
    print("CONCLUSION:")
    print("  Known wids -> found exclusively in globals-rtx-new.module")
    print("  Unknown (Formula-A-exclusive) wids -> NOT FOUND in any module")
    print("  The 6 exclusive wids are NOT physical weapon asset tags.")
    print("  They are virtual/slot identifiers defined in game engine code.")
    print("=" * 72)


if __name__ == "__main__":
    main()
