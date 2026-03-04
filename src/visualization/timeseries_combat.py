"""Graphiques de séries temporelles — Combat (Sprint 16).

Fonctions déplacées depuis ``timeseries.py`` pour alléger le module principal.
"""

import math

import plotly.graph_objects as go
import polars as pl
from plotly.subplots import make_subplots

from src.analysis.performance_config import SCORE_THRESHOLDS
from src.config import HALO_COLORS, PLOT_CONFIG
from src.ui.components.chart_annotations import add_extreme_annotations  # noqa: F401
from src.ui.date_formats import FMT_DATETIME_FR, FMT_TICK_DATETIME
from src.ui.i18n.viz import viz_t
from src.visualization._compat import DataFrameLike, ensure_polars, smart_scatter  # noqa: F401
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom
from src.visualization.timeseries import _normalize_df, _rolling_mean


def plot_average_life(df: DataFrameLike, title: str | None = None, lang: str = "fr") -> go.Figure:
    """Graphique de la durée de vie moyenne.

    Args:
        df: DataFrame (Pandas ou Polars) avec colonne average_life_seconds.
        title: Titre du graphique.

    Returns:
        Figure Plotly.
    """
    # Normaliser en Polars
    d = _normalize_df(df)
    if title is None:
        title = viz_t("title_avg_life", lang)

    colors = HALO_COLORS.as_dict()
    d = d.filter(pl.col("average_life_seconds").is_not_null()).sort("start_time")
    x_idx = list(range(d.height))
    labels = d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    step = max(1, len(labels) // 10) if len(labels) > 1 else 1

    y = d["average_life_seconds"].cast(pl.Float64, strict=False)
    custom = list(
        zip(
            d["deaths"].fill_null(0).cast(pl.Int64).to_list(),
            d["time_played_seconds"].cast(pl.Float64, strict=False).to_list(),
            d["match_id"].cast(pl.Utf8).to_list(),
            strict=False,
        )
    )

    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=y.to_list(),
            name=viz_t("trace_lifespan", lang),
            marker_color=colors["green"],
            opacity=PLOT_CONFIG.bar_opacity,
            customdata=custom,
            hovertemplate=viz_t("hover_lifespan", lang),
        )
    )

    fig.add_trace(
        smart_scatter(
            x=x_idx,
            y=_rolling_mean(y, window=10).to_list(),
            mode="lines",
            name=viz_t("trace_avg_smoothed", lang),
            line={"width": PLOT_CONFIG.line_width, "color": colors["cyan"]},
            hovertemplate=viz_t("hover_avg_smoothed_s", lang),
        )
    )

    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 50, "b": 90},
        hovermode="x unified",
        legend=get_legend_horizontal_bottom(),
    )
    fig.update_yaxes(title_text=viz_t("axis_seconds", lang), rangemode="tozero")
    fig.update_xaxes(
        title_text=viz_t("axis_chronological", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
        type="category",
    )

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.short_height)


