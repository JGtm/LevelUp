"""Recherche empirique du bit offset du player_index dans les fire events.

Acurtis confirme :
- Marqueur FIXE 0b10100100110 (11 bits) pour TOUS les joueurs
- player_index (5 bits, valeurs 0-31) est dans la structure de l'event,
  quelque part dans les 40 bits avant le weapon_id
- L'offset exact est inconnu et doit être trouvé empiriquement

Méthode :
1. Scanner avec le marqueur fixe -> tous les fire events
2. Pour chaque kill connu (avec xuid->pi via PLAYER_METADATA),
   trouver le fire event le plus proche dans la fenêtre temporelle
3. Tester chaque offset possible (0-50 bits) et voter pour celui
   qui correspond au pi du tueur
4. L'offset avec le plus de votes est le bon

Usage :
    python scripts/exp_find_pi_offset.py
"""

from __future__ import annotations

import json
import sys
from collections import Counter, defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

import duckdb
from bitstring import Bits

from src.analysis._weapon_data import WEAPON_IDS_INT
from src.analysis._weapon_scanners import (
    _WEAPON_BIT_OFFSET,
    COMMON_WEAPON_SUFFIX,
    estimate_ts_frames,
)
from src.analysis.packet_index import (
    detect_pi_from_metadata,
    extract_metadata_payload,
    index_chunk,
)
from src.analysis.weapon_parser import detect_player_indices

MATCH_ID = "82f3af9f-c0fa-477b-be9b-df240d62305d"
CACHE_DIR = ROOT / "data/cache/film_chunks" / MATCH_ID[:8]
MANIFEST = ROOT / "data/cache/film_manifests" / f"{MATCH_ID[:8]}.json"
SHARED_DB = ROOT / "data/warehouse/shared_matches_v2.duckdb"

# Marqueur fixe acurtis (= _build_marker(1) dans le code actuel)
FIXED_MARKER = Bits(bin="0b10100100110")  # 11 bits
FIXED_MARKER_LEN = len(FIXED_MARKER)  # 11

KILL_WINDOW_MS = 2000  # ms avant le kill pour chercher un fire event

# Offsets à tester (bits depuis 2 bases différentes)
# Base A : position + 3  (comme le code actuel, "event_start" = après les 3 premiers bits)
# Base B : position + 11 (après le marqueur complet)
MAX_OFFSET = 50  # tester offset 0..MAX_OFFSET-1 pour chaque base


# ── Helpers ──────────────────────────────────────────────────────────────────


def load_chunks() -> dict[int, tuple[bytes, int, int]]:
    manifest = json.loads(MANIFEST.read_text())
    idx_to_meta = {c["index"]: c for c in manifest["chunks"] if c["chunk_type"] == 2}
    chunks: dict[int, tuple[bytes, int, int]] = {}
    for bin_file in sorted(CACHE_DIR.glob("chunk_*.bin")):
        idx = int(bin_file.stem.split("_")[1])
        if idx in idx_to_meta:
            meta = idx_to_meta[idx]
            chunks[idx] = (bin_file.read_bytes(), meta["start_ms"], meta["duration_ms"])
    print(f"[chunks] {len(chunks)} chunks (idx {min(chunks)}-{max(chunks)})")
    return chunks


def resolve_pi_map(chunks: dict) -> dict[str, int]:
    """Resout xuid -> pi via PLAYER_METADATA (fiable a 100%)."""
    with duckdb.connect(str(SHARED_DB), read_only=True) as conn:
        rows = conn.execute(
            "SELECT mp.xuid FROM match_participants mp "
            "WHERE mp.match_id = ? AND mp.xuid NOT LIKE 'bid(%'",
            (MATCH_ID,),
        ).fetchall()
    participants_xuids = [r[0] for r in rows]
    xuid_int_map = {int(x): x for x in participants_xuids if x.isdigit()}

    first_idx = min(chunks)
    first_data, _, _ = chunks[first_idx]
    first_packets = index_chunk(first_data)

    try:
        payload = extract_metadata_payload(first_data, first_packets)
        if payload:
            pi_to_xuid = detect_pi_from_metadata(payload, xuid_int_map)
            if pi_to_xuid:
                missing = {x: g for x, g in xuid_int_map.items() if x not in pi_to_xuid.values()}
                if missing:
                    pov = detect_player_indices(first_data, missing)
                    pi_to_xuid.update(pov)
                xuid_to_pi = {str(v): k for k, v in pi_to_xuid.items()}
                print(f"[pi_map] METADATA OK -> {len(xuid_to_pi)} joueurs")
                for xuid, pi in sorted(xuid_to_pi.items(), key=lambda x: x[1]):
                    print(f"  pi={pi} xuid={xuid}")
                return xuid_to_pi
    except Exception as exc:
        print(f"[pi_map] metadata failed: {exc}")

    pi_to_xuid = detect_player_indices(first_data, xuid_int_map)
    return {str(v): k for k, v in pi_to_xuid.items()}


