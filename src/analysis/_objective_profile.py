"""Analyse de profil joueur et ratio objectifs/kills (Sprint 7.4).

Fonctions pour déterminer le profil de jeu (slayer/support/versatile)
et calculer l'efficacité objectifs du joueur.
"""

from __future__ import annotations

from dataclasses import dataclass

import polars as pl

from src.analysis._objective_helpers import (
    CATEGORY_IDS,
    compute_category_scores,
    filter_awards,
)


@dataclass(frozen=True)
class PlayerProfileResult:
    """Profil du joueur basé sur ses contributions.

    Attributes:
        profile_type: Type de profil ("slayer", "support", "versatile").
        profile_label: Label humain du profil.
        objective_ratio: Ratio objectifs/total (0-1).
        kill_ratio: Ratio kills/total (0-1).
        assist_ratio: Ratio assistances/total (0-1).
        confidence: Niveau de confiance (basé sur le nombre de matchs).
        description: Description du profil.
    """

    profile_type: str
    profile_label: str
    objective_ratio: float
    kill_ratio: float
    assist_ratio: float
    confidence: str
    description: str


_DEFAULT_PROFILE = PlayerProfileResult(
    profile_type="unknown",
    profile_label="Inconnu",
    objective_ratio=0.0,
    kill_ratio=0.0,
    assist_ratio=0.0,
    confidence="faible",
    description="Pas assez de données pour déterminer le profil.",
)


def _confidence_level(matches_count: int) -> str:
    """Retourne le niveau de confiance en fonction du nombre de matchs."""
    if matches_count >= 50:
        return "élevée"
    if matches_count >= 20:
        return "moyenne"
    return "faible"


def _classify_profile(
    kill_ratio: float,
    objective_ratio: float,
    assist_ratio: float,
) -> tuple[str, str, str]:
    """Classifie le profil du joueur.

    Returns:
        Tuple (profile_type, profile_label, description).
    """
    support_ratio = objective_ratio + assist_ratio

    if kill_ratio >= 0.60:
        return (
            "slayer",
            "🎯 Joueur Slayer",
            f"Vous excellez dans les éliminations avec {kill_ratio * 100:.0f}% "
            "de votre score provenant des kills.",
        )
    if support_ratio >= 0.40:
        return (
            "support",
            "🛡️ Joueur Support",
            f"Vous contribuez fortement aux objectifs ({objective_ratio * 100:.0f}%) "
            f"et aux assistances ({assist_ratio * 100:.0f}%).",
        )
    return (
        "versatile",
        "⚔️ Joueur Polyvalent",
        f"Bon équilibre entre kills ({kill_ratio * 100:.0f}%), "
        f"objectifs ({objective_ratio * 100:.0f}%) et assists ({assist_ratio * 100:.0f}%).",
    )


def compute_objective_kill_ratio_polars(
    awards_df: pl.DataFrame,
    match_stats_df: pl.DataFrame,
    *,
    xuid: str | None = None,
) -> pl.DataFrame:
    """Calcule le ratio objectifs/kills par match.

    Cette fonction compare le score objectifs au nombre de kills pour
    identifier les matchs où le joueur a plus contribué aux objectifs.

    Args:
        awards_df: DataFrame des personal_score_awards.
        match_stats_df: DataFrame des match_stats avec kills.
        xuid: Filtrer pour un joueur spécifique (optionnel).

    Returns:
        DataFrame Polars avec match_id, objective_score, kills,
        objective_per_kill, is_objective_focused.
    """
    if awards_df.is_empty() or match_stats_df.is_empty():
        return pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "objective_score": pl.Int64,
                "kills": pl.Int64,
                "objective_per_kill": pl.Float64,
                "is_objective_focused": pl.Boolean,
            }
        )

    filtered_awards = filter_awards(awards_df, xuid=xuid)

    # Calculer score objectifs par match
    objective_ids = list(CATEGORY_IDS["objective"])
    obj_by_match = (
        filtered_awards.filter(pl.col("award_name_id").is_in(objective_ids))
        .group_by("match_id")
        .agg(pl.col("total_points").sum().alias("objective_score"))
    )

    return (
        match_stats_df.select(["match_id", "kills"])
        .join(obj_by_match, on="match_id", how="left")
        .with_columns([pl.col("objective_score").fill_null(0)])
        .with_columns(
            [
                (
                    pl.col("objective_score")
                    / pl.when(pl.col("kills") == 0).then(1).otherwise(pl.col("kills"))
                ).alias("objective_per_kill"),
            ]
        )
        .with_columns([(pl.col("objective_per_kill") > 50).alias("is_objective_focused")])
    )


