"""Graphiques de séries temporelles — Progression (Performance, Rang, LUSR).

Fonctions de visualisation liées à la progression du joueur,
extraites de ``timeseries_combat.py`` pour respecter la limite de 500 lignes.
"""

import math

import plotly.graph_objects as go
import polars as pl
from plotly.subplots import make_subplots

from src.analysis.performance_config import SCORE_THRESHOLDS
from src.config import PLOT_CONFIG
from src.ui.date_formats import FMT_DATETIME_FR
from src.ui.i18n.viz import viz_t
from src.visualization._compat import DataFrameLike, smart_scatter
from src.visualization._timeseries_helpers import COLORS, apply_chrono_xaxis, prepare_time_axis
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom
from src.visualization.timeseries import _normalize_df, _rolling_mean


def _ensure_performance_column(d: pl.DataFrame, history: pl.DataFrame) -> pl.DataFrame:
    """Calcule la colonne performance si absente ou entièrement nulle."""
    from src.analysis.performance_score import compute_performance_series

    if "performance" not in d.columns or d["performance"].is_null().all():
        perf_series = compute_performance_series(d, history)
        if isinstance(perf_series, pl.Series):
            return d.with_columns(perf_series.alias("performance"))
        return d.with_columns(pl.Series("performance", perf_series.to_list()))
    return d


def plot_performance_timeseries(
    df: "DataFrameLike",
    df_history: "DataFrameLike | None" = None,
    title: str | None = None,
    show_smooth: bool = True,
    lang: str = "fr",
) -> go.Figure:
    """Graphique du score de performance dans le temps.

    Args:
        df: DataFrame avec colonnes performance ou kills/deaths/assists/accuracy/outcome.
        df_history: DataFrame complet pour le calcul du score relatif.
        title: Titre du graphique.
        show_smooth: Afficher la courbe de moyenne lissée.

    Returns:
        Figure Plotly.
    """

    d = _normalize_df(df)
    if title is None:
        title = viz_t("title_performance", lang)
    history_pl: pl.DataFrame | None = None
    if df_history is not None:
        history_pl = _normalize_df(df_history)

    d = d.sort("start_time")
    x_idx, labels, step = prepare_time_axis(d)

    # Calculer le score de performance RELATIF
    history = history_pl if history_pl is not None else d
    d = _ensure_performance_column(d, history)

    performance = d["performance"].cast(pl.Float64, strict=False)
    bar_colors = [_perf_color(v) for v in performance.to_list()]

    hover = "performance=%{y:.1f}<br>date=%{customdata[0]}<extra></extra>"
    customdata = list(zip(d["start_time"].dt.strftime(FMT_DATETIME_FR).to_list(), strict=False))

    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=performance.to_list(),
            name=viz_t("trace_performance", lang),
            marker_color=bar_colors,
            opacity=PLOT_CONFIG.bar_opacity,
            customdata=customdata,
            hovertemplate=hover,
        )
    )

    if show_smooth:
        smooth = _rolling_mean(performance, window=10)
        fig.add_trace(
            smart_scatter(
                x=x_idx,
                y=smooth.to_list(),
                mode="lines",
                name=viz_t("trace_avg_smoothed", lang),
                line={"width": PLOT_CONFIG.line_width, "color": COLORS.get("violet", "#8B5CF6")},
                hovertemplate=viz_t("hover_avg_s1", lang),
            )
        )

    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 60, "b": 90},
        hovermode="x unified",
        legend=get_legend_horizontal_bottom(),
    )
    fig.update_yaxes(
        title_text=viz_t("title_performance", lang), rangemode="tozero", range=[0, 100]
    )
    apply_chrono_xaxis(fig, x_idx, labels, step, lang)

    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)


