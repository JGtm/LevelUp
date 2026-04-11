"""
Phase 0 — Exploration non-POV fire events & melee events

═══════════════════════════════════════════════════════════════════
OBJECTIF : Mesurer si les fire events Section 2 sont détectables
et fiables pour les joueurs non-POV (player_index ≠ 1), et si les
melee events (marqueur 0xd340) sont exploitables pour le POV.

LECTURE SEULE — aucune écriture en DB.
Sortie : CSV + JSON dans data/investigation/

Protocole (PLAN_WEAPON_PARSER_V2.md §3.2) :
  E0.1 — Sélectionner 20 matchs diversifiés
  E0.2 — Scanner fire events pour tous les pi (0–7)
  E0.3 — Compter events par pi, taux de corrélation
  E0.4 — Comparer avec l'attribution T1 (Formula A) existante
  E0.5 — Scanner melee events POV (marqueur 0xd340)

Usage :
  python scripts/experimental/explore_non_pov_fire_events.py
  python scripts/experimental/explore_non_pov_fire_events.py --matches 10
  python scripts/experimental/explore_non_pov_fire_events.py --match-id abc12345
═══════════════════════════════════════════════════════════════════

Date : 2026-03-12
"""

from __future__ import annotations

import argparse
import csv
import json
import logging
import sys
from collections import Counter, defaultdict
from pathlib import Path

# ── Setup ──────────────────────────────────────────────────────────────

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

from src.analysis._weapon_data import (
    WEAPON_ID_MAP,
    WEAPON_IDS_INT,
    WEAPON_INT_TO_NAME,
)
from src.analysis.packet_index import (
    detect_pi_from_metadata,
    extract_metadata_payload,
    index_chunk,
)
from src.analysis.player_index import detect_player_indices
from src.analysis.weapon_parser import (
    KILL_WINDOW_MS,
    scan_fire_events,
)
from src.utils.db import duckdb_read_only
from src.utils.paths import get_shared_matches_path

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%H:%M:%S",
)
logger = logging.getLogger("phase0")

SHARED_DB = get_shared_matches_path()
CHUNKS_BASE = ROOT / "data" / "investigation" / "chunks"
OUTPUT_DIR = ROOT / "data" / "investigation"
_WEAPON_KILLS_BIT = 1 << 21

# Durée approximative d'un chunk (observée : ~19 977 ms via FRAME packets)
APPROX_CHUNK_MS = 20_000

# Melee marker (acurtis, inv43)
MELEE_LEAD_BYTE = 0xD3
MELEE_SECOND_BYTE = 0x40


# ═══════════════════════════════════════════════════════════════════════
# E0.1 — Sélection des matchs
# ═══════════════════════════════════════════════════════════════════════


def select_matches(conn, limit: int, match_id: str | None = None) -> list[dict]:
    """Sélectionne des matchs avec weapon_kills déjà processés ET chunks en cache."""
    if match_id:
        rows = conn.execute(
            """
            SELECT mr.match_id, mr.start_time,
                   COALESCE(mr.map_name, 'unknown') AS map_name,
                   COALESCE(mr.game_variant_name, 'unknown') AS category,
                   (SELECT COUNT(*) FROM match_participants mp2
                    WHERE mp2.match_id = mr.match_id
                    AND mp2.xuid NOT LIKE 'bid(%%') AS n_players
            FROM match_registry mr
            WHERE mr.match_id LIKE ?
            LIMIT 1
            """,
            (match_id + "%",),
        ).fetchall()
    else:
        rows = conn.execute(
            """
            SELECT mr.match_id, mr.start_time,
                   COALESCE(mr.map_name, 'unknown') AS map_name,
                   COALESCE(mr.game_variant_name, 'unknown') AS category,
                   (SELECT COUNT(*) FROM match_participants mp2
                    WHERE mp2.match_id = mr.match_id
                    AND mp2.xuid NOT LIKE 'bid(%%') AS n_players
            FROM match_registry mr
            WHERE (COALESCE(mr.backfill_completed, 0) & ?) != 0
              AND COALESCE(mr.is_firefight, FALSE) = FALSE
            ORDER BY mr.start_time DESC
            LIMIT ?
            """,
            (_WEAPON_KILLS_BIT, limit * 3),
        ).fetchall()

    # Filtrer : seuls les matchs avec chunks en cache
    results = []
    for match_id_full, start_time, map_name, category, n_players in rows:
        cache_dir = CHUNKS_BASE / match_id_full[:8]
        if cache_dir.exists() and any(cache_dir.glob("chunk_*.bin")):
            n_chunks = len(list(cache_dir.glob("chunk_*.bin")))
            results.append({
                "match_id": match_id_full,
                "short": match_id_full[:8],
                "start_time": str(start_time),
                "map": map_name,
                "category": category,
                "n_players": n_players,
                "n_chunks_cached": n_chunks,
            })
            if len(results) >= limit:
                break

    return results


