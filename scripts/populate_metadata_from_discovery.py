#!/usr/bin/env python3
"""
Script pour créer et peupler metadata.duckdb depuis Discovery UGC.

Ce script :
1. Crée metadata.duckdb s'il n'existe pas
2. Crée les tables nécessaires (playlists, maps, playlist_map_mode_pairs, game_variants)
   avec colonnes i18n (name_en, name_fr, mode_name, playlist_canonical_fr, ...)
3. Récupère les asset IDs uniques depuis shared_matches.duckdb (marché v5.1+)
4. Peuple les tables depuis Discovery UGC API
5. Enrichit les colonnes i18n depuis les tables mode_translations / playlist_translations
6. Écrit metadata_populate_errors.txt à la racine pour les cas non résolus

Usage:
    python scripts/populate_metadata_from_discovery.py
    python scripts/populate_metadata_from_discovery.py --dry-run
"""

from __future__ import annotations

import argparse
import asyncio
import logging
import sys
from pathlib import Path
from typing import Any

# Ajouter le répertoire parent au path (src/) et le dossier scripts/ (_metadata_db)
_SCRIPTS_DIR = Path(__file__).parent
sys.path.insert(0, str(_SCRIPTS_DIR.parent))
sys.path.insert(0, str(_SCRIPTS_DIR))

try:
    import duckdb
except ImportError:
    print("ERREUR: DuckDB non installé. Exécutez: pip install duckdb")
    sys.exit(1)

from _metadata_db import create_metadata_db, enrich_i18n

from src.data.sync.api_client import get_tokens_from_env
from src.data.sync.api_factory import create_api_client
from src.data.sync.api_port import HaloAPIPort

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

# Configuration des chemins
DATA_DIR = Path(__file__).parent.parent / "data"
WAREHOUSE_DIR = DATA_DIR / "warehouse"
METADATA_DB_PATH = WAREHOUSE_DIR / "metadata.duckdb"
SHARED_MATCHES_DB_PATH = WAREHOUSE_DIR / "shared_matches.duckdb"


def get_unique_asset_ids() -> dict[str, set[str]]:
    """Récupère les asset IDs uniques depuis shared_matches.duckdb.

    Compatible v5.1+ : lit match_registry (shared_matches.duckdb),
    pas match_stats des DBs joueurs (supprimée en v5.1).
    """
    assets: dict[str, set[str]] = {
        "playlists": set(),
        "maps": set(),
        "pairs": set(),
        "variants": set(),
    }

    if not SHARED_MATCHES_DB_PATH.exists():
        logger.warning("shared_matches.duckdb non trouvé : %s", SHARED_MATCHES_DB_PATH)
        return assets

    column_map = {
        "playlist_id": "playlists",
        "map_id": "maps",
        "pair_id": "pairs",
        "game_variant_id": "variants",
    }

    with duckdb.connect(str(SHARED_MATCHES_DB_PATH), read_only=True) as conn:
        for col, key in column_map.items():
            rows = conn.execute(
                f"SELECT DISTINCT {col} FROM match_registry WHERE {col} IS NOT NULL"
            ).fetchall()
            for (asset_id,) in rows:
                if asset_id:
                    assets[key].add(asset_id)

    logger.info(
        "Asset IDs depuis match_registry : %d playlists, %d maps, %d pairs, %d variants",
        len(assets["playlists"]),
        len(assets["maps"]),
        len(assets["pairs"]),
        len(assets["variants"]),
    )
    return assets


async def fetch_asset_from_api(
    client: HaloAPIPort,
    asset_type: str,
    asset_id: str,
    version_id: str,
) -> dict[str, Any] | None:
    """Récupère un asset depuis Discovery UGC API.

    Args:
        client: Client API SPNKr.
        asset_type: Type d'asset ('Playlists', 'Maps', 'PlaylistMapModePairs', 'GameVariants').
        asset_id: ID de l'asset.
        version_id: Version de l'asset (peut être vide).

    Returns:
        JSON de l'asset ou None si erreur.
    """
    try:
        asset = await client.get_asset(asset_type, asset_id, version_id)
        return asset
    except Exception as e:
        logger.debug(f"Erreur récupération {asset_type} {asset_id}: {e}")
        return None