def load_kills() -> list[dict]:
    with duckdb.connect(str(SHARED_DB), read_only=True) as conn:
        rows = conn.execute(
            "SELECT xuid, time_ms FROM highlight_events "
            "WHERE match_id = ? AND event_type = 'kill' "
            "ORDER BY time_ms",
            (MATCH_ID,),
        ).fetchall()
    return [{"xuid": r[0], "time_ms": r[1]} for r in rows]


# ── Scan avec marqueur fixe ───────────────────────────────────────────────────


def scan_chunk_fixed_marker(
    data: bytes,
    start_ms: int,
    dur_ms: int,
    chunk_idx: int,
) -> list[dict]:
    """Scan un chunk avec le marqueur fixe, extrait tous les champs possibles."""
    bits = Bits(bytes=data)
    total_bits = len(bits)
    estimate_ts = estimate_ts_frames(data, start_ms, dur_ms)
    events: list[dict] = []

    for pos in bits.findall(FIXED_MARKER, bytealigned=False):
        # Base A : position + 3 (current code convention)
        base_a = pos + 3
        # Base B : position + 11 (after full marker)
        base_b = pos + FIXED_MARKER_LEN

        weapon_start_a = base_a + _WEAPON_BIT_OFFSET
        weapon_start_b = base_b + _WEAPON_BIT_OFFSET

        # Valider avec base_a (comme le code actuel)
        if weapon_start_a + 64 > total_bits:
            continue

        weapon_int_a = bits[weapon_start_a : weapon_start_a + 64].uint
        weapon_bytes_a = weapon_int_a.to_bytes(8, byteorder="big")

        valid_a = weapon_int_a in WEAPON_IDS_INT or weapon_bytes_a[4:] == COMMON_WEAPON_SUFFIX

        # Valider aussi avec base_b
        valid_b = False
        weapon_int_b = 0
        weapon_bytes_b = b"\x00" * 8
        if weapon_start_b + 64 <= total_bits:
            weapon_int_b = bits[weapon_start_b : weapon_start_b + 64].uint
            weapon_bytes_b = weapon_int_b.to_bytes(8, byteorder="big")
            valid_b = weapon_int_b in WEAPON_IDS_INT or weapon_bytes_b[4:] == COMMON_WEAPON_SUFFIX

        if not valid_a and not valid_b:
            continue

        byte_pos = pos // 8
        t_ms = estimate_ts(byte_pos)

        # Extraire toutes les valeurs 5-bit possibles dans les 50 bits après chaque base
        fields_a: dict[int, int] = {}
        for off in range(MAX_OFFSET):
            bit_pos = base_a + off
            if bit_pos + 5 <= total_bits:
                fields_a[off] = bits[bit_pos : bit_pos + 5].uint

        fields_b: dict[int, int] = {}
        for off in range(MAX_OFFSET):
            bit_pos = base_b + off
            if bit_pos + 5 <= total_bits:
                fields_b[off] = bits[bit_pos : bit_pos + 5].uint

        # Extraire aussi les bits AVANT base_a (b0 region, offsets négatifs)
        # Théorie: pi = top 3 bits de b0, à bits[base_a - 8 : base_a - 5] (3 bits)
        # Tester offsets -15 à -1 depuis base_a (3-bit values)
        fields_pre: dict[int, int] = {}
        for off in range(1, 16):
            bit_pos = base_a - off - 3  # 3-bit field ending at base_a - off
            if bit_pos >= 0 and bit_pos + 3 <= total_bits:
                fields_pre[-off] = bits[bit_pos : bit_pos + 3].uint

        # Fire counter (used for dedup) — base_a offset 24-32 (comme le code actuel)
        fire_counter = 0
        if base_a + 32 <= total_bits:
            fire_counter = bits[base_a + 24 : base_a + 32].uint

        events.append(
            {
                "t_ms": t_ms,
                "chunk_idx": chunk_idx,
                "pos": pos,
                "byte_pos": byte_pos,
                "fire_counter": fire_counter,
                "weapon_bytes_a": weapon_bytes_a if valid_a else None,
                "weapon_bytes_b": weapon_bytes_b if valid_b else None,
                "valid_a": valid_a,
                "valid_b": valid_b,
                "fields_a": fields_a,  # offset -> 5-bit value (base_a = pos+3)
                "fields_b": fields_b,  # offset -> 5-bit value (base_b = pos+11)
                "fields_pre": fields_pre,  # neg offset -> 3-bit value (b0 region before marker)
            }
        )

    # Dedup par (fire_counter, weapon_bytes_a) pour base_a
    deduped: dict[tuple, dict] = {}
    for ev in sorted(events, key=lambda e: e["t_ms"]):
        key = (ev["fire_counter"], ev["weapon_bytes_a"])
        if key not in deduped:
            deduped[key] = ev
    return sorted(deduped.values(), key=lambda e: e["t_ms"])


