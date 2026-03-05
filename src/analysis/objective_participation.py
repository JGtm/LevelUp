"""Analyse de la participation aux objectifs avec Polars.

Sprint 4 - Fonctions d'analyse des personal_score_awards pour :
- Calculer les scores de participation aux objectifs
- Classer les joueurs par contribution aux objectifs
- Décomposer les types d'assistances

Ce module est un hub de réexport. L'implémentation est répartie dans :
- ``_objective_helpers.py`` : Helpers DRY (filtrage, catégories)
- ``_objective_profile.py`` : Profil joueur et ratio objectifs/kills (Sprint 7.4)
- ``_objective_summary.py`` : Résumés par match et fréquences d'awards

Références :
- src/data/domain/refdata.py : Enums et catégorisation des scores
- src/data/repositories/duckdb_repo.py : Chargement des données
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import TYPE_CHECKING

import polars as pl

from src.analysis._objective_helpers import (
    CATEGORY_IDS,
    compute_category_scores,
    filter_awards,
)
from src.analysis._objective_profile import (  # noqa: F401
    PlayerProfileResult,
    compute_objective_efficiency_polars,
    compute_objective_kill_ratio_polars,
    compute_player_profile_polars,
)
from src.analysis._objective_summary import (  # noqa: F401
    compute_award_frequency_polars,
    compute_objective_summary_by_match_polars,
    get_assist_awards_with_points,
    get_objective_mode_awards,
    is_objective_mode_match,
)
from src.data.domain.refdata import (
    ASSIST_SCORES,
    PersonalScoreNameId,
)

if TYPE_CHECKING:
    pass


# =============================================================================
# Dataclasses de résultats
# =============================================================================


@dataclass(frozen=True)
class ObjectiveParticipationResult:
    """Résultat du calcul de participation aux objectifs."""

    match_id: str | None
    xuid: str | None
    objective_score: int
    assist_score: int
    kill_score: int
    negative_score: int
    total_score: int
    objective_ratio: float
    assist_ratio: float
    objective_count: int
    assist_count: int
    kill_count: int


@dataclass(frozen=True)
class AssistBreakdownResult:
    """Décomposition détaillée des assistances."""

    kill_assists: int
    mark_assists: int
    emp_assists: int
    driver_assists: int
    sensor_assists: int
    flag_assists: int
    total_assists: int
    total_assist_points: int
    high_value_ratio: float


@dataclass(frozen=True)
class PlayerObjectiveRanking:
    """Classement d'un joueur par contribution aux objectifs."""

    xuid: str
    gamertag: str | None
    objective_score: int
    assist_score: int
    total_score: int
    matches_played: int
    avg_objective_per_match: float
    objective_focus_ratio: float


# =============================================================================
# Helpers internes
# =============================================================================

_EMPTY_PARTICIPATION = ObjectiveParticipationResult(
    match_id=None,
    xuid=None,
    objective_score=0,
    assist_score=0,
    kill_score=0,
    negative_score=0,
    total_score=0,
    objective_ratio=0.0,
    assist_ratio=0.0,
    objective_count=0,
    assist_count=0,
    kill_count=0,
)


def _empty_participation(
    match_id: str | None = None, xuid: str | None = None
) -> ObjectiveParticipationResult:
    """Retourne un résultat de participation vide."""
    if match_id is None and xuid is None:
        return _EMPTY_PARTICIPATION
    return ObjectiveParticipationResult(
        match_id=match_id,
        xuid=xuid,
        objective_score=0,
        assist_score=0,
        kill_score=0,
        negative_score=0,
        total_score=0,
        objective_ratio=0.0,
        assist_ratio=0.0,
        objective_count=0,
        assist_count=0,
        kill_count=0,
    )


_EMPTY_ASSIST = AssistBreakdownResult(
    kill_assists=0,
    mark_assists=0,
    emp_assists=0,
    driver_assists=0,
    sensor_assists=0,
    flag_assists=0,
    total_assists=0,
    total_assist_points=0,
    high_value_ratio=0.0,
)


# =============================================================================
# Fonctions d'analyse Polars
# =============================================================================


def compute_objective_participation_score_polars(
    awards_df: pl.DataFrame,
    match_id: str | None = None,
    xuid: str | None = None,
) -> ObjectiveParticipationResult:
    """Calcule le score de participation aux objectifs avec Polars.

    Args:
        awards_df: DataFrame Polars avec colonnes match_id, xuid,
                   award_name_id, count, total_points.
        match_id: Filtrer pour un match spécifique (optionnel).
        xuid: Filtrer pour un joueur spécifique (optionnel).

    Returns:
        ObjectiveParticipationResult avec scores détaillés.
    """
    if awards_df.is_empty():
        return _empty_participation(match_id, xuid)

    filtered_df = filter_awards(awards_df, match_id=match_id, xuid=xuid)
    if filtered_df.is_empty():
        return _empty_participation(match_id, xuid)

    scores = compute_category_scores(filtered_df)
    total = scores["total_score"]

    return ObjectiveParticipationResult(
        match_id=match_id,
        xuid=xuid,
        objective_score=scores["objective_score"],
        assist_score=scores["assist_score"],
        kill_score=scores["kill_score"],
        negative_score=scores["negative_score"],
        total_score=total,
        objective_ratio=scores["objective_score"] / total if total > 0 else 0.0,
        assist_ratio=scores["assist_score"] / total if total > 0 else 0.0,
        objective_count=scores["objective_count"],
        assist_count=scores["assist_count"],
        kill_count=scores["kill_count"],
    )


