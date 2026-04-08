#!/usr/bin/env python3
"""
compare_perf_lusr.py — Comparaison performance_score + LUSR local vs VPS

Pour chaque joueur, affiche les 25 derniers matchs avec :
  - match_id, date/heure, carte
  - performance_score local vs VPS
  - LUSR (rating_value) local vs VPS
  - Flag si différence détectée

Usage:
    python compare_perf_lusr.py

Prérequis :
  - DuckDB installé en local (pip install duckdb)
  - Alias SSH "levlup" configuré dans ~/.ssh/config
  - python3 + duckdb disponibles sur le VPS
"""

import json
import subprocess
import sys
from pathlib import Path

import duckdb

# ─── Config ──────────────────────────────────────────────────────────────────
LOCAL_BASE = Path("C:/Users/Guillaume/Downloads/Scripts/LevelUp")
VPS_BASE = "/app"
SSH_ALIAS = "lvelup"
DOCKER_CONTAINER = "levelup-levelup-1"
GAMERTAGS = ["Madina97294", "JGtm", "Chocoboflor"]
N_MATCHES = 25

# Tolérance pour considérer deux valeurs "différentes"
PERF_TOLERANCE = 0.001
LUSR_TOLERANCE = 0.01
PSA_TOLERANCE = 0  # entier exact

# Couleurs ANSI (désactiver si besoin)
USE_COLOR = True

RED = "\033[91m" if USE_COLOR else ""
YLW = "\033[93m" if USE_COLOR else ""
GRN = "\033[92m" if USE_COLOR else ""
RST = "\033[0m" if USE_COLOR else ""
BOLD = "\033[1m" if USE_COLOR else ""


# ─── Script exécuté sur le VPS via SSH stdin ─────────────────────────────────
_VPS_SCRIPT_TEMPLATE = """\
import duckdb, json, sys, os

PLAYER_DB = "__PLAYER_DB__"
BASE_DIR = "__BASE_DIR__"
N = __N__

shared_db = None
for name in ("shared_matches_v2.duckdb", "shared_matches.duckdb"):
    path = BASE_DIR + "/data/warehouse/" + name
    if os.path.exists(path):
        shared_db = path
        break

if shared_db is None:
    print(json.dumps({"error": "shared db introuvable"}))
    sys.exit(1)

if not os.path.exists(PLAYER_DB):
    print(json.dumps({"error": "player db introuvable: " + PLAYER_DB}))
    sys.exit(1)

try:
    con = duckdb.connect()
    con.execute("ATTACH '" + PLAYER_DB + "' AS pdb (READ_ONLY)")
    con.execute("ATTACH '" + shared_db + "' AS sdb (READ_ONLY)")

    sql = (
        "SELECT pme.match_id,"
        "       CAST(mr.start_time AS VARCHAR)[:16] AS start_time,"
        "       mr.map_name,"
        "       ROUND(CAST(pme.performance_score AS DOUBLE), 3) AS perf_score,"
        "       ROUND(CAST(msr.rating_value AS DOUBLE), 2) AS lusr,"
        "       msr.rating_type,"
        "       ROUND(CAST(msr.rating_delta AS DOUBLE), 2) AS lusr_delta,"
        "       CAST(COALESCE(psa.psa_score, 0) AS INTEGER) AS psa_score,"
        "       CAST(COALESCE(psa.psa_count, 0) AS INTEGER) AS psa_count"
        "  FROM pdb.player_match_enrichment pme"
        "  LEFT JOIN sdb.match_registry mr ON mr.match_id = pme.match_id"
        "  LEFT JOIN pdb.match_skill_rank msr ON msr.match_id = pme.match_id"
        "  LEFT JOIN ("
        "    SELECT match_id, SUM(award_score) AS psa_score, SUM(award_count) AS psa_count"
        "      FROM pdb.personal_score_awards GROUP BY match_id"
        "  ) psa ON psa.match_id = pme.match_id"
        "  ORDER BY mr.start_time DESC NULLS LAST"
        f" LIMIT {N}"
    )

    rows = con.execute(sql).fetchall()
    cols = ["match_id", "start_time", "map_name", "perf_score", "lusr",
            "rating_type", "lusr_delta", "psa_score", "psa_count"]
    data = [dict(zip(cols, r)) for r in rows]
    print(json.dumps(data))

except Exception as e:
    import traceback
    print(json.dumps({"error": str(e), "trace": traceback.format_exc()}))
"""


def _build_vps_script(gamertag: str) -> str:
    player_db = f"{VPS_BASE}/data/players/{gamertag}/stats.duckdb"
    return (
        _VPS_SCRIPT_TEMPLATE
        .replace("__PLAYER_DB__", player_db)
        .replace("__BASE_DIR__", VPS_BASE)
        .replace("__N__", str(N_MATCHES))
    )


