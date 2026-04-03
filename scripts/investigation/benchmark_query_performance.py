#!/usr/bin/env python
"""Benchmark des requêtes analytiques DuckDB — Étape 2 Performance Données.

Mesure l'impact des index et optimisations de requêtes sur les opérations
les plus lourdes du projet.

Usage:
    python scripts/benchmark_query_performance.py
    python scripts/benchmark_query_performance.py --runs 5
    python scripts/benchmark_query_performance.py --baseline --output .ai/reports/query_perf_baseline.json
    python scripts/benchmark_query_performance.py --compare .ai/reports/query_perf_baseline.json
"""

from __future__ import annotations

import argparse
import json
import statistics
import subprocess
import sys
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(PROJECT_ROOT))

import duckdb

from src.data.repositories import DuckDBRepository
from src.utils.paths import SHARED_MATCHES_DB_FILENAME


@dataclass
class QueryBenchResult:
    """Résultat d'un benchmark de requête."""

    name: str
    description: str = ""
    times_ms: list[float] = field(default_factory=list)
    rows_returned: int = 0
    success: bool = True
    error: str | None = None

    @property
    def mean_ms(self) -> float:
        """Temps moyen en ms."""
        return statistics.mean(self.times_ms) if self.times_ms else 0.0

    @property
    def min_ms(self) -> float:
        """Temps minimum en ms."""
        return min(self.times_ms) if self.times_ms else 0.0

    @property
    def p50_ms(self) -> float:
        """Médiane en ms."""
        return statistics.median(self.times_ms) if self.times_ms else 0.0

    def to_dict(self) -> dict:
        """Sérialise en dict."""
        return {
            "name": self.name,
            "description": self.description,
            "mean_ms": round(self.mean_ms, 2),
            "min_ms": round(self.min_ms, 2),
            "p50_ms": round(self.p50_ms, 2),
            "rows_returned": self.rows_returned,
            "success": self.success,
            "error": self.error,
        }


def _resolve_profile() -> tuple[str, str]:
    """Retourne (db_path, xuid) du premier profil valide."""
    profiles_path = PROJECT_ROOT / "db_profiles.json"
    if not profiles_path.exists():
        raise FileNotFoundError("db_profiles.json introuvable")

    with open(profiles_path, encoding="utf-8") as f:
        data = json.load(f)

    for _gt, profile in data.get("profiles", {}).items():
        db_path = profile.get("db_path", "")
        xuid = profile.get("xuid", "")
        full_path = PROJECT_ROOT / db_path
        if full_path.exists() and xuid:
            return str(full_path), xuid

    raise FileNotFoundError("Aucun profil valide trouvé")


def _get_git_hash() -> str:
    """Retourne le hash court du commit courant."""
    try:
        result = subprocess.run(
            ["git", "rev-parse", "--short", "HEAD"],
            capture_output=True,
            text=True,
            cwd=str(PROJECT_ROOT),
            timeout=5,
        )
        return result.stdout.strip() if result.returncode == 0 else "unknown"
    except Exception:
        return "unknown"


# ---------------------------------------------------------------------------
# Benchmarks individuels
# ---------------------------------------------------------------------------


def bench_load_matches_warm(db_path: str, xuid: str, runs: int) -> QueryBenchResult:
    """Benchmark: load_matches warm (connexion réutilisée)."""
    result = QueryBenchResult(
        name="load_matches_warm",
        description="Chargement matchs (connexion réutilisée, ORDER BY start_time)",
    )
    try:
        repo = DuckDBRepository(db_path, xuid)
        try:
            for _ in range(runs):
                t0 = time.perf_counter()
                matches = repo.load_matches()
                elapsed = (time.perf_counter() - t0) * 1000
                result.times_ms.append(elapsed)
                result.rows_returned = len(matches) if matches else 0
        finally:
            repo.close()
    except Exception as e:
        result.success = False
        result.error = str(e)
    return result


def bench_top_teammates(db_path: str, xuid: str, runs: int) -> QueryBenchResult:
    """Benchmark: list_top_teammates (self-join O(n²))."""
    result = QueryBenchResult(
        name="list_top_teammates",
        description="Self-JOIN match_participants pour coéquipiers (xuid, team_id)",
    )
    try:
        for _ in range(runs):
            repo = DuckDBRepository(db_path, xuid)
            try:
                t0 = time.perf_counter()
                data = repo.list_top_teammates(limit=20)
                elapsed = (time.perf_counter() - t0) * 1000
                result.times_ms.append(elapsed)
                result.rows_returned = len(data) if data else 0
            finally:
                repo.close()
    except Exception as e:
        result.success = False
        result.error = str(e)
    return result


