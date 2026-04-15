"""Benchmark de l'API Python LevelUp — Sprint 3.

Mesure les latences p50/p95/p99 de chaque endpoint pour constituer la baseline
de référence. Si Go > 2× le p95 Python sur un endpoint = bug de perf à investiguer.

Usage :
    python apps/go-api/scripts/benchmark_python_api.py
    python apps/go-api/scripts/benchmark_python_api.py --base-url http://localhost:8000 --player ref_player --samples 20
    python apps/go-api/scripts/benchmark_python_api.py --out apps/go-api/tests/fixtures/baselines.json
"""

from __future__ import annotations

import argparse
import json
import statistics
import sys
import time
from datetime import datetime, timezone
from pathlib import Path

import httpx

BASE_URL_DEFAULT = "http://127.0.0.1:8000"
SAMPLES_DEFAULT = 10
OUT_DEFAULT = Path(__file__).parent.parent / "tests" / "fixtures" / "baselines.json"


def _measure(client: httpx.Client, method: str, path: str, body: dict | None = None) -> list[float]:
    """Mesure N appels successifs et retourne les latences en ms."""
    latencies: list[float] = []
    for _ in range(SAMPLES_DEFAULT):
        t0 = time.perf_counter()
        r = client.get(path) if method == "GET" else client.post(path, json=body)
        t1 = time.perf_counter()
        # On accepte 200, 422 (pas de données) et 404 normaux
        if r.status_code not in (200, 201, 404, 422):
            print(f"  WARN {method} {path} → HTTP {r.status_code}", file=sys.stderr)
        latencies.append((t1 - t0) * 1000)
    return latencies


def _stats(latencies: list[float]) -> dict[str, float]:
    """Calcule p50/p95/p99/mean/min/max depuis une liste de latences (ms)."""
    s = sorted(latencies)
    n = len(s)

    def pct(p: float) -> float:
        idx = int(p / 100 * n)
        return round(s[min(idx, n - 1)], 2)

    return {
        "p50_ms": pct(50),
        "p95_ms": pct(95),
        "p99_ms": pct(99),
        "mean_ms": round(statistics.mean(latencies), 2),
        "min_ms": round(min(latencies), 2),
        "max_ms": round(max(latencies), 2),
        "samples": n,
    }


def _endpoints(player: str, match_id: str) -> list[tuple[str, str, str, dict | None]]:
    """Retourne la liste (label, method, path, body) à benchmarker."""
    slug = player
    return [
        ("GET /api/v1/health", "GET", "/api/v1/health", None),
        ("GET /api/v1/bootstrap", "GET", "/api/v1/bootstrap", None),
        ("GET /api/v1/players", "GET", "/api/v1/players", None),
        (
            f"POST /api/v1/players/{slug}/filters/resolve",
            "POST",
            f"/api/v1/players/{slug}/filters/resolve",
            {},
        ),
        (
            f"GET /api/v1/players/{slug}/pages/career",
            "GET",
            f"/api/v1/players/{slug}/pages/career",
            None,
        ),
        (
            f"POST /api/v1/players/{slug}/pages/match-history/query",
            "POST",
            f"/api/v1/players/{slug}/pages/match-history/query",
            {
                "filters": {},
                "sort": {"field": "played_at", "direction": "desc"},
                "page": 1,
                "page_size": 20,
            },
        ),
        (
            "GET /api/v1/directory/gamertags/search?q=ref",
            "GET",
            "/api/v1/directory/gamertags/search?q=ref",
            None,
        ),
        (
            f"GET /api/v1/players/{slug}/matches/{match_id}",
            "GET",
            f"/api/v1/players/{slug}/matches/{match_id}",
            None,
        ),
    ]


def _discover_match_id(client: httpx.Client, slug: str) -> str:
    """Tente de récupérer un match_id réel depuis l'historique."""
    try:
        r = client.post(
            f"/api/v1/players/{slug}/pages/match-history/query",
            json={
                "filters": {},
                "sort": {"field": "played_at", "direction": "desc"},
                "page": 1,
                "page_size": 1,
            },
            timeout=10,
        )
        if r.status_code == 200:
            data = r.json()
            # format: {"summary": ..., "table": {"items": [...]}} ou {"items": [...]}
            items = (
                data.get("table", {}).get("items")
                or data.get("items")
                or data.get("data", {}).get("items", [])
            )
            if items:
                return items[0].get("match_id", "UNKNOWN_MATCH_ID")
    except Exception:
        pass
    return "UNKNOWN_MATCH_ID"


def run(base_url: str, player: str, samples: int, out: Path) -> None:
    """Point d'entrée principal du benchmark."""
    global SAMPLES_DEFAULT
    SAMPLES_DEFAULT = samples

    print(f"Benchmark Python API @ {base_url}  player={player}  samples={samples}")
    print("=" * 60)

    with httpx.Client(base_url=base_url, timeout=30) as client:
        # Warm-up
        try:
            client.get("/health")
        except httpx.ConnectError:
            print(f"ERREUR : API Python non joignable sur {base_url}", file=sys.stderr)
            print(
                "Lancer d'abord : python -m uvicorn apps.api.app.main:app --port 8000",
                file=sys.stderr,
            )
            sys.exit(1)

        match_id = _discover_match_id(client, player)
        print(f"match_id découvert : {match_id}")

        results: dict[str, dict] = {}
        endpoints = _endpoints(player, match_id)

        for label, method, path, body in endpoints:
            print(f"  {method:4s} {path[:60]}", end=" ... ", flush=True)
            latencies = _measure(client, method, path, body)
            s = _stats(latencies)
            results[label] = s
            print(f"p50={s['p50_ms']}ms  p95={s['p95_ms']}ms  p99={s['p99_ms']}ms")

    output = {
        "_meta": {
            "version": "1.0",
            "captured_at": datetime.now(timezone.utc).isoformat(),
            "base_url": base_url,
            "player": player,
            "samples_per_endpoint": samples,
            "match_id": match_id,
            "note": (
                "Baseline Python. Règle : si Go > 2× p95 Python sur un endpoint "
                "= bug de performance à investiguer (hors cold-start)."
            ),
        },
        **results,
    }

    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(output, indent=2, ensure_ascii=False), encoding="utf-8")
    print(f"\nBaselines écrites dans : {out}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Benchmark API Python LevelUp")
    parser.add_argument("--base-url", default=BASE_URL_DEFAULT)
    parser.add_argument("--player", default="demo-player")
    parser.add_argument("--samples", type=int, default=SAMPLES_DEFAULT)
    parser.add_argument("--out", type=Path, default=OUT_DEFAULT)
    args = parser.parse_args()
    run(args.base_url, args.player, args.samples, args.out)


if __name__ == "__main__":
    main()
