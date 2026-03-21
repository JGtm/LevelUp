"""Page Carrière — Chargement des données rencontres et antagonistes."""

from __future__ import annotations

import logging
from datetime import datetime

logger = logging.getLogger(__name__)


def _load_top_encountered(
    xuid: str,
    db_path: str,
    limit: int = 10,
    exclude_xuids: set[str] | None = None,
    *,
    since: datetime | None = None,
) -> list[dict]:
    """Charge les joueurs les plus croisés depuis shared_matches.duckdb."""
    from src.ui._cache_core import get_cached_repository_st

    repo = get_cached_repository_st(db_path, xuid)
    return repo.load_top_encountered(limit=limit, exclude_xuids=exclude_xuids, since=since)


def _load_top_nemeses(
    xuid: str,
    db_path: str,
    limit: int = 10,
    exclude_xuids: set[str] | None = None,
    *,
    since: datetime | None = None,
) -> list[dict]:
    """Charge les adversaires qui nous tuent le plus (némésis)."""
    from src.ui._cache_core import get_cached_repository_st

    repo = get_cached_repository_st(db_path, xuid)
    return repo.load_antagonists(
        mode="nemesis", limit=limit, exclude_xuids=exclude_xuids, since=since
    )


def _load_top_victims(
    xuid: str,
    db_path: str,
    limit: int = 10,
    exclude_xuids: set[str] | None = None,
    *,
    since: datetime | None = None,
) -> list[dict]:
    """Charge les adversaires qu'on tue le plus (souffre-douleurs)."""
    from src.ui._cache_core import get_cached_repository_st

    repo = get_cached_repository_st(db_path, xuid)
    return repo.load_antagonists(
        mode="victim", limit=limit, exclude_xuids=exclude_xuids, since=since
    )
