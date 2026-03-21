"""Helpers partagés pour les graphiques de séries temporelles.

Centralise les calculs d'axe chronologique et la mise en forme commune
pour éviter la duplication dans chaque fonction de plot.
"""

from __future__ import annotations

import polars as pl

from src.config import HALO_COLORS
from src.ui.date_formats import FMT_TICK_DATETIME
from src.ui.i18n.viz import viz_t

# Palette de couleurs pré-résolue (évite `HALO_COLORS.as_dict()` dans chaque fonction)
COLORS = HALO_COLORS.as_dict()


def prepare_time_axis(d: pl.DataFrame) -> tuple[list[int], list[str], int]:
    """Prépare les données d'axe X chronologique.

    Si ``map_name`` est présente dans ``d``, les labels prennent la forme
    ``#N<br>NomDeLaMap`` ; sinon les dates formatées sont utilisées.

    Args:
        d: DataFrame Polars trié par ``start_time``.

    Returns:
        Tuple ``(x_idx, labels, step)`` :
        - ``x_idx`` : indices entiers ``[0 .. n-1]``
        - ``labels`` : étiquettes formatées pour affichage
        - ``step`` : pas de tick pour éviter le chevauchement
    """
    x_idx = list(range(d.height))
    if "map_name" in d.columns:
        labels = [
            f"#{i + 1}<br>{mn}" if mn else f"#{i + 1}"
            for i, mn in enumerate(d["map_name"].fill_null("").to_list())
        ]
    else:
        labels = d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    step = max(1, len(labels) // 10) if len(labels) > 1 else 1
    return x_idx, labels, step


def apply_chrono_xaxis(  # noqa: PLR0913
    fig: object,
    x_idx: list[int],
    labels: list[str],
    step: int,
    lang: str = "fr",
    *,
    as_category: bool = True,
) -> None:
    """Applique la mise en forme chronologique à l'axe X.

    Args:
        fig: Figure Plotly.
        x_idx: Liste d'indices entiers.
        labels: Labels de dates formatés.
        step: Pas de tick.
        lang: Langue pour le titre d'axe.
        as_category: Ajouter ``type='category'`` (défaut True).
    """
    kwargs: dict = {
        "title_text": viz_t("axis_chronological", lang),
        "tickmode": "array",
        "tickvals": x_idx[::step],
        "ticktext": labels[::step],
        "tickangle": -45,
    }
    if as_category:
        kwargs["type"] = "category"
    fig.update_xaxes(**kwargs)
