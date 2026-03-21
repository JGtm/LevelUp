"""Expérimental — corrélation kills->armes via b2_to_pi exclusivement.

Compare l'approche actuelle (global pool) avec l'approche b2_to_pi :
- global pool : fire_events_global, aucun filtre joueur
- b2_to_pi    : fire_events groupés par joueur via map_b2_to_player,
                chaque kill ne cherche QUE dans le pool de son tueur

Usage :
    python scripts/exp_b2_pi_correlation.py
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

import duckdb

from src.analysis._weapon_data import WEAPON_BYTES_TO_INT, WEAPON_TIMING_BY_ID
from src.analysis.packet_index import (
    detect_pi_from_metadata,
    extract_metadata_payload,
    index_chunk,
)
from src.analysis.weapon_parser import (
    KILL_WINDOW_MS,
    build_weapon_timelines,
    compute_confidence,
    detect_player_indices,
    find_chunk_at_time,
    group_events_by_pi,
    map_b2_to_player,
    scan_fire_events_all,
)

MATCH_ID = "82f3af9f-c0fa-477b-be9b-df240d62305d"
CACHE_DIR = ROOT / "data/cache/film_chunks" / MATCH_ID[:8]
MANIFEST = ROOT / "data/cache/film_manifests" / f"{MATCH_ID[:8]}.json"
SHARED_DB = ROOT / "data/warehouse/shared_matches_v2.duckdb"
META_DB = ROOT / "data/warehouse/metadata.duckdb"

_DEFAULT_TIMING = (650, 2000)


# ── helpers ──────────────────────────────────────────────────────────────────


def load_chunks() -> dict[int, tuple[bytes, int, int]]:
    """Charge les chunks REPLICATION_DATA depuis le cache."""
    manifest = json.loads(MANIFEST.read_text())
    idx_to_meta = {c["index"]: c for c in manifest["chunks"] if c["chunk_type"] == 2}

    chunks: dict[int, tuple[bytes, int, int]] = {}
    for bin_file in sorted(CACHE_DIR.glob("chunk_*.bin")):
        idx = int(bin_file.stem.split("_")[1])
        if idx in idx_to_meta:
            meta = idx_to_meta[idx]
            data = bin_file.read_bytes()
            chunks[idx] = (data, meta["start_ms"], meta["duration_ms"])
    print(f"[chunks] {len(chunks)} chunks chargés (idx {min(chunks)}-{max(chunks)})")
    return chunks


def resolve_pi_map(
    chunks: dict[int, tuple[bytes, int, int]],
    participants: dict[str, str],
) -> dict[str, int]:
    """Résout xuid -> player_index depuis le premier chunk disponible.

    Essaie PLAYER_METADATA puis fallback acurtis.
    Retourne {xuid_str: pi}.
    """
    xuid_int_map = {int(x): x for x in participants if x.isdigit()}

    # Chunk 0 (type 1) absent du cache -> fallback direct
    first_idx = min(chunks)
    first_data, _, _ = chunks[first_idx]
    first_packets = index_chunk(first_data)

    # Tentative metadata
    try:
        payload = extract_metadata_payload(first_data, first_packets)
        if payload:
            pi_to_xuid = detect_pi_from_metadata(payload, xuid_int_map)
            if pi_to_xuid:
                resolved = set(pi_to_xuid.values())
                missing = {x: g for x, g in xuid_int_map.items() if x not in resolved}
                if missing:
                    pov = detect_player_indices(first_data, missing)
                    pi_to_xuid.update(pov)
                xuid_to_pi = {str(v): k for k, v in pi_to_xuid.items()}
                print(f"[resolve_pi] METADATA OK -> {len(xuid_to_pi)} joueurs")
                return xuid_to_pi
    except Exception as exc:
        print(f"[resolve_pi] metadata failed: {exc}")

    # Fallback acurtis
    pi_to_xuid = detect_player_indices(first_data, xuid_int_map)
    xuid_to_pi = {str(v): k for k, v in pi_to_xuid.items()}
    print(f"[resolve_pi] acurtis fallback -> {len(xuid_to_pi)} joueurs")
    return xuid_to_pi


def load_match_data(conn: duckdb.DuckDBPyConnection) -> tuple[dict, list[dict]]:
    """Charge participants et kills depuis la DB."""
    rows = conn.execute(
        "SELECT mp.xuid, COALESCE(xa.gamertag, mp.xuid) "
        "FROM match_participants mp "
        "LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid "
        "WHERE mp.match_id = ? AND mp.xuid NOT LIKE 'bid(%'",
        (MATCH_ID,),
    ).fetchall()
    participants = {xuid: gt for xuid, gt in rows if xuid}

    kill_rows = conn.execute(
        "SELECT k.xuid, k.time_ms "
        "FROM highlight_events k "
        "WHERE k.match_id = ? AND k.event_type = 'kill' "
        "ORDER BY k.xuid, k.time_ms",
        (MATCH_ID,),
    ).fetchall()
    kills = [{"xuid": r[0], "time_ms": r[1]} for r in kill_rows]

    return participants, kills


def load_weapon_labels(meta_conn: duckdb.DuckDBPyConnection) -> dict[int, str]:
    rows = meta_conn.execute("SELECT weapon_id, name_fr FROM weapon_labels").fetchall()
    return {r[0]: r[1] for r in rows}


def correlate_b2_pi(
    kills: list[dict],
    fire_events_by_pi: dict[int, list[dict]],
    xuid_to_pi: dict[str, int],
    timeline: dict,
    swap_pis: dict,
    timing: list,
    chunks_sorted: list,
    timeline_ns: dict,
) -> list[dict]:
    """Corrélation kills->armes en n'utilisant QUE les events du pi du tueur."""
    results = []

    for kill in sorted(kills, key=lambda k: k["time_ms"]):
        xuid = kill["xuid"]
        t_ms = kill["time_ms"]
        pi = xuid_to_pi.get(xuid)

        # Pool restreint au tueur
        pool = list(fire_events_by_pi.get(pi, [])) if pi is not None else []

        # Chercher dans la fenêtre
        candidates = [
            (i, ev)
            for i, ev in enumerate(pool)
            if (t_ms - KILL_WINDOW_MS) <= ev["timestamp_ms"] <= t_ms
        ]

        if candidates:
            best_idx, best_ev = max(candidates, key=lambda x: x[1]["timestamp_ms"])
            pool.pop(best_idx)
            if pi is not None:
                fire_events_by_pi[pi] = pool  # update (claim-and-remove)

            wid = WEAPON_BYTES_TO_INT.get(best_ev["weapon_bytes"])
            if wid is None:
                chunk_idx = find_chunk_at_time(chunks_sorted, timing, int(best_ev["timestamp_ms"]))
                ns_wb = timeline_ns.get(chunk_idx, {}).get(best_ev.get("player_index"))
                if ns_wb:
                    wid = WEAPON_BYTES_TO_INT.get(ns_wb)
            if wid is None:
                wid = int.from_bytes(best_ev["weapon_bytes"], byteorder="big")

            delta = int(t_ms - best_ev["timestamp_ms"])
            conf = compute_confidence(wid, delta)
            swap_ms, _ = WEAPON_TIMING_BY_ID.get(wid, _DEFAULT_TIMING)
            results.append(
                {
                    "xuid": xuid,
                    "time_ms": t_ms,
                    "weapon_id": wid,
                    "delta_ms": delta,
                    "confidence": conf,
                    "swap_detected": delta >= swap_ms,
                    "source": "b2_pi",
                    "pi": pi,
                    "ev_pi": best_ev.get("player_index"),
                }
            )
        else:
            results.append(
                {
                    "xuid": xuid,
                    "time_ms": t_ms,
                    "weapon_id": None,
                    "delta_ms": None,
                    "confidence": "none",
                    "swap_detected": False,
                    "source": "no_match",
                    "pi": pi,
                    "ev_pi": None,
                }
            )

    return results


