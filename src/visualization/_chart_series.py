"""Abstractions ChartData / MatchSeries pour les graphes multi-joueurs escouade.

Centralise :
- les constantes de hauteur (HEIGHT_*)
- MAX_PLOT_POINTS pour le downsampling
- MatchSeries : une série de données par joueur
- ChartData : container complet pour un graphe multi-joueurs
  - add_record_overlays(fig) : dispatche vers add_record_shapes / add_overlay_record_shapes
  - downsample(max_points) : réduit le nombre de points pour Plotly

Pas d'import Streamlit, pas d'accès DB — testable en isolation.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Literal

import plotly.graph_objects as go

# ─── Constantes hauteur (consolide les magic numbers éparpillés) ──────────────
HEIGHT_COMPACT: int = 320   # barres multi-joueurs (match_bars.py)
HEIGHT_NORMAL: int = 360    # graphes trio standard (sync avec PLOT_CONFIG.default_height)
HEIGHT_PM: int = 350        # stats par-minute (_teammates_trio_helpers.py)

# ─── Downsampling ─────────────────────────────────────────────────────────────
MAX_PLOT_POINTS: int = 200  # au-delà Plotly devient lent


@dataclass
class SquadRecordSet:
    """Enveloppe légère regroupant records + records_per_map pour une escouade.

    Remplace le passage de deux dicts séparés (records, records_per_map) à travers
    toute la chaîne caller→viz.  Construit une seule fois dans ``_teammates_trio.py``
    et transmis tel quel jusqu'aux fonctions de visualisation.

    Attributs :
        records      : ``{nom_joueur: {métrique: valeur_record}}``
        per_map      : ``{nom_joueur: {nom_carte: {métrique: valeur_record}}}``
    """

    records: dict[str, dict[str, float | None]] = field(default_factory=dict)
    per_map: dict[str, dict[str, dict[str, float | None]]] = field(default_factory=dict)


@dataclass
class MatchSeries:
    """Une série de données pour un joueur sur l'axe X commun de matchs."""

    name: str                    # nom du joueur (= offsetgroup Plotly)
    x: list[int]                 # positions entières normalisées 0..N-1
    y: list[float | None]        # valeurs métriques
    color: str                   # couleur hex du joueur
    map_names: list[str | None]  # carte à chaque position x (pour records par carte)


@dataclass
class ChartData:
    """Container pour un graphe multi-joueurs par match.

    Transporte :
    - les séries (MatchSeries) avec positions x normalisées, valeurs y, couleurs
    - les labels de l'axe X (tick labels)
    - le barmode Plotly
    - les records globaux : {nom_joueur: valeur_record | None}
    - les records par carte : {nom_joueur: {nom_carte: valeur_record | None}}

    Usage minimal :
        chart_data = ChartData(series=[...], x_labels=[...], barmode="group")
        fig = build_figure(chart_data)
        chart_data.add_record_overlays(fig)
    """

    series: list[MatchSeries]
    x_labels: list[str]
    barmode: Literal["group", "overlay", "categorical"]
    global_records: dict[str, float | None] = field(default_factory=dict)
    per_map_records: dict[str, dict[str, float | None]] = field(default_factory=dict)
    is_negative: bool = False

    # ── Propriétés dérivées ────────────────────────────────────────────────────

    @property
    def player_names(self) -> list[str]:
        return [s.name for s in self.series]

    @property
    def colors_by_name(self) -> dict[str, str]:
        return {s.name: s.color for s in self.series}

    @property
    def tick_step(self) -> int:
        n = len(self.x_labels)
        return max(1, n // 10) if n else 1

    # ── Downsampling ───────────────────────────────────────────────────────────

    def downsample(self, max_points: int = MAX_PLOT_POINTS) -> ChartData:
        """Réduit le nombre de matchs affichés si trop de points pour Plotly.

        Retourne self si déjà sous le seuil (pas de copie inutile).
        """
        n = max((len(s.x) for s in self.series), default=0)
        if n <= max_points:
            return self
        step = max(1, n // max_points)
        new_series = [
            MatchSeries(
                name=s.name,
                color=s.color,
                x=s.x[::step],
                y=s.y[::step],
                map_names=s.map_names[::step],
            )
            for s in self.series
        ]
        return ChartData(
            series=new_series,
            x_labels=self.x_labels[::step],
            barmode=self.barmode,
            global_records=self.global_records,
            per_map_records=self.per_map_records,
            is_negative=self.is_negative,
        )

    # ── Overlays records ───────────────────────────────────────────────────────

    def add_record_overlays(self, fig: go.Figure) -> None:
        """Ajoute les traces de records sur la figure selon le barmode.

        - "group"   → barres fantômes hachurées via add_record_shapes
        - "overlay" → lignes horizontales via add_overlay_record_shapes
        - "categorical" → barres fantômes catégorielles via _add_categorical_record_bars
        """
        if not self.global_records and not self.per_map_records:
            return

        if self.barmode == "overlay":
            from src.visualization._squad_record_shapes import add_overlay_record_shapes

            xs_ref = self.series[0].x if self.series else []
            add_overlay_record_shapes(
                fig,
                xs=xs_ref,
                records=self.global_records,
                player_names=self.player_names,
                colors_by_name=self.colors_by_name,
            )

        elif self.barmode == "categorical":
            _add_categorical_record_bars(fig, self)

        else:  # "group"
            from src.visualization._squad_record_shapes import add_record_shapes

            n_players = len(self.series)
            for s in self.series:
                pm = (
                    {s.name: self.per_map_records[s.name]}
                    if s.name in self.per_map_records
                    else None
                )
                map_names_per_x = s.map_names[: len(s.x)]
                add_record_shapes(
                    fig,
                    xs=s.x,
                    records={s.name: self.global_records.get(s.name)},
                    player_names=self.player_names,
                    n_players=n_players,
                    is_negative=self.is_negative,
                    colors_by_name=self.colors_by_name,
                    per_map_records=pm,
                    map_names_per_x=map_names_per_x,
                )


def _add_categorical_record_bars(fig: go.Figure, chart_data: ChartData) -> None:
    """Ajoute des barres fantômes hachurées sur un graphe catégoriel (stats/min).

    Chaque joueur a une barre ghost par catégorie (frags/min, morts/min, assists/min)
    via le même offsetgroup que les barres réelles.
    Les valeurs des records sont dans chart_data.global_records :
    {nom: (r_kpm, r_dpm, r_apm)} — tuple attendu de 3 float|None.
    Les morts sont stockées positives mais affichées négatives (y = -r_dpm).
    """
    for s in chart_data.series:
        rec = chart_data.global_records.get(s.name)
        if rec is None:
            continue
        # rec est un tuple (r_kpm, r_dpm, r_apm)
        if not isinstance(rec, tuple) or len(rec) != 3:
            continue
        r_kpm, r_dpm, r_apm = rec
        y_rec = [
            r_kpm if r_kpm is not None else None,
            (-r_dpm) if r_dpm is not None else None,
            r_apm if r_apm is not None else None,
        ]
        if all(v is None for v in y_rec):
            continue
        fig.add_trace(
            go.Bar(
                name=s.name,
                x=chart_data.x_labels,
                y=y_rec,
                showlegend=False,
                legendgroup=s.name,
                offsetgroup=s.name,
                marker_color="rgba(0,0,0,0)",
                marker_pattern_shape="/",
                marker_pattern_fgcolor=s.color,
                marker_pattern_fgopacity=0.55,
                marker_pattern_size=8,
                marker_pattern_solidity=0.35,
                marker_line_color=s.color,
                marker_line_width=2.0,
                hoverinfo="skip",
            )
        )
