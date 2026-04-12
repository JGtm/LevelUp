"""Catalogue des défis Halo : extraction, i18n et référentiel metadata."""

from __future__ import annotations

import hashlib
import json
import logging
import re
from dataclasses import dataclass
from typing import Any

from src.data.sync._asset_langs import DEFAULT_LANG, to_bcp47
from src.data.sync.challenge_migrations import (
    ensure_challenge_definitions_table,
    ensure_challenge_translations_table,
)
from src.utils.db import duckdb_read_only, duckdb_read_write
from src.utils.paths import get_metadata_db_path

logger = logging.getLogger(__name__)

_SHORT_LANG_RE = re.compile(r"^[a-z]{2,3}$")
_COMPACT_LANG_RE = re.compile(r"^([a-z]{2})([A-Z]{2})$")


@dataclass(frozen=True)
class StoredChallengeDefinition:
    """Définition de défi relue depuis metadata.duckdb."""

    challenge_path: str
    content_hash: str
    category: str | None = None
    difficulty: str | None = None
    threshold_for_success: int | None = None
    reward_xp: int = 0
    secondary_reward_xp: int = 0
    title: str | None = None
    description: str | None = None


def collect_challenge_paths_from_decks(data: dict) -> list[str]:
    """Retourne tous les paths de défis visibles dans les decks joueur."""
    decks = data.get("AssignedDecks")
    if not isinstance(decks, list):
        return []

    paths: list[str] = []
    for deck in decks:
        for key in ("ActiveChallenges", "CompletedChallenges", "UpcomingChallenges"):
            for challenge in deck.get(key) or []:
                path = challenge.get("Path")
                if isinstance(path, str) and path.strip():
                    paths.append(path.strip())
    return list(dict.fromkeys(paths))


def extract_challenge_xp(data: dict) -> int:
    """Extrait l'XP d'une définition CMS de défi."""
    reward = data.get("Reward") or {}
    xp = reward.get("SoftExperience")
    if isinstance(xp, int):
        return xp
    secondary = data.get("SecondaryReward") or {}
    xp = secondary.get("SoftExperience")
    return xp if isinstance(xp, int) else 0


def extract_secondary_challenge_xp(data: dict) -> int:
    """Extrait l'XP secondaire d'une définition CMS si présente."""
    secondary = data.get("SecondaryReward") or {}
    xp = secondary.get("SoftExperience")
    return xp if isinstance(xp, int) else 0


def extract_threshold_for_success(data: dict) -> int | None:
    """Extrait le seuil de complétion d'une définition CMS de défi."""
    threshold = data.get("ThresholdForSuccess")
    if isinstance(threshold, int):
        return threshold
    if isinstance(threshold, float) and threshold.is_integer():
        return int(threshold)
    if isinstance(threshold, dict):
        for key in ("value", "Value", "threshold", "Threshold", "count", "Count"):
            coerced = _coerce_int(threshold.get(key))
            if coerced is not None:
                return coerced
    return None


def resolve_localized_value(data: Any, lang: str) -> str | None:
    """Résout une valeur localisée CMS vers la langue demandée."""
    if isinstance(data, str):
        text = data.strip()
        return text or None
    if not isinstance(data, dict):
        return None

    translations = {
        normalize_challenge_lang(key): value
        for key, value in (data.get("translations") or {}).items()
        if isinstance(value, str) and value.strip()
    }
    requested = normalize_challenge_lang(lang)
    for key in _language_candidates(requested):
        value = translations.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()

    fallback = data.get("value")
    if isinstance(fallback, str) and fallback.strip():
        return fallback.strip()
    return None


def build_challenge_badge_candidates(
    challenge_path: str | None,
    category: str | None,
    difficulty: str | None,
) -> list[str]:
    """Construit des stems candidats pour les badges Waypoint d'un défi."""
    normalized_path = (challenge_path or "").replace("\\", "/").lower()
    cat = _slugify_token(category)
    diff = _slugify_token(difficulty)
    weekly_family = _infer_weekly_family(normalized_path)
    candidates: list[str] = []

    if "dailychallenges" in normalized_path and diff:
        candidates.append(f"daily-{diff}")
    if "weeklychallenges" in normalized_path and weekly_family and diff:
        candidates.append(f"weekly-{weekly_family}-{diff}")
    if "ultimate" in normalized_path or "capstone" in normalized_path:
        candidates.append(f"capstone-{diff or 'mythic'}")

    if cat == "daily" and diff:
        candidates.append(f"daily-{diff}")
    if cat == "weekly" and weekly_family and diff:
        candidates.append(f"weekly-{weekly_family}-{diff}")
    if cat in {"ultimate", "capstone"}:
        candidates.append(f"capstone-{diff or 'mythic'}")
    if diff == "mythic":
        candidates.append("capstone-mythic")
    if cat and diff:
        candidates.append(f"{cat}-{diff}")

    return list(dict.fromkeys(candidate for candidate in candidates if candidate))


