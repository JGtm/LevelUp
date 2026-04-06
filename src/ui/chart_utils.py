"""Utilitaires de rendu de charts avec gestion d'erreurs centralisée.

Fournit un context manager ``safe_chart_render()`` qui remplace les
blocs ``try/except`` dispersés autour des ``st.plotly_chart()``.

Fournit également ``render_chart_or_info()`` qui remplace le pattern
``if fig is not None: st.plotly_chart(...) else: st.info(...)``.
"""

from __future__ import annotations

import contextlib
from typing import TYPE_CHECKING, Any

import streamlit as st

from src.ui.i18n import t

if TYPE_CHECKING:
    from collections.abc import Generator

    import plotly.graph_objects as go


@contextlib.contextmanager
def safe_chart_render(error_key: str = "error_chart") -> Generator[None, None, None]:
    """Context manager attrapant les erreurs de rendu de chart.

    Usage::

        from src.ui.chart_utils import safe_chart_render

        with safe_chart_render():
            fig = create_chart(data)
            st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)

        # Avec message d'erreur personnalisé
        with safe_chart_render("career_gauge_error"):
            fig = create_gauge(data)
            st.plotly_chart(fig, ...)

    Args:
        error_key: Clé de traduction pour le message d'erreur.
                   Par défaut ``"error_chart"`` (message générique).
    """
    try:
        yield
    except Exception as e:
        st.warning(t(error_key, error=e))


def render_chart_or_info(
    fig: go.Figure | None,
    *,
    key: str | None = None,
    config: dict[str, Any] | None = None,
    info_key: str = "no_data",
) -> None:
    """Rend une figure Plotly ou affiche un message si la figure est None.

    Remplace le pattern dispersé::

        if fig is not None:
            st.plotly_chart(fig, width="stretch", config=...)
        else:
            st.info(t("no_data"))

    Usage::

        from src.ui.chart_utils import render_chart_or_info
        from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG

        render_chart_or_info(fig, key="my_chart", config=PLOTLY_CLEAN_CONFIG)

    Args:
        fig: Figure Plotly ou None.
        key: Clé unique Streamlit pour le graphique (évite les collisions).
        config: Config Plotly. Défaut : ``{"displayModeBar": False}``.
        info_key: Clé i18n du message d'info affiché si fig est None.
    """
    if fig is not None:
        effective_config = config if config is not None else {"displayModeBar": False}
        with safe_chart_render():
            st.plotly_chart(fig, width="stretch", key=key, config=effective_config)
    else:
        st.info(t(info_key))
