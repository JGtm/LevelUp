"""Vérifie les limites de taille des fonctions et modules.

Mode ratchet : compare les violations actuelles au baseline enregistré.
Échoue uniquement si de NOUVELLES violations apparaissent (pas dans le baseline).

Usage :
    python scripts/enforce_size_limits.py              # vérifier vs baseline
    python scripts/enforce_size_limits.py --update     # régénérer le baseline
    python scripts/enforce_size_limits.py --list       # lister toutes les violations sans check
"""

from __future__ import annotations

import ast
import sys
from pathlib import Path

MAX_FUNCTION_LINES = 80
MAX_MODULE_LINES = 500

# Chemins exclus du check (dictionnaires de traduction, migrations séquentielles, archives)
WHITELIST_PATHS: tuple[str, ...] = (
    "src/ui/i18n/",
    "src/utils/launcher_i18n.py",
    "src/data/sync/migrations.py",
    "src/data/sync/migrations_player.py",  # DDL séquentiel — exempté taille (H3)
    "src/data/sync/migrations_shared.py",  # DDL séquentiel — exempté taille (H3)
    "scripts/migration/",
    "scripts/_archive/",
)

BASELINE_FILE = Path("scripts/size_baseline.txt")


def _is_whitelisted(path: Path) -> bool:
    path_str = path.as_posix()
    return any(w in path_str for w in WHITELIST_PATHS)


def _scan_violations(root: Path = Path("src")) -> list[str]:
    """Retourne une liste triée de toutes les violations actuelles."""
    violations: list[str] = []

    for path in sorted(root.rglob("*.py")):
        if _is_whitelisted(path):
            continue

        try:
            source = path.read_text(encoding="utf-8")
        except OSError:
            continue

        # Vérifier la taille du module
        module_lines = source.count("\n") + 1
        if module_lines > MAX_MODULE_LINES:
            violations.append(f"MODULE:{path.as_posix()}:{module_lines}L")

        # Vérifier la taille des fonctions
        try:
            tree = ast.parse(source)
        except SyntaxError:
            continue

        for node in ast.walk(tree):
            if isinstance(node, ast.FunctionDef | ast.AsyncFunctionDef):
                end = getattr(node, "end_lineno", None)
                if end is None:
                    continue
                fn_lines = end - node.lineno
                if fn_lines > MAX_FUNCTION_LINES:
                    violations.append(
                        f"FUNC:{path.as_posix()}:{node.lineno}:{node.name}:{fn_lines}L"
                    )

    return sorted(violations)


def _load_baseline() -> set[str]:
    if not BASELINE_FILE.exists():
        return set()
    lines = BASELINE_FILE.read_text(encoding="utf-8").splitlines()
    return {line for line in lines if line.strip() and not line.startswith("#")}


def _save_baseline(violations: list[str]) -> None:
    BASELINE_FILE.parent.mkdir(parents=True, exist_ok=True)
    header = (
        "# Baseline des violations de taille — généré automatiquement\n"
        "# Chaque ligne = une violation connue (dette documentée)\n"
        "# Réduire ce fichier = réduire la dette technique\n"
        "# Format FUNC: FUNC:<fichier>:<ligne>:<nom>:<taille>L\n"
        "# Format MODULE: MODULE:<fichier>:<taille>L\n"
        "#\n"
    )
    content = header + "\n".join(violations) + ("\n" if violations else "")
    with open(BASELINE_FILE, "w", encoding="utf-8", newline="\n") as f:
        f.write(content)


def main() -> int:
    update_mode = "--update" in sys.argv
    list_mode = "--list" in sys.argv

    current = _scan_violations()

    if list_mode:
        modules = [v for v in current if v.startswith("MODULE:")]
        funcs = [v for v in current if v.startswith("FUNC:")]
        print(f"\nViolations actuelles : {len(current)} total")
        print(f"  Modules  > {MAX_MODULE_LINES}L : {len(modules)}")
        print(f"  Fonctions > {MAX_FUNCTION_LINES}L : {len(funcs)}\n")
        for v in modules:
            print(f"  {v}")
        for v in funcs:
            print(f"  {v}")
        return 0

    if update_mode:
        _save_baseline(current)
        modules = sum(1 for v in current if v.startswith("MODULE:"))
        funcs = sum(1 for v in current if v.startswith("FUNC:"))
        print(
            f"[OK] Baseline mis a jour : {len(current)} violation(s) enregistree(s)\n"
            f"     Modules   > {MAX_MODULE_LINES}L : {modules}\n"
            f"     Fonctions > {MAX_FUNCTION_LINES}L : {funcs}\n"
            f"     -> {BASELINE_FILE}"
        )
        return 0

    baseline = _load_baseline()
    new_violations = [v for v in current if v not in baseline]
    fixed_count = sum(1 for v in baseline if v not in current)
    known_count = len(current) - len(new_violations)

    if fixed_count:
        print(
            f"[OK] {fixed_count} violation(s) corrigee(s) depuis le baseline.\n"
            f"     Pensez a relancer --update pour resserrer le ratchet."
        )

    if known_count:
        print(
            f"[INFO] Violations connues (baseline) : {known_count} -- dette en cours de reduction"
        )

    if new_violations:
        print(f"\n[FAIL] {len(new_violations)} NOUVELLE(S) violation(s) detectee(s) :\n")
        for v in new_violations:
            print(f"  {v}")
        print(
            "\nOptions :\n"
            "  1. Corriger la violation (split de la fonction/module)\n"
            "  2. Si dette intentionnelle : python scripts/enforce_size_limits.py --update\n"
            f"     Limites : fonctions <= {MAX_FUNCTION_LINES}L, modules <= {MAX_MODULE_LINES}L"
        )
        return 1

    if not fixed_count and not known_count:
        print("[OK] Aucune violation -- toutes les fonctions et modules respectent les limites.")

    return 0


if __name__ == "__main__":
    sys.exit(main())
