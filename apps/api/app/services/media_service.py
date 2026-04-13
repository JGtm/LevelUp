"""Service Médiathèque — chargement, filtrage, tri et pagination des médias."""

from __future__ import annotations

import logging
from datetime import datetime
from pathlib import Path

from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.common import PaginatedResponse, PaginationMeta
from apps.api.app.schemas.media import (
    MediaItemRow,
    MediaPageResponse,
    MediaQueryRequest,
)

logger = logging.getLogger(__name__)

_VALID_SORTS = {"date_desc", "date_asc"}
_VALID_KINDS = {"screenshot", "video"}
_VALID_SECTIONS = {"mine", "teammate", "unassigned"}


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def get_media_page(player: PlayerContext, request: MediaQueryRequest) -> MediaPageResponse:
    """Retourne la page médiathèque avec pagination et filtres."""
    raw_df = _load_media_raw(player)

    if raw_df is None or (hasattr(raw_df, "is_empty") and raw_df.is_empty()):
        return MediaPageResponse(
            items=PaginatedResponse(items=[], total=0, page=1, page_size=50),
            total_mine=0,
            total_teammates=0,
            total_unassigned=0,
        )

    # Comptages par section avant filtrage
    total_mine = _count_section(raw_df, "mine")
    total_teammates = _count_section(raw_df, "teammate")
    total_unassigned = _count_section(raw_df, "unassigned")

    df = _apply_kind_filter(raw_df, request.kind_filter)
    df = _apply_section_filter(df, request.section_filter)
    df = _sort_df(df, request.sort or "date_desc")
    df, total = _paginate(df, request.pagination)

    items = [_row_to_schema(r) for r in df.iter_rows(named=True)]

    page_num = 1
    page_size = 50
    if request.pagination:
        page_num = request.pagination.page or 1
        page_size = request.pagination.page_size or 50

    return MediaPageResponse(
        items=PaginatedResponse(
            items=items,
            pagination=PaginationMeta(
                total=total,
                page=page_num,
                page_size=page_size,
                has_next=(page_num * page_size) < total,
                has_prev=page_num > 1,
            ),
        ),
        total_mine=total_mine,
        total_teammates=total_teammates,
        total_unassigned=total_unassigned,
    )


# ---------------------------------------------------------------------------
# Chargement
# ---------------------------------------------------------------------------


def _load_media_raw(player: PlayerContext):
    """Charge les médias via MediaIndexer.load_media_for_ui."""
    try:
        from src.data.media_indexer import MediaIndexer

        db_path = Path(player.db_path)
        return MediaIndexer.load_media_for_ui(db_path, player.xuid or "")
    except Exception:
        logger.debug("_load_media_raw(%s): erreur", player.player_slug, exc_info=True)
        try:
            import polars as pl

            return pl.DataFrame()
        except ImportError:
            return None


# ---------------------------------------------------------------------------
# Filtres
# ---------------------------------------------------------------------------


def _apply_kind_filter(df, kind_filter: str | None):
    """Filtre par type de média (screenshot/video)."""
    try:
        import polars as pl

        if not kind_filter or kind_filter not in _VALID_KINDS:
            return df
        if "kind" not in df.columns:
            return df
        return df.filter(pl.col("kind").cast(pl.Utf8) == kind_filter)
    except Exception:
        return df


def _apply_section_filter(df, section_filter: str | None):
    """Filtre par section (mine/teammate/unassigned)."""
    try:
        import polars as pl

        if not section_filter or section_filter not in _VALID_SECTIONS:
            return df
        if "section" not in df.columns:
            return df
        return df.filter(pl.col("section").cast(pl.Utf8) == section_filter)
    except Exception:
        return df


# ---------------------------------------------------------------------------
# Tri
# ---------------------------------------------------------------------------


def _sort_df(df, sort: str):
    """Trie le DataFrame par date de capture."""
    try:
        col_name = "capture_end_utc"
        if col_name not in df.columns:
            return df
        descending = sort != "date_asc"
        return df.sort(col_name, descending=descending, nulls_last=True)
    except Exception:
        return df


# ---------------------------------------------------------------------------
# Pagination
# ---------------------------------------------------------------------------


def _paginate(df, pagination):
    """Applique la pagination et retourne (df_page, total)."""
    try:
        total = len(df)
        if pagination is None:
            return df.head(50), total
        page = max(1, int(pagination.page or 1))
        page_size = max(1, int(pagination.page_size or 50))
        offset = (page - 1) * page_size
        return df.slice(offset, page_size), total
    except Exception:
        return df.head(50), len(df)


# ---------------------------------------------------------------------------
# Conversion ligne → schéma
# ---------------------------------------------------------------------------


def _row_to_schema(row: dict) -> MediaItemRow:
    """Convertit une ligne brute Polars en MediaItemRow."""

    def _dt(val) -> datetime | None:
        if val is None:
            return None
        if isinstance(val, datetime):
            return val
        if isinstance(val, str):
            try:
                return datetime.fromisoformat(val)
            except Exception:
                return None
        return None

    basename = str(row.get("file_name") or row.get("basename") or "")
    file_path = str(row.get("file_path") or "")
    kind = str(row.get("kind") or "screenshot")
    thumbnail_path = row.get("thumbnail_path")
    match_id = str(row.get("match_id") or "").strip() or None
    section = str(row.get("section") or "mine")
    owner_gamertag = str(row.get("gamertag") or row.get("owner_gamertag") or "")
    map_name = str(row.get("map_name") or "")

    return MediaItemRow(
        basename=basename,
        file_path=file_path,
        kind=kind,
        thumbnail_path=thumbnail_path,
        match_id=match_id,
        capture_end_utc=_dt(row.get("capture_end_utc")),
        match_start_time=_dt(row.get("match_start_time")),
        section=section,
        owner_gamertag=owner_gamertag or None,
        map_name=map_name or None,
    )


# ---------------------------------------------------------------------------
# Comptage par section
# ---------------------------------------------------------------------------


def _count_section(df, section: str) -> int:
    """Compte les entrées d'une section dans le DataFrame."""
    try:
        import polars as pl

        if "section" not in df.columns:
            return 0
        return int(df.filter(pl.col("section").cast(pl.Utf8) == section).height)
    except Exception:
        return 0