def bench_first_event_times(db_path: str, xuid: str, runs: int) -> QueryBenchResult:
    """Benchmark: load_first_event_times (event_type filter)."""
    result = QueryBenchResult(
        name="load_first_event_times",
        description="MIN(time_ms) par match avec filtre event_type (plus LOWER())",
    )
    try:
        repo = DuckDBRepository(db_path, xuid)
        try:
            # Récupérer des match_ids pour le test
            matches = repo.load_matches()
            match_ids = [m.match_id for m in matches[:100]] if matches else []

            if match_ids:
                for _ in range(runs):
                    t0 = time.perf_counter()
                    repo.load_first_event_times(match_ids, event_type="Kill")
                    elapsed = (time.perf_counter() - t0) * 1000
                    result.times_ms.append(elapsed)
            else:
                result.error = "Pas de match_ids disponibles"
                result.success = False
        finally:
            repo.close()
    except Exception as e:
        result.success = False
        result.error = str(e)
    return result


def bench_top_medals(db_path: str, xuid: str, runs: int) -> QueryBenchResult:
    """Benchmark: load_top_medals (agrégation medals_earned)."""
    result = QueryBenchResult(
        name="load_top_medals",
        description="SUM(count) GROUP BY medal_name_id sur medals_earned",
    )
    try:
        repo = DuckDBRepository(db_path, xuid)
        try:
            matches = repo.load_matches()
            match_ids = [m.match_id for m in (matches or [])[:100]]

            if match_ids:
                for _ in range(runs):
                    t0 = time.perf_counter()
                    medals = repo.load_top_medals(match_ids, top_n=20)
                    elapsed = (time.perf_counter() - t0) * 1000
                    result.times_ms.append(elapsed)
                    result.rows_returned = len(medals) if medals else 0
            else:
                result.error = "Pas de match_ids"
                result.success = False
        finally:
            repo.close()
    except Exception as e:
        result.success = False
        result.error = str(e)
    return result


def bench_materialized_views(db_path: str, xuid: str, runs: int) -> QueryBenchResult:
    """Benchmark: refresh_materialized_views (GROUP BY sur match_stats)."""
    result = QueryBenchResult(
        name="refresh_materialized_views",
        description="Refresh vues matérialisées (GROUP BY session_id, map_id, etc.)",
    )
    try:
        for _ in range(runs):
            repo = DuckDBRepository(db_path, xuid, read_only=False)
            try:
                t0 = time.perf_counter()
                repo.refresh_materialized_views()
                elapsed = (time.perf_counter() - t0) * 1000
                result.times_ms.append(elapsed)
            finally:
                repo.close()
    except Exception as e:
        result.success = False
        result.error = str(e)
    return result


def bench_match_count_by_outcome(db_path: str, xuid: str, runs: int) -> QueryBenchResult:
    """Benchmark: requête analytique groupée par outcome (simule sidebar)."""
    result = QueryBenchResult(
        name="match_count_by_outcome",
        description="COUNT(*) GROUP BY outcome sur match_stats (simule sidebar)",
    )
    try:
        conn = duckdb.connect(db_path, read_only=True)
        try:
            for _ in range(runs):
                t0 = time.perf_counter()
                rows = conn.execute("""
                    SELECT outcome, COUNT(*) as cnt
                    FROM match_stats
                    GROUP BY outcome
                """).fetchall()
                elapsed = (time.perf_counter() - t0) * 1000
                result.times_ms.append(elapsed)
                result.rows_returned = len(rows)
        except Exception:
            # match_stats peut ne pas exister
            result.error = "match_stats non disponible"
            result.success = False
        finally:
            conn.close()
    except Exception as e:
        result.success = False
        result.error = str(e)
    return result


def list_indexes(db_path: str) -> dict[str, list[str]]:
    """Liste les index existants sur la DB joueur."""
    result: dict[str, list[str]] = {"local": [], "shared": []}
    try:
        conn = duckdb.connect(db_path, read_only=True)
        try:
            rows = conn.execute(
                "SELECT index_name, table_name FROM duckdb_indexes() "
                "WHERE database_name = 'memory' OR database_name NOT LIKE 'shared%'"
            ).fetchall()
            result["local"] = [f"{r[0]} ON {r[1]}" for r in rows]
        except Exception:
            pass

        # Attacher shared pour lister ses index
        shared_path = Path(db_path).parent.parent.parent / "warehouse" / SHARED_MATCHES_DB_FILENAME
        if shared_path.exists():
            try:
                conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")
                rows = conn.execute(
                    "SELECT index_name, table_name FROM duckdb_indexes() "
                    "WHERE database_name = 'shared'"
                ).fetchall()
                result["shared"] = [f"{r[0]} ON {r[1]}" for r in rows]
            except Exception:
                pass
        conn.close()
    except Exception:
        pass
    return result


# ---------------------------------------------------------------------------
# Orchestration
# ---------------------------------------------------------------------------


