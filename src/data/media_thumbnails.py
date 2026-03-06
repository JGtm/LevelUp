"""Génération de thumbnails média (vidéos GIF + images miniatures).

Extrait de ``MediaIndexer`` pour réduire le volume du module principal.
"""

from __future__ import annotations

import contextlib
import importlib.util
import logging
from pathlib import Path

import duckdb

from src.data.media_helpers import generate_image_thumbnail, get_image_thumbnail_path

logger = logging.getLogger(__name__)


def generate_thumbnails_for_new(
    db_path: Path,
    videos_dir: Path | None = None,
    screens_dir: Path | None = None,
    *,
    ensure_schema_fn: callable | None = None,
) -> tuple[int, int]:
    """Génère les thumbnails vidéo (GIF) et image (miniatures).

    Args:
        db_path: Chemin vers la DB joueur.
        videos_dir: Dossier des vidéos.
        screens_dir: Dossier des captures d'écran.
        ensure_schema_fn: Callback optionnel pour créer le schéma si nécessaire.

    Returns:
        (generated, errors)
    """
    total_gen = 0
    total_err = 0

    if videos_dir and videos_dir.exists():
        gen, err = _generate_video_thumbnails(db_path, videos_dir, ensure_schema_fn)
        total_gen += gen
        total_err += err

    if screens_dir and screens_dir.exists():
        gen, err = _generate_image_thumbnails(db_path, screens_dir, ensure_schema_fn)
        total_gen += gen
        total_err += err

    return total_gen, total_err


def _generate_video_thumbnails(  # noqa: PLR0912
    db_path: Path,
    videos_dir: Path,
    ensure_schema_fn: callable | None = None,
) -> tuple[int, int]:
    """Génère les thumbnails GIF pour les vidéos."""
    try:
        from scripts.generate_thumbnails import (
            check_ffmpeg,
            generate_thumbnail_gif,
            get_thumbnail_path,
        )
    except ImportError:
        return 0, 0
    if not check_ffmpeg():
        return 0, 0

    if ensure_schema_fn:
        with contextlib.suppress(Exception):
            ensure_schema_fn()

    with duckdb.connect(str(db_path), read_only=False) as conn:
        videos = conn.execute(
            """
            SELECT file_path, file_name, thumbnail_path FROM media_files
            WHERE kind = 'video' AND status = 'active'
            ORDER BY mtime DESC
            """
        ).fetchall()
        if not videos:
            return 0, 0
        to_process = [
            (fp, fn) for fp, fn, tp in videos if not tp or not tp.strip() or not Path(tp).exists()
        ]
        if not to_process:
            return 0, 0
        thumbs_dir = videos_dir / "thumbs"
        thumbs_dir.mkdir(exist_ok=True)
        generated = errors = 0
        for path_str, _fname in to_process:
            p = Path(path_str)
            if not p.exists():
                continue
            thumb = get_thumbnail_path(p, thumbs_dir)
            if thumb.exists():
                conn.execute(
                    "UPDATE media_files SET thumbnail_path = ? WHERE file_path = ?",
                    [str(thumb), path_str],
                )
                generated += 1
                continue
            try:
                if generate_thumbnail_gif(p, thumb):
                    conn.execute(
                        "UPDATE media_files SET thumbnail_path = ? WHERE file_path = ?",
                        [str(thumb), path_str],
                    )
                    generated += 1
                else:
                    errors += 1
            except Exception:
                errors += 1
        conn.commit()
    return generated, errors


def _generate_image_thumbnails(
    db_path: Path,
    screens_dir: Path,
    ensure_schema_fn: callable | None = None,
) -> tuple[int, int]:
    """Génère les miniatures pour les images (redimensionnement)."""
    if importlib.util.find_spec("PIL") is None:
        return 0, 0
    if ensure_schema_fn:
        ensure_schema_fn()
    with duckdb.connect(str(db_path), read_only=False) as conn:
        images = conn.execute(
            """
            SELECT file_path, file_name FROM media_files
            WHERE kind = 'image' AND status = 'active'
            AND (thumbnail_path IS NULL OR thumbnail_path = '')
            ORDER BY mtime DESC
            """
        ).fetchall()
        if not images:
            return 0, 0
        thumbs_dir = screens_dir / "thumbs"
        thumbs_dir.mkdir(exist_ok=True)
        generated = errors = 0
        max_width = 320
        for path_str, _fname in images:
            p = Path(path_str)
            if not p.exists():
                continue
            thumb = get_image_thumbnail_path(p, thumbs_dir)
            if thumb.exists():
                conn.execute(
                    "UPDATE media_files SET thumbnail_path = ? WHERE file_path = ?",
                    [str(thumb), path_str],
                )
                generated += 1
                continue
            try:
                if generate_image_thumbnail(p, thumb, max_width=max_width):
                    conn.execute(
                        "UPDATE media_files SET thumbnail_path = ? WHERE file_path = ?",
                        [str(thumb), path_str],
                    )
                    generated += 1
                else:
                    errors += 1
            except Exception:
                errors += 1
        conn.commit()
    return generated, errors
