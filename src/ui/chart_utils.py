"""Utilitaires de rendu de charts avec gestion d'erreurs centralisée.

Fournit un context manager ``safe_chart_render()`` qui remplace les
blocs ``try/except`` dispersés autour des ``st.plotly_chart()``.
"""

from __future__ import annotations

import contextlib
from typing import TYPE_CHECKING

import streamlit as st

from src.ui.i18n import t

if TYPE_CHECKING:
    from collections.abc import Generator


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
