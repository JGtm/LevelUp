"""inv132 — Validation de la formule fire_seq % n_players = player_index.

Découverte inv#132 : le champ fire_seq (byte[2] dans la structure NS d'un fire
event, noté b2_stream dans le code) encode directement :

    fire_seq = player_index + life_number * n_players

→ pi = fire_seq % n_players  (sans cross-ref NS timeline)

Résultat attendu sur 7 points de validation lors de l'investigation initiale :
    b2=1 → pi=1, b2=3 → pi=3, b2=6 → pi=6 (déjà résolus par NS timeline)
    b2=27 → pi=3 (27%8=3), b2=28 → pi=4, b2=34 → pi=2, b2=46 → pi=6 : 7/7 ✓

Ce script valide la formule sur les 10 derniers matchs en cache :
1. Compare le dispatch formule vs dispatch NS actuel (map_b2_to_player)
2. Mesure l'impact sur les weapon_id unknown (raw FA handles → TYPE IDs via
   timeline_ns[chunk][pi_formula])
3. Vérifie la cohérence sur les matchs à n_players variable (8 ou 10)

Usage :
    python scripts/experimental/inv132_b2_pi_formula.py            # tous les matchs
    python scripts/experimental/inv132_b2_pi_formula.py --limit 50 # les 50 derniers
"""

from __future__ import annotations

import argparse
import json
import sys
from collections import Counter, defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

import duckdb
from rich.console import Console
from rich.table import Table

from src.analysis._weapon_data import WEAPON_ID_MAP, WEAPON_IDS_INT, WEAPON_INT_TO_NAME
from src.analysis._weapon_scanners import (
    COMMON_WEAPON_SUFFIX,
    scan_fire_events_bitstring,
    estimate_ts_frames,
)
from src.analysis.weapon_parser import (
    build_weapon_timelines,
    map_b2_to_player,
    group_events_by_pi,
    scan_fire_events_all,
)

CACHE_DIR = ROOT / "data" / "cache" / "film_chunks"
MANIFEST_DIR = ROOT / "data" / "cache" / "film_manifests"
SHARED_DB = ROOT / "data" / "warehouse" / "shared_matches_v2.duckdb"

console = Console()


# ── Helpers ────────────────────────────────────────────────────────────────


def load_chunks_from_cache(match_id: str) -> dict[int, tuple[bytes, int, int]]:
    """Charge les chunks depuis le cache local + manifest.

    Returns:
        {chunk_index: (data, start_ms, duration_ms)}
    """
    short = match_id[:8]
    manifest_path = MANIFEST_DIR / f"{short}.json"
    chunks_dir = CACHE_DIR / short

    if not manifest_path.exists() or not chunks_dir.exists():
        return {}

    manifest = json.loads(manifest_path.read_text())
    # manifest["chunks"] contient [{index, chunk_type, start_ms, duration_ms, ...}]
    chunks_meta = {c["index"]: c for c in manifest.get("chunks", [])}

    result: dict[int, tuple[bytes, int, int]] = {}
    for bin_file in sorted(chunks_dir.glob("chunk_*.bin")):
        idx = int(bin_file.stem.split("_")[1])
        meta = chunks_meta.get(idx)
        if meta is None:
            continue
        data = bin_file.read_bytes()
        result[idx] = (data, meta["start_ms"], meta["duration_ms"])

    return result


def dispatch_by_formula(
    all_fire_events: list[dict],
    n_players: int,
) -> dict[int, list[dict]]:
    """Dispatch fire events par pi via fire_seq % n_players (sans cross-ref timeline)."""
    by_pi: dict[int, list[dict]] = defaultdict(list)
    for ev in all_fire_events:
        pi = ev["fire_seq"] % n_players
        ev_copy = dict(ev)
        ev_copy["player_index"] = pi
        by_pi[pi].append(ev_copy)
    return dict(by_pi)


def resolve_weapon_from_ns(
    weapon_bytes: bytes,
    chunk_idx: int,
    pi: int,
    timeline_ns: dict[int, dict[int, bytes]],
) -> bytes:
    """Si weapon_bytes est un handle raw FA inconnu, tente une résolution NS.

    Cherche dans timeline_ns[chunk_idx][pi] et les chunks voisins (±2).
    Retourne le TYPE ID NS si trouvé, sinon weapon_bytes inchangé.
    """
    if weapon_bytes in WEAPON_ID_MAP:
        return weapon_bytes
    chunks_sorted = sorted(timeline_ns)
    # Cherche dans le chunk courant et les voisins proches
    for candidate_idx in [chunk_idx, chunk_idx - 1, chunk_idx + 1, chunk_idx - 2, chunk_idx + 2]:
        ns_wid = timeline_ns.get(candidate_idx, {}).get(pi)
        if ns_wid is not None and ns_wid in WEAPON_ID_MAP:
            return ns_wid
    return weapon_bytes


