"""
inv131 -- Diagnostic attribution player_index dans les fire events Section 2.

Problème établi (thought_log 2026-03-13) :
  scan_fire_events(pi=1) capte TOUS les fire events du match (1177 ≈ Σ acurtis=1178).
  scan_fire_events(pi≠1) retourne 0 résultats.
  → byte[1] = 0x26 = (pi=1) est un marqueur structurel fixe, pas un "player_index".

Question : où est réellement encodé le player_index dans la structure nibble-shiftée
d'un fire event, et comment acurtis les attribue-t-il par joueur ?

Méthode :
  1. Scanner tous les fire events avec pi=1 sur un chunk (tous détectés)
  2. Pour chaque fire event, extraire un contexte large (±32 bytes) dans la couche
     nibble-shiftée
  3. Chercher des patterns qui corrèlent avec le joueur réel
     a. fire_counter (shot counter 0-127) — unique par arme dans le chunk ?
     b. b2_stream (byte[2]) — discriminateur dual-stream, peut-il être par joueur ?
     c. b5_correlated (byte[5]) — corrélé avec b2_stream
     d. Bytes hors-struct (avant le lead byte, après le weapon_id)
  4. Comparer les fire_counter entre joueurs : si les compteurs recommencent à 0
     par joueur, on peut les regrouper.
  5. Essayer decode PI depuis les bits raw autour du marqueur (bits avant le lead byte)

Corpus : match 147ffd4d (Super Fiesta Bazaar, 10 joueurs)
  Chunks disponibles : 03..27
  Acurtis baseline (from thought_log) : total 1178 fire events répartis sur 9 joueurs

Usage :
  python scripts/experimental/inv131_fire_event_player_attribution.py
"""

from __future__ import annotations

import sys
from collections import Counter, defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

from bitstring import Bits
from rich.console import Console
from rich.table import Table

from src.analysis._weapon_data import WEAPON_IDS_INT, WEAPON_INT_TO_NAME
from src.analysis.packet_index import (
    PacketType,
    detect_pi_from_metadata,
    extract_metadata_payload,
    index_chunk,
)
from src.analysis.weapon_parser import (
    COMMON_WEAPON_SUFFIX,
    _WEAPON_BIT_OFFSET,
    _build_marker,
)

# ── Config ─────────────────────────────────────────────────────────────────────

MATCH_ID = "147ffd4d-3d1d-4b90-a46d-5570009f8c36"
CHUNKS_DIR = ROOT / "data" / "investigation" / "chunks" / MATCH_ID[:8]
CONTEXT_BYTES = 16  # bytes de contexte dans la couche NS avant/après le fire event struct

console = Console()


# ── Helpers nibble-shift ───────────────────────────────────────────────────────


def nibble_shift(data: bytes) -> bytes:
    """Retourne la couche nibble-shiftée (décalage de 4 bits) des données brutes."""
    return bytes(
        (data[i] << 4 | data[i + 1] >> 4) & 0xFF for i in range(len(data) - 1)
    )


def nibble_shift_offset_in_raw(ns_byte_offset: int) -> int:
    """Offset en bits dans les données brutes correspondant à ns_byte_offset dans NS.

    NS[n] = bits bruts [n*8+4 .. n*8+11], donc bit brut = n*8 + 4.
    """
    return ns_byte_offset * 8 + 4


# ── Scanner principal ──────────────────────────────────────────────────────────


