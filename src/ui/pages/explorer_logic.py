"""Logique pure pour la page Explorer.

Fonctions de filtrage, recherche floue et classification — testables
sans Streamlit ni accès DB.
"""

from __future__ import annotations

import difflib
from datetime import date

import polars as pl

# ---------------------------------------------------------------------------
# Recherche floue de gamertags
# ---------------------------------------------------------------------------


def fuzzy_search_gamertags(
    query: str,
    all_gamertags: list[str],
    n: int = 8,
    cutoff: float = 0.4,
) -> list[str]:
    """Recherche un gamertag par saisie approximative.

    Combine ``difflib.get_close_matches`` (distance SequenceMatcher) et
    un fallback substring insensible à la casse pour couvrir les cas
    où l'utilisateur tape un fragment exact (ex: "Pr0" → "Pr0gamer").

    Args:
        query: Saisie utilisateur (≥ 2 caractères recommandés).
        all_gamertags: Liste complète des gamertags connus.
        n: Nombre max de résultats.
        cutoff: Seuil de similarité pour difflib (0.0–1.0).

    Returns:
        Liste de gamertags correspondants, triée par pertinence.
    """
    if not query or len(query) < 2 or not all_gamertags:
        return []

    q_lower = query.strip().casefold()

    # 1) difflib — correspondance floue globale
    close = difflib.get_close_matches(query, all_gamertags, n=n, cutoff=cutoff)

    # 2) Fallback substring — match partiel exact
    substring_matches = [gt for gt in all_gamertags if q_lower in gt.casefold()]

    # Fusionner sans doublons, difflib en priorité
    seen: set[str] = set()
    result: list[str] = []
    for gt in close + substring_matches:
        if gt not in seen:
            seen.add(gt)
            result.append(gt)
        if len(result) >= n:
            break

    return result


# ---------------------------------------------------------------------------
# Filtrage des matchs
# ---------------------------------------------------------------------------


def filter_by_date(dff: pl.DataFrame, target: date) -> pl.DataFrame:
    """Filtre les matchs pour une date donnée.

    Args:
        dff: DataFrame avec colonne ``date`` (type ``Date``).
        target: Date cible.

    Returns:
        DataFrame filtré (peut être vide).
    """
    if "date" not in dff.columns or dff.is_empty():
        return dff.clear()
    return dff.filter(pl.col("date") == target)


def find_closest_date(dff: pl.DataFrame, target: date) -> date | None:
    """Trouve la date la plus proche ayant des matchs.

    Args:
        dff: DataFrame avec colonne ``date``.
        target: Date cible.

    Returns:
        La date la plus proche ou None si DataFrame vide.
    """
    if "date" not in dff.columns or dff.is_empty():
        return None
    dates = dff["date"].unique().sort()
    if dates.is_empty():
        return None
    # Calcul de la distance en jours (absolu)
    diffs = dates.cast(pl.Date).to_list()
    closest = min(diffs, key=lambda d: abs((d - target).days))
    return closest


def filter_by_squad(
    dff: pl.DataFrame,
    is_with_friends_map: dict[str, bool],
    mode: str,
) -> pl.DataFrame:
    """Filtre les matchs par type de groupe (solo/escouade/tous).

    Args:
        dff: DataFrame avec colonne ``match_id``.
        is_with_friends_map: Mapping match_id → is_with_friends.
        mode: ``"all"`` | ``"solo"`` | ``"squad"``.

    Returns:
        DataFrame filtré.
    """
    if mode == "all" or not is_with_friends_map:
        return dff
    if "match_id" not in dff.columns:
        return dff

    friends_series = (
        dff["match_id"]
        .cast(pl.Utf8)
        .map_elements(
            lambda mid: is_with_friends_map.get(str(mid)),
            return_dtype=pl.Boolean,
        )
    )
    if mode == "squad":
        return dff.filter(friends_series == True)  # noqa: E712
    if mode == "solo":
        return dff.filter(friends_series == False)  # noqa: E712
    return dff


def classify_experience_type(playlist_name: str) -> str:
    """Classifie une playlist en type d'expérience.

    Logique alignée sur ``_apply_experience_filter`` de ``_filters_cascade.py``.

    Args:
        playlist_name: Nom de la playlist (traduit ou non).

    Returns:
        ``"ranked"`` | ``"pve"`` | ``"unranked"``.
    """
    lower = (playlist_name or "").strip().lower()
    if "firefight" in lower or "baptême" in lower or "bapteme" in lower:
        return "pve"
    if "classé" in lower or "ranked" in lower:
        return "ranked"
    return "unranked"


def split_by_team(
    common_matches: pl.DataFrame,
) -> tuple[pl.DataFrame, pl.DataFrame]:
    """Sépare les matchs communs en alliés et adversaires.

    Args:
        common_matches: DataFrame avec colonnes player_team_id et target_team_id.

    Returns:
        Tuple (alliés, adversaires).
    """
    if common_matches.is_empty():
        return common_matches, common_matches
    same_team = common_matches.filter(pl.col("player_team_id") == pl.col("target_team_id"))
    diff_team = common_matches.filter(pl.col("player_team_id") != pl.col("target_team_id"))
    return same_team, diff_team


def get_distinct_dates(dff: pl.DataFrame) -> list[date]:
    """Retourne les dates uniques triées ayant des matchs.

    Args:
        dff: DataFrame avec colonne ``date``.

    Returns:
        Liste triée de dates.
    """
    if "date" not in dff.columns or dff.is_empty():
        return []
    return dff["date"].unique().sort().to_list()
