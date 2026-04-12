"""Catalogue metadata partagé du battle pass Halo Infinite."""

from __future__ import annotations

import logging
from dataclasses import dataclass
from typing import Any

from src.data._battlepass_catalog_utils import (
    build_content_hash as _build_content_hash,
)
from src.data._battlepass_catalog_utils import (
    coerce_int as _coerce_int,
)
from src.data._battlepass_catalog_utils import (
    coerce_str as _coerce_str,
)
from src.data._battlepass_catalog_utils import (
    dump_payload as _dump_payload,
)
from src.data._battlepass_catalog_utils import (
    extract_display_path as _extract_display_path,
)
from src.data._battlepass_catalog_utils import (
    extract_localized_texts as _extract_localized_texts,
)
from src.data._battlepass_catalog_utils import (
    loads_payload as _loads_payload,
)
from src.data.challenges import normalize_challenge_lang
from src.data.sync._asset_langs import DEFAULT_LANG
from src.data.sync.migrations_metadata import (
    ensure_battlepass_item_definitions_table,
    ensure_battlepass_item_translations_table,
    ensure_battlepass_track_definitions_table,
    ensure_battlepass_track_translations_table,
)
from src.utils.db import duckdb_read_only, duckdb_read_write
from src.utils.paths import get_metadata_db_path

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class StoredBattlepassTrackDefinition:
    """Reward track battle pass relu depuis metadata.duckdb."""

    reward_track_path: str
    content_hash: str
    xp_per_rank: int | None = None
    battlepass_image_path: str | None = None
    background_image_path: str | None = None
    track_name: str | None = None
    raw_payload_json: str | None = None

    @property
    def payload(self) -> dict[str, Any]:
        """Retourne le payload CMS brut si présent."""
        return _loads_payload(self.raw_payload_json)


@dataclass(frozen=True)
class StoredBattlepassItemDefinition:
    """Item battle pass relu depuis metadata.duckdb."""

    inventory_item_path: str
    content_hash: str
    quality: str | None = None
    item_type: str | None = None
    display_path: str | None = None
    title: str | None = None
    description: str | None = None
    raw_payload_json: str | None = None

    @property
    def payload(self) -> dict[str, Any]:
        """Retourne le payload CMS brut si présent."""
        return _loads_payload(self.raw_payload_json)


def persist_battlepass_track_definition(track_path: str, payload: dict[str, Any]) -> str | None:
    """Persiste une définition de reward track dans metadata.duckdb."""
    if not track_path or not payload:
        return None
    metadata_path = get_metadata_db_path()
    metadata_path.parent.mkdir(parents=True, exist_ok=True)
    content_hash = _build_content_hash(payload)
    try:
        with duckdb_read_write(metadata_path) as conn:
            ensure_battlepass_track_definitions_table(conn)
            ensure_battlepass_track_translations_table(conn)
            _upsert_track_definition(conn, track_path, content_hash, payload)
            _upsert_track_translations(conn, track_path, content_hash, payload)
    except Exception as exc:
        logger.debug("battlepass_catalog: persistance track ignorée: %s", exc)
        return None
    return content_hash


def load_battlepass_track_definition(
    track_path: str,
    lang: str = "fr",
) -> StoredBattlepassTrackDefinition | None:
    """Charge une définition de reward track depuis metadata.duckdb."""
    if not track_path:
        return None
    metadata_path = get_metadata_db_path()
    if not metadata_path.exists():
        return None
    requested_lang = normalize_challenge_lang(lang)
    try:
        with duckdb_read_only(metadata_path) as conn:
            if not _track_tables_available(conn):
                return None
            entry = _fetch_track_entry(conn, track_path, requested_lang)
            if entry is not None or requested_lang == DEFAULT_LANG:
                return entry
            return _fetch_track_entry(conn, track_path, DEFAULT_LANG)
    except Exception as exc:
        logger.debug("battlepass_catalog: lecture track ignorée: %s", exc)
        return None


def persist_battlepass_item_catalog(definitions: dict[str, dict[str, Any]]) -> dict[str, str]:
    """Persiste un lot de définitions d'items battle pass."""
    if not definitions:
        return {}
    metadata_path = get_metadata_db_path()
    metadata_path.parent.mkdir(parents=True, exist_ok=True)
    hashes: dict[str, str] = {}
    try:
        with duckdb_read_write(metadata_path) as conn:
            ensure_battlepass_item_definitions_table(conn)
            ensure_battlepass_item_translations_table(conn)
            for item_path, payload in definitions.items():
                if not item_path or not payload:
                    continue
                content_hash = _build_content_hash(payload)
                hashes[item_path] = content_hash
                _upsert_item_definition(conn, item_path, content_hash, payload)
                _upsert_item_translations(conn, item_path, content_hash, payload)
    except Exception as exc:
        logger.debug("battlepass_catalog: persistance items ignorée: %s", exc)
        return {}
    return hashes