def scan_fire_events_full_context(chunk_data: bytes) -> list[dict]:
    """Scanne tous les fire events (pi=1) et extrait un contexte large.

    Retourne une liste de dicts avec :
      - raw_bit_pos : position du marqueur dans les bits bruts
      - ns_byte_pos : byte offset dans la couche NS (ns_byte_pos*8+4 = raw_bit_pos)
      - b0_lead     : lead byte (byte[0] dans NS)
      - b1          : byte[1] = 0x26 pour pi=1
      - b2_stream   : byte[2] dual-stream discriminator
      - b3_filter   : byte[3] (doit être 0x40-0x43)
      - fire_counter : byte[4] shot counter
      - b5           : byte[5] corrélé à b2_stream
      - weapon_bytes : 8 bytes weapon_id
      - weapon_name  : nom résolu
      - post4        : 4 bytes après weapon_id (byte[14..17])
      - pre_context  : bytes NS avant le lead byte (context_bytes)
      - post_context : bytes NS après le weapon_id struct
    """
    bits = Bits(bytes=chunk_data)
    marker = _build_marker(1)  # pi=1 = marqueur fixe pour tous les fire events
    total_bits = len(bits)
    ns_data = nibble_shift(chunk_data)
    ns_len = len(ns_data)

    events = []
    seen: set[tuple] = set()

    for raw_bit_pos in bits.findall(marker, bytealigned=False):
        event_start = raw_bit_pos + 3  # bytes[2..] après le marqueur 11-bit
        weapon_start = event_start + _WEAPON_BIT_OFFSET

        if weapon_start + 64 > total_bits:
            continue

        weapon_int = bits[weapon_start : weapon_start + 64].uint
        weapon_bytes = weapon_int.to_bytes(8, byteorder="big")
        if weapon_int not in WEAPON_IDS_INT and weapon_bytes[4:] != COMMON_WEAPON_SUFFIX:
            continue

        # Déduplication par (fire_counter, weapon_bytes)
        fire_counter = (
            bits[event_start + 24 : event_start + 32].uint
            if event_start + 32 <= total_bits
            else 0
        )
        dedup_key = (fire_counter, weapon_bytes)
        if dedup_key in seen:
            continue
        seen.add(dedup_key)

        # Le marqueur couvre les bits 5..15 du fire event dans NS
        # ns_byte offset = (raw_bit_pos - 5) // 8 idéalement (position du lead byte NS)
        # Plus précis : (raw_bit_pos - 4) / 8 = début du byte NS qui contient bit 4 des données brutes
        # raw_bit_pos = ns_byte * 8 + 4 + 5 (où +5 = les 5 premiers bits du lead byte avant les 3 LSBs)
        # → ns_byte_of_marker_start = (raw_bit_pos - 4) // 8
        ns_byte_lead = (raw_bit_pos - 5)  # bit position du début du lead byte dans NS (bits bruts = ns_byte*8 + 4)
        ns_idx = (ns_byte_lead - 4) // 8  # byte index dans NS

        # Lire les bytes NS si l'index est valide
        def ns_byte(offset: int) -> int:
            idx = ns_idx + offset
            return ns_data[idx] if 0 <= idx < ns_len else 0

        b0_lead = ns_byte(0)
        b1 = ns_byte(1)
        b2_stream = ns_byte(2)
        b3_filter = ns_byte(3)
        b4_counter = ns_byte(4)
        b5 = ns_byte(5)
        wid_bytes = bytes(ns_byte(6 + i) for i in range(8))
        post4 = bytes(ns_byte(14 + i) for i in range(8))

        # Contexte avant (bytes NS précédant le lead byte)
        ns_pre_start = max(0, ns_idx - CONTEXT_BYTES)
        pre_context = ns_data[ns_pre_start : ns_idx]

        # Contexte après (bytes NS après le struct de base ~18 bytes)
        ns_post_end = min(ns_len, ns_idx + 22 + CONTEXT_BYTES)
        post_context = ns_data[ns_idx + 22 : ns_post_end]

        weapon_name = WEAPON_INT_TO_NAME.get(weapon_int, f"UNK({weapon_bytes.hex()})")

        events.append({
            "raw_bit_pos": raw_bit_pos,
            "ns_idx": ns_idx,
            "b0_lead": b0_lead,
            "b1": b1,
            "b2_stream": b2_stream,
            "b3_filter": b3_filter,
            "fire_counter": b4_counter,
            "b5": b5,
            "weapon_bytes": weapon_bytes,
            "weapon_name": weapon_name,
            "weapon_int": weapon_int,
            "post4": post4,
            "pre_context": pre_context,
            "post_context": post_context,
        })

    return sorted(events, key=lambda e: e["raw_bit_pos"])


