"""Formes Plotly pour afficher les records historiques sur les graphes escouade.

Chaque record est représenté par une petite barre blanche grasse positionnée
à la hauteur de la valeur record, de la largeur exacte du baton correspondant.
"""

from __future__ import annotations

import plotly.graph_objects as go

# Paramètres de mise en forme
_BAR_GAP: float = 0.2        # bargap Plotly par défaut (gap entre groupes)
_BAR_FILL: float = 0.85      # fraction de la largeur de baton à couvrir
_LINE_WIDTH: float = 3.0     # épaisseur de la barre record


def _bar_half_width(n_players: int) -> float:
    """Demi-largeur d'un baton pour n joueurs groupés."""
    bar_w = (1.0 - _BAR_GAP) / n_players
    return bar_w * _BAR_FILL / 2


def _bar_center_offset(player_idx: int, n_players: int) -> float:
    """Offset horizontal du centre du baton pour le joueur à l'index player_idx."""
    bar_w = (1.0 - _BAR_GAP) / n_players
    return (player_idx - (n_players - 1) / 2) * bar_w


def add_record_shapes(  # noqa: PLR0913
    fig: go.Figure,
    *,
    xs: list[int],
    records: dict[str, float | None],
    player_names: list[str],
    n_players: int,
    is_negative: bool = False,
) -> None:
    """Ajoute les barres blanches de record sur une figure barres groupées.

    Trace une petite ligne horizontale blanche grasse à la hauteur du record
    pour chaque joueur à chaque position de match.

    Args:
        fig: Figure Plotly à modifier in-place.
        xs: Positions X des matchs (entiers 0..N-1).
        records: Dict {nom_joueur: valeur_record (ou None)}.
        player_names: Noms des joueurs dans l'ordre des traces.
        n_players: Nombre total de joueurs (pour calcul des offsets).
        is_negative: Si True, le record est tracé en négatif (ex : morts).
    """
    half_w = _bar_half_width(n_players)
    for p_idx, name in enumerate(player_names):
        record_val = records.get(name)
        if record_val is None:
            continue
        y_val = -record_val if is_negative else record_val
        offset = _bar_center_offset(p_idx, n_players)
        for x in xs:
            fig.add_shape(
                type="line",
                xref="x",
                yref="y",
                x0=x + offset - half_w,
                x1=x + offset + half_w,
                y0=y_val,
                y1=y_val,
                line={"color": "white", "width": _LINE_WIDTH},
                opacity=0.9,
                layer="above",
            )


def add_overlay_record_shapes(
    fig: go.Figure,
    *,
    xs: list[int],
    records: dict[str, float | None],
    player_names: list[str],
) -> None:
    """Ajoute les barres de record pour barmode=overlay (pas d'offset horizontal).

    Utilisé pour plot_hs_pk_stacked où les barres se superposent au même X.
    En mode overlay, chaque barre occupe la largeur totale du groupe → half_w fixe.

    Args:
        fig: Figure Plotly à modifier in-place.
        xs: Positions X des matchs.
        records: Dict {nom_joueur: valeur_record (ou None)}.
        player_names: Noms des joueurs dans l'ordre des traces.
    """
    # En overlay, barre pleine = (1 - bargap) * fill, indépendant du nb de joueurs
    half_w = (1.0 - _BAR_GAP) * _BAR_FILL / 2
    for name in player_names:
        record_val = records.get(name)
        if record_val is None:
            continue
        for x in xs:
            fig.add_shape(
                type="line",
                xref="x",
                yref="y",
                x0=x - half_w,
                x1=x + half_w,
                y0=record_val,
                y1=record_val,
                line={"color": "white", "width": _LINE_WIDTH},
                opacity=0.9,
                layer="above",
            )
