"""Smoke tests du routeur de pages Streamlit.

Ces tests valident la configuration de page_router :
- Ordre des pages dans PAGE_KEYS
- Paramètres de la barre de navigation
- Cohérence des mappings internes
"""

from __future__ import annotations


class TestPageOrder:
    """Vérifie l'ordre et la position des pages dans PAGE_KEYS."""

    def test_teammates_is_third(self) -> None:
        """teammates doit être en 3ème position (index 2)."""
        from src.app.page_router import PAGE_KEYS

        assert PAGE_KEYS.index("teammates") == 2, (
            f"teammates attendu à l'index 2, trouvé à l'index {PAGE_KEYS.index('teammates')}"
        )

    def test_timeseries_is_first(self) -> None:
        """timeseries doit être en première position."""
        from src.app.page_router import PAGE_KEYS

        assert PAGE_KEYS[0] == "timeseries"

    def test_session_compare_is_second(self) -> None:
        """session_compare doit être en deuxième position."""
        from src.app.page_router import PAGE_KEYS

        assert PAGE_KEYS[1] == "session_compare"

    def test_settings_is_last(self) -> None:
        """settings doit être en dernière position."""
        from src.app.page_router import PAGE_KEYS

        assert PAGE_KEYS[-1] == "settings"

    def test_all_slugs_have_url_path(self) -> None:
        """Chaque slug de PAGE_KEYS doit avoir un url_path défini."""
        from src.app.page_router import _PAGE_URL_PATHS, PAGE_KEYS

        for slug in PAGE_KEYS:
            assert slug in _PAGE_URL_PATHS, f"{slug} manquant dans _PAGE_URL_PATHS"

    def test_all_slugs_have_icon(self) -> None:
        """Chaque slug de PAGE_KEYS doit avoir une icône définie."""
        from src.app.page_router import _PAGE_ICONS, PAGE_KEYS

        for slug in PAGE_KEYS:
            assert slug in _PAGE_ICONS, f"{slug} manquant dans _PAGE_ICONS"

    def test_all_slugs_have_i18n_key(self) -> None:
        """Chaque slug de PAGE_KEYS doit avoir une clé i18n."""
        from src.app.page_router import _PAGE_I18N_KEYS, PAGE_KEYS

        for slug in PAGE_KEYS:
            assert slug in _PAGE_I18N_KEYS, f"{slug} manquant dans _PAGE_I18N_KEYS"


class TestNavBarConfig:
    """Vérifie la configuration de la barre de navigation."""

    def test_render_page_selector_nav_uses_stretch_width(self) -> None:
        """render_page_selector_nav doit appeler st.segmented_control avec width='stretch'."""
        import inspect

        from src.app import page_router as mod

        src = inspect.getsource(mod.render_page_selector_nav)
        assert 'width="stretch"' in src or "width='stretch'" in src, (
            "render_page_selector_nav doit passer width='stretch' à st.segmented_control"
        )

    def test_render_page_selector_nav_uses_label_visibility_collapsed(self) -> None:
        """Le label du segmented_control doit être caché."""
        import inspect

        from src.app import page_router as mod

        src = inspect.getsource(mod.render_page_selector_nav)
        assert 'label_visibility="collapsed"' in src or "label_visibility='collapsed'" in src