def plot_rank_score(
    df: "DataFrameLike",
    title: str | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique du rang et du score personnel par match.

    Args:
        df: DataFrame avec colonnes rank, personal_score, start_time.
        title: Titre du graphique.

    Returns:
        Figure Plotly avec axe Y secondaire pour le rang.
    """
    d = _normalize_df(df)
    if title is None:
        title = viz_t("title_rank_score", lang)

    d = d.sort("start_time")
    x_idx, labels, step = prepare_time_axis(d)

    fig = make_subplots(rows=1, cols=1, specs=[[{"secondary_y": True}]])

    if "personal_score" in d.columns:
        score = d["personal_score"].cast(pl.Float64, strict=False).fill_null(0)
        fig.add_trace(
            go.Bar(
                x=x_idx,
                y=score.to_list(),
                name=viz_t("trace_personal_score", lang),
                marker_color=COLORS["amber"],
                opacity=0.75,
                hovertemplate="score=%{y:.0f}<extra></extra>",
            ),
            secondary_y=False,
        )

    if "rank" in d.columns:
        rank = d["rank"].cast(pl.Float64, strict=False)
        fig.add_trace(
            smart_scatter(
                x=x_idx,
                y=rank.to_list(),
                mode="lines+markers",
                name=viz_t("trace_rank", lang),
                line={"width": PLOT_CONFIG.line_width, "color": COLORS["cyan"]},
                marker={"size": 4},
                hovertemplate="rang=%{y}<extra></extra>",
            ),
            secondary_y=True,
        )

    apply_chrono_xaxis(fig, x_idx, labels, step, lang, as_category=False)

    fig.update_layout(
        title=title,
        height=400,
        margin={"l": 40, "r": 50, "t": 40, "b": 90},
        legend=get_legend_horizontal_bottom(),
        hovermode="x unified",
    )

    fig.update_yaxes(
        title_text=viz_t("trace_personal_score", lang), rangemode="tozero", secondary_y=False
    )
    fig.update_yaxes(
        title_text=viz_t("trace_rank", lang),
        autorange="reversed",
        rangemode="tozero",
        secondary_y=True,
    )

    return apply_halo_plot_style(fig, height=400)


def plot_lusr_timeseries(  # noqa: PLR0913
    df: "DataFrameLike",
    title: str | None = None,
    show_confidence: bool = True,
    show_smooth: bool = True,
    playlist_group: str | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique d'évolution du LUSR (ou CSR) dans le temps.

    Affiche la courbe du rating avec zones de tier en arrière-plan,
    bande de confiance (± σ) et tendance lissée optionnelle.

    Args:
        df: DataFrame avec colonnes : ``rating_value``, ``start_time``,
            optionnel : ``rating_deviation``, ``tier_label``, ``rating_type``,
            ``playlist_group``.
        title: Titre du graphique.
        show_confidence: Afficher la bande de confiance (± rating_deviation).
        show_smooth: Afficher la courbe de tendance lissée (rolling mean 20).
        playlist_group: Filtrer sur un groupe spécifique (None = tous).

    Returns:
        Figure Plotly.
    """
    from src.analysis.skill_rating_config import SKILL_TIERS

    d = _normalize_df(df)
    if title is None:
        title = viz_t("trace_lusr_default_title", lang)

    if playlist_group and "playlist_group" in d.columns:
        d = d.filter(pl.col("playlist_group") == playlist_group)

    if d.is_empty():
        fig = go.Figure()
        fig.update_layout(title=title)
        return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)

    d = d.sort("start_time")
    x_idx, labels, step = prepare_time_axis(d)

    rating_values = d["rating_value"].cast(pl.Float64, strict=False).to_list()

    y_min = max(
        0.0, (min(v for v in rating_values if v is not None) - 100) if rating_values else 800.0
    )
    y_max = max(v for v in rating_values if v is not None) + 200 if rating_values else 2400.0
    y_max = max(y_max, 2200.0)

    fig = go.Figure()

    _add_tier_background(fig, SKILL_TIERS, y_min, y_max)

    if show_confidence and "rating_deviation" in d.columns:
        _add_confidence_band(fig, x_idx, rating_values, d, lang)

    _add_main_rating_trace(fig, x_idx, rating_values, d, lang)

    if show_smooth and len(x_idx) >= 5:
        rating_series = pl.Series("rating", rating_values)
        smooth = _rolling_mean(rating_series, window=min(20, max(3, len(x_idx) // 5)))
        fig.add_trace(
            smart_scatter(
                x=x_idx,
                y=smooth.to_list(),
                mode="lines",
                name=viz_t("trace_trend", lang),
                line={
                    "width": PLOT_CONFIG.line_width,
                    "color": COLORS.get("violet", "#8B5CF6"),
                    "dash": "dashdot",
                },
                hovertemplate=viz_t("hover_trend_smooth", lang),
            )
        )

    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 60, "b": 140},
        hovermode="x unified",
        legend={**get_legend_horizontal_bottom(), "y": -0.45},
    )
    fig.update_yaxes(
        title_text=viz_t("trace_lusr_axis", lang),
        range=[y_min, y_max],
    )
    apply_chrono_xaxis(fig, x_idx, labels, step, lang)

    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)


# ── Helpers internes ──────────────────────────────────────────────────────


