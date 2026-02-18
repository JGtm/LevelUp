"""Save heatmap as HTML to inspect rendering."""

from src.app.main_helpers import mark_firefight
from src.ui.cache_loaders import _enrich_matches_df, _load_matches_duckdb_v4_polars
from src.visualization.distributions_outcomes import plot_win_ratio_heatmap

db_path = "data/players/Chocoboflor/stats.duckdb"
df = _load_matches_duckdb_v4_polars(db_path)
df = _enrich_matches_df(df)
df = mark_firefight(df)

fig = plot_win_ratio_heatmap(df, title=None, min_matches=2)
fig.write_html("_debug_heatmap.html")
print("Saved to _debug_heatmap.html")
print(f"Traces: {len(fig.data)}")
print(f"Z non-none: {sum(1 for row in fig.data[0].z for v in row if v is not None)}")