def _build_player_scores_df(
    filtered_df: pl.DataFrame, *, min_matches: int, top_n: int
) -> pl.DataFrame:
    """Construit le DataFrame agrégé des scores par joueur."""
    objective_ids = list(CATEGORY_IDS["objective"])
    assist_ids = list(CATEGORY_IDS["assist"])

    objective_by_player = (
        filtered_df.filter(pl.col("award_name_id").is_in(objective_ids))
        .group_by("xuid")
        .agg(
            pl.col("total_points").sum().alias("objective_score"),
            pl.col("match_id").n_unique().alias("objective_matches"),
        )
    )
    assist_by_player = (
        filtered_df.filter(pl.col("award_name_id").is_in(assist_ids))
        .group_by("xuid")
        .agg(pl.col("total_points").sum().alias("assist_score"))
    )
    total_by_player = filtered_df.group_by("xuid").agg(
        pl.col("total_points").sum().alias("total_score"),
        pl.col("match_id").n_unique().alias("matches_played"),
    )

    return (
        total_by_player.join(objective_by_player, on="xuid", how="left")
        .join(assist_by_player, on="xuid", how="left")
        .with_columns(
            pl.col("objective_score").fill_null(0),
            pl.col("assist_score").fill_null(0),
        )
        .filter(pl.col("matches_played") >= min_matches)
        .with_columns(
            (pl.col("objective_score") / pl.col("matches_played")).alias("avg_objective_per_match"),
            (pl.col("objective_score") / pl.col("total_score"))
            .fill_nan(0)
            .fill_null(0)
            .alias("objective_focus_ratio"),
        )
        .sort("avg_objective_per_match", descending=True)
        .head(top_n)
    )


def rank_players_by_objective_contribution_polars(
    awards_df: pl.DataFrame,
    *,
    match_ids: list[str] | None = None,
    top_n: int = 20,
    min_matches: int = 1,
) -> list[PlayerObjectiveRanking]:
    """Classe les joueurs par leur contribution aux objectifs.

    Args:
        awards_df: DataFrame Polars avec colonnes match_id, xuid,
                   award_name_id, count, total_points.
        match_ids: Liste de matchs à analyser (tous si None).
        top_n: Nombre de joueurs à retourner.
        min_matches: Nombre minimum de matchs pour être inclus.

    Returns:
        Liste de PlayerObjectiveRanking triée par contribution.
    """
    if awards_df.is_empty():
        return []

    filtered_df = awards_df
    if match_ids:
        filtered_df = filtered_df.filter(pl.col("match_id").is_in(match_ids))
    if filtered_df.is_empty():
        return []

    result_df = _build_player_scores_df(filtered_df, min_matches=min_matches, top_n=top_n)

    return [
        PlayerObjectiveRanking(
            xuid=row["xuid"],
            gamertag=None,
            objective_score=int(row["objective_score"]),
            assist_score=int(row["assist_score"]),
            total_score=int(row["total_score"]),
            matches_played=int(row["matches_played"]),
            avg_objective_per_match=float(row["avg_objective_per_match"]),
            objective_focus_ratio=float(row["objective_focus_ratio"]),
        )
        for row in result_df.iter_rows(named=True)
    ]


def compute_assist_breakdown_polars(
    awards_df: pl.DataFrame,
    *,
    match_id: str | None = None,
    xuid: str | None = None,
) -> AssistBreakdownResult:
    """Décompose les types d'assistances avec Polars.

    Args:
        awards_df: DataFrame Polars avec colonnes match_id, xuid,
                   award_name_id, count, total_points.
        match_id: Filtrer pour un match spécifique (optionnel).
        xuid: Filtrer pour un joueur spécifique (optionnel).

    Returns:
        AssistBreakdownResult avec détail des assistances.
    """
    if awards_df.is_empty():
        return _EMPTY_ASSIST

    filtered_df = filter_awards(awards_df, match_id=match_id, xuid=xuid)

    assist_ids = list(ASSIST_SCORES)
    assist_df = filtered_df.filter(pl.col("award_name_id").is_in(assist_ids))
    if assist_df.is_empty():
        return _EMPTY_ASSIST

    # Mapper les IDs aux noms des colonnes
    assist_mapping = {
        PersonalScoreNameId.KILL_ASSIST: "kill_assists",
        PersonalScoreNameId.MARK_ASSIST: "mark_assists",
        PersonalScoreNameId.EMP_ASSIST: "emp_assists",
        PersonalScoreNameId.DRIVER_ASSIST: "driver_assists",
        PersonalScoreNameId.SENSOR_ASSIST: "sensor_assists",
        PersonalScoreNameId.FLAG_CAPTURE_ASSIST: "flag_assists",
    }

    counts = dict.fromkeys(assist_mapping.values(), 0)
    total_points = 0

    for row in assist_df.iter_rows(named=True):
        award_id = row["award_name_id"]
        for score_id, col_name in assist_mapping.items():
            if award_id == int(score_id):
                counts[col_name] += row["count"]
                break
        total_points += row["total_points"]

    total_assists = sum(counts.values())

    # Haute valeur : 50+ pts (KILL_ASSIST, EMP_ASSIST, DRIVER_ASSIST, FLAG_CAPTURE_ASSIST)
    high_value_count = (
        counts["kill_assists"]
        + counts["emp_assists"]
        + counts["driver_assists"]
        + counts["flag_assists"]
    )

    return AssistBreakdownResult(
        kill_assists=counts["kill_assists"],
        mark_assists=counts["mark_assists"],
        emp_assists=counts["emp_assists"],
        driver_assists=counts["driver_assists"],
        sensor_assists=counts["sensor_assists"],
        flag_assists=counts["flag_assists"],
        total_assists=total_assists,
        total_assist_points=total_points,
        high_value_ratio=high_value_count / total_assists if total_assists > 0 else 0.0,
    )
