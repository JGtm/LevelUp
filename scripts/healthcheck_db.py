#!/usr/bin/env python3
"""CLI du healthcheck des bases DuckDB LevelUp.

Vérifie l'état de santé des DB et vues SQL v6 après déploiement.

Usage:
    python scripts/healthcheck_db.py              # Check rapide
    python scripts/healthcheck_db.py --verbose     # Affiche aussi les checks OK
    python scripts/healthcheck_db.py --deep        # + intégrité référentielle, doublons
    python scripts/healthcheck_db.py --player GT   # Un joueur spécifique
    python scripts/healthcheck_db.py --no-repair   # Pas d'auto-repair des vues
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(REPO_ROOT))

# Forcer l'encodage UTF-8 sur Windows pour les emojis
if sys.platform == "win32":
    import io

    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding="utf-8", errors="replace")
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding="utf-8", errors="replace")


def main() -> None:
    """Point d'entrée CLI."""
    parser = argparse.ArgumentParser(description="LevelUp — DB Healthcheck")
    parser.add_argument("--deep", action="store_true", help="Checks profonds (intégrité, doublons)")
    parser.add_argument(
        "--verbose", "-v", action="store_true", help="Affiche tous les checks (y compris OK)"
    )
    parser.add_argument("--player", type=str, help="Vérifier un joueur spécifique uniquement")
    parser.add_argument(
        "--no-repair", action="store_true", help="Désactiver l'auto-repair des vues"
    )
    parser.add_argument("--json", action="store_true", help="Sortie JSON structurée")
    args = parser.parse_args()

    from src.utils.healthcheck_db import format_results, run_healthcheck

    results = run_healthcheck(
        deep=args.deep,
        auto_repair=not args.no_repair,
        player=args.player,
    )

    if args.json:
        import json

        output = []
        for r in results:
            output.append(
                {
                    "db_name": r.db_name,
                    "db_path": str(r.db_path) if r.db_path else None,
                    "status": r.status,
                    "size_bytes": r.size_bytes,
                    "checks": [
                        {
                            "category": c.category,
                            "name": c.name,
                            "status": c.status,
                            "message": c.message,
                        }
                        for c in r.checks
                    ],
                }
            )
        print(json.dumps(output, indent=2, ensure_ascii=False))
    else:
        print(format_results(results, verbose=args.verbose))

    # Exit code basé sur les erreurs
    has_errors = any(r.status == "error" for r in results)
    sys.exit(1 if has_errors else 0)


if __name__ == "__main__":
    main()
