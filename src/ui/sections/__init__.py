"""Sections UI (Streamlit) pour découper streamlit_app.py."""

from src.ui.sections.openspartan import render_openspartan_tools
from src.ui.sections.source import render_source_section

__all__ = ["render_source_section", "render_openspartan_tools"]