def find_chunk_for_ts(
    ts_ms: float,
    chunks_sorted: list[int],
    timing: list[tuple[int, int]],
) -> int:
    """Retourne l'index du chunk couvrant ts_ms."""
    for i, cidx in enumerate(chunks_sorted):
        start, end = timing[i]
        if start <= ts_ms < end:
            return cidx
    return chunks_sorted[-1] if chunks_sorted else 0


# ── Analyse d'un seul match ─────────────────────────────────────────────────


def analyse_match(
    match_id: str,
    n_players: int,
    db_unknown_count: int,
) -> dict:
    """Analyse un match et retourne les métriques comparatives."""
    chunks = load_chunks_from_cache(match_id)
    if not chunks:
        return {"match_id": match_id, "error": "no_chunks"}

    # Timelines Section 1 (raw + NS)
    timeline, timeline_ns, swap_pis, timing = build_weapon_timelines(chunks)
    chunks_sorted = sorted(chunks)

    # Scan tous les fire events (marker fixe 0x26)
    all_raw_events: list[dict] = []
    for idx in chunks_sorted:
        data, start_ms, dur_ms = chunks[idx]
        events = scan_fire_events_all(data, start_ms, dur_ms)
        all_raw_events.extend(events)
    all_raw_events.sort(key=lambda e: e["timestamp_ms"])

    total_events = len(all_raw_events)
    if total_events == 0:
        return {"match_id": match_id, "error": "no_events"}

    # ── Méthode actuelle : NS timeline cross-ref ────────────────────────
    b2_to_pi_ns = map_b2_to_player(all_raw_events, timeline_ns, timing, chunks_sorted)
    events_by_pi_ns = group_events_by_pi(all_raw_events, b2_to_pi_ns)
    dispatched_ns = sum(len(v) for v in events_by_pi_ns.values())
    dropped_ns = total_events - dispatched_ns

    # ── Nouvelle méthode : fire_seq % n_players ─────────────────────────
    events_by_pi_formula = dispatch_by_formula(all_raw_events, n_players)
    dispatched_formula = sum(len(v) for v in events_by_pi_formula.values())
    # Pas de drop avec la formule (tous les b2 sont résolus par modulo)

    # ── Comparaison des deux méthodes sur les b2 résolus par NS ─────────
    agreement = 0
    disagreement = 0
    for ev in all_raw_events:
        b2 = ev["fire_seq"]
        pi_ns = b2_to_pi_ns.get(b2)
        if pi_ns is None:
            continue
        pi_formula = b2 % n_players
        if pi_ns == pi_formula:
            agreement += 1
        else:
            disagreement += 1

    # ── Résolution weapon_id via NS pour les handles raw FA ─────────────
    unknown_in_events = 0
    resolved_via_ns = 0
    still_unknown = 0

    for ev in all_raw_events:
        wb = ev["weapon_bytes"]
        if wb not in WEAPON_ID_MAP:
            unknown_in_events += 1
            pi_formula = ev["fire_seq"] % n_players
            chunk_idx = find_chunk_for_ts(ev["timestamp_ms"], chunks_sorted, timing)
            resolved = resolve_weapon_from_ns(wb, chunk_idx, pi_formula, timeline_ns)
            if resolved in WEAPON_ID_MAP:
                resolved_via_ns += 1
            else:
                still_unknown += 1

    # ── Couverture NS timeline par pi ───────────────────────────────────
    pi_with_ns = set()
    pi_without_ns = set()
    pi_formula_all = set(ev["fire_seq"] % n_players for ev in all_raw_events)
    for pidx in pi_formula_all:
        has_ns = any(pidx in chunk_state for chunk_state in timeline_ns.values())
        (pi_with_ns if has_ns else pi_without_ns).add(pidx)

    return {
        "match_id": match_id,
        "n_players": n_players,
        "total_events": total_events,
        # NS method
        "dispatched_ns": dispatched_ns,
        "dropped_ns": dropped_ns,
        "drop_rate_ns": dropped_ns / total_events * 100,
        # Formula method
        "dispatched_formula": dispatched_formula,
        "dropped_formula": 0,
        # Agreement
        "ns_resolved_b2": len(b2_to_pi_ns),
        "agree": agreement,
        "disagree": disagreement,
        "agree_rate": agreement / (agreement + disagreement) * 100 if (agreement + disagreement) else 0,
        # Weapon ID resolution
        "unknown_in_events": unknown_in_events,
        "resolved_via_ns": resolved_via_ns,
        "still_unknown_after_ns": still_unknown,
        "db_unknown": db_unknown_count,
        # NS coverage
        "pi_with_ns": sorted(pi_with_ns),
        "pi_without_ns": sorted(pi_without_ns),
    }


