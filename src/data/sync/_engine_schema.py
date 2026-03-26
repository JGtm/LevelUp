"""Schéma DDL et mixin de gestion du schéma pour le DuckDBSyncEngine.

Contient les définitions de tables DuckDB (player DB) et le mixin
qui applique le schéma lors de la première connexion.
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)


# =============================================================================
# Schéma DuckDB pour les tables joueur
# =============================================================================

SYNC_SCHEMA_DDL = """
-- Table personal_score_awards (Sprint 8 - Décomposition du score personnel)
-- Stocke les awards individuels pour analyse de la contribution aux objectifs
CREATE SEQUENCE IF NOT EXISTS personal_score_awards_id_seq;
CREATE TABLE IF NOT EXISTS personal_score_awards (
    id INTEGER PRIMARY KEY DEFAULT nextval('personal_score_awards_id_seq'),
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,
    award_name VARCHAR NOT NULL,
    award_category VARCHAR,
    award_count INTEGER DEFAULT 1,
    award_score INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_psa_match ON personal_score_awards(match_id);
CREATE INDEX IF NOT EXISTS idx_psa_xuid ON personal_score_awards(xuid);
CREATE INDEX IF NOT EXISTS idx_psa_category ON personal_score_awards(award_category);

-- Table player_match_enrichment (V5 finale - Enrichissements personnels uniquement)
-- Données spécifiques au POV du joueur, ne vont PAS dans shared
CREATE TABLE IF NOT EXISTS player_match_enrichment (
    match_id VARCHAR PRIMARY KEY,
    performance_score FLOAT,
    session_id VARCHAR,
    session_label VARCHAR,
    is_with_friends BOOLEAN,
    teammates_signature VARCHAR,
    known_teammates_count SMALLINT,
    friends_xuids VARCHAR,
    had_bot_teammate BOOLEAN,  -- coéquipier bot IA détecté (v5.5)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_pme_session ON player_match_enrichment(session_id);

-- Table sync_meta (métadonnées de synchronisation)
CREATE TABLE IF NOT EXISTS sync_meta (
    key VARCHAR PRIMARY KEY,
    value VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table career_progression (Phase 5 - Rang carrière)
CREATE SEQUENCE IF NOT EXISTS career_progression_id_seq;
CREATE TABLE IF NOT EXISTS career_progression (
    id INTEGER PRIMARY KEY DEFAULT nextval('career_progression_id_seq'),
    xuid VARCHAR NOT NULL,
    rank INTEGER NOT NULL,
    rank_name VARCHAR,
    rank_tier VARCHAR,
    current_xp INTEGER,
    xp_for_next_rank INTEGER,
    xp_total INTEGER,
    is_max_rank BOOLEAN DEFAULT FALSE,
    adornment_path VARCHAR,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_career_xuid ON career_progression(xuid);
CREATE INDEX IF NOT EXISTS idx_career_date ON career_progression(recorded_at);

-- Tables media_files et media_match_associations : créées et migrées uniquement par
-- MediaIndexer.ensure_schema() (plan onglet Médias, refonte à partir de zéro).
-- Ne pas les créer ici pour éviter un schéma divergent.
"""


class SchemaMixin:
    """Mixin de gestion du schéma DuckDB pour DuckDBSyncEngine."""

    def _ensure_schema(self) -> None:
        """S'assure que les tables nécessaires existent."""
        conn = self._connection
        if conn is None:
            return

        # Nettoyage one-shot : supprimer les tables legacy migrées vers shared_matches.duckdb
        _LEGACY_PLAYER_TABLES = [
            "medals_earned",
            "player_match_stats",
            "highlight_events",
            "match_participants",
            "xuid_aliases",
            "backfill_status",
        ]
        for tbl in _LEGACY_PLAYER_TABLES:
            try:
                conn.execute(f"DROP TABLE IF EXISTS {tbl} CASCADE")
            except Exception as e:
                logger.debug("event=drop_legacy_table_failed table=%s error=%s", tbl, e)

        # Tables de sync (player-only)
        for stmt in SYNC_SCHEMA_DDL.split(";"):
            stmt = stmt.strip()
            if stmt:
                try:
                    conn.execute(stmt)
                except Exception as e:
                    if "already exists" not in str(e).lower():
                        logger.warning("Schema DDL warning: %s", e)

        # Migrations
        self._ensure_career_progression_sequence()
        self._ensure_career_progression_spartan_id()

        # Colonne had_bot_teammate sur player_match_enrichment (migration v5.5)
        from src.data.sync.migrations import ensure_bot_teammate_column

        ensure_bot_teammate_column(conn)

        # Index de performance sur tables locales (v5.1 Étape 2)
        from src.data.sync.migrations import ensure_player_performance_indexes

        ensure_player_performance_indexes(conn)

    def _ensure_match_stats_table(self) -> None:
        """V5 finale - Ne crée PLUS match_stats (table obsolète, données dans shared)."""
        conn = self._connection
        if conn is None:
            return
        logger.debug("V5 finale - match_stats non créée (obsolète)")

    def _ensure_career_progression_sequence(self) -> None:
        """S'assure que career_progression.id utilise une séquence auto-increment."""
        conn = self._connection
        if conn is None:
            return
        from src.data.sync.migrations import ensure_career_progression_autoincrement

        ensure_career_progression_autoincrement(conn)

    def _ensure_career_progression_spartan_id(self) -> None:
        """Ajoute la colonne spartan_id à career_progression si absente."""
        conn = self._connection
        if conn is None:
            return
        from src.data.sync.migrations import add_spartan_id_to_career_progression

        add_spartan_id_to_career_progression(conn)