def load_battlepass_item_metadata_map(
    item_paths: list[str],
    lang: str = "fr",
) -> dict[str, StoredBattlepassItemDefinition]:
    """Charge un lot de définitions d'items battle pass depuis metadata.duckdb."""
    if not item_paths:
        return {}
    metadata_path = get_metadata_db_path()
    if not metadata_path.exists():
        return {}
    unique_paths = list(dict.fromkeys(path for path in item_paths if path))
    requested_lang = normalize_challenge_lang(lang)
    try:
        with duckdb_read_only(metadata_path) as conn:
            if not _item_tables_available(conn):
                return {}
            entries = _fetch_item_entries(conn, unique_paths, requested_lang)
            if requested_lang != DEFAULT_LANG:
                _merge_default_item_entries(conn, unique_paths, entries)
            return entries
    except Exception as exc:
        logger.debug("battlepass_catalog: lecture items ignorée: %s", exc)
        return {}


def _track_tables_available(conn: Any) -> bool:
    tables = _available_tables(conn)
    return {"battlepass_track_definitions", "battlepass_track_translations"}.issubset(tables)


def _item_tables_available(conn: Any) -> bool:
    tables = _available_tables(conn)
    return {"battlepass_item_definitions", "battlepass_item_translations"}.issubset(tables)


def _available_tables(conn: Any) -> set[str]:
    rows = conn.execute(
        "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'"
    ).fetchall()
    return {row[0] for row in rows}


def _fetch_track_entry(
    conn: Any,
    track_path: str,
    lang: str,
) -> StoredBattlepassTrackDefinition | None:
    row = conn.execute(
        """
        SELECT
            d.reward_track_path,
            d.content_hash,
            d.xp_per_rank,
            d.battlepass_image_path,
            d.background_image_path,
            d.raw_payload_json,
            t.track_name
        FROM battlepass_track_definitions d
        LEFT JOIN battlepass_track_translations t
          ON d.reward_track_path = t.reward_track_path
         AND d.content_hash = t.content_hash
         AND t.lang = ?
        WHERE d.reward_track_path = ? AND d.is_current = TRUE
        """,
        [lang, track_path],
    ).fetchone()
    if row is None:
        return None
    return StoredBattlepassTrackDefinition(
        reward_track_path=row[0],
        content_hash=row[1],
        xp_per_rank=row[2],
        battlepass_image_path=row[3],
        background_image_path=row[4],
        raw_payload_json=row[5],
        track_name=row[6],
    )


def _fetch_item_entries(
    conn: Any,
    item_paths: list[str],
    lang: str,
) -> dict[str, StoredBattlepassItemDefinition]:
    placeholders = ", ".join("?" for _ in item_paths)
    rows = conn.execute(
        f"""
        SELECT
            d.inventory_item_path,
            d.content_hash,
            d.quality,
            d.item_type,
            d.display_path,
            d.raw_payload_json,
            t.title,
            t.description
        FROM battlepass_item_definitions d
        LEFT JOIN battlepass_item_translations t
          ON d.inventory_item_path = t.inventory_item_path
         AND d.content_hash = t.content_hash
         AND t.lang = ?
        WHERE d.is_current = TRUE AND d.inventory_item_path IN ({placeholders})
        """,
        [lang, *item_paths],
    ).fetchall()
    return {
        row[0]: StoredBattlepassItemDefinition(
            inventory_item_path=row[0],
            content_hash=row[1],
            quality=row[2],
            item_type=row[3],
            display_path=row[4],
            raw_payload_json=row[5],
            title=row[6],
            description=row[7],
        )
        for row in rows
    }


def _merge_default_item_entries(
    conn: Any,
    unique_paths: list[str],
    entries: dict[str, StoredBattlepassItemDefinition],
) -> None:
    missing = [path for path in unique_paths if path not in entries or not entries[path].title]
    if not missing:
        return
    fallback_entries = _fetch_item_entries(conn, missing, DEFAULT_LANG)
    for path, fallback in fallback_entries.items():
        current = entries.get(path)
        if current is None:
            entries[path] = fallback
            continue
        entries[path] = StoredBattlepassItemDefinition(
            inventory_item_path=current.inventory_item_path,
            content_hash=current.content_hash,
            quality=current.quality,
            item_type=current.item_type,
            display_path=current.display_path,
            raw_payload_json=current.raw_payload_json,
            title=current.title or fallback.title,
            description=current.description or fallback.description,
        )


