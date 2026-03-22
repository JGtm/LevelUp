"""Timeline de performance d'escouade — score + MMR par session.

Génère un graphique à double axe Y :
- Barres (axe gauche) : score de performance d'escouade 0-100
- Ligne (axe droit)   : MMR moyen d'équipe
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl
from plotly.subplots import make_subplots

from src.analysis.performance_config import SCORE_THRESHOLDS
from src.config import PLOT_CONFIG
from src.ui.i18n.viz import viz_t
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization._timeseries_helpers import COLORS
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom


def _bar_color(value: float | None) -> str:
    """Couleur de barre selon le score (même palette que timeseries individuelle)."""
    if value is None:
        return COLORS.get("gray", "#6B7280")
    if value >= SCORE_THRESHOLDS["excellent"]:
        return COLORS.get("green", "#10B981")
    if value >= SCORE_THRESHOLDS["good"]:
        return COLORS.get("cyan", "#06B6D4")
    if value >= SCORE_THRESHOLDS["average"]:
        return COLORS.get("amber", "#F59E0B")
    if value >= SCORE_THRESHOLDS["below_average"]:
        return COLORS.get("orange", "#F97316")
    return COLORS.get("red", "#EF4444")


def _build_hover_texts(
    x_labels: list[str],
    perf: list[float | None],
    counts: list[int],
    lang: str,
    wl: tuple[list[int], list[int]] | None = None,
) -> list[str]:
    """Textes de survol des barres de la timeline escouade.

    Args:
        wl: Tuple (wins, losses) si les données de résultat sont disponibles.
    """
    texts = []
    for i in range(len(x_labels)):
        if perf[i] is None:
            texts.append("")
            continue
        parts = [
            f"<b>{x_labels[i]}</b>",
            f"{viz_t('trace_squad_perf', lang)}: {perf[i]:.1f}",
            f"{viz_t('trace_match_count', lang)}: {counts[i]}",
        ]
        if wl is not None:
            w, l_ = wl[0][i], wl[1][i]
            parts.append(f"{viz_t('hover_wins', lang)}: {w} — {viz_t('hover_losses', lang)}: {l_}")
        texts.append("<br>".join(parts))
    return texts


def _add_winrate_trace(
    fig: go.Figure, win_rates: list[float | None], x_idx: list[int], lang: str
) -> None:
    """Ajoute la courbe Win Rate (%) sur l'axe principal (0-100)."""
    fig.add_trace(
        go.Scatter(
            x=x_idx,
            y=win_rates,
            name=viz_t("trace_squad_win_rate", lang),
            mode="lines+markers",
            line={
                "color": COLORS.get("green", "#10B981"),
                "width": PLOT_CONFIG.line_width,
                "dash": "dot",
            },
            marker={"size": 7, "symbol": "diamond"},
            hovertemplate=f"{viz_t('trace_squad_win_rate', lang)}: %{{y:.0f}}%<extra></extra>",
        ),
        secondary_y=False,
    )


def _add_mmr_trace(fig: go.Figure, d: pl.DataFrame, x_idx: list[int], lang: str) -> None:
    """Ajoute la courbe MMR moyen sur l'axe secondaire."""
    mmr = d["team_mmr_avg"].cast(pl.Float64, strict=False).to_list()
    fig.add_trace(
        go.Scatter(
            x=x_idx,
            y=mmr,
            name=viz_t("trace_team_mmr", lang),
            mode="lines+markers",
            line={"color": COLORS.get("violet", "#8B5CF6"), "width": PLOT_CONFIG.line_width},
            marker={"size": 5},
            hovertemplate=f"{viz_t('trace_team_mmr', lang)}: %{{y:.0f}}<extra></extra>",
        ),
        secondary_y=True,
    )


def _configure_axes(
    fig: go.Figure,
    x_labels: list[str],
    x_idx: list[int],
    has_mmr: bool,
    lang: str,
) -> None:
    """Configure les axes X et Y de la timeline escouade."""
    step = max(1, len(x_idx) // 12)
    fig.update_xaxes(
        title_text=viz_t("axis_sessions", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=x_labels[::step],
        tickangle=-45,
    )
    fig.update_yaxes(
        title_text=viz_t("axis_performance", lang),
        range=[0, 100],
        rangemode="tozero",
        secondary_y=False,
    )
    if has_mmr:
        fig.update_yaxes(
            title_text=viz_t("axis_team_mmr", lang),
            showgrid=False,
            secondary_y=True,
        )


def plot_squad_performance_timeline(
    df: DataFrameLike,
    title: str | None = None,
    lang: str = "fr",
) -> go.Figure | None:
    """Graphique d'évolution de la performance d'escouade par session.

    Args:
        df: DataFrame avec colonnes bucket_label, squad_perf, team_mmr_avg, match_count.
        title: Titre du graphique.
        lang: Code langue.

    Returns:
        Figure Plotly ou None si données insuffisantes.
    """
    d = ensure_polars(df)
    if d.is_empty() or "squad_perf" not in d.columns:
        return None

    if title is None:
        title = viz_t("title_squad_timeline", lang)

    x_labels = d["bucket_label"].to_list()
    x_idx = list(range(len(x_labels)))
    perf = d["squad_perf"].cast(pl.Float64, strict=False).to_list()
    counts = d["match_count"].to_list() if "match_count" in d.columns else [1] * len(x_idx)
    has_mmr = "team_mmr_avg" in d.columns and d["team_mmr_avg"].drop_nulls().len() > 0
    has_winrate = "win_rate" in d.columns and d["win_rate"].drop_nulls().len() > 0

    wins = d["wins"].to_list() if "wins" in d.columns else None
    losses = d["losses"].to_list() if "losses" in d.columns else None
    win_rates = d["win_rate"].cast(pl.Float64, strict=False).to_list() if has_winrate else None
    wl = (wins, losses) if wins is not None and losses is not None else None

    bar_colors = [_bar_color(v) for v in perf]
    hover_texts = _build_hover_texts(x_labels, perf, counts, lang, wl)

    fig = make_subplots(specs=[[{"secondary_y": has_mmr}]])
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=perf,
            name=viz_t("trace_squad_perf", lang),
            marker_color=bar_colors,
            opacity=PLOT_CONFIG.bar_opacity,
            hovertext=hover_texts,
            hovertemplate="%{hovertext}<extra></extra>",
        ),
        secondary_y=False,
    )
    if has_winrate:
        _add_winrate_trace(fig, win_rates, x_idx, lang)
    if has_mmr:
        _add_mmr_trace(fig, d, x_idx, lang)

    _configure_axes(fig, x_labels, x_idx, has_mmr, lang)
    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 60, "t": 60, "b": 90},
        hovermode="x unified",
        legend=get_legend_horizontal_bottom(),
    )
    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)
