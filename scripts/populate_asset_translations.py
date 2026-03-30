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
          AND fetched_at >= now() - INTERVAL {_FRESHNESS_DAYS} DAY
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
        VALUES (?, ?, ?, ?, ?, now())
        ON CONFLICT (asset_id, asset_type, lang) DO UPDATE SET
            name = EXCLUDED.name,
            description = EXCLUDED.description,
            fetched_at = now()
        """,
        [asset_id, asset_type, lang, name, description],
    )


_CONCURRENCY = 10  # requêtes parallèles max par langue (intra-langue)

# Correspondance asset_type → clé dans MatchInfo du JSON de match stats
_MATCH_INFO_KEY_MAP: dict[str, str] = {
    "map": "MapVariant",
    "playlist": "Playlist",
    "pair": "PlaylistMapModePair",
    "game_variant": "UgcGameVariant",
}


async def _build_version_id_cache(
    asset_ids_by_type: dict[str, set[str]],
    client: Any,
) -> dict[tuple[str, str], str]:
    """Construit un cache {(asset_id, asset_type) -> version_id} via l'API match stats.

    Pour chaque (asset_id, asset_type), trouve un match_id dans match_registry,
    déduplique les match_ids, les fetche en parallèle, et extrait les VersionId.
    """
    if not SHARED_MATCHES_DB_PATH.exists():
        return {}

    # Récupérer un match_id représentatif par (asset_id, asset_type)
    match_id_per_asset: dict[tuple[str, str], str] = {}
    with duckdb.connect(str(SHARED_MATCHES_DB_PATH), read_only=True) as conn:
        for asset_type, asset_ids in asset_ids_by_type.items():
            col = _REGISTRY_COL_MAP[asset_type]
            rows = conn.execute(
                f"""
                SELECT {col}, match_id
                FROM (
                    SELECT {col}, match_id,
                           ROW_NUMBER() OVER (PARTITION BY {col}) AS rn
                    FROM match_registry
                    WHERE {col} IS NOT NULL
                )
                WHERE rn = 1
                """
            ).fetchall()
            for asset_id, match_id in rows:
                if asset_id in asset_ids:
                    match_id_per_asset[(asset_id, asset_type)] = match_id

    # Dédupliquer les match_ids — un match donne les version_ids de TOUS ses assets
    unique_match_ids: set[str] = set(match_id_per_asset.values())
    logger.info("Chargement version_ids via %d matchs API...", len(unique_match_ids))

    version_cache: dict[tuple[str, str], str] = {}
    sem = asyncio.Semaphore(_CONCURRENCY)

    async def _fetch_match(mid: str) -> dict[str, dict[str, str]]:
        """Retourne {asset_type: {asset_id: version_id}} depuis un match."""
        async with sem:
            stats_json = await client.get_match_stats(mid)
        if not isinstance(stats_json, dict):
            return {}
        match_info = stats_json.get("MatchInfo", {})
        result: dict[str, dict[str, str]] = {}
        for asset_type, json_key in _MATCH_INFO_KEY_MAP.items():
            ref = match_info.get(json_key)
            if isinstance(ref, dict):
                aid = ref.get("AssetId")
                vid = ref.get("VersionId")
                if aid and vid:
                    result.setdefault(asset_type, {})[aid] = vid
        return result

    match_results = await asyncio.gather(*[_fetch_match(mid) for mid in unique_match_ids])

    for per_type in match_results:
        for asset_type, id_version_map in per_type.items():
            for asset_id, version_id in id_version_map.items():
                version_cache[(asset_id, asset_type)] = version_id

    covered = sum(1 for key in match_id_per_asset if key in version_cache)
    total = len(match_id_per_asset)
    logger.info("version_ids couverts : %d/%d assets", covered, total)
    return version_cache


async def _fetch_lang_rows(
    asset_ids: set[str],
    asset_type: str,
    api_type: str,
    lang: str,
    already: set[str],
    version_cache: dict[tuple[str, str], str],
) -> list[tuple[str, str, str | None]]:
    """Fetch les traductions pour une langue. Retourne les lignes sans toucher la DB."""
    to_fetch = asset_ids - already
    if not to_fetch:
        return []

    rows: list[tuple[str, str, str | None]] = []
    sem = asyncio.Semaphore(_CONCURRENCY)

    async def _fetch_one(client: Any, asset_id: str) -> None:
        version_id = version_cache.get((asset_id, asset_type), "")
        if not version_id:
            logger.debug("version_id manquant pour %s %s — ignoré", asset_type, asset_id)
            return
        async with sem:
            try:
                asset_json = await client.get_asset(api_type, asset_id, version_id)
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

    return rows


async def _fetch_and_save_lang(
    lang: str,
    asset_ids: set[str],
    asset_type: str,
    api_type: str,
    already: set[str],
    conn: duckdb.DuckDBPyConnection,
    db_lock: asyncio.Lock,
    version_cache: dict[tuple[str, str], str],
    *,
    dry_run: bool,
) -> int:
    """Fetch et sauvegarde les traductions pour une langue. Commit immédiat après écriture."""
    rows = await _fetch_lang_rows(asset_ids, asset_type, api_type, lang, already, version_cache)
    if not rows:
        return 0
    if dry_run:
        return len(rows)
    async with db_lock:
        for asset_id, name, description in rows:
            _upsert_translation(conn, asset_id, asset_type, lang, name, description)
        conn.commit()
    logger.info("    [%s] %s : %d écrites ✓", lang, asset_type, len(rows))
    return len(rows)


async def populate_translations(
    asset_types: list[str],
    langs: list[str],
    conn: duckdb.DuckDBPyConnection,
    *,
    dry_run: bool,
    force: bool,
) -> dict[str, int]:
    """Peuple asset_translations pour les types et langues demandés.

    Toutes les langues sont fetchées en parallèle pour chaque type d'asset.
    Les écritures DB sont sérialisées via asyncio.Lock, avec commit après chaque langue.
    En cas de crash, la reprise est automatique (langues déjà commitées sont skippées).
    """
    totals: dict[str, int] = {}

    # Collecter tous les asset_ids d'abord pour construire le cache version_ids en une passe
    asset_ids_by_type = {
        asset_type: _get_asset_ids(asset_type) for asset_type in asset_types
    }

    # Construire le cache version_id via l'API match stats (une fois pour tous les types)
    async with create_api_client() as version_client:
        version_cache = await _build_version_id_cache(
            {at: ids for at, ids in asset_ids_by_type.items() if ids},
            version_client,
        )

    for asset_type in asset_types:
        asset_ids = asset_ids_by_type.get(asset_type, set())
        if not asset_ids:
            continue
        api_type = _ASSET_TYPE_MAP[asset_type]
        logger.info("=== %s (%d assets × %d langues en parallèle) ===", asset_type, len(asset_ids), len(langs))

        # Lire l'état courant de la DB pour toutes les langues (avant le fetch parallèle)
        already_by_lang = {
            lang: _get_already_fetched(conn, asset_type, lang, force=force)
            for lang in langs
        }
        for lang in langs:
            n_already = len(already_by_lang[lang])
            n_todo = len(asset_ids - already_by_lang[lang])
            if n_todo == 0:
                logger.info("    [%s] %s : tout déjà présent (%d)", lang, asset_type, n_already)
            else:
                logger.info("    [%s] %s : %d à récupérer (%d déjà présents)", lang, asset_type, n_todo, n_already)

        langs_to_fetch = [lg for lg in langs if asset_ids - already_by_lang[lg]]
        if not langs_to_fetch:
            totals[asset_type] = 0
            continue

        db_lock = asyncio.Lock()
        results = await asyncio.gather(*[
            _fetch_and_save_lang(
                lang, asset_ids, asset_type, api_type, already_by_lang[lang],
                conn, db_lock, version_cache, dry_run=dry_run,
            )
            for lang in langs_to_fetch
        ])
        totals[asset_type] = sum(results)
        logger.info("  %s terminé : %d traductions au total", asset_type, totals[asset_type])

    return totals


def _parse_langs(raw: str) -> list[str]:
    """Parse '--langs fr-FR,de-DE' en liste, valide chaque code."""
    valid = set(TARGET_LANGS)
    langs = [code.strip() for code in raw.split(",") if code.strip()]
    unknown = [code for code in langs if code not in valid]
    if unknown:
        logger.warning("Langues inconnues ignorées : %s", unknown)
    return [code for code in langs if code in valid] or [DEFAULT_LANG]


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