# ── Recherche empirique ───────────────────────────────────────────────────────


def find_pi_offset(
    all_events: list[dict],
    kills: list[dict],
    xuid_to_pi: dict[str, int],
) -> None:
    """Vote pour chaque offset selon la correspondance event.fields[off] == killer_pi."""

    # Organiser kills par xuid
    kills_by_xuid: dict[str, list[int]] = defaultdict(list)
    for k in kills:
        if k["xuid"] in xuid_to_pi:
            kills_by_xuid[k["xuid"]].append(k["time_ms"])

    votes_a: Counter = Counter()  # base_a (pos+3), 5-bit values
    votes_b: Counter = Counter()  # base_b (pos+11), 5-bit values
    votes_pre: Counter = Counter()  # before base_a, 3-bit values (b0 region)
    total_matched = 0
    misses = 0

    for xuid, kill_times in kills_by_xuid.items():
        pi = xuid_to_pi[xuid]
        for t_kill in kill_times:
            # Trouver le fire event le plus récent dans la fenêtre [t_kill-WINDOW, t_kill]
            candidates = [
                ev for ev in all_events if (t_kill - KILL_WINDOW_MS) <= ev["t_ms"] <= t_kill
            ]
            if not candidates:
                misses += 1
                continue

            best = max(candidates, key=lambda e: e["t_ms"])
            total_matched += 1

            # Voter pour chaque offset qui prédit correctement le pi
            for off, val in best["fields_a"].items():
                if val == pi:
                    votes_a[off] += 1
            for off, val in best["fields_b"].items():
                if val == pi:
                    votes_b[off] += 1
            # Offsets négatifs (b0 region), 3-bit field
            for off, val in best["fields_pre"].items():
                if val == pi:
                    votes_pre[off] += 1

    total_kills = sum(len(v) for v in kills_by_xuid.values())
    print(
        f"\n[empirique] {total_kills} kills avec pi connu, "
        f"{total_matched} avec event dans la fenetre, "
        f"{misses} sans event"
    )

    # Afficher top-10 offsets pour chaque base
    print("\n--- Base A (pos+3, convention actuelle), 5-bit ---")
    _print_top_offsets(votes_a, total_matched)

    print("\n--- Base B (pos+11, apres marqueur complet), 5-bit ---")
    _print_top_offsets(votes_b, total_matched)

    print("\n--- Region b0 AVANT base_a (offsets negatifs), 3-bit ---")
    print("  Theorie: pi = bits[base_a-8 : base_a-5] = offset -8 (3 bits)")
    _print_top_offsets_3bit(votes_pre, total_matched)

    # Verification : pour le meilleur offset, afficher la distribution pi
    best_a_off = votes_a.most_common(1)[0][0] if votes_a else None
    best_pre_off = votes_pre.most_common(1)[0][0] if votes_pre else None

    if best_a_off is not None:
        print(f"\n[verification] Base A, offset {best_a_off} : valeurs extractees vs pi connu")
        _verify_offset(all_events, kills, xuid_to_pi, "a", best_a_off)

    if best_pre_off is not None:
        print(f"\n[verification] Region b0, offset {best_pre_off} (3-bit) vs pi connu")
        _verify_offset_pre(all_events, kills, xuid_to_pi, best_pre_off)

    # Test filtré : si offset 31 est le vrai pi, filtrer par pi AVANT de prendre le plus proche
    print("\n--- TEST FILTRE par fields_a[31] == pi ---")
    _verify_filtered_by_pi(all_events, kills, xuid_to_pi, 31)

    # Distribution fields_a[31] sur TOUS les 773 events
    print("\n--- Distribution fields_a[31] sur TOUS les events ---")
    dist_all: Counter = Counter()
    for ev in all_events:
        v = ev["fields_a"].get(31)
        if v is not None:
            dist_all[v] += 1
    print(f"  {dict(sorted(dist_all.items()))}")
    print(f"  Total events avec valeur: {sum(dist_all.values())}")

    # Test spécifique : pi = top-3-bits de b0 (pos-5 à pos-2 dans bitstream)
    print("\n--- TEST DIRECT : top-3-bits de b0 (bits[pos-5:pos-2]) vs pi ---")
    _test_b0_top3_vs_pi(all_events, kills, xuid_to_pi)

    # Test final : global pool (état actuel) vs filtre POV uniquement
    print("\n--- COMPARAISON : global pool vs filtre POV (pi=1 seulement) ---")
    _compare_global_vs_pov_filter(all_events, kills, xuid_to_pi)


