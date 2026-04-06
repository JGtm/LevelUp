import duckdb

c = duckdb.connect("data/players/Madina97294/stats.duckdb", read_only=True)
rows = c.execute(
    "SELECT start_time, rating_value, rating_deviation, tier_label, rating_delta FROM match_skill_rank WHERE playlist_group='arena' ORDER BY start_time DESC LIMIT 10"
).fetchall()
for r in rows:
    print(f"{str(r[0])[:16]} rating={r[1]:.1f} sigma={r[2]} tier={r[3]} delta={r[4]}")
c.close()
