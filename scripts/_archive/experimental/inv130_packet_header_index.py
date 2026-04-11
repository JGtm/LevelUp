"""inv130 — Packet Header Index (acurtis 2026-03).

Explore la structure par paquets 16 octets des chunks REPLICATION_DATA.
Deux objectifs :
  T1 : Examiner les paquets PLAYER_METADATA (type 8) — mapping pi→xuid ?
  T4 : Cataloguer tous les types de paquets (2, 6, 10, 12…)

Header 16 bytes (little-endian) :
  Type      uint16le
  ???       uint8
  ???       uint8
  Size      uint32le
  Timestamp uint64le (microsecondes)

Usage :
  python scripts/experimental/inv130_packet_header_index.py
  python scripts/experimental/inv130_packet_header_index.py --match 000d5950
  python scripts/experimental/inv130_packet_header_index.py --match 000d5950 --dump-type 8
"""

from __future__ import annotations

import argparse
import os
import struct
import sys
from collections import Counter, defaultdict
from dataclasses import dataclass
from enum import IntEnum
from pathlib import Path

# ══════════════════════════════════════════════════════════════════════════════
# Packet structures (acurtis)
# ══════════════════════════════════════════════════════════════════════════════

HEADER_STRUCT = struct.Struct("<HBBIQ")  # 16 bytes


class PacketType(IntEnum):
    FRAME = 0
    START_CHUNK = 1
    TYPE_2 = 2
    TYPE_6 = 6
    END_CHUNK = 7
    PLAYER_METADATA = 8
    TYPE_10 = 10
    TYPE_12 = 12


_TYPE_NAMES = {
    0: "FRAME",
    1: "START_CHUNK",
    2: "TYPE_2",
    6: "TYPE_6",
    7: "END_CHUNK",
    8: "PLAYER_METADATA",
    10: "TYPE_10",
    12: "TYPE_12",
}


@dataclass
class Packet:
    type: int
    byte2: int
    byte3: int
    size: int
    microseconds: int
    offset: int  # offset du payload (juste après le header)


# ══════════════════════════════════════════════════════════════════════════════
# Indexation
# ══════════════════════════════════════════════════════════════════════════════


def index_chunk(data: bytes) -> list[Packet]:
    """Indexe tous les paquets d'un chunk REPLICATION_DATA."""
    packets: list[Packet] = []
    pos = 0
    while pos + 16 <= len(data):
        pkt_type, byte2, byte3, size, microseconds = HEADER_STRUCT.unpack_from(data, pos)
        payload_offset = pos + 16
        packets.append(
            Packet(
                type=pkt_type,
                byte2=byte2,
                byte3=byte3,
                size=size,
                microseconds=microseconds,
                offset=payload_offset,
            )
        )
        if pkt_type == PacketType.END_CHUNK:
            break
        pos = payload_offset + size
        if pos > len(data):
            break
    return packets


# ══════════════════════════════════════════════════════════════════════════════
# Analyse
# ══════════════════════════════════════════════════════════════════════════════


