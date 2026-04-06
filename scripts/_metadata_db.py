"""DDL et schéma pour metadata.duckdb.

Contient :
- METADATA_SCHEMA_DDL : schéma des tables (playlists, maps, pairs, game_variants,
  asset_translations, medal_translations)
- create_metadata_db : création idempotente du schéma
- enrich_i18n : no-op depuis v6.3 (remplacé par populate_asset_translations.py)
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    import duckdb

logger = logging.getLogger(__name__)

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


# ---------------------------------------------------------------------------
# API publique
# ---------------------------------------------------------------------------


def enrich_i18n(conn: duckdb.DuckDBPyConnection) -> None:  # noqa: ARG001
    """(Obsolète) Enrichissement i18n via tables mode_translations / playlist_translations.

    Ces tables n'existent plus depuis v6.3 (remplacées par asset_translations).
    Les traductions sont désormais peuplées par :
    - scripts/populate_asset_translations.py (maps, playlists, pairs, game_variants)
    - scripts/populate_medal_metadata.py (médailles)
    """
    logger.info("enrich_i18n() : no-op depuis v6.3 — utiliser populate_asset_translations.py")