def _extract_asset_fields(
    asset_key: str, asset_json: dict[str, Any]
) -> tuple[str, str, str, dict[str, Any]]:
    """Extrait (public_name, description, version_id, extra) depuis le JSON API."""
    public_name = asset_json.get("PublicName", "")
    description = asset_json.get("Description", "")
    version_id = asset_json.get("VersionId", "")
    extra: dict[str, Any] = {}
    if asset_key == "playlists":
        tags = asset_json.get("Tags", [])
        extra["is_ranked"] = "ranked" in [t.lower() for t in tags if isinstance(t, str)]
        extra["category"] = asset_json.get("Category", "")
    elif asset_key == "maps":
        extra["thumbnail_path"] = asset_json.get("ThumbnailPath", "")
    elif asset_key == "variants":
        extra["category"] = asset_json.get("Category", "")
    return public_name, description, version_id, extra


def _upsert_asset(  # noqa: PLR0912, PLR0913
    conn: duckdb.DuckDBPyConnection,
    asset_key: str,
    asset_id: str,
    ver_id: str,
    public_name: str,
    desc: str,
    raw: str,
    extra: dict[str, Any],
) -> None:
    """INSERT OR UPDATE un asset dans la table correspondante."""
    if asset_key == "playlists":
        conn.execute(
            "INSERT INTO playlists"
            " (asset_id,version_id,public_name,name_en,description,is_ranked,category,raw_json,updated_at)"
            " VALUES (?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)"
            " ON CONFLICT (asset_id,version_id) DO UPDATE SET"
            " public_name=EXCLUDED.public_name,name_en=EXCLUDED.name_en,"
            " description=EXCLUDED.description,is_ranked=EXCLUDED.is_ranked,"
            " category=EXCLUDED.category,raw_json=EXCLUDED.raw_json,updated_at=CURRENT_TIMESTAMP",
            [
                asset_id,
                ver_id,
                public_name,
                public_name,
                desc,
                extra.get("is_ranked", False),
                extra.get("category"),
                raw,
            ],
        )
    elif asset_key == "maps":
        conn.execute(
            "INSERT INTO maps"
            " (asset_id,version_id,public_name,name_en,description,thumbnail_path,raw_json,updated_at)"
            " VALUES (?,?,?,?,?,?,?,CURRENT_TIMESTAMP)"
            " ON CONFLICT (asset_id,version_id) DO UPDATE SET"
            " public_name=EXCLUDED.public_name,name_en=EXCLUDED.name_en,"
            " description=EXCLUDED.description,thumbnail_path=EXCLUDED.thumbnail_path,"
            " raw_json=EXCLUDED.raw_json,updated_at=CURRENT_TIMESTAMP",
            [asset_id, ver_id, public_name, public_name, desc, extra.get("thumbnail_path"), raw],
        )
    elif asset_key == "pairs":
        conn.execute(
            "INSERT INTO playlist_map_mode_pairs"
            " (asset_id,version_id,public_name,name_en,description,raw_json,updated_at)"
            " VALUES (?,?,?,?,?,?,CURRENT_TIMESTAMP)"
            " ON CONFLICT (asset_id,version_id) DO UPDATE SET"
            " public_name=EXCLUDED.public_name,name_en=EXCLUDED.name_en,"
            " description=EXCLUDED.description,raw_json=EXCLUDED.raw_json,updated_at=CURRENT_TIMESTAMP",
            [asset_id, ver_id, public_name, public_name, desc, raw],
        )
    elif asset_key == "variants":
        conn.execute(
            "INSERT INTO game_variants"
            " (asset_id,version_id,public_name,name_en,description,category,raw_json,updated_at)"
            " VALUES (?,?,?,?,?,?,?,CURRENT_TIMESTAMP)"
            " ON CONFLICT (asset_id,version_id) DO UPDATE SET"
            " public_name=EXCLUDED.public_name,name_en=EXCLUDED.name_en,"
            " description=EXCLUDED.description,category=EXCLUDED.category,"
            " raw_json=EXCLUDED.raw_json,updated_at=CURRENT_TIMESTAMP",
            [asset_id, ver_id, public_name, public_name, desc, extra.get("category"), raw],
        )