def _compare_global_vs_pov_filter(
    all_events: list[dict],
    kills: list[dict],
    xuid_to_pi: dict[str, int],
) -> None:
    """Compare global pool (actuel) vs filtre POV uniquement (pi=1 only)."""
    # La POV est identifiable : le seul pi qui a des fire events valides
    # fields_a[31] == 0 → vrai fire event → tous ont player_index=1 dans scan_fire_events_all
    valid = [ev for ev in all_events if ev["fields_a"].get(31) == 0]
    POV_PI = 1  # hardcodé dans le code actuel via _build_marker(1)

    kills_sorted = sorted(kills, key=lambda k: k["time_ms"])
    kills_by_xuid: dict[str, list[int]] = defaultdict(list)
    for k in kills:
        if k["xuid"] in xuid_to_pi:
            kills_by_xuid[k["xuid"]].append(k["time_ms"])

    # 1. Global pool : état actuel
    pool_global = sorted(valid, key=lambda e: e["t_ms"])
    matched_global = 0
    pov_matched = 0  # kills de POV matchés
    non_pov_matched = 0  # kills NON-POV matchés (attribution croisée = bug)
    for k in kills_sorted:
        pi = xuid_to_pi.get(k["xuid"])
        if pi is None:
            continue
        t_ms = k["time_ms"]
        cands = [
            (i, ev)
            for i, ev in enumerate(pool_global)
            if (t_ms - KILL_WINDOW_MS) <= ev["t_ms"] <= t_ms
        ]
        if cands:
            best_idx, _ = max(cands, key=lambda x: x[1]["t_ms"])
            pool_global.pop(best_idx)
            matched_global += 1
            if pi == POV_PI:
                pov_matched += 1
            else:
                non_pov_matched += 1

    # 2. Filtre POV : seulement attribuer aux kills du POV
    pool_pov = sorted(valid, key=lambda e: e["t_ms"])
    matched_pov_only = 0
    pov_xuid = next((x for x, p in xuid_to_pi.items() if p == POV_PI), None)
    pov_kills = kills_by_xuid.get(pov_xuid or "", [])
    for t_kill in sorted(pov_kills):
        cands = [
            (i, ev)
            for i, ev in enumerate(pool_pov)
            if (t_kill - KILL_WINDOW_MS) <= ev["t_ms"] <= t_kill
        ]
        if cands:
            best_idx, _ = max(cands, key=lambda x: x[1]["t_ms"])
            pool_pov.pop(best_idx)
            matched_pov_only += 1

    total_kills = len([k for k in kills if k["xuid"] in xuid_to_pi])
    pov_total = len(pov_kills)
    non_pov_total = total_kills - pov_total

    print(f"  POV player : pi={POV_PI} xuid={pov_xuid}  kills={pov_total}/{total_kills}")
    print(f"  Events valides disponibles: {len(valid)}")
    print()
    print("  GLOBAL POOL (actuel):")
    print(f"    Total matchés: {matched_global}/{total_kills}")
    print(f"    POV kills matchés: {pov_matched}/{pov_total} (CORRECT)")
    print(
        f"    Non-POV kills matchés: {non_pov_matched}/{non_pov_total} (ATTRIBUTION CROISEE = BUG)"
    )
    print()
    print("  FILTRE POV (fix proposé):")
    print(f"    POV kills matchés: {matched_pov_only}/{pov_total}")
    print(f"    Non-POV kills matchés: 0/{non_pov_total} (-> fallback Formula A)")


