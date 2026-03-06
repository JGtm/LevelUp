"""Tests de qualité structurelle — taille, complexité, SRP.

Ces tests font partie de la suite stable (python -m pytest -q --ignore=tests/integration).
Ils échouent uniquement si de NOUVELLES violations apparaissent (mode ratchet).

Trois règles enforced :
1. Taille des fonctions (> 80L) et modules (> 500L) via check_code_size.py
2. Propreté Ruff (C901 complexité, imports, style)
3. SRP : flag les noms de fonctions contenant '_and_' hors exceptions validées
"""

from __future__ import annotations

import ast
import subprocess
import sys
from pathlib import Path

import pytest


@pytest.mark.quality
def test_no_new_size_violations() -> None:
    """Aucune nouvelle fonction/module ne dépasse les limites de taille.

    Fonctionne en mode ratchet : les violations existantes (baseline) sont tolérées,
    seules les NOUVELLES violations font échouer le test.
    Mettre à jour le baseline après un refactoring : python scripts/check_code_size.py --update
    """
    result = subprocess.run(
        [sys.executable, "scripts/check_code_size.py"],
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, (
        f"Nouvelles violations de taille detectees :\n{result.stdout}\n"
        "Options :\n"
        "  1. Corriger la violation (split de la fonction/module)\n"
        "  2. Apres refactoring : python scripts/check_code_size.py --update"
    )


@pytest.mark.quality
def test_ruff_no_errors() -> None:
    """Ruff ne doit signaler aucune erreur dans src/.

    Couvre : complexité cyclomatique (C901), imports inutilisés (F401),
    bugs potentiels (B), style (E/W), simplifications (SIM/UP).
    """
    result = subprocess.run(
        [sys.executable, "-m", "ruff", "check", "src/", "--quiet"],
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, (
        f"Violations Ruff detectees dans src/ :\n{result.stdout}\n"
        "Corriger les violations ou ajouter '# noqa: <CODE>' avec justification."
    )


@pytest.mark.quality
def test_no_srp_violation_in_function_names() -> None:
    """Aucune fonction nommée avec '_and_' (indicateur SRP) dans src/.

    Une fonction dont le nom contient '_and_' fait probablement deux choses.
    Exemple : render_and_compute_X() -> compute_X() + render_X()

    Si le nom est légitime (domaine métier), l'ajouter à _SRP_EXCEPTIONS ci-dessous.
    """
    violations: list[str] = []

    for path in sorted(Path("src").rglob("*.py")):
        # Exclure les traductions (noms de clés i18n) et migrations
        if "i18n" in str(path) or "migration" in str(path):
            continue
        try:
            source = path.read_text(encoding="utf-8")
            tree = ast.parse(source)
        except (OSError, SyntaxError):
            continue

        for node in ast.walk(tree):
            if isinstance(node, ast.FunctionDef | ast.AsyncFunctionDef):
                name = node.name
                if "_and_" in name and name not in _SRP_EXCEPTIONS:
                    violations.append(f"{path}:{node.lineno} -- {name}()")

    assert not violations, (
        f"{len(violations)} fonction(s) avec responsabilites multiples suspectes :\n"
        + "\n".join(f"  {v}" for v in violations)
        + "\n\nSi le nom est un terme metier legitime, l'ajouter a _SRP_EXCEPTIONS "
        "dans tests/test_code_quality.py."
    )


# ---------------------------------------------------------------------------
# Exceptions SRP validées — fonctions existantes avec '_and_' acceptées
# ---------------------------------------------------------------------------
# Règle : toute NOUVELLE fonction contenant '_and_' fait échouer le test.
# Les exceptions ci-dessous sont des dettes documentées ou des termes de domaine.
# Pour supprimer une exception : refactoriser la fonction d'abord, retirer ensuite.
# ---------------------------------------------------------------------------
_SRP_EXCEPTIONS: frozenset[str] = frozenset(
    {
        # Termes de domaine métier (Halo / pipeline ETL)
        # "killer_and_victim",                # terme Halo — non utilisé pour l'instant
        "validate_and_adjust_pairs",  # killer_victim.py — validation + normalisation (pipeline atomique)
        "scan_and_index",  # media_indexer.py — scan filesystem + écriture DB (ETL atomique)
        # Dette technique existante (à refactoriser progressivement)
        "validate_and_fix_db_path",  # main_helpers.py — valide + corrige chemin au démarrage
        "compute_and_store_for_match",  # citations/engine.py — calcul + écriture citations (atomique)
        # _compute_and_update_performance_score — découpé en _compute + _update (Phase P5)
        "_render_map_and_rank",  # match_view.py — composant UI composite (carte + rang côte à côte)
    }
)
