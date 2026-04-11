"""Helpers de scan disque pour MediaIndexer."""

from __future__ import annotations

import logging
import os
from pathlib import Path
from typing import Any

import duckdb

from src.data.media_helpers import (
    IMAGE_EXTENSIONS,
    VIDEO_EXTENSIONS,
    ScanResult,
    compute_file_hash,
    get_file_metadata,
    is_generated_thumbnail_path,
)

logger = logging.getLogger(__name__)


def load_existing_media(conn: duckdb.DuckDBPyConnection) -> dict[str, dict[str, Any]]:
    """Charge l'état courant des médias non supprimés."""
    rows = conn.execute(
        "SELECT file_path, file_hash, mtime FROM media_files WHERE status != 'deleted'"
    ).fetchall()
    return {row[0]: {"hash": row[1], "mtime": row[2]} for row in rows}


def resolve_scan_targets(
    player_captures_dir: Path | None,
    videos_dir: Path | None,
    screens_dir: Path | None,
) -> list[tuple[Path | None, set[str]]]:
    """Construit la liste des dossiers et extensions à scanner."""
    if player_captures_dir and Path(player_captures_dir).exists():
        return [(player_captures_dir, VIDEO_EXTENSIONS | IMAGE_EXTENSIONS)]
    return [(videos_dir, VIDEO_EXTENSIONS), (screens_dir, IMAGE_EXTENSIONS)]


def scan_media_dirs(
    dirs_exts: list[tuple[Path | None, set[str]]],
    existing: dict[str, dict[str, Any]],
    force_rescan: bool,
    result: ScanResult,
) -> tuple[set[str], list[dict[str, Any]]]:
    """Scanne tous les dossiers média configurés."""
    paths_on_disk: set[str] = set()
    files_to_process: list[dict[str, Any]] = []
    for media_dir, exts in dirs_exts:
        scan_single_media_dir(
            media_dir,
            exts,
            existing,
            force_rescan,
            result,
            paths_on_disk,
            files_to_process,
        )
    return paths_on_disk, files_to_process


def scan_single_media_dir(  # noqa: PLR0913
    media_dir: Path | None,
    exts: set[str],
    existing: dict[str, dict[str, Any]],
    force_rescan: bool,
    result: ScanResult,
    paths_on_disk: set[str],
    files_to_process: list[dict[str, Any]],
) -> None:
    """Scanne un dossier média unique et accumule les candidats."""
    if not media_dir or not Path(media_dir).exists():
        return
    try:
        walk_iter = os.walk(media_dir)
    except OSError as exc:
        result.errors.append(f"Dossier inaccessible {media_dir}: {exc}")
        logger.warning("Scan dossier %s: %s", media_dir, exc)
        return

    for root, dirs, files in walk_iter:
        dirs[:] = [dirname for dirname in dirs if dirname.lower() != "thumbs"]
        for name in files:
            append_media_candidate(
                Path(root) / name,
                exts,
                existing,
                force_rescan,
                result,
                paths_on_disk,
                files_to_process,
            )


def append_media_candidate(  # noqa: PLR0913
    file_path: Path,
    exts: set[str],
    existing: dict[str, dict[str, Any]],
    force_rescan: bool,
    result: ScanResult,
    paths_on_disk: set[str],
    files_to_process: list[dict[str, Any]],
) -> None:
    """Analyse un fichier du disque et l'ajoute aux candidats si nécessaire."""
    if is_generated_thumbnail_path(file_path) or file_path.suffix.lower() not in exts:
        return

    result.n_scanned += 1
    try:
        meta = get_file_metadata(file_path)
    except Exception as exc:
        result.errors.append(f"Métadonnées {file_path}: {exc}")
        logger.debug("Métadonnées %s: %s", file_path, exc)
        return
    if not meta:
        return

    path_str = meta["file_path"]
    paths_on_disk.add(path_str)
    if (
        path_str in existing
        and not force_rescan
        and abs(meta["mtime"] - existing[path_str]["mtime"]) < 1.0
    ):
        return

    file_hash = compute_file_hash(file_path)
    if not file_hash:
        result.errors.append(f"Hash impossible: {path_str}")
        return
    meta["file_hash"] = file_hash
    files_to_process.append(meta)


def mark_deleted_media(
    conn: duckdb.DuckDBPyConnection,
    existing: dict[str, dict[str, Any]],
    paths_on_disk: set[str],
    result: ScanResult,
) -> None:
    """Marque en deleted les médias absents du disque source."""
    for path_str in existing:
        if path_str in paths_on_disk:
            continue
        try:
            conn.execute(
                "UPDATE media_files SET status = 'deleted' WHERE file_path = ?",
                [path_str],
            )
            result.n_deleted += 1
        except Exception as exc:
            result.errors.append(f"Delete {path_str}: {exc}")
