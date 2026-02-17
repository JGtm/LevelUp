"""Tests de non-régression pour l'étape 8ter : Modernisation Streamlit.

Vérifie :
- PLOTLY_STATIC_CONFIG et sa propagation dans les pages
- @fragment_if_available déployé sur les bonnes pages
- build_navigation et page_router modernisé
- duckdb_read_only context manager
- Réduction st.rerun (checkbox_filter)
- html.escape dans les composants KPI
"""

from __future__ import annotations

import importlib
import re
from pathlib import Path

import pytest

SRC_ROOT = Path(__file__).resolve().parent.parent.parent / "src"


# =====================================================================
# 1. PLOTLY_STATIC_CONFIG
# =====================================================================


class TestPlotlyStaticConfig:
    """Vérifie la config staticPlot sur les charts read-only."""

    def test_plotly_static_config_exists(self) -> None:
        from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG

        assert "staticPlot" in PLOTLY_STATIC_CONFIG
        assert PLOTLY_STATIC_CONFIG["staticPlot"] is True

    def test_plotly_clean_config_not_static(self) -> None:
        from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG

        assert PLOTLY_CLEAN_CONFIG.get("staticPlot") is False

    def test_static_plot_count_in_pages(self) -> None:
        """Au moins 25 usages de staticPlot: True dans src/ui/pages/."""
        pages_dir = SRC_ROOT / "ui" / "pages"
        count = 0
        for py_file in pages_dir.glob("*.py"):
            text = py_file.read_text(encoding="utf-8")
            count += text.count('"staticPlot": True')
            count += text.count("'staticPlot': True")
            count += text.count('"staticPlot":True')
        assert count >= 25, f"Seulement {count} staticPlot trouvés (attendu ≥25)"


# =====================================================================
# 2. @fragment_if_available
# =====================================================================


class TestFragmentDecorator:
    """Vérifie le déploiement de @fragment_if_available."""

    def test_fragment_decorator_count(self) -> None:
        """Au moins 20 décorateurs dans src/ui/pages/."""
        pages_dir = SRC_ROOT / "ui" / "pages"
        count = 0
        for py_file in pages_dir.glob("*.py"):
            text = py_file.read_text(encoding="utf-8")
            count += text.count("@fragment_if_available")
        assert count >= 20, f"Seulement {count} @fragment_if_available (attendu ≥20)"

    def test_fragment_on_timeseries_page(self) -> None:
        """timeseries.py doit avoir des fragments."""
        text = (SRC_ROOT / "ui" / "pages" / "timeseries.py").read_text(encoding="utf-8")
        assert "@fragment_if_available" in text

    def test_fragment_on_win_loss_page(self) -> None:
        """win_loss.py doit avoir des fragments."""
        text = (SRC_ROOT / "ui" / "pages" / "win_loss.py").read_text(encoding="utf-8")
        assert "@fragment_if_available" in text

    def test_fragment_on_match_view_charts(self) -> None:
        """match_view_charts.py doit avoir des fragments."""
        text = (SRC_ROOT / "ui" / "pages" / "match_view_charts.py").read_text(encoding="utf-8")
        assert "@fragment_if_available" in text

    def test_fragment_on_teammates_charts(self) -> None:
        """teammates_charts.py doit avoir des fragments."""
        text = (SRC_ROOT / "ui" / "pages" / "teammates_charts.py").read_text(encoding="utf-8")
        assert "@fragment_if_available" in text


# =====================================================================
# 3. st.navigation / page_router
# =====================================================================


class TestPageRouterModernisation:
    """Vérifie la modernisation du page_router."""

    def test_build_navigation_importable(self) -> None:
        from src.app.page_router import build_navigation

        assert callable(build_navigation)

    def test_render_page_selector_nav_importable(self) -> None:
        from src.app.page_router import render_page_selector_nav

        assert callable(render_page_selector_nav)

    def test_page_url_paths_defined(self) -> None:
        from src.app.page_router import _PAGE_URL_PATHS

        assert isinstance(_PAGE_URL_PATHS, dict)
        assert len(_PAGE_URL_PATHS) >= 10

    def test_page_icons_defined(self) -> None:
        from src.app.page_router import _PAGE_ICONS

        assert isinstance(_PAGE_ICONS, dict)
        assert len(_PAGE_ICONS) >= 10

    def test_has_navigation_flag(self) -> None:
        from src.ui.streamlit_modern import HAS_NAVIGATION

        assert isinstance(HAS_NAVIGATION, bool)

    def test_legacy_dispatch_page_still_exists(self) -> None:
        """dispatch_page doit exister pour le fallback legacy."""
        from src.app.page_router import dispatch_page

        assert callable(dispatch_page)


# =====================================================================
# 4. duckdb_read_only context manager
# =====================================================================


