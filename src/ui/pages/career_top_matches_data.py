"""Page Carrière — Chargement Top 10 meilleurs / pires matchs."""

from __future__ import annotations

import logging

from src.data.repositories._career_encounters_repo import (
    _BADGE_PRIORITY_EXPR,
    _BTB_FILTER_SQL,
    _TOP_MATCHES_SQL,
    MIN_MATCH_DURATION_SECONDS,
)

__all__ = [
    "MIN_MATCH_DURATION_SECONDS",
    "_BADGE_PRIORITY_EXPR",
    "_BTB_FILTER_SQL",
    "_TOP_MATCHES_SQL",
    "_load_top_matches",
]

logger = logging.getLogger(__name__)


def _load_top_matches(
    db_path: str,
    xuid: str,
    *,
    best: bool,
    exclude_btb: bool = False,
) -> list[dict]:
    """Charge le Top 10 meilleurs ou pires matchs.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        xuid: XUID du joueur.
        best: True pour les meilleures perf, False pour les pires.
        exclude_btb: Si True, exclut les matchs BTB.

    Returns:
        Liste de dicts avec les colonnes du match.
    """
    from src.ui._cache_core import get_cached_repository_st

    repo = get_cached_repository_st(db_path, xuid)
    matches = repo.load_top_match_list(best=best, exclude_btb=exclude_btb)
    logger.debug(
        "Top matches chargés (best=%s, exclude_btb=%s): %d résultats",
        best,
        exclude_btb,
        len(matches),
    )
    return matches


def load_top_best_matches(db_path: str, xuid: str, *, exclude_btb: bool = False) -> list[dict]:
    """Top 10 meilleures performances (victoires dominantes)."""
    return _load_top_matches(db_path, xuid, best=True, exclude_btb=exclude_btb)


def load_top_worst_matches(db_path: str, xuid: str, *, exclude_btb: bool = False) -> list[dict]:
    """Top 10 pires performances (défaites humiliantes)."""
    return _load_top_matches(db_path, xuid, best=False, exclude_btb=exclude_btb)
