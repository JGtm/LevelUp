"""inv136 -- Validation experimentale du marqueur melee 0xd340.

=== STRUCTURE CONFIRMEE (acurtis 2026-03-18 + inv136 2026-03-19) ===

Event melee dans le layer nibble-shifted (NS), offset a partir de mel_start :
  [0]  b0 : (b0 & 0x07) == 0x03  (last 3 bits du lead byte)
  [1]  b1 : variable PAR MATCH (0x40 pour a974fdeb, autre valeur pour autres matchs)
  [2]  b2 : compteur/sequence (incrementiel par evenement)
  [3]  b3 : 0x20 (CONSTANT)  <- discrimine melee de fire event en NS
  [4]  b4 : 0x00 (CONSTANT)
  [5]  b5ctx : 0x00 (CONSTANT)
  [6]  b6 : 0x0d (fire event lead, constant)
  [7]  b7 : 0x26 (fire event b1 constant)
  [8]  b5melee : (player_index << 4) | slot   <- pi = b5 >> 4
  [9]  b9 : 0x40-0x43 (fire b3 constant)
  [10] b10 : fire_counter
  [11] b11 : (pi_fire << 4) | slot_fire (fire b5 de la victime?)
  [12:20] weapon_id (8 bytes) = arme TENUE par l'attaquant

Layer NS = nibble-shift de 4 bits gauche sur les raw bytes.

La sequence "Before" acurtis (d3 40 13 00 a6 80 26 01 51 00 42) est en raw bytes,
avec b5_melee = raw[+8] = 0x51 = (pi=5 << 4) | 1. Les 9 events a974fdeb confirmes
sont dans le layer NS (meme formule pi = ns[mel_start+8] >> 4).

=== BIAIS ET LIMITES ===

Le filtre de base (b0&7==3, b3=0x20, b4=b5=0, b6=0x0d, b7=0x26, suffix@+12) capture
AUSSI les fire events en NS (meme signature a un decalage de 6 bytes). Le filtre b1
discrimine melee vs fire MAIS sa valeur est specifique au match.

Detection sur 3 matchs :
  a974fdeb : b1=0x40 -> 9 events / 18 api_melee (50% detection, ~50% faux negatifs)
  d9329229 : b3 != 0x20 -> 0 events detectes (structure differente ou b3 variable)
  f2f81265 : 318 events avec le filtre fort (b1 a distinguer)

CONCLUSION : Methode experimentale, pas encore fiable en production.
  => Continuer a utiliser le systeme sentinel (inv135) pour l'attribution des kills.
  => La formule pi = b5 >> 4 est validee pour les events confirmes.

Usage:
    python scripts/experimental/inv136_melee_marker.py
    python scripts/experimental/inv136_melee_marker.py --match <match_id>
    python scripts/experimental/inv136_melee_marker.py --b1 0x40  (b1 specifique)
    python scripts/experimental/inv136_melee_marker.py --debug
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

from src.analysis._weapon_data import WEAPON_ID_MAP, WEAPON_INT_TO_NAME
from src.analysis.packet_index import (
    build_packet_estimator,
    detect_pi_from_metadata,
    extract_metadata_payload,
    index_chunk,
)

CACHE_DIR = ROOT / "data" / "cache" / "film_chunks"
MANIFEST_DIR = ROOT / "data" / "cache" / "film_manifests"
SHARED_DB = ROOT / "data" / "warehouse" / "shared_matches_v2.duckdb"

ALL_SUFFIXES = [
    bytes.fromhex("42c9679f"),  # suffixe commun ~90% armes
    bytes.fromhex("a730e49f"),  # Gravity Hammer alt
    bytes.fromhex("8978aa7a"),  # Energy Sword alt
    bytes.fromhex("e7232c0b"),  # MA5K Avenger
    bytes.fromhex("c8fb11d0"),  # Mythic Sandwich
]

# Constantes confirmees dans le layer NS pour events melee
_B3_CONST = 0x20  # ns[mel_start+3]
_B4_CONST = 0x00  # ns[mel_start+4]
_B5CTX_CONST = 0x00  # ns[mel_start+5]
_B6_CONST = 0x0d  # fire lead
_B7_CONST = 0x26  # fire b1
_B5_MELEE_OFFSET = 8   # pi = ns[mel_start+8] >> 4
_WID_OFFSET = 12        # weapon a offset 12 depuis mel_start


def nibble_shift(data: bytes) -> bytes:
    out = bytearray(len(data) - 1)
    for i in range(len(out)):
        out[i] = ((data[i] << 4) & 0xFF) | (data[i + 1] >> 4)
    return bytes(out)


def _wid_name(wid_bytes: bytes) -> str:
    wid_int = int.from_bytes(wid_bytes, "big")
    if wid_int in WEAPON_INT_TO_NAME:
        return WEAPON_INT_TO_NAME[wid_int]
    if wid_bytes in WEAPON_ID_MAP:
        return WEAPON_ID_MAP[wid_bytes]
    return wid_bytes.hex()[:16]


def load_chunks(match_id: str) -> dict[int, tuple[bytes, int, int]]:
    short = match_id[:8]
    mp = MANIFEST_DIR / f"{short}.json"
    cd = CACHE_DIR / short
    if not mp.exists() or not cd.exists():
        return {}
    meta = {c["index"]: c for c in json.loads(mp.read_text()).get("chunks", [])}
    result = {}
    for f in sorted(cd.glob("chunk_*.bin")):
        idx = int(f.stem.split("_")[1])
        m = meta.get(idx)
        if m:
            result[idx] = (f.read_bytes(), m["start_ms"], m["duration_ms"])
    return result


def scan_melee_ns(
    chunk_data: bytes,
    estimate_ts,
    b1_filter: int | None,
) -> list[dict]:
    """Scanne les events melee dans le layer NS avec filtre constant confirme.

    Structure melee NS : b3=0x20, b4=b5ctx=0, b6=0x0d, b7=0x26, weapon@+12.
    b1_filter : si None, accepte tous les b1 ; sinon filtre sur ce b1.
    """
    ns = nibble_shift(chunk_data)
    ns_len = len(ns)
    events: list[dict] = []

    for suf in ALL_SUFFIXES:
        pos = 0
        while True:
            idx = ns.find(suf, pos)
            if idx == -1:
                break
            wid_start = idx - 4
            mel_start = wid_start - _WID_OFFSET
            if mel_start < 0 or mel_start + 20 > ns_len:
                pos = idx + 1
                continue

            b0 = ns[mel_start]
            b1 = ns[mel_start + 1]
            b3 = ns[mel_start + 3]
            b4 = ns[mel_start + 4]
            b5ctx = ns[mel_start + 5]
            b6 = ns[mel_start + 6]
            b7 = ns[mel_start + 7]
            b9 = ns[mel_start + 9]  # fire b3

            if not (
                (b0 & 0x07) == 0x03
                and b3 == _B3_CONST
                and b4 == _B4_CONST
                and b5ctx == _B5CTX_CONST
                and b6 == _B6_CONST
                and b7 == _B7_CONST
                and (b9 & 0xFC) == 0x40
            ):
                pos = idx + 1
                continue

            if b1_filter is not None and b1 != b1_filter:
                pos = idx + 1
                continue

            b5 = ns[mel_start + _B5_MELEE_OFFSET]
            pi = b5 >> 4
            slot = b5 & 0x0F
            b2 = ns[mel_start + 2]
            b11 = ns[mel_start + 11]  # b5_fire (victime?)
            wid_bytes = bytes(ns[mel_start + _WID_OFFSET : mel_start + _WID_OFFSET + 8])
            ts = estimate_ts(mel_start)

            events.append({
                "ts": ts,
                "pi": pi,
                "slot": slot,
                "b1": b1,
                "b2": b2,
                "b5": b5,
                "b11": b11,
                "wid": _wid_name(wid_bytes),
                "wid_hex": wid_bytes.hex(),
                "byte_pos": mel_start,
            })
            pos = idx + 1

    # Dedup par proximite (< 4 bytes = meme event physique)
    events.sort(key=lambda x: x["byte_pos"])
    deduped: list[dict] = []
    last_pos = -999
    for ev in events:
        if ev["byte_pos"] - last_pos > 4:
            deduped.append(ev)
            last_pos = ev["byte_pos"]

    return sorted(deduped, key=lambda x: x["ts"])


def _b1_distribution(chunk_data: bytes) -> dict[int, int]:
    """Retourne la distribution de b1 pour tous les candidats melee (sans filtre b1)."""
    ns = nibble_shift(chunk_data)
    ns_len = len(ns)
    dist: dict[int, int] = defaultdict(int)

    for suf in ALL_SUFFIXES:
        pos = 0
        while True:
            idx = ns.find(suf, pos)
            if idx == -1:
                break
            mel_start = idx - 4 - _WID_OFFSET
            if mel_start < 0 or mel_start + 12 > ns_len:
                pos = idx + 1
                continue
            b0 = ns[mel_start]
            b3 = ns[mel_start + 3]
            b4 = ns[mel_start + 4]
            b5ctx = ns[mel_start + 5]
            b6 = ns[mel_start + 6]
            b7 = ns[mel_start + 7]
            if (
                (b0 & 0x07) == 0x03
                and b3 == _B3_CONST
                and b4 == _B4_CONST
                and b5ctx == _B5CTX_CONST
                and b6 == _B6_CONST
                and b7 == _B7_CONST
            ):
                dist[ns[mel_start + 1]] += 1
            pos = idx + 1

    return dict(dist)


def run(match_id: str, *, b1_filter: int | None, debug: bool = False) -> None:  # noqa: C901
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
        """SELECT mp.xuid::TEXT, xa.gamertag, mp.kills,
                  mp.grenade_kills, mp.melee_kills
           FROM match_participants mp
           LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid::TEXT
           WHERE mp.match_id=? AND mp.xuid NOT LIKE 'bid(%'""",
        [match_id],
    ).fetchall()
    xuid_to_gt = {r[0]: r[1] or r[0][:16] for r in players}
    xuid_int_to_str = {int(r[0]): r[0] for r in players}
    api = {r[0]: {"kills": r[2], "grenade": r[3], "melee": r[4]} for r in players}

    chunks = load_chunks(match_id)
    if not chunks:
        print("No cached chunks.")
        return
    chunks_sorted = sorted(chunks)
    print(f"Chunks : {len(chunks)}")

    # pi -> xuid via PLAYER_METADATA
    pi_to_xuid_int: dict[int, int] = {}
    for idx in chunks_sorted:
        data, _, _ = chunks[idx]
        pkts = index_chunk(data)
        payload = extract_metadata_payload(data, pkts)
        if payload:
            pi_to_xuid_int.update(detect_pi_from_metadata(payload, xuid_int_to_str))
    xuid_to_pi: dict[str, int] = {str(v): k for k, v in pi_to_xuid_int.items()}
    for r in players:
        if r[0] not in xuid_to_pi:
            xuid_to_pi[r[0]] = 0

    known_pis = set(xuid_to_pi.values())

    # Distribution b1 (diagnostic pour choisir le bon b1)
    if b1_filter is None:
        print("\nDistribution b1 (candidats melee, filtre fort sans b1) :")
        b1_total: dict[int, int] = defaultdict(int)
        for idx in chunks_sorted:
            data, _, _ = chunks[idx]
            for k, v in _b1_distribution(data).items():
                b1_total[k] += v
        top_b1 = sorted(b1_total.items(), key=lambda x: -x[1])[:12]
        for b1, cnt in top_b1:
            print(f"  b1=0x{b1:02x} ({cnt:3d} events)")
        print("\n=> Utiliser --b1 0xXX pour filtrer sur un b1 specifique.")
        print("   (melee = b1 constant par match, fire = b1 varie)")
        return

    print(f"b1 filtre : 0x{b1_filter:02x}\n")

    # Scan melee events
    melee_by_pi: dict[int, list[dict]] = defaultdict(list)
    total_found = 0
    total_valid_pi = 0

    for idx in chunks_sorted:
        data, start_ms, _ = chunks[idx]
        pkts = index_chunk(data)
        ts_fn = build_packet_estimator(pkts, start_ms)
        for ev in scan_melee_ns(data, ts_fn, b1_filter):
            total_found += 1
            if ev["pi"] in known_pis:
                total_valid_pi += 1
            melee_by_pi[ev["pi"]].append(ev)

    print(f"Events melee trouves (b1=0x{b1_filter:02x}) : {total_found}")
    pct_valid = 100 * total_valid_pi // max(total_found, 1)
    print(f"  pi valide (known_pis) : {total_valid_pi} ({pct_valid}%)")
    b5_set = sorted({ev["b5"] for evs in melee_by_pi.values() for ev in evs})
    print(f"  b5 unique ({len(b5_set)}) : {[f'0x{v:02x}' for v in b5_set[:16]]}")
    print()

    # Comparaison API
    print(
        f"{'Player':<22} {'pi':>3} {'api_m':>5} {'ev':>4} | "
        f"{'ev/api':>6} | match?"
    )
    print("-" * 70)

    total_api_m = total_ev = 0

    for r in sorted(players, key=lambda x: xuid_to_pi.get(x[0], 99)):
        xuid = r[0]
        gt = xuid_to_gt[xuid]
        pi = xuid_to_pi.get(xuid, 0)
        a = api[xuid]
        if a["kills"] == 0:
            continue

        api_melee = a["melee"]
        pool = sorted(melee_by_pi.get(pi, []), key=lambda e: e["ts"])
        ev_count = len(pool)

        ratio = f"{ev_count/api_melee:.1f}x" if api_melee > 0 else ("EV?" if ev_count > 0 else "OK")
        match_ok = (ev_count == api_melee)
        match_str = "OK" if match_ok else f"diff={ev_count-api_melee:+d}"

        print(
            f"  {gt:<22} pi={pi} {api_melee:>5} {ev_count:>4} | "
            f"{ratio:>6} | {match_str}"
        )

        if debug and pool:
            for ev in pool[:8]:
                print(
                    f"    ev@{ev['ts']:.0f}ms pi={ev['pi']} b2=0x{ev['b2']:02x}"
                    f" b5=0x{ev['b5']:02x} b11=0x{ev['b11']:02x} wid={ev['wid'][:20]}"
                )

        total_api_m += api_melee
        total_ev += ev_count

    print("-" * 70)
    print(
        f"  {'TOTAL':<26} {total_api_m:>5} {total_ev:>4} | "
        f"{'':>6} | diff={total_ev-total_api_m:+d}"
    )
    print()
    print("Notes :")
    print("  ev/api >= 1 = normal (swings sans kill comptes)")
    print("  ev = 0 avec api > 0 = events manques (b1 wrong ou autre format)")
    print("  ev > 0 avec api = 0 = probablement des swings sans kill")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--match", default=None)
    parser.add_argument("--b1", default=None, help="Valeur b1 en hex (ex: 0x40)")
    parser.add_argument("--debug", action="store_true")
    args = parser.parse_args()

    b1_filter: int | None = None
    if args.b1:
        b1_filter = int(args.b1, 16)

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

    run(match_id, b1_filter=b1_filter, debug=args.debug)


if __name__ == "__main__":
    main()