def run_all_benchmarks(db_path: str, xuid: str, runs: int = 3) -> list[QueryBenchResult]:
    """Exécute tous les benchmarks."""
    benchmarks = [
        bench_load_matches_warm,
        bench_top_teammates,
        bench_first_event_times,
        bench_top_medals,
        bench_materialized_views,
        bench_match_count_by_outcome,
    ]

    results: list[QueryBenchResult] = []
    for bench_fn in benchmarks:
        print(f"  ⏱️  {bench_fn.__name__}...", end="", flush=True)
        res = bench_fn(db_path, xuid, runs)
        status = f" {res.mean_ms:.1f}ms" if res.success else f" ❌ {res.error}"
        print(status)
        results.append(res)

    return results


def print_report(results: list[QueryBenchResult], indexes: dict) -> None:
    """Affiche le rapport de benchmark."""
    print("\n" + "=" * 70)
    print("📊 RAPPORT BENCHMARK REQUÊTES ANALYTIQUES")
    print("=" * 70)

    print(f"\n📁 Index locaux : {len(indexes.get('local', []))}")
    for idx in indexes.get("local", []):
        print(f"   • {idx}")

    print(f"\n📁 Index shared : {len(indexes.get('shared', []))}")
    for idx in indexes.get("shared", []):
        print(f"   • {idx}")

    print(f"\n{'Requête':<30} {'Moy (ms)':>10} {'Min (ms)':>10} {'P50 (ms)':>10} {'Rows':>8}")
    print("-" * 70)
    for r in results:
        if r.success:
            print(
                f"{r.name:<30} {r.mean_ms:>10.1f} {r.min_ms:>10.1f} "
                f"{r.p50_ms:>10.1f} {r.rows_returned:>8}"
            )
        else:
            print(f"{r.name:<30} {'SKIP':>10} — {r.error or ''}")


def compare_with_baseline(results: list[QueryBenchResult], baseline_path: str) -> None:
    """Compare les résultats avec un fichier baseline."""
    with open(baseline_path, encoding="utf-8") as f:
        baseline = json.load(f)

    baseline_map = {r["name"]: r for r in baseline.get("results", [])}

    print(f"\n{'Requête':<30} {'Avant':>10} {'Après':>10} {'Delta':>10} {'%':>8}")
    print("-" * 70)
    for r in results:
        if not r.success:
            continue
        base = baseline_map.get(r.name)
        if base and base.get("success"):
            before = base["mean_ms"]
            after = r.mean_ms
            delta = after - before
            pct = (delta / before * 100) if before > 0 else 0
            sign = "+" if delta > 0 else ""
            emoji = "🟢" if delta < 0 else "🔴" if delta > 0 else "⚪"
            print(
                f"{emoji} {r.name:<28} {before:>10.1f} {after:>10.1f} "
                f"{sign}{delta:>9.1f} {sign}{pct:>7.1f}%"
            )
        else:
            print(f"⚪ {r.name:<28} {'N/A':>10} {r.mean_ms:>10.1f}")


def main() -> None:
    """Point d'entrée principal."""
    parser = argparse.ArgumentParser(description="Benchmark requêtes DuckDB")
    parser.add_argument("--runs", type=int, default=3, help="Nombre d'itérations")
    parser.add_argument("--baseline", action="store_true", help="Sauvegarder comme baseline")
    parser.add_argument("--output", type=str, help="Fichier de sortie JSON")
    parser.add_argument("--compare", type=str, help="Comparer avec un fichier baseline")
    args = parser.parse_args()

    db_path, xuid = _resolve_profile()
    print(f"🎯 DB: {db_path}")
    print(f"🎮 XUID: {xuid}")
    print(f"🔄 Runs: {args.runs}")
    print(f"📌 Git: {_get_git_hash()}")

    # Lister les index
    indexes = list_indexes(db_path)

    # Exécuter les benchmarks
    print("\n⏱️  Exécution des benchmarks...")
    results = run_all_benchmarks(db_path, xuid, args.runs)

    # Rapport
    print_report(results, indexes)

    # Comparaison
    if args.compare and Path(args.compare).exists():
        compare_with_baseline(results, args.compare)

    # Sauvegarde
    if args.baseline or args.output:
        output_path = args.output or ".ai/reports/query_perf_baseline.json"
        Path(output_path).parent.mkdir(parents=True, exist_ok=True)
        report = {
            "timestamp": datetime.now(tz=timezone.utc).isoformat(),
            "git_hash": _get_git_hash(),
            "runs": args.runs,
            "db_path": db_path,
            "indexes": indexes,
            "results": [r.to_dict() for r in results],
        }
        with open(output_path, "w", encoding="utf-8") as f:
            json.dump(report, f, indent=2, ensure_ascii=False)
        print(f"\n💾 Résultats sauvegardés: {output_path}")


if __name__ == "__main__":
    main()
