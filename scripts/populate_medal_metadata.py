#!/usr/bin/env python3
"""Peuple medal_definitions et medal_translations dans metadata.duckdb.

Deux opérations distinctes :
1. ``populate_medal_definitions`` : maintient medal_definitions (medal_name_id, is_custom).
   Source : API gamecms_hacs (noms EN+FR uniquement pour compatibilité legacy).
2. ``populate_medal_translations`` : peuple le pivot multi-langue medal_translations.
   Source : champ ``translations`` de l'API (toutes les langues en un seul appel).
   Médailles custom (is_custom=True) : migrées depuis medal_definitions uniquement.

Usage:
    python scripts/populate_medal_metadata.py           # API + migration complète
    python scripts/populate_medal_metadata.py --dry-run
    python scripts/populate_medal_metadata.py --force   # écrase les entrées existantes
    python scripts/populate_medal_metadata.py --skip-api  # migration locale uniquement
    python scripts/populate_medal_metadata.py --verbose
"""

from __future__ import annotations

import argparse
import asyncio
import logging
import sys
from pathlib import Path
from typing import Any

logger = logging.getLogger("levelup.medals")

REPO_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(REPO_ROOT))

METADATA_DB = REPO_ROOT / "data" / "warehouse" / "metadata.duckdb"

_CUSTOM_MEDAL_THRESHOLD = 9_000_000_000

# Langues disponibles dans le champ translations de l'API médailles
# (format BCP-47 sans tiret, ex: "fr-FR" → "frFR" dans certaines versions)
# On normalise vers BCP-47 standard.
_API_LANG_MAP: dict[str, str] = {
    "pt-BR": "pt-BR",
    "zh-CN": "zh-CN",
    "zh-TW": "zh-TW",
    "de-DE": "de-DE",
    "fr-FR": "fr-FR",
    "it-IT": "it-IT",
    "ja-JP": "ja-JP",
    "ko-KR": "ko-KR",
    "es-MX": "es-MX",
    "nl-NL": "nl-NL",
    "pl-PL": "pl-PL",
    "ru-RU": "ru-RU",
    "es-ES": "es-ES",
}


# ─────────────────────────────────────────────────────────────────────────────
# Fetch API
# ─────────────────────────────────────────────────────────────────────────────


async def _fetch_medals_from_api() -> list[dict[str, Any]]:
    """Appelle gamecms_hacs.get_medal_metadata() et retourne la liste brute."""
    from src.data.sync.api_client import SPNKrAPIClient

    async with SPNKrAPIClient() as spnkr:
        resp = await spnkr.client.gamecms_hacs.get_medal_metadata()
        data = await resp.json()
    medals_raw = data.get("Medals") or data.get("medals") or []
    return medals_raw if isinstance(medals_raw, list) else []


# ─────────────────────────────────────────────────────────────────────────────
# medal_definitions (legacy EN+FR, maintenu pour is_custom)
# ─────────────────────────────────────────────────────────────────────────────


def _build_definitions_rows(
    medals_raw: list[dict[str, Any]],
) -> list[tuple[int, str, str, str, str, bool]]:
    """Construit les lignes pour medal_definitions depuis la liste API."""
    rows = []
    for m in medals_raw:
        name_id = m.get("nameId") or m.get("name_id")
        if not name_id:
            continue
        medal_id = int(name_id)

        name_obj = m.get("name") or {}
        desc_obj = m.get("description") or {}

        name_en = name_obj.get("value", "") if isinstance(name_obj, dict) else str(name_obj)
        desc_en = desc_obj.get("value", "") if isinstance(desc_obj, dict) else str(desc_obj)

        tr_name = name_obj.get("translations", {}) if isinstance(name_obj, dict) else {}
        tr_desc = desc_obj.get("translations", {}) if isinstance(desc_obj, dict) else {}

        name_fr = tr_name.get("fr-FR", name_en)
        desc_fr = tr_desc.get("fr-FR", desc_en)
        is_custom = medal_id >= _CUSTOM_MEDAL_THRESHOLD

        if not name_en and not name_fr:
            continue
        rows.append((medal_id, name_fr, name_en, desc_fr, desc_en, is_custom))
    return rows