# ═══════════════════════════════════════════════════════════════════════
# E0.2 — Chargement chunks depuis cache
# ═══════════════════════════════════════════════════════════════════════


def load_cached_chunks(match_short: str) -> list[tuple[int, bytes]]:
    """Charge les chunks binaires depuis le cache disque."""
    cache_dir = CHUNKS_BASE / match_short
    if not cache_dir.exists():
        return []
    result = []
    for p in sorted(cache_dir.glob("chunk_*.bin")):
        try:
            idx = int(p.stem.split("_")[1])
            result.append((idx, p.read_bytes()))
        except (ValueError, IndexError):
            continue
    return result


# ═══════════════════════════════════════════════════════════════════════
# E0.2 — Résolution player_index
# ═══════════════════════════════════════════════════════════════════════


def resolve_all_player_indices(
    chunks: list[tuple[int, bytes]],
    participants: dict[str, str],
) -> dict[int, int]:
    """Résout xuid_int → player_index via METADATA puis fallback acurtis.

    Returns: {xuid_int: player_index}
    """
    xuid_int_map = {}
    for xuid_str in participants:
        if xuid_str.isdigit():
            xuid_int_map[int(xuid_str)] = xuid_str

    if not xuid_int_map or not chunks:
        return {}

    first_chunk_data = chunks[0][1]

    # Méthode 1 : PLAYER_METADATA packet
    result: dict[int, int] = {}
    packets = index_chunk(first_chunk_data)
    metadata = extract_metadata_payload(first_chunk_data, packets)
    if metadata is not None:
        pi_to_xuid = detect_pi_from_metadata(metadata, xuid_int_map)
        if pi_to_xuid:
            for pi, xuid_int in pi_to_xuid.items():
                result[xuid_int] = pi

    # Méthode 2 : fallback acurtis pour les non-résolus
    missing = {xi: xs for xi, xs in xuid_int_map.items() if xi not in result}
    if missing:
        for _, chunk_data in chunks:
            pi_to_xuid = detect_player_indices(chunk_data, missing)
            for pi, xuid_int in pi_to_xuid.items():
                result[xuid_int] = pi
            missing = {xi: xs for xi, xs in missing.items() if xi not in result}
            if not missing:
                break

    return result


# ═══════════════════════════════════════════════════════════════════════
# E0.2+E0.3 — Scan fire events pour tous les pi
# ═══════════════════════════════════════════════════════════════════════


