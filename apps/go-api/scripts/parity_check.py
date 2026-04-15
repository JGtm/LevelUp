#!/usr/bin/env python3
"""Validation de parité Phase 1 — Sprint 7.

Compare les réponses du serveur Go contre les golden values capturées.
Génère un rapport 'parity_report.json' dans le répertoire courant.

Usage :
    # Démarrer le serveur Go d'abord :
    cd apps/go-api && make run

    # Puis lancer la vérification :
    python scripts/parity_check.py
    python scripts/parity_check.py --go-url http://localhost:8000 --player Chocoboflor
    python scripts/parity_check.py --only health bootstrap

Variables d'environnement :
    LEVELUP_GO_URL  : URL de base du serveur Go (défaut : http://localhost:8000)
    LEVELUP_PLAYER  : Slug du joueur (défaut : Chocoboflor)
"""

from __future__ import annotations

import argparse
import json
import math
import os
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

try:
    import httpx
except ImportError:
    print("Installer httpx : pip install httpx", file=sys.stderr)
    sys.exit(1)

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

GO_URL = os.environ.get("LEVELUP_GO_URL", "http://localhost:8000")
PLAYER = os.environ.get("LEVELUP_PLAYER", "Chocoboflor")
FIXTURES_DIR = Path(__file__).parent.parent / "tests" / "fixtures" / "golden_values"
REPORT_PATH = Path(__file__).parent.parent / "tests" / "fixtures" / "parity_report.json"
DEFAULT_FLOAT_TOL = 0.01

# ---------------------------------------------------------------------------
# Endpoints Phase 1
# ---------------------------------------------------------------------------

ENDPOINTS: list[dict[str, Any]] = [
    {
        "name": "health",
        "fixture": "health_ok.json",
        "method": "GET",
        "url": f"{GO_URL}/health",
        "ignore_fields": ["db_version"],  # version string may differ
    },
    {
        "name": "bootstrap",
        "fixture": "bootstrap_no_auth.json",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/bootstrap",
    },
    {
        "name": "players",
        "fixture": "players_list.json",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/players",
    },
    {
        "name": "filters_resolve",
        "fixture": "filters_resolve_all.json",
        "method": "POST",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/filters/resolve",
        "body": {},
    },
    {
        "name": "match_history",
        "fixture": "match_history_page1_nofilter.json",
        "method": "POST",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/match-history/query",
        "body": {"page": 1, "page_size": 25},
        "ignore_fields": ["_meta"],
    },
    {
        "name": "gamertag_search",
        "fixture": "gamertag_search_cho.json",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/directory/gamertags/search",
        "params": {"q": "Cho"},
    },
]

# ---------------------------------------------------------------------------
# Comparaison JSON
# ---------------------------------------------------------------------------


def _load_fixture(name: str) -> dict[str, Any] | None:
    path = FIXTURES_DIR / name
    if not path.exists():
        return None
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def _float_equal(a: float, b: float, tol: float) -> bool:
    if math.isnan(a) and math.isnan(b):
        return True
    if math.isnan(a) or math.isnan(b):
        return False
    if a == 0 and b == 0:
        return True
    return abs(a - b) <= tol * max(abs(a), abs(b), 1.0)


def _compare(  # noqa: C901, PLR0912, PLR0913
    golden: Any,
    got: Any,
    path: str,
    ignore_fields: set[str],
    float_tol: float,
    diffs: list[dict[str, Any]],
) -> None:
    """Récursion JSON — accumule les diffs dans `diffs`."""
    if isinstance(golden, dict):
        if not isinstance(got, dict):
            diffs.append(
                {"path": path, "expected": type(golden).__name__, "got": type(got).__name__}
            )
            return
        all_keys = set(golden) | set(got)
        for k in sorted(all_keys):
            if k in ignore_fields or k == "_meta":
                continue
            child_path = f"{path}.{k}" if path else k
            if k not in golden:
                diffs.append({"path": child_path, "note": "extra_key_in_go", "got_value": got[k]})
            elif k not in got:
                diffs.append(
                    {"path": child_path, "note": "missing_in_go", "expected_value": golden[k]}
                )
            else:
                _compare(golden[k], got[k], child_path, ignore_fields, float_tol, diffs)
    elif isinstance(golden, list):
        if not isinstance(got, list):
            diffs.append({"path": path, "expected": "list", "got": type(got).__name__})
            return
        if len(golden) != len(got):
            diffs.append(
                {"path": path, "note": "length_mismatch", "expected": len(golden), "got": len(got)}
            )
        for i, (g, r) in enumerate(zip(golden, got, strict=False)):
            _compare(g, r, f"{path}[{i}]", ignore_fields, float_tol, diffs)
    elif isinstance(golden, float) or isinstance(got, float):
        gv = float(golden) if golden is not None else None
        rv = float(got) if got is not None else None
        if gv is None and rv is None:
            return
        if gv is None or rv is None or not _float_equal(gv, rv, float_tol):
            diffs.append({"path": path, "expected": gv, "got": rv})
    elif golden != got:
        diffs.append({"path": path, "expected": golden, "got": got})