# ── Analyse patterns ───────────────────────────────────────────────────────────


def analyze_b2_b5_patterns(events: list[dict]) -> None:
    """Analyse b2_stream et b5 pour voir s'ils peuvent encoder le player_index."""
    console.rule("[cyan]Analyse b2_stream / b5 / b3_filter")

    b2_counts = Counter(e["b2_stream"] for e in events)
    b3_counts = Counter(e["b3_filter"] for e in events)
    b5_counts = Counter(e["b5"] for e in events)
    b1_counts = Counter(e["b1"] for e in events)

    table = Table(show_header=True, header_style="bold")
    table.add_column("Byte")
    table.add_column("Valeurs distinctes (count)")
    table.add_column("Distribution")

    for name, ctr in [
        ("b1 (supposé pi)", b1_counts),
        ("b2_stream", b2_counts),
        ("b3_filter", b3_counts),
        ("b5_correlated", b5_counts),
    ]:
        vals = sorted(ctr.items(), key=lambda x: -x[1])[:10]
        table.add_row(
            name,
            str(len(ctr)),
            " | ".join(f"0x{v:02x}×{c}" for v, c in vals),
        )
    console.print(table)


def analyze_fire_counter_gaps(events: list[dict]) -> None:
    """Cherche des séquences fire_counter cohérentes — pourraient indiquer des joueurs."""
    console.rule("[cyan]Analyse fire_counter — groupes par arme")

    by_weapon: dict[bytes, list[int]] = defaultdict(list)
    for e in events:
        by_weapon[e["weapon_bytes"]].append(e["fire_counter"])

    table = Table(show_header=True, header_style="bold")
    table.add_column("Arme")
    table.add_column("N events")
    table.add_column("Counters (premiers 30)")
    table.add_column("Resets détectés (counter décroît)")

    for wbytes, counters in sorted(by_weapon.items(), key=lambda x: -len(x[1]))[:12]:
        wname = WEAPON_INT_TO_NAME.get(
            int.from_bytes(wbytes, "big"), wbytes.hex()[:16]
        )
        resets = sum(1 for i in range(1, len(counters)) if counters[i] < counters[i - 1])
        counters_str = " ".join(f"{c}" for c in counters[:30])
        table.add_row(wname, str(len(counters)), counters_str, str(resets))
    console.print(table)


def analyze_pre_context_patterns(events: list[dict]) -> None:
    """Cherche des patterns stables dans les bytes précédant le fire event.

    Hypothèse : le player_index est peut-être encodé dans le paquet FRAME
    juste avant le fire event.
    """
    console.rule("[cyan]Analyse contexte pré-fire (bytes NS avant lead byte)")

    # Regrouper par arme + regarder les derniers 4 bytes du pre_context
    pre_4bytes: Counter[bytes] = Counter()
    for e in events:
        if len(e["pre_context"]) >= 4:
            pre_4bytes[bytes(e["pre_context"][-4:])] += 1

    console.print(f"Patterns pré-fire distincts sur 4 bytes : [bold]{len(pre_4bytes)}[/bold]")

    table = Table(show_header=True, header_style="bold")
    table.add_column("Pre-4B (hex)")
    table.add_column("Count")
    for k, c in sorted(pre_4bytes.items(), key=lambda x: -x[1])[:15]:
        table.add_row(k.hex(), str(c))
    console.print(table)


