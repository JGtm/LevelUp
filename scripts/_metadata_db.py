"""DDL et enrichissement i18n pour metadata.duckdb.

Contient :
- METADATA_SCHEMA_DDL : schéma des 4 tables (playlists, maps, pairs, game_variants)
- create_metadata_db : création idempotente du schéma
- enrich_i18n : enrichissement FR depuis mode_translations / playlist_translations
"""

from __future__ import annotations

import logging
from datetime import datetime
from pathlib import Path
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    import duckdb

logger = logging.getLogger(__name__)

ERRORS_FILE_PATH = Path(__file__).parent.parent / "metadata_populate_errors.txt"

METADATA_SCHEMA_DDL = """
-- Table playlists
CREATE TABLE IF NOT EXISTS playlists (
    asset_id VARCHAR NOT NULL,
    version_id VARCHAR NOT NULL,
    public_name VARCHAR,
    name_en VARCHAR,                  -- = public_name (alias explicite, stable)
    name_fr VARCHAR,                  -- depuis playlist_translations (enrichi post-populate)
    playlist_canonical_en VARCHAR,    -- préfixe avant ':' dans public_name
    playlist_canonical_fr VARCHAR,    -- = name_fr (même regroupement sémantique)
    description VARCHAR,
    is_ranked BOOLEAN DEFAULT FALSE,
    category VARCHAR,
    raw_json JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (asset_id, version_id)
);

-- Table maps
CREATE TABLE IF NOT EXISTS maps (
    asset_id VARCHAR NOT NULL,
    version_id VARCHAR NOT NULL,
    public_name VARCHAR,
    name_en VARCHAR,                  -- = public_name
    name_fr VARCHAR,                  -- NULL par défaut (pas de source disponible)
    description VARCHAR,
    thumbnail_path VARCHAR,
    raw_json JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (asset_id, version_id)
);

-- Table playlist_map_mode_pairs
CREATE TABLE IF NOT EXISTS playlist_map_mode_pairs (
    asset_id VARCHAR NOT NULL,
    version_id VARCHAR NOT NULL,
    public_name VARCHAR,
    name_en VARCHAR,                  -- = public_name
    name_fr VARCHAR,                  -- depuis mode_translations (correspondance exacte)
    description VARCHAR,
    raw_json JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (asset_id, version_id)
);

-- Table game_variants
CREATE TABLE IF NOT EXISTS game_variants (
    asset_id VARCHAR NOT NULL,
    version_id VARCHAR NOT NULL,
    public_name VARCHAR,
    name_en VARCHAR,                  -- = public_name
    name_fr VARCHAR,                  -- dérivé via mode_translations (post-populate)
    mode_name VARCHAR,                -- partie après ':' dans public_name (ex: 'Slayer')
    mode_name_fr VARCHAR,             -- traduction de mode_name (post-populate)
    description VARCHAR,
    category VARCHAR,
    raw_json JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (asset_id, version_id)
);

-- Index pour recherche rapide
CREATE INDEX IF NOT EXISTS idx_playlists_asset_id ON playlists(asset_id);
CREATE INDEX IF NOT EXISTS idx_maps_asset_id ON maps(asset_id);
CREATE INDEX IF NOT EXISTS idx_pairs_asset_id ON playlist_map_mode_pairs(asset_id);
CREATE INDEX IF NOT EXISTS idx_variants_asset_id ON game_variants(asset_id);

-- Table pivot multi-langue pour tous les assets Discovery UGC (v6.3)
CREATE TABLE IF NOT EXISTS asset_translations (
    asset_id    VARCHAR NOT NULL,
    asset_type  VARCHAR NOT NULL,
    lang        VARCHAR NOT NULL,
    name        VARCHAR NOT NULL,
    description VARCHAR,
    fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (asset_id, asset_type, lang)
);
CREATE INDEX IF NOT EXISTS idx_asset_tr_id_type
    ON asset_translations(asset_id, asset_type);

-- Table pivot multi-langue pour les médailles (v6.3)
CREATE TABLE IF NOT EXISTS medal_translations (
    medal_name_id BIGINT  NOT NULL,
    lang          VARCHAR NOT NULL,
    name          VARCHAR NOT NULL,
    description   VARCHAR,
    PRIMARY KEY (medal_name_id, lang)
);
"""