# ---------------------------------------------------------------------------
# Appels HTTP
# ---------------------------------------------------------------------------


def _call(client: httpx.Client, ep: dict[str, Any]) -> tuple[int, Any]:
    method = ep["method"]
    url = ep["url"]
    params = ep.get("params")
    body = ep.get("body")
    try:
        if method == "GET":
            resp = client.get(url, params=params, timeout=10)
        else:
            resp = client.post(url, json=body, timeout=10)
        try:
            return resp.status_code, resp.json()
        except Exception:
            return resp.status_code, resp.text
    except httpx.ConnectError:
        return -1, "connection_refused"
    except Exception as exc:
        return -1, str(exc)


# ---------------------------------------------------------------------------
# Logique principale
# ---------------------------------------------------------------------------


def run_parity_check(only: list[str] | None = None) -> dict[str, Any]:
    report: dict[str, Any] = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "go_url": GO_URL,
        "player": PLAYER,
        "results": [],
        "summary": {"total": 0, "passed": 0, "failed": 0, "skipped": 0},
    }

    endpoints = [ep for ep in ENDPOINTS if only is None or ep["name"] in only]

    with httpx.Client() as client:
        for ep in endpoints:
            name = ep["name"]
            fixture_data = _load_fixture(ep.get("fixture", ""))
            report["summary"]["total"] += 1

            if fixture_data is None:
                print(f"  [SKIP] {name} — fixture introuvable")
                report["summary"]["skipped"] += 1
                report["results"].append(
                    {"name": name, "status": "skipped", "reason": "fixture_missing"}
                )
                continue

            # Récupérer les tolérances depuis _meta du fixture
            meta = fixture_data.get("_meta", {})
            float_tol = meta.get("tolerances", {}).get("float_fields", DEFAULT_FLOAT_TOL)

            status_code, got = _call(client, ep)

            if status_code == -1:
                print(f"  [FAIL] {name} — {got}")
                report["summary"]["failed"] += 1
                report["results"].append({"name": name, "status": "failed", "error": str(got)})
                continue

            if status_code not in (200, 201):
                print(f"  [FAIL] {name} — HTTP {status_code}")
                report["summary"]["failed"] += 1
                report["results"].append(
                    {"name": name, "status": "failed", "http_status": status_code}
                )
                continue

            # Nettoyage du fixture (retirer _meta pour comparaison)
            golden_clean = {k: v for k, v in fixture_data.items() if k != "_meta"}
            ignore = set(ep.get("ignore_fields", []))

            diffs: list[dict[str, Any]] = []
            _compare(golden_clean, got, "", ignore, float_tol, diffs)

            if diffs:
                print(f"  [FAIL] {name} — {len(diffs)} écart(s)")
                report["summary"]["failed"] += 1
                report["results"].append(
                    {
                        "name": name,
                        "status": "failed",
                        "http_status": status_code,
                        "diffs": diffs,
                    }
                )
            else:
                print(f"  [PASS] {name}")
                report["summary"]["passed"] += 1
                report["results"].append(
                    {"name": name, "status": "passed", "http_status": status_code}
                )

    total = report["summary"]["total"]
    passed = report["summary"]["passed"]
    failed = report["summary"]["failed"]
    skipped = report["summary"]["skipped"]
    print(f"\nRésultat : {passed}/{total} OK, {failed} écart(s), {skipped} sauté(s)")

    return report


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


def main() -> int:
    parser = argparse.ArgumentParser(description="Vérification de parité Go vs golden values")
    parser.add_argument("--go-url", default=GO_URL, help="URL de base du serveur Go")
    parser.add_argument("--player", default=PLAYER, help="Slug du joueur")
    parser.add_argument("--only", nargs="+", metavar="NAME", help="Noms d'endpoints à tester")
    parser.add_argument("--report", default=str(REPORT_PATH), help="Chemin du rapport JSON")
    args = parser.parse_args()

    global GO_URL, PLAYER
    GO_URL = args.go_url
    PLAYER = args.player
    # Mettre à jour les URLs dans ENDPOINTS
    for ep in ENDPOINTS:
        ep["url"] = (
            ep["url"].replace("http://localhost:8000", GO_URL).replace("Chocoboflor", PLAYER)
        )

    print(f"Parité Phase 1 — {GO_URL} / joueur={PLAYER}")
    print("-" * 60)

    report = run_parity_check(only=args.only)

    report_path = Path(args.report)
    report_path.parent.mkdir(parents=True, exist_ok=True)
    with report_path.open("w", encoding="utf-8") as f:
        json.dump(report, f, indent=2, ensure_ascii=False)
    print(f"Rapport sauvegardé : {report_path}")

    return 0 if report["summary"]["failed"] == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