async def populate_metadata_from_api(
    conn: duckdb.DuckDBPyConnection,
    client: HaloAPIPort,
    assets: dict[str, set[str]],
    dry_run: bool = False,
) -> dict[str, int]:
    """Peuple metadata.duckdb depuis Discovery UGC API.

    Args:
        conn: Connexion DuckDB.
        client: Client API SPNKr.
        assets: Dict asset IDs à récupérer.
        dry_run: Si True, ne fait que simuler.

    Returns:
        Nombre d'assets insérés par type.
    """
    results: dict[str, int] = dict.fromkeys(("playlists", "maps", "pairs", "variants"), 0)
    type_map = {
        "playlists": "Playlists",
        "maps": "Maps",
        "pairs": "PlaylistMapModePairs",
        "variants": "GameVariants",
    }
    for asset_key, api_type in type_map.items():
        asset_set = assets.get(asset_key, set())
        if not asset_set:
            continue
        logger.info("Récupération de %d %s...", len(asset_set), asset_key)
        for asset_id in asset_set:
            asset_json = await fetch_asset_from_api(client, api_type, asset_id, "")
            if not asset_json:
                logger.debug("Asset non trouvé: %s %s", api_type, asset_id)
                continue
            public_name, desc, ver_id, extra = _extract_asset_fields(asset_key, asset_json)
            if not dry_run:
                try:
                    _upsert_asset(
                        conn, asset_key, asset_id, ver_id, public_name, desc, str(asset_json), extra
                    )
                    results[asset_key] += 1
                    if results[asset_key] % 10 == 0:
                        logger.info("  %d %s insérés...", results[asset_key], asset_key)
                except Exception as e:
                    logger.warning("Erreur insertion %s %s: %s", asset_key, asset_id, e)
            else:
                results[asset_key] += 1
            await asyncio.sleep(0.1)
    return results


async def main_async(
    dry_run: bool = False,
    verbose: bool = False,
) -> int:
    """Point d'entrée principal async."""
    if verbose:
        logging.getLogger().setLevel(logging.DEBUG)

    logger.info("=" * 50)
    logger.info("Peuplement metadata.duckdb depuis Discovery UGC")
    logger.info("=" * 50)

    # Vérifier les tokens API
    tokens = get_tokens_from_env()
    if not tokens:
        logger.error("Tokens API non trouvés. Configurez HALO_SPARTAN_TOKEN et HALO_XBL_TOKEN")
        return 1

    # Créer le dossier warehouse si nécessaire
    WAREHOUSE_DIR.mkdir(parents=True, exist_ok=True)

    # Créer ou ouvrir metadata.duckdb
    if dry_run:
        conn = duckdb.connect(":memory:")
        logger.info("[DRY-RUN] Utilisation d'une base en mémoire")
    else:
        conn = duckdb.connect(str(METADATA_DB_PATH))
        logger.info("Ouverture de: %s", METADATA_DB_PATH)

    try:
        # Créer le schéma si nécessaire
        create_metadata_db(conn)

        # Récupérer les asset IDs depuis shared_matches.duckdb
        logger.info("Récupération des asset IDs depuis shared_matches.duckdb...")
        assets = get_unique_asset_ids()

        if not any(assets.values()):
            logger.warning(
                "Aucun asset ID trouvé. Vérifiez que %s est bien peuplé.",
                SHARED_MATCHES_DB_PATH,
            )
            return 1

        logger.info(
            "Assets à récupérer : %d playlists, %d maps, %d pairs, %d variants",
            len(assets.get("playlists", set())),
            len(assets.get("maps", set())),
            len(assets.get("pairs", set())),
            len(assets.get("variants", set())),
        )

        # Créer le client API
        client = create_api_client(tokens=tokens)

        # Peupler depuis l'API
        logger.info("Récupération des métadonnées depuis Discovery UGC...")
        results = await populate_metadata_from_api(conn, client, assets, dry_run=dry_run)

        if not dry_run:
            conn.commit()
            # Enrichir les colonnes i18n depuis les tables de traduction
            logger.info("Enrichissement i18n...")
            enrich_i18n(conn)
            conn.commit()

        # Afficher le résumé
        logger.info("\n" + "=" * 50)
        logger.info("RÉSUMÉ")
        logger.info("=" * 50)
        for asset_type, count in results.items():
            logger.info("%s : %d assets", asset_type, count)

    finally:
        conn.close()

    return 0


def main() -> int:
    """Point d'entrée principal."""
    parser = argparse.ArgumentParser(
        description="Peuple metadata.duckdb depuis Discovery UGC API",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Simule sans écrire (base en mémoire)",
    )
    parser.add_argument(
        "--verbose",
        "-v",
        action="store_true",
        help="Affiche plus de détails",
    )

    args = parser.parse_args()

    return asyncio.run(
        main_async(
            dry_run=args.dry_run,
            verbose=args.verbose,
        )
    )


if __name__ == "__main__":
    sys.exit(main())