def scan_all_fire_events(
    chunks: list[tuple[int, bytes]],
    n_players: int = 8,
) -> dict[int, list[dict]]:
    """Scanne les fire events pour tous les player_index (0–n_players).

    Accumule cross-chunk en une liste plate par pi.
    Le chunk_start_ms est estimé par chunk_idx * APPROX_CHUNK_MS pour
    aligner les timestamps fire events avec les time_ms de highlight_events.
    Les timestamps µs internes aux paquets FRAME fournissent la précision
    intra-chunk (~16ms) via build_packet_estimator.
    """
    events_by_pi: dict[int, list[dict]] = {pi: [] for pi in range(n_players)}

    for chunk_idx, chunk_data in chunks:
        packets = index_chunk(chunk_data)
        chunk_start_ms = chunk_idx * APPROX_CHUNK_MS

        for pi in range(n_players):
            chunk_events = scan_fire_events(
                chunk_data, pi, chunk_start_ms, APPROX_CHUNK_MS, packets=packets
            )
            for ev in chunk_events:
                ev["chunk_idx"] = chunk_idx
                ev["player_index"] = pi
            events_by_pi[pi].extend(chunk_events)

    return events_by_pi


def correlate_fire_events_to_kills(
    kills: list[dict],
    fire_events: list[dict],
    window_ms: int = KILL_WINDOW_MS,
) -> list[dict]:
    """Corrélation kill → fire event (claim-and-remove).

    Pour chaque kill trié par time_ms croissant, cherche le dernier
    fire event non-claimé dans [kill_t - window_ms, kill_t].
    """
    kills_sorted = sorted(kills, key=lambda k: k["time_ms"])
    pool = sorted(fire_events, key=lambda e: e["timestamp_ms"])
    claimed = set()  # indices dans pool
    results = []

    for kill in kills_sorted:
        t = kill["time_ms"]
        best_idx = None
        best_ts = -1

        for i, ev in enumerate(pool):
            if i in claimed:
                continue
            ev_t = ev["timestamp_ms"]
            if ev_t < t - window_ms:
                continue
            if ev_t > t:
                break  # pool trié, plus rien après
            if ev_t > best_ts:
                best_idx = i
                best_ts = ev_t

        if best_idx is not None:
            claimed.add(best_idx)
            ev = pool[best_idx]
            weapon_bytes = ev.get("weapon_bytes", b"")
            weapon_int = int.from_bytes(weapon_bytes, "big") if weapon_bytes else None
            weapon_name = ev.get("weapon_name", "?")
            delta_ms = t - best_ts
            results.append({
                "time_ms": t,
                "xuid": kill.get("xuid", ""),
                "fire_event_ts": best_ts,
                "delta_ms": delta_ms,
                "weapon_name": weapon_name,
                "weapon_id": weapon_int,
                "is_known": weapon_int in WEAPON_IDS_INT if weapon_int else False,
                "correlated": True,
            })
        else:
            results.append({
                "time_ms": t,
                "xuid": kill.get("xuid", ""),
                "fire_event_ts": None,
                "delta_ms": None,
                "weapon_name": None,
                "weapon_id": None,
                "is_known": False,
                "correlated": False,
            })

    return results


# ═══════════════════════════════════════════════════════════════════════
# E0.5 — Scan melee events POV (marqueur 0xd340)
# ═══════════════════════════════════════════════════════════════════════


def _nibble_shift(data: bytes) -> bytes:
    """Nibble-shift layer extraction."""
    out = bytearray(len(data) - 1)
    for i in range(len(out)):
        out[i] = ((data[i] << 4) & 0xFF) | (data[i + 1] >> 4)
    return bytes(out)


