"""Fonctions libres et constantes pour l'indexation média.

Extraites de ``media_indexer.py`` pour réduire le volume du module principal.
"""

from __future__ import annotations

import hashlib
import logging
import subprocess
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import duckdb

from src.ui.tz import db_ts_to_utc
from src.utils.paths import PLAYER_DB_FILENAME, PLAYERS_DIR, get_shared_matches_path_from_player

logger = logging.getLogger(__name__)

# Extensions supportées
IMAGE_EXTENSIONS = {".png", ".jpg", ".jpeg", ".webp"}
VIDEO_EXTENSIONS = {".mp4", ".webm", ".mkv", ".mov", ".avi"}

# Version du schéma pour migrations
SCAN_VERSION = 2


@dataclass
class ScanResult:
    """Résultat d'un scan de médias."""

    n_scanned: int = 0
    n_new: int = 0
    n_updated: int = 0
    n_deleted: int = 0
    n_associated: int = 0
    errors: list[str] = None  # type: ignore[assignment]

    def __post_init__(self) -> None:
        if self.errors is None:
            self.errors = []


def get_gamertag_from_db_path(db_path: Path | str) -> str | None:
    """Extrait le gamertag depuis data/players/{gamertag}/stats.duckdb."""
    try:
        from src.utils.paths import PLAYER_DB_FILENAME

        p = Path(db_path).resolve()
        if p.name == PLAYER_DB_FILENAME:
            return p.parent.name
    except Exception:
        pass
    return None


def match_start_to_epoch(start_time: datetime | str | float) -> float | None:
    """Convertit start_time (DB/API) en epoch seconds."""
    try:
        if isinstance(start_time, int | float):
            return float(start_time)
        if isinstance(start_time, datetime):
            dt = start_time
        elif isinstance(start_time, str):
            if start_time.endswith("Z"):
                dt = datetime.fromisoformat(start_time[:-1] + "+00:00")
            elif "+" in start_time or start_time.count("-") > 2:
                dt = datetime.fromisoformat(start_time)
            else:
                dt = datetime.fromisoformat(start_time + "+00:00")
        else:
            return None
        if dt.tzinfo is None:
            dt = db_ts_to_utc(dt)
        return dt.timestamp()
    except Exception:
        return None


def get_video_duration(file_path: Path) -> float | None:
    """Récupère la durée d'une vidéo via ffprobe."""
    try:
        result = subprocess.run(
            [
                "ffprobe",
                "-v",
                "error",
                "-show_entries",
                "format=duration",
                "-of",
                "default=noprint_wrappers=1:nokey=1",
                str(file_path),
            ],
            capture_output=True,
            text=True,
            timeout=30,
        )
        if result.returncode == 0 and result.stdout.strip():
            return float(result.stdout.strip())
    except Exception:
        pass
    return None


def get_image_exif_datetime(file_path: Path) -> datetime | None:
    """Récupère DateTimeOriginal ou CreateDate depuis EXIF (PIL)."""
    try:
        from PIL import Image
        from PIL.ExifTags import TAGS
    except ImportError:
        return None

    try:
        with Image.open(file_path) as img:
            exif = img.getexif()
            if not exif:
                return None
            for tag_id, value in exif.items():
                if TAGS.get(tag_id) in ("DateTimeOriginal", "DateTime", "CreateDate") and value:
                    s = str(value).strip()
                    if " " in s:
                        date_part, time_part = s.split(" ", 1)
                        s = date_part.replace(":", "-") + " " + time_part
                    try:
                        return datetime.fromisoformat(s.replace(":", "-", 2))
                    except ValueError:
                        pass
    except Exception:
        pass
    return None


def get_image_thumbnail_path(image_path: Path, thumbs_dir: Path) -> Path:
    """Chemin du thumbnail pour une image (miniature dédiée)."""
    path_hash = hashlib.md5(str(image_path.resolve()).encode()).hexdigest()[:12]
    stem = image_path.stem[:50]
    ext = image_path.suffix.lower()
    out_ext = ".jpg" if ext in {".jpg", ".jpeg"} else ".png"
    return thumbs_dir / f"{stem}_{path_hash}{out_ext}"


