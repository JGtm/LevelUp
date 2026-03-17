"""Chargement de données et logique temporelle pour la bibliothèque médias.

Fonctions extraites de ``media_library.py`` pour séparer accès BDD / logique
d'association et rendu Streamlit.
"""

from __future__ import annotations

import logging
import os
from datetime import datetime, timedelta

import polars as pl

from src.ui.pages.match_view_helpers import index_media_dir
from src.ui.pages.media_library_temporal import (  # noqa: F401
    epoch_seconds_paris,
    to_paris_naive,
)
from src.ui.settings import AppSettings
from src.visualization._compat import DataFrameLike, ensure_polars

logger = logging.getLogger(__name__)

# ------------------------------------------------------------------
# Fenêtres temporelles des matchs
# ------------------------------------------------------------------


def compute_match_windows(df_full: DataFrameLike, settings: AppSettings) -> pl.DataFrame:
    """Construit les fenêtres temporelles des matchs (epoch seconds) pour l'association média."""
    _empty = pl.DataFrame(
        schema={
            "match_id": pl.Utf8,
            "start_epoch": pl.Float64,
            "end_epoch": pl.Float64,
            "start_time": pl.Datetime,
        }
    )
    if df_full is None:
        return _empty
    df_full = ensure_polars(df_full)
    if df_full.is_empty():
        return _empty

    tol_min = settings.media_tolerance_minutes
    tol = timedelta(minutes=max(0, tol_min))

    needed = {"match_id", "start_time"}
    if not needed.issubset(set(df_full.columns)):
        return _empty

    cols = [c for c in ["match_id", "start_time", "time_played_seconds"] if c in df_full.columns]
    base = df_full.select(cols)

    rows: list[dict[str, object]] = []
    for rec in base.iter_rows(named=True):
        start = to_paris_naive(rec.get("start_time"))
        if not isinstance(start, datetime):
            continue
        dur_s = rec.get("time_played_seconds")
        try:
            dur = float(dur_s) if dur_s is not None else None
        except Exception:
            dur = None
        if dur is None or dur <= 0:
            end = start + timedelta(minutes=12)
        else:
            end = start + timedelta(seconds=dur)
        t0 = start - tol
        t1 = end + tol
        se = epoch_seconds_paris(t0)
        ee = epoch_seconds_paris(t1)
        mid = rec.get("match_id")
        if mid is None or se is None or ee is None:
            continue
        rows.append(
            {
                "match_id": str(mid),
                "start_epoch": se,
                "end_epoch": ee,
                "start_time": start,
            }
        )

    if not rows:
        return _empty

    return pl.DataFrame(rows).sort("start_epoch")


# ------------------------------------------------------------------
# Indexation disque
# ------------------------------------------------------------------


def index_all_media(settings: AppSettings) -> pl.DataFrame:
    """Indexe les médias configurés (captures + vidéos) depuis le disque."""
    screens_dir = settings.media_screens_dir.strip()
    videos_dir = settings.media_videos_dir.strip()
    frames: list[pl.DataFrame] = []

    if screens_dir and os.path.isdir(screens_dir):
        img_df = index_media_dir(screens_dir, ("png", "jpg", "jpeg", "webp"))
        if not img_df.is_empty():
            img_df = img_df.with_columns(pl.lit("image").alias("kind"))
            frames.append(img_df)

    if videos_dir and os.path.isdir(videos_dir):
        vid_df = index_media_dir(videos_dir, ("mp4", "webm", "mkv", "mov"))
        if not vid_df.is_empty():
            vid_df = vid_df.with_columns(pl.lit("video").alias("kind"))
            frames.append(vid_df)

    if not frames:
        return pl.DataFrame(
            schema={"path": pl.Utf8, "mtime": pl.Float64, "ext": pl.Utf8, "kind": pl.Utf8}
        )

    df = pl.concat(frames)
    if df.is_empty():
        return df

    df = df.with_columns(
        [
            pl.col("path").cast(pl.Utf8),
            pl.col("path")
            .cast(pl.Utf8)
            .str.replace_all(r"\\", "/")
            .str.split("/")
            .list.last()
            .alias("basename"),
        ]
    ).drop_nulls(subset=["mtime"])
    return df.sort("mtime", descending=True)


# ------------------------------------------------------------------
# Association média → match
# ------------------------------------------------------------------


