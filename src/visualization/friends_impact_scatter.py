"""Scatter plot d'impact des coéquipiers — version points avec emojis.

Chaque type d'événement est une trace Plotly distincte avec son propre
signe emoji. Les outcomes
sont représentés par des rectangles de fond colorés (vert/rouge/mauve).

Usage::

    from src.visualization.friends_impact_scatter import plot_friends_impact_scatter
    fig = plot_friends_impact_scatter(impact_matrix, max_matches=40)
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.config import HALO_COLORS
from src.data.domain.refdata import Outcome
from src.ui.i18n.viz import viz_t
from src.visualization.friends_impact_heatmap import OUTCOME_COLORS
from src.visualization.theme import apply_halo_plot_style

# (event_type, label_fr, color, emoji)
_EVENT_SCATTER_CONFIG: list[tuple[str, str, str, str]] = [
    ("first_blood", "Premier sang", "#009E73", "⚡"),
    ("clutch_finisher", "Finisseur", "#E69F00", "🎯"),
    ("last_casualty", "Boulet", "#D55E00", "💀"),
    ("last_group_kill", "Touriste", "#56B4E9", "🐌"),
    ("first_group_death", "1ère victime", "#CC79A7", "🪦"),
    ("silent_hero", "Héros silencieux", "#3DFFB5", "🛡️"),
    ("false_brother", "Faux-frère", "#FF4B4B", "🗡️"),
]

_EVENT_TYPE_SET: frozenset[str] = frozenset(et for et, *_ in _EVENT_SCATTER_CONFIG)


def _extract_outcomes(
    impact_matrix: pl.DataFrame,
    all_gamertags: list[str],
) -> dict[str, int]:
    """Extrait match_id → outcome depuis la ligne virtuelle 'Résultat'."""
    if "Résultat" not in all_gamertags:
        return {}
    result: dict[str, int] = {}
    for row in impact_matrix.filter(pl.col("gamertag") == "Résultat").iter_rows(named=True):
        if row["outcome"] is not None:
            result[str(row["match_id"])] = int(row["outcome"])
    return result


def _collect_scatter_coords(
    impact_matrix: pl.DataFrame,
    match_to_x: dict[str, int],
    gamertag_to_y: dict[str, int],
) -> dict[str, tuple[list, list, list]]:
    """Retourne {event_type: (xs, ys, hovers)} pour les traces scatter.

    Quand un joueur a plusieurs événements dans le même match, un léger
    décalage horizontal est appliqué pour éviter la superposition.
    """
    # 1re passe : collecter les events par cellule (match_id, gamertag)
    cell_events: dict[tuple[str, str], list[str]] = {}
    for row in impact_matrix.filter(pl.col("gamertag") != "Résultat").iter_rows(named=True):
        mid, gt = str(row["match_id"]), row["gamertag"]
        if mid not in match_to_x or gt not in gamertag_to_y:
            continue
        evts = [e["event"] for e in (row["events"] or []) if e["event"] in _EVENT_TYPE_SET]
        if evts:
            cell_events[(mid, gt)] = evts

    # 2e passe : offsets X et construction des listes de coordonnées
    coords: dict[str, tuple[list, list, list]] = {
        et: ([], [], []) for et, *_ in _EVENT_SCATTER_CONFIG
    }
    for (mid, gt), evts in cell_events.items():
        x_base = float(match_to_x[mid])
        y_val = float(gamertag_to_y[gt])
        n = len(evts)
        offsets = [0.0] if n == 1 else [(-0.28 + i * 0.56 / (n - 1)) for i in range(n)]
        for et, x_off in zip(evts, offsets, strict=False):
            if et not in coords:
                continue
            xs, ys, hovers = coords[et]
            xs.append(x_base + x_off)
            ys.append(y_val)
            hovers.append(f"<b>{gt}</b><br>Match #{match_to_x[mid] + 1}")
    return coords


def _build_scatter_outcome_shapes(
    match_ids: list[str],
    match_outcomes_map: dict[str, int],
    n_players: int,
) -> list[dict]:
    """Rectangles de fond colorés par outcome (une colonne entière par match)."""
    shapes = []
    for x_idx, mid in enumerate(match_ids):
        outcome = match_outcomes_map.get(mid, 0)
        if outcome == Outcome.WIN:
            fc = OUTCOME_COLORS["win"]
        elif outcome == Outcome.LOSS:
            fc = OUTCOME_COLORS["loss"]
        elif outcome == Outcome.TIE:
            fc = OUTCOME_COLORS["tie"]
        else:
            continue
        shapes.append(
            {
                "type": "rect",
                "x0": x_idx - 0.45,
                "x1": x_idx + 0.45,
                "y0": -0.5,
                "y1": n_players - 0.5,
                "fillcolor": fc,
                "opacity": 0.12,
                "line": {"width": 0},
                "layer": "below",
            }
        )
    return shapes


def _add_scatter_traces(
    fig: go.Figure,
    coords: dict,
    n_matches: int,
    n_players: int,
) -> None:
    """Ajoute la grille fantôme et les traces d'événements emoji."""
    fig.add_trace(
        go.Scatter(
            x=[i for i in range(n_matches) for _ in range(n_players)],
            y=[j for _ in range(n_matches) for j in range(n_players)],
            mode="markers",
            marker={"size": 34, "color": "rgba(0,0,0,0)"},
            showlegend=False,
            hoverinfo="skip",
        )
    )
    for event_type, label, color, emoji in _EVENT_SCATTER_CONFIG:
        xs, ys, hovers = coords[event_type]
        if not xs:
            continue
        fig.add_trace(
            go.Scatter(
                x=xs,
                y=ys,
                mode="text",
                name=f"{emoji} {label}",
                hovertext=hovers,
                hoverinfo="text",
                text=[emoji] * len(xs),
                textposition="middle center",
                textfont={
                    "size": 19,
                    "color": color,
                },
            )
        )


