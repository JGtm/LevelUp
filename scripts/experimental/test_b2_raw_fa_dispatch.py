"""Script de test — B2 dispatch étendu à la raw FA timeline.

Hypothèse : les fire events des joueurs encodés exclusivement en raw FA
(pi=4 MrsBZ671, pi=5 Madina97294) ont des weapon_bytes avec suffix 42c9679f
qui existent dans timeline (raw FA) mais pas dans timeline_ns.
Si vrai → augmenter map_b2_to_player pour voter aussi depuis timeline (raw FA)
permettrait de dispatcher leurs events.

Usage :
    .venv/Scripts/python.exe scripts/experimental/test_b2_raw_fa_dispatch.py
"""

from __future__ import annotations

import sys
sys.path.insert(0, ".")

import glob
from collections import Counter

from src.analysis.packet_index import index_chunk, build_packet_estimator
from src.analysis.weapon_parser import (
    scan_fire_events_all,
    build_weapon_timelines,
    map_b2_to_player,
    group_events_by_pi,
    find_chunk_at_time,
)
from src.analysis._weapon_data import WEAPON_ID_MAP

MATCH_SHORT = "51556275"
COMMON_SUFFIX = bytes.fromhex("42c9679f")

PI_NAMES = {
    0: "xGoDlyKe xX",
    1: "short torso99",
    2: "JGtm",
    3: "Cali s finest33",
    4: "MrsBZ671",
    5: "Madina97294",
    6: "Danilo SBC",
    7: "MyJam22",
}

KNOWN_HEX = {k.hex() for k in WEAPON_ID_MAP.keys()}


def load_chunks(match_short: str) -> dict[int, tuple[bytes, int, int]]:
    """Charge les chunks avec les vrais timings via packet estimator."""
    paths = sorted(glob.glob(f"data/cache/film_chunks/{match_short}/chunk_*.bin"))
    if not paths:
        raise FileNotFoundError(f"Aucun chunk pour {match_short}")

    chunks: dict[int, tuple[bytes, int, int]] = {}
    # Timing réel : 90s par chunk (valeur typique)
    chunk_dur_ms = 90_000
    for i, path in enumerate(paths):
        with open(path, "rb") as f:
            data = f.read()
        chunks[i] = (data, i * chunk_dur_ms, chunk_dur_ms)
    print(f"Chargé {len(chunks)} chunks pour {match_short}")
    return chunks


def scan_with_real_ts(
    chunks: dict[int, tuple[bytes, int, int]],
) -> tuple[list[dict], list[tuple[int, int]], list[int]]:
    """Scanne les fire events avec le vrai packet estimator (comme le service)."""
    all_raw_events: list[dict] = []
    timing: list[tuple[int, int]] = []
    chunks_sorted: list[int] = []

    for idx in sorted(chunks):
        data, start_ms, dur_ms = chunks[idx]
        try:
            packets = index_chunk(data)
        except Exception:
            packets = []
        events = scan_fire_events_all(data, start_ms, dur_ms, packets=packets)
        all_raw_events.extend(events)
        timing.append((start_ms, start_ms + dur_ms))
        chunks_sorted.append(idx)

    all_raw_events.sort(key=lambda e: e["timestamp_ms"])
    return all_raw_events, timing, chunks_sorted


def map_b2_to_player_extended(
    all_fire_events: list[dict],
    timeline: dict[int, dict[int, bytes]],
    timeline_ns: dict[int, dict[int, bytes]],
    timing: list[tuple[int, int]],
    chunks_sorted: list[int],
) -> tuple[dict[int, int], dict[int, str]]:
    """Version étendue : vote depuis NS + raw FA.

    Returns:
        (b2_to_pi, b2_to_source) — source = "ns" ou "raw_fa"
    """
    b2_votes_ns: dict[int, Counter] = {}
    b2_votes_raw: dict[int, Counter] = {}

    for ev in all_fire_events:
        b2 = ev["fire_seq"]
        weapon_bytes = ev["weapon_bytes"]
        t_ms = int(ev["timestamp_ms"])
        chunk_idx = find_chunk_at_time(chunks_sorted, timing, t_ms)

        # Vote NS (comportement actuel)
        chunk_ns = timeline_ns.get(chunk_idx, {})
        for pi, pi_wid in chunk_ns.items():
            if pi_wid == weapon_bytes:
                b2_votes_ns.setdefault(b2, Counter())[pi] += 1
                break

        # Vote raw FA (nouveau)
        chunk_raw = timeline.get(chunk_idx, {})
        for pi, pi_wid in chunk_raw.items():
            if pi_wid == weapon_bytes:
                b2_votes_raw.setdefault(b2, Counter())[pi] += 1
                break

    # Fusion : NS prioritaire, raw FA en fallback
    b2_to_pi: dict[int, int] = {}
    b2_to_source: dict[int, str] = {}

    all_b2 = set(b2_votes_ns) | set(b2_votes_raw)
    for b2 in all_b2:
        if b2 in b2_votes_ns and b2_votes_ns[b2]:
            b2_to_pi[b2] = b2_votes_ns[b2].most_common(1)[0][0]
            b2_to_source[b2] = "ns"
        elif b2 in b2_votes_raw and b2_votes_raw[b2]:
            b2_to_pi[b2] = b2_votes_raw[b2].most_common(1)[0][0]
            b2_to_source[b2] = "raw_fa"

    return b2_to_pi, b2_to_source