class TestDuckDBReadOnly:
    """Vérifie le context manager duckdb_read_only."""

    def test_import(self) -> None:
        from src.utils.db import duckdb_read_only

        assert callable(duckdb_read_only)

    def test_context_manager_with_temp_db(self) -> None:
        """Vérifie le pattern with sur une DB temporaire."""
        import os
        import tempfile

        import duckdb

        from src.utils.db import duckdb_read_only

        # Créer un fichier DuckDB dans un dossier temp
        tmp_dir = tempfile.mkdtemp()
        tmp_path = os.path.join(tmp_dir, "test.duckdb")

        try:
            # Créer une table
            conn = duckdb.connect(tmp_path)
            conn.execute("CREATE TABLE test_t (id INT, name VARCHAR)")
            conn.execute("INSERT INTO test_t VALUES (1, 'hello')")
            conn.close()

            # Lire via duckdb_read_only
            with duckdb_read_only(tmp_path) as ro_conn:
                result = ro_conn.execute("SELECT name FROM test_t WHERE id = 1").fetchone()
                assert result is not None
                assert result[0] == "hello"
        finally:
            os.unlink(tmp_path)
            os.rmdir(tmp_dir)

    def test_career_no_direct_duckdb_import(self) -> None:
        """career.py ne doit plus importer duckdb directement au top-level."""
        text = (SRC_ROOT / "ui" / "pages" / "career.py").read_text(encoding="utf-8")
        # Les imports duckdb directs doivent être remplacés par duckdb_read_only
        lines = text.split("\n")
        top_level_imports = [
            line.strip()
            for line in lines
            if line.strip().startswith("import duckdb") and not line.strip().startswith("#")
        ]
        # Pas d'import duckdb au top-level (module-scope)
        assert (
            len(top_level_imports) == 0
        ), f"career.py a encore un import duckdb top-level : {top_level_imports}"


# =====================================================================
# 5. Réduction st.rerun
# =====================================================================


class TestRerunReduction:
    """Vérifie la réduction des st.rerun() dans checkbox_filter."""

    def test_checkbox_filter_no_rerun(self) -> None:
        """checkbox_filter.py ne doit plus contenir st.rerun()."""
        text = (SRC_ROOT / "ui" / "components" / "checkbox_filter.py").read_text(encoding="utf-8")
        count = text.count("st.rerun()")
        assert count == 0, f"checkbox_filter.py contient encore {count} st.rerun()"

    def test_checkbox_filter_uses_on_click(self) -> None:
        """checkbox_filter.py doit utiliser on_click pour les boutons."""
        text = (SRC_ROOT / "ui" / "components" / "checkbox_filter.py").read_text(encoding="utf-8")
        assert "on_click=" in text

    def test_checkbox_filter_uses_on_change(self) -> None:
        """checkbox_filter.py doit utiliser on_change pour les checkboxes."""
        text = (SRC_ROOT / "ui" / "components" / "checkbox_filter.py").read_text(encoding="utf-8")
        assert "on_change=" in text

    def test_total_rerun_count_under_threshold(self) -> None:
        """Le nombre total de st.rerun() dans src/ doit être ≤ 15."""
        count = 0
        for py_file in SRC_ROOT.rglob("*.py"):
            text = py_file.read_text(encoding="utf-8")
            count += text.count("st.rerun()")
        assert count <= 15, f"Encore {count} st.rerun() dans src/ (limite: 15)"


# =====================================================================
# 6. html.escape dans les composants
# =====================================================================


class TestHtmlEscapeInComponents:
    """Vérifie que les composants HTML échappent les données."""

    def test_kpi_uses_html_escape(self) -> None:
        """kpi.py doit utiliser html.escape ou html_mod.escape."""
        text = (SRC_ROOT / "ui" / "components" / "kpi.py").read_text(encoding="utf-8")
        assert "html_mod.escape" in text or "html.escape" in text

    def test_performance_uses_html_escape(self) -> None:
        """performance.py doit utiliser html.escape."""
        text = (SRC_ROOT / "ui" / "components" / "performance.py").read_text(encoding="utf-8")
        assert "html_mod.escape" in text or "html.escape" in text

    def test_sidebar_no_unsafe_allow_html(self) -> None:
        """sidebar.py ne doit plus contenir unsafe_allow_html."""
        text = (SRC_ROOT / "app" / "sidebar.py").read_text(encoding="utf-8")
        assert "unsafe_allow_html" not in text


# =====================================================================
# 7. Post-sync pre-computation
# =====================================================================


class TestPostSyncCompute:
    """Vérifie le script de pré-calcul post-sync."""

    def test_script_importable(self) -> None:
        """post_sync_compute.py doit être importable."""
        mod = importlib.import_module("scripts.post_sync_compute")
        assert hasattr(mod, "post_sync_compute")
        assert callable(mod.post_sync_compute)

    def test_engine_hook_exists(self) -> None:
        """Le hook post_sync_compute doit être présent dans engine.py."""
        text = (SRC_ROOT / "data" / "sync" / "engine.py").read_text(encoding="utf-8")
        assert "post_sync_compute" in text


# =====================================================================
# 8. duckdb.connect direct — régression guard
# =====================================================================


class TestNoDuckDBConnectDirect:
    """Vérifie l'absence de duckdb.connect() direct dans les fichiers migrés."""

    _MIGRATED_FILES = [
        "ui/pages/career.py",
        "ui/aliases.py",
        "ui/cache_loaders.py",
        "app/data_loader.py",
    ]

    @pytest.mark.parametrize("relpath", _MIGRATED_FILES)
    def test_no_duckdb_connect_in_migrated(self, relpath: str) -> None:
        filepath = SRC_ROOT / relpath
        text = filepath.read_text(encoding="utf-8")
        # Chercher duckdb.connect mais pas dans les commentaires
        matches = re.findall(r"^\s*[^#]*duckdb\.connect\(", text, re.MULTILINE)
        assert len(matches) == 0, f"{relpath} contient encore duckdb.connect() direct : {matches}"