def _apply_scatter_layout(  # noqa: PLR0913
    fig: go.Figure,
    shapes: list,
    n_matches: int,
    n_players: int,
    gamertags: list[str],
    *,
    title: str | None,
    calc_height: int,
    lang: str,
) -> None:
    """Configure le layout axes/légende/fond du scatter."""
    fig.update_layout(
        shapes=shapes,
        height=calc_height,
        showlegend=False,
        margin={"l": 130, "r": 40, "t": 64 if title else 34, "b": 50},
        xaxis={
            "title": viz_t("axis_matches", lang),
            "tickvals": list(range(n_matches)),
            "ticktext": [f"#{i + 1}" for i in range(n_matches)],
            "showgrid": True,
            "gridcolor": "rgba(200,200,200,0.3)",
            "zeroline": False,
            "range": [-0.5, n_matches - 0.5],
        },
        yaxis={
            "tickvals": list(range(n_players)),
            "ticktext": gamertags,
            "showgrid": True,
            "gridcolor": "rgba(160,160,160,0.16)",
            "zeroline": False,
            "range": [-0.5, n_players - 0.5],
        },
        plot_bgcolor="rgba(248,250,252,0.92)",
    )


def plot_friends_impact_scatter(
    impact_matrix: pl.DataFrame,
    *,
    title: str | None = None,
    max_matches: int = 50,
    height: int | None = None,
    match_ids_order: list[str] | None = None,
) -> go.Figure:
    """Trace la matrice d'impact joueur × match en vue emoji."""
    colors = HALO_COLORS.as_dict()
    lang = "fr"

    if impact_matrix.is_empty():
        fig = go.Figure()
        fig.add_annotation(
            text=viz_t("empty_no_impact_events", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 14, "color": colors.get("slate", "#64748b")},
        )
        return apply_halo_plot_style(fig, title=title, height=height or 300)

    all_gt = sorted(impact_matrix["gamertag"].unique().to_list())
    gamertags = [g for g in all_gt if g != "Résultat"]
    available_match_ids = {str(m) for m in impact_matrix["match_id"].to_list()}
    if match_ids_order:
        match_ids = [str(m) for m in match_ids_order if str(m) in available_match_ids]
    else:
        # Fallback deterministe: ordre d'apparition dans la matrice.
        match_ids = []
        seen: set[str] = set()
        for mid in impact_matrix["match_id"].to_list():
            mid_s = str(mid)
            if mid_s not in seen:
                seen.add(mid_s)
                match_ids.append(mid_s)
    match_outcomes_map = _extract_outcomes(impact_matrix, all_gt)

    if len(match_ids) > max_matches:
        match_ids = match_ids[:max_matches]
        impact_matrix = impact_matrix.filter(pl.col("match_id").is_in(match_ids))

    if not match_ids or not gamertags:
        return apply_halo_plot_style(go.Figure(), title=title, height=height or 300)

    n_matches, n_players = len(match_ids), len(gamertags)
    match_to_x = {mid: i for i, mid in enumerate(match_ids)}
    gamertag_to_y = {gt: i for i, gt in enumerate(gamertags)}

    coords = _collect_scatter_coords(impact_matrix, match_to_x, gamertag_to_y)
    shapes = _build_scatter_outcome_shapes(match_ids, match_outcomes_map, n_players)

    fig = go.Figure()
    _add_scatter_traces(fig, coords, n_matches, n_players)

    calc_height = height or max(300, 55 * n_players + 120)
    _apply_scatter_layout(
        fig,
        shapes,
        n_matches,
        n_players,
        gamertags,
        title=title,
        calc_height=calc_height,
        lang=lang,
    )
    return apply_halo_plot_style(fig, title=title, height=calc_height)
