"""Graphiques killer-victim : barres empilées, timeseries, heatmap.

Extraits de antagonist_charts.py — graphiques de niveau match (K/V pairs).
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.ui.i18n.viz import viz_t
from src.visualization._antagonist_colors import COLORS, PLAYER_COLORS
from src.visualization.theme import apply_halo_plot_style


def plot_killer_victim_stacked_bars(  # noqa: PLR0913
    pairs_df: pl.DataFrame,
    match_id: str | None = None,
    *,
    me_xuid: str | None = None,
    rank_by_xuid: dict[str, int] | None = None,
    title: str | None = None,
    height: int = 400,
    lang: str = "fr",
) -> go.Figure:
    """Graphique barres empilées : une ligne par tueur, segments = victimes."""
    colors = COLORS
    player_colors = PLAYER_COLORS
    if title is None:
        title = viz_t("title_elim_victim", lang)
    fig = go.Figure()

    if pairs_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": colors["neutral"]},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    filtered_df = pairs_df
    if match_id:
        filtered_df = filtered_df.filter(pl.col("match_id") == match_id)

    if filtered_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_match_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": colors["neutral"]},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    agg_df = filtered_df.group_by(
        "killer_xuid", "killer_gamertag", "victim_xuid", "victim_gamertag"
    ).agg(pl.col("kill_count").sum().alias("count"))

    killer_totals = agg_df.group_by("killer_xuid", "killer_gamertag").agg(
        pl.col("count").sum().alias("total")
    )
    killers_sorted: list[tuple[str, str, int]] = [
        (row["killer_xuid"], row["killer_gamertag"], row["total"])
        for row in killer_totals.iter_rows(named=True)
    ]
    rank_map = rank_by_xuid or {}
    killers_sorted.sort(key=lambda k: (rank_map.get(k[0], 999), -k[2], k[1]))
    killer_xuids = [k[0] for k in killers_sorted]
    killer_labels = [k[1] for k in killers_sorted]
    n_killers = len(killer_labels)

    victim_totals = (
        agg_df.group_by("victim_xuid", "victim_gamertag")
        .agg(pl.col("count").sum().alias("total"))
        .sort("total", descending=True)
    )
    victim_order: list[tuple[str, str]] = [
        (row["victim_xuid"], row["victim_gamertag"]) for row in victim_totals.iter_rows(named=True)
    ]

    kv_lookup: dict[tuple[str, str], int] = {}
    for row in agg_df.iter_rows(named=True):
        kv_lookup[(row["killer_xuid"], row["victim_xuid"])] = int(row["count"])

    for idx, (v_xuid, v_label) in enumerate(victim_order):
        color = player_colors[idx % len(player_colors)]
        x_vals = [kv_lookup.get((k_xuid, v_xuid), 0) for k_xuid in killer_xuids]
        if sum(x_vals) == 0:
            continue
        safe_v = str(v_label).replace("&", "&amp;").replace("<", "&lt;")
        fig.add_trace(
            go.Bar(
                name=v_label,
                y=killer_labels,
                x=x_vals,
                orientation="h",
                marker={"color": color},
                hovertemplate=f"<b>%{{y}}</b> → <b>{safe_v}</b><br>{viz_t('axis_kills', lang)}: %{{x}}<extra></extra>",
            )
        )

    fig.update_layout(
        barmode="stack",
        xaxis_title=viz_t("axis_frag_count", lang),
        yaxis_title=viz_t("axis_killer", lang),
        yaxis={"categoryorder": "array", "categoryarray": killer_labels, "autorange": "reversed"},
        margin={"l": 140},
        showlegend=True,
        legend={"orientation": "v", "yanchor": "top", "y": 1, "xanchor": "left", "x": 1.02},
    )

    plot_height = max(height, 80 + 24 * n_killers)
    return apply_halo_plot_style(fig, title=title, height=plot_height)


def _add_kd_cumulative_trace(
    fig: go.Figure,
    minutes: list,
    timeseries_df: pl.DataFrame,
    colors: dict,
    lang: str,
) -> None:
    """Ajoute la trace cumulative K/D en overlay si disponible."""
    cumulative = timeseries_df["cumulative_net_kd"].to_list()
    final_color = colors["positive_kd"] if cumulative[-1] >= 0 else colors["negative_kd"]
    fig.add_trace(
        go.Scatter(
            name=viz_t("trace_kd_cumul", lang),
            x=minutes,
            y=cumulative,
            mode="lines+markers",
            line={"color": final_color, "width": 3},
            marker={"size": 6},
            yaxis="y2",
            hovertemplate=viz_t("hover_net_kd_cumul", lang),
        )
    )


def plot_kd_timeseries(
    timeseries_df: pl.DataFrame,
    *,
    title: str | None = None,
    show_cumulative: bool = True,
    height: int = 350,
    lang: str = "fr",
) -> go.Figure:
    """Graphique timeseries du K/D par minute."""
    colors = COLORS
    title = title or viz_t("title_kd_per_min", lang)
    fig = go.Figure()

    if timeseries_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": colors["neutral"]},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    minutes = timeseries_df["minute"].to_list()
    kills = timeseries_df["kills"].to_list()
    deaths = timeseries_df["deaths"].to_list()
    timeseries_df["net_kd"].to_list()

    fig.add_trace(
        go.Bar(
            name=viz_t("trace_kills", lang),
            x=minutes,
            y=kills,
            marker={"color": colors["kills"], "opacity": 0.7},
            hovertemplate=viz_t("hover_kills_per_min", lang),
        )
    )

    fig.add_trace(
        go.Bar(
            name=viz_t("trace_deaths", lang),
            x=minutes,
            y=[-d for d in deaths],
            marker={"color": colors["deaths"], "opacity": 0.7},
            hovertemplate=viz_t("hover_deaths_per_min", lang),
            customdata=deaths,
        )
    )

    if show_cumulative and "cumulative_net_kd" in timeseries_df.columns:
        _add_kd_cumulative_trace(fig, minutes, timeseries_df, colors, lang)

    fig.update_layout(
        barmode="relative",
        xaxis_title=viz_t("axis_minute", lang),
        yaxis_title=viz_t("axis_kills_deaths", lang),
        yaxis2={
            "title": viz_t("trace_kd_cumul", lang),
            "overlaying": "y",
            "side": "right",
            "showgrid": False,
        },
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "xanchor": "center", "x": 0.5},
    )
    fig.add_hline(y=0, line_width=2, line_color="rgba(255,255,255,0.75)")

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_killer_victim_heatmap(
    matrix_df: pl.DataFrame,
    *,
    title: str | None = None,
    height: int = 500,
    lang: str = "fr",
) -> go.Figure:
    """Heatmap de la matrice killer-victim."""
    colors = COLORS
    if title is None:
        title = viz_t("title_killer_victim_matrix", lang)
    fig = go.Figure()

    if matrix_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": colors["neutral"]},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    killers = matrix_df["killer_gamertag"].to_list()
    victim_cols = [c for c in matrix_df.columns if c != "killer_gamertag"]

    z_values = []
    for row in matrix_df.iter_rows(named=True):
        z_values.append([row.get(col, 0) or 0 for col in victim_cols])

    fig.add_trace(
        go.Heatmap(
            z=z_values,
            x=victim_cols,
            y=killers,
            colorscale=[
                [0, "#1a1a2e"],
                [0.25, "#16213e"],
                [0.5, "#0f3460"],
                [0.75, "#e94560"],
                [1, "#ff6b6b"],
            ],
            hoverongaps=False,
            hovertemplate=viz_t("hover_heatmap_kills", lang),
        )
    )

    fig.update_layout(
        xaxis_title=viz_t("axis_victim", lang),
        yaxis_title=viz_t("axis_killer", lang),
        xaxis={"side": "bottom"},
    )

    return apply_halo_plot_style(fig, title=title, height=height)
