import duckdb

players = [
    "Chocoboflor",
    "JGtm",
    "Madina97294",
    "XxDaemonGamerxX",
]

for p in players:
    db = f"data/players/{p}/stats.duckdb"
    try:
        con = duckdb.connect(db, read_only=True)
        # Résumé par groupe
        groups = con.execute("""
            SELECT playlist_group, COUNT(*) as n,
                   MIN(start_time) as first, MAX(start_time) as last,
                   MIN(rating_value) as min_r, MAX(rating_value) as max_r,
                   LAST(rating_value ORDER BY start_time) as last_r,
                   LAST(tier_label ORDER BY start_time) as last_tier
            FROM match_skill_rank
            GROUP BY playlist_group
            ORDER BY playlist_group
        """).fetchall()
        print(f"\n=== {p} ===")
        for g in groups:
            print(
                f"  [{g[0]:10s}] n={g[1]:3d}  last_rating={g[6]:7.1f} ({g[7] or 'Non classe'})  "
                f"min={g[4]:.0f}  max={g[5]:.0f}"
            )

        # Détecte drops brutaux (delta entre matchs consécutifs > 50 pts)
        drops = con.execute("""
            SELECT start_time, playlist_group, rating_value,
                   LAG(rating_value) OVER (PARTITION BY playlist_group ORDER BY start_time) as prev_r,
                   rating_value - LAG(rating_value) OVER (PARTITION BY playlist_group ORDER BY start_time) as drop
            FROM match_skill_rank
            QUALIFY ABS(drop) > 50
            ORDER BY drop ASC
            LIMIT 10
        """).fetchall()
        if drops:
            print(f"  * DROPS > 50 pts :")
            for d in drops:
                print(f"    {str(d[0])[:16]} [{d[1]}] {d[3]:.0f} -> {d[2]:.0f}  ({d[4]:+.0f})")
        con.close()
    except Exception as e:
        print(f"\n=== {p} === ERREUR: {e}")
