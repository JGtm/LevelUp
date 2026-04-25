#!/usr/bin/env python3
"""Analyse Prestige tuning — rapport de calibration post-alpha.

Lit `prestige_telemetry` (par joueur, stats.duckdb) et `prestige_events`
(cross-joueurs, shared_social.duckdb) pour produire un rapport texte
permettant de calibrer les seuils du système.

Sortie : rapport Markdown imprimable, écrit sur stdout.

Usage :
    python scripts/analyze_prestige_tuning.py \\
        --shared-social data/warehouse/shared_social.duckdb \\
        --players-dir data/players

Métriques calculées (Annexe E du plan IMPL_PRESTIGE.md) :
  - Distribution des paliers créés (Normal/Heroic/Legendary/Mythic)
  - Taux de complétion par palier
  - Temps moyen "created → completed" par palier
  - Distribution mode libre vs pilote
  - Anomalies : Mythic à 0 % ou Normal à 100 %
  - Total PP émis sur la période

Le script tourne sans modification de l'état (lecture seule).
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

try:
    import duckdb  # type: ignore
except ImportError:
    print("ERROR: duckdb python package required. Install via 'pip install duckdb'.",
          file=sys.stderr)
    sys.exit(2)


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--shared-social", required=True,
                   help="Chemin vers shared_social.duckdb")
    p.add_argument("--players-dir", required=True,
                   help="Dossier contenant data/players/{gamertag}/stats.duckdb")
    p.add_argument("--days", type=int, default=30,
                   help="Fenêtre d'analyse en jours (défaut: 30)")
    return p.parse_args()


def aggregate_telemetry(players_dir: Path, days: int) -> dict:
    """Aggrege la telemetrie de tous les joueurs."""
    counts = {"created": {"normal": 0, "heroic": 0, "legendary": 0, "mythic": 0}}
    completed = {"normal": 0, "heroic": 0, "legendary": 0, "mythic": 0}
    expired = {"normal": 0, "heroic": 0, "legendary": 0, "mythic": 0}
    abandoned = {"normal": 0, "heroic": 0, "legendary": 0, "mythic": 0}
    mode_libre = 0
    mode_pilote = 0
    total_time = {"normal": 0.0, "heroic": 0.0, "legendary": 0.0, "mythic": 0.0}
    total_time_n = {"normal": 0, "heroic": 0, "legendary": 0, "mythic": 0}

    for stats_db in players_dir.glob("*/stats.duckdb"):
        try:
            conn = duckdb.connect(str(stats_db), read_only=True)
        except Exception as e:
            print(f"WARN: cannot open {stats_db}: {e}", file=sys.stderr)
            continue

        try:
            rows = conn.execute(f"""
                SELECT event_type, palier, mode, time_since_create_seconds
                FROM prestige_telemetry
                WHERE created_at > NOW() - INTERVAL {int(days)} DAY
            """).fetchall()
        except Exception:
            # Table absente → joueur sans Prestige
            conn.close()
            continue
        conn.close()

        for event_type, palier, mode, time_secs in rows:
            tier = (palier or "").lower()
            if tier not in counts["created"]:
                tier = ""

            if event_type == "created" and tier:
                counts["created"][tier] += 1
                if mode == "libre":
                    mode_libre += 1
                elif mode == "pilote":
                    mode_pilote += 1
            elif event_type == "completed" and tier:
                completed[tier] += 1
                if time_secs:
                    total_time[tier] += float(time_secs)
                    total_time_n[tier] += 1
            elif event_type == "expired" and tier:
                expired[tier] += 1
            elif event_type == "abandoned" and tier:
                abandoned[tier] += 1

    return {
        "created": counts["created"],
        "completed": completed,
        "expired": expired,
        "abandoned": abandoned,
        "mode_libre": mode_libre,
        "mode_pilote": mode_pilote,
        "avg_time_seconds": {
            t: (total_time[t] / total_time_n[t]) if total_time_n[t] > 0 else 0
            for t in ("normal", "heroic", "legendary", "mythic")
        },
    }


def total_pp_emitted(shared_social: Path, days: int) -> dict:
    """Somme des PP emises par source sur la fenetre."""
    try:
        conn = duckdb.connect(str(shared_social), read_only=True)
    except Exception as e:
        print(f"WARN: cannot open {shared_social}: {e}", file=sys.stderr)
        return {}

    try:
        rows = conn.execute(f"""
            SELECT source_type, COALESCE(tier, ''), COUNT(*) AS n, SUM(pp_amount) AS total
            FROM prestige_events
            WHERE created_at > NOW() - INTERVAL {int(days)} DAY
            GROUP BY source_type, tier
            ORDER BY total DESC
        """).fetchall()
    except Exception:
        rows = []
    conn.close()
    return {(src, tier): {"count": n, "total_pp": total} for src, tier, n, total in rows}


def render_report(telemetry: dict, pp_data: dict, days: int) -> str:
    out = []
    out.append(f"# Prestige tuning report ({days}-day window)\n")

    out.append("## Distribution paliers a la creation\n")
    out.append("| Palier | Crees | Completes | Taux | Expires | Abandonnes | Temps moyen (s) |")
    out.append("|--------|------:|----------:|-----:|--------:|-----------:|----------------:|")
    for tier in ("normal", "heroic", "legendary", "mythic"):
        created = telemetry["created"][tier]
        comp = telemetry["completed"][tier]
        rate = (comp / created * 100) if created > 0 else 0.0
        avg_t = telemetry["avg_time_seconds"][tier]
        out.append(f"| {tier.capitalize():<9} | {created:>5} | {comp:>9} | {rate:>4.0f}% | "
                   f"{telemetry['expired'][tier]:>7} | {telemetry['abandoned'][tier]:>10} | "
                   f"{avg_t:>15.0f} |")

    out.append("\n## Mode libre vs pilote\n")
    total_mode = telemetry["mode_libre"] + telemetry["mode_pilote"]
    if total_mode > 0:
        ratio_libre = telemetry["mode_libre"] / total_mode * 100
        out.append(f"- Mode libre  : {telemetry['mode_libre']} ({ratio_libre:.0f}%)")
        out.append(f"- Mode pilote : {telemetry['mode_pilote']} ({100 - ratio_libre:.0f}%)")
    else:
        out.append("(aucune donnee)")

    out.append("\n## PP emis par source\n")
    if pp_data:
        out.append("| Source | Tier | Count | Total PP |")
        out.append("|--------|------|------:|---------:|")
        for (src, tier), v in pp_data.items():
            tier_disp = tier or "-"
            out.append(f"| {src} | {tier_disp} | {v['count']} | {v['total_pp']} |")
    else:
        out.append("(aucun evenement PP)")

    out.append("\n## Anomalies detectees\n")
    anomalies = []
    for tier in ("normal", "heroic", "legendary", "mythic"):
        created = telemetry["created"][tier]
        if created == 0:
            continue
        comp = telemetry["completed"][tier]
        rate = comp / created * 100
        if tier == "mythic" and rate < 1:
            anomalies.append(f"- **Mythic completion rate near 0%** ({rate:.0f}%) - "
                             "stretch threshold trop strict ?")
        if tier == "normal" and rate > 95:
            anomalies.append(f"- **Normal completion rate trop eleve** ({rate:.0f}%) - "
                             "stretch threshold trop laxiste ?")
        if tier == "heroic" and (rate < 30 or rate > 80):
            anomalies.append(f"- Heroic completion rate hors cible ({rate:.0f}%, cible 50-65%)")
    if not anomalies:
        anomalies.append("- Aucune anomalie majeure detectee")
    out.extend(anomalies)

    return "\n".join(out)


def main() -> int:
    args = parse_args()
    shared_social = Path(args.shared_social)
    players_dir = Path(args.players_dir)

    if not shared_social.exists():
        print(f"ERROR: {shared_social} not found", file=sys.stderr)
        return 1
    if not players_dir.is_dir():
        print(f"ERROR: {players_dir} not a directory", file=sys.stderr)
        return 1

    telemetry = aggregate_telemetry(players_dir, args.days)
    pp_data = total_pp_emitted(shared_social, args.days)
    print(render_report(telemetry, pp_data, args.days))
    return 0


if __name__ == "__main__":
    sys.exit(main())