def normalize_challenge_lang(lang: str | None) -> str:
    """Normalise un code langue de défi vers BCP-47."""
    value = str(lang or "").strip().replace("_", "-")
    if not value:
        return DEFAULT_LANG
    if _SHORT_LANG_RE.match(value.lower()):
        return to_bcp47(value.lower())
    compact = _COMPACT_LANG_RE.match(value)
    if compact:
        return f"{compact.group(1).lower()}-{compact.group(2).upper()}"
    parts = value.split("-")
    if len(parts) == 2 and len(parts[0]) == 2 and len(parts[1]) == 2:
        return f"{parts[0].lower()}-{parts[1].upper()}"
    return value


def persist_challenge_catalog(definitions: dict[str, dict]) -> dict[str, str]:
    """Persiste les définitions et traductions de défis dans metadata.duckdb."""
    if not definitions:
        return {}

    metadata_path = get_metadata_db_path()
    metadata_path.parent.mkdir(parents=True, exist_ok=True)
    hashes: dict[str, str] = {}
    try:
        with duckdb_read_write(metadata_path) as conn:
            ensure_challenge_definitions_table(conn)
            ensure_challenge_translations_table(conn)
            for challenge_path, payload in definitions.items():
                content_hash = _build_content_hash(payload)
                hashes[challenge_path] = content_hash
                _upsert_challenge_definition(conn, challenge_path, content_hash, payload)
                _upsert_challenge_translations(conn, challenge_path, content_hash, payload)
    except Exception as exc:
        logger.debug("challenge_catalog: persistance metadata ignorée: %s", exc)
        return {}
    return hashes


def load_challenge_metadata_map(
    challenge_paths: list[str],
    lang: str = "fr",
) -> dict[str, StoredChallengeDefinition]:
    """Charge les définitions courantes depuis metadata.duckdb avec fallback EN."""
    if not challenge_paths:
        return {}

    metadata_path = get_metadata_db_path()
    if not metadata_path.exists():
        return {}

    unique_paths = list(dict.fromkeys(path for path in challenge_paths if path))
    bcp_lang = normalize_challenge_lang(lang)
    try:
        with duckdb_read_only(metadata_path) as conn:
            if not _challenge_tables_available(conn):
                return {}
            rows = _fetch_challenge_metadata_rows(conn, unique_paths, bcp_lang)
            entries = {row.challenge_path: row for row in rows}
            if bcp_lang != DEFAULT_LANG:
                _merge_default_language_entries(conn, unique_paths, entries)
            return entries
    except Exception as exc:
        logger.debug("challenge_catalog: lecture metadata ignorée: %s", exc)
        return {}


def _challenge_tables_available(conn: Any) -> bool:
    tables = {
        row[0]
        for row in conn.execute(
            "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'"
        ).fetchall()
    }
    return {"challenge_definitions", "challenge_translations"}.issubset(tables)


def _fetch_challenge_metadata_rows(
    conn: Any,
    challenge_paths: list[str],
    lang: str,
) -> list[StoredChallengeDefinition]:
    placeholders = ", ".join("?" for _ in challenge_paths)
    rows = conn.execute(
        f"""
        SELECT
            d.challenge_path,
            d.content_hash,
            d.category,
            d.difficulty,
            d.threshold_for_success,
            d.reward_xp,
            d.secondary_reward_xp,
            t.title,
            t.description
        FROM challenge_definitions d
        LEFT JOIN challenge_translations t
          ON d.challenge_path = t.challenge_path
         AND d.content_hash = t.content_hash
         AND t.lang = ?
        WHERE d.is_current = TRUE AND d.challenge_path IN ({placeholders})
        """,
        [lang, *challenge_paths],
    ).fetchall()
    return [
        StoredChallengeDefinition(
            challenge_path=row[0],
            content_hash=row[1],
            category=row[2],
            difficulty=row[3],
            threshold_for_success=row[4],
            reward_xp=row[5] or 0,
            secondary_reward_xp=row[6] or 0,
            title=row[7],
            description=row[8],
        )
        for row in rows
    ]