def scan_melee_events(
    chunk_data: bytes,
    chunk_start_ms: int = 0,
    chunk_duration_ms: int = 0,
) -> list[dict]:
    """Scanne les melee events dans le nibble-shifted layer.

    Structure (inv43, acurtis) :
      [0] = 0xD3 (lead byte)
      [1] = 0x40
      [2:11] = context
      [11] = animation type (0x05 ou 0x0d)
      [12:20] = weapon_id (8 bytes)

    Mode exploration : accepte tous les weapon_bytes, catégorise en
    known/unknown pour évaluer la couverture sans filtrer.
    """
    shifted = _nibble_shift(chunk_data)
    events = []

    pos = 0
    while pos < len(shifted) - 20:
        idx = shifted.find(b"\xd3\x40", pos)
        if idx == -1 or idx + 20 > len(shifted):
            break

        anim_byte = shifted[idx + 11]
        if anim_byte not in (0x05, 0x0D):
            pos = idx + 1
            continue

        weapon_bytes = shifted[idx + 12 : idx + 20]
        weapon_int = int.from_bytes(weapon_bytes, "big")
        weapon_name = WEAPON_INT_TO_NAME.get(
            weapon_int,
            WEAPON_ID_MAP.get(weapon_bytes, f"?melee({weapon_bytes.hex()})"),
        )
        is_known = weapon_int in WEAPON_IDS_INT

        events.append({
            "byte_pos": idx,
            "animation": "lunge" if anim_byte == 0x05 else "regular",
            "weapon_bytes": weapon_bytes,
            "weapon_id": weapon_int,
            "weapon_name": weapon_name,
            "is_known": is_known,
        })
        pos = idx + 20

    return events


# ═══════════════════════════════════════════════════════════════════════
# Analyse principale
# ═══════════════════════════════════════════════════════════════════════