def main() -> None:
    chunks = load_chunks(MATCH_SHORT)

    print("\n── 1. Scan fire events (packet estimator réel) ──")
    all_raw_events, timing, chunks_sorted = scan_with_real_ts(chunks)
    print(f"Total raw fire events : {len(all_raw_events)}")
    raw_by_pi = Counter(ev["player_index"] for ev in all_raw_events)
    for pi, n in sorted(raw_by_pi.items()):
        print(f"  pi={pi} ({PI_NAMES.get(pi, '?')}): {n}")

    print("\n── 2. Build timelines ──")
    timeline, timeline_ns, swap_pis, _ = build_weapon_timelines(chunks)
    ns_pis: set[int] = set()
    raw_pis: set[int] = set()
    for d in timeline_ns.values():
        ns_pis.update(d.keys())
    for d in timeline.values():
        raw_pis.update(d.keys())
    print(f"NS timeline  couvre pi : {sorted(ns_pis)}")
    print(f"Raw FA timeline couvre pi : {sorted(raw_pis)}")

    print("\n── 3. Dispatch B2 original (NS uniquement) ──")
    b2_to_pi_orig = map_b2_to_player(all_raw_events, timeline_ns, timing, chunks_sorted)
    fire_orig = group_events_by_pi(all_raw_events, b2_to_pi_orig)
    dispatched_orig = sum(len(v) for v in fire_orig.values())
    print(f"b2 résolus : {len(b2_to_pi_orig)}")
    for pi in range(8):
        n = len(fire_orig.get(pi, []))
        print(f"  pi={pi} ({PI_NAMES.get(pi, '?')}): {n} events")
    print(f"Dropped : {len(all_raw_events) - dispatched_orig} / {len(all_raw_events)}")

    print("\n── 4. Analyse des fire events droppés (weapon_bytes) ──")
    resolved_b2 = set(b2_to_pi_orig)
    dropped_events = [ev for ev in all_raw_events if ev["fire_seq"] not in resolved_b2]
    wb_counter = Counter(ev["weapon_bytes"].hex() for ev in dropped_events)
    print(f"Events droppés : {len(dropped_events)}")
    print("Top weapon_bytes droppés :")
    for wbhex, cnt in wb_counter.most_common(10):
        flag = "CONNU" if wbhex in KNOWN_HEX else "inconnu"
        suffix = wbhex[8:] if len(wbhex) == 16 else "?"
        print(f"  {wbhex}  x{cnt:4d}  [{flag}]  suffix={suffix}")

    print("\n── 5. Les weapon_bytes droppés sont-ils dans raw FA timeline ? ──")
    for wbhex, cnt in wb_counter.most_common(10):
        wb = bytes.fromhex(wbhex)
        present_in_raw = []
        for chunk_idx, chunk_state in timeline.items():
            for pi, pi_wb in chunk_state.items():
                if pi_wb == wb:
                    present_in_raw.append((chunk_idx, pi))
        if present_in_raw:
            pis_found = sorted({p for _, p in present_in_raw})
            print(f"  {wbhex} → raw FA pi={pis_found}  (chunks: {sorted({c for c,_ in present_in_raw[:3]})}...)")
        else:
            print(f"  {wbhex} → ABSENT de raw FA timeline")

    print("\n── 6. Dispatch B2 étendu (NS + raw FA) ──")
    b2_to_pi_ext, b2_to_source = map_b2_to_player_extended(
        all_raw_events, timeline, timeline_ns, timing, chunks_sorted
    )
    fire_ext = group_events_by_pi(all_raw_events, b2_to_pi_ext)
    dispatched_ext = sum(len(v) for v in fire_ext.values())
    print(f"b2 résolus : {len(b2_to_pi_ext)} (NS: {sum(1 for s in b2_to_source.values() if s=='ns')}, raw_FA: {sum(1 for s in b2_to_source.values() if s=='raw_fa')})")
    for pi in range(8):
        n_orig = len(fire_orig.get(pi, []))
        n_ext = len(fire_ext.get(pi, []))
        delta = f"+{n_ext - n_orig}" if n_ext > n_orig else str(n_ext - n_orig)
        marker = " ← NOUVEAU" if n_ext > 0 and n_orig == 0 else ""
        print(f"  pi={pi} ({PI_NAMES.get(pi, '?')}): {n_ext} events ({delta}){marker}")
    print(f"Dropped : {len(all_raw_events) - dispatched_ext} / {len(all_raw_events)}")

    print("\n── 7. Pourquoi MyJam22 (pi=7, MA40) n'est pas résolu ? ──")
    # MA40 = 48c19d2d42c9679f — 116 events droppés, en théorie MyJam22
    MA40 = bytes.fromhex("48c19d2d42c9679f")
    ma40_events = [ev for ev in all_raw_events if ev["weapon_bytes"] == MA40]
    print(f"Events MA40 ({MA40.hex()}) : {len(ma40_events)}")

    # Pour chaque event MA40, voir ce que la NS timeline dit à ce chunk
    vote_counter: Counter = Counter()
    for ev in ma40_events[:20]:  # premiers 20 pour analyse
        t_ms = int(ev["timestamp_ms"])
        chunk_idx = find_chunk_at_time(chunks_sorted, timing, t_ms)
        chunk_ns = timeline_ns.get(chunk_idx, {})
        matches = [(pi, wb.hex()) for pi, wb in chunk_ns.items() if wb == MA40]
        vote_counter.update([str(m) for m in matches])
        if not matches:
            # Voir ce que timeline_ns contient pour ce chunk
            contents = {pi: wb.hex() for pi, wb in chunk_ns.items()}
            print(f"  t={t_ms}ms chunk={chunk_idx}: aucun pi NS avec MA40 | NS={contents}")

    print(f"Votes NS pour MA40 events (top 10) : {vote_counter.most_common(10)}")

    # Checker : pi=6 (Danilo SBC) tient-il aussi MA40 dans NS ?
    print("\nChunks où pi=6 ET pi=7 tiennent MA40 simultanément (NS) :")
    collisions = 0
    for chunk_idx, chunk_ns in timeline_ns.items():
        pi6_wb = chunk_ns.get(6)
        pi7_wb = chunk_ns.get(7)
        if pi6_wb == MA40 and pi7_wb == MA40:
            collisions += 1
    print(f"  Chunks en collision (pi=6 ET pi=7 = MA40) : {collisions}")

    print("\nChunks où pi=7 tient MA40 (NS) :")
    pi7_ma40_chunks = [c for c, d in timeline_ns.items() if d.get(7) == MA40]
    print(f"  {pi7_ma40_chunks[:10]}... ({len(pi7_ma40_chunks)} chunks)")

    print("\nChunks où pi=6 tient MA40 (NS) :")
    pi6_ma40_chunks = [c for c, d in timeline_ns.items() if d.get(6) == MA40]
    print(f"  {pi6_ma40_chunks[:10]}... ({len(pi6_ma40_chunks)} chunks)")

    print("\n── 8. Vote b2→pi pour les b2 contenant MA40 ──")
    # Grouper les events MA40 par b2 et voir leurs votes
    from collections import defaultdict
    b2_ma40_map: dict[int, list[dict]] = defaultdict(list)
    for ev in ma40_events:
        b2_ma40_map[ev["fire_seq"]].append(ev)

    print(f"b2 distincts portant MA40 : {len(b2_ma40_map)}")
    for b2, evts in sorted(b2_ma40_map.items())[:15]:
        resolved_pi = b2_to_pi_orig.get(b2, "DROP")
        print(f"  b2={b2:3d} → résolu={resolved_pi} ({PI_NAMES.get(resolved_pi,'?') if isinstance(resolved_pi, int) else resolved_pi})  events={len(evts)}")


if __name__ == "__main__":
    main()
