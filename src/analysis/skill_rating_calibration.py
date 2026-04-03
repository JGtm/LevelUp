"""Module de calibration des poids LUSR par comparaison avec un MMR individuel décorrélé.

Charge l'historique des matchs d'un joueur, exécute compute_skill_ratings_batch()
avec différentes combinaisons de poids pour COMPOSITE_WEIGHTS, et minimise le MAE
(Mean Absolute Error) entre le LUSR calculé et le MMR individuel décorrélé.

Le MMR individuel décorrélé est calculé comme :
    individual_mmr = team_mmr × (kills_expected_joueur / kills_expected_moyen_du_match)

Contrairement au team_mmr brut (compressé autour de 1000-1200 pour tous les joueurs),
ce MMR amplifie les différences individuelles en pondérant par kills_expected,
l'estimation Halo de combien de kills le joueur devrait faire dans ce contexte de match.

Usage CLI :
    python -m src.analysis.skill_rating_calibration --player <Gamertag>
    python -m src.analysis.skill_rating_calibration --player <GT> --n-samples 300
    python -m src.analysis.skill_rating_calibration --player <GT> --metric corr --quiet

Les poids optimaux peuvent être copiés dans skill_rating_config.py > COMPOSITE_WEIGHTS.
"""

from __future__ import annotations

import argparse
import logging
import random
import sys
from pathlib import Path
from typing import Any

import polars as pl

from src.analysis._calibration_loaders import (
    _compute_individual_mmr_map,  # noqa: F401
    _detect_shared_db,
    _load_matches_for_calibration,
    _resolve_xuid_from_gamertag,
)
from src.analysis._calibration_scoring import _score_corr, _score_mae
from src.utils.paths import get_shared_matches_path

logger = logging.getLogger(__name__)

# Clés canoniques des poids (ordre stable pour affichage)
_WEIGHT_KEYS = (
    "kills_vs_expected",
    "deaths_vs_expected",
    "win_factor",
    "damage_efficiency",
    "accuracy_delta",
)


# =============================================================================
# Génération des candidats de poids
# =============================================================================


def _generate_candidates(
    n_samples: int,
    *,
    rng_seed: int = 42,
    include_default: bool = True,
) -> list[dict[str, float]]:
    """Génère n_samples combinaisons de poids normalisées (somme = 1).

    Utilise un échantillonnage de style Dirichlet (uniforme sur le simplexe)
    pour couvrir l'espace des poids de façon équilibrée.
    """
    from src.analysis.skill_rating_config import COMPOSITE_WEIGHTS

    rng = random.Random(rng_seed)
    candidates: list[dict[str, float]] = []

    if include_default:
        candidates.append(dict(COMPOSITE_WEIGHTS))

    remaining = n_samples - len(candidates)
    for _ in range(remaining):
        # Méthode des espacements : uniforme sur le simplexe
        cuts = sorted(rng.random() for _ in range(len(_WEIGHT_KEYS) - 1))
        boundaries = [0.0] + cuts + [1.0]
        # Valeurs brutes (somment à 1, mais peuvent être petites)
        raw = [boundaries[i + 1] - boundaries[i] for i in range(len(_WEIGHT_KEYS))]
        # Garantir un minimum de 2% par dimension
        raw = [max(r, 0.02) for r in raw]
        total = sum(raw)
        weights = {k: v / total for k, v in zip(_WEIGHT_KEYS, raw, strict=False)}
        candidates.append(weights)

    return candidates


# =============================================================================
# Cœur de la calibration
# =============================================================================


