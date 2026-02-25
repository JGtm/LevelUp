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
# Chargement des données
# =============================================================================


def _load_matches_for_calibration(
    shared_db: Path,
    xuid: str,
    *,
    min_matches_with_mmr: int = 30,
) -> tuple[pl.DataFrame, pl.DataFrame, dict[str, float]]:
    """Charge les matchs, participants et MMR individuel décorrélé depuis shared_matches.duckdb.

    Le MMR individuel décorrélé est :
        individual_mmr = team_mmr × (kills_expected_joueur / kills_expected_moyen_du_match)

    Ce signal est bien plus discriminant que le team_mmr brut (compressé 1000-1200
    pour tous les niveaux), car il amplifie les différences de kills_expected
    — l'estimation Halo du skill individuel dans le contexte du match.

    Returns:
        (df_matches, df_participants, individual_mmr_map) où :
        - df_matches : TOUS les matchs triés ASC (pour LUSR complet)
        - df_participants : tous les participants de ces matchs
        - individual_mmr_map : individual_mmr indexé par match_id

    Raises:
        ValueError: si moins de min_matches_with_mmr matchs ont team_mmr + kills_expected.
    """
    import duckdb

    conn = duckdb.connect(str(shared_db), read_only=True)
    try:
        # Tous les matchs du joueur (sans Firefight), triés ASC
        df_matches = conn.execute(
            """
            SELECT
                mp.match_id,
                mr.start_time,
                COALESCE(mr.playlist_name, '') AS playlist_name,
                COALESCE(mr.pair_name, '')     AS pair_name,
                COALESCE(mp.outcome, 3)        AS outcome,
                COALESCE(mp.kills, 0)          AS kills,
                COALESCE(mp.deaths, 1)         AS deaths,
                COALESCE(mp.assists, 0)        AS assists,
                mp.kills_expected,
                mp.deaths_expected,
                mp.damage_dealt,
                mp.damage_taken,
                mp.accuracy,
                mp.team_id,
                mp.team_mmr,
                mp.enemy_mmr,
                COALESCE(mr.is_ranked, FALSE)  AS is_ranked,
                COALESCE(mr.is_firefight, FALSE) AS is_firefight
            FROM match_participants mp
            JOIN match_registry mr ON mr.match_id = mp.match_id
            WHERE mp.xuid = ?
              AND COALESCE(mr.is_firefight, FALSE) = FALSE
            ORDER BY mr.start_time ASC
            """,
            [xuid],
        ).pl()

        if df_matches.is_empty():
            raise ValueError(f"Aucun match trouvé pour XUID {xuid}.")

        # Tous participants (pour estimation μ adversaires + calcul kills_expected moyen)
        match_ids = df_matches["match_id"].to_list()
        placeholders = ", ".join(["?"] * len(match_ids))
        df_participants = conn.execute(
            f"""
            SELECT
                match_id,
                xuid::TEXT AS xuid,
                team_id,
                kills_expected,
                deaths_expected
            FROM match_participants
            WHERE match_id IN ({placeholders})
            """,
            match_ids,
        ).pl()

        # Calculer kills_expected moyen par match (sur tous les participants)
        ke_avg_by_match: dict[str, float] = {}
        for row in (
            df_participants.filter(
                pl.col("kills_expected").is_not_null() & (pl.col("kills_expected") > 0)
            )
            .group_by("match_id")
            .agg(pl.col("kills_expected").mean().alias("ke_avg"))
            .iter_rows(named=True)
        ):
            ke_avg_by_match[row["match_id"]] = row["ke_avg"]

        # Construire individual_mmr_map :
        #   individual_mmr = team_mmr × (kills_expected_joueur / kills_expected_moyen_match)
        individual_mmr_map: dict[str, float] = {}
        for row in df_matches.iter_rows(named=True):
            mid = row["match_id"]
            team_mmr = row["team_mmr"]
            ke_me = row["kills_expected"]
            ke_avg = ke_avg_by_match.get(mid)
            if team_mmr is None or ke_me is None or ke_avg is None or ke_avg <= 0:
                continue
            individual_mmr_map[mid] = team_mmr * (ke_me / ke_avg)

        n_with_mmr = len(individual_mmr_map)
        if n_with_mmr < min_matches_with_mmr:
            raise ValueError(
                f"Seulement {n_with_mmr} matchs avec individual_mmr calculable "
                f"(minimum {min_matches_with_mmr}). "
                "Effectuez un backfill --skill pour enrichir les données."
            )

        return df_matches, df_participants, individual_mmr_map

    finally:
        conn.close()


# =============================================================================
# Métriques d'évaluation
# =============================================================================


