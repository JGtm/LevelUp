"""Cache local du manifest film (blob_prefix + chunks_meta).

Évite un appel API `get_film_by_match_id` par match sur les re-runs :
le manifest (~2KB JSON) est sauvegardé dans un dossier dédié
(`data/cache/film_manifests/{match_id[:8]}.json`).

Structure JSON :
  { "blob_prefix": str,
    "chunks": [ { "index", "chunk_type", "start_ms", "duration_ms", "file_relative_path" } ] }
"""

from __future__ import annotations

import json
import logging
from pathlib import Path
from types import SimpleNamespace

logger = logging.getLogger(__name__)

MANIFEST_FILENAME = "manifest.json"

# KILL_WINDOW_MS est défini dans weapon_parser — on l'importe pour éviter la
# duplication mais on peut aussi passer la valeur comme paramètre.
_KILL_WINDOW_MS_DEFAULT = 5000  # ms — doit rester synchronisé avec weapon_parser.KILL_WINDOW_MS


def compute_needed_chunks(
    kill_times_ms: list[int],
    chunks_meta: list,
    kill_window_ms: int = _KILL_WINDOW_MS_DEFAULT,
) -> set[int]:
    """Identifie les chunks couvrant les fenêtres [kill_t - kill_window_ms, kill_t]."""
    needed: set[int] = set()
    for kill_t in kill_times_ms:
        window_start = kill_t - kill_window_ms
        for ch in chunks_meta:
            if ch.chunk_type.value != 2:
                continue
            ch_start = ch.chunk_start_time_offset_milliseconds
            ch_end = ch_start + ch.duration_milliseconds
            if ch_end >= window_start and ch_start <= kill_t:
                needed.add(ch.index)
    return needed


def write_manifest_cache(
    manifest_dir: Path, match_id: str, blob_prefix: str, chunks_meta: list
) -> None:
    """Sérialise le manifest film dans manifest_dir/{match_id[:8]}.json."""
    manifest_dir.mkdir(parents=True, exist_ok=True)
    payload = {
        "blob_prefix": blob_prefix,
        "chunks": [
            {
                "index": ch.index,
                "chunk_type": ch.chunk_type.value,
                "start_ms": ch.chunk_start_time_offset_milliseconds,
                "duration_ms": ch.duration_milliseconds,
                "file_relative_path": ch.file_relative_path,
            }
            for ch in chunks_meta
        ],
    }
    try:
        (manifest_dir / f"{match_id[:8]}.json").write_text(
            json.dumps(payload, separators=(",", ":")),
            encoding="utf-8",
        )
    except OSError as exc:
        logger.debug("manifest_cache: write failed: %s", exc)


def load_manifest_cache(manifest_dir: Path, match_id: str) -> tuple[str, list] | None:
    """Charge blob_prefix + chunks_meta depuis manifest_dir/{match_id[:8]}.json.

    Returns:
        ``(blob_prefix, chunks_meta)`` reconstruit en SimpleNamespace, ou None.
    """
    path = manifest_dir / f"{match_id[:8]}.json"
    if not path.exists():
        return None
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
        blob_prefix: str = data["blob_prefix"]
        chunks_meta = [
            SimpleNamespace(
                index=d["index"],
                chunk_type=SimpleNamespace(value=d["chunk_type"]),
                chunk_start_time_offset_milliseconds=d["start_ms"],
                duration_milliseconds=d["duration_ms"],
                file_relative_path=d["file_relative_path"],
            )
            for d in data["chunks"]
        ]
        return blob_prefix, chunks_meta
    except Exception as exc:
        logger.debug("manifest_cache: load failed: %s", exc)
        return None
