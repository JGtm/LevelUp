"""Formes Plotly pour afficher les records historiques sur les graphes escouade.

Chaque record est représenté par un rectangle transparent de la largeur exacte
du baton correspondant, avec une bordure épaisse en couleur du joueur et un
remplissage légèrement teinté (simulant un hachurage).
"""

from __future__ import annotations

import plotly.graph_objects as go

# Paramètres de mise en forme
_BAR_GAP: float = 0.2        # bargap Plotly par défaut (gap entre groupes)
_BAR_FILL: float = 0.85      # fraction de la largeur de baton à couvrir
_BORDER_WIDTH: float = 2.0   # épaisseur de la bordure colorée
_FILL_OPACITY: float = 0.14  # opacité du remplissage teinté
_SHAPE_OPACITY: float = 0.88 # opacité globale de la forme


def _hex_to_rgba(hex_color: str, opacity: float) -> str:
    """Convertit #RRGGBB en rgba(r,g,b,opacity)."""
    h = hex_color.lstrip("#")
    if len(h) < 6:
        return f"rgba(255,255,255,{opacity})"
    r, g, b = int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16)
    return f"rgba({r},{g},{b},{opacity})"


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
    colors_by_name: dict[str, str] | None = None,
    per_map_records: dict[str, dict[str, float | None]] | None = None,
    map_names_per_x: list[str | None] | None = None,
) -> None:
    """Ajoute les rectangles de record sur une figure barres groupées.

    Pour chaque joueur et chaque position de match, trace un rectangle
    transparent avec bordure colorée de la largeur exacte du baton.

    Args:
        fig: Figure Plotly à modifier in-place.
        xs: Positions X des matchs (entiers 0..N-1).
        records: Dict {nom_joueur: valeur_record_globale (ou None)}.
        player_names: Noms des joueurs dans l'ordre des traces.
        n_players: Nombre total de joueurs (pour calcul des offsets).
        is_negative: Si True, le record est tracé sous l'axe X (ex: morts).
        colors_by_name: Mapping nom_joueur → couleur hex pour les bordures.
        per_map_records: Dict {nom_joueur: {map_name: valeur}} pour le mode
            par-carte. Utilisé si map_names_per_x est aussi fourni.
        map_names_per_x: Nom de la carte à chaque position X. Quand fourni
            avec per_map_records, utilise le record propre à chaque carte.
    """
    half_w = _bar_half_width(n_players)
    for p_idx, name in enumerate(player_names):
        color = (colors_by_name or {}).get(name, "#ffffff")
        fill = _hex_to_rgba(color, _FILL_OPACITY) if color.startswith("#") else f"rgba(255,255,255,{_FILL_OPACITY})"
        offset = _bar_center_offset(p_idx, n_players)
        for xi, x in enumerate(xs):
            record_val = _resolve_record(
                name, xi, records, per_map_records, map_names_per_x
            )
            if record_val is None:
                continue
            y_val = -record_val if is_negative else record_val
            y0, y1 = (y_val, 0.0) if is_negative else (0.0, y_val)
            fig.add_shape(
                type="rect",
                xref="x", yref="y",
                x0=x + offset - half_w,
                x1=x + offset + half_w,
                y0=y0, y1=y1,
                fillcolor=fill,
                line={"color": color, "width": _BORDER_WIDTH},
                opacity=_SHAPE_OPACITY,
                layer="above",
            )


def add_overlay_record_shapes(
    fig: go.Figure,
    *,
    xs: list[int],
    records: dict[str, float | None],
    player_names: list[str],
    colors_by_name: dict[str, str] | None = None,
) -> None:
    """Ajoute les rectangles de record pour barmode=overlay (sans offset horizontal).

    Utilisé pour plot_hs_pk_stacked où les barres se superposent au même X.
    En mode overlay, chaque barre occupe la largeur totale du groupe.

    Args:
        fig: Figure Plotly à modifier in-place.
        xs: Positions X des matchs.
        records: Dict {nom_joueur: valeur_record (ou None)}.
        player_names: Noms des joueurs dans l'ordre des traces.
        colors_by_name: Mapping nom_joueur → couleur hex.
    """
    half_w = (1.0 - _BAR_GAP) * _BAR_FILL / 2
    for name in player_names:
        record_val = records.get(name)
        if record_val is None:
            continue
        color = (colors_by_name or {}).get(name, "#ffffff")
        fill = _hex_to_rgba(color, _FILL_OPACITY) if color.startswith("#") else f"rgba(255,255,255,{_FILL_OPACITY})"
        for x in xs:
            fig.add_shape(
                type="rect",
                xref="x", yref="y",
                x0=x - half_w,
                x1=x + half_w,
                y0=0.0, y1=record_val,
                fillcolor=fill,
                line={"color": color, "width": _BORDER_WIDTH},
                opacity=_SHAPE_OPACITY,
                layer="above",
            )


def _resolve_record(
    name: str,
    xi: int,
    records: dict[str, float | None],
    per_map_records: dict[str, dict[str, float | None]] | None,
    map_names_per_x: list[str | None] | None,
) -> float | None:
    """Retourne la valeur record à utiliser : par-carte si possible, sinon globale."""
    if per_map_records is not None and map_names_per_x is not None:
        map_name = map_names_per_x[xi] if xi < len(map_names_per_x) else None
        if map_name:
            return (per_map_records.get(name) or {}).get(map_name)
    return records.get(name)
