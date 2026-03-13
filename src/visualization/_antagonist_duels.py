"""Graphiques antagonistes : duels, résumé nemesis/victimes, top antagonists, indicateur K/D.

Extraits de antagonist_charts.py — graphiques de niveau rival/antagoniste.
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl
from plotly.subplots import make_subplots

from src.ui.i18n.viz import viz_t
from src.visualization._antagonist_colors import COLORS
from src.visualization.theme import apply_halo_plot_style


def plot_duel_history(  # noqa: C901, PLR0912, PLR0913
    duel_df: pl.DataFrame,
    me_gamertag: str,
    opponent_gamertag: str,
    *,
    title: str | None = None,
    height: int = 300,
    lang: str = "fr",
) -> go.Figure:
    """Graphique d'historique de duels entre deux joueurs (barres groupées + net)."""
    colors = COLORS
    if title is None:
        title = f"Historique des duels : {me_gamertag} vs {opponent_gamertag}"
    fig = go.Figure()

    if duel_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_duel", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": colors["neutral"]},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    matches = list(range(len(duel_df)))
    my_kills = duel_df["my_kills"].to_list()
    opponent_kills = duel_df["opponent_kills"].to_list()
    net_values = duel_df["net"].to_list()

    fig.add_trace(
        go.Bar(
            name=me_gamertag,
            x=matches,
            y=my_kills,
            marker={"color": colors["kills"]},
            hovertemplate=f"<b>{me_gamertag}</b><br>"
            f"{viz_t('axis_kills', lang)}: %{{y}}<extra></extra>",
        )
    )

    fig.add_trace(
        go.Bar(
            name=opponent_gamertag,
            x=matches,
            y=opponent_kills,
            marker={"color": colors["deaths"]},
            hovertemplate=f"<b>{opponent_gamertag}</b><br>"
            f"{viz_t('axis_kills', lang)}: %{{y}}<extra></extra>",
        )
    )

    fig.add_trace(
        go.Scatter(
            name=viz_t("label_net", lang),
            x=matches,
            y=net_values,
            mode="lines+markers",
            line={"color": colors["highlight"], "width": 2},
            yaxis="y2",
            hovertemplate="Net: %{y:+d}<extra></extra>",
        )
    )

    total_my = sum(my_kills)
    total_opp = sum(opponent_kills)
    total_net = total_my - total_opp
    win_status = (
        viz_t("label_win", lang)
        if total_net > 0
        else (viz_t("label_tie", lang) if total_net == 0 else viz_t("label_loss", lang))
    )

    fig.add_annotation(
        text=f"Total: {total_my}-{total_opp} ({win_status})",
        xref="paper",
        yref="paper",
        x=1,
        y=1.15,
        showarrow=False,
        font={
            "size": 14,
            "color": colors["positive_kd"] if total_net >= 0 else colors["negative_kd"],
        },
        xanchor="right",
    )

    fig.update_layout(
        barmode="group",
        xaxis_title=viz_t("axis_match_number", lang),
        yaxis_title=viz_t("axis_frag_count", lang),
        yaxis2={
            "title": viz_t("label_balance", lang),
            "overlaying": "y",
            "side": "right",
            "showgrid": False,
        },
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "xanchor": "center", "x": 0.5},
    )

    fig.add_hline(y=0, line_width=1, line_color=colors["neutral"], line_dash="dash", yref="y2")

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_nemesis_victim_summary(
    nemesis_data: dict,
    victim_data: dict,
    *,
    title: str | None = None,
    height: int = 250,
    lang: str = "fr",
) -> go.Figure:
    """Indicateurs résumé : nemesis et victime principale."""
    colors = COLORS
    if title is None:
        title = viz_t("title_nemesis_victim", lang)

    fig = make_subplots(
        rows=1,
        cols=2,
        subplot_titles=(viz_t("label_nemesis", lang), viz_t("label_punching_bag", lang)),
        specs=[[{"type": "indicator"}, {"type": "indicator"}]],
    )

    nem_gt = nemesis_data.get("gamertag", "N/A")
    nem_count = nemesis_data.get("times_killed_by", 0)
    fig.add_trace(
        go.Indicator(
            mode="number",
            value=nem_count,
            number={
                "suffix": f" {viz_t('suffix_deaths', lang)}",
                "font": {"size": 36, "color": colors["nemesis"]},
            },
            title={"text": f"<b>{nem_gt}</b>", "font": {"size": 16}},
        ),
        row=1,
        col=1,
    )

    vic_gt = victim_data.get("gamertag", "N/A")
    vic_count = victim_data.get("times_killed", 0)
    fig.add_trace(
        go.Indicator(
            mode="number",
            value=vic_count,
            number={
                "suffix": f" {viz_t('suffix_kills', lang)}",
                "font": {"size": 36, "color": colors["victim"]},
            },
            title={"text": f"<b>{vic_gt}</b>", "font": {"size": 16}},
        ),
        row=1,
        col=2,
    )

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_top_antagonists_bars(  # noqa: PLR0913
    nemeses: list[dict],
    victims: list[dict],
    *,
    top_n: int = 5,
    title: str | None = None,
    height: int = 400,
    lang: str = "fr",
) -> go.Figure:
    """Barres horizontales des top antagonistes (némésis et souffre-douleurs)."""
    colors = COLORS
    if title is None:
        title = viz_t("title_top_antagonists", lang)

    fig = make_subplots(
        rows=1,
        cols=2,
        subplot_titles=(
            f"Top {top_n} {viz_t('label_nemesis', lang)}",
            f"Top {top_n} {viz_t('label_punching_bag', lang)}",
        ),
        horizontal_spacing=0.15,
    )

    top_nemeses = nemeses[:top_n] if nemeses else []
    if top_nemeses:
        nem_names = [n.get("killer_gamertag", "?") for n in reversed(top_nemeses)]
        nem_counts = [n.get("times_killed_by", 0) for n in reversed(top_nemeses)]
        fig.add_trace(
            go.Bar(
                y=nem_names,
                x=nem_counts,
                orientation="h",
                marker={"color": colors["nemesis"]},
                name=viz_t("trace_deaths", lang),
                hovertemplate=viz_t("hover_killed_by", lang),
            ),
            row=1,
            col=1,
        )

    top_victims = victims[:top_n] if victims else []
    if top_victims:
        vic_names = [v.get("victim_gamertag", "?") for v in reversed(top_victims)]
        vic_counts = [v.get("times_killed", 0) for v in reversed(top_victims)]
        fig.add_trace(
            go.Bar(
                y=vic_names,
                x=vic_counts,
                orientation="h",
                marker={"color": colors["victim"]},
                name=viz_t("trace_kills", lang),
                hovertemplate=viz_t("hover_i_killed", lang),
            ),
            row=1,
            col=2,
        )

    fig.update_layout(showlegend=False)
    fig.update_xaxes(title_text=viz_t("axis_deaths", lang), row=1, col=1)
    fig.update_xaxes(title_text=viz_t("axis_kills", lang), row=1, col=2)

    return apply_halo_plot_style(fig, title=title, height=height)