def generate_image_thumbnail(
    image_path: Path,
    output_path: Path,
    *,
    max_width: int = 320,
) -> bool:
    """Génère une miniature pour une image (PIL resize)."""
    try:
        from PIL import Image
    except ImportError:
        return False
    try:
        with Image.open(image_path) as img:
            img.load()
            w, h = img.size
            if w <= max_width and h <= max_width:
                ratio = 1.0
            elif w >= h:
                ratio = max_width / w
            else:
                ratio = max_width / h
            new_w = max(1, int(w * ratio))
            new_h = max(1, int(h * ratio))
            resample = getattr(Image.Resampling, "LANCZOS", Image.LANCZOS)
            thumb = img.resize((new_w, new_h), resample)
            output_path.parent.mkdir(parents=True, exist_ok=True)
            if output_path.suffix.lower() in {".jpg", ".jpeg"}:
                thumb.save(output_path, "JPEG", quality=85, optimize=True)
            else:
                thumb.save(output_path, "PNG", optimize=True)
            return output_path.exists()
    except Exception:
        return False


# ---------------------------------------------------------------------------
#  Fonctions utilitaires pour le core indexer
# ---------------------------------------------------------------------------


def get_existing_columns(conn: duckdb.DuckDBPyConnection, table: str) -> set[str]:
    """Retourne les noms de colonnes d'une table DuckDB."""
    try:
        cols = conn.execute(
            """
            SELECT column_name FROM information_schema.columns
            WHERE table_schema = 'main' AND table_name = ?
            """,
            [table],
        ).fetchall()
        return {row[0] for row in cols}
    except Exception:
        return set()


def compute_file_hash(file_path: Path) -> str:
    """Calcule le hash MD5 d'un fichier."""
    try:
        h = hashlib.md5()
        with open(file_path, "rb") as f:
            for chunk in iter(lambda: f.read(4096), b""):
                h.update(chunk)
        return h.hexdigest()
    except Exception as e:
        logger.warning("Hash %s: %s", file_path, e)
        return ""


def get_file_metadata(file_path: Path) -> dict[str, Any] | None:  # noqa: PLR0912
    """Récupère les métadonnées (capture_start_utc, capture_end_utc, duration_seconds, title)."""
    try:
        if not file_path.exists():
            return None
        stat = file_path.stat()
        ext = file_path.suffix.lower()
        kind = (
            "video"
            if ext in VIDEO_EXTENSIONS
            else "image"
            if ext in IMAGE_EXTENSIONS
            else "unknown"
        )
        if kind == "unknown":
            return None

        mtime = float(stat.st_mtime)
        capture_end_utc = datetime.fromtimestamp(mtime, tz=timezone.utc).replace(tzinfo=None)
        capture_start_utc: datetime | None = None
        duration_seconds: float | None = None
        title: str | None = None

        if kind == "video":
            duration_seconds = get_video_duration(file_path)
            if duration_seconds is not None and duration_seconds > 0:
                capture_start_utc = datetime.fromtimestamp(
                    mtime - duration_seconds, tz=timezone.utc
                ).replace(tzinfo=None)
            else:
                capture_start_utc = capture_end_utc
        else:
            exif_dt = get_image_exif_datetime(file_path)
            # EXIF sans timezone = heure locale appareil (non UTC) → ignorer
            if exif_dt and exif_dt.tzinfo is not None:
                exif_dt = exif_dt.astimezone(timezone.utc).replace(tzinfo=None)
                capture_end_utc = exif_dt
                capture_start_utc = exif_dt
            else:
                capture_start_utc = capture_end_utc

        return {
            "file_path": str(file_path.resolve()),
            "file_name": file_path.name,
            "file_size": stat.st_size,
            "file_ext": ext.lstrip("."),
            "kind": kind,
            "mtime": mtime,
            "capture_start_utc": capture_start_utc,
            "capture_end_utc": capture_end_utc,
            "duration_seconds": duration_seconds,
            "title": title,
        }
    except Exception as e:
        logger.warning("Métadonnées %s: %s", file_path, e)
        return None


