"""inv133 — Attribution armes par joueur via fire_seq % n_players.

Usage :
    python scripts/experimental/inv133_fire_seq_attribution.py
    python scripts/experimental/inv133_fire_seq_attribution.py --match <match_id>
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

from src.analysis.weapon_parser import (
    build_weapon_timelines,
    scan_fire_events_all,
)
from src.analysis.packet_index import (
    detect_pi_from_metadata,
    extract_metadata_payload,
    index_chunk,
)
from src.analysis._weapon_data import WEAPON_INT_TO_NAME, WEAPON_BYTES_TO_INT

CACHE_DIR = ROOT / "data" / "cache" / "film_chunks"
MANIFEST_DIR = ROOT / "data" / "cache" / "film_manifests"
SHARED_DB = ROOT / "data" / "warehouse" / "shared_matches_v2.duckdb"
META_DB = ROOT / "data" / "warehouse" / "metadata.duckdb"
KILL_WINDOW_MS = 3000


def load_chunks(match_id: str) -> dict[int, tuple[bytes, int, int]]:
    short = match_id[:8]
    manifest_path = MANIFEST_DIR / f"{short}.json"
    chunks_dir = CACHE_DIR / short
    if not manifest_path.exists() or not chunks_dir.exists():
        return {}
    chunks_meta = {c["index"]: c for c in json.loads(manifest_path.read_text()).get("chunks", [])}
    result = {}
    for f in sorted(chunks_dir.glob("chunk_*.bin")):
        idx = int(f.stem.split("_")[1])
        m = chunks_meta.get(idx)
        if m:
            result[idx] = (f.read_bytes(), m["start_ms"], m["duration_ms"])
    return result


def resolve_weapon_name(weapon_bytes: bytes | None, pi: int, chunk_idx: int, timeline_ns: dict) -> str:
    if not weapon_bytes:
        return "—"
    # Les weapon_bytes sont encodés en big-endian dans la prod
    wi = WEAPON_BYTES_TO_INT.get(weapon_bytes)
    if wi is None:
        wi = int.from_bytes(weapon_bytes, "big")
    name = WEAPON_INT_TO_NAME.get(wi)
    if name:
        return name
    # Tentative résolution via NS timeline (Section 1)
    for c in [chunk_idx, chunk_idx - 1, chunk_idx + 1, chunk_idx - 2, chunk_idx + 2]:
        ns_wb = timeline_ns.get(c, {}).get(pi)
        if ns_wb:
            ns_wi = WEAPON_BYTES_TO_INT.get(ns_wb) or int.from_bytes(ns_wb, "big")
            ns_name = WEAPON_INT_TO_NAME.get(ns_wi)
            if ns_name:
                return f"{ns_name}*"
    return "—"


def find_chunk_for_ts(ts_ms: float, chunks_sorted: list[int], timing: list[tuple[int, int]]) -> int:
    for i, cidx in enumerate(chunks_sorted):
        start, end = timing[i]
        if start <= ts_ms < end:
            return cidx
    return chunks_sorted[-1] if chunks_sorted else 0


def run(match_id: str) -> None:
    conn = duckdb.connect(str(SHARED_DB), read_only=True)
    conn.execute(f"ATTACH '{META_DB}' AS meta (READ_ONLY)")

    # Infos match
    mr = conn.execute(
        "SELECT start_time, playlist_name FROM match_registry WHERE match_id=?", [match_id]
    ).fetchone()
    if not mr:
        print(f"Match {match_id} introuvable.")
        return
    print(f"\nMatch : {match_id}")
    print(f"Date  : {mr[0]}  |  {mr[1]}\n")

    # Joueurs
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

    # Chunks
    chunks = load_chunks(match_id)
    if not chunks:
        print("Aucun chunk en cache pour ce match.")
        return
    chunks_sorted = sorted(chunks)

    # Build timeline Section 1 (NS) pour résolution weapon_id
    _, timeline_ns, _, timing = build_weapon_timelines(chunks)

    # detect_pi via PLAYER_METADATA
    pi_to_xuid_int: dict[int, int] = {}
    for idx in chunks_sorted:
        data, _, _ = chunks[idx]
        pkts = index_chunk(data)
        payload = extract_metadata_payload(data, pkts)
        if payload:
            pi_to_xuid_int.update(detect_pi_from_metadata(payload, xuid_int_to_str))

    xuid_to_pi: dict[str, int] = {str(v): k for k, v in pi_to_xuid_int.items()}
    # POV = non trouvé par detect_pi_from_metadata → pi=0
    for xs in xuid_to_gt:
        if xs not in xuid_to_pi:
            xuid_to_pi[xs] = 0

    max_pi = max(xuid_to_pi.values()) if xuid_to_pi else 8
    n_eff = max_pi + 1
    print(f"n_eff = {n_eff}  |  chunks = {len(chunks)}  |  joueurs = {len(players)}\n")

    # Scan fire events
    all_events: list[dict] = []
    for idx in chunks_sorted:
        data, start_ms, dur_ms = chunks[idx]
        all_events.extend(scan_fire_events_all(data, start_ms, dur_ms))
    all_events.sort(key=lambda e: e["timestamp_ms"])
    print(f"Fire events scannés : {len(all_events)}\n")

    # Dispatch par fire_seq % n_eff
    events_by_pi: dict[int, list[dict]] = defaultdict(list)
    for ev in all_events:
        pi = ev["fire_seq"] % n_eff
        events_by_pi[pi].append(ev)

    # Kills avec timestamps
    kills_db = conn.execute(
        "SELECT killer_xuid, time_ms FROM killer_victim_pairs WHERE match_id=? AND time_ms IS NOT NULL ORDER BY time_ms",
        [match_id],
    ).fetchall()

    # Résultats par joueur
    print(f"{'Joueur':<22} {'pi':>3}  {'k':>3}  {'matched':>7}  Armes (fire_seq % n_eff)")
    print("─" * 80)

    for xuid, gt in sorted(xuid_to_gt.items(), key=lambda x: xuid_to_pi.get(x[0], 99)):
        pi = xuid_to_pi.get(xuid, 0)
        k_total = xuid_kills_count.get(xuid, 0)
        if k_total == 0:
            continue

        my_kill_ts = [t for k, t in kills_db if k == xuid]
        pool = list(events_by_pi.get(pi, []))

        weapon_counts: dict[str, int] = defaultdict(int)
        matched = 0
        for t_ms in my_kill_ts:
            candidates = [e for e in pool if (t_ms - KILL_WINDOW_MS) <= e["timestamp_ms"] <= t_ms]
            if candidates:
                ev = max(candidates, key=lambda e: e["timestamp_ms"])
                chunk_idx = find_chunk_for_ts(ev["timestamp_ms"], chunks_sorted, timing)
                wname = resolve_weapon_name(ev["weapon_bytes"], pi, chunk_idx, timeline_ns)
                weapon_counts[wname] += 1
                pool.remove(ev)
                matched += 1
            else:
                weapon_counts["—"] += 1

        wlist = "  ".join(f"{w}×{c}" for w, c in sorted(weapon_counts.items(), key=lambda x: -x[1]))
        print(f"  {gt:<22} pi={pi}  {k_total:>3}k  {matched:>3}/{k_total:<3}  {wlist}")

    print()
    print("Légende : — = aucun fire event dans la fenêtre de 3s | (NS) = weapon_id résolu via Section 1")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--match", default=None)
    args = parser.parse_args()

    conn = duckdb.connect(str(SHARED_DB), read_only=True)
    if args.match:
        match_id = args.match
    else:
        row = conn.execute(
            "SELECT mr.match_id FROM match_registry mr "
            "JOIN (SELECT DISTINCT match_id FROM film_chunks_meta UNION ALL "
            "SELECT match_id FROM match_registry ORDER BY start_time DESC LIMIT 1) x "
            "ON mr.match_id=x.match_id LIMIT 1"
        ).fetchone()
        # Fallback : dernier match avec chunks en cache
        cached = sorted(CACHE_DIR.iterdir(), key=lambda p: p.stat().st_mtime, reverse=True)
        if cached:
            short = cached[0].name
            row = conn.execute(
                "SELECT match_id FROM match_registry WHERE match_id LIKE ? ORDER BY start_time DESC LIMIT 1",
                [f"{short}%"],
            ).fetchone()
            match_id = row[0] if row else short
        else:
            print("Aucun chunk en cache.")
            return

    run(match_id)


if __name__ == "__main__":
    main()