def analyze_match(
    conn,
    match_info: dict,
) -> dict:
    """Analyse complète d'un match : fire events non-POV + melee events."""
    match_id = match_info["match_id"]
    short = match_info["short"]
    logger.info("━━━ Match %s (%s, %s) ━━━", short, match_info["map"], match_info["category"])

    # Charger participants
    participants_rows = conn.execute(
        "SELECT mp.xuid, COALESCE(xa.gamertag, mp.xuid) "
        "FROM match_participants mp "
        "LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid "
        "WHERE mp.match_id = ? AND mp.xuid NOT LIKE 'bid(%%'",
        (match_id,),
    ).fetchall()
    participants = {x: gt for x, gt in participants_rows if x}

    # Charger kills par joueur
    kills_rows = conn.execute(
        "SELECT xuid, time_ms FROM highlight_events "
        "WHERE match_id = ? AND event_type = 'kill' ORDER BY time_ms",
        (match_id,),
    ).fetchall()
    kills_by_xuid: dict[str, list[dict]] = defaultdict(list)
    for xuid, t_ms in kills_rows:
        kills_by_xuid[xuid].append({"xuid": xuid, "time_ms": t_ms})

    total_kills = sum(len(v) for v in kills_by_xuid.values())

    # Charger attributions weapon_kills existantes
    existing_wk = {}
    try:
        wk_rows = conn.execute(
            "SELECT xuid, time_ms, weapon_id, confidence "
            "FROM weapon_kills WHERE match_id = ?",
            (match_id,),
        ).fetchall()
        for xuid, t_ms, wid, conf in wk_rows:
            existing_wk[(xuid, t_ms)] = {"weapon_id": wid, "confidence": conf}
    except Exception:
        pass

    # Charger chunks
    chunks = load_cached_chunks(short)
    if not chunks:
        logger.warning("  Pas de chunks en cache pour %s", short)
        return {"match_id": match_id, "error": "no_chunks"}

    logger.info("  %d participants, %d kills, %d chunks", len(participants), total_kills, len(chunks))

    # Résoudre player_index
    pi_map = resolve_all_player_indices(chunks, participants)
    xuid_to_pi: dict[str, int] = {}
    for xuid_str in participants:
        if xuid_str.isdigit():
            xi = int(xuid_str)
            if xi in pi_map:
                xuid_to_pi[xuid_str] = pi_map[xi]

    logger.info("  PI résolus : %d/%d joueurs", len(xuid_to_pi), len(participants))

    # Scanner fire events pour tous les pi
    max_pi = max(pi_map.values(), default=0) + 1
    max_pi = max(max_pi, 8)
    all_fire_events = scan_all_fire_events(chunks, n_players=max_pi)

    # Résumé par pi
    pi_summary = []
    for pi in range(max_pi):
        events = all_fire_events.get(pi, [])
        # Trouver le xuid associé à ce pi
        xuid_for_pi = None
        for xuid_s, p in xuid_to_pi.items():
            if p == pi:
                xuid_for_pi = xuid_s
                break

        is_pov = pi == 1
        n_events = len(events)
        n_kills_for_pi = len(kills_by_xuid.get(xuid_for_pi, [])) if xuid_for_pi else 0

        # Corrélation fire events → kills pour ce pi
        n_correlable = 0
        if xuid_for_pi and n_events > 0 and n_kills_for_pi > 0:
            corr = correlate_fire_events_to_kills(
                kills_by_xuid[xuid_for_pi], events
            )
            n_correlable = sum(1 for c in corr if c["correlated"])

        coverage_pct = (n_correlable / n_kills_for_pi * 100) if n_kills_for_pi > 0 else 0.0

        pi_summary.append({
            "match_id": match_id,
            "match_short": short,
            "pi": pi,
            "is_pov": is_pov,
            "xuid": xuid_for_pi or "",
            "gamertag": participants.get(xuid_for_pi, "") if xuid_for_pi else "",
            "n_fire_events": n_events,
            "n_kills": n_kills_for_pi,
            "n_correlable": n_correlable,
            "coverage_pct": round(coverage_pct, 1),
        })

        if n_events > 0:
            weapons_seen = Counter(
                ev.get("weapon_name", "?") for ev in events
            )
            top3 = weapons_seen.most_common(3)
            logger.info(
                "  pi=%d %s%s: %d events, %d/%d kills corrélés (%.1f%%) — top: %s",
                pi,
                "POV " if is_pov else "",
                participants.get(xuid_for_pi, "?")[:12] if xuid_for_pi else "unresolved",
                n_events,
                n_correlable,
                n_kills_for_pi,
                coverage_pct,
                ", ".join(f"{w}({c})" for w, c in top3),
            )

    # ── E0.4 : Comparaison avec l'attribution T1 existante ──
    comparison_rows = []
    for xuid_str, pi in xuid_to_pi.items():
        if pi == 1:  # skip POV, on compare seulement les non-POV
            continue
        kills = kills_by_xuid.get(xuid_str, [])
        if not kills:
            continue

        fire_events = all_fire_events.get(pi, [])
        fire_corr = correlate_fire_events_to_kills(kills, fire_events)

        for fc in fire_corr:
            t_ms = fc["time_ms"]
            existing = existing_wk.get((xuid_str, t_ms), {})
            t1_wid = existing.get("weapon_id")
            t1_conf = existing.get("confidence", "none")
            fire_wid = fc.get("weapon_id")
            fire_correlated = fc["correlated"]

            # Comparer
            if fire_correlated and fire_wid is not None:
                if t1_wid is None or t1_conf == "none":
                    verdict = "fire_better"
                elif t1_wid == fire_wid:
                    verdict = "equal"
                elif (
                    t1_conf in ("low", "none")
                    or (fire_wid in WEAPON_IDS_INT and t1_wid not in WEAPON_IDS_INT)
                ):
                    verdict = "fire_better"
                elif t1_wid in WEAPON_IDS_INT and fire_wid not in WEAPON_IDS_INT:
                    verdict = "t1_better"
                else:
                    verdict = "different"
            elif t1_wid is not None and t1_conf not in ("none",):
                verdict = "t1_only"
            else:
                verdict = "neither"

            comparison_rows.append({
                "match_id": match_id,
                "match_short": short,
                "xuid": xuid_str,
                "pi": pi,
                "time_ms": t_ms,
                "fire_weapon_id": fire_wid,
                "fire_weapon_name": fc.get("weapon_name", ""),
                "fire_delta_ms": fc.get("delta_ms"),
                "t1_weapon_id": t1_wid,
                "t1_confidence": t1_conf,
                "verdict": verdict,
            })

    verdicts = Counter(r["verdict"] for r in comparison_rows)
    logger.info("  Comparaison T1 vs Fire (non-POV) : %s", dict(verdicts))

    # ── E0.5 : Melee events POV ──
    melee_results = []
    for chunk_idx, chunk_data in chunks:
        melee_events = scan_melee_events(chunk_data)
        for me in melee_events:
            me["chunk_idx"] = chunk_idx
            me["match_id"] = match_id
        melee_results.extend(melee_events)

    logger.info("  Melee events POV détectés : %d", len(melee_results))

    return {
        "match_id": match_id,
        "match_short": short,
        "n_participants": len(participants),
        "n_kills": total_kills,
        "n_chunks": len(chunks),
        "n_pi_resolved": len(xuid_to_pi),
        "pi_summary": pi_summary,
        "comparison": comparison_rows,
        "melee_events_pov": len(melee_results),
        "melee_weapons": Counter(me["weapon_name"] for me in melee_results).most_common(10),
        "verdicts": dict(verdicts),
    }