def _merge_default_language_entries(
    conn: Any,
    unique_paths: list[str],
    entries: dict[str, StoredChallengeDefinition],
) -> None:
    missing = [path for path in unique_paths if path not in entries or not entries[path].title]
    if not missing:
        return
    for row in _fetch_challenge_metadata_rows(conn, missing, DEFAULT_LANG):
        current = entries.get(row.challenge_path)
        if current is None:
            entries[row.challenge_path] = row
            continue
        entries[row.challenge_path] = StoredChallengeDefinition(
            challenge_path=current.challenge_path,
            content_hash=current.content_hash,
            category=current.category,
            difficulty=current.difficulty,
            threshold_for_success=current.threshold_for_success,
            reward_xp=current.reward_xp,
            secondary_reward_xp=current.secondary_reward_xp,
            title=current.title or row.title,
            description=current.description or row.description,
        )


def _upsert_challenge_definition(
    conn: Any,
    challenge_path: str,
    content_hash: str,
    payload: dict,
) -> None:
    conn.execute(
        "UPDATE challenge_definitions SET is_current = FALSE WHERE challenge_path = ? AND content_hash <> ?",
        [challenge_path, content_hash],
    )
    conn.execute(
        """
        INSERT INTO challenge_definitions (
            challenge_path,
            content_hash,
            category,
            difficulty,
            threshold_for_success,
            reward_xp,
            secondary_reward_xp,
            first_seen_at,
            last_seen_at,
            is_current
        )
        VALUES (?, ?, ?, ?, ?, ?, ?, now(), now(), TRUE)
        ON CONFLICT (challenge_path, content_hash) DO UPDATE SET
            category = EXCLUDED.category,
            difficulty = EXCLUDED.difficulty,
            threshold_for_success = EXCLUDED.threshold_for_success,
            reward_xp = EXCLUDED.reward_xp,
            secondary_reward_xp = EXCLUDED.secondary_reward_xp,
            last_seen_at = now(),
            is_current = TRUE
        """,
        [
            challenge_path,
            content_hash,
            payload.get("Category"),
            payload.get("Difficulty"),
            extract_threshold_for_success(payload),
            extract_challenge_xp(payload),
            extract_secondary_challenge_xp(payload),
        ],
    )


def _upsert_challenge_translations(
    conn: Any,
    challenge_path: str,
    content_hash: str,
    payload: dict,
) -> None:
    titles = _extract_localized_texts(payload.get("Title"), default_lang=DEFAULT_LANG)
    descriptions = _extract_localized_texts(payload.get("Description"), default_lang=DEFAULT_LANG)
    for lang in sorted(set(titles) | set(descriptions)):
        title = titles.get(lang)
        description = descriptions.get(lang)
        if not title and not description:
            continue
        conn.execute(
            """
            INSERT INTO challenge_translations (
                challenge_path,
                content_hash,
                lang,
                title,
                description,
                first_seen_at,
                last_seen_at
            )
            VALUES (?, ?, ?, ?, ?, now(), now())
            ON CONFLICT (challenge_path, content_hash, lang) DO UPDATE SET
                title = EXCLUDED.title,
                description = EXCLUDED.description,
                last_seen_at = now()
            """,
            [challenge_path, content_hash, lang, title, description],
        )


def _extract_localized_texts(data: Any, default_lang: str) -> dict[str, str]:
    if isinstance(data, str):
        text = data.strip()
        return {default_lang: text} if text else {}
    if not isinstance(data, dict):
        return {}

    results: dict[str, str] = {}
    translations = data.get("translations") or {}
    if isinstance(translations, dict):
        for raw_lang, raw_value in translations.items():
            if not isinstance(raw_value, str) or not raw_value.strip():
                continue
            results[normalize_challenge_lang(str(raw_lang))] = raw_value.strip()
    fallback = data.get("value")
    if isinstance(fallback, str) and fallback.strip():
        results.setdefault(default_lang, fallback.strip())
    return results


def _build_content_hash(payload: dict) -> str:
    canonical = json.dumps(payload, sort_keys=True, separators=(",", ":"), ensure_ascii=True)
    return hashlib.sha1(canonical.encode("utf-8")).hexdigest()


def _language_candidates(lang: str) -> tuple[str, ...]:
    if lang == "fr-FR":
        return ("fr-FR", "fr")
    if lang == "en-US":
        return ("en-US", "en-GB", "en")
    short = lang.split("-", 1)[0]
    return (lang, short)


def _slugify_token(value: str | None) -> str | None:
    if not isinstance(value, str):
        return None
    slug = value.strip().lower().replace("_", "-").replace(" ", "-")
    cleaned = "".join(char for char in slug if char.isalnum() or char == "-").strip("-")
    return cleaned or None


def _infer_weekly_family(normalized_path: str) -> str | None:
    for token in ("action", "gametype", "weapon"):
        if f"/{token}/" in normalized_path:
            return token
    return None


def _coerce_int(value: Any) -> int | None:
    if isinstance(value, int):
        return value
    if isinstance(value, float) and value.is_integer():
        return int(value)
    return None
