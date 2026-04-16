#!/usr/bin/env python3
"""Validation de parité Phase 1 → Sprint 36 — 24 endpoints.

Compare les réponses du serveur Go contre les golden values capturées.
Les endpoints sans golden values sont testés en mode `status_only` (HTTP 200).
Génère un rapport 'parity_report.json' dans le répertoire courant.

Usage :
    # Démarrer le serveur Go d'abord :
    cd apps/go-api && make run

    # Puis lancer la vérification :
    python scripts/parity_check.py
    python scripts/parity_check.py --go-url http://localhost:8000 --player Chocoboflor
    python scripts/parity_check.py --only health bootstrap
    python scripts/parity_check.py --status-only    # teste uniquement HTTP 200 partout

Variables d'environnement :
    LEVELUP_GO_URL  : URL de base du serveur Go (défaut : http://localhost:8000)
    LEVELUP_PLAYER  : Slug du joueur (défaut : Chocoboflor)
    LEVELUP_MATCH_ID : match_id Slayer pour match_view (facultatif)
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
MATCH_ID = os.environ.get("LEVELUP_MATCH_ID", "")
FIXTURES_DIR = Path(__file__).parent.parent / "tests" / "fixtures" / "golden_values"
REPORT_PATH = Path(__file__).parent.parent / "tests" / "fixtures" / "parity_report.json"
DEFAULT_FLOAT_TOL = 0.01

# ---------------------------------------------------------------------------
# Endpoints — 24 au total (Sprint 36)
#
# status_only=True  : vérifie uniquement HTTP 200 (pas de golden value)
# status_only=False : comparaison complète avec la fixture golden
# ---------------------------------------------------------------------------


ENDPOINTS: list[dict[str, Any]] = [
    # --- Endpoints avec golden values complètes (Phase 1) ---
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
    {
        "name": "career",
        "fixture": "career_page_chocoboflor.json",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/career",
        "ignore_fields": ["_meta"],
    },
    {
        "name": "match_view",
        "fixture": "match_view_slayer.json",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/matches/{MATCH_ID or '_PLACEHOLDER_'}",
        "ignore_fields": ["_meta"],
        "skip_if_placeholder": True,  # ignoré si LEVELUP_MATCH_ID non défini
    },

    # --- Endpoints status_only (Sprint 36 — golden values non encore capturées) ---
    {
        "name": "settings",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/settings",
        "status_only": True,
    },
    {
        "name": "career_top_matches",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/career/top-matches",
        "status_only": True,
    },
    {
        "name": "career_encounters",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/career/encounters",
        "status_only": True,
    },
    {
        "name": "sessions",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/sessions",
        "status_only": True,
    },
    {
        "name": "home",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/home",
        "status_only": True,
    },
    {
        "name": "battlepass",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/battlepass",
        "status_only": True,
    },
    {
        "name": "squad",
        "method": "GET",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/squad",
        "status_only": True,
    },
    {
        "name": "explorer_player",
        "method": "POST",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/explorer/player-query",
        "body": {"target_gamertag": PLAYER, "page": 1},
        "status_only": True,
    },
    {
        "name": "stats_query",
        "method": "POST",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/stats/query",
        "body": {},
        "status_only": True,
    },
    {
        "name": "synthesis",
        "method": "POST",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/synthesis",
        "body": {},
        "status_only": True,
    },
    {
        "name": "citations",
        "method": "POST",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/citations",
        "body": {},
        "status_only": True,
    },
    {
        "name": "commendations",
        "method": "POST",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/commendations",
        "body": {},
        "status_only": True,
    },
    {
        "name": "media",
        "method": "POST",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/media",
        "body": {},
        "status_only": True,
    },
    {
        "name": "teammates",
        "method": "POST",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/teammates",
        "body": {},
        "status_only": True,
    },
    {
        "name": "timeseries",
        "method": "POST",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/timeseries",
        "body": {},
        "status_only": True,
    },
    {
        "name": "last_match_resolve",
        "method": "POST",
        "url": f"{GO_URL}/api/v1/players/{PLAYER}/pages/last-match/resolve",
        "body": {},
        "status_only": True,
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


def run_parity_check(only: list[str] | None = None, force_status_only: bool = False) -> dict[str, Any]:
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
            report["summary"]["total"] += 1

            # skip_if_placeholder : ignorer si une valeur sentinelle est dans l'URL
            if ep.get("skip_if_placeholder") and "_PLACEHOLDER_" in ep.get("url", ""):
                print(f"  [SKIP] {name} — LEVELUP_MATCH_ID non défini")
                report["summary"]["skipped"] += 1
                report["results"].append(
                    {"name": name, "status": "skipped", "reason": "match_id_missing"}
                )
                continue

            # Mode status_only : vérifie uniquement HTTP 200, pas de diffing
            if ep.get("status_only") or force_status_only:
                status_code, _ = _call(client, ep)
                if status_code == -1:
                    print(f"  [FAIL] {name} — connexion refusée")
                    report["summary"]["failed"] += 1
                    report["results"].append({"name": name, "status": "failed", "error": "connection_refused"})
                elif status_code not in (200, 201):
                    print(f"  [FAIL] {name} — HTTP {status_code}")
                    report["summary"]["failed"] += 1
                    report["results"].append({"name": name, "status": "failed", "http_status": status_code})
                else:
                    print(f"  [PASS] {name} (status_only)")
                    report["summary"]["passed"] += 1
                    report["results"].append({"name": name, "status": "passed", "http_status": status_code, "mode": "status_only"})
                continue

            fixture_data = _load_fixture(ep.get("fixture", ""))

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
    parser.add_argument(
        "--status-only",
        action="store_true",
        default=False,
        help="Tester uniquement le HTTP 200 pour tous les endpoints (pas de diffing)",
    )
    parser.add_argument("--report", default=str(REPORT_PATH), help="Chemin du rapport JSON")
    args = parser.parse_args()

    global GO_URL, PLAYER, MATCH_ID
    GO_URL = args.go_url
    PLAYER = args.player
    MATCH_ID = os.environ.get("LEVELUP_MATCH_ID", "")
    # Mettre à jour les URLs dans ENDPOINTS
    for ep in ENDPOINTS:
        ep["url"] = (
            ep["url"]
            .replace("http://localhost:8000", GO_URL)
            .replace("Chocoboflor", PLAYER)
        )

    print(f"Parité Sprint 36 — {GO_URL} / joueur={PLAYER} / {len(ENDPOINTS)} endpoints")
    print("-" * 60)

    report = run_parity_check(only=args.only, force_status_only=args.status_only)

    report_path = Path(args.report)
    report_path.parent.mkdir(parents=True, exist_ok=True)
    with report_path.open("w", encoding="utf-8") as f:
        json.dump(report, f, indent=2, ensure_ascii=False)
    print(f"Rapport sauvegardé : {report_path}")

    return 0 if report["summary"]["failed"] == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