def analyze_b2_as_player_index(events: list[dict]) -> None:
    """Hypothèse H1 : b2_stream encode le player_index.

    On s'attend à ce que b2_stream soit constant pour un joueur donné sur un match.
    Vérifiez en regroupant les fire_counter par b2_stream.
    """
    console.rule("[cyan]Hypothèse H1 : b2_stream = player_index ?")

    by_b2: dict[int, list] = defaultdict(list)
    for e in events:
        by_b2[e["b2_stream"]].append(e)

    table = Table(show_header=True, header_style="bold")
    table.add_column("b2_stream (hex)")
    table.add_column("N events")
    table.add_column("Armes distinctes")
    table.add_column("Counter range")
    table.add_column("Resets")

    for b2, evts in sorted(by_b2.items())[:20]:
        counters = [e["fire_counter"] for e in evts]
        weapons = {e["weapon_name"] for e in evts}
        resets = sum(1 for i in range(1, len(counters)) if counters[i] < counters[i - 1] - 4)
        table.add_row(
            f"0x{b2:02x}",
            str(len(evts)),
            ", ".join(sorted(weapons)[:3]),
            f"{min(counters)}..{max(counters)}",
            str(resets),
        )
    console.print(table)


def analyze_post4_as_player_index(events: list[dict]) -> None:
    """Hypothèse H2 : bytes après weapon_id contiennent le player_index.

    Selon certaines spécifications acurtis, des bits supplémentaires après
    le weapon_id pourraient encoder le joueur.
    """
    console.rule("[cyan]Hypothèse H2 : post4 bytes (après weapon_id) encodent le joueur")

    post4_b0: Counter[int] = Counter(e["post4"][0] for e in events if e["post4"])
    post4_b1: Counter[int] = Counter(e["post4"][1] for e in events if len(e["post4"]) > 1)

    console.print(f"post4[0] valeurs distinctes : {len(post4_b0)}")
    console.print("post4[0] distribution :", {f"0x{k:02x}": v for k, v in sorted(post4_b0.items())})
    console.print(f"post4[1] valeurs distinctes : {len(post4_b1)}")
    console.print("post4[1] distribution :", {f"0x{k:02x}": v for k, v in sorted(post4_b1.items())})


def analyze_pi_candidates_in_context(events: list[dict], pi_to_xuid: dict[int, int]) -> None:
    """Cherche les player_indices connus dans les bytes de contexte des fire events."""
    console.rule("[cyan]Cherche pi connus (METADATA) dans le contexte pré/post-fire")

    known_pi_vals = set(pi_to_xuid.keys())
    console.print(f"Player indices connus : {sorted(known_pi_vals)}")

    # Cherche les bytes qui correspondent à (pi << 5) | 0x06 dans chaque contexte
    for pi in sorted(known_pi_vals):
        pi_byte = (pi << 5) | 0x06
        count_pre = sum(1 for e in events if pi_byte in e["pre_context"])
        count_post = sum(1 for e in events if pi_byte in e["post4"])
        if count_pre + count_post > 0:
            console.print(
                f"  pi={pi} → 0x{pi_byte:02x} : pré={count_pre}/{len(events)}, "
                f"post4={count_post}/{len(events)}"
            )


def analyze_unique_fire_counter_per_b2(events: list[dict]) -> None:
    """Vérifie si (b2_stream, fire_counter) est unique — indicateur fort que b2 = joueur."""
    console.rule("[cyan]Unicité (b2_stream, fire_counter)")

    pairs: Counter[tuple[int, int]] = Counter(
        (e["b2_stream"], e["fire_counter"]) for e in events
    )
    collisions = sum(1 for c in pairs.values() if c > 1)
    console.print(
        f"Paires (b2, fc) uniques : {len(pairs)}, collisions : {collisions}, "
        f"events total : {len(events)}"
    )
    if collisions == 0:
        console.print("[bold green]✓ (b2_stream, fire_counter) est une clé unique — b2 peut encoder le joueur.")
    else:
        console.print("[bold red]✗ Collisions détectées — b2_stream ne suffit pas seul.")