def populate_medal_definitions(
    medals_raw: list[dict[str, Any]],
    conn: Any,
    *,
    force: bool = False,
) -> int:
    """Insère/met à jour medal_definitions depuis les données API brutes."""
    from src.data.sync.migrations import ensure_medal_definitions_table

    ensure_medal_definitions_table(conn)
    rows = _build_definitions_rows(medals_raw)
    if not rows:
        logger.warning("Aucune ligne medal_definitions construite depuis l'API")
        return 0

    sql = (
        "INSERT OR REPLACE INTO medal_definitions "
        "(medal_name_id, name_fr, name_en, description_fr, description_en, is_custom) "
        "VALUES (?, ?, ?, ?, ?, ?)"
        if force
        else "INSERT OR IGNORE INTO medal_definitions "
        "(medal_name_id, name_fr, name_en, description_fr, description_en, is_custom) "
        "VALUES (?, ?, ?, ?, ?, ?)"
    )
    conn.executemany(sql, rows)
    logger.info("medal_definitions : %d lignes insérées/ignorées", len(rows))
    return len(rows)


# ─────────────────────────────────────────────────────────────────────────────
# medal_translations (pivot multi-langue)
# ─────────────────────────────────────────────────────────────────────────────


def _upsert_medal_translation(
    conn: Any,
    medal_id: int,
    lang: str,
    name: str,
    description: str | None,
    *,
    force: bool,
) -> None:
    if force:
        conn.execute(
            "INSERT OR REPLACE INTO medal_translations "
            "(medal_name_id, lang, name, description) VALUES (?, ?, ?, ?)",
            [medal_id, lang, name, description],
        )
    else:
        conn.execute(
            "INSERT OR IGNORE INTO medal_translations "
            "(medal_name_id, lang, name, description) VALUES (?, ?, ?, ?)",
            [medal_id, lang, name, description],
        )


def _populate_official_translations(
    medals_raw: list[dict[str, Any]],
    conn: Any,
    *,
    force: bool,
) -> int:
    """Peuple medal_translations pour les médailles officielles depuis l'API."""
    count = 0
    for m in medals_raw:
        name_id = m.get("nameId") or m.get("name_id")
        if not name_id:
            continue
        medal_id = int(name_id)
        if medal_id >= _CUSTOM_MEDAL_THRESHOLD:
            continue  # Custom : géré séparément

        name_obj = m.get("name") or {}
        desc_obj = m.get("description") or {}

        name_en = name_obj.get("value", "") if isinstance(name_obj, dict) else str(name_obj)
        desc_en = desc_obj.get("value") if isinstance(desc_obj, dict) else None

        tr_name: dict[str, str] = name_obj.get("translations", {}) if isinstance(name_obj, dict) else {}
        tr_desc: dict[str, str] = desc_obj.get("translations", {}) if isinstance(desc_obj, dict) else {}

        if not name_en:
            continue

        # Langue de base : en-US (valeur par défaut de l'API)
        _upsert_medal_translation(conn, medal_id, "en-US", name_en, desc_en, force=force)
        count += 1

        # Toutes les langues disponibles dans translations
        for api_lang, bcp_lang in _API_LANG_MAP.items():
            name_tr = tr_name.get(api_lang)
            desc_tr = tr_desc.get(api_lang) or None
            if name_tr:
                _upsert_medal_translation(conn, medal_id, bcp_lang, name_tr, desc_tr, force=force)
                count += 1

    return count


def _migrate_custom_medals(conn: Any, *, force: bool) -> int:
    """Migre les médailles custom de medal_definitions → medal_translations.

    Les médailles custom n'ont pas d'endpoint API : on utilise les labels
    FR+EN déjà présents dans medal_definitions.
    """
    rows = conn.execute(
        "SELECT medal_name_id, name_fr, name_en, description_fr, description_en "
        "FROM medal_definitions WHERE is_custom = TRUE"
    ).fetchall()
    count = 0
    for medal_id, name_fr, name_en, desc_fr, desc_en in rows:
        if name_en:
            _upsert_medal_translation(
                conn, medal_id, "en-US", name_en, desc_en or None, force=force
            )
            count += 1
        if name_fr and name_fr != name_en:
            _upsert_medal_translation(
                conn, medal_id, "fr-FR", name_fr, desc_fr or None, force=force
            )
            count += 1
    logger.info("medal_translations : %d lignes custom migrées", count)
    return count


