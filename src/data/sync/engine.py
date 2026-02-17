"""Moteur de synchronisation DuckDB unifié.

Ce module contient le DuckDBSyncEngine qui orchestre tout le pipeline :
API SPNKr → Transformation → DuckDB (direct, sans intermédiaire)

Usage:
    engine = DuckDBSyncEngine(
        player_db_path="data/players/Chocoboflor/stats.duckdb",
        xuid="123456789",
        gamertag="Chocoboflor",
    )

    # Sync incrémentale (rapide)
    result = await engine.sync_delta()
    print(result.to_message())

    # Sync complète (backfill)
    result = await engine.sync_full(SyncOptions(max_matches=500))
"""

from __future__ import annotations

import asyncio
import contextlib
import logging
import time
from collections.abc import Callable
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import duckdb

from src.data.sync.api_client import (
    SPNKrAPIClient,
    Tokens,
    enrich_match_info_with_assets,
    get_tokens_from_env,
)
from src.data.sync.batch_insert import (
    ALIAS_COLUMNS,
    HIGHLIGHT_EVENT_COLUMNS,
    PARTICIPANT_COLUMNS,
    PERSONAL_SCORE_COLUMNS,
    batch_insert_rows,
    batch_upsert_rows,
)
from src.data.sync.migrations import (
    BACKFILL_FLAGS,
    ensure_backfill_completed_column,
    ensure_highlight_events_autoincrement,
    ensure_player_performance_indexes,
)
from src.data.sync.models import (
    CareerRankData,
    MatchStatsRow,
    SyncOptions,
    SyncResult,
)
from src.data.sync.transformers import (
    create_metadata_resolver,
    extract_aliases,
    extract_all_medals,
    extract_match_registry_data,
    extract_participants,
    extract_personal_score_awards,
    extract_xuids_from_match,
    transform_highlight_events,
    transform_match_stats,
    transform_personal_score_awards,
    transform_skill_stats,
)

logger = logging.getLogger(__name__)

# Import pour le calcul des scores de performance
try:
    import polars as pl

    from src.analysis.performance_config import MIN_MATCHES_FOR_RELATIVE
    from src.analysis.performance_score import compute_relative_performance_score

    _PERF_SCORE_AVAILABLE = True
except ImportError:
    pl = None
    compute_relative_performance_score = None
    MIN_MATCHES_FOR_RELATIVE = 10
    _PERF_SCORE_AVAILABLE = False


# =============================================================================
# Schéma DuckDB pour les nouvelles tables
# =============================================================================

