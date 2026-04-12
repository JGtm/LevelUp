"""Migrations de schéma DuckDB — DB metadata et PvE.

Ce module regroupe les fonctions de migration relatives à metadata.duckdb
(weapon_labels, medal_definitions, asset_translations, medal_translations)
et shared_pve.duckdb (ensure_pve_schema).

Extrait de migrations.py lors du housekeeping pré-v7 (H3).
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    import duckdb

logger = logging.getLogger(__name__)

PVE_SCHEMA_DDL = """
CREATE TABLE IF NOT EXISTS pve_match_stats (
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,

    -- Stats globales PvE (validées depuis interface PveStats API)
    total_enemy_kills  INTEGER,          -- API: Kills
    boss_kills         INTEGER,          -- API: BossKills

    -- Kills par type d'ennemi (API confirmée)
    grunt_kills    INTEGER DEFAULT 0,    -- API: GruntKills
    elite_kills    INTEGER DEFAULT 0,    -- API: EliteKills
    jackal_kills   INTEGER DEFAULT 0,    -- API: JackalKills
    brute_kills    INTEGER DEFAULT 0,    -- API: BruteKills
    hunter_kills   INTEGER DEFAULT 0,    -- API: HunterKills
    skimmer_kills  INTEGER DEFAULT 0,    -- API: SkimmerKills
    sentinel_kills INTEGER DEFAULT 0,    -- API: SentinelKills
    marine_kills   INTEGER DEFAULT 0,    -- API: MarineKills

    -- Bitmask granulaire (v5.2) — quels champs ont été récupérés
    -- Voir PveBits dans src/data/sync/constants.py
    pve_bits       INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (match_id, xuid)
);

CREATE INDEX IF NOT EXISTS idx_pve_xuid     ON pve_match_stats(xuid);
CREATE INDEX IF NOT EXISTS idx_pve_match_id ON pve_match_stats(match_id);
"""


def ensure_pve_schema(conn: duckdb.DuckDBPyConnection) -> None:
    """Initialise le schéma PvE dans shared_pve.duckdb (idempotent).

    Crée la table ``pve_match_stats`` et ses index si absents.
    Applique aussi les migrations de colonnes manquantes si la table existait
    avec une ancienne version du schéma.
    À appeler au démarrage de toute connexion vers shared_pve.duckdb.

    Args:
        conn: Connexion DuckDB vers shared_pve.duckdb.
    """
    try:
        # DDL complet (idempotent via IF NOT EXISTS)
        for stmt in PVE_SCHEMA_DDL.strip().split(";"):
            stmt = stmt.strip()
            if stmt:
                try:
                    conn.execute(stmt)
                except Exception as e:
                    err = str(e).lower()
                    if "already exists" not in err:
                        logger.warning("Erreur DDL PvE : %s", e)

        # Migration v5.2 : ajout des colonnes manquantes si ancienne version du schéma
        _pve_migrations = [
            ("sentinel_kills", "INTEGER DEFAULT 0"),
            ("marine_kills", "INTEGER DEFAULT 0"),
        ]
        existing_cols = {
            row[0]
            for row in conn.execute(
                "SELECT column_name FROM information_schema.columns"
                " WHERE table_name = 'pve_match_stats'"
            ).fetchall()
        }
        for col_name, col_def in _pve_migrations:
            if col_name not in existing_cols:
                try:
                    conn.execute(f"ALTER TABLE pve_match_stats ADD COLUMN {col_name} {col_def}")
                    logger.info("Migration PvE : colonne '%s' ajoutée à pve_match_stats", col_name)
                except Exception as e:
                    logger.warning("Migration PvE '%s': %s", col_name, e)

        logger.debug("Schéma PvE initialisé (shared_pve.duckdb)")
    except Exception as e:
        logger.error("Impossible d'initialiser le schéma PvE : %s", e)


# ─────────────────────────────────────────────────────────────────────────────
# Schéma LUSR / CSR — match_skill_rank (v5.2, stats.duckdb par joueur)
# ─────────────────────────────────────────────────────────────────────────────


_WEAPON_LABELS_DDL = """
CREATE TABLE IF NOT EXISTS weapon_labels (
    weapon_id UBIGINT PRIMARY KEY,
    name_en   VARCHAR NOT NULL,
    name_fr   VARCHAR NOT NULL
)
"""


def ensure_weapon_labels(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée et peuple ``weapon_labels`` dans metadata.duckdb (idempotente).

    Stocke les labels EN/FR pour chaque weapon_id filmshell (UBIGINT) et les
    3 IDs sentinelles (0=Grenade, 1=Melee, 2=Vehicle).
    Les lignes existantes ne sont jamais écrasées (INSERT OR IGNORE).
    """
    from src.analysis._weapon_data import (
        WEAPON_FUSION_MAP,
        WEAPON_INT_TO_NAME,
        WEAPON_NAME_FR,
    )

    conn.execute(_WEAPON_LABELS_DDL)

    rows: list[tuple[int, str, str]] = [
        # Sentinelles
        (0, "Grenade", "Grenade"),
        (1, "Melee", "Corps à corps"),
        (2, "Vehicle", "Véhicule"),
    ]

    for wid, name_en in WEAPON_INT_TO_NAME.items():
        canonical = WEAPON_FUSION_MAP.get(name_en, name_en)
        name_fr = WEAPON_NAME_FR.get(canonical, canonical)
        rows.append((wid, name_en, name_fr))

    conn.executemany(
        "INSERT OR IGNORE INTO weapon_labels (weapon_id, name_en, name_fr) VALUES (?, ?, ?)",
        rows,
    )
    logger.debug("weapon_labels : %d lignes insérées/ignorées", len(rows))


