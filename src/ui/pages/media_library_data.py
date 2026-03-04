"""Chargement de données et logique temporelle pour la bibliothèque médias.

Fonctions extraites de ``media_library.py`` pour séparer accès BDD / logique
d'association et rendu Streamlit.
"""

from __future__ import annotations

import os
from datetime import datetime, timedelta

import polars as pl

from src.ui.formatting import PARIS_TZ
from src.ui.pages.match_view_helpers import index_media_dir
from src.ui.settings import AppSettings
from src.visualization._compat import DataFrameLike, ensure_polars

# ------------------------------------------------------------------
# Utilitaires temporels
# ------------------------------------------------------------------


def epoch_seconds_paris(dt_value: datetime | None) -> float | None:
    """Convertit un datetime en secondes epoch (fuseau Paris)."""
    if dt_value is None:
        return None
    try:
        aware = (
            PARIS_TZ.localize(dt_value)
            if dt_value.tzinfo is None
            else dt_value.astimezone(PARIS_TZ)
        )
        return float(aware.timestamp())
    except Exception:
        return None


def to_paris_naive(dt_value: object) -> datetime | None:
    """Convertit une valeur datetime en datetime naïve (fuseau Paris)."""
    try:
        if dt_value is None:
            return None
        if isinstance(dt_value, datetime):
            ts = dt_value
        elif isinstance(dt_value, str):
            s = str(dt_value).strip()
            if not s:
                return None
            if s.endswith("Z"):
                s = s[:-1] + "+00:00"
            ts = datetime.fromisoformat(s)
        else:
            return None
        if ts.tzinfo is None:
            return ts
        return ts.astimezone(PARIS_TZ).replace(tzinfo=None)
    except Exception:
        return None


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

    tol_min = int(getattr(settings, "media_tolerance_minutes", 0) or 0)
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
    screens_dir = str(getattr(settings, "media_screens_dir", "") or "").strip()
    videos_dir = str(getattr(settings, "media_videos_dir", "") or "").strip()
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


def load_match_windows_from_db(db_path: str) -> pl.DataFrame:  # noqa: C901, PLR0912, PLR0915
    """Charge les fenêtres temporelles des matchs depuis la DB.

    V5 : Utilise shared_matches.duckdb si disponible.
    Fallback v4 : Parcourt les DBs joueurs individuelles.
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
        from src.utils.db import duckdb_read_only
        from src.utils.paths import PLAYERS_DIR

        # --- V5 : requête unique via shared_matches.duckdb ---
        shared_db = PLAYERS_DIR.parent / "warehouse" / "shared_matches.duckdb"
        if shared_db.exists():
            try:
                with duckdb_read_only(shared_db) as conn:
                    matches = conn.execute(
                        """
                        SELECT match_id, start_time, duration_seconds
                        FROM match_registry
                        WHERE start_time IS NOT NULL
                        """
                    ).fetchall()

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
                        except Exception:
                            continue

                    if all_windows:
                        return pl.DataFrame(all_windows).sort("start_epoch")
            except Exception:
                pass  # Fallback vers v4

        # --- Fallback V4 : parcourir les DBs joueurs ---
        from src.utils.paths import PLAYER_DB_FILENAME

        all_windows = []

        if PLAYERS_DIR.exists():
            for player_dir in PLAYERS_DIR.iterdir():
                if not player_dir.is_dir():
                    continue

                player_db = player_dir / PLAYER_DB_FILENAME
                if not player_db.exists():
                    continue

                try:
                    with duckdb_read_only(player_db) as conn:
                        from src.utils.db import has_table

                        if not has_table(conn, "match_stats"):
                            continue

                        matches = conn.execute(
                            """
                            SELECT match_id, start_time, time_played_seconds
                            FROM match_stats
                            WHERE start_time IS NOT NULL
                            """
                        ).fetchall()

                        if matches:
                            for match_id, start_time, time_played in matches:
                                try:
                                    if isinstance(start_time, datetime):
                                        dt_start = start_time
                                    elif isinstance(start_time, str):
                                        if start_time.endswith("Z"):
                                            dt_start = datetime.fromisoformat(
                                                start_time[:-1] + "+00:00"
                                            )
                                        elif "+" in start_time or start_time.count("-") > 2:
                                            dt_start = datetime.fromisoformat(start_time)
                                        else:
                                            dt_start = datetime.fromisoformat(start_time + "+00:00")
                                    else:
                                        continue

                                    start_epoch = epoch_seconds_paris(dt_start)
                                    if start_epoch is None:
                                        continue

                                    duration = float(time_played or 0) if time_played else 12 * 60
                                    end_epoch = start_epoch + duration

                                    all_windows.append(
                                        {
                                            "match_id": str(match_id),
                                            "start_epoch": start_epoch,
                                            "end_epoch": end_epoch,
                                            "start_time": dt_start,
                                        }
                                    )
                                except Exception:
                                    continue
                except Exception:
                    continue

        if not all_windows:
            return _empty

        return pl.DataFrame(all_windows).sort("start_epoch")

    except Exception:
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
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
            from src.utils.db import has_table

            if not has_table(conn, "media_files"):
                return pl.DataFrame()

            if xuid or gamertag:
                uids = [u for u in (xuid, gamertag) if u]
                uids = list(dict.fromkeys(uids))
                if not uids:
                    uid_filter = "1=0"
                    params: list[str] = []
                elif len(uids) == 1:
                    uid_filter = "mma.xuid = ?"
                    params = [uids[0]]
                else:
                    uid_filter = "(mma.xuid = ? OR mma.xuid = ?)"
                    params = list(uids[:2])
                result = conn.execute(
                    f"""
                    SELECT DISTINCT
                        mf.file_path AS path,
                        mf.mtime,
                        mf.mtime_paris_epoch,
                        mf.file_ext AS ext,
                        mf.kind,
                        mf.file_name AS basename,
                        mf.thumbnail_path,
                        mma.match_id,
                        mma.match_start_time,
                        mma.association_confidence,
                        mma.xuid
                    FROM media_files mf
                    LEFT JOIN media_match_associations mma
                        ON mf.file_path = mma.media_path
                        AND ({uid_filter})
                    ORDER BY mf.mtime_paris_epoch DESC
                    """,
                    params,
                ).fetchall()
            else:
                result = conn.execute(
                    """
                    SELECT DISTINCT
                        mf.file_path AS path,
                        mf.mtime,
                        mf.mtime_paris_epoch,
                        mf.file_ext AS ext,
                        mf.kind,
                        mf.file_name AS basename,
                        mf.thumbnail_path,
                        mma.match_id,
                        mma.match_start_time,
                        mma.association_confidence,
                        mma.xuid
                    FROM media_files mf
                    LEFT JOIN media_match_associations mma
                        ON mf.file_path = mma.media_path
                    ORDER BY mf.mtime_paris_epoch DESC
                    """
                ).fetchall()

            if not result:
                return pl.DataFrame()

            rows = [dict(zip(_col_names, row, strict=False)) for row in result]
            return pl.DataFrame(rows)

    except Exception:
        return pl.DataFrame()