def _load_xuid_by_gamertag(shared_path: Path) -> dict[str, str]:
    """Charge le mapping gamertag (lower) → xuid depuis v_gamertag_lookup."""
    xuid_by_gamertag: dict[str, str] = {}
    if not shared_path.exists():
        return xuid_by_gamertag
    try:
        with duckdb.connect(str(shared_path), read_only=True) as sc:
            try:
                rows = sc.execute("SELECT xuid, gamertag FROM v_gamertag_lookup").fetchall()
            except Exception:
                rows = sc.execute("SELECT xuid, gamertag FROM xuid_aliases").fetchall()
            for xuid_val, gamertag_val in rows:
                if gamertag_val and xuid_val:
                    xuid_by_gamertag[str(gamertag_val).lower()] = str(xuid_val)
    except Exception as e:
        logger.debug("_load_xuid_by_gamertag: %s", e)
    return xuid_by_gamertag


def get_all_player_dbs() -> list[tuple[Path, str]]:
    """Retourne les (db_path, xuid) de tous les joueurs.

    Lit les xuids depuis shared_matches_v2.duckdb/v_gamertag_lookup.
    """
    player_dbs: list[tuple[Path, str]] = []
    if not PLAYERS_DIR.exists():
        return player_dbs

    shared_path = get_shared_matches_path_from_player(PLAYERS_DIR / "_any" / PLAYER_DB_FILENAME)
    for player_dir in PLAYERS_DIR.iterdir():
        if player_dir.is_dir():
            first_db = player_dir / PLAYER_DB_FILENAME
            if first_db.exists():
                shared_path = get_shared_matches_path_from_player(first_db)
                break

    xuid_by_gamertag = _load_xuid_by_gamertag(shared_path) if shared_path else {}

    for player_dir in sorted(PLAYERS_DIR.iterdir(), key=lambda p: p.name):
        if not player_dir.is_dir():
            continue
        db_path = player_dir / PLAYER_DB_FILENAME
        if not db_path.exists():
            continue
        gamertag = player_dir.name
        xuid = xuid_by_gamertag.get(gamertag.lower(), gamertag)
        player_dbs.append((db_path, xuid))
    return player_dbs


# ---------------------------------------------------------------------------
#  Fonctions d'écriture DB (upsert médias)
# ---------------------------------------------------------------------------


def insert_new_media(
    conn: duckdb.DuckDBPyConnection,
    meta: dict[str, Any],
    has_owner_xuid: bool,
    owner_xuid_val: str,
    now: datetime,
) -> None:
    """Insère un nouveau fichier média dans la DB."""
    base_cols = (
        "file_path, file_hash, file_name, file_size, file_ext, kind, "
        "capture_start_utc, capture_end_utc, duration_seconds, title, "
        "mtime, mtime_paris_epoch, status, first_seen_at, last_scan_at, scan_version"
    )
    base_vals = [
        meta["file_path"],
        meta["file_hash"],
        meta["file_name"],
        meta["file_size"],
        meta["file_ext"],
        meta["kind"],
        meta["capture_start_utc"],
        meta["capture_end_utc"],
        meta["duration_seconds"],
        meta["title"],
        meta["mtime"],
        meta["mtime"],
        "active",
        now,
        now,
        SCAN_VERSION,
    ]
    if has_owner_xuid:
        conn.execute(
            f"INSERT INTO media_files ({base_cols}, owner_xuid) "  # noqa: S608
            f"VALUES ({','.join('?' for _ in base_vals)}, ?)",
            [*base_vals, owner_xuid_val],
        )
    else:
        conn.execute(
            f"INSERT INTO media_files ({base_cols}) "  # noqa: S608
            f"VALUES ({','.join('?' for _ in base_vals)})",
            base_vals,
        )


def update_existing_media(
    conn: duckdb.DuckDBPyConnection,
    meta: dict[str, Any],
    now: datetime,
) -> None:
    """Met à jour un fichier média existant."""
    conn.execute(
        """
        UPDATE media_files SET
            file_hash = ?, file_size = ?, capture_start_utc = ?,
            capture_end_utc = ?, duration_seconds = ?, title = ?,
            mtime = ?, mtime_paris_epoch = ?, status = 'active',
            last_scan_at = ?
        WHERE file_path = ?
        """,
        [
            meta["file_hash"],
            meta["file_size"],
            meta["capture_start_utc"],
            meta["capture_end_utc"],
            meta["duration_seconds"],
            meta["title"],
            meta["mtime"],
            meta["mtime"],
            now,
            meta["file_path"],
        ],
    )