# ─────────────────────────────────────────────────────────────────────────────
# metadata.duckdb — Référentiel medal_definitions
# ─────────────────────────────────────────────────────────────────────────────

_MEDAL_DEFINITIONS_DDL = """
CREATE TABLE IF NOT EXISTS medal_definitions (
    medal_name_id  BIGINT PRIMARY KEY,
    name_fr        VARCHAR NOT NULL,
    name_en        VARCHAR NOT NULL,
    description_fr VARCHAR DEFAULT '',
    description_en VARCHAR DEFAULT '',
    is_custom      BOOLEAN DEFAULT FALSE
)
"""

_CUSTOM_MEDAL_THRESHOLD = 9_000_000_000


def ensure_medal_definitions_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée la table ``medal_definitions`` dans metadata.duckdb (idempotente).

    Seul le schéma est créé ici. La population depuis les JSON est assurée
    par ``scripts/populate_medal_metadata.py``.
    """
    conn.execute(_MEDAL_DEFINITIONS_DDL)


# ─────────────────────────────────────────────────────────────────────────────
# metadata.duckdb — Référentiels multi-langue (v6.3)
# ─────────────────────────────────────────────────────────────────────────────

_ASSET_TRANSLATIONS_DDL = """
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
"""

_MEDAL_TRANSLATIONS_DDL = """
CREATE TABLE IF NOT EXISTS medal_translations (
    medal_name_id BIGINT  NOT NULL,
    lang          VARCHAR NOT NULL,
    name          VARCHAR NOT NULL,
    description   VARCHAR,
    PRIMARY KEY (medal_name_id, lang)
);
"""

_CHALLENGE_DEFINITIONS_DDL = """
CREATE TABLE IF NOT EXISTS challenge_definitions (
    challenge_path         VARCHAR NOT NULL,
    content_hash           VARCHAR NOT NULL,
    category               VARCHAR,
    difficulty             VARCHAR,
    threshold_for_success  INTEGER,
    reward_xp              INTEGER DEFAULT 0,
    secondary_reward_xp    INTEGER DEFAULT 0,
    first_seen_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_current             BOOLEAN DEFAULT TRUE,
    PRIMARY KEY (challenge_path, content_hash)
);
CREATE INDEX IF NOT EXISTS idx_challenge_definitions_current
    ON challenge_definitions(challenge_path, is_current);
CREATE INDEX IF NOT EXISTS idx_challenge_definitions_category
    ON challenge_definitions(category, difficulty);
"""

_CHALLENGE_TRANSLATIONS_DDL = """
CREATE TABLE IF NOT EXISTS challenge_translations (
    challenge_path  VARCHAR NOT NULL,
    content_hash    VARCHAR NOT NULL,
    lang            VARCHAR NOT NULL,
    title           VARCHAR,
    description     VARCHAR,
    first_seen_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (challenge_path, content_hash, lang)
);
CREATE INDEX IF NOT EXISTS idx_challenge_translations_lookup
    ON challenge_translations(challenge_path, lang);
"""

_BATTLEPASS_TRACK_DEFINITIONS_DDL = """
CREATE TABLE IF NOT EXISTS battlepass_track_definitions (
    reward_track_path      VARCHAR NOT NULL,
    content_hash           VARCHAR NOT NULL,
    xp_per_rank            INTEGER,
    battlepass_image_path  VARCHAR,
    background_image_path  VARCHAR,
    raw_payload_json       VARCHAR NOT NULL,
    first_seen_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_current             BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (reward_track_path, content_hash)
);
CREATE INDEX IF NOT EXISTS idx_battlepass_track_definitions_lookup
    ON battlepass_track_definitions(reward_track_path, is_current);
"""

_BATTLEPASS_TRACK_TRANSLATIONS_DDL = """
CREATE TABLE IF NOT EXISTS battlepass_track_translations (
    reward_track_path  VARCHAR NOT NULL,
    content_hash       VARCHAR NOT NULL,
    lang               VARCHAR NOT NULL,
    track_name         VARCHAR,
    first_seen_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (reward_track_path, content_hash, lang)
);
CREATE INDEX IF NOT EXISTS idx_battlepass_track_translations_lookup
    ON battlepass_track_translations(reward_track_path, lang);
