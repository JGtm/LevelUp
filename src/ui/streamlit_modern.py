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

            if get_script_run_ctx() is not None:
                return fragmented(*args, **kwargs)
        except (ImportError, Exception):
            pass
        return func(*args, **kwargs)

    return _wrapper  # type: ignore[return-value]


# ---------------------------------------------------------------------------
#  Plotly chart config par défaut
# ---------------------------------------------------------------------------

PLOTLY_CLEAN_CONFIG: dict[str, Any] = {
    "displayModeBar": False,
    "staticPlot": False,
}
# Config Plotly pour masquer la barre d'outils en gardant le zoom/pan.


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
