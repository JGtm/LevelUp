"""Références metadata des assets battle pass mis en cache localement."""

from __future__ import annotations

import logging
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from src.data.sync.migrations_metadata import ensure_battlepass_asset_refs_table
from src.utils.db import duckdb_read_only, duckdb_read_write
from src.utils.paths import DATA_DIR, get_metadata_db_path

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class StoredBattlepassAssetRef:
    """Référence d'asset battle pass stockée dans metadata.duckdb."""

    asset_kind: str
    source_path: str
    cache_rel_path: str
    mime_type: str
    image_source_path: str | None = None
    source_origin: str | None = None

    @property
    def cache_path(self) -> Path:
        """Retourne le chemin absolu du fichier de cache."""
        path = Path(self.cache_rel_path)
        return path if path.is_absolute() else DATA_DIR / path


def load_battlepass_asset_ref(
    asset_kind: str,
    source_path: str,
) -> StoredBattlepassAssetRef | None:
    """Charge une référence d'asset battle pass depuis metadata.duckdb."""
    metadata_path = get_metadata_db_path()
    if not metadata_path.exists():
        return None
    try:
        with duckdb_read_only(metadata_path) as conn:
            if not _battlepass_asset_refs_available(conn):
                return None
            row = conn.execute(
                """
                SELECT asset_kind, source_path, cache_rel_path, mime_type, image_source_path, source_origin
                FROM battlepass_asset_refs
                WHERE asset_kind = ? AND source_path = ?
                """,
                [asset_kind, source_path],
            ).fetchone()
            if row is None:
                return None
            return StoredBattlepassAssetRef(
                asset_kind=row[0],
                source_path=row[1],
                cache_rel_path=row[2],
                mime_type=row[3],
                image_source_path=row[4],
                source_origin=row[5],
            )
    except Exception as exc:
        logger.debug("battlepass_asset_refs: lecture ignorée: %s", exc)
        return None


def persist_battlepass_asset_ref(  # noqa: PLR0913
    asset_kind: str,
    source_path: str,
    cache_path: Path,
    *,
    mime_type: str,
    image_source_path: str | None,
    source_origin: str,
) -> None:
    """Persiste ou met à jour une référence d'asset battle pass."""
    metadata_path = get_metadata_db_path()
    metadata_path.parent.mkdir(parents=True, exist_ok=True)
    cache_rel_path = _to_cache_rel_path(cache_path)
    try:
        with duckdb_read_write(metadata_path) as conn:
            ensure_battlepass_asset_refs_table(conn)
            conn.execute(
                """
                INSERT INTO battlepass_asset_refs (
                    asset_key,
                    asset_kind,
                    source_path,
                    cache_rel_path,
                    mime_type,
                    image_source_path,
                    source_origin,
                    first_cached_at,
                    last_cached_at,
                    last_accessed_at
                ) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
                ON CONFLICT(asset_key) DO UPDATE SET
                    cache_rel_path = excluded.cache_rel_path,
                    mime_type = excluded.mime_type,
                    image_source_path = excluded.image_source_path,
                    source_origin = excluded.source_origin,
                    last_cached_at = CURRENT_TIMESTAMP,
                    last_accessed_at = CURRENT_TIMESTAMP
                """,
                [
                    _asset_key(asset_kind, source_path),
                    asset_kind,
                    source_path,
                    cache_rel_path,
                    mime_type,
                    image_source_path,
                    source_origin,
                ],
            )
    except Exception as exc:
        logger.debug("battlepass_asset_refs: persistance ignorée: %s", exc)


def touch_battlepass_asset_ref(asset_kind: str, source_path: str) -> None:
    """Met à jour le timestamp d'accès d'un asset battle pass référencé."""
    metadata_path = get_metadata_db_path()
    if not metadata_path.exists():
        return
    try:
        with duckdb_read_write(metadata_path) as conn:
            if not _battlepass_asset_refs_available(conn):
                return
            conn.execute(
                """
                UPDATE battlepass_asset_refs
                SET last_accessed_at = CURRENT_TIMESTAMP
                WHERE asset_key = ?
                """,
                [_asset_key(asset_kind, source_path)],
            )
    except Exception as exc:
        logger.debug("battlepass_asset_refs: touch ignoré: %s", exc)


def _battlepass_asset_refs_available(conn: Any) -> bool:
    row = conn.execute(
        """
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'main' AND table_name = 'battlepass_asset_refs'
        """
    ).fetchone()
    return row is not None


def _asset_key(asset_kind: str, source_path: str) -> str:
    return f"{asset_kind}:{source_path}"


def _to_cache_rel_path(cache_path: Path) -> str:
    try:
        return cache_path.resolve().relative_to(DATA_DIR.resolve()).as_posix()
    except ValueError:
        return cache_path.as_posix()
