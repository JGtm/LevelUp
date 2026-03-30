#!/usr/bin/env python3
"""Peuplement de asset_translations depuis Discovery UGC API (multi-langue).

Pour chaque asset connu (maps, playlists, pairs, game_variants) récupéré depuis
match_registry, ce script appelle l'API avec Accept-Language pour chaque langue
cible et stocke le PublicName localisé dans asset_translations.

Usage:
    python scripts/populate_asset_translations.py
    python scripts/populate_asset_translations.py --langs fr-FR,de-DE
    python scripts/populate_asset_translations.py --asset-type map --langs fr-FR
    python scripts/populate_asset_translations.py --dry-run
    python scripts/populate_asset_translations.py --force  # re-fetch même si déjà présent
"""

from __future__ import annotations

import argparse
import asyncio
import logging
import sys
from pathlib import Path
from typing import Any

_SCRIPTS_DIR = Path(__file__).parent
sys.path.insert(0, str(_SCRIPTS_DIR.parent))
sys.path.insert(0, str(_SCRIPTS_DIR))

try:
    import duckdb
except ImportError:
    print("ERREUR: DuckDB non installé.")
    sys.exit(1)

from _metadata_db import create_metadata_db

from src.data.sync._asset_langs import DEFAULT_LANG, TARGET_LANGS
from src.data.sync.api_factory import create_api_client
from src.data.sync.migrations import ensure_asset_translations_table

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

WAREHOUSE_DIR = Path(__file__).parent.parent / "data" / "warehouse"
METADATA_DB_PATH = WAREHOUSE_DIR / "metadata.duckdb"
SHARED_MATCHES_DB_PATH = WAREHOUSE_DIR / "shared_matches_v2.duckdb"

# Fraîcheur des traductions : re-fetch si plus ancien que N jours
_FRESHNESS_DAYS = 30

_ASSET_TYPE_MAP: dict[str, str] = {
    "map": "Maps",
    "playlist": "Playlists",
    "pair": "PlaylistMapModePairs",
    "game_variant": "GameVariants",
}

_REGISTRY_COL_MAP: dict[str, str] = {
    "map": "map_id",
    "playlist": "playlist_id",
    "pair": "pair_id",
    "game_variant": "game_variant_id",
}


def _get_asset_ids(asset_type: str) -> set[str]:
    """Retourne les asset_ids distincts depuis match_registry pour un type donné."""
    col = _REGISTRY_COL_MAP[asset_type]
    if not SHARED_MATCHES_DB_PATH.exists():
        logger.warning("shared_matches_v2.duckdb non trouvé : %s", SHARED_MATCHES_DB_PATH)
        return set()
    with duckdb.connect(str(SHARED_MATCHES_DB_PATH), read_only=True) as conn:
        rows = conn.execute(
            f"SELECT DISTINCT {col} FROM match_registry WHERE {col} IS NOT NULL"
        ).fetchall()
    ids = {r[0] for r in rows if r[0]}
    logger.info("  %d asset_ids distincts pour '%s'", len(ids), asset_type)
    return ids


def _get_already_fetched(
    conn: duckdb.DuckDBPyConnection,
    asset_type: str,
    lang: str,
    *,
    force: bool,
) -> set[str]:
    """Retourne les asset_ids déjà présents (et frais) dans asset_translations."""
    if force:
        return set()
    rows = conn.execute(
        f"""
        SELECT asset_id FROM asset_translations
        WHERE asset_type = ? AND lang = ?
          AND fetched_at >= CURRENT_TIMESTAMP - INTERVAL {_FRESHNESS_DAYS} DAY
        """,
        [asset_type, lang],
    ).fetchall()
    return {r[0] for r in rows}


def _upsert_translation(
    conn: duckdb.DuckDBPyConnection,
    asset_id: str,
    asset_type: str,
    lang: str,
    name: str,
    description: str | None,
) -> None:
    conn.execute(
        """
        INSERT INTO asset_translations (asset_id, asset_type, lang, name, description, fetched_at)
        VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
        ON CONFLICT (asset_id, asset_type, lang) DO UPDATE SET
            name = EXCLUDED.name,
            description = EXCLUDED.description,
            fetched_at = CURRENT_TIMESTAMP
        """,
        [asset_id, asset_type, lang, name, description],
    )


_CONCURRENCY = 10  # requêtes parallèles max par langue