"""

_BATTLEPASS_ITEM_DEFINITIONS_DDL = """
CREATE TABLE IF NOT EXISTS battlepass_item_definitions (
    inventory_item_path  VARCHAR NOT NULL,
    content_hash         VARCHAR NOT NULL,
    quality              VARCHAR,
    item_type            VARCHAR,
    display_path         VARCHAR,
    raw_payload_json     VARCHAR NOT NULL,
    first_seen_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_current           BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (inventory_item_path, content_hash)
);
CREATE INDEX IF NOT EXISTS idx_battlepass_item_definitions_lookup
    ON battlepass_item_definitions(inventory_item_path, is_current);
"""

_BATTLEPASS_ITEM_TRANSLATIONS_DDL = """
CREATE TABLE IF NOT EXISTS battlepass_item_translations (
    inventory_item_path  VARCHAR NOT NULL,
    content_hash         VARCHAR NOT NULL,
    lang                 VARCHAR NOT NULL,
    title                VARCHAR,
    description          VARCHAR,
    first_seen_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (inventory_item_path, content_hash, lang)
);
CREATE INDEX IF NOT EXISTS idx_battlepass_item_translations_lookup
    ON battlepass_item_translations(inventory_item_path, lang);
"""

_BATTLEPASS_ASSET_REFS_DDL = """
CREATE TABLE IF NOT EXISTS battlepass_asset_refs (
    asset_key         VARCHAR PRIMARY KEY,
    asset_kind        VARCHAR NOT NULL,
    source_path       VARCHAR NOT NULL,
    cache_rel_path    VARCHAR NOT NULL,
    mime_type         VARCHAR NOT NULL DEFAULT 'image/png',
    image_source_path VARCHAR,
    source_origin     VARCHAR NOT NULL DEFAULT 'cms',
    first_cached_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_cached_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_battlepass_asset_refs_kind_source
    ON battlepass_asset_refs(asset_kind, source_path);
CREATE INDEX IF NOT EXISTS idx_battlepass_asset_refs_accessed
    ON battlepass_asset_refs(last_accessed_at);
"""


def ensure_asset_translations_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``asset_translations`` dans metadata.duckdb (idempotente).

    Table pivot multi-langue pour les assets Discovery UGC :
    maps, playlists, playlist_map_mode_pairs, game_variants.
    Peuplée par ``scripts/populate_asset_translations.py``.
    """
    conn.execute(_ASSET_TRANSLATIONS_DDL)


def ensure_medal_translations_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``medal_translations`` dans metadata.duckdb (idempotente).

    Table pivot multi-langue pour les médailles Halo Infinite.
    Peuplée par ``scripts/populate_medal_metadata.py`` (champ ``translations``
    de l'API + migration depuis medal_definitions pour les médailles custom).
    """
    conn.execute(_MEDAL_TRANSLATIONS_DDL)


def ensure_challenge_definitions_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``challenge_definitions`` dans metadata.duckdb (idempotente).

    Référentiel versionné des définitions CMS de défis Halo Infinite.
    Chaque ligne représente un contenu unique identifié par ``content_hash``.
    """
    conn.execute(_CHALLENGE_DEFINITIONS_DDL)


def ensure_challenge_translations_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``challenge_translations`` dans metadata.duckdb (idempotente).

    Table pivot multi-langue pour les titres et descriptions de défis.
    Les langues sont stockées au format BCP-47 quand elles sont connues.
    """
    conn.execute(_CHALLENGE_TRANSLATIONS_DDL)


def ensure_battlepass_track_definitions_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``battlepass_track_definitions`` dans metadata.duckdb (idempotente).

    Référentiel versionné des reward tracks GameCMS du battle pass.
    Le payload brut est conservé pour reconstruire la structure des ranks.
    """
    conn.execute(_BATTLEPASS_TRACK_DEFINITIONS_DDL)


def ensure_battlepass_track_translations_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``battlepass_track_translations`` dans metadata.duckdb (idempotente).

    Table pivot multi-langue pour les noms des reward tracks battle pass.
    """
    conn.execute(_BATTLEPASS_TRACK_TRANSLATIONS_DDL)


def ensure_battlepass_item_definitions_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``battlepass_item_definitions`` dans metadata.duckdb (idempotente).

    Référentiel versionné des items inventaire utilisés par les rewards battle pass.
    """
    conn.execute(_BATTLEPASS_ITEM_DEFINITIONS_DDL)


def ensure_battlepass_item_translations_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``battlepass_item_translations`` dans metadata.duckdb (idempotente).

    Table pivot multi-langue pour les titres et descriptions des items battle pass.
    """
    conn.execute(_BATTLEPASS_ITEM_TRANSLATIONS_DDL)


def ensure_battlepass_asset_refs_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``battlepass_asset_refs`` dans metadata.duckdb (idempotente).

    Référence les visuels de pass de combat stockés sur disque local
    (tracks, rewards, placeholders currency) sans stocker les blobs en DB.
    """
    conn.execute(_BATTLEPASS_ASSET_REFS_DDL)


# ─────────────────────────────────────────────────────────────────────────────
# Migration match_registry : playable_duration_seconds + real_start_time (v6.3)
# ─────────────────────────────────────────────────────────────────────────────
