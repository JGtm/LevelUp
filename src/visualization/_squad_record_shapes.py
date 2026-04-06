"""Formes Plotly pour afficher les records historiques sur les graphes escouade.

Chaque record est représenté par une trace go.Bar fantôme hachurée dans la
couleur du joueur (marker_pattern_shape="/"). Les barres de données doivent
utiliser offsetgroup=name pour que les barres fantômes se positionnent et
se dimensionnent exactement comme les barres réelles.

Pour le mode overlay (hs_pk_stacked), on utilise add_overlay_record_shapes
qui trace une ligne horizontale colorée par joueur.
"""

from __future__ import annotations

import plotly.graph_objects as go

from src.visualization._plot_options import DEFAULT_THEME

# Paramètres de mise en forme
_BAR_GAP: float = 0.2  # bargap Plotly par défaut (gap entre groupes)
_BAR_FILL: float = 1.0  # fraction de la largeur de baton (1.0 = exacte)
_OVERLAY_LINE_WIDTH: float = 3.0  # épaisseur de la ligne record en mode overlay
# Pattern hachurage — source unique : DEFAULT_THEME
_PATTERN_SHAPE: str = DEFAULT_THEME.record_pattern_shape
_PATTERN_OPACITY: float = DEFAULT_THEME.record_pattern_opacity
_PATTERN_SIZE: int = DEFAULT_THEME.record_pattern_size
_PATTERN_SOLIDITY: float = DEFAULT_THEME.record_pattern_solidity
_GHOST_BORDER_WIDTH: float = DEFAULT_THEME.record_border_width


def _bar_half_width(n_players: int) -> float:
    """Demi-largeur d'un baton pour n joueurs groupés (pour tests uniquement)."""
    bar_w = (1.0 - _BAR_GAP) / n_players
    return bar_w * _BAR_FILL / 2


def _bar_center_offset(player_idx: int, n_players: int) -> float:
    """Offset horizontal du centre du baton (pour tests uniquement)."""
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
    """Ajoute des traces go.Bar fantômes hachurées pour les records historiques.

    Chaque barre fantôme utilise offsetgroup=name pour se superposer
    exactement à la barre de données correspondante (même position, même
    largeur). La couleur et le hachurage correspondent à la couleur du joueur.

    Prérequis : les traces de données doivent utiliser offsetgroup=name.

    Args:
        fig: Figure Plotly à modifier in-place.
        xs: Positions X des matchs (entiers 0..N-1).
        records: Dict {nom_joueur: valeur_record_globale (ou None)}.
        player_names: Noms des joueurs dans l'ordre des traces.
        n_players: Non utilisé (garde pour compatibilité signature).
        is_negative: Si True, barres sous l'axe X (ex: morts).
        colors_by_name: Mapping nom_joueur → couleur hex.
        per_map_records: {joueur: {métrique: {carte: valeur}}}.
        map_names_per_x: Nom de la carte à chaque position X.
    """
    for name in player_names:
        color = (colors_by_name or {}).get(name, "#ffffff")
        y_vals: list[float | None] = []
        for xi in range(len(xs)):
            rv = _resolve_record(name, xi, records, per_map_records, map_names_per_x)
            y_vals.append((-rv if is_negative else rv) if rv is not None else None)
        if all(v is None for v in y_vals):
            continue
        fig.add_trace(
            go.Bar(
                x=xs,
                y=y_vals,
                name=name,
                showlegend=False,
                legendgroup=name,
                offsetgroup=name,
                marker_color="rgba(0,0,0,0)",
                marker_pattern_shape=_PATTERN_SHAPE,
                marker_pattern_fgcolor=color,
                marker_pattern_fgopacity=_PATTERN_OPACITY,
                marker_pattern_size=_PATTERN_SIZE,
                marker_pattern_solidity=_PATTERN_SOLIDITY,
                marker_line_color=color,
                marker_line_width=_GHOST_BORDER_WIDTH,
                hoverinfo="skip",
            )
        )


def add_overlay_record_shapes(
    fig: go.Figure,
    *,
    xs: list[int],
    records: dict[str, float | None],
    player_names: list[str],
    colors_by_name: dict[str, str] | None = None,
) -> None:
    """Ajoute les lignes de record pour barmode=overlay.

    Utilisé pour plot_hs_pk_stacked où les barres se superposent au même X.
    En overlay, les rects de joueurs différents se cacheraient mutuellement.
    On dessine donc une ligne horizontale colorée par joueur à la hauteur
    de son record — chaque joueur a sa propre couleur, distinguable même
    si plusieurs lignes se croisent.

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
        for x in xs:
            fig.add_shape(
                type="line",
                xref="x",
                yref="y",
                x0=x - half_w,
                x1=x + half_w,
                y0=record_val,
                y1=record_val,
                line={"color": color, "width": _OVERLAY_LINE_WIDTH},
                opacity=0.88,
                layer="above",
            )


def _resolve_record(
    name: str,
    xi: int,
    records: dict[str, float | None],
    per_map_records: dict[str, dict[str, float | None]] | None,
    map_names_per_x: list[str | None] | None,
) -> float | None:
    """Retourne la valeur record à utiliser : par-carte si possible, sinon globale.

    Priorité : record par carte > record global. Si le record par carte est None
    (carte absente ou dict vide), fallback systématique sur le record global.
    """
    if per_map_records is not None and map_names_per_x is not None:
        map_name = map_names_per_x[xi] if xi < len(map_names_per_x) else None
        if map_name:
            per_map_val = (per_map_records.get(name) or {}).get(map_name)
            if per_map_val is not None:
                return per_map_val
            # Fallback : carte introuvable dans per_map → afficher le record global
    return records.get(name)
