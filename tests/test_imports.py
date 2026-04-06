"""Tests anti-régressions sur les imports Python.

Deux règles enforced :
1. Aucun __init__.py de package applicatif ne doit importer depuis ses propres sous-modules
   (anti-pattern "god __init__" qui cause des KeyError lors des hot-reloads Streamlit).
2. Les modules critiques doivent s'importer sans erreur en isolation (subprocess).
"""

from __future__ import annotations

import ast
import subprocess
import sys
from pathlib import Path

import pytest

ROOT = Path(__file__).parent.parent
SRC = ROOT / "src"

# Packages applicatifs dont les __init__.py ne doivent PAS importer leurs sous-modules.
#
# Règle : un __init__.py qui importe massivement ses sous-modules crée un graphe
# d'import fragile — lors des hot-reloads Streamlit, Python invalide partiellement
# sys.modules et laisse les parents dans un état incohérent (KeyError: 'src.xxx').
#
# Exclusions justifiées :
# - src/analysis et src/visualization : re-exports légitimes, de nombreux appelants
#   font `from src.analysis import X` (trop de callers pour refactorer).
# - src/ui, src/data : idem.
#
# À surveiller : si un nouveau package est créé SANS callers existants sur son __init__,
# l'ajouter ici pour prévenir l'anti-pattern dès le départ.
STRICT_PACKAGES = [
    "src/app",
]

# Modules à importer en isolation pour vérifier l'absence de circular import / KeyError.
CRITICAL_MODULES = [
    "src.utils",
    "src.utils.formatting",
    "src.ui.formatting",
    "src.app.data_loader",
    "src.app.state",
    "src.app.filters",
]


class _IntraPackageImportChecker(ast.NodeVisitor):
    """Détecte les imports d'un sous-module depuis le __init__ du même package."""

    def __init__(self, package_name: str) -> None:
        self.package_name = package_name  # e.g. "src.app"
        self.violations: list[tuple[int, str]] = []

    def visit_ImportFrom(self, node: ast.ImportFrom) -> None:  # noqa: N802
        if node.module and node.module.startswith(self.package_name + "."):
            self.violations.append((node.lineno, node.module))
        self.generic_visit(node)


@pytest.mark.parametrize("pkg_path", STRICT_PACKAGES)
def test_init_no_intra_package_imports(pkg_path: str) -> None:
    """Vérifie que le __init__.py d'un package n'importe pas ses propres sous-modules.

    Ce pattern cause des KeyError lors des hot-reloads Streamlit car Python invalide
    partiellement sys.modules, laissant les parents de packages dans un état incohérent.
    """
    init_file = ROOT / pkg_path / "__init__.py"
    if not init_file.exists():
        pytest.skip(f"Pas de __init__.py dans {pkg_path}")

    package_name = pkg_path.replace("/", ".")
    source = init_file.read_text(encoding="utf-8")
    tree = ast.parse(source, filename=str(init_file))

    checker = _IntraPackageImportChecker(package_name)
    checker.visit(tree)

    if checker.violations:
        details = "\n".join(
            f"  ligne {ln}: from {mod} import ..." for ln, mod in checker.violations
        )
        pytest.fail(
            f"{init_file.relative_to(ROOT)} importe ses propres sous-modules "
            f"(god __init__ → risque KeyError hot-reload) :\n{details}\n\n"
            "Fix : importer directement depuis le sous-module dans le code appelant."
        )


@pytest.mark.parametrize("module", CRITICAL_MODULES)
def test_module_imports_in_isolation(module: str) -> None:
    """Vérifie qu'un module critique peut être importé depuis un process vierge.

    Détecte les circular imports et les erreurs d'import qui ne surviennent
    qu'au premier chargement (avant que sys.modules soit peuplé).
    """
    result = subprocess.run(
        [sys.executable, "-c", f"import {module}"],
        capture_output=True,
        text=True,
        cwd=str(ROOT),
    )
    if result.returncode != 0:
        pytest.fail(f"Impossible d'importer `{module}` en isolation :\n{result.stderr.strip()}")


# Packages dont les __init__.py NE DOIVENT PAS être importés au niveau module
# dans streamlit_app.py (car ils chargent des dizaines de sous-modules, ce qui
# casse le hot-reload Streamlit quand un sous-module est en état partiel).
_PACKAGES_BANNED_AT_MODULE_LEVEL_IN_APP = [
    "src.ui.pages",  # 15 pages chargées en cascade → hot-reload fragile
]


def _get_module_level_imports_from_package(filepath: Path, package: str) -> list[tuple[int, str]]:
    """Retourne les imports du package au niveau module (hors corps de fonction/classe)."""
    source = filepath.read_text(encoding="utf-8")
    tree = ast.parse(source, filename=str(filepath))
    violations = []
    for node in ast.walk(tree):
        # On ne veut que les imports au niveau module (directement dans le Module)
        if not isinstance(node, ast.Module):
            continue
        for child in ast.iter_child_nodes(node):
            if isinstance(child, ast.ImportFrom):
                mod = child.module or ""
                # `from src.ui.pages import X` → charge src/ui/pages/__init__.py
                if mod == package or mod.startswith(package + ".") and mod == package:
                    names = ", ".join(alias.name for alias in child.names)
                    violations.append((child.lineno, f"from {mod} import {names}"))
    return violations


@pytest.mark.parametrize("package", _PACKAGES_BANNED_AT_MODULE_LEVEL_IN_APP)
def test_streamlit_app_no_toplevel_init_import(package: str) -> None:
    """Vérifie que streamlit_app.py n'importe pas le __init__ d'un package sensible
    au niveau module.

    Un tel import charge en cascade tous les sous-modules du package, rendant
    le graphe d'import fragile lors des hot-reloads Streamlit.
    Fix : remplacer `from src.ui.pages import X` par `from src.ui.pages.module import X`.
    """
    app_file = ROOT / "streamlit_app.py"
    violations = _get_module_level_imports_from_package(app_file, package)
    if violations:
        details = "\n".join(f"  ligne {ln}: {stmt}" for ln, stmt in violations)
        pytest.fail(
            f"streamlit_app.py charge le __init__ de `{package}` au niveau module :\n"
            f"{details}\n\n"
            "Fix : utiliser l'import direct (ex: `from src.ui.pages.match_view import render_match_view`)."
        )
