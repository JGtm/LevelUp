"""Graphiques liés aux résultats (victoires/défaites/heatmap/top).

Extrait de distributions.py (Sprint 16 – refactoring).
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.config import HALO_COLORS, OUTCOME_CODES, PLOT_CONFIG
from src.ui.i18n.viz import viz_t
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization._distributions_outcomes_helpers import (
    add_outcome_traces,
    build_outcome_pivot,
    compute_outcome_buckets,
    determine_top_period,
    ensure_datetime,
    safe_col,
)
from src.visualization.theme import (
    apply_halo_plot_style,
    get_legend_horizontal_bottom,
)


def plot_outcomes_over_time(
    df: DataFrameLike, *, session_style: bool = False, lang: str = "fr"
) -> tuple[go.Figure, str]:
    """Graphique d'évolution des victoires/défaites dans le temps.

    Returns:
        Tuple (figure, bucket_label).
    """
    d = ensure_polars(df)
    colors = HALO_COLORS.as_dict()
    d = d.drop_nulls(subset=["outcome"])

    if d.is_empty():
        fig = go.Figure()
        fig.update_layout(
            height=PLOT_CONFIG.default_height, margin={"l": 40, "r": 20, "t": 30, "b": 40}
        )
        fig.update_yaxes(title_text=viz_t("axis_count", lang))
        return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height), viz_t(
            "bucket_period", lang
        )

    d, bucket_label = compute_outcome_buckets(d, session_style=session_style, lang=lang)

    # Pivot : bucket × outcome → count
    pivot = (
        d.group_by("bucket", "outcome")
        .agg(pl.col("match_id").count().alias("count"))
        .pivot(on="outcome", index="bucket", values="count")
        .fill_null(0)
        .sort("bucket")
    )

    # Extraire les séries par code outcome (colonnes nommées en str)
    buckets = pivot["bucket"].to_list()
    wins = safe_col(pivot, str(OUTCOME_CODES.WIN))
    losses = safe_col(pivot, str(OUTCOME_CODES.LOSS))
    ties = safe_col(pivot, str(OUTCOME_CODES.TIE))
    nofin = safe_col(pivot, str(OUTCOME_CODES.NO_FINISH))
    losses_neg = [-v for v in losses]

    _wins_lbl = viz_t("trace_wins", lang)
    _losses_lbl = viz_t("trace_losses", lang)
    _ties_lbl = viz_t("trace_ties", lang)
    _unfinished_lbl = viz_t("trace_unfinished", lang)

    fig = go.Figure()
    fig.add_bar(
        x=buckets,
        y=wins,
        name=_wins_lbl,
        marker_color=colors["green"],
        hovertemplate=f"%{{x}}<br>{_wins_lbl}: %{{y}}<extra></extra>",
    )
    fig.add_bar(
        x=buckets,
        y=losses_neg,
        name=_losses_lbl,
        marker_color=colors["red"],
        customdata=losses,
        hovertemplate=f"%{{x}}<br>{_losses_lbl}: %{{customdata}}<extra></extra>",
    )

    if sum(ties) > 0:
        fig.add_bar(
            x=buckets,
            y=ties,
            name=_ties_lbl,
            marker_color=colors["violet"],
            hovertemplate=f"%{{x}}<br>{_ties_lbl}: %{{y}}<extra></extra>",
        )
    if sum(nofin) > 0:
        fig.add_bar(
            x=buckets,
            y=nofin,
            name=_unfinished_lbl,
            marker_color=colors["violet"],
            hovertemplate=f"%{{x}}<br>{_unfinished_lbl}: %{{y}}<extra></extra>",
        )

    fig.update_layout(
        barmode="relative",
        height=PLOT_CONFIG.default_height,
        margin={"l": 40, "r": 20, "t": 30, "b": 40},
    )
    fig.update_yaxes(title_text=viz_t("axis_count", lang), zeroline=True)

    if bucket_label == viz_t("bucket_match", lang) and len(buckets) > 30:
        fig.update_xaxes(showticklabels=False, title_text="")

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height), bucket_label


# ---------------------------------------------------------------------------
# plot_stacked_outcomes_by_category
# ---------------------------------------------------------------------------


def plot_stacked_outcomes_by_category(  # noqa: PLR0913
    df: DataFrameLike,
    category_col: str,
    *,
    title: str | None = None,
    min_matches: int = 1,
    sort_by: str = "total",
    max_categories: int = 20,
    lang: str = "fr",
) -> go.Figure:
    """Graphique de colonnes empilées Win/Loss/Tie/Left par catégorie."""
    d = ensure_polars(df)
    colors = HALO_COLORS.as_dict()
    d = d.drop_nulls(subset=[category_col, "outcome"])

    if d.is_empty():
        fig = go.Figure()
        fig.update_layout(height=PLOT_CONFIG.default_height)
        return apply_halo_plot_style(fig, title=title)

    pivot = build_outcome_pivot(d, category_col, min_matches, sort_by, max_categories)
    if pivot is None:
        fig = go.Figure()
        fig.update_layout(height=PLOT_CONFIG.default_height)
        return apply_halo_plot_style(fig, title=title)

    fig = go.Figure()
    add_outcome_traces(fig, pivot, colors, category_col=category_col, lang=lang)

    height = PLOT_CONFIG.tall_height if pivot.height > 10 else PLOT_CONFIG.default_height
    fig.update_layout(
        barmode="stack",
        bargap=0.15,
        height=height,
        margin={"l": 40, "r": 20, "t": 60 if title else 30, "b": 100},
        legend=get_legend_horizontal_bottom(),
    )
    fig.update_xaxes(tickangle=45, title_text="")
    fig.update_yaxes(title_text=viz_t("trace_matches", lang))
    return apply_halo_plot_style(fig, title=title, height=height)


# ---------------------------------------------------------------------------
# plot_win_ratio_heatmap  (98L – OK, déplacé tel quel)
# ---------------------------------------------------------------------------


def plot_win_ratio_heatmap(
    df: DataFrameLike,
    *,
    title: str | None = None,
    min_matches: int = 2,
    lang: str = "fr",
) -> go.Figure | None:
    """Heatmap du Win Ratio par jour de la semaine et heure.

    Retourne ``None`` si les données sont insuffisantes pour afficher
    au moins une cellule avec un win rate significatif.
    """
    import numpy as np

    d = ensure_polars(df)
    colors = HALO_COLORS.as_dict()
    d = d.drop_nulls(subset=["start_time", "outcome"])

    if d.is_empty():
        return None

    d = ensure_datetime(d, "start_time")
    d = d.drop_nulls(subset=["start_time"])

    if d.is_empty():
        return None

    d = d.with_columns(
        (pl.col("start_time").dt.weekday() - 1).cast(pl.Int32).alias("day_of_week"),
        pl.col("start_time").dt.hour().cast(pl.Int32).alias("hour"),
        (pl.col("outcome") == OUTCOME_CODES.WIN).cast(pl.Int32).alias("is_win"),
    )

    agg = d.group_by("day_of_week", "hour").agg(
        pl.col("is_win").sum().alias("wins"),
        pl.col("match_id").count().alias("total"),
    )
    agg = agg.with_columns(
        (pl.col("wins").cast(pl.Float64) / pl.col("total")).fill_null(0.0).alias("win_rate"),
    )
    agg = agg.with_columns(
        pl.when(pl.col("total") < min_matches)
        .then(None)
        .otherwise(pl.col("win_rate"))
        .alias("win_rate"),
    )

    # Si aucun créneau n'atteint le seuil min_matches → données insuffisantes
    if agg["win_rate"].drop_nulls().is_empty():
        return None

    # Grille complète 7 jours × 24 heures
    all_hours = list(range(24))
    all_days = list(range(7))
    full_grid = pl.DataFrame(
        {
            "day_of_week": [dow for dow in all_days for _ in all_hours],
            "hour": [h for _ in all_days for h in all_hours],
        }
    ).cast({"day_of_week": pl.Int32, "hour": pl.Int32})

    merged = full_grid.join(agg, on=["day_of_week", "hour"], how="left").sort("day_of_week", "hour")
    merged = merged.with_columns(
        pl.col("total").fill_null(0).cast(pl.Int64),
    )

    # Construire les matrices numpy 7×24
    win_rate_vals = merged["win_rate"].to_numpy().reshape(7, 24)
    count_vals = merged["total"].to_numpy().reshape(7, 24).astype(int)

    # Vérification finale : si tout est NaN après reshape, pas de graphe
    if np.all(np.isnan(win_rate_vals.astype(float))):
        return None

    if lang == "en":
        day_labels = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
    else:
        day_labels = ["Lun", "Mar", "Mer", "Jeu", "Ven", "Sam", "Dim"]
    hour_labels = [f"{h:02d}h" for h in all_hours]
    text_matrix = count_vals.astype(str)
    text_matrix[count_vals == 0] = ""
    _matches_lbl = viz_t("trace_matches", lang)
    _win_rate_lbl = viz_t("hover_win_rate", lang)

    fig = go.Figure(
        data=go.Heatmap(
            z=win_rate_vals,
            x=hour_labels,
            y=day_labels,
            colorscale=[[0.0, colors["red"]], [0.5, colors["amber"]], [1.0, colors["green"]]],
            zmin=0,
            zmax=1,
            text=text_matrix,
            texttemplate="%{text}",
            textfont={"size": 10},
            hovertemplate=f"%{{y}} %{{x}}<br>{_win_rate_lbl}: %{{z:.1%}}<br>{_matches_lbl}: %{{text}}<extra></extra>",
            colorbar={"title": _win_rate_lbl, "tickformat": ".0%"},
        )
    )
    fig.update_layout(
        height=PLOT_CONFIG.default_height,
        margin={"l": 60, "r": 20, "t": 60 if title else 30, "b": 40},
    )
    fig.update_xaxes(title_text=viz_t("axis_hour_label", lang), side="bottom")
    fig.update_yaxes(title_text=viz_t("axis_day_label", lang), autorange="reversed")
    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)


# ---------------------------------------------------------------------------
# plot_matches_at_top_by_week
# ---------------------------------------------------------------------------


def plot_matches_at_top_by_week(
    df: DataFrameLike,
    *,
    title: str | None = None,
    rank_col: str = "rank",
    top_n_ranks: int = 1,
    lang: str = "fr",
) -> go.Figure:
    """Graphique comparant les matchs 'Top' vs Total par période."""
    d = ensure_polars(df)
    colors = HALO_COLORS.as_dict()
    d = d.drop_nulls(subset=["start_time"])

    if d.is_empty():
        fig = go.Figure()
        fig.update_layout(height=PLOT_CONFIG.default_height)
        return apply_halo_plot_style(fig, title=title)

    d, period_label = determine_top_period(d, lang=lang)

    if rank_col in d.columns:
        d = d.with_columns(
            (pl.col(rank_col).cast(pl.Float64, strict=False).fill_null(99.0) <= top_n_ranks).alias(
                "is_top"
            )
        )
    elif "outcome" in d.columns:
        d = d.with_columns((pl.col("outcome") == OUTCOME_CODES.WIN).alias("is_top"))
    else:
        d = d.with_columns(pl.lit(False).alias("is_top"))

    agg = (
        d.group_by("period")
        .agg(
            pl.col("match_id").count().alias("total"),
            pl.col("is_top").sum().alias("top_count"),
        )
        .sort("period")
    )
    agg = agg.with_columns(
        (pl.col("total") - pl.col("top_count")).alias("other_count"),
        (pl.col("top_count").cast(pl.Float64) / pl.col("total") * 100.0).round(1).alias("top_rate"),
    )

    periods = agg["period"].to_list()
    top_counts = agg["top_count"].to_list()
    other_counts = agg["other_count"].to_list()
    top_rates = agg["top_rate"].to_list()

    text_other = (
        agg.select(
            pl.when(pl.col("other_count") > 0)
            .then(pl.col("other_count").cast(pl.String))
            .otherwise(pl.lit(""))
        )
        .to_series()
        .to_list()
    )

    fig = go.Figure()
    _others_lbl = viz_t("trace_others", lang)
    _top_rate_lbl = viz_t("trace_top_rate", lang)

    fig.add_trace(
        go.Bar(
            x=periods,
            y=top_counts,
            name=f"Top {top_n_ranks}",
            marker_color=colors["green"],
            opacity=0.85,
            text=top_counts,
            textposition="inside",
            hovertemplate=f"%{{x}}<br>Top: %{{y}}<br>{viz_t('trace_top_rate', lang)}: %{{customdata:.1f}}%<extra></extra>",
            customdata=top_rates,
        )
    )
    fig.add_trace(
        go.Bar(
            x=periods,
            y=other_counts,
            name=_others_lbl,
            marker_color=colors["slate"],
            opacity=0.55,
            text=text_other,
            textposition="inside",
            hovertemplate=f"%{{x}}<br>{_others_lbl}: %{{y}}<extra></extra>",
        )
    )
    fig.add_trace(
        go.Scatter(
            x=periods,
            y=top_rates,
            mode="lines+markers",
            name=_top_rate_lbl,
            yaxis="y2",
            line={"color": colors["amber"], "width": 2},
            marker={"size": 6},
            hovertemplate=f"%{{x}}<br>{_top_rate_lbl}: %{{y:.1f}}%<extra></extra>",
        )
    )

    fig.update_layout(
        barmode="stack",
        bargap=0.15,
        height=PLOT_CONFIG.default_height,
        margin={"l": 40, "r": 60, "t": 60 if title else 30, "b": 80},
        legend=get_legend_horizontal_bottom(),
        yaxis2={
            "title": viz_t("axis_rate_pct", lang),
            "overlaying": "y",
            "side": "right",
            "range": [0, 100],
            "showgrid": False,
        },
    )
    fig.update_xaxes(tickangle=45, title_text=period_label)
    fig.update_yaxes(title_text=viz_t("trace_matches", lang))
    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)