def print_comparison(
    players: dict[str, str],
    kills_b2: list[dict],
    kills_current: dict[str, dict[int, str]],
    labels: dict[int, str],
    xuid_to_pi: dict[str, int],
) -> None:
    """Affiche tableau comparatif b2_pi vs actuel par joueur."""
    from collections import Counter, defaultdict

    b2_by_player: dict[str, list] = defaultdict(list)
    for k in kills_b2:
        b2_by_player[k["xuid"]].append(k)

    print()
    print("=" * 80)
    print(f"MATCH {MATCH_ID[:8]}  —  b2_pi vs actuel")
    print("=" * 80)

    for xuid, gt in sorted(players.items(), key=lambda x: x[1] or x[0]):
        pi = xuid_to_pi.get(xuid, "?")
        kills = b2_by_player.get(xuid, [])
        print(f"\n  {gt or xuid}  (xuid={xuid}  pi={pi})")
        print(f"  {'-'*70}")

        # b2_pi
        wcount_b2: Counter = Counter()
        for k in kills:
            name = (
                labels.get(k["weapon_id"], f"ID:{k['weapon_id']}") if k["weapon_id"] else "INCONNU"
            )
            wcount_b2[name] += 1

        # actuel
        wcount_cur = kills_current.get(xuid, {})

        all_weapons = sorted(set(list(wcount_b2.keys()) + list(wcount_cur.keys())))
        print(f"  {'Arme':35}  {'b2_pi':>7}  {'actuel':>7}  {'diff':>6}")
        changed = False
        for w in all_weapons:
            n_b2 = wcount_b2.get(w, 0)
            n_cur = wcount_cur.get(w, 0)
            diff = n_b2 - n_cur
            marker = "  !" if diff != 0 else ""
            print(f"  {w:35}  {n_b2:>7}  {n_cur:>7}  {diff:>+6}{marker}")
            if diff != 0:
                changed = True

        unmatched = sum(1 for k in kills if k["weapon_id"] is None)
        total = len(kills)
        high_conf = sum(1 for k in kills if k["confidence"] == "high")
        print(
            f"  -> Total kills: {total}  matched: {total-unmatched}  high_conf: {high_conf}  changed: {changed}"
        )


