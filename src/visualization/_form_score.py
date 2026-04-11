"""Graphe historique du score de forme (rolling avg_14 - avg_90).

Deux modes :
- Standard : courbe form_score lissée, 1 point par match.
- Détail   : courbe unique fusionnant les points match et les buckets intra-match
  (≤ DETAIL_THRESHOLD matchs). Les buckets sont triés dans la timeline globale.
"""

from __future__ import annotations

import contextlib

import plotly.graph_objects as go
import polars as pl

from src.ui.i18n.viz import viz_t
from src.visualization._plot_options import DEFAULT_THEME
from src.visualization.theme import apply_halo_plot_style


def _merge_series_with_buckets(
    match_data: list[dict],
    bucket_df: pl.DataFrame,
) -> tuple[list, list, list]:
    """Fusionne les points match et les buckets intra-match en une courbe unique.

    Returns:
        Tuple (x_vals, y_vals, match_ids) triés chronologiquement.
        Les bucket points ont un match_id vide (non surlignés).
    """
    combined = [
        {"t": d["start_time"], "v": d.get("form_score"), "mid": str(d.get("match_id", ""))}
        for d in match_data
    ]
    for row in bucket_df.sort("bucket_start").to_dicts():
        combined.append({"t": row["bucket_start"], "v": row.get("bucket_value"), "mid": ""})
    with contextlib.suppress(TypeError):
        combined.sort(key=lambda r: r["t"] if r["t"] is not None else "")
    return (
        [r["t"] for r in combined],
        [r["v"] for r in combined],
        [r["mid"] for r in combined],
    )


def _add_fill_traces(fig: go.Figure, x_vals: list, y_vals: list) -> None:
    """Ajoute le remplissage vert/rouge autour de zéro (vue joueur unique)."""
    y_pos = [max(0.0, v) if v is not None else 0.0 for v in y_vals]
    y_neg = [min(0.0, v) if v is not None else 0.0 for v in y_vals]
    for y_fill, fill_color in (
        (y_pos, "rgba(0,158,115,0.18)"),
        (y_neg, "rgba(213,94,0,0.18)"),
    ):
        fig.add_trace(
            go.Scatter(
                x=x_vals,
                y=y_fill,
                fill="tozeroy",
                fillcolor=fill_color,
                line={"color": "rgba(0,0,0,0)"},
                hoverinfo="skip",
                showlegend=False,
            )
        )


def _highlighted_points(
    x_vals: list,
    y_vals: list,
    match_ids: list[str],
    highlight_ids: set[str],
) -> tuple[list, list]:
    """Retourne les coordonnées (x, y) des points à surcharger."""
    pairs = [
        (xv, yv)
        for xv, yv, mid in zip(x_vals, y_vals, match_ids, strict=False)
        if mid in highlight_ids
    ]
    if not pairs:
        return [], []
    xs, ys = zip(*pairs, strict=False)
    return list(xs), list(ys)


def _key_label_indices(y_vals: list) -> set[int]:
    """Retourne les indices des points clés à annoter : premier, dernier, min, max."""
    valid = [(i, v) for i, v in enumerate(y_vals) if v is not None]
    if not valid:
        return set()
    indices = {valid[0][0], valid[-1][0]}
    min_i = min(valid, key=lambda t: t[1])[0]
    max_i = max(valid, key=lambda t: t[1])[0]
    indices.add(min_i)
    indices.add(max_i)
    return indices