def _test_b0_top3_vs_pi(
    all_events: list[dict],
    kills: list[dict],
    xuid_to_pi: dict[str, int],
) -> None:
    """Test direct : extrait les 3 bits MSB de b0 (avant le marqueur) pour les events valides."""

    # Filtrer les events avec slot valide (les 58 vrais)
    valid: list[dict] = []
    for ev in all_events:
        fa = ev["fields_a"]
        # slot en base_a+32..39 → fields_a[32] 5-bit couvre bits 32-36
        # Pour vérifier slot exact, on lit depuis fields_pre mais plus simple:
        # fields_pre[-5] = bits[pos-5:pos-2] = top-3-bits de b0
        # On utilise la connaissance que les vrais events ont fields_a[31] == 0
        if fa.get(31) == 0:  # vrai pour les 58 events valides
            valid.append(ev)

    print(f"  {len(valid)} events valides (slot filter proxy via fields_a[31]=0)")

    # Distribution des top-3-bits de b0 pour les events valides
    dist_valid: Counter = Counter()
    for ev in valid:
        v = ev["fields_pre"].get(-5)
        if v is not None:
            dist_valid[v] += 1
    print(f"  Distribution top-3-bits b0 sur events valides: {dict(sorted(dist_valid.items()))}")

    # Distribution sur TOUS les 773 events
    dist_all: Counter = Counter()
    for ev in all_events:
        v = ev["fields_pre"].get(-5)
        if v is not None:
            dist_all[v] += 1
    print(f"  Distribution top-3-bits b0 sur TOUS (773) events: {dict(sorted(dist_all.items()))}")

    # Corrélation avec les kills pour les events valides uniquement
    kills_by_xuid: dict[str, list[int]] = defaultdict(list)
    for k in kills:
        if k["xuid"] in xuid_to_pi:
            kills_by_xuid[k["xuid"]].append(k["time_ms"])

    correct = 0
    total = 0
    for xuid, kill_times in kills_by_xuid.items():
        pi = xuid_to_pi[xuid]
        for t_kill in kill_times:
            candidates = [ev for ev in valid if (t_kill - KILL_WINDOW_MS) <= ev["t_ms"] <= t_kill]
            if not candidates:
                continue
            best = max(candidates, key=lambda e: e["t_ms"])
            extracted = best["fields_pre"].get(-5)
            if extracted is not None:
                total += 1
                if extracted == pi:
                    correct += 1

    pct = correct * 100 // total if total else 0
    print(f"  Corrélation (events valides) : {correct}/{total} ({pct}%) top-3-bits = pi")

    # Test: filtrer par top-3-bits=pi puis prendre le plus proche
    correct_filtered = 0
    no_match_filtered = 0
    total_filtered = 0
    per_player: dict[int, tuple[int, int]] = {}
    for xuid, kill_times in kills_by_xuid.items():
        pi = xuid_to_pi[xuid]
        pk = len(kill_times)
        pm = 0
        for t_kill in kill_times:
            candidates = [
                ev
                for ev in valid
                if (t_kill - KILL_WINDOW_MS) <= ev["t_ms"] <= t_kill
                and ev["fields_pre"].get(-5) == pi
            ]
            total_filtered += 1
            if candidates:
                correct_filtered += 1
                pm += 1
            else:
                no_match_filtered += 1
        per_player[pi] = (pk, pm)

    pct2 = correct_filtered * 100 // total_filtered if total_filtered else 0
    print(f"  Filtré par top-3-bits=pi : {correct_filtered}/{total_filtered} ({pct2}%)")
    print("  Par joueur:")
    for pi in sorted(per_player):
        pk, pm = per_player[pi]
        ppct = pm * 100 // pk if pk else 0
        print(f"    pi={pi}: {pm}/{pk} ({ppct}%)")