def _normalize(series: pl.Series) -> pl.Series:
    """Normalise une série à [0, 1] par min-max."""
    lo = series.min()
    hi = series.max()
    if hi is None or lo is None or hi == lo:
        return pl.Series([0.5] * len(series))
    return (series - lo) / (hi - lo)


def _score_mae(
    df_ratings: pl.DataFrame,
    team_mmr_map: dict[str, float],
) -> float:
    """Calcule MAE normalisé entre LUSR et individual_mmr décorrélé (plus bas = meilleur)."""
    rows = [
        (mid, rv, team_mmr_map[mid])
        for mid, rv in zip(
            df_ratings["match_id"].to_list(),
            df_ratings["rating_value"].to_list(),
            strict=False,
        )
        if mid in team_mmr_map
    ]
    if len(rows) < 10:
        return 1.0

    r_series = pl.Series([r[1] for r in rows], dtype=pl.Float64)
    t_series = pl.Series([r[2] for r in rows], dtype=pl.Float64)

    r_norm = _normalize(r_series)
    t_norm = _normalize(t_series)
    return float((r_norm - t_norm).abs().mean())


def _score_corr(
    df_ratings: pl.DataFrame,
    team_mmr_map: dict[str, float],
) -> float:
    """Corrélation de Pearson entre LUSR et individual_mmr décorrélé (plus haut = meilleur)."""
    rows = [
        (mid, rv, team_mmr_map[mid])
        for mid, rv in zip(
            df_ratings["match_id"].to_list(),
            df_ratings["rating_value"].to_list(),
            strict=False,
        )
        if mid in team_mmr_map
    ]
    if len(rows) < 10:
        return -1.0

    r_series = pl.Series([r[1] for r in rows], dtype=pl.Float64)
    t_series = pl.Series([r[2] for r in rows], dtype=pl.Float64)

    corr = r_series.pearson_corr(t_series)
    return float(corr) if corr is not None else 0.0


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


