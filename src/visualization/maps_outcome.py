"""Graphiques par carte centrés sur l'outcome (lollipop, timeline, bullet, perf vs historique).

Extrait de maps.py pour respecter la limite de 500 lignes.
Ces 4 fonctions partagent la dimension outcome (W/D/L) et complètent
plot_map_ratio_with_winloss pour les vues session courte et longue période.
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.analysis.performance_config import SCORE_THRESHOLDS
from src.config import HALO_COLORS, PLOT_CONFIG
from src.data.domain.refdata import Outcome
from src.visualization._compat import DataFrameLike, ensure_polars, to_pandas_for_plotly
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom

# ─── Constantes ─────────────────────────────────────────────────────────────

_MAX_MAPS_TIMELINE = 15
_BULLET_DELTA_THRESHOLD = 0.05  # ±5 % considéré comme équivalent
_OKABE_ROSE = "#CC79A7"  # Okabe-Ito reddish-purple (rose/lilas daltonien-safe)

# (outcome_group_int, colors_key, label_fr, label_en)
_TIMELINE_OUTCOMES: list[tuple[int, str, str, str]] = [
    (int(Outcome.WIN), "green", "Victoire", "Win"),
    (int(Outcome.LOSS), "red", "Défaite", "Loss"),
    (0, "violet", "Autre", "Other"),
]

# ─── Helpers privés ─────────────────────────────────────────────────────────


def _perf_color(v: float) -> str:
    """Retourne la couleur de performance selon SCORE_THRESHOLDS."""
    c = HALO_COLORS.as_dict()
    if v >= SCORE_THRESHOLDS["excellent"]:
        return c["green"]
    if v >= SCORE_THRESHOLDS["good"]:
        return c["cyan"]
    if v >= SCORE_THRESHOLDS["average"]:
        return c["amber"]
    if v >= SCORE_THRESHOLDS["below_average"]:
        return c.get("orange", "#FF8C00")
    return c["red"]


def _sort_by_map_order(
    df: pl.DataFrame, map_order: list[str], descending: bool = True
) -> pl.DataFrame:
    """Trie un DataFrame par ordre chronologique de cartes.

    Args:
        df: DataFrame à trier (doit avoir une colonne 'map_name').
        map_order: Ordre des cartes (oldest=index 0, newest=last).
        descending: True → oldest apparaît en haut dans Plotly (recommandé).

    Returns:
        DataFrame trié.
    """
    map_pos = {m: i for i, m in enumerate(map_order)}
    return (
        df.with_columns(
            pl.col("map_name")
            .replace_strict(map_pos, default=len(map_order), return_dtype=pl.Int64)
            .alias("_order")
        )
        .sort("_order", descending=descending)
        .drop("_order")
    )


def _empty_map_figure() -> go.Figure:
    """Figure vide standardisée pour les graphiques de carte."""
    fig = go.Figure()
    fig.update_layout(
        height=PLOT_CONFIG.default_height, margin={"l": 40, "r": 20, "t": 30, "b": 40}
    )
    return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height)


def _prepare_timeline_df(df_pl: pl.DataFrame, session_ids: set[str]) -> pl.DataFrame:
    """Trie et enrichit le DataFrame pour la timeline (index temporel, groupe outcome)."""
    return (
        df_pl.filter(
            pl.col("map_name").is_not_null() & (pl.col("map_name").str.strip_chars() != "")
        )
        .sort("start_time")
        .with_row_index("_global_idx")
        .with_columns(
            (pl.col("_global_idx") - pl.col("_global_idx").min().over("map_name")).alias("_idx"),
            pl.when(pl.col("outcome") == int(Outcome.WIN))
            .then(int(Outcome.WIN))
            .when(pl.col("outcome") == int(Outcome.LOSS))
            .then(int(Outcome.LOSS))
            .otherwise(0)
            .alias("_outcome_group"),
            pl.col("match_id").cast(pl.Utf8).is_in(list(session_ids)).alias("_is_session"),
        )
    )


def _add_timeline_traces(fig: go.Figure, sub: pl.DataFrame, color: str, label: str) -> None:
    """Ajoute les traces historique (petits) + session (grands, anneau or) pour un outcome."""
    sub_hist = sub.filter(~pl.col("_is_session"))
    sub_sess = sub.filter(pl.col("_is_session"))
    if not sub_hist.is_empty():
        fig.add_trace(
            go.Scatter(
                x=sub_hist["_idx"].to_list(),
                y=sub_hist["map_name"].to_list(),
                mode="markers",
                name=label,
                legendgroup=label,
                marker={"color": color, "size": 8, "opacity": 0.4},
                showlegend=True,
            )
        )
    if not sub_sess.is_empty():
        fig.add_trace(
            go.Scatter(
                x=sub_sess["_idx"].to_list(),
                y=sub_sess["map_name"].to_list(),
                mode="markers",
                name=label,
                legendgroup=label,
                marker={
                    "color": color,
                    "size": 14,
                    "opacity": 1.0,
                    "line": {"color": "white", "width": 2},
                },
                showlegend=False,
            )
        )


# ─── Option A — Lollipop ────────────────────────────────────────────────────


def plot_map_lollipop(
    df_breakdown: DataFrameLike,
    lang: str = "fr",
    map_order: list[str] | None = None,
    color_by_perf: bool = False,
) -> go.Figure:
    """Lollipop chart : win rate par carte (outcome-focused).

    Chaque carte = tige grise + cercle coloré.
    Couleur par défaut : vert >= 50 %, rouge sinon. Si color_by_perf=True: gamme performance.
    Taille du cercle proportionnelle au nombre de matchs.

    Args:
        df_breakdown: DataFrame issu de compute_map_breakdown.
        lang: Langue cible.
        map_order: Ordre chronologique des cartes (oldest=index 0). Si None, tri par win_rate.
        color_by_perf: Si True, couleur selon le score de performance (gamme globale).

    Returns:
        Figure Plotly.
    """
    df_pl = ensure_polars(df_breakdown).drop_nulls(subset=["win_rate", "matches"])
    if df_pl.is_empty():
        return _empty_map_figure()

    if map_order is not None:
        df_pl = _sort_by_map_order(df_pl, map_order)
    else:
        df_pl = df_pl.sort("win_rate")
    d = to_pandas_for_plotly(df_pl)
    colors = HALO_COLORS.as_dict()
    win_rates = list(d["win_rate"])
    map_names = list(d["map_name"])
    match_counts = list(d["matches"])

    stems_x: list[float | None] = []
    stems_y: list[str | None] = []
    for wr, mn in zip(win_rates, map_names, strict=False):
        stems_x += [0.0, wr, None]
        stems_y += [mn, mn, None]

    if color_by_perf and "performance_avg" in d.columns:
        dot_colors = [_perf_color(v if v == v else 45.0) for v in d["performance_avg"]]
    else:
        dot_colors = [colors["green"] if wr >= 0.5 else colors["red"] for wr in win_rates]
    sizes = [min(24, 12 + int(n**0.5 * 3)) for n in match_counts]
    texts = [
        f"{wr:.0%}" if n > 1 else ("V" if wr >= 0.5 else "D")
        for wr, n in zip(win_rates, match_counts, strict=False)
    ]
    fig = go.Figure()
    fig.add_trace(
        go.Scatter(
            x=stems_x,
            y=stems_y,
            mode="lines",
            line={"color": "rgba(160,160,160,0.4)", "width": 2},
            showlegend=False,
            hoverinfo="skip",
        )
    )
    fig.add_trace(
        go.Scatter(
            x=win_rates,
            y=map_names,
            mode="markers+text",
            marker={"color": dot_colors, "size": sizes, "line": {"color": "white", "width": 1.5}},
            text=texts,
            textposition="middle center",
            textfont={"size": 9, "color": "white", "weight": "bold"},
            customdata=list(zip(match_counts, win_rates, strict=False)),
            hovertemplate="%{y}<br>Win=%{customdata[1]:.0%}  N=%{customdata[0]}<extra></extra>",
            showlegend=False,
        )
    )
    fig.add_vline(x=0.5, line={"dash": "dot", "color": "rgba(180,180,180,0.5)", "width": 1})
    fig.update_layout(height=PLOT_CONFIG.tall_height, margin={"l": 40, "r": 30, "t": 30, "b": 40})
    fig.update_xaxes(tickformat=".0%", range=[-0.05, 1.15])
    return apply_halo_plot_style(fig, height=PLOT_CONFIG.tall_height)


# ─── Option B — Timeline dots ────────────────────────────────────────────────


def plot_map_outcome_timeline(
    df_matches: DataFrameLike,
    session_match_ids: list[str] | None = None,
    lang: str = "fr",
) -> go.Figure | None:
    """Timeline des outcomes par carte : un cercle coloré par match, ordonné chronologiquement.

    Les matchs de la sélection courante sont mis en évidence (plus grands, contour blanc).

    Args:
        df_matches: DataFrame brut (map_name, start_time, outcome, match_id).
        session_match_ids: IDs des matchs a mettre en evidence.
        lang: Langue.

    Returns:
        Figure Plotly ou None si donnees insuffisantes.
    """
    required = {"map_name", "start_time", "outcome", "match_id"}
    df_pl = ensure_polars(df_matches)
    if df_pl.is_empty() or not required.issubset(set(df_pl.columns)):
        return None

    session_ids: set[str] = set(session_match_ids or [])
    top_maps = (
        df_pl.filter(pl.col("map_name").is_not_null())
        .group_by("map_name")
        .agg(pl.len().alias("_cnt"))
        .sort("_cnt", descending=True)
        .head(_MAX_MAPS_TIMELINE)["map_name"]
        .to_list()
    )
    df_pl = df_pl.filter(pl.col("map_name").is_in(top_maps))
    if df_pl.is_empty():
        return None

    df_proc = _prepare_timeline_df(df_pl, session_ids)
    colors = HALO_COLORS.as_dict()
    fig = go.Figure()
    for outcome_val, color_key, label_fr, label_en in _TIMELINE_OUTCOMES:
        label = label_fr if lang == "fr" else label_en
        sub = df_proc.filter(pl.col("_outcome_group") == outcome_val)
        if not sub.is_empty():
            _add_timeline_traces(fig, sub, colors[color_key], label)

    if not fig.data:
        return None

    # Overlay doré : met en évidence TOUS les matchs de la session (quel que soit l'outcome)
    sess_all = df_proc.filter(pl.col("_is_session"))
    if not sess_all.is_empty():
        sess_lbl = "Session actuelle" if lang == "fr" else "Current session"
        fig.add_trace(
            go.Scatter(
                x=sess_all["_idx"].to_list(),
                y=sess_all["map_name"].to_list(),
                mode="markers",
                name=sess_lbl,
                marker={
                    "symbol": "circle-open",
                    "color": "rgba(0,0,0,0)",
                    "size": 18,
                    "line": {"color": "#FFD700", "width": 2.5},
                },
                showlegend=True,
            )
        )

    height = max(PLOT_CONFIG.default_height, len(top_maps) * 34 + 80)
    fig.update_layout(
        height=height,
        margin={"l": 40, "r": 20, "t": 30, "b": 80},
        xaxis={"visible": False},
        legend=get_legend_horizontal_bottom(),
    )
    return apply_halo_plot_style(fig, height=height)


# ─── Option C — Bullet win rate ──────────────────────────────────────────────


def _prepare_bullet_joined_data(
    bd_current: DataFrameLike,
    bd_history: DataFrameLike,
    map_order: list[str] | None,
) -> pl.DataFrame | None:
    """Jointure + tri pour plot_map_winrate_bullet."""
    bd_curr = ensure_polars(bd_current).drop_nulls(subset=["win_rate"])
    bd_hist = (
        ensure_polars(bd_history)
        .drop_nulls(subset=["win_rate"])
        .select(["map_name", "win_rate", "matches"])
        .rename({"win_rate": "_hist_wr", "matches": "_hist_n"})
    )
    joined = bd_curr.join(bd_hist, on="map_name", how="inner")
    if joined.is_empty():
        return None
    return (
        _sort_by_map_order(joined, map_order)
        if map_order is not None
        else joined.sort("_hist_wr", descending=True)
    )


def _add_bullet_bar_traces(  # noqa: PLR0913
    fig: go.Figure,
    d: object,
    over_mask: list[bool],
    under_colors: list[str],
    over_colors: list[str],
    rose_behind: str,
    rose_front: str,
    hist_lbl: str,
    sess_lbl: str,
) -> None:
    """Ajoute les 4 traces bar overlay (under/over × hist/sess)."""
    under = [not ov for ov in over_mask]
    under_hist_x = [h if u else None for h, u in zip(d["_hist_wr"], under, strict=False)]  # type: ignore[index]
    under_sess_x = [s if u else None for s, u in zip(d["win_rate"], under, strict=False)]  # type: ignore[index]
    over_sess_x = [s if ov else None for s, ov in zip(d["win_rate"], over_mask, strict=False)]  # type: ignore[index]
    over_hist_x = [h if ov else None for h, ov in zip(d["_hist_wr"], over_mask, strict=False)]  # type: ignore[index]
    _bar = {"orientation": "h", "width": 0.55}
    hist_cd = list(zip(d["_hist_wr"], d["_hist_n"], strict=False))  # type: ignore[index]
    sess_cd = list(zip(d["win_rate"], d["matches"], strict=False))  # type: ignore[index]
    h_tmpl = f"%{{y}}<br>{hist_lbl}=%{{customdata[0]:.0%}} (N=%{{customdata[1]}})<extra></extra>"
    s_tmpl = f"%{{y}}<br>{sess_lbl}=%{{customdata[0]:.0%}} (N=%{{customdata[1]}})<extra></extra>"
    _mn = d["map_name"]  # type: ignore[index]
    for trace_kwargs in (
        {
            "x": under_hist_x,
            "name": hist_lbl,
            "legendgroup": "hist",
            "marker": {"color": rose_behind, "line": {"width": 0}},
            "showlegend": True,
            "customdata": hist_cd,
            "hovertemplate": h_tmpl,
        },
        {
            "x": under_sess_x,
            "name": sess_lbl,
            "legendgroup": "sess",
            "marker": {"color": under_colors, "line": {"width": 0}},
            "showlegend": True,
            "customdata": sess_cd,
            "hovertemplate": s_tmpl,
        },
        {
            "x": over_sess_x,
            "name": sess_lbl,
            "legendgroup": "sess",
            "marker": {"color": over_colors, "line": {"width": 0}},
            "opacity": 0.60,
            "showlegend": False,
            "customdata": sess_cd,
            "hovertemplate": s_tmpl,
        },
        {
            "x": over_hist_x,
            "name": hist_lbl,
            "legendgroup": "hist",
            "marker": {"color": rose_front, "line": {"width": 0}},
            "showlegend": False,
            "customdata": hist_cd,
            "hovertemplate": h_tmpl,
        },
    ):
        fig.add_trace(go.Bar(y=_mn, **trace_kwargs, **_bar))


def _add_bullet_color_legend_traces(fig: go.Figure, colors: dict, lang: str) -> None:
    """Ajoute les 3 entrées de légende couleur (vert/ambre/rouge)."""
    _pct = int(_BULLET_DELTA_THRESHOLD * 100)
    _h = "hist." if lang == "fr" else "history"
    for _c, _lbl in [
        (colors["green"], f"Session > {_h} (+{_pct} %)"),
        (colors["amber"], f"Session ≈ {_h} (±{_pct} %)"),
        (colors["red"], f"Session < {_h} (-{_pct} %)"),
    ]:
        fig.add_trace(
            go.Bar(
                x=[None],
                y=[None],
                orientation="h",
                name=_lbl,
                marker={"color": _c},
                showlegend=True,
                legendgroup="color_key",
            )
        )


def _add_bullet_overlay_traces(  # noqa: PLR0913
    fig: go.Figure,
    d: object,
    curr_colors: list[str],
    over_mask: list[bool],
    hist_lbl: str,
    sess_lbl: str,
    colors: dict,
    lang: str = "fr",
) -> None:
    """4 traces overlay + entrées légende couleur pour le bullet chart win rate."""
    rose_behind = "rgba(204,121,167,0.35)"
    rose_front = "rgba(204,121,167,0.88)"
    transp = "rgba(0,0,0,0)"
    under = [not ov for ov in over_mask]
    under_colors = [c if u else transp for c, u in zip(curr_colors, under, strict=False)]
    over_colors = [c if ov else transp for c, ov in zip(curr_colors, over_mask, strict=False)]
    _add_bullet_bar_traces(
        fig, d, over_mask, under_colors, over_colors, rose_behind, rose_front, hist_lbl, sess_lbl
    )
    _add_bullet_color_legend_traces(fig, colors, lang)


def plot_map_winrate_bullet(
    bd_current: DataFrameLike,
    bd_history: DataFrameLike,
    lang: str = "fr",
    map_order: list[str] | None = None,
) -> go.Figure | None:
    """Overlay bars : win rate session vs historique par carte.

    Barre rose Okabe (#CC79A7) devant quand la session dépasse l'historique.
    """
    joined = _prepare_bullet_joined_data(bd_current, bd_history, map_order)
    if joined is None:
        return None

    d = to_pandas_for_plotly(joined)
    colors = HALO_COLORS.as_dict()
    hist_lbl = "Historique" if lang == "fr" else "History"
    sess_lbl = "Session actuelle" if lang == "fr" else "Current session"

    def _sess_color(curr: float, hist: float) -> str:
        if curr >= hist + _BULLET_DELTA_THRESHOLD:
            return colors["green"]
        if curr <= hist - _BULLET_DELTA_THRESHOLD:
            return colors["red"]
        return colors["amber"]

    curr_colors = [_sess_color(c, h) for c, h in zip(d["win_rate"], d["_hist_wr"], strict=False)]
    over_mask = [c > h for c, h in zip(d["win_rate"], d["_hist_wr"], strict=False)]

    fig = go.Figure()
    _add_bullet_overlay_traces(fig, d, curr_colors, over_mask, hist_lbl, sess_lbl, colors, lang)

    fig.add_vline(x=0.5, line={"dash": "dot", "color": "rgba(180,180,180,0.6)", "width": 1.5})
    fig.update_layout(
        barmode="overlay",
        height=PLOT_CONFIG.tall_height,
        margin={"l": 40, "r": 20, "t": 30, "b": 70},
        legend=get_legend_horizontal_bottom(),
    )
    fig.update_xaxes(tickformat=".0%", range=[-0.05, 1.1])
    return apply_halo_plot_style(fig, height=PLOT_CONFIG.tall_height)


# ─── Feature 2 — Perf vs historique ─────────────────────────────────────────


def plot_map_perf_vs_history(
    bd_current: DataFrameLike,
    bd_history: DataFrameLike,
    lang: str = "fr",
    map_order: list[str] | None = None,
) -> go.Figure | None:
    """Barres horizontales groupées : performance session vs historique par carte.

    Les barres session sont colorées selon la gamme de performance (vert/cyan/ambre/orange/rouge).
    Les barres historique restent grises (référence neutre).

    Args:
        bd_current: Breakdown session (map_name, performance_avg, matches).
        bd_history: Breakdown historique (map_name, performance_avg, matches).
        lang: Langue.
        map_order: Ordre chronologique des cartes (oldest=index 0). Si None, tri par perf.

    Returns:
        Figure Plotly ou None si aucune carte commune.
    """
    bd_curr = ensure_polars(bd_current).drop_nulls(subset=["performance_avg"])
    bd_hist = (
        ensure_polars(bd_history)
        .drop_nulls(subset=["performance_avg"])
        .select(["map_name", "performance_avg", "matches"])
        .rename({"performance_avg": "_hist_perf", "matches": "_hist_n"})
    )
    joined = bd_curr.join(bd_hist, on="map_name", how="inner")
    if joined.is_empty():
        return None

    if map_order is not None:
        joined = _sort_by_map_order(joined, map_order)
    else:
        joined = joined.sort("performance_avg", descending=False).head(20)
    d = to_pandas_for_plotly(joined)
    hist_lbl = "Historique" if lang == "fr" else "History"
    sess_lbl = "Session actuelle" if lang == "fr" else "Current session"

    # Couleurs session basées sur la gamme de performance
    bar_colors = [_perf_color(v) for v in d["performance_avg"]]

    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            x=d["_hist_perf"],
            y=d["map_name"],
            orientation="h",
            name=hist_lbl,
            marker_color="rgba(120,120,120,0.45)",
            customdata=d["_hist_n"],
            hovertemplate=f"%{{y}}<br>{hist_lbl}=%{{x:.1f}} (N=%{{customdata}})<extra></extra>",
        )
    )
    fig.add_trace(
        go.Bar(
            x=d["performance_avg"],
            y=d["map_name"],
            orientation="h",
            name=sess_lbl,
            marker_color=bar_colors,
            opacity=0.85,
            customdata=d["matches"],
            hovertemplate=f"%{{y}}<br>{sess_lbl}=%{{x:.1f}} (N=%{{customdata}})<extra></extra>",
        )
    )
    fig.add_vline(x=0.0, line={"dash": "dot", "color": "rgba(180,180,180,0.6)", "width": 1})
    fig.update_layout(
        barmode="group",
        height=PLOT_CONFIG.tall_height,
        margin={"l": 40, "r": 20, "t": 30, "b": 70},
        legend=get_legend_horizontal_bottom(),
    )
    return apply_halo_plot_style(fig, height=PLOT_CONFIG.tall_height)