def get_antagonist_chart_colors() -> dict[str, str]:
    """Retourne une copie des couleurs des graphiques antagonistes."""
    return COLORS.copy()


def create_kd_indicator(
    kills: int,
    deaths: int,
    *,
    title: str = "F/D",
    height: int = 150,
) -> go.Figure:
    """Indicateur K/D unique avec delta par rapport à 1.0."""
    colors = COLORS
    kd_ratio = kills / deaths if deaths > 0 else kills

    if kd_ratio >= 1.5:
        color = colors["positive_kd"]
    elif kd_ratio >= 1.0:
        color = colors["highlight"]
    else:
        color = colors["negative_kd"]

    fig = go.Figure()
    fig.add_trace(
        go.Indicator(
            mode="number+delta",
            value=kd_ratio,
            number={"valueformat": ".2f", "font": {"size": 48, "color": color}},
            delta={
                "reference": 1.0,
                "valueformat": ".2f",
                "increasing": {"color": colors["positive_kd"]},
                "decreasing": {"color": colors["negative_kd"]},
            },
            title={
                "text": f"<b>{title}</b><br><span style='font-size:12px'>{kills}K / {deaths}D</span>",
                "font": {"size": 14},
            },
        )
    )

    return apply_halo_plot_style(fig, title="", height=height)