# ═══════════════════════════════════════════════════════════════════════
# Export résultats
# ═══════════════════════════════════════════════════════════════════════


def export_results(all_results: list[dict]) -> None:
    """Exporte les résultats en CSV et JSON."""
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

    # ── CSV : fire events par pi ──
    csv_path = OUTPUT_DIR / "non_pov_fire_events_exploration.csv"
    fieldnames = [
        "match_id", "match_short", "pi", "is_pov", "xuid", "gamertag",
        "n_fire_events", "n_kills", "n_correlable", "coverage_pct",
    ]
    with open(csv_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        for r in all_results:
            if "error" in r:
                continue
            for row in r.get("pi_summary", []):
                writer.writerow({k: row.get(k, "") for k in fieldnames})

    logger.info("CSV exporté : %s", csv_path)

    # ── CSV : comparaison T1 vs fire events ──
    cmp_path = OUTPUT_DIR / "non_pov_t1_vs_fire_comparison.csv"
    cmp_fields = [
        "match_id", "match_short", "xuid", "pi", "time_ms",
        "fire_weapon_id", "fire_weapon_name", "fire_delta_ms",
        "t1_weapon_id", "t1_confidence", "verdict",
    ]
    with open(cmp_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=cmp_fields)
        writer.writeheader()
        for r in all_results:
            if "error" in r:
                continue
            for row in r.get("comparison", []):
                writer.writerow({k: row.get(k, "") for k in cmp_fields})

    logger.info("CSV comparaison exporté : %s", cmp_path)

    # ── JSON : résumé global ──
    summary = {
        "date": "2026-03-12",
        "matches_analyzed": len([r for r in all_results if "error" not in r]),
        "matches_errors": len([r for r in all_results if "error" in r]),
    }

    # Agrégation globale
    all_pi_rows = []
    for r in all_results:
        if "error" in r:
            continue
        all_pi_rows.extend(r.get("pi_summary", []))

    # Stats par catégorie (POV vs non-POV)
    for label, is_pov_val in [("pov", True), ("non_pov", False)]:
        rows = [r for r in all_pi_rows if r["is_pov"] == is_pov_val and r["n_kills"] > 0]
        if rows:
            total_kills = sum(r["n_kills"] for r in rows)
            total_corr = sum(r["n_correlable"] for r in rows)
            total_events = sum(r["n_fire_events"] for r in rows)
            summary[f"{label}_players_with_kills"] = len(rows)
            summary[f"{label}_total_kills"] = total_kills
            summary[f"{label}_total_fire_events"] = total_events
            summary[f"{label}_total_correlable"] = total_corr
            summary[f"{label}_coverage_pct"] = round(total_corr / total_kills * 100, 1) if total_kills else 0
        else:
            summary[f"{label}_players_with_kills"] = 0

    # Agrégation verdicts
    all_verdicts: Counter = Counter()
    total_melee = 0
    for r in all_results:
        if "error" in r:
            continue
        all_verdicts.update(r.get("verdicts", {}))
        total_melee += r.get("melee_events_pov", 0)

    summary["verdicts_global"] = dict(all_verdicts)
    summary["melee_events_pov_total"] = total_melee

    # Décision go/no-go
    non_pov_coverage = summary.get("non_pov_coverage_pct", 0)
    fire_better = all_verdicts.get("fire_better", 0)
    t1_better = all_verdicts.get("t1_better", 0)
    if non_pov_coverage >= 60 and fire_better > t1_better:
        summary["decision"] = "GO — Path A unifié (fire events prioritaires pour tous)"
    else:
        summary["decision"] = "NO-GO — Hybrid maintenu (POV=Path A, T1=Path B)"
    summary["decision_criteria"] = {
        "non_pov_coverage_pct": non_pov_coverage,
        "fire_better": fire_better,
        "t1_better": t1_better,
        "threshold_coverage": 60,
        "threshold_fire_better_gt_t1": True,
    }

    json_path = OUTPUT_DIR / "non_pov_exploration_summary.json"
    with open(json_path, "w", encoding="utf-8") as f:
        json.dump(summary, f, indent=2, ensure_ascii=False, default=str)

    logger.info("JSON résumé exporté : %s", json_path)

    # ── Affichage résumé terminal ──
    print("\n" + "=" * 70)
    print("  RÉSUMÉ PHASE 0 — Exploration non-POV fire events")
    print("=" * 70)
    print(f"  Matchs analysés : {summary['matches_analyzed']}")
    print(f"  Matchs erreurs  : {summary['matches_errors']}")
    print()
    for label in ["pov", "non_pov"]:
        print(f"  [{label.upper()}]")
        print(f"    Joueurs avec kills : {summary.get(f'{label}_players_with_kills', 0)}")
        print(f"    Kills totaux       : {summary.get(f'{label}_total_kills', 0)}")
        print(f"    Fire events        : {summary.get(f'{label}_total_fire_events', 0)}")
        print(f"    Corrélables        : {summary.get(f'{label}_total_correlable', 0)}")
        print(f"    Coverage           : {summary.get(f'{label}_coverage_pct', 0):.1f}%")
        print()
    print(f"  Verdicts (non-POV T1 vs Fire) : {dict(all_verdicts)}")
    print(f"  Melee events POV détectés     : {total_melee}")
    print()
    print(f"  ▶ DÉCISION : {summary['decision']}")
    print("=" * 70)


# ═══════════════════════════════════════════════════════════════════════
# Main
# ═══════════════════════════════════════════════════════════════════════


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Phase 0 — Exploration non-POV fire events & melee events"
    )
    parser.add_argument("--matches", type=int, default=20, help="Nombre de matchs à analyser")
    parser.add_argument("--match-id", type=str, help="Analyser un match spécifique (préfixe)")
    args = parser.parse_args()

    if not SHARED_DB.exists():
        logger.error("DB partagée introuvable : %s", SHARED_DB)
        sys.exit(1)

    with duckdb_read_only(SHARED_DB) as conn:
        # Sélection des matchs
        matches = select_matches(conn, args.matches, args.match_id)
        if not matches:
            logger.error("Aucun match trouvé avec chunks en cache.")
            logger.info("Tip: lancez d'abord un backfill --weapons pour peupler le cache.")
            sys.exit(1)

        logger.info("▶ %d matchs sélectionnés pour l'exploration", len(matches))

        # Analyse
        all_results = []
        for i, match_info in enumerate(matches, 1):
            logger.info("[%d/%d] Match %s", i, len(matches), match_info["short"])
            result = analyze_match(conn, match_info)
            all_results.append(result)

    # Export
    export_results(all_results)


if __name__ == "__main__":
    main()