def create_metadata_db(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée le schéma metadata.duckdb (idempotent)."""
    logger.info("Création du schéma metadata.duckdb...")
    conn.execute(METADATA_SCHEMA_DDL)
    logger.info("✓ Schéma créé")


# ---------------------------------------------------------------------------
# Helpers d'enrichissement i18n (privés)
# ---------------------------------------------------------------------------


def _enrich_game_variants(conn: duckdb.DuckDBPyConnection, errors: list[str]) -> None:
    """Calcule mode_name, name_fr et mode_name_fr pour game_variants."""
    conn.execute("""
        UPDATE game_variants
        SET mode_name = CASE
            WHEN CONTAINS(public_name, ':')
            THEN TRIM(SPLIT_PART(public_name, ':', 2))
            ELSE NULL
        END
        WHERE mode_name IS NULL AND public_name IS NOT NULL
    """)
    unresolved = conn.execute("""
        SELECT asset_id, public_name FROM game_variants
        WHERE mode_name IS NULL AND public_name IS NOT NULL
    """).fetchall()
    for asset_id, public_name in unresolved:
        errors.append(
            f'game_variants | asset_id={asset_id} | public_name="{public_name}" '
            "| raison=mode_name_extraction_failed (pas de ':' dans public_name)"
        )
    conn.execute("""
        UPDATE game_variants g
        SET name_fr = (
            SELECT mt.name_fr FROM mode_translations mt
            WHERE STARTS_WITH(mt.name_en, g.public_name || ' on ')
            LIMIT 1
        )
        WHERE g.name_fr IS NULL AND g.public_name IS NOT NULL
    """)
    conn.execute("""
        UPDATE game_variants
        SET mode_name_fr = CASE
            WHEN name_fr IS NOT NULL AND CONTAINS(name_fr, ' : ')
            THEN TRIM(SPLIT_PART(name_fr, ' : ', 2))
            WHEN name_fr IS NOT NULL THEN name_fr
            ELSE NULL
        END
        WHERE mode_name_fr IS NULL AND name_fr IS NOT NULL
    """)


def _enrich_pairs(conn: duckdb.DuckDBPyConnection) -> None:
    """Calcule name_fr pour playlist_map_mode_pairs depuis mode_translations."""
    conn.execute("""
        UPDATE playlist_map_mode_pairs p
        SET name_fr = (
            SELECT mt.name_fr FROM mode_translations mt
            WHERE TRIM(LOWER(mt.name_en)) = TRIM(LOWER(p.public_name))
            LIMIT 1
        )
        WHERE p.name_fr IS NULL AND p.public_name IS NOT NULL
    """)


def _enrich_playlists(conn: duckdb.DuckDBPyConnection, errors: list[str]) -> None:
    """Calcule name_fr, playlist_canonical_en/fr pour playlists."""
    conn.execute("""
        UPDATE playlists p
        SET name_fr = pt.name_fr
        FROM playlist_translations pt
        WHERE p.asset_id = pt.uuid AND p.name_fr IS NULL
    """)
    unresolved = conn.execute("""
        SELECT asset_id, public_name FROM playlists
        WHERE name_fr IS NULL AND public_name IS NOT NULL
    """).fetchall()
    for asset_id, public_name in unresolved:
        errors.append(
            f'playlists | asset_id={asset_id} | public_name="{public_name}" '
            "| raison=no_translation_found (UUID absent de playlist_translations)"
        )
    conn.execute("""
        UPDATE playlists
        SET playlist_canonical_en = TRIM(SPLIT_PART(public_name, ':', 1))
        WHERE playlist_canonical_en IS NULL AND public_name IS NOT NULL
    """)
    conn.execute("""
        UPDATE playlists
        SET playlist_canonical_fr = name_fr
        WHERE playlist_canonical_fr IS NULL AND name_fr IS NOT NULL
    """)


def _write_enrich_errors(errors: list[str]) -> None:
    """Écrit les erreurs d'enrichissement dans ERRORS_FILE_PATH."""
    if not errors:
        logger.info("✓ Enrichissement i18n complet, aucune erreur")
        return
    ts = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    with ERRORS_FILE_PATH.open("a", encoding="utf-8") as f:
        for err in errors:
            f.write(f"[{ts}] {err}\n")
    logger.warning(
        "populate_metadata: %d erreur(s) écrite(s) dans %s",
        len(errors),
        ERRORS_FILE_PATH,
    )


# ---------------------------------------------------------------------------
# API publique
# ---------------------------------------------------------------------------


def enrich_i18n(conn: duckdb.DuckDBPyConnection) -> None:
    """Enrichit les colonnes i18n depuis les tables de traduction existantes.

    Sources :
    - playlists.name_fr : depuis playlist_translations (join par UUID)
    - game_variants.name_fr + mode_name + mode_name_fr : depuis mode_translations
    - playlist_map_mode_pairs.name_fr : depuis mode_translations (correspondance exacte)
    - playlist_canonical_en/fr : calculés depuis public_name et name_fr

    Erreurs non résolues : écrites dans metadata_populate_errors.txt.
    """
    errors: list[str] = []
    _enrich_game_variants(conn, errors)
    _enrich_pairs(conn)
    _enrich_playlists(conn, errors)
    n_modes = conn.execute(
        "SELECT COUNT(DISTINCT mode_name) FROM game_variants WHERE mode_name IS NOT NULL"
    ).fetchone()[0]
    logger.info("mode_name distincts dans game_variants : %d (attendu ~27)", n_modes)
    _write_enrich_errors(errors)