# ── Main ────────────────────────────────────────────────────────────────────


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--limit", type=int, default=0, help="Nombre max de matchs (0 = tous)")
    args = parser.parse_args()

    conn = duckdb.connect(str(SHARED_DB), read_only=True)
    conn.execute(f"ATTACH '{ROOT / 'data' / 'warehouse' / 'metadata.duckdb'}' AS metadata (READ_ONLY)")

    limit_clause = f"LIMIT {args.limit}" if args.limit > 0 else ""
    rows = conn.execute(f"""
        SELECT mr.match_id,
               COUNT(DISTINCT wk.xuid) as players,
               COUNT(wk.match_id) as kills,
               COUNT(CASE WHEN wk.attribution_path = 'fire_event' THEN 1 END) as fe_kills,
               COUNT(CASE WHEN wk.weapon_id IS NULL THEN 1 END) as null_wid,
               SUM(CASE WHEN wl.weapon_id IS NULL AND wk.weapon_id IS NOT NULL
                        THEN 1 ELSE 0 END) as unknown_wid
        FROM match_registry mr
        JOIN weapon_kills wk ON wk.match_id = mr.match_id
        LEFT JOIN metadata.weapon_labels wl
               ON CAST(wl.weapon_id AS UBIGINT) = CAST(wk.weapon_id AS UBIGINT)
        GROUP BY mr.match_id, mr.start_time
        ORDER BY mr.start_time DESC
        {limit_clause}
    """).fetchall()
    conn.close()

    console.rule("[bold cyan]inv#132 — Validation fire_seq % n_players = player_index")
    total_matches = len(rows)
    console.print(f"[dim]Matchs à analyser : {total_matches}[/dim]\n")

    results = []
    errors: Counter = Counter()
    for i, r in enumerate(rows, 1):
        match_id, n_players, kills, fe_kills, null_wid, unknown_wid = r
        if i % 50 == 1 or i == total_matches:
            console.print(f"  [cyan]{i}/{total_matches}[/cyan] — {match_id[:8]} …", end="\r")
        res = analyse_match(match_id, n_players, unknown_wid)
        if "error" in res:
            errors[res["error"]] += 1
        else:
            results.append(res)

    console.print(f"\n[green]Analysés : {len(results)}/{total_matches}[/green]  |  Erreurs : {dict(errors)}\n")

    if not results:
        console.print("[red]Aucun résultat[/red]")
        return

    # ── Regroupement par n_players ─────────────────────────────────────
    # Bandes : 8 (standard), 12 (ranked/big), 16+, autres
    def n_band(n: int) -> str:
        if n <= 8:
            return "≤8"
        if n <= 12:
            return "9-12"
        if n <= 16:
            return "13-16"
        return "17+"

    from collections import defaultdict as _dd
    by_band: dict[str, list[dict]] = _dd(list)
    for res in results:
        by_band[n_band(res["n_players"])].append(res)

    # ── Tableau synthèse par bande ─────────────────────────────────────
    console.rule("[bold]Synthèse par taille de match")
    t = Table(show_header=True, header_style="bold", show_footer=True)
    t.add_column("Taille", style="cyan", footer="TOTAL")
    t.add_column("Matchs", justify="right")
    t.add_column("Events", justify="right")
    t.add_column("Drop NS", justify="right")
    t.add_column("Drop NS %", justify="right")
    t.add_column("Drop formule", justify="right")
    t.add_column("Accord NS/formule", justify="right")
    t.add_column("Unknown wid DB", justify="right")
    t.add_column("Résolu NS", justify="right")

    total_ev = total_drop_ns = total_agree = total_disagree = 0
    total_uk_db = total_uk_ev = total_resolved = 0

    for band in ["≤8", "9-12", "13-16", "17+"]:
        band_res = by_band.get(band, [])
        if not band_res:
            continue
        ev = sum(r["total_events"] for r in band_res)
        drop_ns = sum(r["dropped_ns"] for r in band_res)
        ag = sum(r["agree"] for r in band_res)
        disag = sum(r["disagree"] for r in band_res)
        uk_db = sum(r["db_unknown"] for r in band_res)
        uk_ev = sum(r["unknown_in_events"] for r in band_res)
        res_ns = sum(r["resolved_via_ns"] for r in band_res)
        drop_pct = drop_ns * 100 // ev if ev else 0
        agree_pct = ag * 100 // (ag + disag) if (ag + disag) else 0
        res_pct = res_ns * 100 // uk_ev if uk_ev else 0

        drop_col = f"[red]{drop_ns:,}[/red]" if drop_pct > 5 else f"[green]{drop_ns:,}[/green]"
        drop_pct_col = f"[red]{drop_pct}%[/red]" if drop_pct > 5 else f"[green]{drop_pct}%[/green]"
        agree_col = f"[green]{agree_pct}%[/green] ({ag}/{ag+disag})" if (ag+disag) else "[dim]N/A[/dim]"
        res_col = f"[green]{res_ns}[/green] ({res_pct}%)" if uk_ev else "[dim]–[/dim]"

        t.add_row(
            band, str(len(band_res)), f"{ev:,}",
            drop_col, drop_pct_col, "[green]0[/green]",
            agree_col, str(uk_db), res_col,
        )
        total_ev += ev; total_drop_ns += drop_ns
        total_agree += ag; total_disagree += disag
        total_uk_db += uk_db; total_uk_ev += uk_ev; total_resolved += res_ns

    total_drop_pct = total_drop_ns * 100 // total_ev if total_ev else 0
    total_agree_pct = total_agree * 100 // (total_agree + total_disagree) if (total_agree + total_disagree) else 0
    total_res_pct = total_resolved * 100 // total_uk_ev if total_uk_ev else 0
    t.columns[1].footer = str(len(results))
    t.columns[2].footer = f"{total_ev:,}"
    t.columns[3].footer = f"{total_drop_ns:,}"
    t.columns[4].footer = f"{total_drop_pct}%"
    t.columns[5].footer = "0"
    t.columns[6].footer = f"{total_agree_pct}% ({total_agree}/{total_agree+total_disagree})"
    t.columns[7].footer = str(total_uk_db)
    t.columns[8].footer = f"{total_resolved} ({total_res_pct}%)" if total_uk_ev else "–"
    console.print(t)

    # ── Détail des matchs avec worst drop NS ──────────────────────────
    worst = sorted(results, key=lambda r: r["drop_rate_ns"], reverse=True)[:10]
    console.rule("[bold]Top 10 matchs avec le plus grand drop NS")
    t2 = Table(show_header=True, header_style="bold")
    t2.add_column("Match", style="cyan")
    t2.add_column("N", justify="right")
    t2.add_column("Events", justify="right")
    t2.add_column("Drop NS %", justify="right")
    t2.add_column("Pi actifs", justify="right")
    t2.add_column("Pi sans NS", justify="left")

    for res in worst:
        t2.add_row(
            res["match_id"][:8],
            str(res["n_players"]),
            str(res["total_events"]),
            f"[red]{res['drop_rate_ns']:.1f}%[/red]",
            str(len(res["pi_with_ns"]) + len(res["pi_without_ns"])),
            ", ".join(str(p) for p in res["pi_without_ns"]) or "–",
        )
    console.print(t2)

    # ── Bilan accord NS / formule ──────────────────────────────────────
    console.rule("[bold]Accord NS cross-ref vs formule directe")
    console.print(
        f"  Accord global        : [bold]{'[green]' if total_agree_pct >= 90 else '[yellow]'}{total_agree_pct}%"
        f"[/{'green]' if total_agree_pct >= 90 else 'yellow]'} ({total_agree}/{total_agree+total_disagree})[/bold]"
    )
    console.print(f"  Drop NS (évités)     : [bold green]{total_drop_ns:,}[/bold green] events recouvrés avec la formule")
    console.print(f"  Unknown weapon (DB)  : {total_uk_db:,} kills avec weapon_id inconnu")
    console.print()
    if total_agree_pct >= 90:
        console.print("[bold green]✓ Formule validée[/bold green] — accord ≥ 90% avec NS cross-ref")
    elif total_agree_pct >= 70:
        console.print("[bold yellow]⚠ Accord partiel[/bold yellow] — NS cross-ref et formule divergent sur certains cas")
    else:
        console.print("[bold red]✗ Divergence importante[/bold red] — investiguer les désaccords NS/formule")
    console.print()
    console.print("[dim]Note : un faible accord NS/formule peut indiquer que NS vote faux (ambiguïté multi-joueurs")
    console.print("même arme au même chunk) plutôt que la formule est incorrecte.[/dim]")


if __name__ == "__main__":
    main()