def plot_spree_headshots_accuracy(
    df: DataFrameLike,
    perfect_counts: dict[str, int] | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique combiné: Spree, Tirs à la tête, Précision et Perfect kills.

    Args:
        df: DataFrame (Pandas ou Polars) avec colonnes max_killing_spree, headshot_kills, accuracy.
        perfect_counts: Dict optionnel {match_id: count} pour les médailles Perfect.

    Returns:
        Figure Plotly avec axe Y secondaire pour la précision.
    """
    # Normaliser en Polars
    d = _normalize_df(df)

    colors = HALO_COLORS.as_dict()
    d = d.sort("start_time")
    x_idx = list(range(d.height))

    if "max_killing_spree" in d.columns:
        spree = d["max_killing_spree"].cast(pl.Float64, strict=False).to_list()
    else:
        spree = [float("nan")] * d.height

    fig = make_subplots(rows=1, cols=1, specs=[[{"secondary_y": True}]])

    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=spree,
            name=viz_t("trace_killing_spree", lang),
            marker_color=colors["amber"],
            opacity=PLOT_CONFIG.bar_opacity,
            alignmentgroup="spree_hs",
            offsetgroup="spree",
            width=0.42,
            hovertemplate=viz_t("hover_killing_spree", lang),
        ),
        secondary_y=False,
    )

    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=d["headshot_kills"].to_list(),
            name=viz_t("trace_headshots", lang),
            marker_color=colors["red"],
            opacity=0.70,
            alignmentgroup="spree_hs",
            offsetgroup="headshots",
            width=0.42,
            hovertemplate=viz_t("hover_headshots", lang),
        ),
        secondary_y=False,
    )

    # Frags parfaits (médaille Perfect = tuer sans prendre de dégâts) — toujours afficher la série
    if "match_id" in d.columns and perfect_counts is not None:
        match_ids = d["match_id"].cast(pl.Utf8).to_list()
        perfect_series = [perfect_counts.get(mid, 0) for mid in match_ids]
    else:
        perfect_series = [0] * d.height
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=perfect_series,
            name=viz_t("trace_perfect_kills", lang),
            marker_color=colors["green"],
            opacity=0.65,
            alignmentgroup="spree_hs",
            offsetgroup="perfect",
            width=0.28,
            hovertemplate=viz_t("hover_perfect_sprees", lang),
        ),
        secondary_y=False,
    )

    labels = d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    step = max(1, len(labels) // 10) if labels else 1
    fig.update_xaxes(
        title_text=viz_t("axis_chronological", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
    )

    fig.update_layout(
        height=420,
        margin={"l": 40, "r": 50, "t": 30, "b": 90},
        legend=get_legend_horizontal_bottom(),
        hovermode="x unified",
        barmode="group",
        bargap=0.15,
        bargroupgap=0.06,
    )

    fig.update_yaxes(
        title_text=viz_t("axis_spree_headshots", lang), rangemode="tozero", secondary_y=False
    )

    return apply_halo_plot_style(fig, height=420)


def plot_performance_timeseries(
    df: DataFrameLike,
    df_history: DataFrameLike | None = None,
    title: str | None = None,
    show_smooth: bool = True,
    lang: str = "fr",
) -> go.Figure:
    """Graphique du score de performance dans le temps.

    Args:
        df: DataFrame (Pandas ou Polars) avec colonnes performance ou kills/deaths/assists/accuracy/outcome.
        df_history: DataFrame complet (Pandas ou Polars) pour le calcul du score relatif.
        title: Titre du graphique.
        show_smooth: Afficher la courbe de moyenne lissée.

    Returns:
        Figure Plotly.
    """
    from src.analysis.performance_score import compute_performance_series

    # Normaliser en Polars
    d = _normalize_df(df)
    if title is None:
        title = viz_t("title_performance", lang)
    history_pl: pl.DataFrame | None = None
    if df_history is not None:
        history_pl = _normalize_df(df_history)

    colors = HALO_COLORS.as_dict()
    d = d.sort("start_time")
    x_idx = list(range(d.height))
    labels = d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    step = max(1, len(labels) // 10) if len(labels) > 1 else 1

    # Calculer le score de performance RELATIF
    history = history_pl if history_pl is not None else d
    if "performance" not in d.columns or d["performance"].is_null().all():
        perf_series = compute_performance_series(d, history)
        if isinstance(perf_series, pl.Series):
            d = d.with_columns(perf_series.alias("performance"))
        else:
            d = d.with_columns(pl.Series("performance", perf_series.to_list()))

    performance = d["performance"].cast(pl.Float64, strict=False)

    # Déterminer la couleur en fonction du score
    def _get_perf_color(val: float) -> str:
        if val >= SCORE_THRESHOLDS["excellent"]:
            return colors.get("green", "#50C878")
        elif val >= SCORE_THRESHOLDS["good"]:
            return colors.get("cyan", "#00B7EB")
        elif val >= SCORE_THRESHOLDS["average"]:
            return colors.get("amber", "#FFBF00")
        elif val >= SCORE_THRESHOLDS["below_average"]:
            return colors.get("orange", "#FF8C00")
        else:
            return colors.get("red", "#FF4444")

    bar_colors = [
        _get_perf_color(v)
        if not (v is None or (isinstance(v, float) and math.isnan(v)))
        else colors.get("gray", "#888888")
        for v in performance.to_list()
    ]

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
                line={"width": PLOT_CONFIG.line_width, "color": colors.get("violet", "#8B5CF6")},
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
    fig.update_xaxes(
        title_text=viz_t("axis_chronological", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
        type="category",
    )

    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)


# =============================================================================
# Sprint 7 — Nouvelles fonctions de visualisation
# =============================================================================


def plot_streak_chart(
    df: DataFrameLike,
    title: str | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique des séries de victoires et défaites dans le temps.

    Affiche des barres positives (victoires) et négatives (défaites)
    colorées par le type de résultat.

    Args:
        df: DataFrame avec colonnes outcome, start_time.
        title: Titre du graphique.

    Returns:
        Figure Plotly.
    """
    d = _normalize_df(df)
    if title is None:
        title = viz_t("title_streaks", lang)
    colors = HALO_COLORS.as_dict()

    d = d.sort("start_time")

    # Filtrer : ne garder que V/D
    d = d.filter(pl.col("outcome").is_in([2, 3]))
    if d.height == 0:
        fig = go.Figure()
        fig.add_annotation(
            text=viz_t("empty_no_streak_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16},
        )
        return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.short_height)

    x_idx = list(range(d.height))

    # Calculer la série : cumul dans chaque streak
    outcome_col = d["outcome"]
    is_win = (outcome_col == 2).cast(pl.Int64)
    new_streak = (outcome_col != outcome_col.shift(1)).fill_null(True)
    streak_group = new_streak.cast(pl.Int64).cum_sum()

    streak_counter: list[int] = []
    prev_group = -1
    count = 0
    for g in streak_group.to_list():
        if g != prev_group:
            count = 1
            prev_group = g
        else:
            count += 1
        streak_counter.append(count)

    is_win_list = is_win.to_list()
    streak_values = [c if w == 1 else -c for c, w in zip(streak_counter, is_win_list, strict=False)]

    bar_colors = [colors["green"] if v > 0 else colors["red"] for v in streak_values]

    labels = d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    step = max(1, len(labels) // 10) if labels else 1

    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=streak_values,
            marker_color=bar_colors,
            opacity=0.85,
            hovertemplate=viz_t("hover_streak", lang),
            customdata=labels,
            showlegend=False,
        )
    )

    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 40, "b": 90},
        hovermode="x unified",
    )
    fig.update_yaxes(title_text=viz_t("axis_streak", lang), zeroline=True)
    fig.update_xaxes(
        title_text=viz_t("axis_chronological", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
        type="category",
    )

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.short_height)


