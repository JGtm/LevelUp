# -*- coding: utf-8 -*-
"""Génère .ai/suspects_film_start_jgtm.md depuis le log backfill."""
import duckdb
import os
import re

LOG_PATH = "data/logs/film_start_backfill.log"
CHUNK_DIR = "data/cache/film_chunks"
DB_PATH = "data/warehouse/shared_matches_v2.duckdb"
JGTM_XUID = "2533274823110022"
GAMERTAG = "JGtm"
BASE_URL = f"https://www.halowaypoint.com/halo-infinite/players/{GAMERTAG}/matches"
OUT_PATH = ".ai/suspects_film_start_jgtm.md"

# -- Extraire suspects du log ------------------------------------------------
log = open(LOG_PATH, encoding="utf-8", errors="replace").read()
suspects_gap: dict[str, float] = {}
current_match = None
for line in log.splitlines():
    if line.startswith("MATCH "):
        current_match = line.replace("MATCH ", "").strip()
    if current_match and "CORR SUSPECT" in line:
        m = re.search(r"gap min=([+-][\d.]+)s", line)
        if m:
            suspects_gap[current_match] = float(m.group(1))
            current_match = None

ids = list(suspects_gap.keys())
ph = ", ".join(["?" for _ in ids])

# -- Requête DB --------------------------------------------------------------
con = duckdb.connect(DB_PATH, read_only=True)
rows = con.execute(
    f"""
    SELECT r.match_id, r.start_time,
           COALESCE(r.map_name, r.map_id, '?') AS map_name,
           COALESCE(r.playlist_name, r.game_variant_name, r.mode_category, '?') AS mode_label,
           r.film_match_start_ms
    FROM match_registry r
    JOIN match_participants mp ON mp.match_id = r.match_id
    WHERE r.match_id IN ({ph}) AND mp.xuid = ?
    ORDER BY r.start_time
    """,
    [*ids, JGTM_XUID],
).fetchall()
con.close()

results = []
for mid, dt, map_name, mode, db_start in rows:
    short = mid[:8]
    local_dir = f"{CHUNK_DIR}/{short}"
    chunks_local = sorted(os.listdir(local_dir)) if os.path.isdir(local_dir) else []
    gap = suspects_gap.get(mid, 0)
    results.append((mid, dt, map_name, mode, db_start, gap, chunks_local))

pvp = [(r[0], r[1]) for r in results if "firefight" not in r[3].lower()]
ignored = [r for r in results if "firefight" in r[3].lower()]

# -- Génération du doc -------------------------------------------------------
doc = []
doc.append("# Suspects film_match_start \u2014 JGtm")
doc.append("")
doc.append("> Matchs dont la corr\u00e9lation filmshell \u2194 highlight_events est suspecte")
doc.append("> (**gap_min > +60s** : l'estimation est trop pr\u00e9coce, captur\u00e9e pendant le countdown).")
doc.append("")
doc.append(f"**Nb total** : {len(results)} matchs suspects ({len(pvp)} PvP + {len(ignored)} Firefight ignor\u00e9)")
doc.append("")
doc.append("---")
doc.append("")
doc.append("## Commande de retraitement (autre PC avec chunks locaux)")
doc.append("")
doc.append("Copier le r\u00e9pertoire `data/cache/film_chunks/` de l'autre PC vers ce PC (ou lancer")
doc.append("directement sur l'autre PC), puis ex\u00e9cuter :")
doc.append("")
doc.append("```bash")
doc.append("# --max-chunks 10 = couvre jusqu'a 200s, d\u00e9tecte les countdowns longs")
mids_join = " ".join(m for m, _ in pvp)
doc.append(f"for mid in {mids_join}; do")
doc.append("  python scripts/_exp_spawn_download.py \\")
doc.append("    --match-id \"$mid\" \\")
doc.append("    --cached-only \\")
doc.append("    --write-db \\")
doc.append("    --max-chunks 10")
doc.append("done")
doc.append("```")
doc.append("")
doc.append("> `--match-id` ignore `--skip-done` et **\u00e9crase** la valeur d\u00e9j\u00e0 en DB.")
doc.append("")
doc.append("---")
doc.append("")
doc.append("## Matchs \u00e0 v\u00e9rifier en mode cin\u00e9ma")
doc.append("")
doc.append("| # | Date | Heure | Map | Mode | Chunks locaux | dbstart | gap_min | Lien |")
doc.append("|---|------|-------|-----|------|:---:|---:|---:|:---:|")

n = 0
for mid, dt, map_name, mode, db_start, gap, chunks_local in results:
    if "firefight" in mode.lower():
        continue
    n += 1
    date_str = dt.strftime("%Y-%m-%d")
    time_str = dt.strftime("%H:%M")
    start_s = f"{db_start / 1000:.1f}s" if db_start else "NULL"
    gap_str = f"+{gap:.0f}s"
    nb = str(len(chunks_local)) if chunks_local else "0"
    link = f"[Waypoint]({BASE_URL}/{mid})"
    doc.append(f"| {n} | {date_str} | {time_str} | {map_name} | {mode} | {nb} | {start_s} | {gap_str} | {link} |")

doc.append("")
if ignored:
    doc.append("---")
    doc.append("")
    doc.append("## Ignor\u00e9s (Firefight \u2014 pas de film POV)")
    doc.append("")
    for mid, dt, map_name, mode, db_start, gap, chunks_local in ignored:
        doc.append(
            f"- `{mid[:8]}\u2026` \u2014 {dt.strftime('%Y-%m-%d %H:%M')}"
            f" \u2014 {map_name} \u2014 gap={gap:+.0f}s (normal\u00a0: Firefight sans events kill/death)"
        )

doc.append("")
doc.append("---")
doc.append("")
doc.append("## Interpr\u00e9tation du gap")
doc.append("")
doc.append("- **gap_min ~+3s** : estimation correcte (premier frag 3s apr\u00e8s le d\u00e9but du match)")
doc.append("- **gap_min ~+60-80s** : estimation trop pr\u00e9coce \u2014 le script a d\u00e9tect\u00e9 un mouvement")
doc.append("  pendant le **countdown** (~3-5s dans l'enregistrement) alors que le match a")
doc.append("  vraiment commenc\u00e9 ~63-85s plus tard.")
doc.append("- **Correction attendue** : avec `--max-chunks 10`, la d\u00e9tection remontera les")
doc.append("  chunks suivants et trouvera la vraie rupture de mouvement post-countdown.")

out = "\n".join(doc)
with open(OUT_PATH, "w", encoding="utf-8") as f:
    f.write(out)
print(f"Genere: {OUT_PATH} ({len(pvp)} PvP, {len(ignored)} ignores)")