def _print_top_offsets(votes: Counter, total: int) -> None:
    if not votes:
        print("  Aucun vote")
        return
    for off, count in votes.most_common(15):
        pct = count * 100 // total if total else 0
        bar = "#" * (count * 30 // max(votes.values()))
        print(f"  offset {off:3d} : {count:4d}/{total} ({pct:3d}%) {bar}")


def _print_top_offsets_3bit(votes: Counter, total: int) -> None:
    if not votes:
        print("  Aucun vote")
        return
    for off, count in votes.most_common(15):
        pct = count * 100 // total if total else 0
        bar = "#" * (count * 30 // max(votes.values()))
        print(f"  offset {off:4d} : {count:4d}/{total} ({pct:3d}%) {bar}")


def _validate_event_structure(
    all_events: list[dict],
    chunks: dict,
) -> None:
    """Valide la structure des events : pos%8, counter div 4, slot in {1,3}."""

    pos_mod8: Counter = Counter()
    counter_valid = 0
    counter_invalid = 0
    slot_valid = 0
    slot_invalid = 0
    slot_dist: Counter = Counter()

    for ev in all_events:
        pos = ev["pos"]
        pos_mod8[pos % 8] += 1

        # Acceder aux bits bruts depuis les chunks
        chunk_idx = ev["chunk_idx"]
        if chunk_idx not in chunks:
            continue
        data, _, _ = chunks[chunk_idx]
        bits = Bits(bytes=data)
        base_a = pos + 3
        total_bits = len(bits)

        # Counter : bits[base_a+24 : base_a+32]
        if base_a + 32 <= total_bits:
            ctr = bits[base_a + 24 : base_a + 32].uint
            if ctr % 4 == 0:
                counter_valid += 1
            else:
                counter_invalid += 1

        # Slot : bits[base_a+32 : base_a+40]
        if base_a + 40 <= total_bits:
            sl = bits[base_a + 32 : base_a + 40].uint
            slot_dist[sl] += 1
            if sl in (0x01, 0x03):
                slot_valid += 1
            else:
                slot_invalid += 1

    total = len(all_events)
    print(f"\n--- Validation structurelle ({total} events) ---")
    print(f"  pos%8 distribution: {dict(sorted(pos_mod8.items()))}")
    print(
        f"  counter div4: {counter_valid}/{counter_valid+counter_invalid} ({counter_valid*100//(counter_valid+counter_invalid) if counter_valid+counter_invalid else 0}%)"
    )
    print(
        f"  slot in {{1,3}}: {slot_valid}/{slot_valid+slot_invalid} ({slot_valid*100//(slot_valid+slot_invalid) if slot_valid+slot_invalid else 0}%)"
    )
    print(f"  slot dist top10: {dict(slot_dist.most_common(10))}")


def _verify_filtered_by_pi(
    all_events: list[dict],
    kills: list[dict],
    xuid_to_pi: dict[str, int],
    off: int,
) -> None:
    """Test filtré : pour chaque kill, chercher events WHERE fields_a[off]==pi, puis le plus proche."""
    correct = 0
    no_match = 0
    total = 0

    kills_by_xuid: dict[str, list[int]] = defaultdict(list)
    for k in kills:
        if k["xuid"] in xuid_to_pi:
            kills_by_xuid[k["xuid"]].append(k["time_ms"])

    per_player: dict[int, tuple[int, int]] = {}  # pi -> (total_kills, matched)

    for xuid, kill_times in kills_by_xuid.items():
        pi = xuid_to_pi[xuid]
        pk = 0
        pm = 0
        for t_kill in kill_times:
            pk += 1
            # Filtrer par pi d'abord
            candidates = [
                ev
                for ev in all_events
                if (t_kill - KILL_WINDOW_MS) <= ev["t_ms"] <= t_kill
                and ev["fields_a"].get(off) == pi
            ]
            if candidates:
                correct += 1
                pm += 1
            else:
                no_match += 1
            total += 1
        per_player[pi] = (pk, pm)

    pct = correct * 100 // total if total else 0
    print(f"  Match: {correct}/{total} ({pct}%)  No-match: {no_match}")
    print("  Par joueur (pi: kills_total / matched):")
    for pi in sorted(per_player):
        pk, pm = per_player[pi]
        ppct = pm * 100 // pk if pk else 0
        print(f"    pi={pi}: {pm}/{pk} ({ppct}%)")


def _verify_offset_pre(
    all_events: list[dict],
    kills: list[dict],
    xuid_to_pi: dict[str, int],
    off: int,
) -> None:
    """Verification pour les offsets negatifs (3-bit b0 region)."""
    correct = 0
    wrong = 0
    total = 0
    pi_dist: Counter = Counter()

    kills_by_xuid: dict[str, list[int]] = defaultdict(list)
    for k in kills:
        if k["xuid"] in xuid_to_pi:
            kills_by_xuid[k["xuid"]].append(k["time_ms"])

    for xuid, kill_times in kills_by_xuid.items():
        pi = xuid_to_pi[xuid]
        for t_kill in kill_times:
            candidates = [
                ev for ev in all_events if (t_kill - KILL_WINDOW_MS) <= ev["t_ms"] <= t_kill
            ]
            if not candidates:
                continue
            best = max(candidates, key=lambda e: e["t_ms"])
            extracted = best["fields_pre"].get(off)
            if extracted is None:
                continue
            total += 1
            pi_dist[extracted] += 1
            if extracted == pi:
                correct += 1
            else:
                wrong += 1

    pct = correct * 100 // total if total else 0
    print(f"  Correct: {correct}/{total} ({pct}%), Mauvais: {wrong}")
    print(f"  Distribution (3-bit): {dict(sorted(pi_dist.items()))}")

    # Bonus: distribution b0 sur TOUS les events (pas seulement les matched kills)
    b0_all: Counter = Counter()
    for ev in all_events:
        v = ev["fields_pre"].get(off)
        if v is not None:
            b0_all[v] += 1
    print(f"  Distribution b0 sur 773 events: {dict(sorted(b0_all.items()))}")


def _verify_offset(
    all_events: list[dict],
    kills: list[dict],
    xuid_to_pi: dict[str, int],
    base: str,  # "a" or "b"
    off: int,
) -> None:
    field_key = f"fields_{base}"
    correct = 0
    wrong = 0
    total = 0
    pi_dist: Counter = Counter()

    kills_by_xuid: dict[str, list[int]] = defaultdict(list)
    for k in kills:
        if k["xuid"] in xuid_to_pi:
            kills_by_xuid[k["xuid"]].append(k["time_ms"])

    for xuid, kill_times in kills_by_xuid.items():
        pi = xuid_to_pi[xuid]
        for t_kill in kill_times:
            candidates = [
                ev for ev in all_events if (t_kill - KILL_WINDOW_MS) <= ev["t_ms"] <= t_kill
            ]
            if not candidates:
                continue
            best = max(candidates, key=lambda e: e["t_ms"])
            extracted = best[field_key].get(off)
            if extracted is None:
                continue
            total += 1
            pi_dist[extracted] += 1
            if extracted == pi:
                correct += 1
            else:
                wrong += 1

    pct = correct * 100 // total if total else 0
    print(f"  Correct: {correct}/{total} ({pct}%), Mauvais: {wrong}")
    print(f"  Distribution valeurs extraites: {dict(sorted(pi_dist.items()))}")

    # Afficher les erreurs
    if wrong > 0 and wrong <= 20:
        print("  Cas incorrects :")
        for xuid, kill_times in kills_by_xuid.items():
            pi = xuid_to_pi[xuid]
            for t_kill in kill_times:
                candidates = [
                    ev for ev in all_events if (t_kill - KILL_WINDOW_MS) <= ev["t_ms"] <= t_kill
                ]
                if not candidates:
                    continue
                best = max(candidates, key=lambda e: e["t_ms"])
                extracted = best[field_key].get(off)
                if extracted is not None and extracted != pi:
                    print(
                        f"    xuid={xuid} pi={pi} extracted={extracted} t={t_kill} ev_t={best['t_ms']:.0f}"
                    )


# ── Scan nibble-shifted pour différents b0 ────────────────────────────────────


def _read_shifted(data: bytes, i: int) -> int:
    return ((data[i] << 4) | (data[i + 1] >> 4)) & 0xFF


def _scan_b0_variants(chunks: dict) -> None:
    """Scan nibble-shifted pour b0 = 0x0d|(pi<<5) pour pi=0..7, vérifie b1=0x26, slot, counter."""
    from src.analysis._weapon_data import WEAPON_IDS_INT
    from src.analysis._weapon_scanners import COMMON_WEAPON_SUFFIX

    b0_candidates = {pi: (0x0D | (pi << 5)) for pi in range(8)}
    totals: dict[int, int] = {}
    valid_counts: dict[int, int] = {}

    for pi, b0_val in b0_candidates.items():
        total = 0
        valid = 0
        for data, _, _ in chunks.values():
            for i in range(len(data) - 14):
                b0 = _read_shifted(data, i)
                if b0 != b0_val:
                    continue
                b1 = _read_shifted(data, i + 1)
                if b1 != 0x26:
                    continue
                total += 1
                ctr = _read_shifted(data, i + 4)
                slot = _read_shifted(data, i + 5)
                if slot not in (0x01, 0x03):
                    continue
                if ctr % 4 != 0:
                    continue
                # Weapon check (8 nibble-shifted bytes at i+6..i+13)
                wb = bytes(_read_shifted(data, i + 6 + j) for j in range(8))
                wid = int.from_bytes(wb, byteorder="big")
                if wid not in WEAPON_IDS_INT and wb[4:] != COMMON_WEAPON_SUFFIX:
                    continue
                valid += 1
        totals[pi] = total
        valid_counts[pi] = valid
        print(
            f"  pi={pi} b0=0x{b0_val:02x} : {total:4d} events (b1=0x26), {valid:3d} avec slot+ctr+weapon valides"
        )


# ── Main ──────────────────────────────────────────────────────────────────────


def main() -> None:
    print(f"Match : {MATCH_ID}")

    chunks = load_chunks()
    xuid_to_pi = resolve_pi_map(chunks)
    kills = load_kills()
    print(f"[kills] {len(kills)} kills, {len(xuid_to_pi)} joueurs avec pi resolu")

    # Scanner tous les chunks
    print(f"\n[scan] Marqueur fixe 0b10100100110 sur {len(chunks)} chunks...")
    all_events: list[dict] = []
    chunks_sorted = sorted(chunks)
    for idx in chunks_sorted:
        data, start_ms, dur_ms = chunks[idx]
        evs = scan_chunk_fixed_marker(data, start_ms, dur_ms, idx)
        all_events.extend(evs)
    all_events.sort(key=lambda e: e["t_ms"])
    print(f"[scan] {len(all_events)} fire events trouves")

    # Distribution par base valide
    n_a = sum(1 for e in all_events if e["valid_a"])
    n_b = sum(1 for e in all_events if e["valid_b"])
    print(f"[scan] valid_a={n_a}, valid_b={n_b}")

    # Recherche empirique
    find_pi_offset(all_events, kills, xuid_to_pi)
    _validate_event_structure(all_events, chunks)

    # Scan direct nibble-shifted pour b0 variants
    print("\n--- SCAN NIBBLE-SHIFTED direct pour differents b0 ---")
    _scan_b0_variants(chunks)


if __name__ == "__main__":
    main()