async def _fetch_one_lang(
    asset_ids: set[str],
    asset_type: str,
    api_type: str,
    lang: str,
    conn: duckdb.DuckDBPyConnection,
    *,
    dry_run: bool,
    force: bool,
) -> int:
    """Fetch et stocke les traductions pour un type d'asset + une langue."""
    already = _get_already_fetched(conn, asset_type, lang, force=force)
    to_fetch = asset_ids - already
    if not to_fetch:
        logger.info("    [%s] %s : tout déjà présent (%d)", lang, asset_type, len(already))
        return 0

    logger.info("    [%s] %s : %d à récupérer (%d déjà présents)", lang, asset_type, len(to_fetch), len(already))
    rows: list[tuple[str, str, str | None]] = []
    sem = asyncio.Semaphore(_CONCURRENCY)

    async def _fetch_one(client: Any, asset_id: str) -> None:
        async with sem:
            try:
                asset_json = await client.get_asset(api_type, asset_id, "")
            except Exception as exc:
                logger.debug("Erreur fetch %s %s [%s]: %s", api_type, asset_id, lang, exc)
                return
            if not isinstance(asset_json, dict):
                return
            name = asset_json.get("PublicName", "")
            if not name or not name.strip():
                logger.debug("PublicName vide pour %s %s [%s]", api_type, asset_id, lang)
                return
            rows.append((asset_id, name.strip(), asset_json.get("Description") or None))

    async with create_api_client(lang=lang) as client:
        await asyncio.gather(*[_fetch_one(client, aid) for aid in to_fetch])

    if not dry_run:
        for asset_id, name, description in rows:
            _upsert_translation(conn, asset_id, asset_type, lang, name, description)
    return len(rows)


async def populate_translations(
    asset_types: list[str],
    langs: list[str],
    conn: duckdb.DuckDBPyConnection,
    *,
    dry_run: bool,
    force: bool,
) -> dict[str, int]:
    """Peuple asset_translations pour les types et langues demandés."""
    totals: dict[str, int] = {}
    for asset_type in asset_types:
        api_type = _ASSET_TYPE_MAP[asset_type]
        asset_ids = _get_asset_ids(asset_type)
        if not asset_ids:
            continue
        logger.info("=== %s (%d assets) ===", asset_type, len(asset_ids))
        type_total = 0
        for lang in langs:
            n = await _fetch_one_lang(
                asset_ids, asset_type, api_type, lang, conn, dry_run=dry_run, force=force
            )
            type_total += n
        totals[asset_type] = type_total
        if not dry_run:
            conn.commit()
    return totals


def _parse_langs(raw: str) -> list[str]:
    """Parse '--langs fr-FR,de-DE' en liste, valide chaque code."""
    valid = set(TARGET_LANGS)
    langs = [l.strip() for l in raw.split(",") if l.strip()]
    unknown = [l for l in langs if l not in valid]
    if unknown:
        logger.warning("Langues inconnues ignorées : %s", unknown)
    return [l for l in langs if l in valid] or [DEFAULT_LANG]


async def main_async(args: argparse.Namespace) -> int:
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)

    langs = _parse_langs(args.langs) if args.langs else list(TARGET_LANGS)
    asset_types = [args.asset_type] if args.asset_type else list(_ASSET_TYPE_MAP)

    logger.info("Langues : %s", langs)
    logger.info("Types   : %s", asset_types)
    if args.dry_run:
        logger.info("[DRY-RUN] aucune écriture")

    WAREHOUSE_DIR.mkdir(parents=True, exist_ok=True)

    if args.dry_run:
        conn = duckdb.connect(":memory:")
        create_metadata_db(conn)
    else:
        conn = duckdb.connect(str(METADATA_DB_PATH))
        ensure_asset_translations_table(conn)

    try:
        totals = await populate_translations(
            asset_types, langs, conn, dry_run=args.dry_run, force=args.force
        )
    finally:
        conn.close()

    logger.info("\n=== RÉSUMÉ ===")
    for asset_type, n in totals.items():
        label = "[DRY-RUN] " if args.dry_run else ""
        logger.info("  %s%s : %d traductions", label, asset_type, n)
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Peuple asset_translations depuis Discovery UGC API (multi-langue)"
    )
    parser.add_argument(
        "--langs",
        metavar="fr-FR,de-DE,...",
        help="Langues BCP-47 séparées par des virgules (défaut : toutes TARGET_LANGS)",
    )
    parser.add_argument(
        "--asset-type",
        choices=list(_ASSET_TYPE_MAP),
        help="Limiter à un type d'asset",
    )
    parser.add_argument("--dry-run", action="store_true", help="Simule sans écrire")
    parser.add_argument("--force", action="store_true", help="Re-fetch même si déjà présent")
    parser.add_argument("--verbose", "-v", action="store_true")
    args = parser.parse_args()
    return asyncio.run(main_async(args))


if __name__ == "__main__":
    sys.exit(main())