def dump_first_n_events(events: list[dict], n: int = 20) -> None:
    """Affiche les N premiers fire events avec leur contexte complet."""
    console.rule("[cyan]Dump des premiers fire events")

    table = Table(show_header=True, header_style="bold", expand=True)
    table.add_column("ns_idx")
    table.add_column("b0")
    table.add_column("b1")
    table.add_column("b2")
    table.add_column("b3")
    table.add_column("fc")
    table.add_column("b5")
    table.add_column("weapon")
    table.add_column("post4")
    table.add_column("pre[-4:]")

    for e in events[:n]:
        pre4 = e["pre_context"][-4:] if len(e["pre_context"]) >= 4 else e["pre_context"]
        table.add_row(
            str(e["ns_idx"]),
            f"0x{e['b0_lead']:02x}",
            f"0x{e['b1']:02x}",
            f"0x{e['b2_stream']:02x}",
            f"0x{e['b3_filter']:02x}",
            str(e["fire_counter"]),
            f"0x{e['b5']:02x}",
            e["weapon_name"][:20],
            e["post4"][:4].hex(),
            pre4.hex(),
        )
    console.print(table)


# ── Main ───────────────────────────────────────────────────────────────────────


def main() -> None:
    console.rule(f"[bold yellow]inv131 — Attribution joueur fire events — {MATCH_ID[:8]}")

    # Charger chunk_07 (dense en kills selon inv87)
    chunk_path = CHUNKS_DIR / "chunk_07.bin"
    if not chunk_path.exists():
        console.print(f"[red]Chunk non trouvé : {chunk_path}")
        return

    chunk_data = chunk_path.read_bytes()
    console.print(f"Chunk 07 : {len(chunk_data):,} bytes")

    # Résoudre les player_indices via METADATA
    packets = index_chunk(chunk_data)
    metadata = extract_metadata_payload(chunk_data, packets)
    pi_to_xuid: dict[int, int] = {}
    if metadata is not None:
        pi_to_xuid = detect_pi_from_metadata(metadata, {})
        console.print(f"METADATA player_indices : {dict(sorted(pi_to_xuid.items()))}")
    else:
        console.print("[yellow]Pas de METADATA dans chunk_07")

    # Compter les paquets FRAME
    n_frames = sum(1 for p in packets if p.type == PacketType.FRAME)
    console.print(f"Paquets FRAME dans chunk_07 : {n_frames}")

    # Scanner tous les fire events
    events = scan_fire_events_full_context(chunk_data)
    console.print(f"\nFire events détectés (pi=1 marker) : [bold]{len(events)}[/bold]")
    console.print()

    # Analyses
    dump_first_n_events(events, n=25)
    analyze_b2_b5_patterns(events)
    analyze_b2_as_player_index(events)
    analyze_unique_fire_counter_per_b2(events)
    analyze_fire_counter_gaps(events)
    analyze_post4_as_player_index(events)
    analyze_pre_context_patterns(events)
    analyze_pi_candidates_in_context(events, pi_to_xuid)

    # Analyse multi-chunk pour voir si b2_stream est stable par joueur
    console.rule("[cyan]Analyse multi-chunk (03..10) — stabilité b2_stream")
    multi_b2_weapon: dict[int, Counter] = defaultdict(Counter)
    multi_b2_count: Counter[int] = Counter()

    for cidx in range(3, 11):
        cp = CHUNKS_DIR / f"chunk_{cidx:02d}.bin"
        if not cp.exists():
            continue
        evts = scan_fire_events_full_context(cp.read_bytes())
        for e in evts:
            multi_b2_weapon[e["b2_stream"]][e["weapon_name"]] += 1
            multi_b2_count[e["b2_stream"]] += 1

    table = Table(show_header=True, header_style="bold")
    table.add_column("b2_stream")
    table.add_column("Total events")
    table.add_column("Armes dominantes")

    for b2, total in sorted(multi_b2_count.items()):
        weapons = sorted(multi_b2_weapon[b2].items(), key=lambda x: -x[1])[:3]
        table.add_row(
            f"0x{b2:02x}",
            str(total),
            " | ".join(f"{w}×{c}" for w, c in weapons),
        )
    console.print(table)


if __name__ == "__main__":
    main()