# ─── Requête locale ───────────────────────────────────────────────────────────
def get_local_data(gamertag: str) -> list[dict]:
    shared_db = None
    for name in ("shared_matches_v2.duckdb", "shared_matches.duckdb"):
        p = LOCAL_BASE / "data" / "warehouse" / name
        if p.exists():
            shared_db = str(p).replace("\\", "/")
            break

    if not shared_db:
        _warn("Shared DB introuvable en local")
        return []

    player_db_path = LOCAL_BASE / "data" / "players" / gamertag / "stats.duckdb"
    if not player_db_path.exists():
        _warn(f"Player DB introuvable : {player_db_path}")
        return []

    player_db = str(player_db_path).replace("\\", "/")

    try:
        con = duckdb.connect()
        con.execute(f"ATTACH '{player_db}' AS pdb (READ_ONLY)")
        con.execute(f"ATTACH '{shared_db}' AS sdb (READ_ONLY)")

        rows = con.execute(f"""
            SELECT
                pme.match_id,
                CAST(mr.start_time AS VARCHAR)[:16] AS start_time,
                mr.map_name,
                ROUND(CAST(pme.performance_score AS DOUBLE), 3) AS perf_score,
                ROUND(CAST(msr.rating_value AS DOUBLE), 2) AS lusr,
                msr.rating_type,
                ROUND(CAST(msr.rating_delta AS DOUBLE), 2) AS lusr_delta,
                CAST(COALESCE(psa.psa_score, 0) AS INTEGER) AS psa_score,
                CAST(COALESCE(psa.psa_count, 0) AS INTEGER) AS psa_count
            FROM pdb.player_match_enrichment pme
            LEFT JOIN sdb.match_registry mr ON mr.match_id = pme.match_id
            LEFT JOIN pdb.match_skill_rank msr ON msr.match_id = pme.match_id
            LEFT JOIN (
                SELECT match_id, SUM(award_score) AS psa_score, SUM(award_count) AS psa_count
                FROM pdb.personal_score_awards GROUP BY match_id
            ) psa ON psa.match_id = pme.match_id
            ORDER BY mr.start_time DESC NULLS LAST
            LIMIT {N_MATCHES}
        """).fetchall()

        cols = ["match_id", "start_time", "map_name", "perf_score", "lusr",
                "rating_type", "lusr_delta", "psa_score", "psa_count"]
        return [dict(zip(cols, r)) for r in rows]

    except Exception as e:  # noqa: BLE001
        _warn(f"Erreur locale : {e}")
        return []


# ─── Requête VPS via SSH ──────────────────────────────────────────────────────
def get_vps_data(gamertag: str) -> list[dict]:
    script = _build_vps_script(gamertag)

    try:
        result = subprocess.run(
            ["ssh", SSH_ALIAS, f"docker exec -i {DOCKER_CONTAINER} python3"],
            input=script,
            capture_output=True,
            text=True,
            timeout=45,
        )

        if result.returncode != 0:
            _warn(f"SSH code {result.returncode} : {result.stderr.strip()[:200]}")
            return []

        stdout = result.stdout.strip()
        if not stdout:
            _warn("Sortie SSH vide")
            return []

        # Prendre la dernière ligne JSON valide (ignorer warnings pip, etc.)
        for line in reversed(stdout.splitlines()):
            line = line.strip()
            if line.startswith("[") or line.startswith("{"):
                data = json.loads(line)
                if isinstance(data, dict) and "error" in data:
                    _warn(f"VPS erreur : {data['error']}")
                    if "trace" in data:
                        print(f"    Traceback: {data['trace'][:400]}")
                    return []
                return data if isinstance(data, list) else []

        _warn("Pas de JSON dans la sortie VPS")
        print(f"    Sortie brute : {stdout[:300]}")
        return []

    except subprocess.TimeoutExpired:
        _warn("Timeout SSH (45s)")
        return []
    except FileNotFoundError:
        _warn("Commande 'ssh' introuvable — installer OpenSSH ou configurer le PATH")
        return []
    except Exception as e:  # noqa: BLE001
        _warn(f"Erreur SSH : {e}")
        return []