def calibrate_lusr_weights(
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
    """Calibre les poids COMPOSITE_WEIGHTS par comparaison avec team_mmr API.

    Args:
        db_path: Chemin vers stats.duckdb du joueur (pour auto-détection shared).
        xuid: XUID du joueur.
        shared_db_path: Chemin vers shared_matches.duckdb (auto-détecté si None).
        n_samples: Nombre de combinaisons de poids à tester.
        min_matches_with_mmr: Minimum de matchs avec team_mmr pour calibration valide.
        metric: "mae" (MAE normalisé, minimiser) ou "corr" (Pearson, maximiser).
        rng_seed: Graine aléatoire pour reproductibilité.
        verbose: Affiche la progression sur stdout.

    Returns:
        Dict avec :
            best_weights (dict[str, float]) — poids optimaux,
            best_score (float) — score optimal,
            default_score (float) — score avec poids par défaut,
            improvement_pct (float) — amélioration en %,
            n_matches (int) — matchs totaux utilisés,
            n_matches_with_mmr (int) — matchs avec team_mmr disponible,
            metric (str),
            n_samples_tested (int),
            top_results (list[dict]) — top 10 combinaisons.
    """
    import src.analysis.skill_rating as _sr

    path = Path(db_path)

    # ── Auto-détection shared ──
    if shared_db_path is None:
        candidates_paths = [
            path.parent.parent.parent / "warehouse" / "shared_matches.duckdb",
            Path(__file__).resolve().parents[2] / "data" / "warehouse" / "shared_matches.duckdb",
        ]
        shared_db = next((p for p in candidates_paths if p.exists()), None)
    else:
        shared_db = Path(shared_db_path)

    if shared_db is None or not shared_db.exists():
        raise FileNotFoundError(
            "shared_matches.duckdb introuvable. " "Vérifiez le chemin ou utilisez --shared-db."
        )

    if verbose:
        print(f"Chargement des matchs depuis {shared_db}...")

    df_matches, df_participants, team_mmr_map = _load_matches_for_calibration(
        shared_db,
        xuid,
        min_matches_with_mmr=min_matches_with_mmr,
    )

    n_matches = df_matches.height
    n_with_mmr = len(team_mmr_map)

    if verbose:
        print(
            f"{n_matches} matchs chargés ({n_with_mmr} avec team_mmr) — "
            f"test de {n_samples} combinaisons de poids..."
        )

    # ── Scoring ──
    is_mae = metric == "mae"
    best_score: float = float("inf") if is_mae else float("-inf")
    best_weights: dict[str, float] = {}
    default_score: float | None = None
    results: list[dict[str, Any]] = []

    score_fn = _score_mae if is_mae else _score_corr
    is_better = (lambda s: s < best_score) if is_mae else (lambda s: s > best_score)

    weight_candidates = _generate_candidates(n_samples, rng_seed=rng_seed)
    original_weights = _sr.COMPOSITE_WEIGHTS

    for i, weights in enumerate(weight_candidates):
        if verbose and i > 0 and i % 50 == 0:
            print(f"  [{i}/{len(weight_candidates)}] ...")

        try:
            # Patch temporaire des poids dans le module skill_rating
            _sr.COMPOSITE_WEIGHTS = weights

            df_ratings = _sr.compute_skill_ratings_batch(df_matches, df_participants)

        except Exception as exc:
            logger.debug("Erreur avec poids #%d: %s", i, exc)
            continue
        finally:
            _sr.COMPOSITE_WEIGHTS = original_weights

        if df_ratings.is_empty():
            continue

        score = score_fn(df_ratings, team_mmr_map)

        if i == 0:
            default_score = score

        results.append({"weights": dict(weights), "score": score})

        if is_better(score):
            best_score = score
            best_weights = dict(weights)
            is_better = (
                (lambda s, ref=best_score: s < ref)
                if is_mae
                else (lambda s, ref=best_score: s > ref)
            )

    # Garantir que les poids par défaut sont restaurés en cas d'exception
    _sr.COMPOSITE_WEIGHTS = original_weights

    # Trier par score
    results.sort(key=lambda x: x["score"], reverse=not is_mae)

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


def _resolve_xuid_from_gamertag(gamertag: str, shared_db: Path) -> str | None:
    """Résout le XUID depuis le gamertag via xuid_aliases."""
    import duckdb

    if not shared_db.exists():
        return None
    conn = duckdb.connect(str(shared_db), read_only=True)
    try:
        row = conn.execute(
            "SELECT xuid FROM xuid_aliases WHERE LOWER(gamertag) = LOWER(?) LIMIT 1",
            [gamertag],
        ).fetchone()
        return str(row[0]) if row and row[0] else None
    except Exception:
        return None
    finally:
        conn.close()


def main(argv: list[str] | None = None) -> int:
    """Point d'entrée CLI.

    Usage:
        python -m src.analysis.skill_rating_calibration --player <Gamertag>
        python -m src.analysis.skill_rating_calibration --player <GT> --xuid <XUID>
        python -m src.analysis.skill_rating_calibration --player <GT> --n-samples 300 --metric corr
    """
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
    parser.add_argument("--shared-db", metavar="PATH", help="Chemin vers shared_matches.duckdb")
    parser.add_argument("--seed", type=int, default=42, help="Graine aléatoire (défaut: 42)")
    parser.add_argument("--quiet", action="store_true", help="Sortie minimale")

    args = parser.parse_args(argv)

    root = Path(__file__).resolve().parents[2]
    player_dir = root / "data" / "players" / args.player
    db_path = player_dir / "stats.duckdb"

    if not db_path.exists():
        print(f"Erreur : DB joueur introuvable : {db_path}", file=sys.stderr)
        return 1

    # Résolution XUID
    xuid = args.xuid
    if not xuid:
        shared_db_for_resolve = (
            Path(args.shared_db)
            if args.shared_db
            else (root / "data" / "warehouse" / "shared_matches.duckdb")
        )
        xuid = _resolve_xuid_from_gamertag(args.player, shared_db_for_resolve)

    if not xuid:
        print(
            f"Erreur : XUID introuvable pour « {args.player} ». "
            "Utilisez --xuid pour le fournir manuellement.",
            file=sys.stderr,
        )
        return 1

    verbose = not args.quiet

    try:
        result = calibrate_lusr_weights(
            db_path=db_path,
            xuid=xuid,
            shared_db_path=args.shared_db,
            n_samples=args.n_samples,
            min_matches_with_mmr=args.min_matches,
            metric=args.metric,
            rng_seed=args.seed,
            verbose=verbose,
        )
    except (FileNotFoundError, ValueError) as exc:
        print(f"Erreur : {exc}", file=sys.stderr)
        return 1

    # ── Affichage ──
    is_mae = result["metric"] == "mae"

    def _fmt(s: float) -> str:
        suffix = " (bas = meilleur)" if is_mae else " (haut = meilleur)"
        return f"{s:.4f}{suffix}"

    sep = "=" * 62
    print(f"\n{sep}")
    print("  CALIBRATION LUSR — RÉSULTATS")
    print(sep)
    print(f"  Joueur             : {args.player}")
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

    from src.analysis.skill_rating_config import COMPOSITE_WEIGHTS as _DEFAULT_W

    for key, val in result["best_weights"].items():
        default_val = _DEFAULT_W.get(key, 0.0)
        marker = "  # [modifié]" if abs(val - default_val) > 0.01 else ""
        print(f'      "{key}": {val:.4f},{marker}')
    print("  }")

    if not args.quiet and len(result["top_results"]) >= 3:
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
    return 0


if __name__ == "__main__":
    sys.exit(main())
