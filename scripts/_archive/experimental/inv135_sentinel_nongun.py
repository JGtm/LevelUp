"""inv135 -- Attribution armes + sentinels grenade/melee via counts API.

Logique :
  1. b5>>4 pour identifier le player_index de chaque fire event (inv134)
  2. Pour chaque kill : trouver le fire event le plus recent avant le kill
  3. Sentinel grenade/melee : pour chaque joueur, les N kills avec le plus grand
     gap sont marques comme GRENADE ou MELEE (N = grenade_kills + melee_kills API)
     => comble les trous sans surcompenser (N borne par les counts API)

L'idee : les kills les plus "eloignes" d'un fire event sont les plus probables
d'etre des kills grenade/melee. On les marque en priorite jusqu'a epuiser le
quota API. Les kills restants gardent l'arme du fire event.

Usage:
    python scripts/experimental/inv135_sentinel_nongun.py
    python scripts/experimental/inv135_sentinel_nongun.py --match <match_id>
    python scripts/experimental/inv135_sentinel_nongun.py --debug
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

_MARKER = Bits("0b10100100110")
_WEAPON_BIT_OFFSET = 40
_B5_BIT_OFFSET = 32

# Kills avec gap > ce seuil sont candidats prioritaires au sentinel
GAP_CONF_MS = 500
# Gap infini pour no_event (pas de fire event avant le kill)
_INF_GAP = float("inf")

SENTINEL_GRENADE = "GRENADE"
SENTINEL_MELEE = "MELEE"
SENTINEL_NONGUN = "GRENADE/MELEE"  # quand on ne peut pas distinguer


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


def scan_b5(chunk_data: bytes, estimate_ts) -> list[dict]:
    """Scan fire events, extrait player_index via b5>>4."""
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
        weapon_int = bits[weapon_start : weapon_start + 64].uint
        weapon_bytes = weapon_int.to_bytes(8, "big")
        if weapon_int not in WEAPON_INT_TO_NAME and weapon_bytes[4:] != COMMON_WEAPON_SUFFIX:
            continue
        byte_pos = position // 8
        events.append({
            "ts": estimate_ts(byte_pos),
            "pi": b5_int >> 4,
            "weapon": WEAPON_INT_TO_NAME.get(weapon_int, WEAPON_ID_MAP.get(weapon_bytes, "?")),
            "byte_pos": byte_pos,
        })

    # Dedup par proximite byte_pos (< 2 bytes = meme event physique)
    events.sort(key=lambda x: x["byte_pos"])
    deduped: list[dict] = []
    last_pos = -999
    for ev in events:
        if ev["byte_pos"] - last_pos > 2:
            deduped.append(ev)
            last_pos = ev["byte_pos"]
    return sorted(deduped, key=lambda x: x["ts"])


def attribute_kills(
    kill_ts_list: list[float],
    event_pool: list[dict],
) -> list[tuple[float, float | None, str | None]]:
    """Pour chaque kill : (kill_ts, gap_ms|None, weapon|None).

    gap=None = no fire event avant ce kill dans les chunks charges.
    """
    pool = list(event_pool)
    results = []
    for t_ms in kill_ts_list:
        candidates = [e for e in pool if e["ts"] <= t_ms]
        if candidates:
            ev = max(candidates, key=lambda e: e["ts"])
            gap = t_ms - ev["ts"]
            results.append((t_ms, gap, ev["weapon"]))
            pool = [e for e in pool if e is not ev]
        else:
            results.append((t_ms, None, None))
    return results


def apply_sentinel(
    attributions: list[tuple[float, float | None, str | None]],
    n_grenade: int,
    n_melee: int,
) -> list[tuple[float, str, str]]:
    """Applique les sentinels grenade/melee sur les N kills les plus incertains.

    Strategie :
      - Trier les kills par gap decroissant (None = gap infini = le plus incertain)
      - Marquer les n_grenade premiers comme GRENADE, les n_melee suivants comme MELEE
      - Les kills restants gardent leur arme film (gun)
      - Si n_grenade + n_melee > nombre de kills avec gap >= GAP_CONF_MS,
        on descend dans les kills < GAP_CONF_MS (du plus grand gap au plus petit)
        pour ne pas surcompenser.

    Returns:
        [(kill_ts, label, source)]
        source: 'film_gun' | 'sentinel_grenade' | 'sentinel_melee'
    """
    n_total_sentinel = n_grenade + n_melee
    if n_total_sentinel == 0:
        return [
            (t, w or "(no_event)", "film_gun" if w else "no_event")
            for t, gap, w in attributions
        ]

    # Trier par gap decroissant pour identifier les plus incertains
    indexed = list(enumerate(attributions))
    # gap None = infini (pas d'event) -> le plus incertain
    indexed_sorted = sorted(
        indexed,
        key=lambda x: (_INF_GAP if x[1][1] is None else x[1][1]),
        reverse=True,
    )

    sentinel_indices: dict[int, str] = {}
    quota_g = n_grenade
    quota_m = n_melee

    for orig_idx, (t_ms, gap, weapon) in indexed_sorted:
        if quota_g + quota_m == 0:
            break
        if quota_g > 0:
            sentinel_indices[orig_idx] = "sentinel_grenade"
            quota_g -= 1
        elif quota_m > 0:
            sentinel_indices[orig_idx] = "sentinel_melee"
            quota_m -= 1

    results = []
    for orig_idx, (t_ms, gap, weapon) in enumerate(attributions):
        if orig_idx in sentinel_indices:
            src = sentinel_indices[orig_idx]
            label = SENTINEL_GRENADE if src == "sentinel_grenade" else SENTINEL_MELEE
        elif weapon is None:
            label, src = "(no_event)", "no_event"
        else:
            label, src = weapon, "film_gun"
        results.append((t_ms, label, src))

    return results


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
        """SELECT mp.xuid::TEXT, xa.gamertag, mp.kills,
                  mp.grenade_kills, mp.melee_kills,
                  mp.kills - mp.grenade_kills - mp.melee_kills AS gun_kills
           FROM match_participants mp
           LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid::TEXT
           WHERE mp.match_id=? AND mp.xuid NOT LIKE 'bid(%'""",
        [match_id],
    ).fetchall()
    xuid_to_gt = {r[0]: r[1] or r[0][:16] for r in players}
    xuid_int_to_str = {int(r[0]): r[0] for r in players}
    api = {r[0]: {"kills": r[2], "grenade": r[3], "melee": r[4], "gun": r[5]} for r in players}

    chunks = load_chunks(match_id)
    if not chunks:
        print("No cached chunks.")
        return
    chunks_sorted = sorted(chunks)
    print(f"Chunks : {len(chunks)}\n")

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

    # Scan tous les fire events
    events_by_pi: dict[int, list[dict]] = defaultdict(list)
    for idx in chunks_sorted:
        data, start_ms, _ = chunks[idx]
        pkts = index_chunk(data)
        ts_fn = build_packet_estimator(pkts, start_ms)
        for ev in scan_b5(data, ts_fn):
            events_by_pi[ev["pi"]].append(ev)

    kills_db = conn.execute(
        "SELECT killer_xuid, time_ms FROM killer_victim_pairs "
        "WHERE match_id=? AND time_ms IS NOT NULL ORDER BY time_ms",
        [match_id],
    ).fetchall()
    kills_by_xuid: dict[str, list[float]] = defaultdict(list)
    for k, t in kills_db:
        kills_by_xuid[k].append(float(t))

    # En-tete
    print(
        f"{'Player':<22} {'pi':>3} {'k':>3} {'gun_api':>7} "
        f"{'gren_api':>8} {'mel_api':>7} | "
        f"{'film_gun':>8} {'sent_g':>6} {'sent_m':>6} | Match?"
    )
    print("-" * 100)

    total_film_gun = total_sent_g = total_sent_m = 0
    total_api_gun = total_api_g = total_api_m = 0

    for r in sorted(players, key=lambda x: xuid_to_pi.get(x[0], 99)):
        xuid = r[0]
        gt = xuid_to_gt[xuid]
        pi = xuid_to_pi.get(xuid, 0)
        a = api[xuid]
        if a["kills"] == 0:
            continue

        my_kills = kills_by_xuid.get(xuid, [])
        pool = sorted(events_by_pi.get(pi, []), key=lambda e: e["ts"])

        # Attributions brutes
        raw = attribute_kills(my_kills, pool)

        # Application sentinels
        final = apply_sentinel(raw, a["grenade"], a["melee"])

        film_gun = sum(1 for _, lbl, src in final if src == "film_gun")
        sent_g = sum(1 for _, lbl, src in final if src == "sentinel_grenade")
        sent_m = sum(1 for _, lbl, src in final if src == "sentinel_melee")
        no_ev = sum(1 for _, lbl, src in final if src == "no_event")

        match_ok = (
            film_gun == a["gun"]
            and sent_g == a["grenade"]
            and sent_m == a["melee"]
        )
        match_str = "OK" if match_ok else f"DIFF gun={film_gun-a['gun']:+d}"

        print(
            f"  {gt:<22} pi={pi} {a['kills']:>3}k {a['gun']:>7} "
            f"{a['grenade']:>8} {a['melee']:>7} | "
            f"{film_gun:>8} {sent_g:>6} {sent_m:>6} | {match_str}"
        )

        if debug:
            for t_ms, label, src in final:
                print(f"      kill@{t_ms:.0f}ms  {label:<20} [{src}]")

        total_film_gun += film_gun
        total_sent_g += sent_g
        total_sent_m += sent_m
        total_api_gun += a["gun"]
        total_api_g += a["grenade"]
        total_api_m += a["melee"]

    print("-" * 100)
    print(
        f"  {'TOTAL':<22}     {total_api_gun+total_api_g+total_api_m:>3}k {total_api_gun:>7} "
        f"{total_api_g:>8} {total_api_m:>7} | "
        f"{total_film_gun:>8} {total_sent_g:>6} {total_sent_m:>6} | "
        f"gun_diff={total_film_gun-total_api_gun:+d}"
    )

    print()
    print("Legend:")
    print("  film_gun     = kill attribue a une arme via fire event (b5>>4)")
    print("  sent_g/sent_m = kill marque grenade/melee via counts API (sentinel)")
    print("  Match? OK    = film_gun == API gun_kills ET sentinels == API counts")
    print("  DIFF         = ecart residuel (tue par une arme mais no fire event nearby)")


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