def plot_damage_dealt_taken(
    df: DataFrameLike,
    title: str | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique des dégâts infligés et subis par match.

    Barres groupées pour damage_dealt et damage_taken.

    Args:
        df: DataFrame avec colonnes damage_dealt, damage_taken, start_time.
        title: Titre du graphique.

    Returns:
        Figure Plotly.
    """
    d = _normalize_df(df)
    if title is None:
        title = viz_t("title_damage", lang)
    colors = HALO_COLORS.as_dict()

    d = d.sort("start_time")
    x_idx = list(range(d.height))
    labels = d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    step = max(1, len(labels) // 10) if labels else 1

    fig = go.Figure()

    if "damage_dealt" in d.columns:
        dealt = d["damage_dealt"].cast(pl.Float64, strict=False).fill_null(0)
        fig.add_trace(
            go.Bar(
                x=x_idx,
                y=dealt.to_list(),
                name=viz_t("trace_dmg_dealt", lang),
                marker_color=colors["cyan"],
                opacity=0.80,
                hovertemplate=viz_t("hover_dmg_dealt", lang),
            )
        )
        fig.add_trace(
            smart_scatter(
                x=x_idx,
                y=_rolling_mean(dealt, window=10).to_list(),
                mode="lines",
                name=viz_t("trace_dmg_dealt_avg", lang),
                line={"width": PLOT_CONFIG.line_width, "color": colors["cyan"]},
                hovertemplate=viz_t("hover_avg0", lang),
            )
        )

    if "damage_taken" in d.columns:
        taken = d["damage_taken"].cast(pl.Float64, strict=False).fill_null(0)
        fig.add_trace(
            go.Bar(
                x=x_idx,
                y=taken.to_list(),
                name=viz_t("trace_dmg_taken", lang),
                marker_color=colors["red"],
                opacity=0.65,
                hovertemplate=viz_t("hover_dmg_taken", lang),
            )
        )
        fig.add_trace(
            smart_scatter(
                x=x_idx,
                y=_rolling_mean(taken, window=10).to_list(),
                mode="lines",
                name=viz_t("trace_dmg_taken_avg", lang),
                line={"width": PLOT_CONFIG.line_width, "color": colors["red"], "dash": "dot"},
                hovertemplate=viz_t("hover_avg0", lang),
            )
        )

    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 40, "b": 90},
        hovermode="x unified",
        legend=get_legend_horizontal_bottom(),
        barmode="group",
        bargap=0.15,
        bargroupgap=0.06,
    )
    fig.update_yaxes(title_text=viz_t("axis_damage", lang), rangemode="tozero")
    fig.update_xaxes(
        title_text=viz_t("axis_chronological", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
        type="category",
    )

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height)


def plot_shots_accuracy(
    df: DataFrameLike,
    title: str | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique des tirs (tirés/touchés) en barres groupées avec courbe de précision.

    Args:
        df: DataFrame avec colonnes shots_fired, shots_hit, accuracy, start_time.
        title: Titre du graphique.

    Returns:
        Figure Plotly avec axe Y secondaire pour la précision.
    """
    d = _normalize_df(df)
    if title is None:
        title = viz_t("title_shots", lang)
    colors = HALO_COLORS.as_dict()

    d = d.sort("start_time")
    x_idx = list(range(d.height))
    labels = d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    step = max(1, len(labels) // 10) if labels else 1

    fig = make_subplots(rows=1, cols=1, specs=[[{"secondary_y": True}]])

    if "shots_fired" in d.columns:
        fired = d["shots_fired"].cast(pl.Float64, strict=False).fill_null(0)
        fig.add_trace(
            go.Bar(
                x=x_idx,
                y=fired.to_list(),
                name=viz_t("trace_shots_fired", lang),
                marker_color=colors["amber"],
                opacity=0.70,
                alignmentgroup="shots",
                offsetgroup="fired",
                width=0.42,
                hovertemplate=viz_t("hover_shots_fired", lang),
            ),
            secondary_y=False,
        )

    if "shots_hit" in d.columns:
        hit = d["shots_hit"].cast(pl.Float64, strict=False).fill_null(0)
        fig.add_trace(
            go.Bar(
                x=x_idx,
                y=hit.to_list(),
                name=viz_t("trace_shots_hit", lang),
                marker_color=colors["green"],
                opacity=0.70,
                alignmentgroup="shots",
                offsetgroup="hit",
                width=0.42,
                hovertemplate=viz_t("hover_shots_hit", lang),
            ),
            secondary_y=False,
        )

    if "accuracy" in d.columns:
        accuracy = d["accuracy"].cast(pl.Float64, strict=False)
        fig.add_trace(
            smart_scatter(
                x=x_idx,
                y=accuracy.to_list(),
                mode="lines",
                name=viz_t("trace_accuracy", lang),
                line={"width": PLOT_CONFIG.line_width, "color": colors["violet"]},
                hovertemplate=viz_t("hover_accuracy_pct", lang),
            ),
            secondary_y=True,
        )

    fig.update_xaxes(
        title_text=viz_t("axis_chronological", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
    )

    fig.update_layout(
        title=title,
        height=420,
        margin={"l": 40, "r": 50, "t": 40, "b": 90},
        legend=get_legend_horizontal_bottom(),
        hovermode="x unified",
        barmode="group",
        bargap=0.15,
        bargroupgap=0.06,
    )

    fig.update_yaxes(title_text=viz_t("axis_shots", lang), rangemode="tozero", secondary_y=False)
    fig.update_yaxes(
        title_text=viz_t("trace_accuracy", lang),
        ticksuffix="%",
        rangemode="tozero",
        secondary_y=True,
    )

    return apply_halo_plot_style(fig, height=420)


def plot_rank_score(
    df: DataFrameLike,
    title: str | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique du rang et du score personnel par match.

    Barres pour le score personnel, ligne pour le rang.

    Args:
        df: DataFrame avec colonnes rank, personal_score, start_time.
        title: Titre du graphique.

    Returns:
        Figure Plotly avec axe Y secondaire pour le rang.
    """
    d = _normalize_df(df)
    if title is None:
        title = viz_t("title_rank_score", lang)
    colors = HALO_COLORS.as_dict()

    d = d.sort("start_time")
    x_idx = list(range(d.height))
    labels = d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    step = max(1, len(labels) // 10) if labels else 1

    fig = make_subplots(rows=1, cols=1, specs=[[{"secondary_y": True}]])

    if "personal_score" in d.columns:
        score = d["personal_score"].cast(pl.Float64, strict=False).fill_null(0)
        fig.add_trace(
            go.Bar(
                x=x_idx,
                y=score.to_list(),
                name=viz_t("trace_personal_score", lang),
                marker_color=colors["amber"],
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
                line={"width": PLOT_CONFIG.line_width, "color": colors["cyan"]},
                marker={"size": 4},
                hovertemplate="rang=%{y}<extra></extra>",
            ),
            secondary_y=True,
        )

    fig.update_xaxes(
        title_text=viz_t("axis_chronological", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
    )

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
    df: DataFrameLike,
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

    # Filtre groupe de playlist
    if playlist_group and "playlist_group" in d.columns:
        d = d.filter(pl.col("playlist_group") == playlist_group)

    if d.is_empty():
        fig = go.Figure()
        fig.update_layout(title=title)
        return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)

    d = d.sort("start_time")
    x_idx = list(range(d.height))
    labels = d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    step = max(1, len(labels) // 10) if len(labels) > 1 else 1
    colors = HALO_COLORS.as_dict()

    rating_values = d["rating_value"].cast(pl.Float64, strict=False).to_list()

    # Calcul des bornes Y (pour les zones de tier)
    y_min = max(
        0.0, (min(v for v in rating_values if v is not None) - 100) if rating_values else 800.0
    )
    y_max = max(v for v in rating_values if v is not None) + 200 if rating_values else 2400.0
    y_max = max(y_max, 2200.0)

    fig = go.Figure()

    # ── Zones de tier (arrière-plan) ──
    _tier_alphas = {
        "Bronze": "rgba(205,127,50,0.08)",
        "Silver": "rgba(192,192,192,0.08)",
        "Gold": "rgba(255,215,0,0.10)",
        "Platinum": "rgba(0,206,209,0.08)",
        "Diamond": "rgba(185,242,255,0.10)",
        "Onyx": "rgba(28,28,28,0.12)",
    }
    for tier in SKILL_TIERS:
        band_y0 = max(tier.min_rating, y_min)
        band_y1 = min(tier.max_rating, y_max)
        if band_y1 <= band_y0:
            continue
        fill_color = _tier_alphas.get(tier.name, "rgba(128,128,128,0.06)")
        fig.add_hrect(
            y0=band_y0,
            y1=band_y1,
            fillcolor=fill_color,
            line_width=0,
            annotation_text=tier.name_fr,
            annotation_position="top right",
            annotation_font={"size": 10, "color": tier.color},
        )

    # ── Bande de confiance (± rating_deviation) ──
    if show_confidence and "rating_deviation" in d.columns:
        dev_values = d["rating_deviation"].cast(pl.Float64, strict=False).to_list()
        upper = [
            (rv + dv) if (rv is not None and dv is not None) else None
            for rv, dv in zip(rating_values, dev_values, strict=False)
        ]
        lower = [
            (rv - dv) if (rv is not None and dv is not None) else None
            for rv, dv in zip(rating_values, dev_values, strict=False)
        ]
        # Trace supérieure
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
        # Trace inférieure avec fill
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

    # ── Courbe principale (LUSR/CSR) ──
    has_csr = "rating_type" in d.columns and "CSR" in (d["rating_type"].to_list())
    has_lusr = "rating_type" in d.columns and "LUSR" in (d["rating_type"].to_list())

    tier_labels = d["tier_label"].to_list() if "tier_label" in d.columns else [""] * len(x_idx)

    hover_tpl = "Rating=%{y:.0f}<br>Rang=%{customdata[0]}<br>Date=%{customdata[1]}<extra></extra>"
    customdata = list(
        zip(tier_labels, d["start_time"].dt.strftime(FMT_DATETIME_FR).to_list(), strict=False)
    )

    # Couleur selon type dominant
    main_color = colors.get("cyan", "#00B7EB") if has_lusr else colors.get("gold", "#FFD700")
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

    # ── Tendance lissée (rolling mean 20) ──
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
                    "color": colors.get("violet", "#8B5CF6"),
                    "dash": "dashdot",
                },
                hovertemplate=viz_t("hover_trend_smooth", lang),
            )
        )

    # ── Mise en forme ──
    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 60, "b": 90},
        hovermode="x unified",
        legend=get_legend_horizontal_bottom(),
    )
    fig.update_yaxes(
        title_text=viz_t("trace_lusr_axis", lang),
        range=[y_min, y_max],
    )
    fig.update_xaxes(
        title_text=viz_t("axis_chronological", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
        type="category",
    )

    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)