def _run_weight_optimization(  # noqa: PLR0913
    weight_candidates: list[dict[str, float]],
    df_matches: pl.DataFrame,
    df_participants: pl.DataFrame,
    team_mmr_map: dict[str, float],
    is_mae: bool,
    verbose: bool,
) -> tuple[dict[str, float], float, float | None, list[dict[str, Any]]]:
    """Évalue toutes les combinaisons de poids et retourne la meilleure."""
    import src.analysis.skill_rating as _sr

    score_fn = _score_mae if is_mae else _score_corr
    best_score: float = float("inf") if is_mae else float("-inf")
    best_weights: dict[str, float] = {}
    default_score: float | None = None
    results: list[dict[str, Any]] = []

    for i, weights in enumerate(weight_candidates):
        if verbose and i > 0 and i % 50 == 0:
            print(f"  [{i}/{len(weight_candidates)}] ...")
        try:
            df_ratings = _sr.compute_skill_ratings_batch(
                df_matches, df_participants, weights=weights
            )
        except Exception as exc:
            logger.debug("Erreur avec poids #%d: %s", i, exc)
            continue
        if df_ratings.is_empty():
            continue
        score = score_fn(df_ratings, team_mmr_map)
        if i == 0:
            default_score = score
        results.append({"weights": dict(weights), "score": score})
        if (is_mae and score < best_score) or (not is_mae and score > best_score):
            best_score = score
            best_weights = dict(weights)

    results.sort(key=lambda x: x["score"], reverse=not is_mae)
    return best_weights, best_score, default_score, results


def calibrate_lusr_weights(  # noqa: PLR0913
    db_path: str | Path,
    xuid: str,
    *,
    shared_db_path: str | Path | None = None,
    n_samples: int = 200,
    min_matches_with_mmr: int = 30,
    metric: str = "mae",
    rng_seed: int = 42,
    verbose: bool = True,
) -> dict[str, Any]:
    """Calibre les poids COMPOSITE_WEIGHTS par comparaison avec team_mmr API."""
    path = Path(db_path)
    shared_db = _detect_shared_db(path, shared_db_path)

    if verbose:
        print(f"Chargement des matchs depuis {shared_db}...")

    df_matches, df_participants, team_mmr_map = _load_matches_for_calibration(
        shared_db,
        xuid,
        min_matches_with_mmr=min_matches_with_mmr,
    )

    n_matches = df_matches.height
    n_with_mmr = len(team_mmr_map)
    is_mae = metric == "mae"

    if verbose:
        print(
            f"{n_matches} matchs chargés ({n_with_mmr} avec team_mmr) — "
            f"test de {n_samples} combinaisons de poids..."
        )

    weight_candidates = _generate_candidates(n_samples, rng_seed=rng_seed)
    best_weights, best_score, default_score, results = _run_weight_optimization(
        weight_candidates,
        df_matches,
        df_participants,
        team_mmr_map,
        is_mae,
        verbose,
    )

    improvement_pct: float | None = None
    if default_score is not None and abs(default_score) > 1e-9:
        improvement_pct = abs(default_score - best_score) / abs(default_score) * 100.0

    return {
        "best_weights": best_weights,
        "best_score": best_score,
        "default_score": default_score,
        "improvement_pct": improvement_pct,
        "n_matches": n_matches,
        "n_matches_with_mmr": n_with_mmr,
        "metric": metric,
        "n_samples_tested": len(results),
        "top_results": results[:10],
    }


# =============================================================================
# Point d'entrée CLI
# =============================================================================


def _build_cli_parser() -> argparse.ArgumentParser:
    """Construit le parser CLI pour la calibration LUSR."""
    parser = argparse.ArgumentParser(
        prog="python -m src.analysis.skill_rating_calibration",
        description="Calibration des poids LUSR par comparaison avec team_mmr API.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--player", required=True, metavar="GAMERTAG", help="Gamertag du joueur")
    parser.add_argument("--xuid", metavar="XUID", help="XUID (résolu depuis gamertag si absent)")
    parser.add_argument(
        "--n-samples",
        type=int,
        default=200,
        metavar="N",
        help="Nombre de combinaisons à tester (défaut: 200)",
    )
    parser.add_argument(
        "--min-matches",
        type=int,
        default=30,
        metavar="N",
        help="Matchs minimum avec team_mmr (défaut: 30)",
    )
    parser.add_argument(
        "--metric",
        choices=["mae", "corr"],
        default="mae",
        help="Métrique : mae (minimiser) ou corr (maximiser, défaut: mae)",
    )
    parser.add_argument("--shared-db", metavar="PATH", help="Chemin vers shared_matches_v2.duckdb")
    parser.add_argument("--seed", type=int, default=42, help="Graine aléatoire (défaut: 42)")
    parser.add_argument("--quiet", action="store_true", help="Sortie minimale")
    return parser


