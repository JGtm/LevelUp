"""diff_openapi.py — Compare les routes FastAPI vs Go et produit un rapport de parité (Sprint 29).

Utilisation :
  # Compare le YAML Go avec les routes de l'API FastAPI en live :
  python apps/go-api/scripts/diff_openapi.py --fastapi-url http://localhost:8000/api/openapi.json

  # Compare deux fichiers YAML :
  python apps/go-api/scripts/diff_openapi.py --fastapi-yaml apps/go-api/api/openapi_fastapi_reference.yaml

  # Sortie JSON :
  python apps/go-api/scripts/diff_openapi.py --fastapi-url http://localhost:8000/api/openapi.json --json

Codes de sortie :
  0 — aucune divergence
  1 — routes manquantes côté Go ou méthodes incompatibles détectées
  2 — erreur de lecture de l'un des fichiers
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

try:
    import urllib.request

    import yaml  # PyYAML
except ImportError:
    print("Installer pyyaml : pip install pyyaml", file=sys.stderr)
    sys.exit(2)

_GO_OPENAPI = Path(__file__).parent.parent / "api" / "openapi.yaml"


def _load_yaml(path: Path) -> dict[str, Any]:
    with open(path, encoding="utf-8") as f:
        return yaml.safe_load(f)


def _load_url(url: str) -> dict[str, Any]:
    """Télécharge le JSON OpenAPI depuis une URL."""
    with urllib.request.urlopen(url, timeout=10) as resp:  # noqa: S310
        return json.loads(resp.read())


def _extract_routes(spec: dict[str, Any]) -> dict[str, set[str]]:
    """Extrait {path → {method, ...}} depuis un schéma OpenAPI (v3.x)."""
    result: dict[str, set[str]] = {}
    http_methods = {"get", "post", "put", "patch", "delete", "head", "options"}
    for path, path_item in spec.get("paths", {}).items():
        methods = {m for m in path_item if m in http_methods}
        if methods:
            result[path] = methods
    return result


def _normalize_path(path: str) -> str:
    """Normalise les segments de path paramétré pour comparaison : {player_slug} = {player}."""
    import re

    return re.sub(r"\{[^}]+\}", "{*}", path)


def _compare(
    fastapi_routes: dict[str, set[str]],
    go_routes: dict[str, set[str]],
) -> tuple[list[dict], list[dict], list[dict]]:
    """Retourne (missing_in_go, extra_in_go, method_mismatches)."""
    # Construire des dicts normalisés → chemin original
    fa_norm = {_normalize_path(p): (p, ms) for p, ms in fastapi_routes.items()}
    go_norm = {_normalize_path(p): (p, ms) for p, ms in go_routes.items()}

    missing: list[dict] = []
    extra: list[dict] = []
    mismatches: list[dict] = []

    for norm, (fa_path, fa_methods) in fa_norm.items():
        if norm not in go_norm:
            missing.append({"path": fa_path, "methods": sorted(fa_methods)})
        else:
            go_path, go_methods = go_norm[norm]
            if fa_methods != go_methods:
                mismatches.append(
                    {
                        "fastapi_path": fa_path,
                        "go_path": go_path,
                        "fastapi_methods": sorted(fa_methods),
                        "go_methods": sorted(go_methods),
                        "missing_methods": sorted(fa_methods - go_methods),
                        "extra_methods": sorted(go_methods - fa_methods),
                    }
                )

    for norm, (go_path, go_methods) in go_norm.items():
        if norm not in fa_norm:
            extra.append({"path": go_path, "methods": sorted(go_methods)})

    return missing, extra, mismatches


def _print_report(
    missing: list[dict],
    extra: list[dict],
    mismatches: list[dict],
    as_json: bool,
) -> int:
    report = {
        "summary": {
            "missing_in_go": len(missing),
            "extra_in_go": len(extra),
            "method_mismatches": len(mismatches),
            "status": "OK" if not (missing or mismatches) else "DIVERGENCES",
        },
        "missing_in_go": missing,
        "extra_in_go": extra,
        "method_mismatches": mismatches,
    }
    if as_json:
        print(json.dumps(report, indent=2, ensure_ascii=False))
    else:
        s = report["summary"]
        print(
            f"\n=== Rapport de parité OpenAPI ===\n"
            f"  Statut          : {s['status']}\n"
            f"  Manquants en Go : {s['missing_in_go']}\n"
            f"  Extras en Go    : {s['extra_in_go']}\n"
            f"  Méthodes wrong  : {s['method_mismatches']}\n"
        )
        if missing:
            print("Routes absentes côté Go (à implémenter) :")
            for r in missing:
                print(f"  {', '.join(m.upper() for m in r['methods'])} {r['path']}")
        if extra:
            print("\nRoutes Go sans équivalent FastAPI (vérifier si intentionnel) :")
            for r in extra:
                print(f"  {', '.join(m.upper() for m in r['methods'])} {r['path']}")
        if mismatches:
            print("\nDivergences de méthodes HTTP :")
            for m in mismatches:
                print(f"  {m['fastapi_path']}")
                if m["missing_methods"]:
                    print(f"    → manquantes Go : {m['missing_methods']}")
                if m["extra_methods"]:
                    print(f"    → extras Go     : {m['extra_methods']}")

    return 1 if (missing or mismatches) else 0


def main() -> None:
    parser = argparse.ArgumentParser(description="Compare les routes FastAPI vs Go (sprint 29).")
    src = parser.add_mutually_exclusive_group(required=True)
    src.add_argument("--fastapi-url", metavar="URL", help="URL du /api/openapi.json FastAPI")
    src.add_argument("--fastapi-yaml", metavar="FILE", help="Fichier YAML OpenAPI FastAPI local")
    parser.add_argument("--go-yaml", default=str(_GO_OPENAPI), metavar="FILE", help="Fichier YAML OpenAPI Go (défaut: apps/go-api/api/openapi.yaml)")
    parser.add_argument("--json", dest="as_json", action="store_true", help="Sortie JSON")
    parser.add_argument(
        "--save-report",
        metavar="FILE",
        help="Enregistrer le rapport JSON dans ce fichier",
    )
    args = parser.parse_args()

    # Charger l'OpenAPI FastAPI
    try:
        if args.fastapi_url:
            fastapi_spec = _load_url(args.fastapi_url)
        else:
            fastapi_spec = _load_yaml(Path(args.fastapi_yaml))
    except Exception as exc:
        print(f"Erreur chargement FastAPI spec : {exc}", file=sys.stderr)
        sys.exit(2)

    # Charger l'OpenAPI Go
    try:
        go_spec = _load_yaml(Path(args.go_yaml))
    except Exception as exc:
        print(f"Erreur chargement Go spec : {exc}", file=sys.stderr)
        sys.exit(2)

    fa_routes = _extract_routes(fastapi_spec)
    go_routes = _extract_routes(go_spec)

    missing, extra, mismatches = _compare(fa_routes, go_routes)

    if args.save_report:
        report = {
            "summary": {
                "missing_in_go": len(missing),
                "extra_in_go": len(extra),
                "method_mismatches": len(mismatches),
                "status": "OK" if not (missing or mismatches) else "DIVERGENCES",
            },
            "missing_in_go": missing,
            "extra_in_go": extra,
            "method_mismatches": mismatches,
        }
        Path(args.save_report).write_text(json.dumps(report, indent=2, ensure_ascii=False))
        print(f"Rapport enregistré : {args.save_report}")

    exit_code = _print_report(missing, extra, mismatches, as_json=args.as_json)
    sys.exit(exit_code)


if __name__ == "__main__":
    main()
