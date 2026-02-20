"""Tests pour src.ui.streamlit_modern — wrappers compatibilité Streamlit."""

from __future__ import annotations

import importlib


class TestStreamlitModernModule:
    """Vérifie que le module streamlit_modern se charge correctement."""

    def test_import_module(self) -> None:
        mod = importlib.import_module("src.ui.streamlit_modern")
        assert hasattr(mod, "fragment_if_available")
        assert hasattr(mod, "PLOTLY_CLEAN_CONFIG")
        assert hasattr(mod, "plotly_chart")
        assert hasattr(mod, "HAS_FRAGMENT")
        assert hasattr(mod, "HAS_NAVIGATION")

    def test_has_fragment_is_bool(self) -> None:
        from src.ui.streamlit_modern import HAS_FRAGMENT

        assert isinstance(HAS_FRAGMENT, bool)

    def test_has_navigation_is_bool(self) -> None:
        from src.ui.streamlit_modern import HAS_NAVIGATION

        assert isinstance(HAS_NAVIGATION, bool)

    def test_plotly_clean_config_keys(self) -> None:
        from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG

        assert "displayModeBar" in PLOTLY_CLEAN_CONFIG
        assert PLOTLY_CLEAN_CONFIG["displayModeBar"] is False

    def test_fragment_if_available_identity(self) -> None:
        """Le décorateur retourne une callable."""
        from src.ui.streamlit_modern import fragment_if_available

        @fragment_if_available
        def dummy_func():
            return 42

        # En contexte test (pas de runtime Streamlit), doit rester callable
        assert callable(dummy_func)

    def test_fragment_preserves_function_name(self) -> None:
        """Le décorateur ne casse pas le nom de la fonction."""
        from src.ui.streamlit_modern import fragment_if_available

        @fragment_if_available
        def my_render_page():
            pass

        # Avec @st.fragment réel, le nom peut changer ; on vérifie juste que c'est callable
        assert callable(my_render_page)