def _display_calibration_results(
    result: dict[str, Any],
    player: str,
    quiet: bool,
) -> None:
    """Affiche les résultats formatés de la calibration LUSR."""
    from src.analysis.skill_rating_config import COMPOSITE_WEIGHTS as _DEFAULT_W

    is_mae = result["metric"] == "mae"

    def _fmt(s: float) -> str:
        suffix = " (bas = meilleur)" if is_mae else " (haut = meilleur)"
        return f"{s:.4f}{suffix}"

    sep = "=" * 62
    print(f"\n{sep}")
    print("  CALIBRATION LUSR — RÉSULTATS")
    print(sep)
    print(f"  Joueur             : {player}")
    print(f"  Matchs totaux      : {result['n_matches']}")
    print(f"  Matchs avec MMR ind: {result['n_matches_with_mmr']}")
    print(f"  Métrique           : {result['metric'].upper()}")
    print(f"  Combinaisons testées : {result['n_samples_tested']}")
    print()

    if result["default_score"] is not None:
        print(f"  Score poids défaut : {_fmt(result['default_score'])}")
    print(f"  Meilleur score     : {_fmt(result['best_score'])}")
    if result["improvement_pct"] is not None:
        direction = "baisse" if is_mae else "hausse"
        print(f"  Amélioration       : {result['improvement_pct']:.1f}% ({direction})")

    print()
    print("  POIDS OPTIMAUX — à copier dans skill_rating_config.py :")
    print("-" * 62)
    print("  COMPOSITE_WEIGHTS: dict[str, float] = {")
    for key, val in result["best_weights"].items():
        default_val = _DEFAULT_W.get(key, 0.0)
        marker = "  # [modifié]" if abs(val - default_val) > 0.01 else ""
        print(f'      "{key}": {val:.4f},{marker}')
    print("  }")

    if not quiet and len(result["top_results"]) >= 3:
        print()
        print("  TOP 3 COMBINAISONS :")
        print("-" * 62)
        for rank, entry in enumerate(result["top_results"][:3], 1):
            print(f"\n  #{rank}  Score: {entry['score']:.4f}")
            for k, v in entry["weights"].items():
                default_val = _DEFAULT_W.get(k, 0.0)
                marker = " [mod]" if abs(v - default_val) > 0.01 else ""
                print(f"      {k}: {v:.4f}{marker}")

    print(f"\n{sep}\n")


def main(argv: list[str] | None = None) -> int:
    """Point d'entrée CLI pour la calibration LUSR."""
    args = _build_cli_parser().parse_args(argv)

    root = Path(__file__).resolve().parents[2]
    db_path = root / "data" / "players" / args.player / "stats.duckdb"

    if not db_path.exists():
        print(f"Erreur : DB joueur introuvable : {db_path}", file=sys.stderr)
        return 1

    # Résolution XUID
    xuid = args.xuid
    if not xuid:
        shared_db_for_resolve = (
            Path(args.shared_db) if args.shared_db else get_shared_matches_path()
        )
        xuid = _resolve_xuid_from_gamertag(args.player, shared_db_for_resolve)

    if not xuid:
        print(
            f"Erreur : XUID introuvable pour « {args.player} ». "
            "Utilisez --xuid pour le fournir manuellement.",
            file=sys.stderr,
        )
        return 1

    try:
        result = calibrate_lusr_weights(
            db_path=db_path,
            xuid=xuid,
            shared_db_path=args.shared_db,
            n_samples=args.n_samples,
            min_matches_with_mmr=args.min_matches,
            metric=args.metric,
            rng_seed=args.seed,
            verbose=not args.quiet,
        )
    except (FileNotFoundError, ValueError) as exc:
        print(f"Erreur : {exc}", file=sys.stderr)
        return 1

    _display_calibration_results(result, args.player, args.quiet)
    return 0


if __name__ == "__main__":
    sys.exit(main())