# ─── Comparaison & affichage ──────────────────────────────────────────────────
def compare_player(gamertag: str) -> None:
    sep = "-" * 130
    print(f"\n{BOLD}{sep}{RST}")
    print(f"{BOLD}  Joueur : {gamertag}{RST}")
    print(f"{BOLD}{sep}{RST}")

    print("  Recuperation locale...", end=" ", flush=True)
    local_data = get_local_data(gamertag)
    print(f"{GRN}{len(local_data)} matchs{RST}")

    print("  Recuperation VPS...", end=" ", flush=True)
    vps_data = get_vps_data(gamertag)
    print(f"{GRN}{len(vps_data)} matchs{RST}")

    if not local_data and not vps_data:
        print("  Aucune donnée disponible.")
        return

    local_by_id = {r["match_id"]: r for r in local_data}
    vps_by_id = {r["match_id"]: r for r in vps_data}

    # Union triée par date (les plus récents en premier)
    all_ids = sorted(
        set(local_by_id) | set(vps_by_id),
        key=lambda mid: (local_by_id.get(mid) or vps_by_id.get(mid)).get("start_time") or "",
        reverse=True,
    )[:N_MATCHES]

    # ─ Header tableau ─
    print()
    h_mid = "Match ID"
    h_dt = "Date & Heure"
    h_map = "Carte"
    h_pl = "Perf Local"
    h_pv = "Perf VPS"
    h_ll = "LUSR Local"
    h_lv = "LUSR VPS"
    h_sl = "PSA Local"
    h_sv = "PSA VPS"
    h_d = "Statut"
    print(f"  {BOLD}{h_mid:<38} {h_dt:<17} {h_map:<24} {h_pl:>10} {h_pv:>9} {h_ll:>10} {h_lv:>9} {h_sl:>9} {h_sv:>8}  {h_d}{RST}")
    print(f"  {'-'*38} {'-'*17} {'-'*24} {'-'*10} {'-'*9} {'-'*10} {'-'*9} {'-'*9} {'-'*8}  {'-'*18}")

    n_diff = 0
    n_local_only = 0
    n_vps_only = 0

    for mid in all_ids:
        loc = local_by_id.get(mid)
        vps = vps_by_id.get(mid)
        ref = loc or vps

        date_str = (ref.get("start_time") or "?")
        map_str = (ref.get("map_name") or "?")[:24]
        mid_disp = mid[:36] + ".." if len(mid) > 36 else mid

        def fmt_f(val: float | None, decimals: int) -> str:
            return f"{val:.{decimals}f}" if val is not None else "n/a"

        def fmt_i(val: int | None) -> str:
            return str(val) if val is not None else "n/a"

        perf_l = fmt_f(loc.get("perf_score") if loc else None, 3)
        perf_v = fmt_f(vps.get("perf_score") if vps else None, 3)
        lusr_l = fmt_f(loc.get("lusr") if loc else None, 2)
        lusr_v = fmt_f(vps.get("lusr") if vps else None, 2)
        psa_l = fmt_i(loc.get("psa_score") if loc else None)
        psa_v = fmt_i(vps.get("psa_score") if vps else None)

        # Détection différence
        color = ""
        diffs: list[str] = []

        if loc and vps:
            if abs((loc.get("perf_score") or 0.0) - (vps.get("perf_score") or 0.0)) > PERF_TOLERANCE:
                diffs.append("perf")
            lusr_diff = loc.get("lusr") is not None and vps.get("lusr") is not None and \
                        abs(loc["lusr"] - vps["lusr"]) > LUSR_TOLERANCE
            lusr_presence = (loc.get("lusr") is None) != (vps.get("lusr") is None)
            if lusr_diff or lusr_presence:
                diffs.append("LUSR")
            if abs((loc.get("psa_score") or 0) - (vps.get("psa_score") or 0)) > PSA_TOLERANCE:
                diffs.append("PSA")

            if diffs:
                flag = "DIFF " + "+".join(diffs)
                color = RED if len(diffs) >= 2 else YLW
                n_diff += 1
            else:
                flag = ""
                color = ""
        elif loc and not vps:
            flag = "local only"
            color = YLW
            n_local_only += 1
        else:  # vps only
            flag = "vps only"
            color = YLW
            n_vps_only += 1

        line = (
            f"  {mid_disp:<38} {date_str:<17} {map_str:<24}"
            f" {perf_l:>10} {perf_v:>9} {lusr_l:>10} {lusr_v:>9} {psa_l:>9} {psa_v:>8}  {flag}"
        )
        print(f"{color}{line}{RST}")

    print()
    total_diffs = n_diff + n_local_only + n_vps_only
    summary_color = RED if total_diffs > 0 else GRN
    print(
        f"  {BOLD}Resume{RST} : {len(all_ids)} matchs | "
        f"{summary_color}{total_diffs} difference(s){RST} "
        f"({n_diff} valeurs, {n_local_only} local-only, {n_vps_only} vps-only)"
    )


# ─── Point d'entrée ───────────────────────────────────────────────────────────
def _warn(msg: str) -> None:
    print(f"{YLW}  [!] {msg}{RST}", flush=True)


def main() -> None:
    print(f"{BOLD}{'='*130}")
    print("  Comparaison Local vs VPS - performance_score, LUSR & Personal Score Awards")
    print(f"  Joueurs : {', '.join(GAMERTAGS)} | {N_MATCHES} derniers matchs chacun")
    print(f"  Local : {LOCAL_BASE}")
    print(f"  VPS   : {SSH_ALIAS} -> docker({DOCKER_CONTAINER}) -> {VPS_BASE}")
    print(f"{'='*130}{RST}")

    for gt in GAMERTAGS:
        compare_player(gt)

    print(f"\n{BOLD}Terminé.{RST}\n")


if __name__ == "__main__":
    main()