def _perf_color(val: float | None) -> str:
    """Retourne la couleur en fonction du score de performance."""
    if val is None or (isinstance(val, float) and math.isnan(val)):
        return COLORS.get("gray", "#888888")
    if val >= SCORE_THRESHOLDS["excellent"]:
        return COLORS.get("green", "#50C878")
    elif val >= SCORE_THRESHOLDS["good"]:
        return COLORS.get("cyan", "#00B7EB")
    elif val >= SCORE_THRESHOLDS["average"]:
        return COLORS.get("amber", "#FFBF00")
    elif val >= SCORE_THRESHOLDS["below_average"]:
        return COLORS.get("orange", "#FF8C00")
    return COLORS.get("red", "#FF4444")


_TIER_ALPHAS = {
    "Bronze": "rgba(205,127,50,0.08)",
    "Silver": "rgba(192,192,192,0.08)",
    "Gold": "rgba(255,215,0,0.10)",
    "Platinum": "rgba(0,206,209,0.08)",
    "Diamond": "rgba(185,242,255,0.10)",
    "Onyx": "rgba(28,28,28,0.12)",
}


def _add_tier_background(fig: go.Figure, skill_tiers: list, y_min: float, y_max: float) -> None:
    """Ajoute les zones colorées de tier en arrière-plan."""
    for tier in skill_tiers:
        band_y0 = max(tier.min_rating, y_min)
        band_y1 = min(tier.max_rating, y_max)
        if band_y1 <= band_y0:
            continue
        fill_color = _TIER_ALPHAS.get(tier.name, "rgba(128,128,128,0.06)")
        fig.add_hrect(
            y0=band_y0,
            y1=band_y1,
            fillcolor=fill_color,
            line_width=0,
            annotation_text=tier.name_fr,
            annotation_position="top right",
            annotation_font={"size": 10, "color": tier.color},
        )


def _add_confidence_band(
    fig: go.Figure,
    x_idx: list[int],
    rating_values: list[float | None],
    d: pl.DataFrame,
    lang: str,
) -> None:
    """Ajoute la bande de confiance (± rating_deviation)."""
    dev_values = d["rating_deviation"].cast(pl.Float64, strict=False).to_list()
    upper = [
        (rv + dv) if (rv is not None and dv is not None) else None
        for rv, dv in zip(rating_values, dev_values, strict=False)
    ]
    lower = [
        (rv - dv) if (rv is not None and dv is not None) else None
        for rv, dv in zip(rating_values, dev_values, strict=False)
    ]
    fig.add_trace(
        smart_scatter(
            x=x_idx,
            y=upper,
            mode="lines",
            name=viz_t("trace_confidence", lang),
            line={"width": 0},
            showlegend=True,
            hoverinfo="skip",
        )
    )
    fig.add_trace(
        smart_scatter(
            x=x_idx,
            y=lower,
            mode="lines",
            fill="tonexty",
            fillcolor="rgba(0,183,235,0.12)",
            line={"width": 0},
            showlegend=False,
            hoverinfo="skip",
        )
    )


def _add_main_rating_trace(
    fig: go.Figure,
    x_idx: list[int],
    rating_values: list[float | None],
    d: pl.DataFrame,
    lang: str,
) -> None:
    """Ajoute la courbe principale LUSR/CSR."""
    has_csr = "rating_type" in d.columns and "CSR" in (d["rating_type"].to_list())
    has_lusr = "rating_type" in d.columns and "LUSR" in (d["rating_type"].to_list())

    tier_labels = d["tier_label"].to_list() if "tier_label" in d.columns else [""] * len(x_idx)

    hover_tpl = "Rating=%{y:.0f}<br>Rang=%{customdata[0]}<br>Date=%{customdata[1]}<extra></extra>"
    customdata = list(
        zip(tier_labels, d["start_time"].dt.strftime(FMT_DATETIME_FR).to_list(), strict=False)
    )

    main_color = COLORS.get("cyan", "#00B7EB") if has_lusr else COLORS.get("gold", "#FFD700")
    main_name = "LUSR" if has_lusr else "CSR"
    if has_lusr and has_csr:
        main_name = "LUSR / CSR"

    fig.add_trace(
        smart_scatter(
            x=x_idx,
            y=rating_values,
            mode="lines+markers",
            name=main_name,
            line={"width": PLOT_CONFIG.line_width, "color": main_color},
            marker={"size": 5, "color": main_color},
            customdata=customdata,
            hovertemplate=hover_tpl,
        )
    )