# ── main ─────────────────────────────────────────────────────────────────────


def main() -> None:
    print(f"Match : {MATCH_ID}")

    chunks = load_chunks()
    chunks_sorted = sorted(chunks)
    timing = [(chunks[i][1], chunks[i][1] + chunks[i][2]) for i in chunks_sorted]

    with duckdb.connect(str(META_DB), read_only=True) as meta_conn:
        labels = load_weapon_labels(meta_conn)

    with duckdb.connect(str(SHARED_DB), read_only=True) as conn:
        participants, kills = load_match_data(conn)
        print(f"[data] {len(participants)} joueurs, {len(kills)} kills total")

        # Résolution pi
        xuid_to_pi = resolve_pi_map(chunks, participants)
        print("[pi_map]", {v: k for k, v in xuid_to_pi.items()})

        # Scan fire events
        print("[scan] scan_fire_events_all sur tous les chunks...")
        all_raw: list[dict] = []
        for idx in chunks_sorted:
            data, start_ms, dur_ms = chunks[idx]
            evs = scan_fire_events_all(data, start_ms, dur_ms)
            all_raw.extend(evs)
        all_raw.sort(key=lambda e: e["timestamp_ms"])
        print(f"[scan] {len(all_raw)} fire events bruts")

        # Timelines
        timeline, timeline_ns, swap_pis, _ = build_weapon_timelines(chunks)

        # b2 -> pi mapping — utilise timeline_NS (type IDs) au lieu de timeline (instance IDs)
        b2_to_pi = map_b2_to_player(all_raw, timeline_ns, timing, chunks_sorted)
        print(f"[b2_to_pi] {len(b2_to_pi)} b2_streams resolus")
        pi_distribution: dict[int, int] = {}
        for pi in b2_to_pi.values():
            pi_distribution[pi] = pi_distribution.get(pi, 0) + 1
        print(f"[b2_to_pi] distribution pi: {dict(sorted(pi_distribution.items()))}")

        # Grouper par pi
        fire_by_pi = group_events_by_pi(all_raw, b2_to_pi)
        print(
            f"[b2_to_pi] events par pi: { {pi: len(evs) for pi, evs in sorted(fire_by_pi.items())} }"
        )
        print(
            f"[b2_to_pi] joueurs sans pool: { {xuid: pi for xuid, pi in xuid_to_pi.items() if pi not in fire_by_pi} }"
        )

        # Corrélation b2_pi
        kills_b2 = correlate_b2_pi(
            kills,
            fire_by_pi,
            xuid_to_pi,
            timeline,
            swap_pis,
            timing,
            chunks_sorted,
            timeline_ns,
        )

        # Chargement résultats actuels
        conn.execute(f"ATTACH '{META_DB}' AS meta (READ_ONLY)")
        rows = conn.execute(
            "SELECT wk.xuid, COALESCE(wl.name_fr, wk.weapon_id::TEXT), COUNT(*) "
            "FROM weapon_kills wk "
            "LEFT JOIN meta.weapon_labels wl ON wl.weapon_id = wk.weapon_id "
            "WHERE wk.match_id = ? AND wk.weapon_id IS NOT NULL "
            "GROUP BY wk.xuid, wk.weapon_id, wl.name_fr",
            [MATCH_ID],
        ).fetchall()

    current: dict[str, dict[str, int]] = {}
    for xuid, wname, cnt in rows:
        current.setdefault(xuid, {})[wname] = cnt

    print_comparison(participants, kills_b2, current, labels, xuid_to_pi)

    # Stats globales b2_pi
    total = len(kills_b2)
    matched = sum(1 for k in kills_b2 if k["weapon_id"] is not None)
    high = sum(1 for k in kills_b2 if k["confidence"] == "high")
    no_match = sum(1 for k in kills_b2 if k["source"] == "no_match")
    print()
    print(
        f"GLOBAL  total={total}  matched={matched} ({matched*100//total}%)  "
        f"high_conf={high}  no_match={no_match}"
    )


if __name__ == "__main__":
    main()
