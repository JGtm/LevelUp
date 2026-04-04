"""Bullet chart win rate session vs historique par carte.

Extrait de maps_outcome.py pour respecter la limite de 500 lignes.
Contient : _sort_by_map_order, plot_map_winrate_bullet et ses helpers privés.
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.config import HALO_COLORS, PLOT_CONFIG
from src.visualization._compat import DataFrameLike, ensure_polars, to_pandas_for_plotly
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom

# ─── Constantes bullet ───────────────────────────────────────────────────────

_BULLET_DELTA_THRESHOLD = 0.05  # ±5 % considéré comme équivalent
_OKABE_ROSE = "#CC79A7"  # Okabe-Ito reddish-purple (rose/lilas daltonien-safe)


# ─── Helper partagé (réexporté vers maps_outcome.py) ─────────────────────────


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
            # 1er : hist arrière-plan pour le cas under
            "x": under_hist_x,
            "name": hist_lbl,
            "legendgroup": "hist",
            "marker": {"color": rose_behind, "line": {"width": 0}},
            "showlegend": True,
            "customdata": hist_cd,
            "hovertemplate": h_tmpl,
        },
        {
            # 2e : barre session arrière pour le cas over (60% opacité)
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
            # 3e : hist cap au premier plan pour le cas over
            "x": over_hist_x,
            "name": hist_lbl,
            "legendgroup": "hist",
            "marker": {"color": rose_front, "line": {"width": 0}},
            "showlegend": False,
            "customdata": hist_cd,
            "hovertemplate": h_tmpl,
        },
        {
            # 4e (LAST = premier plan Plotly) : barre session pour le cas under
            "x": under_sess_x,
            "name": sess_lbl,
            "legendgroup": "sess",
            "marker": {"color": under_colors, "line": {"width": 0}},
            "showlegend": True,
            "customdata": sess_cd,
            "hovertemplate": s_tmpl,
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
    # rose_behind = rose_front : under_sess est en dernier (premier plan), donc la barre rose
    # n'est visible que là où elle dépasse la barre de session — pas besoin de transparence.
    rose_behind = _OKABE_ROSE
    rose_front = _OKABE_ROSE
    transp = "rgba(0,0,0,0)"
    under = [not ov for ov in over_mask]
    under_colors = [c if u else transp for c, u in zip(curr_colors, under, strict=False)]
    over_colors = [c if ov else transp for c, ov in zip(curr_colors, over_mask, strict=False)]
    _add_bullet_bar_traces(
        fig, d, over_mask, under_colors, over_colors, rose_behind, rose_front, hist_lbl, sess_lbl
    )
    _add_bullet_color_legend_traces(fig, colors, lang)

    # Marqueurs visibles pour les cartes où le win rate session = 0 %
    # (barre de longueur 0 = invisible sans cet indicateur)
    zero_maps = [m for m, wr in zip(d["map_name"], d["win_rate"], strict=False) if float(wr) == 0.0]  # type: ignore[index]
    if zero_maps:
        _wr0_lbl = "0% (toutes défaites)" if lang == "fr" else "0% (all losses)"
        fig.add_trace(
            go.Scatter(
                x=[0.0] * len(zero_maps),
                y=zero_maps,
                mode="markers",
                marker={
                    "symbol": "line-ns",
                    "size": 14,
                    "color": colors["red"],
                    "line": {"color": colors["red"], "width": 3},
                },
                name=_wr0_lbl,
                showlegend=False,
                hovertemplate=f"%{{y}}<br>{sess_lbl}=0%<extra></extra>",
            )
        )


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
