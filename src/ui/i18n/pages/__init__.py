"""Sub-package i18n : chaînes spécifiques aux pages Streamlit.

Ce package regroupe les traductions par page, fusionnées dans un dict
global ``STRINGS`` pour compatibilité avec le registre ``_build_registry()``.
"""

from __future__ import annotations

from src.ui.i18n.pages import (
    career,
    explorer,
    last_match,
    match_view,
    media,
    objectives,
    session_compare,
    settings,
    shared,
    synthesis,
    teammates,
    timeseries,
    wl,
    xbox,
)

STRINGS: dict[str, dict[str, str] | str] = {}

# Fusion de tous les sous-modules (ordre stable)
for _mod in (
    career,
    explorer,
    last_match,
    match_view,
    media,
    objectives,
    session_compare,
    settings,
    shared,
    synthesis,
    teammates,
    timeseries,
    wl,
    xbox,
):
    STRINGS.update(_mod.STRINGS)

__all__ = ["STRINGS"]
