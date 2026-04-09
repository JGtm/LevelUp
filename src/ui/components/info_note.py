"""Composant partagé — encadré informatif post-graphe (style ts-note)."""

from __future__ import annotations

import re

import streamlit as st

from src.ui.components.browser_storage import hints_visible


def render_info_note(text: str) -> None:
    """Affiche un encadré conclusif discret sous un graphe (thème Halo).

    N'affiche rien si hints_visible() est False.
    Interprète le markdown basique :
    - Listes à puces (lignes commençant par ``- ``)
    - **gras** converti en ``<strong>``

    Args:
        text: Texte de la note, peut contenir des lignes ``- item`` et ``**gras**``.
    """
    if not hints_visible():
        return
    lines = text.split("\n")
    parts: list[str] = []
    in_list = False
    for line in lines:
        if line.startswith("- "):
            if not in_list:
                parts.append("<ul>")
                in_list = True
            item = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", line[2:])
            parts.append(f"<li>{item}</li>")
        else:
            if in_list:
                parts.append("</ul>")
                in_list = False
            if line.strip():
                item = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", line)
                parts.append(f"<p>{item}</p>")
    if in_list:
        parts.append("</ul>")
    st.markdown(
        f'<div class="ts-note">{"".join(parts)}</div>',
        unsafe_allow_html=True,
    )
