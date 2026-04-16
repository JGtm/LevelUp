"""export_fastapi_openapi.py — Exporte le schéma OpenAPI FastAPI vers un fichier YAML de référence (Sprint 29).

Ce fichier YAML devient la **source de vérité contractuelle** pour comparer
avec l'OpenAPI Go (`apps/go-api/api/openapi.yaml`).

Utilisation :
  python apps/go-api/scripts/export_fastapi_openapi.py \\
      --out apps/go-api/api/openapi_fastapi_reference.yaml

Prérequis : FastAPI installé (pip install -e ".[dev]" depuis la racine).
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("Installer pyyaml : pip install pyyaml", file=sys.stderr)
    sys.exit(2)


def main() -> None:
    parser = argparse.ArgumentParser(description="Exporte l'OpenAPI FastAPI en YAML (sprint 29).")
    parser.add_argument(
        "--out",
        default="apps/go-api/api/openapi_fastapi_reference.yaml",
        metavar="FILE",
        help="Fichier de sortie YAML",
    )
    args = parser.parse_args()

    # Import de l'app FastAPI (requiert le virtualenv Python)
    try:
        # Ajouter la racine au path si besoin
        root = Path(__file__).parents[3]
        if str(root) not in sys.path:
            sys.path.insert(0, str(root))
        from apps.api.app.main import create_app

        import os
        os.environ.setdefault("LEVELUP_DEMO_MODE", "true")
        app = create_app()
        schema = app.openapi()
    except ImportError as exc:
        print(
            f"Impossible d'importer l'app FastAPI : {exc}\n"
            "Assurez-vous d'être dans le venv Python avec `pip install -e .[dev]`.",
            file=sys.stderr,
        )
        sys.exit(2)

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    with open(out_path, "w", encoding="utf-8") as f:
        yaml.dump(
            json.loads(json.dumps(schema)),  # convertit les types python en types yaml standards
            f,
            default_flow_style=False,
            allow_unicode=True,
            sort_keys=True,
        )

    print(f"OpenAPI FastAPI exporté → {out_path}")

    # Afficher un résumé des routes
    paths = schema.get("paths", {})
    print(f"  {len(paths)} routes trouvées :")
    for path in sorted(paths):
        methods = [m.upper() for m in paths[path] if m in ("get", "post", "put", "patch", "delete")]
        print(f"    {', '.join(methods):<20} {path}")


if __name__ == "__main__":
    main()
