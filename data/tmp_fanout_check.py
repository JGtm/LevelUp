import duckdb

# Vérifie la dérive progressive de Chocoboflor (bug subtil ?)
con = duckdb.connect("data/players/Chocoboflor/stats.duckdb", read_only=True)

# Progression chronologique arena
rows = con.execute("""
    SELECT start_time, rating_value, tier_label, rating_delta
    FROM match_skill_rank
    WHERE playlist_group = 'arena'
    ORDER BY start_time
""").fetchall()

print(f"Chocoboflor arena : {len(rows)} matchs")
print(f"PREMIER : {rows[0][0]}  r={rows[0][1]:.1f}  tier={rows[0][2]}")
print(f"DERNIER : {rows[-1][0]}  r={rows[-1][1]:.1f}  tier={rows[-1][2]}")
print()

# Les 10 derniers
print("--- 10 derniers matchs ---")
for r in rows[-10:]:
    print(f"  {str(r[0])[:16]}  rating={r[1]:7.1f}  {str(r[2] or 'Non classe'):20s}  delta={r[3]}")

con.close()

# Vérifie pour JGtm la même chose
print()
con2 = duckdb.connect("data/players/JGtm/stats.duckdb", read_only=True)
rows2 = con2.execute("""
    SELECT start_time, rating_value, tier_label, rating_delta
    FROM match_skill_rank
    WHERE playlist_group = 'arena'
    ORDER BY start_time
""").fetchall()
print(f"JGtm arena : {len(rows2)} matchs")
print(f"PREMIER : {rows2[0][0]}  r={rows2[0][1]:.1f}  tier={rows2[0][2]}")
print(f"DERNIER : {rows2[-1][0]}  r={rows2[-1][1]:.1f}  tier={rows2[-1][2]}")
print()
print("--- 10 derniers matchs ---")
for r in rows2[-10:]:
    print(f"  {str(r[0])[:16]}  rating={r[1]:7.1f}  {str(r[2] or 'Non classe'):20s}  delta={r[3]}")
con2.close()