def associate_media_to_matches(media_df: pl.DataFrame, windows_df: pl.DataFrame) -> pl.DataFrame:
    """Associe chaque média à un match (best-effort) via join_asof + check de fenêtre."""
    if media_df is None or media_df.is_empty():
        extra_cols = {"match_id": pl.Utf8, "match_start_time": pl.Datetime}
        if media_df is not None:
            schema = {
                **{c: media_df.dtypes[i] for i, c in enumerate(media_df.columns)},
                **extra_cols,
            }
            return pl.DataFrame(schema=schema)
        return pl.DataFrame()

    if windows_df is None or windows_df.is_empty():
        return media_df.with_columns(
            [
                pl.lit(None).cast(pl.Utf8).alias("match_id"),
                pl.lit(None).cast(pl.Datetime).alias("match_start_time"),
            ]
        )

    m = media_df.drop_nulls(subset=["mtime"]).sort("mtime")
    w = windows_df.sort("start_epoch")

    joined = m.join_asof(
        w,
        left_on="mtime",
        right_on="start_epoch",
        strategy="nearest",
    )

    joined = joined.with_columns(
        [
            pl.when(
                pl.col("start_epoch").is_not_null()
                & pl.col("end_epoch").is_not_null()
                & (pl.col("mtime") >= pl.col("start_epoch"))
                & (pl.col("mtime") <= pl.col("end_epoch"))
            )
            .then(pl.col("match_id"))
            .otherwise(pl.lit(None))
            .alias("match_id"),
            pl.when(
                pl.col("start_epoch").is_not_null()
                & pl.col("end_epoch").is_not_null()
                & (pl.col("mtime") >= pl.col("start_epoch"))
                & (pl.col("mtime") <= pl.col("end_epoch"))
            )
            .then(pl.col("start_time"))
            .otherwise(pl.lit(None))
            .alias("start_time"),
        ]
    )

    drop_cols = [c for c in ["start_epoch", "end_epoch"] if c in joined.columns]
    joined = joined.drop(drop_cols)
    if "start_time" in joined.columns:
        joined = joined.rename({"start_time": "match_start_time"})

    return joined.sort("mtime", descending=True)


# ------------------------------------------------------------------
# Chargement depuis BDD DuckDB
# ------------------------------------------------------------------


def load_match_windows_from_db(db_path: str, xuid: str = "") -> pl.DataFrame:  # noqa: C901, PLR0912, PLR0915
    """Charge les fenêtres temporelles des matchs depuis la DB.

    V5 : Utilise shared_matches.duckdb via le repository.
    """
    _empty = pl.DataFrame(
        schema={
            "match_id": pl.Utf8,
            "start_epoch": pl.Float64,
            "end_epoch": pl.Float64,
            "start_time": pl.Datetime,
        }
    )
    try:
        from src.ui._cache_core import get_cached_repository_st

        repo = get_cached_repository_st(db_path, xuid)
        matches = repo.load_match_registry_raw()

        if matches:
            all_windows: list[dict[str, object]] = []
            for match_id, start_time, duration in matches:
                try:
                    if isinstance(start_time, datetime):
                        dt_start = start_time
                    elif isinstance(start_time, str):
                        if start_time.endswith("Z"):
                            dt_start = datetime.fromisoformat(start_time[:-1] + "+00:00")
                        elif "+" in start_time or start_time.count("-") > 2:
                            dt_start = datetime.fromisoformat(start_time)
                        else:
                            dt_start = datetime.fromisoformat(start_time + "+00:00")
                    else:
                        continue

                    start_epoch = epoch_seconds_paris(dt_start)
                    if start_epoch is None:
                        continue

                    dur = float(duration or 0) if duration else 12 * 60
                    end_epoch = start_epoch + dur

                    all_windows.append(
                        {
                            "match_id": str(match_id),
                            "start_epoch": start_epoch,
                            "end_epoch": end_epoch,
                            "start_time": dt_start,
                        }
                    )
                except Exception as e:
                    logger.debug("media_library: erreur parsing window entry: %s", e)
                    continue

            if all_windows:
                return pl.DataFrame(all_windows).sort("start_epoch")

        return _empty

    except Exception as e:
        logger.debug("media_library: load_match_windows échoué: %s", e)
        return _empty


def gamertag_from_db_path(db_path: str) -> str | None:
    """Extrait le gamertag (nom du dossier joueur) depuis le chemin de la DB."""
    if not db_path:
        return None
    try:
        from pathlib import Path

        p = Path(db_path)
        if p.name and p.name.endswith(".duckdb"):
            return p.parent.name or None
        return None
    except Exception:
        return None


def load_media_from_db(
    db_path: str,
    xuid: str | None = None,
    gamertag: str | None = None,
) -> pl.DataFrame:
    """Charge les médias depuis la BDD DuckDB.

    Args:
        db_path: Chemin vers la DB DuckDB.
        xuid: XUID du joueur pour filtrer les associations (optionnel).
        gamertag: Gamertag (nom du dossier) pour inclure les associations fallback.

    Returns:
        DataFrame avec colonnes: path, mtime, ext, kind, basename, match_id, match_start_time, xuid
    """
    _col_names = [
        "path",
        "mtime",
        "mtime_paris_epoch",
        "ext",
        "kind",
        "basename",
        "thumbnail_path",
        "match_id",
        "match_start_time",
        "association_confidence",
        "xuid",
    ]
    try:
        from src.ui._cache_core import get_cached_repository_st

        repo = get_cached_repository_st(db_path, xuid or "")
        result = repo.load_media_files_raw(xuid, gamertag)
        if result is None:
            return pl.DataFrame()
        if not result:
            return pl.DataFrame()
        rows = [dict(zip(_col_names, row, strict=False)) for row in result]
        return pl.DataFrame(rows)

    except Exception as e:
        logger.debug("media_library: load_media_from_db échoué: %s", e)
        return pl.DataFrame()