def _upsert_track_definition(
    conn: Any,
    track_path: str,
    content_hash: str,
    payload: dict[str, Any],
) -> None:
    conn.execute(
        "UPDATE battlepass_track_definitions SET is_current = FALSE WHERE reward_track_path = ? AND content_hash <> ?",
        [track_path, content_hash],
    )
    conn.execute(
        """
        INSERT INTO battlepass_track_definitions (
            reward_track_path,
            content_hash,
            xp_per_rank,
            battlepass_image_path,
            background_image_path,
            raw_payload_json,
            first_seen_at,
            last_seen_at,
            is_current
        ) VALUES (?, ?, ?, ?, ?, ?, now(), now(), TRUE)
        ON CONFLICT (reward_track_path, content_hash) DO UPDATE SET
            xp_per_rank = EXCLUDED.xp_per_rank,
            battlepass_image_path = EXCLUDED.battlepass_image_path,
            background_image_path = EXCLUDED.background_image_path,
            raw_payload_json = EXCLUDED.raw_payload_json,
            last_seen_at = now(),
            is_current = TRUE
        """,
        [
            track_path,
            content_hash,
            _coerce_int(payload.get("XpPerRank")),
            _coerce_str(payload.get("BattlePassImage")),
            _coerce_str(payload.get("BackgroundImagePath")),
            _dump_payload(payload),
        ],
    )


def _upsert_track_translations(
    conn: Any,
    track_path: str,
    content_hash: str,
    payload: dict[str, Any],
) -> None:
    for lang, name in _extract_localized_texts(payload.get("Name")).items():
        conn.execute(
            """
            INSERT INTO battlepass_track_translations (
                reward_track_path,
                content_hash,
                lang,
                track_name,
                first_seen_at,
                last_seen_at
            ) VALUES (?, ?, ?, ?, now(), now())
            ON CONFLICT (reward_track_path, content_hash, lang) DO UPDATE SET
                track_name = EXCLUDED.track_name,
                last_seen_at = now()
            """,
            [track_path, content_hash, lang, name],
        )


def _upsert_item_definition(
    conn: Any,
    item_path: str,
    content_hash: str,
    payload: dict[str, Any],
) -> None:
    common = payload.get("CommonData") or {}
    conn.execute(
        "UPDATE battlepass_item_definitions SET is_current = FALSE WHERE inventory_item_path = ? AND content_hash <> ?",
        [item_path, content_hash],
    )
    conn.execute(
        """
        INSERT INTO battlepass_item_definitions (
            inventory_item_path,
            content_hash,
            quality,
            item_type,
            display_path,
            raw_payload_json,
            first_seen_at,
            last_seen_at,
            is_current
        ) VALUES (?, ?, ?, ?, ?, ?, now(), now(), TRUE)
        ON CONFLICT (inventory_item_path, content_hash) DO UPDATE SET
            quality = EXCLUDED.quality,
            item_type = EXCLUDED.item_type,
            display_path = EXCLUDED.display_path,
            raw_payload_json = EXCLUDED.raw_payload_json,
            last_seen_at = now(),
            is_current = TRUE
        """,
        [
            item_path,
            content_hash,
            _coerce_str(common.get("Quality") or common.get("Rarity")),
            _coerce_str(common.get("Type")),
            _extract_display_path(payload, common),
            _dump_payload(payload),
        ],
    )


def _upsert_item_translations(
    conn: Any,
    item_path: str,
    content_hash: str,
    payload: dict[str, Any],
) -> None:
    common = payload.get("CommonData") or {}
    titles = _extract_localized_texts(common.get("Title"))
    descriptions = _extract_localized_texts(common.get("Description"))
    for lang in sorted(set(titles) | set(descriptions)):
        title = titles.get(lang)
        description = descriptions.get(lang)
        if not title and not description:
            continue
        conn.execute(
            """
            INSERT INTO battlepass_item_translations (
                inventory_item_path,
                content_hash,
                lang,
                title,
                description,
                first_seen_at,
                last_seen_at
            ) VALUES (?, ?, ?, ?, ?, now(), now())
            ON CONFLICT (inventory_item_path, content_hash, lang) DO UPDATE SET
                title = EXCLUDED.title,
                description = EXCLUDED.description,
                last_seen_at = now()
            """,
            [item_path, content_hash, lang, title, description],
        )