def _add_player_line(  # noqa: PLR0913
    fig: go.Figure,
    name: str,
    x_vals: list,
    y_vals: list,
    color: str,
    highlight_match_ids: set[str] | None,
    match_ids: list[str],
) -> None:
    """Ajoute la ligne de forme et les points de session surlignés pour un joueur."""
    key_indices = _key_label_indices(y_vals)
    text_labels = [
        f"{v:+.1f}" if (v is not None and i in key_indices) else "" for i, v in enumerate(y_vals)
    ]
    # Positionner les étiquettes : min en bas, autres en haut
    min_i = min(
        (i for i, v in enumerate(y_vals) if v is not None),
        key=lambda i: y_vals[i],
        default=None,
    )
    text_positions = ["bottom center" if i == min_i else "top center" for i in range(len(y_vals))]
    fig.add_trace(
        go.Scatter(
            x=x_vals,
            y=y_vals,
            mode="lines+text",
            name=name,
            line={"color": color, "width": 2},
            text=text_labels,
            textposition=text_positions,
            textfont={"size": 11, "color": color},
            hovertemplate="%{customdata}<extra></extra>",
            customdata=[
                f"<b>{name}</b><br>Forme : {v:+.1f}" if v is not None else "" for v in y_vals
            ],
        )
    )
    if not highlight_match_ids:
        return
    hi_x, hi_y = _highlighted_points(x_vals, y_vals, match_ids, highlight_match_ids)
    if hi_x:
        fig.add_trace(
            go.Scatter(
                x=hi_x,
                y=hi_y,
                mode="markers",
                name=f"{name} (session)",
                marker={"size": 9, "color": color, "line": {"width": 2, "color": "white"}},
                hoverinfo="skip",
                showlegend=False,
            )
        )


def plot_form_score_history(  # noqa: PLR0913
    series_by_name: dict[str, pl.DataFrame],
    *,
    highlight_match_ids: set[str] | None = None,
    colors_by_name: dict[str, str] | None = None,
    lang: str = "fr",
    height: int = 280,
    bucket_series_by_name: dict[str, pl.DataFrame] | None = None,
) -> go.Figure | None:
    """Graphe de l'évolution du score de forme dans le temps.

    Pour un joueur : ligne colorée + remplissage vert/rouge selon le signe.
    Pour une escouade : une ligne colorée par joueur.
    La session sélectionnée est mise en évidence par des points encerclés.

    En mode détail (bucket_series_by_name non vide), ajoute un scatter de points
    intra-match ancrés sur le form_score du match parent.

    Args:
        series_by_name: Mapping nom → DataFrame avec start_time + form_score + match_id.
        highlight_match_ids: Match IDs à surcharger (session courante).
        colors_by_name: Couleur par nom de joueur.
        lang: Langue d'affichage.
        height: Hauteur du graphe en pixels.
        bucket_series_by_name: Mapping nom → DataFrame buckets (bucket_start, bucket_value).
            Si None ou vide, mode standard sans buckets.

    Returns:
        Figure Plotly ou None si toutes les séries sont vides / sans form_score.
    """
    non_empty = {
        name: df
        for name, df in series_by_name.items()
        if not df.is_empty() and "form_score" in df.columns
    }
    if not non_empty:
        return None

    fig = go.Figure()
    is_single = len(non_empty) == 1

    for name, df in non_empty.items():
        color = (colors_by_name or {}).get(name, DEFAULT_THEME.solo_color)
        data = df.sort("start_time").to_dicts()
        x_vals = [d["start_time"] for d in data]
        y_vals = [d.get("form_score") for d in data]
        match_ids = [str(d.get("match_id", "")) for d in data]

        # Mode détail : fusion buckets + matchs en courbe unique
        if bucket_series_by_name:
            bucket_df = bucket_series_by_name.get(name)
            if bucket_df is not None and not bucket_df.is_empty():
                x_vals, y_vals, match_ids = _merge_series_with_buckets(data, bucket_df)
        if is_single:
            _add_fill_traces(fig, x_vals, y_vals)
        _add_player_line(fig, name, x_vals, y_vals, color, highlight_match_ids, match_ids)

    fig.add_hline(
        y=0,
        line_width=1.5,
        line_color="rgba(255,255,255,0.4)",
        annotation_text=viz_t("label_balance", lang),
        annotation_position="right",
    )
    fig.update_layout(
        yaxis_title=viz_t("axis_form_score", lang),
        xaxis_title="",
        hovermode="x unified",
        showlegend=not is_single,
        legend={
            "orientation": "h",
            "yanchor": "top",
            "y": -0.25,
            "xanchor": "center",
            "x": 0.5,
        },
        margin={"t": 60, "b": 10, "l": 50, "r": 20},
    )
    return apply_halo_plot_style(fig, height=height)