def populate_medal_translations(
    medals_raw: list[dict[str, Any]],
    conn: Any,
    *,
    force: bool = False,
) -> int:
    """Peuple medal_translations depuis les données API + migration custom."""
    from src.data.sync.migrations import ensure_medal_translations_table

    ensure_medal_translations_table(conn)

    official = _populate_official_translations(medals_raw, conn, force=force)
    logger.info("medal_translations : %d lignes officielles (toutes langues confondues)", official)

    custom = _migrate_custom_medals(conn, force=force)
    return official + custom


# ─────────────────────────────────────────────────────────────────────────────
# Mode skip-api : migration locale depuis medal_definitions
# ─────────────────────────────────────────────────────────────────────────────


def migrate_from_definitions_only(conn: Any, *, force: bool) -> int:
    """Migre medal_definitions → medal_translations sans appel API.

    Utilisé avec --skip-api. Peuple uniquement fr-FR et en-US.
    """
    from src.data.sync.migrations import ensure_medal_translations_table

    ensure_medal_translations_table(conn)
    rows = conn.execute(
        "SELECT medal_name_id, name_fr, name_en, description_fr, description_en "
        "FROM medal_definitions"
    ).fetchall()
    count = 0
    for medal_id, name_fr, name_en, desc_fr, desc_en in rows:
        if name_en:
            _upsert_medal_translation(
                conn, medal_id, "en-US", name_en, desc_en or None, force=force
            )
            count += 1
        if name_fr and name_fr != name_en:
            _upsert_medal_translation(
                conn, medal_id, "fr-FR", name_fr, desc_fr or None, force=force
            )
            count += 1
    logger.info("medal_translations (migration locale) : %d lignes", count)
    return count


# ─────────────────────────────────────────────────────────────────────────────
# CLI
# ─────────────────────────────────────────────────────────────────────────────


async def main_async(args: argparse.Namespace) -> int:
    import duckdb

    conn = duckdb.connect(":memory:") if args.dry_run else duckdb.connect(str(METADATA_DB))

    try:
        if args.skip_api:
            logger.info("Mode --skip-api : migration locale uniquement")
            n = migrate_from_definitions_only(conn, force=args.force)
            if not args.dry_run:
                conn.commit()
            logger.info("Migration locale : %d lignes écrites dans medal_translations", n)
            return 0

        logger.info("Récupération des métadonnées médailles depuis l'API…")
        try:
            medals_raw = await _fetch_medals_from_api()
        except Exception as exc:
            logger.error("Erreur API : %s — utilisez --skip-api pour migrer localement", exc)
            return 1

        if not medals_raw:
            logger.error("Aucune médaille retournée par l'API")
            return 1

        logger.info("%d médailles reçues de l'API", len(medals_raw))

        n_def = populate_medal_definitions(medals_raw, conn, force=args.force)
        n_tr = populate_medal_translations(medals_raw, conn, force=args.force)

        if not args.dry_run:
            conn.commit()

        logger.info("medal_definitions : %d lignes | medal_translations : %d lignes", n_def, n_tr)

        # Validation
        if not args.dry_run:
            total_tr = conn.execute("SELECT COUNT(*) FROM medal_translations").fetchone()[0]
            langs = conn.execute(
                "SELECT lang, COUNT(*) FROM medal_translations GROUP BY lang ORDER BY lang"
            ).fetchall()
            logger.info("medal_translations total : %d | par langue : %s", total_tr, langs)

    finally:
        conn.close()

    return 0


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Peuple medal_definitions + medal_translations dans metadata.duckdb"
    )
    parser.add_argument("--dry-run", action="store_true", help="Simule sans écriture")
    parser.add_argument("--force", action="store_true", help="Écrase les entrées existantes")
    parser.add_argument(
        "--skip-api",
        action="store_true",
        help="Migration locale uniquement (medal_definitions → medal_translations, pas d'appel API)",
    )
    parser.add_argument("--verbose", action="store_true", help="Logs détaillés")
    args = parser.parse_args()

    level = logging.DEBUG if args.verbose else logging.INFO
    logging.basicConfig(level=level, format="%(levelname)s - %(name)s - %(message)s")

    return asyncio.run(main_async(args))


if __name__ == "__main__":
    sys.exit(main())