def analyze_match(
    match_dir: Path,
    *,
    dump_types: set[int] | None = None,
    verbose: bool = False,
) -> dict:
    """Analyse tous les chunks d'un match, retourne les stats agrégées."""
    match_id = match_dir.name
    chunk_files = sorted(match_dir.glob("chunk_*.bin"))
    if not chunk_files:
        return {"match_id": match_id, "error": "no chunks"}

    type_counter: Counter = Counter()
    type_sizes: defaultdict[int, list[int]] = defaultdict(list)
    type_timestamps: defaultdict[int, list[int]] = defaultdict(list)
    total_packets = 0
    metadata_payloads: list[tuple[str, int, bytes]] = []  # (chunk_name, pkt_idx, payload)
    all_packets_by_type: defaultdict[int, list[tuple[str, Packet]]] = defaultdict(list)

    for cf in chunk_files:
        chunk_name = cf.stem
        data = cf.read_bytes()
        packets = index_chunk(data)

        for i, pkt in enumerate(packets):
            type_counter[pkt.type] += 1
            type_sizes[pkt.type].append(pkt.size)
            type_timestamps[pkt.type].append(pkt.microseconds)
            total_packets += 1

            # T1 : collecter les payloads PLAYER_METADATA
            if pkt.type == PacketType.PLAYER_METADATA:
                end = min(pkt.offset + pkt.size, len(data))
                payload = data[pkt.offset : end]
                metadata_payloads.append((chunk_name, i, payload))

            # Dump demandé pour d'autres types
            if dump_types and pkt.type in dump_types:
                all_packets_by_type[pkt.type].append((chunk_name, pkt))
                if verbose and pkt.size > 0 and pkt.size <= 4096:
                    end = min(pkt.offset + pkt.size, len(data))
                    payload = data[pkt.offset : end]
                    _hexdump_short(f"  [{chunk_name} pkt#{i}]", payload, max_bytes=128)

    # Timestamps → durée du match en ms
    all_ts = []
    for ts_list in type_timestamps.values():
        all_ts.extend(ts_list)
    ts_min = min(all_ts) if all_ts else 0
    ts_max = max(all_ts) if all_ts else 0
    duration_ms = (ts_max - ts_min) / 1000 if all_ts else 0

    result = {
        "match_id": match_id,
        "n_chunks": len(chunk_files),
        "total_packets": total_packets,
        "duration_ms": duration_ms,
        "type_counts": dict(type_counter.most_common()),
        "type_avg_sizes": {
            t: sum(sizes) / len(sizes) for t, sizes in type_sizes.items()
        },
        "metadata_payloads": metadata_payloads,
    }

    return result


def analyze_metadata_payload(payload: bytes, match_id: str = "") -> dict:
    """Tente de décoder un payload PLAYER_METADATA.

    Recherche des patterns connus :
    - XUIDs (uint64le, typiquement 0x0009... range)
    - Gamertags (ASCII strings)
    - Player indices (0-7)
    """
    info: dict = {
        "size": len(payload),
        "xuids_found": [],
        "ascii_strings": [],
        "possible_pi_mappings": [],
    }

    # Recherche de XUIDs (pattern : 8 bytes LE, range raisonnable pour Xbox Live)
    # XUIDs Xbox Live sont typiquement dans la plage 0x0009_0000_0000_0000 - 0x0009_FFFF_FFFF_FFFF
    for i in range(0, len(payload) - 7):
        val = struct.unpack_from("<Q", payload, i)[0]
        if 0x0009_0000_0000_0000 <= val <= 0x000F_FFFF_FFFF_FFFF:
            # Vérifier le contexte : byte avant pourrait être un player_index
            ctx_before = payload[max(0, i - 4) : i]
            ctx_after = payload[i + 8 : min(len(payload), i + 12)]
            info["xuids_found"].append(
                {
                    "offset": i,
                    "xuid": val,
                    "xuid_hex": f"{val:016x}",
                    "before": ctx_before.hex(),
                    "after": ctx_after.hex(),
                }
            )

    # Recherche de strings ASCII (gamertags : 3-16 chars alphanumériques)
    ascii_run = []
    for i, b in enumerate(payload):
        if 0x20 <= b <= 0x7E:
            ascii_run.append((i, chr(b)))
        else:
            if len(ascii_run) >= 3:
                s = "".join(c for _, c in ascii_run)
                start = ascii_run[0][0]
                info["ascii_strings"].append({"offset": start, "string": s, "len": len(s)})
            ascii_run = []
    if len(ascii_run) >= 3:
        s = "".join(c for _, c in ascii_run)
        info["ascii_strings"].append({"offset": ascii_run[0][0], "string": s, "len": len(s)})

    return info


def _hexdump_short(prefix: str, data: bytes, *, max_bytes: int = 64) -> None:
    """Affiche un hexdump court."""
    show = data[:max_bytes]
    hex_str = " ".join(f"{b:02x}" for b in show)
    suffix = f"... (+{len(data) - max_bytes} bytes)" if len(data) > max_bytes else ""
    print(f"{prefix} [{len(data)}B] {hex_str}{suffix}")