def compute_player_profile_polars(
    awards_df: pl.DataFrame,
    *,
    xuid: str | None = None,
    min_matches: int = 5,
) -> PlayerProfileResult:
    """Détermine le profil du joueur basé sur ses contributions.

    Analyse les personal_score_awards pour classifier le joueur en :
    - "slayer" : Orienté kills (ratio kills > 60%)
    - "support" : Orienté objectifs/assists (ratio objectifs+assists > 40%)
    - "versatile" : Équilibré

    Args:
        awards_df: DataFrame des personal_score_awards.
        xuid: XUID du joueur (optionnel, prend tous si non spécifié).
        min_matches: Nombre minimum de matchs pour une analyse fiable.

    Returns:
        PlayerProfileResult avec le profil déterminé.
    """
    if awards_df.is_empty():
        return _DEFAULT_PROFILE

    filtered_df = filter_awards(awards_df, xuid=xuid)
    if filtered_df.is_empty():
        return _DEFAULT_PROFILE

    matches_count = filtered_df.select(pl.col("match_id").n_unique()).item()
    if matches_count < min_matches:
        return PlayerProfileResult(
            profile_type="unknown",
            profile_label="Données insuffisantes",
            objective_ratio=0.0,
            kill_ratio=0.0,
            assist_ratio=0.0,
            confidence="faible",
            description=f"Seulement {matches_count} matchs analysés (min: {min_matches}).",
        )

    scores = compute_category_scores(filtered_df)
    total = scores["total_score"]
    if total == 0:
        return _DEFAULT_PROFILE

    obj_r = scores["objective_score"] / total
    assist_r = scores["assist_score"] / total
    kill_r = scores["kill_score"] / total

    ptype, plabel, desc = _classify_profile(kill_r, obj_r, assist_r)

    return PlayerProfileResult(
        profile_type=ptype,
        profile_label=plabel,
        objective_ratio=round(obj_r, 3),
        kill_ratio=round(kill_r, 3),
        assist_ratio=round(assist_r, 3),
        confidence=_confidence_level(matches_count),
        description=desc,
    )


def compute_objective_efficiency_polars(
    awards_df: pl.DataFrame,
    match_stats_df: pl.DataFrame,
    *,
    xuid: str | None = None,
) -> dict[str, float | None]:
    """Calcule l'efficacité objective du joueur.

    Mesure le ratio entre les points d'objectifs et les ressources investies
    (temps de jeu, morts, etc.).

    Args:
        awards_df: DataFrame des personal_score_awards.
        match_stats_df: DataFrame des match_stats.
        xuid: XUID du joueur (optionnel).

    Returns:
        Dict avec les métriques d'efficacité :
        - objective_per_minute: Points objectifs par minute de jeu.
        - objective_per_death: Points objectifs par mort.
        - objective_contribution_pct: % de la contribution aux objectifs de l'équipe.
    """
    default_result: dict[str, float | None] = {
        "objective_per_minute": None,
        "objective_per_death": None,
        "objective_contribution_pct": None,
    }

    if awards_df.is_empty() or match_stats_df.is_empty():
        return default_result

    filtered_awards = filter_awards(awards_df, xuid=xuid)

    objective_ids = list(CATEGORY_IDS["objective"])
    total_objective = (
        filtered_awards.filter(pl.col("award_name_id").is_in(objective_ids))
        .select(pl.col("total_points").sum())
        .item()
    ) or 0

    if total_objective == 0:
        return default_result

    total_time_seconds = (match_stats_df.select(pl.col("time_played_seconds").sum()).item()) or 0
    total_deaths = (match_stats_df.select(pl.col("deaths").sum()).item()) or 0

    objective_per_minute = (
        total_objective / (total_time_seconds / 60) if total_time_seconds > 0 else None
    )
    objective_per_death = (
        total_objective / total_deaths if total_deaths > 0 else float(total_objective)
    )

    return {
        "objective_per_minute": (round(objective_per_minute, 1) if objective_per_minute else None),
        "objective_per_death": (round(objective_per_death, 1) if objective_per_death else None),
        "objective_contribution_pct": None,
    }
