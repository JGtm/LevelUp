"""
inv131 — Validation de map_b2_to_player() sur le match 147ffd4d.

Objectif : vérifier que la fonction map_b2_to_player() attribue correctement
les b2_stream aux player_index en croisant avec la Formula A timeline.

Usage :
    python scripts/experimental/inv131_validate_map_b2.py
"""

from __future__ import annotations

import sys
from collections import Counter
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

from src.analysis.packet_index import index_chunk
from src.analysis.weapon_parser import (
    build_weapon_timeline,
    build_weapon_timeline_ns,
    map_b2_to_player,
    scan_fire_events,
)

MATCH_ID = "147ffd4d-3d1d-4b90-a46d-5570009f8c36"
CHUNKS_DIR = ROOT / "data" / "investigation" / "chunks" / MATCH_ID[:8]


def load_chunks() -> dict[int, tuple[bytes, int, int]]:
    """Charge tous les chunks disponibles depuis CHUNKS_DIR.

    Les fichiers sont nommés chunk_NN.bin ; on utilise NN comme chunk_idx.
    La durée est estimée à 2000 ms par chunk, l'heure de début est NN × 2000.
    """
    chunks: dict[int, tuple[bytes, int, int]] = {}
    for path in sorted(CHUNKS_DIR.glob("chunk_*.bin")):
        idx = int(path.stem.split("_")[1])
        data = path.read_bytes()
        start_ms = idx * 2000
        dur_ms = 2000
        chunks[idx] = (data, start_ms, dur_ms)
    return chunks


def scan_all_fire_events(chunks: dict[int, tuple[bytes, int, int]]) -> list[dict]:
    """Scanne tous les fire events avec pi=1 (capture universelle, inv#131)."""
    all_events: list[dict] = []
    for idx in sorted(chunks):
        data, start_ms, dur_ms = chunks[idx]
        packets = index_chunk(data)
        evts = scan_fire_events(data, 1, start_ms, dur_ms, packets=packets)
        all_events.extend(evts)
    all_events.sort(key=lambda e: e["timestamp_ms"])
    return all_events


def main() -> None:
    print(f"\n=== Validation map_b2_to_player — match {MATCH_ID[:8]} ===\n")

    # 1. Charger les chunks
    chunks = load_chunks()
    print(f"Chunks chargés : {len(chunks)} (idx {min(chunks)}..{max(chunks)})")

    # 2. Construire la timeline Formula A (raw, pour info) et la timeline NS (TYPE IDs)
    timeline_raw, swap_pis, timing = build_weapon_timeline(chunks)
    timeline_ns = build_weapon_timeline_ns(chunks)
    chunks_sorted = sorted(chunks)
    total_pi_snapshots_raw = sum(len(v) for v in timeline_raw.values())
    total_pi_snapshots_ns = sum(len(v) for v in timeline_ns.values())
    print(f"Timeline Formula A raw : {len(timeline_raw)} chunks, {total_pi_snapshots_raw} snapshots (instance handles)")
    print(f"Timeline Formula A NS  : {len(timeline_ns)} chunks, {total_pi_snapshots_ns} snapshots (TYPE IDs)")

    # 3. Scanner tous les fire events
    all_events = scan_all_fire_events(chunks)
    print(f"Fire events (pi=1 capture) : {len(all_events)} total")

    if not all_events:
        print("AUCUN fire event trouvé — vérifier les chunks.")
        return

    # 4. Appeler map_b2_to_player avec la timeline NS (TYPE IDs)
    b2_to_pi = map_b2_to_player(all_events, timeline_ns, timing, chunks_sorted)
    print(f"\nmap_b2_to_player : {len(b2_to_pi)} b2 uniques → pi résolu\n")

    # 5. Afficher la table b2 → pi avec nombre d'events
    b2_counts = Counter(ev["fire_seq"] for ev in all_events)
    resolved = sum(1 for ev in all_events if ev["fire_seq"] in b2_to_pi)
    unresolved = len(all_events) - resolved

    print(f"{'b2':>6}  {'pi':>3}  {'events':>7}  weapon_bytes (dernière vue)")
    print("-" * 50)
    for b2, pi in sorted(b2_to_pi.items()):
        count = b2_counts[b2]
        # Trouver le dernier weapon_bytes associé à ce b2
        last_wb = next(
            (ev["weapon_bytes"].hex() for ev in reversed(all_events) if ev["fire_seq"] == b2),
            "??",
        )
        print(f"  0x{b2:02x}  {pi:>3}  {count:>7}  {last_wb}")

    print(f"\nTotal resolus  : {resolved}/{len(all_events)} ({100*resolved//len(all_events)}%)")
    print(f"Total non-resolus : {unresolved}")

    # 6. Distribution par pi
    pi_counts: dict[int, int] = {}
    for ev in all_events:
        pi = b2_to_pi.get(ev["fire_seq"])
        if pi is not None:
            pi_counts[pi] = pi_counts.get(pi, 0) + 1

    print("\nFire events par player_index :")
    for pi in sorted(pi_counts):
        bar = "█" * (pi_counts[pi] // 5)
        print(f"  pi={pi}  {pi_counts[pi]:>5}  {bar}")

    # 7. Sanity check : les b2 non résolus
    unresolved_b2s = {ev["fire_seq"] for ev in all_events if ev["fire_seq"] not in b2_to_pi}
    if unresolved_b2s:
        print(f"\nB2 non résolus ({len(unresolved_b2s)}) : {sorted(unresolved_b2s)}")
        print("→ Ces b2 n'ont aucun match dans la Formula A timeline.")


if __name__ == "__main__":
    main()