# ══════════════════════════════════════════════════════════════════════════════
# Cross-validation avec les XUIDs connus
# ══════════════════════════════════════════════════════════════════════════════


def load_known_xuids() -> dict[int, str]:
    """Charge les XUIDs connus depuis xuid_aliases.json."""
    import json

    path = Path("data/xuid_aliases.json")
    if not path.exists():
        return {}
    raw = json.loads(path.read_text(encoding="utf-8"))
    result = {}
    for xuid_str, gamertag in raw.items():
        with_prefix = xuid_str if not xuid_str.startswith("xuid(") else xuid_str
        try:
            xuid_int = int(xuid_str.replace("xuid(", "").replace(")", ""))
            result[xuid_int] = gamertag
        except ValueError:
            continue
    return result


# ══════════════════════════════════════════════════════════════════════════════
# Main
# ══════════════════════════════════════════════════════════════════════════════


def main() -> None:
    parser = argparse.ArgumentParser(description="inv130 — Packet Header Index")
    parser.add_argument("--match", help="Match ID (8 chars) à analyser (sinon : tous)")
    parser.add_argument(
        "--dump-type",
        type=int,
        action="append",
        default=[],
        help="Types de paquets à dump en hex (ex: --dump-type 8 --dump-type 10)",
    )
    parser.add_argument("--max-matches", type=int, default=10, help="Max matchs à analyser")
    parser.add_argument("-v", "--verbose", action="store_true")
    args = parser.parse_args()

    chunks_root = Path("data/investigation/chunks")
    if not chunks_root.exists():
        print(f"ERREUR: {chunks_root} n'existe pas", file=sys.stderr)
        sys.exit(1)

    # Sélection des matchs
    if args.match:
        match_dirs = [chunks_root / args.match]
        if not match_dirs[0].exists():
            # Recherche partielle
            candidates = [d for d in chunks_root.iterdir() if d.is_dir() and args.match in d.name]
            if not candidates:
                print(f"ERREUR: match '{args.match}' non trouvé", file=sys.stderr)
                sys.exit(1)
            match_dirs = candidates[:args.max_matches]
    else:
        match_dirs = sorted(
            [d for d in chunks_root.iterdir() if d.is_dir()]
        )[:args.max_matches]

    dump_types = set(args.dump_type) if args.dump_type else None
    known_xuids = load_known_xuids()
    print(f"XUIDs connus chargés : {len(known_xuids)}")

    # ── T4 : Stats agrégées par type ──────────────────────────────────────
    global_type_counter: Counter = Counter()
    global_type_sizes: defaultdict[int, list[int]] = defaultdict(list)
    all_metadata: list[tuple[str, str, int, bytes]] = []  # (match_id, chunk, idx, payload)
    n_matches = 0

    for match_dir in match_dirs:
        if not match_dir.is_dir():
            continue
        result = analyze_match(match_dir, dump_types=dump_types, verbose=args.verbose)
        if "error" in result:
            print(f"  {result['match_id']}: {result['error']}")
            continue
        n_matches += 1

        for t, count in result["type_counts"].items():
            global_type_counter[t] += count

        for t, avg_size in result["type_avg_sizes"].items():
            global_type_sizes[t].append(avg_size)

        # T1 : collecter metadata
        for chunk_name, pkt_idx, payload in result["metadata_payloads"]:
            all_metadata.append((result["match_id"], chunk_name, pkt_idx, payload))

        if args.verbose or len(match_dirs) <= 5:
            print(f"\n{'='*60}")
            print(f"Match: {result['match_id']} — {result['n_chunks']} chunks, "
                  f"{result['total_packets']} paquets, {result['duration_ms']:.0f}ms")
            for t in sorted(result["type_counts"]):
                name = _TYPE_NAMES.get(t, f"UNKNOWN_{t}")
                avg = result["type_avg_sizes"].get(t, 0)
                print(f"  {name:20s} (type {t:2d}) : {result['type_counts'][t]:5d} paquets, "
                      f"avg size {avg:8.0f}B")

    # ── T4 : Résumé global ───────────────────────────────────────────────
    print(f"\n{'='*60}")
    print(f"RÉSUMÉ GLOBAL — {n_matches} matchs analysés")
    print(f"{'='*60}")
    total = sum(global_type_counter.values())
    print(f"Total paquets : {total}")
    for t, count in global_type_counter.most_common():
        name = _TYPE_NAMES.get(t, f"UNKNOWN_{t}")
        pct = 100 * count / total if total else 0
        avg_sizes = global_type_sizes.get(t, [0])
        avg_sz = sum(avg_sizes) / len(avg_sizes)
        print(f"  {name:20s} (type {t:2d}) : {count:6d} ({pct:5.1f}%), avg size {avg_sz:8.0f}B")

    # ── T1 : Analyse PLAYER_METADATA ─────────────────────────────────────
    print(f"\n{'='*60}")
    print(f"PLAYER_METADATA (type 8) — {len(all_metadata)} paquets trouvés")
    print(f"{'='*60}")

    if not all_metadata:
        print("  Aucun paquet PLAYER_METADATA trouvé dans le corpus.")
        print("  → Ce type n'est peut-être pas présent dans les REPLICATION_DATA décompressés.")
        print("  → Il pourrait être dans les chunks de type 1 (METADATA) non téléchargés.")
    else:
        # Analyser chaque payload
        unique_sizes = Counter(len(payload) for _, _, _, payload in all_metadata)
        print(f"  Tailles uniques : {dict(unique_sizes)}")

        # Analyser en détail les 5 premiers
        for i, (mid, chunk_name, pkt_idx, payload) in enumerate(all_metadata[:10]):
            print(f"\n  ── {mid}/{chunk_name} pkt#{pkt_idx} ({len(payload)}B) ──")
            _hexdump_short("    Header 64B", payload, max_bytes=64)

            info = analyze_metadata_payload(payload, match_id=mid)

            if info["xuids_found"]:
                print(f"    XUIDs trouvés : {len(info['xuids_found'])}")
                for x in info["xuids_found"]:
                    xuid_int = x["xuid"]
                    gt = known_xuids.get(xuid_int, "INCONNU")
                    print(f"      offset={x['offset']:4d}  xuid={x['xuid_hex']}  "
                          f"gamertag={gt}  before={x['before']}  after={x['after']}")
            else:
                print("    Aucun XUID détecté dans la plage Xbox Live")

            # Strings ASCII intéressantes (filtrer le bruit)
            interesting = [s for s in info["ascii_strings"] if s["len"] >= 4]
            if interesting:
                print(f"    Strings ASCII (≥4 chars) : {len(interesting)}")
                for s in interesting[:15]:
                    print(f"      offset={s['offset']:4d}  \"{s['string']}\"")

    # ── Types inconnus : dump si demandé ──────────────────────────────────
    if dump_types:
        for t in sorted(dump_types):
            if t == 8:
                continue  # Déjà traité
            name = _TYPE_NAMES.get(t, f"UNKNOWN_{t}")
            count = global_type_counter.get(t, 0)
            print(f"\n{'='*60}")
            print(f"TYPE {t} ({name}) — {count} paquets total")
            if count == 0:
                print("  Aucun paquet de ce type.")
                continue
            # Dump quelques exemples depuis le premier match
            for match_dir in match_dirs[:3]:
                if not match_dir.is_dir():
                    continue
                for cf in sorted(match_dir.glob("chunk_*.bin"))[:3]:
                    data = cf.read_bytes()
                    packets = index_chunk(data)
                    for pi, pkt in enumerate(packets):
                        if pkt.type == t and pkt.size > 0:
                            end = min(pkt.offset + pkt.size, len(data))
                            payload = data[pkt.offset : end]
                            _hexdump_short(
                                f"  [{match_dir.name}/{cf.stem} pkt#{pi}]",
                                payload,
                                max_bytes=128,
                            )

    print("\nDone.")


if __name__ == "__main__":
    main()
