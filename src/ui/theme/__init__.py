"""Chargement du thème visuel v7."""

from __future__ import annotations

from pathlib import Path


def load_v7_theme_css() -> str:
    """Charge la feuille de style v7.

    Returns:
        CSS encapsulé dans une balise style.
    """
    css_path = Path(__file__).with_name("v7_theme.css")
    try:
        css_content = css_path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return "<style></style>"
    return f"<style>\n{css_content}\n</style>"