SYNC_SCHEMA_DDL = """
-- Table medals_earned (médailles obtenues par match)
CREATE TABLE IF NOT EXISTS medals_earned (
    match_id VARCHAR,
    medal_name_id BIGINT,
    count SMALLINT,
    PRIMARY KEY (match_id, medal_name_id)
);

-- Table player_match_stats (MMR/skill par match)
CREATE TABLE IF NOT EXISTS player_match_stats (
    match_id VARCHAR PRIMARY KEY,
    xuid VARCHAR NOT NULL,
    team_id TINYINT,
    team_mmr FLOAT,
    enemy_mmr FLOAT,
    kills_expected FLOAT,
    kills_stddev FLOAT,
    deaths_expected FLOAT,
    deaths_stddev FLOAT,
    assists_expected FLOAT,
    assists_stddev FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table highlight_events (kills, deaths depuis les films)
CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq;
CREATE TABLE IF NOT EXISTS highlight_events (
    id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
    match_id VARCHAR NOT NULL,
    event_type VARCHAR NOT NULL,
    time_ms INTEGER,
    xuid VARCHAR,
    gamertag VARCHAR,
    type_hint INTEGER,
    raw_json VARCHAR
);
CREATE INDEX IF NOT EXISTS idx_highlight_match ON highlight_events(match_id);

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

-- Table match_participants (Sprint Gamertag Roster Fix)
-- Stocke TOUS les joueurs de chaque match avec team_id/outcome/gamertag/score/rank/kda
-- Source : MatchStats.Players[] (API propre, pas les films corrompus)
-- rank = classement (1 = meilleur), API prioritaire sinon calculé. Score et k/d/a = API
CREATE TABLE IF NOT EXISTS match_participants (
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,
    team_id INTEGER,
    outcome INTEGER,
    gamertag VARCHAR,
    rank SMALLINT,
    score INTEGER,
    kills SMALLINT,
    deaths SMALLINT,
    assists SMALLINT,
    shots_fired INTEGER,
    shots_hit INTEGER,
    damage_dealt FLOAT,
    damage_taken FLOAT,
    avg_life_seconds FLOAT,
    headshot_kills SMALLINT,
    max_killing_spree SMALLINT,
    kda FLOAT,
    accuracy FLOAT,
    time_played_seconds INTEGER,
    grenade_kills SMALLINT,
    melee_kills SMALLINT,
    power_weapon_kills SMALLINT,
    PRIMARY KEY (match_id, xuid)
);
CREATE INDEX IF NOT EXISTS idx_participants_xuid ON match_participants(xuid);
CREATE INDEX IF NOT EXISTS idx_participants_team ON match_participants(match_id, team_id);

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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_pme_session ON player_match_enrichment(session_id);

-- Table xuid_aliases (correspondances XUID → Gamertag)
CREATE TABLE IF NOT EXISTS xuid_aliases (
    xuid VARCHAR PRIMARY KEY,
    gamertag VARCHAR NOT NULL,
    last_seen TIMESTAMP,
    source VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_aliases_gamertag ON xuid_aliases(gamertag);

-- Table sync_meta (métadonnées de synchronisation)
CREATE TABLE IF NOT EXISTS sync_meta (
    key VARCHAR PRIMARY KEY,
    value VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table career_progression (Phase 5 - Rang carrière)
CREATE TABLE IF NOT EXISTS career_progression (
    id INTEGER PRIMARY KEY,
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


# =============================================================================
# DuckDBSyncEngine
# =============================================================================


class DuckDBSyncEngine:
    """Moteur de synchronisation API → DuckDB unifié.

    Gère tout le pipeline en une seule étape :
    1. Fetch depuis l'API SPNKr
    2. Transformation via transformers.py
    3. Upsert direct dans DuckDB
    4. Mise à jour des agrégats

    Thread-safe via lock asyncio pour les écritures DB.
    """

    def __init__(
        self,
        player_db_path: Path | str,
        *,
        xuid: str,
        gamertag: str,
        metadata_db_path: Path | str | None = None,
        shared_db_path: Path | str | None = None,
        tokens: Tokens | None = None,
    ) -> None:
        """
        Args:
            player_db_path: Chemin vers stats.duckdb du joueur.
            xuid: XUID du joueur.
            gamertag: Gamertag pour l'identification API.
            metadata_db_path: Chemin vers metadata.duckdb (auto-détecté si None).
            shared_db_path: Chemin vers shared_matches.duckdb (auto-détecté si None).
            tokens: Tokens SPNKr pré-fournis (sinon récupérés depuis env).
        """
        self._player_db_path = Path(player_db_path)
        self._xuid = xuid
        self._gamertag = gamertag
        self._tokens = tokens

        # Auto-résolution du XUID si vide (défense en profondeur)
        if not self._xuid and self._player_db_path.exists():
            self._xuid = self._resolve_xuid_from_db()

        # Auto-détection du chemin metadata.duckdb
        if metadata_db_path is None:
            data_dir = self._player_db_path.parent.parent.parent
            self._metadata_db_path = data_dir / "warehouse" / "metadata.duckdb"
        else:
            self._metadata_db_path = Path(metadata_db_path)

        # Auto-détection du chemin shared_matches.duckdb (v5)
        if shared_db_path is None:
            data_dir = self._player_db_path.parent.parent.parent
            self._shared_db_path: Path | None = data_dir / "warehouse" / "shared_matches.duckdb"
        else:
            self._shared_db_path = Path(shared_db_path)

        self._connection: duckdb.DuckDBPyConnection | None = None
        self._shared_connection: duckdb.DuckDBPyConnection | None = None
        self._db_lock = asyncio.Lock()
        self._shared_db_lock = asyncio.Lock()
        self._existing_match_ids: set[str] | None = None

        # Créer le resolver pour les métadonnées
        self._metadata_resolver = create_metadata_resolver(self._metadata_db_path)

    def _get_connection(self) -> duckdb.DuckDBPyConnection:
        """Retourne une connexion DuckDB (lecture/écriture)."""
        if self._connection is None:
            # Créer le dossier parent si nécessaire
            self._player_db_path.parent.mkdir(parents=True, exist_ok=True)

            # Connexion en lecture/écriture
            self._connection = duckdb.connect(
                str(self._player_db_path),
                read_only=False,
            )

            # Configuration
            self._connection.execute("SET memory_limit = '512MB'")
            self._connection.execute("SET enable_object_cache = true")

            # S'assurer que le schéma existe
            self._ensure_schema()

        return self._connection

    def _get_shared_connection(self) -> duckdb.DuckDBPyConnection | None:
        """Retourne une connexion vers shared_matches.duckdb (R/W).

        Returns:
            Connexion DuckDB ou None si la base n'existe pas.
        """
        if self._shared_connection is not None:
            return self._shared_connection

        if self._shared_db_path is None or not self._shared_db_path.exists():
            logger.debug("shared_matches.duckdb absent, mode legacy v4")
            return None

        self._shared_connection = duckdb.connect(
            str(self._shared_db_path),
            read_only=False,
        )
        self._shared_connection.execute("SET enable_object_cache = true")

        # Appliquer les migrations de colonnes sur les tables shared (v5)
        try:
            from src.data.sync.migrations import ensure_match_participants_columns

            ensure_match_participants_columns(self._shared_connection)
        except Exception as e:
            logger.debug(f"Migration match_participants shared: {e}")

        # Index de performance sur tables shared (v5.1 Étape 2)
        try:
            from src.data.sync.migrations import ensure_performance_indexes

            ensure_performance_indexes(self._shared_connection)
        except Exception as e:
            logger.debug(f"Index performance shared: {e}")

        return self._shared_connection

    @property
    def shared_enabled(self) -> bool:
        """Indique si le mode shared_matches est activé."""
        return self._shared_db_path is not None and self._shared_db_path.exists()

    def _ensure_schema(self) -> None:
        """S'assure que les tables nécessaires existent."""
        conn = self._connection
        if conn is None:
            return

        # S'assurer que match_stats existe avec toutes les colonnes nécessaires
        self._ensure_match_stats_table()

        # Tables de sync (nouvelles)
        for stmt in SYNC_SCHEMA_DDL.split(";"):
            stmt = stmt.strip()
            if stmt:
                try:
                    conn.execute(stmt)
                except Exception as e:
                    # Index déjà existant ou autre erreur non fatale
                    if "already exists" not in str(e).lower():
                        logger.warning(f"Schema DDL warning: {e}")

        # Colonnes rank/score sur match_participants (migration)
        self._ensure_match_participants_rank_score()

        # S'assurer que la séquence pour highlight_events existe (migration)
        self._ensure_highlight_events_sequence()

        # Colonne bitmask backfill_completed (migration)
        ensure_backfill_completed_column(conn)

        # Index de performance sur tables locales (v5.1 Étape 2)
        ensure_player_performance_indexes(conn)

    def _ensure_match_stats_table(self) -> None:
        """V5 finale - Ne crée PLUS match_stats (table obsolète, données dans shared).

        La table match_stats locale est supprimée en V5 finale.
        Les données sont désormais dans :
        - shared.match_registry (métadonnées match)
        - shared.match_participants (stats multijoue urs)
        - player_match_enrichment (enrichissements personnels)

        Mode legacy : pour les DBs v4 existantes, cette table peut encore exister
        mais ne sera PLUS alimentée par le sync engine.
        """
        conn = self._connection
        if conn is None:
            return

        # V5 finale : NE PLUS créer match_stats
        # La table player_match_enrichment est créée via SYNC_SCHEMA_DDL
        logger.debug("V5 finale - match_stats non créée (obsolète)")

        # Pour les DBs legacy v4 : si match_stats existe déjà, la laisser en l'état
        # mais ne plus l'alimenter

    def _ensure_match_participants_rank_score(self) -> None:
        """Ajoute les colonnes rank, score et k/d/a à match_participants si absentes (migration)."""
        conn = self._connection
        if conn is None:
            return
        from src.data.sync.migrations import ensure_match_participants_columns

        ensure_match_participants_columns(conn)

    def _ensure_highlight_events_sequence(self) -> None:
        """S'assure que highlight_events.id utilise une séquence auto-increment."""
        conn = self._connection
        if conn is None:
            return
        ensure_highlight_events_autoincrement(conn)

    def _resolve_xuid_from_db(self) -> str:
        """Résout le XUID depuis la DB joueur (fallback si non fourni).

        Stratégie identique à cache_loaders._resolve_player_xuid :
        1. sync_meta (key='xuid')
        2. player_match_stats.xuid
        3. xuid_aliases via gamertag
        """
        try:
            conn = duckdb.connect(str(self._player_db_path), read_only=True)

            # 1. sync_meta
            try:
                r = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid'").fetchone()
                if r and r[0] and str(r[0]).strip():
                    xuid = str(r[0]).strip()
                    conn.close()
                    logger.info(f"XUID résolu depuis sync_meta: {xuid}")
                    return xuid
            except Exception:
                pass

            # 2. player_match_stats
            try:
                r = conn.execute(
                    "SELECT DISTINCT xuid FROM player_match_stats WHERE xuid IS NOT NULL LIMIT 1"
                ).fetchone()
                if r and r[0] and str(r[0]).strip():
                    xuid = str(r[0]).strip()
                    conn.close()
                    logger.info(f"XUID résolu depuis player_match_stats: {xuid}")
                    return xuid
            except Exception:
                pass

            # 3. xuid_aliases via shared_matches.duckdb (v5.1)
            if self._gamertag:
                try:
                    from src.utils.paths import get_shared_matches_path_from_player

                    shared_path = get_shared_matches_path_from_player(self._player_db_path)
                    if shared_path and shared_path.exists():
                        shared_conn = duckdb.connect(str(shared_path), read_only=True)
                        r = shared_conn.execute(
                            "SELECT xuid FROM xuid_aliases WHERE gamertag = ? LIMIT 1",
                            [self._gamertag],
                        ).fetchone()
                        shared_conn.close()
                        if r and r[0] and str(r[0]).strip():
                            xuid = str(r[0]).strip()
                            conn.close()
                            logger.info(f"XUID résolu depuis shared.xuid_aliases: {xuid}")
                            return xuid
                except Exception:
                    pass

            conn.close()
        except Exception as e:
            logger.debug(f"Impossible de résoudre le XUID depuis la DB: {e}")

        return ""

    def _load_existing_match_ids(self) -> set[str]:
        """Charge les IDs des matchs existants depuis shared (V5 finale).

        V5 finale : lecture UNIQUEMENT depuis shared.match_participants.
        La table locale match_stats n'est plus alimentée.

        Fallback legacy : si shared non disponible, tente player_match_stats.
        """
        if self._existing_match_ids is not None:
            return self._existing_match_ids

        ids: set[str] = set()

        # Priorité 1 : shared.match_participants (V5)
        shared_conn = self._get_shared_connection()
        if shared_conn is not None and self._xuid:
            try:
                result = shared_conn.execute(
                    "SELECT DISTINCT match_id FROM match_participants WHERE xuid = ?", [self._xuid]
                ).fetchall()
                ids = {str(r[0]) for r in result if r[0]}
                logger.debug(f"Chargé {len(ids)} match IDs depuis shared.match_participants")
                self._existing_match_ids = ids
                return ids
            except Exception as e:
                logger.debug(f"Impossible de lire shared.match_participants: {e}")

        # Fallback legacy : player_match_stats (V4)
        conn = self._get_connection()
        try:
            result = conn.execute(
                "SELECT DISTINCT match_id FROM player_match_stats WHERE xuid = ?", [self._xuid]
            ).fetchall()
            ids = {str(r[0]) for r in result if r[0]}
            logger.debug(f"Fallback V4 : {len(ids)} match IDs depuis player_match_stats")
        except Exception as e:
            logger.warning(f"Impossible de charger les match IDs : {e}")

        self._existing_match_ids = ids
        return ids

    def _update_sync_meta(self, key: str, value: str) -> None:
        """Met à jour une entrée dans sync_meta."""
        conn = self._get_connection()
        now = datetime.now(timezone.utc).isoformat()
        conn.execute(
            """INSERT OR REPLACE INTO sync_meta (key, value, updated_at)
               VALUES (?, ?, ?)""",
            (key, value, now),
        )

    def _get_sync_meta(self, key: str) -> str | None:
        """Récupère une valeur depuis sync_meta."""
        try:
            conn = self._get_connection()
            result = conn.execute("SELECT value FROM sync_meta WHERE key = ?", (key,)).fetchone()
            return result[0] if result else None
        except Exception:
            return None

    async def sync_delta(
        self,
        options: SyncOptions | None = None,
        *,
        progress_callback: Callable[[int, int], None] | None = None,
    ) -> SyncResult:
        """Synchronisation incrémentale (nouveaux matchs uniquement).

        S'arrête dès qu'un match déjà connu est rencontré.
        Optimal pour les synchronisations régulières (< 10s).

        Args:
            options: Options de sync (défauts si None).
            progress_callback: Callback (current, total) pour progression.

        Returns:
            SyncResult avec les détails.
        """
        return await self._sync_internal(
            options or SyncOptions(),
            delta_mode=True,
            progress_callback=progress_callback,
        )

    async def sync_full(
        self,
        options: SyncOptions | None = None,
        *,
        progress_callback: Callable[[int, int], None] | None = None,
    ) -> SyncResult:
        """Synchronisation complète (tous les matchs).

        Continue même si des matchs existent déjà (mise à jour).
        Utile pour backfill de données manquantes.

        Args:
            options: Options de sync (défauts si None).
            progress_callback: Callback (current, total) pour progression.

        Returns:
            SyncResult avec les détails.
        """
        return await self._sync_internal(
            options or SyncOptions(),
            delta_mode=False,
            progress_callback=progress_callback,
        )

    async def _sync_internal(
        self,
        options: SyncOptions,
        *,
        delta_mode: bool,
        progress_callback: Callable[[int, int], None] | None = None,
    ) -> SyncResult:
        """Implémentation interne de la synchronisation."""
        result = SyncResult()
        result.started_at = datetime.now(timezone.utc)
        start_time = time.time()

        try:
            # Récupérer les tokens si nécessaire
            if self._tokens is None:
                self._tokens = await get_tokens_from_env()

            # Charger les matchs existants
            existing_ids = self._load_existing_match_ids()
            logger.info(f"Matchs existants en DB: {len(existing_ids)}")

            if delta_mode and not existing_ids:
                logger.warning("Mode delta mais aucun match existant!")

            # Créer le client API
            async with SPNKrAPIClient(
                tokens=self._tokens,
                requests_per_second=options.requests_per_second,
            ) as client:
                result = await self._process_matches(
                    client,
                    options,
                    existing_ids,
                    delta_mode=delta_mode,
                    progress_callback=progress_callback,
                )

            # Rafraîchir les agrégats après sync
            if result.matches_inserted > 0:
                await self._refresh_aggregates_async()

            # Sprint 6 : Calcul batch des performance scores post-sync
            if (
                result.matches_inserted > 0
                and options.defer_performance_score
                and _PERF_SCORE_AVAILABLE
            ):
                perf_count = self.batch_compute_performance_scores()
                logger.info(f"Performance scores calculés en batch : {perf_count}")

            # Mettre à jour les métadonnées
            self._update_sync_meta("last_sync_at", datetime.now(timezone.utc).isoformat())
            self._update_sync_meta("last_sync_mode", "delta" if delta_mode else "full")
            self._update_sync_meta("last_sync_matches", str(result.matches_inserted))

            # Persister xuid + gamertag pour _resolve_player_xuid (stratégie 1)
            if self._xuid:
                self._update_sync_meta("xuid", self._xuid)
            if self._gamertag:
                self._update_sync_meta("gamertag", self._gamertag)

            # Commit final
            conn = self._get_connection()
            conn.commit()

        except Exception as e:
            result.errors.append(str(e))
            logger.error(f"Erreur sync: {e}")

        result.finished_at = datetime.now(timezone.utc)
        result.duration_seconds = time.time() - start_time

        return result

    async def _process_matches(
        self,
        client: SPNKrAPIClient,
        options: SyncOptions,
        existing_ids: set[str],
        *,
        delta_mode: bool,
        progress_callback: Callable[[int, int], None] | None = None,
    ) -> SyncResult:
        """Traite les matchs depuis l'API."""
        result = SyncResult()
        result.started_at = datetime.now(timezone.utc)

        start = 0
        remaining = options.max_matches
        semaphore = asyncio.Semaphore(options.parallel_matches)

        while remaining > 0:
            # Récupérer un batch d'historique
            batch_size = min(25, remaining)

            history = await client.get_match_history(
                self._gamertag,
                match_type=options.match_type,
                start=start,
                count=batch_size,
            )

            if not history:
                break

            # Traiter les matchs
            for item in history:
                if remaining <= 0:
                    break

                match_id = item.match_id

                # Vérifier si le match existe déjà
                if match_id in existing_ids:
                    if delta_mode:
                        logger.info(f"[DELTA] Match {match_id} déjà connu — arrêt")
                        return result
                    else:
                        result.matches_skipped += 1
                        remaining -= 1
                        start += 1
                        continue

                # Récupérer et traiter le match
                async with semaphore:
                    match_result = await self._process_single_match(
                        client,
                        match_id,
                        options,
                    )

                if match_result.get("inserted"):
                    result.matches_inserted += 1
                    result.highlight_events_inserted += match_result.get("events", 0)
                    result.skill_records_inserted += match_result.get("skill", 0)
                    result.aliases_updated += match_result.get("aliases", 0)
                    existing_ids.add(match_id)

                    # Sprint 6 : Commit intermédiaire tous les N matchs
                    if (
                        options.batch_commit_size > 0
                        and result.matches_inserted % options.batch_commit_size == 0
                    ):
                        conn = self._get_connection()
                        conn.commit()
                        logger.debug(f"Commit intermédiaire après {result.matches_inserted} matchs")

                if match_result.get("error"):
                    result.warnings.append(match_result["error"])

                remaining -= 1
                start += 1

                # Callback de progression
                if progress_callback:
                    progress_callback(
                        options.max_matches - remaining,
                        options.max_matches,
                    )

                # Log de progression
                if result.matches_inserted > 0 and result.matches_inserted % 10 == 0:
                    logger.info(f"Importé {result.matches_inserted} matchs...")

            # Fin du batch
            if len(history) < batch_size:
                break

        return result

    async def _process_single_match(
        self,
        client: SPNKrAPIClient,
        match_id: str,
        options: SyncOptions,
    ) -> dict[str, Any]:
        """Traite un match unique : fetch, transform, insert.

        Si shared_matches est activé, délègue à _process_known_match()
        ou _process_new_match() selon que le match existe déjà dans le
        registre partagé.
        """
        # ── Mode shared v5 ─────────────────────────────────────────
        shared_conn = self._get_shared_connection()
        if shared_conn is not None:
            try:
                registry = shared_conn.execute(
                    """SELECT
                        backfill_completed,
                        participants_loaded,
                        events_loaded,
                        medals_loaded,
                        player_count
                    FROM match_registry
                    WHERE match_id = ?""",
                    (match_id,),
                ).fetchone()
            except Exception:
                registry = None

            if registry is not None:
                logger.info(
                    f"Match {match_id} déjà connu dans shared " f"(player_count={registry[4]})"
                )
                return await self._process_known_match(
                    client,
                    match_id,
                    registry,
                    options,
                )
            else:
                logger.info(f"Nouveau match {match_id} → sync complète vers shared")
                return await self._process_new_match(
                    client,
                    match_id,
                    options,
                )

        # ── Mode legacy v4 non supporté en v5.1 (8bis.B1) ─────────────────
        # shared_matches.duckdb est requis — pas de fallback
        raise RuntimeError(
            f"Mode legacy v4 non supporté en v5.1 — shared_matches.duckdb requis. "
            f"Match {match_id} ne peut pas être traité sans shared DB. "
            f"Exécutez 'python scripts/migrate_to_v5.py' pour créer la DB partagée."
        )

    # 8bis.B1 : _process_single_match_legacy supprimé (v5.1)
    # En v5.1, shared_matches.duckdb est requis — pas de mode legacy.
    # Les données sont stockées dans :
    # - shared.match_registry (métadonnées match)
    # - shared.match_participants (stats multijoueurs)
    # - player_match_enrichment (enrichissements personnels)

    # =====================================================================
    # v5 Shared Matches — Process known / new match
    # =====================================================================

    async def _process_known_match(
        self,
        client: SPNKrAPIClient,
        match_id: str,
        registry: tuple,
        options: SyncOptions,
    ) -> dict[str, Any]:
        """Traite un match déjà présent dans shared_matches (sync allégée).

        Seules les données personnelles (match_stats, enrichment) sont
        insérées dans la DB joueur. Les données communes manquantes sont
        backfillées dans shared si nécessaire.

        Args:
            client: Client API SPNKr.
            match_id: ID du match.
            registry: Tuple (backfill_completed, participants_loaded,
                      events_loaded, medals_loaded, player_count).
            options: Options de sync.

        Returns:
            Dict résultat avec mode='known_match'.
        """
        result: dict[str, Any] = {
            "inserted": False,
            "mode": "known_match",
            "events": 0,
            "skill": 0,
            "aliases": 0,
            "api_calls_saved": 0,
            "error": None,
        }

        _bf_completed, participants_loaded, events_loaded, medals_loaded, _player_count = registry

        try:
            # 1. Télécharger les stats (obligatoire pour extraire les données perso)
            stats_json = await client.get_match_stats(match_id)
            if stats_json is None:
                result["error"] = f"Impossible de récupérer {match_id}"
                return result

            if options.with_assets:
                await enrich_match_info_with_assets(client, stats_json)

            # 2. Transformer en match_row pour la player DB (mode legacy)
            xuids = extract_xuids_from_match(stats_json)
            skill_json = None
            highlight_events: list = []

            # Skill toujours utile pour le joueur (MMR dans match_stats)
            if options.with_skill and xuids:
                skill_json = await client.get_skill_stats(match_id, xuids)

            # Events : seulement si absent du shared
            if options.with_highlight_events and not events_loaded:
                highlight_events = await client.get_highlight_events(match_id)
            elif events_loaded:
                result["api_calls_saved"] += 1

            match_row = transform_match_stats(
                stats_json,
                self._xuid,
                skill_json=skill_json,
                metadata_resolver=self._metadata_resolver,
            )
            if match_row is None:
                result["error"] = f"Transformation échouée pour {match_id}"
                return result

            _skill_row = None
            if skill_json:
                _skill_row = transform_skill_stats(skill_json, match_id, self._xuid)

            # Extraire les médailles perso (player DB)
            from src.data.sync.transformers import extract_medals

            _medal_rows = extract_medals(stats_json, self._xuid)

            # PersonalScores
            personal_scores = extract_personal_score_awards(stats_json, self._xuid)
            personal_score_rows = []
            if personal_scores:
                personal_score_rows = transform_personal_score_awards(
                    match_id,
                    self._xuid,
                    personal_scores,
                )

            alias_rows = []
            if options.with_aliases:
                alias_rows = extract_aliases(stats_json)

            _participant_rows = []
            if options.with_participants:
                _participant_rows = extract_participants(stats_json)

            # 3. V5 finale : Insérer minimalement dans la player DB (seulement enrichissements)
            async with self._db_lock:
                # ❌ SUPPRIMÉ : _insert_match_row (match_stats obsolète)
                # ❌ SUPPRIMÉ : _insert_skill_row (player_match_stats obsolète)
                # ❌ SUPPRIMÉ : _insert_medal_rows (medals_earned locale obsolète)
                # ❌ SUPPRIMÉ : _insert_participant_rows (match_participants locale obsolète)
                # ❌ SUPPRIMÉ : _insert_alias_rows (xuid_aliases locale obsolète)

                # ✅ CONSERVÉ : personal_score_awards (enrichissement personnel)
                if personal_score_rows:
                    self._insert_personal_score_rows(personal_score_rows)

                # ✅ NOUVEAU : player_match_enrichment (performance_score, sessions, etc.)
                self._insert_enrichment_row(match_id, match_row)
                self._compute_and_update_performance_score(match_id, match_row)

                # ❌ SUPPRIMÉ : UPDATE match_stats backfill_completed
                # Le bitmask est dans shared.match_registry

            # 4. Backfill sélectif dans shared si des données manquent
            backfill_needed: list[str] = []
            async with self._shared_db_lock:
                shared_conn = self._get_shared_connection()
                if shared_conn is None:
                    result["error"] = "shared_connection perdue"
                    return result

                if not participants_loaded:
                    participants = extract_participants(stats_json)
                    self._insert_shared_participants(shared_conn, participants)
                    shared_conn.execute(
                        "UPDATE match_registry SET participants_loaded = TRUE WHERE match_id = ?",
                        (match_id,),
                    )
                    backfill_needed.append("participants")

                if not events_loaded and highlight_events:
                    event_rows_shared = transform_highlight_events(highlight_events, match_id)
                    self._insert_shared_events(shared_conn, event_rows_shared)
                    shared_conn.execute(
                        "UPDATE match_registry SET events_loaded = TRUE WHERE match_id = ?",
                        (match_id,),
                    )
                    result["events"] = len(event_rows_shared)
                    backfill_needed.append("events")

                if not medals_loaded:
                    medals_all = extract_all_medals(stats_json)
                    self._insert_shared_medals(shared_conn, medals_all)
                    shared_conn.execute(
                        "UPDATE match_registry SET medals_loaded = TRUE WHERE match_id = ?",
                        (match_id,),
                    )
                    backfill_needed.append("medals")

                # Aliases vers shared
                if alias_rows:
                    self._insert_shared_aliases(shared_conn, alias_rows)

                # Incrémenter player_count
                shared_conn.execute(
                    "UPDATE match_registry "
                    "SET player_count = player_count + 1, "
                    "    last_updated_at = CURRENT_TIMESTAMP "
                    "WHERE match_id = ?",
                    (match_id,),
                )

                # Marquer le bitmask backfill_completed avec les types syncés
                bf_mask = self._compute_backfill_mask(options)
                shared_conn.execute(
                    "UPDATE match_registry "
                    "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                    "WHERE match_id = ?",
                    (bf_mask, match_id),
                )

            if backfill_needed:
                logger.info(f"Backfill shared pour {match_id}: {', '.join(backfill_needed)}")

            result["inserted"] = True

        except Exception as e:
            result["error"] = f"Erreur traitement known {match_id}: {e}"
            logger.warning(result["error"])

        return result

    async def _process_new_match(
        self,
        client: SPNKrAPIClient,
        match_id: str,
        options: SyncOptions,
    ) -> dict[str, Any]:
        """Traite un nouveau match (sync complète → shared + player DB).

        Toutes les données communes sont insérées dans shared_matches,
        les données personnelles dans la player DB.

        Args:
            client: Client API SPNKr.
            match_id: ID du match.
            options: Options de sync.

        Returns:
            Dict résultat avec mode='new_match'.
        """
        result: dict[str, Any] = {
            "inserted": False,
            "mode": "new_match",
            "events": 0,
            "skill": 0,
            "aliases": 0,
            "error": None,
        }

        try:
            # 1. Télécharger les stats
            stats_json = await client.get_match_stats(match_id)
            if stats_json is None:
                result["error"] = f"Impossible de récupérer {match_id}"
                return result

            if options.with_assets:
                await enrich_match_info_with_assets(client, stats_json)

            # 2. Télécharger events et skill
            xuids = extract_xuids_from_match(stats_json)
            skill_json = None
            highlight_events: list = []

            if options.with_skill and xuids:
                skill_json = await client.get_skill_stats(match_id, xuids)

            if options.with_highlight_events:
                highlight_events = await client.get_highlight_events(match_id)

            # 3. Extraire les données communes pour shared
            registry_data = extract_match_registry_data(
                stats_json,
                metadata_resolver=self._metadata_resolver,
            )
            if registry_data is None:
                result["error"] = f"Extraction registry échouée pour {match_id}"
                return result

            participants = extract_participants(stats_json)
            medals_all = extract_all_medals(stats_json)
            alias_rows = extract_aliases(stats_json) if options.with_aliases else []

            event_rows_shared = []
            if highlight_events:
                event_rows_shared = transform_highlight_events(highlight_events, match_id)

            # 4. Insérer dans shared_matches
            async with self._shared_db_lock:
                shared_conn = self._get_shared_connection()
                if shared_conn is None:
                    result["error"] = "shared_connection indisponible"
                    return result

                self._insert_shared_registry(shared_conn, registry_data)
                self._insert_shared_participants(shared_conn, participants)
                self._insert_shared_medals(shared_conn, medals_all)

                if event_rows_shared:
                    self._insert_shared_events(shared_conn, event_rows_shared)
                    result["events"] = len(event_rows_shared)

                if alias_rows:
                    self._insert_shared_aliases(shared_conn, alias_rows)
                    result["aliases"] = len(alias_rows)

                # Mettre à jour les flags du registre
                shared_conn.execute(
                    """UPDATE match_registry SET
                        participants_loaded = TRUE,
                        events_loaded = ?,
                        medals_loaded = TRUE,
                        first_sync_by = ?,
                        first_sync_at = CURRENT_TIMESTAMP,
                        player_count = 1
                    WHERE match_id = ?""",
                    (
                        len(event_rows_shared) > 0,
                        self._gamertag,
                        match_id,
                    ),
                )

                # Marquer le bitmask backfill_completed avec les types syncés
                bf_mask = self._compute_backfill_mask(options)
                shared_conn.execute(
                    "UPDATE match_registry "
                    "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                    "WHERE match_id = ?",
                    (bf_mask, match_id),
                )

            # 5. Insérer les données personnelles dans la player DB
            match_row = transform_match_stats(
                stats_json,
                self._xuid,
                skill_json=skill_json,
                metadata_resolver=self._metadata_resolver,
            )
            if match_row is None:
                result["error"] = f"Transformation match_stats échouée pour {match_id}"
                return result

            _skill_row = None
            if skill_json:
                _skill_row = transform_skill_stats(skill_json, match_id, self._xuid)

            from src.data.sync.transformers import extract_medals

            _medal_rows_personal = extract_medals(stats_json, self._xuid)

            personal_scores = extract_personal_score_awards(stats_json, self._xuid)
            personal_score_rows = []
            if personal_scores:
                personal_score_rows = transform_personal_score_awards(
                    match_id,
                    self._xuid,
                    personal_scores,
                )

            _participant_rows_player = []
            if options.with_participants:
                _participant_rows_player = participants  # Réutiliser l'extraction

            # V5 finale : écriture minimale dans player DB (seulement enrichissements personnels)
            async with self._db_lock:
                # ❌ SUPPRIMÉ : _insert_match_row (match_stats obsolète)
                # ❌ SUPPRIMÉ : _insert_skill_row (player_match_stats obsolète)
                # ❌ SUPPRIMÉ : _insert_medal_rows (medals_earned locale obsolète)
                # ❌ SUPPRIMÉ : _insert_participant_rows (match_participants locale obsolète)
                # ❌ SUPPRIMÉ : _insert_alias_rows (xuid_aliases locale obsolète)

                # ✅ CONSERVÉ : personal_score_awards (enrichissement personnel)
                if personal_score_rows:
                    self._insert_personal_score_rows(personal_score_rows)

                # ✅ NOUVEAU : player_match_enrichment (performance_score, sessions, etc.)
                self._insert_enrichment_row(match_id, match_row)
                self._compute_and_update_performance_score(match_id, match_row)

                # ❌ SUPPRIMÉ : UPDATE match_stats backfill_completed
                # Le bitmask est maintenant dans shared.match_registry
                # (déjà mis à jour dans le bloc shared ci-dessus)

            result["inserted"] = True

        except Exception as e:
            result["error"] = f"Erreur traitement new {match_id}: {e}"
            logger.warning(result["error"])

        return result

    def _compute_backfill_mask(self, options: SyncOptions) -> int:
        """Calcule le bitmask backfill_completed pour un match.

        Args:
            options: Options de sync courantes.

        Returns:
            Bitmask entier.
        """
        bf_mask = 0
        bf_mask |= BACKFILL_FLAGS["medals"]
        bf_mask |= BACKFILL_FLAGS["personal_scores"]
        bf_mask |= BACKFILL_FLAGS["performance_scores"]
        bf_mask |= BACKFILL_FLAGS["accuracy"]
        bf_mask |= BACKFILL_FLAGS["shots"]
        if options.with_skill:
            bf_mask |= BACKFILL_FLAGS["skill"]
            bf_mask |= BACKFILL_FLAGS["enemy_mmr"]
        if options.with_highlight_events:
            bf_mask |= BACKFILL_FLAGS["events"]
        if options.with_participants:
            bf_mask |= BACKFILL_FLAGS["participants"]
            bf_mask |= BACKFILL_FLAGS["participants_scores"]
            bf_mask |= BACKFILL_FLAGS["participants_kda"]
            bf_mask |= BACKFILL_FLAGS["participants_shots"]
            bf_mask |= BACKFILL_FLAGS["participants_damage"]
            bf_mask |= BACKFILL_FLAGS["participants_avg_life"]
        if options.with_aliases:
            bf_mask |= BACKFILL_FLAGS["aliases"]
        if options.with_assets:
            bf_mask |= BACKFILL_FLAGS["assets"]
        return bf_mask

    # =====================================================================
    # v5 Shared Matches — Insertions dans shared_matches.duckdb
    # =====================================================================

    def _insert_shared_registry(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        data: dict[str, Any],
    ) -> None:
        """Insère un match dans match_registry (shared).

        Args:
            shared_conn: Connexion vers shared_matches.duckdb.
            data: Dict retourné par extract_match_registry_data().
        """
        shared_conn.execute(
            """INSERT OR IGNORE INTO match_registry (
                match_id, start_time, end_time,
                playlist_id, playlist_name,
                map_id, map_name,
                pair_id, pair_name,
                game_variant_id, game_variant_name,
                mode_category, is_ranked, is_firefight,
                duration_seconds,
                team_0_score, team_1_score,
                created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)""",
            (
                data["match_id"],
                data["start_time"],
                data["end_time"],
                data["playlist_id"],
                data["playlist_name"],
                data["map_id"],
                data["map_name"],
                data["pair_id"],
                data["pair_name"],
                data["game_variant_id"],
                data["game_variant_name"],
                data["mode_category"],
                data["is_ranked"],
                data["is_firefight"],
                data["duration_seconds"],
                data["team_0_score"],
                data["team_1_score"],
            ),
        )

    def _insert_shared_participants(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        participants: list,
    ) -> None:
        """Insère les participants dans shared.match_participants.

        Args:
            shared_conn: Connexion vers shared_matches.duckdb.
            participants: Liste de MatchParticipantRow.
        """
        if not participants:
            return
        from src.data.sync.batch_insert import batch_upsert_rows

        batch_upsert_rows(shared_conn, "match_participants", participants, PARTICIPANT_COLUMNS)

    def _insert_shared_events(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        event_rows: list,
    ) -> None:
        """Insère les highlight events dans shared.highlight_events.

        Args:
            shared_conn: Connexion vers shared_matches.duckdb.
            event_rows: Liste de HighlightEventRow.
        """
        if not event_rows:
            return
        from src.data.sync.batch_insert import HIGHLIGHT_EVENT_COLUMNS, batch_insert_rows

        batch_insert_rows(shared_conn, "highlight_events", event_rows, HIGHLIGHT_EVENT_COLUMNS)

    def _insert_shared_medals(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        medals: list,
    ) -> None:
        """Insère les médailles de TOUS les joueurs dans shared.medals_earned.

        Args:
            shared_conn: Connexion vers shared_matches.duckdb.
            medals: Liste de SharedMedalEarnedRow.
        """
        if not medals:
            return
        columns = ["match_id", "xuid", "medal_name_id", "count"]

        batch_upsert_rows(shared_conn, "medals_earned", medals, columns)

    def _insert_shared_aliases(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        alias_rows: list,
    ) -> None:
        """Insère les aliases xuid→gamertag dans shared.xuid_aliases.

        Args:
            shared_conn: Connexion vers shared_matches.duckdb.
            alias_rows: Liste de XuidAliasRow.
        """
        if not alias_rows:
            return

        now = datetime.now(timezone.utc)
        alias_dicts = [
            {
                "xuid": row.xuid,
                "gamertag": row.gamertag,
                "last_seen": row.last_seen,
                "source": row.source,
                "updated_at": now,
            }
            for row in alias_rows
        ]
        batch_upsert_rows(shared_conn, "xuid_aliases", alias_dicts, ALIAS_COLUMNS)

    def _ensure_performance_score_column(self) -> None:
        """V5 finale - Plus nécessaire (performance_score dans player_match_enrichment).

        La colonne performance_score n'est plus dans match_stats (table obsolète)
        mais dans player_match_enrichment (créée via SYNC_SCHEMA_DDL).

        Cette fonction est conservée pour compatibilité mais ne fait plus rien.
        """
        # V5 finale : player_match_enrichment est créée via SYNC_SCHEMA_DDL
        # La colonne performance_score y est définie par défaut
        pass

    def _compute_and_update_performance_score(
        self, match_id: str, match_row: MatchStatsRow
    ) -> None:
        """Calcule et met à jour le score de performance pour un match.

        Note: Chaque DB DuckDB est spécifique à un joueur, donc pas besoin de filtrer par xuid.

        Args:
            match_id: ID du match
            match_row: Données du match inséré
        """
        if not _PERF_SCORE_AVAILABLE:
            logger.debug("Modules de calcul de performance non disponibles, skip")
            return

        try:
            conn = self._get_connection()

            # S'assurer que la colonne existe
            self._ensure_performance_score_column()

            # Vérifier si le score existe déjà dans player_match_enrichment
            existing = conn.execute(
                "SELECT performance_score FROM player_match_enrichment WHERE match_id = ?",
                (match_id,),
            ).fetchone()

            if existing and existing[0] is not None:
                # Score déjà calculé, skip
                logger.debug(f"Score de performance déjà présent pour {match_id}")
                return

            # Charger l'historique (tous les matchs AVANT celui-ci, triés par date)
            current_start_time = match_row.start_time
            if current_start_time is None:
                logger.debug(f"Pas de start_time pour {match_id}, skip calcul score")
                return

            # Convertir datetime en format compatible avec DuckDB
            if isinstance(current_start_time, datetime):
                current_start_time_str = current_start_time.isoformat()
            else:
                current_start_time_str = str(current_start_time)

            # V5 finale : lire depuis shared.match_participants + match_registry
            shared_conn = self._get_shared_connection()
            if shared_conn is None:
                logger.debug("shared_connection indisponible pour calcul score")
                return

            # v5 : lire depuis shared avec xuid du joueur
            history_df = shared_conn.execute(
                """
                SELECT
                    mr.match_id, mr.start_time,
                    mp.kills, mp.deaths, mp.assists, mp.kda, mp.accuracy,
                    mp.time_played_seconds, mp.avg_life_seconds,
                    mp.personal_score, mp.damage_dealt,
                    mp.rank, mp.team_mmr,
                    (SELECT p2.team_mmr FROM match_participants p2
                     WHERE p2.match_id = mr.match_id AND p2.team_id != mp.team_id LIMIT 1) AS enemy_mmr
                FROM match_registry mr
                JOIN match_participants mp ON mr.match_id = mp.match_id
                WHERE mp.xuid = ?
                  AND mr.match_id != ?
                  AND mr.start_time IS NOT NULL
                  AND mr.start_time < CAST(? AS TIMESTAMP)
                ORDER BY mr.start_time ASC
                """,
                (self._xuid, match_id, current_start_time_str),
            ).pl()

            if history_df.is_empty() or len(history_df) < MIN_MATCHES_FOR_RELATIVE:
                logger.debug(
                    f"Pas assez d'historique pour calculer le score ({len(history_df)} matchs)"
                )
                return

            # Convertir match_row en dict pour le calcul v4
            match_dict = {
                "kills": match_row.kills or 0,
                "deaths": match_row.deaths or 0,
                "assists": match_row.assists or 0,
                "kda": match_row.kda,
                "accuracy": match_row.accuracy,
                "time_played_seconds": match_row.time_played_seconds or 600.0,
                "personal_score": getattr(match_row, "personal_score", None),
                "damage_dealt": getattr(match_row, "damage_dealt", None),
                "rank": getattr(match_row, "rank", None),
                "team_mmr": getattr(match_row, "team_mmr", None),
                "enemy_mmr": getattr(match_row, "enemy_mmr", None),
            }

            # Calculer le score
            score = compute_relative_performance_score(match_dict, history_df)

            if score is not None:
                # V5 finale : écrire dans player_match_enrichment au lieu de match_stats
                now = datetime.now(timezone.utc)
                conn.execute(
                    """INSERT INTO player_match_enrichment (match_id, performance_score, updated_at)
                    VALUES (?, ?, ?)
                    ON CONFLICT (match_id) DO UPDATE SET
                        performance_score = EXCLUDED.performance_score,
                        updated_at = EXCLUDED.updated_at
                    """,
                    (match_id, score, now),
                )
                # Sprint 6 : pas de commit individuel ici, le commit est géré
                # par le batching dans _process_matches ou le commit final
                logger.debug(f"Score de performance calculé pour {match_id}: {score:.1f}")
            else:
                logger.debug(f"Impossible de calculer le score pour {match_id}")

        except Exception as e:
            # Ne pas bloquer la synchronisation si le calcul échoue
            logger.warning(f"Erreur calcul score performance pour {match_id}: {e}")

    def batch_compute_performance_scores(self) -> int:
        """Calcule les performance_score pour tous les matchs où il est NULL.

        Exécuté post-sync pour ne pas bloquer l'insertion des matchs.
        Utilise le calcul vectorisé de compute_relative_performance_score()
        avec un chargement unique de l'historique complet.

        Returns:
            Nombre de matchs mis à jour.
        """
        if not _PERF_SCORE_AVAILABLE:
            logger.debug("Modules de calcul de performance non disponibles, skip batch")
            return 0

        try:
            # V5 finale : lire depuis shared.match_participants + match_registry
            shared_conn = self._get_shared_connection()
            if shared_conn is None or not self._xuid:
                logger.warning("shared_connection ou xuid manquant pour batch performance scores")
                return 0

            conn = self._get_connection()
            self._ensure_performance_score_column()

            # 1. Charger TOUS les matchs triés par date depuis shared
            all_matches_df = shared_conn.execute(
                """
                SELECT
                    mr.match_id, mr.start_time,
                    mp.kills, mp.deaths, mp.assists, mp.kda, mp.accuracy,
                    mp.time_played_seconds, mp.avg_life_seconds,
                    mp.personal_score, mp.damage_dealt,
                    mp.rank, mp.team_mmr,
                    (SELECT p2.team_mmr FROM match_participants p2
                     WHERE p2.match_id = mr.match_id AND p2.team_id != mp.team_id LIMIT 1) AS enemy_mmr,
                    pme.performance_score
                FROM match_registry mr
                JOIN match_participants mp ON mr.match_id = mp.match_id
                LEFT JOIN player_match_enrichment pme ON mr.match_id = pme.match_id
                WHERE mp.xuid = ?
                  AND mr.start_time IS NOT NULL
                ORDER BY mr.start_time ASC
                """,
                [self._xuid],
            ).pl()

            if all_matches_df.is_empty():
                return 0

            # 2. Identifier les matchs sans score
            null_mask = all_matches_df["performance_score"].is_null()
            if not null_mask.any():
                logger.info("Tous les matchs ont déjà un performance_score")
                return 0

            # 3. Calculer le score pour chaque match NULL
            #    en utilisant l'historique des matchs précédents
            updates: list[tuple[str, float, str]] = []
            match_ids = all_matches_df["match_id"].to_list()
            now = datetime.now(timezone.utc)

            for i in range(len(all_matches_df)):
                if not null_mask[i]:
                    continue

                # Pas assez d'historique ?
                if i < MIN_MATCHES_FOR_RELATIVE:
                    continue

                # Historique = tous les matchs AVANT l'index i
                history_df = all_matches_df.slice(0, i).drop("performance_score")

                # Match courant en dict
                row = all_matches_df.row(i, named=True)
                match_dict = {
                    "kills": row.get("kills") or 0,
                    "deaths": row.get("deaths") or 0,
                    "assists": row.get("assists") or 0,
                    "kda": row.get("kda"),
                    "accuracy": row.get("accuracy"),
                    "time_played_seconds": row.get("time_played_seconds") or 600.0,
                    "personal_score": row.get("personal_score"),
                    "damage_dealt": row.get("damage_dealt"),
                    "rank": row.get("rank"),
                    "team_mmr": row.get("team_mmr"),
                    "enemy_mmr": row.get("enemy_mmr"),
                }

                score = compute_relative_performance_score(match_dict, history_df)
                if score is not None:
                    # V5 finale : UPDATE player_match_enrichment
                    updates.append((match_ids[i], score, str(now)))

            # 4. Batch UPSERT dans player_match_enrichment
            if updates:
                for match_id, score, updated_at in updates:
                    conn.execute(
                        """INSERT INTO player_match_enrichment (match_id, performance_score, updated_at)
                        VALUES (?, ?, ?)
                        ON CONFLICT (match_id) DO UPDATE SET
                            performance_score = EXCLUDED.performance_score,
                            updated_at = EXCLUDED.updated_at
                        """,
                        (match_id, score, updated_at),
                    )
                conn.commit()
                logger.info(f"Performance scores batch : {len(updates)} matchs mis à jour")

            return len(updates)

        except Exception as e:
            logger.warning(f"Erreur batch calcul performance scores : {e}")
            return 0

    # 8bis.B1 : _insert_match_row supprimé (v5.1)
    # Fonction obsolète — table match_stats locale supprimée.
    # Les données sont dans shared.match_registry + shared.match_participants.

    # 8bis.B1 : _insert_skill_row supprimé (v5.1)
    # Fonction obsolète — table player_match_stats locale supprimée.
    # Les données MMR/skill sont dans shared.match_participants.

    def _insert_event_rows(self, rows: list) -> None:
        """Insère des lignes highlight_events en batch (Sprint 15)."""
        if not rows:
            return
        conn = self._get_connection()
        batch_insert_rows(conn, "highlight_events", rows, HIGHLIGHT_EVENT_COLUMNS)

    # 8bis.B3 : _insert_alias_rows supprimée (v5.1 — xuid_aliases dans shared uniquement)

    def _insert_personal_score_rows(self, rows: list) -> None:
        """Insère des lignes personal_score_awards en batch (Sprint 15)."""
        if not rows:
            return
        conn = self._get_connection()
        now = datetime.now(timezone.utc)
        # Enrichir chaque row avec created_at
        score_dicts = []
        for row in rows:
            score_dicts.append(
                {
                    "match_id": row.match_id,
                    "xuid": row.xuid,
                    "award_name": row.award_name,
                    "award_category": row.award_category,
                    "award_count": row.award_count,
                    "award_score": row.award_score,
                    "created_at": now,
                }
            )
        batch_insert_rows(conn, "personal_score_awards", score_dicts, PERSONAL_SCORE_COLUMNS)

    # 8bis.B3 : _insert_medal_rows supprimée (v5.1 — medals_earned dans shared uniquement)
    # 8bis.B3 : _insert_participant_rows supprimée (v5.1 — match_participants dans shared uniquement)

    def _insert_enrichment_row(self, match_id: str, match_row: MatchStatsRow) -> None:
        """Insère/met à jour une ligne dans player_match_enrichment (V5 finale).

        Cette table contient les enrichissements personnels du joueur :
        - performance_score (calculé par _compute_and_update_performance_score)
        - session_id, session_label (calculé par backfill sessions)
        - is_with_friends, teammates_signature, known_teammates_count, friends_xuids

        Args:
            match_id: ID du match
            match_row: Données du match (pour extraire teammates_signature, etc.)
        """
        conn = self._get_connection()
        now = datetime.now(timezone.utc)

        # Extraire les champs pertinents de match_row
        teammates_sig = getattr(match_row, "teammates_signature", None)
        is_with_friends = getattr(match_row, "is_with_friends", None)
        known_teammates = getattr(match_row, "known_teammates_count", None)
        friends_xuids = getattr(match_row, "friends_xuids", None)

        conn.execute(
            """INSERT INTO player_match_enrichment
                (match_id, teammates_signature, is_with_friends,
                 known_teammates_count, friends_xuids, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT (match_id) DO UPDATE SET
                teammates_signature = COALESCE(EXCLUDED.teammates_signature, player_match_enrichment.teammates_signature),
                is_with_friends = COALESCE(EXCLUDED.is_with_friends, player_match_enrichment.is_with_friends),
                known_teammates_count = COALESCE(EXCLUDED.known_teammates_count, player_match_enrichment.known_teammates_count),
                friends_xuids = COALESCE(EXCLUDED.friends_xuids, player_match_enrichment.friends_xuids),
                updated_at = EXCLUDED.updated_at
            """,
            (match_id, teammates_sig, is_with_friends, known_teammates, friends_xuids, now, now),
        )

    async def _refresh_aggregates_async(self) -> None:
        """Rafraîchit les agrégats après sync (async wrapper)."""
        # Exécuter dans un thread pour ne pas bloquer l'event loop
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self.refresh_aggregates)

    def refresh_aggregates(self) -> dict[str, int]:
        """Recalcule les tables d'agrégats après sync.

        Met à jour :
        - Vues matérialisées (mv_*)
        - Tables pré-calculées (precomputed_sessions, precomputed_kda_trend, etc.)

        Returns:
            Dict table_name → rows_affected.
        """
        result: dict[str, int] = {}

        try:
            # Appeler refresh_materialized_views si disponible
            # (implémenté dans DuckDBRepository)
            try:
                from src.data.repositories.duckdb_repo import DuckDBRepository

                repo = DuckDBRepository(
                    self._player_db_path,
                    self._xuid,
                    read_only=False,
                )
                repo.refresh_materialized_views()
                result["materialized_views"] = 1
            except Exception as e:
                logger.debug(f"refresh_materialized_views non disponible: {e}")

            # Sprint 8ter.4 : Pré-calcul post-sync des agrégats UI
            try:
                from scripts.post_sync_compute import post_sync_compute

                precomp = post_sync_compute(str(self._player_db_path))
                result.update(precomp)
            except Exception as e:
                logger.debug(f"post_sync_compute non disponible: {e}")

        except Exception as e:
            logger.warning(f"Erreur refresh_aggregates: {e}")

        return result

    def get_sync_status(self) -> dict[str, Any]:
        """Retourne l'état de la dernière synchronisation.

        Returns:
            Dict avec last_sync_at, total_matches, etc.
        """
        try:
            # V5 finale : compter depuis shared.match_participants
            shared_conn = self._get_shared_connection()
            match_count = 0
            if shared_conn is not None and self._xuid:
                with contextlib.suppress(Exception):
                    match_count = shared_conn.execute(
                        "SELECT COUNT(DISTINCT match_id) FROM match_participants WHERE xuid = ?",
                        [self._xuid],
                    ).fetchone()[0]

            # Récupérer les métadonnées
            last_sync = self._get_sync_meta("last_sync_at")
            last_mode = self._get_sync_meta("last_sync_mode")
            last_matches = self._get_sync_meta("last_sync_matches")

            return {
                "total_matches": match_count,
                "last_sync_at": last_sync,
                "last_sync_mode": last_mode,
                "last_sync_matches": int(last_matches) if last_matches else 0,
                "gamertag": self._gamertag,
                "xuid": self._xuid,
            }
        except Exception as e:
            return {"error": str(e)}

    # =========================================================================
    # Méthodes Phase 5 : Career Rank
    # =========================================================================

    async def sync_career_rank(self) -> CareerRankData | None:
        """Synchronise la progression du rang carrière.

        Récupère les données depuis l'API et les sauvegarde en BDD.
        Crée un snapshot historique pour suivre la progression.

        Returns:
            CareerRankData ou None si erreur.
        """
        try:
            # Récupérer les tokens si nécessaire
            if self._tokens is None:
                self._tokens = await get_tokens_from_env()

            async with SPNKrAPIClient(tokens=self._tokens) as client:
                career_data = await client.get_career_rank_progression(self._xuid)

                if career_data is None:
                    logger.warning(f"Career rank non disponible pour {self._gamertag}")
                    return None

                # Sauvegarder en BDD
                self._save_career_rank(career_data)

                logger.info(
                    f"Career rank sync: {self._gamertag} → "
                    f"Rang {career_data.current_rank} ({career_data.current_rank_name})"
                )

                return career_data

        except Exception as e:
            logger.error(f"Erreur sync_career_rank: {e}")
            return None

    def _save_career_rank(self, data: CareerRankData) -> None:
        """Sauvegarde un snapshot de la progression de rang."""
        conn = self._get_connection()
        now = datetime.now(timezone.utc)

        conn.execute(
            """INSERT INTO career_progression (
                xuid, rank, rank_name, rank_tier,
                current_xp, xp_for_next_rank, xp_total,
                is_max_rank, adornment_path, recorded_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)""",
            (
                data.xuid,
                data.current_rank,
                data.current_rank_name,
                data.current_rank_tier,
                data.current_xp,
                data.xp_for_next_rank,
                data.xp_total,
                data.is_max_rank,
                data.adornment_path,
                now,
            ),
        )
        conn.commit()

        # Mettre à jour sync_meta
        self._update_sync_meta("last_career_sync_at", now.isoformat())
        self._update_sync_meta("current_rank", str(data.current_rank))

    def get_career_rank_history(self, limit: int = 50) -> list[dict[str, Any]]:
        """Récupère l'historique de progression de rang.

        Args:
            limit: Nombre maximum d'entrées.

        Returns:
            Liste des snapshots de progression.
        """
        try:
            conn = self._get_connection()
            result = conn.execute(
                """SELECT rank, rank_name, rank_tier, current_xp,
                          xp_for_next_rank, xp_total, is_max_rank,
                          adornment_path, recorded_at
                   FROM career_progression
                   WHERE xuid = ?
                   ORDER BY recorded_at DESC
                   LIMIT ?""",
                (self._xuid, limit),
            ).fetchall()

            return [
                {
                    "rank": r[0],
                    "rank_name": r[1],
                    "rank_tier": r[2],
                    "current_xp": r[3],
                    "xp_for_next_rank": r[4],
                    "xp_total": r[5],
                    "is_max_rank": r[6],
                    "adornment_path": r[7],
                    "recorded_at": r[8],
                }
                for r in result
            ]

        except Exception as e:
            logger.warning(f"Erreur get_career_rank_history: {e}")
            return []

    def get_latest_career_rank(self) -> dict[str, Any] | None:
        """Récupère le dernier rang carrière enregistré.

        Returns:
            Dict avec les infos du rang ou None.
        """
        history = self.get_career_rank_history(limit=1)
        return history[0] if history else None

    def close(self) -> None:
        """Ferme les connexions DuckDB (player + shared)."""
        if self._connection:
            with contextlib.suppress(Exception):
                self._connection.close()
            self._connection = None
        if self._shared_connection:
            with contextlib.suppress(Exception):
                self._shared_connection.close()
            self._shared_connection = None
