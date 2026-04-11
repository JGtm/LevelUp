"""Utilitaires de compatibilité Streamlit moderne.

Fournit des wrappers graceful-degradation pour les fonctionnalités
Streamlit récentes (`@st.fragment`, `st.navigation`, etc.) afin de
rester compatible avec les versions plus anciennes.

Sprint 8ter — Modernisation Streamlit.
"""

from __future__ import annotations

import functools
from collections.abc import Callable
from typing import Any, TypeVar

import streamlit as st

from src.config import THEME_COLORS_V7

F = TypeVar("F", bound=Callable[..., Any])

# ---------------------------------------------------------------------------
#  Version helpers
# ---------------------------------------------------------------------------

_ST_VERSION: tuple[int, ...] = tuple(int(x) for x in st.__version__.split(".")[:3] if x.isdigit())

HAS_FRAGMENT: bool = _ST_VERSION >= (1, 33, 0)
# True si ``@st.fragment`` est disponible (Streamlit >= 1.33).

HAS_NAVIGATION: bool = _ST_VERSION >= (1, 36, 0)
# True si ``st.navigation`` est disponible (Streamlit >= 1.36).


# ---------------------------------------------------------------------------
#  @fragment_if_available — Décorateur graceful degradation
# ---------------------------------------------------------------------------


def fragment_if_available(func: F) -> F:
    """Applique ``@st.fragment`` si disponible **et** si un runtime Streamlit est actif.

    En mode test (pas de ScriptRunContext), retourne la fonction telle
    quelle pour éviter que le wrapper fragment n'avale l'exécution.

    Exemple
    -------
    >>> @fragment_if_available
    ... def render_timeseries_page(dff, **kwargs):
    ...     ...
    """
    if not HAS_FRAGMENT:
        return func

    fragmented = st.fragment(func)

    @functools.wraps(func)
    def _wrapper(*args: object, **kwargs: object) -> object:
        try:
            from streamlit.runtime.scriptrunner import get_script_run_ctx

            ctx = get_script_run_ctx()
        except ImportError:
            ctx = None
        if ctx is not None:
            return fragmented(*args, **kwargs)
        return func(*args, **kwargs)

    return _wrapper  # type: ignore[return-value]


# ---------------------------------------------------------------------------
#  Plotly chart config par défaut
# ---------------------------------------------------------------------------

PLOTLY_CLEAN_CONFIG: dict[str, Any] = {
    "displayModeBar": False,
    "staticPlot": False,
    "displaylogo": False,
}
# Config Plotly pour masquer la barre d'outils en gardant le zoom/pan.

PLOTLY_V7_CONFIG: dict[str, Any] = {
    **PLOTLY_CLEAN_CONFIG,
    "responsive": True,
    # Le rendu visuel reste pilote par le layout des figures.
    # Cette config V7 fixe surtout le comportement de la toolbar.
    "toImageButtonOptions": {
        "format": "png",
        "filename": "levelup-v7-chart",
        "scale": 2,
        "height": 720,
        "width": 1280,
    },
}


def get_v7_plotly_layout() -> dict[str, Any]:
    """Retourne un layout Plotly léger aligné sur la palette v7.

    Cette fonction ne change pas les couleurs de traces existantes. Elle fixe
    uniquement les fonds, la typographie et les contrastes de base.
    """
    return {
        "paper_bgcolor": THEME_COLORS_V7.bg_plot_rgba(1.0),
        "plot_bgcolor": THEME_COLORS_V7.bg_plot_rgba(1.0),
        "font": {
            "color": THEME_COLORS_V7.text_primary,
            "size": 13,
            "family": "IBM Plex Sans, Segoe UI, sans-serif",
        },
        "hoverlabel": {
            "bgcolor": THEME_COLORS_V7.bg_surface_alt_css(),
            "bordercolor": THEME_COLORS_V7.border,
            "font": {
                "color": THEME_COLORS_V7.text_primary,
                "family": "IBM Plex Mono, Cascadia Code, monospace",
            },
        },
    }


PLOTLY_STATIC_CONFIG: dict[str, Any] = {
    "staticPlot": True,
}
# Config Plotly pour les graphiques en lecture seule (barres, camemberts,
# heatmaps, jauges, radars).  Désactive hover, zoom et toolbar.


def plotly_chart(
    fig: Any,
    *,
    width: str = "stretch",
    config: dict[str, Any] | None = None,
    key: str | None = None,
    **kwargs: Any,
) -> None:
    """Wrapper autour de ``st.plotly_chart`` avec config propre par défaut.

    Applique automatiquement ``PLOTLY_CLEAN_CONFIG`` si aucun ``config``
    n'est fourni explicitement.
    """
    effective_config = config if config is not None else PLOTLY_CLEAN_CONFIG
    st.plotly_chart(fig, width=width, config=effective_config, key=key, **kwargs)
