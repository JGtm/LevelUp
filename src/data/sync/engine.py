"""Moteur de synchronisation DuckDB unifié.

Ce module contient le DuckDBSyncEngine qui orchestre tout le pipeline :
API SPNKr → Transformation → DuckDB (direct, sans intermédiaire)

Architecture mixin :
    Le code métier est réparti dans 6 modules mixin :
    - _shared_writes.py  : insertions dans shared_matches.duckdb
    - _performance.py    : calcul des scores de performance
    - _skill_rating.py   : CSR (ranked) + LUSR (TrueSkill 2)
    - _career.py         : synchronisation du rang carrière
    - _aggregates.py     : rafraîchissement agrégats post-sync
    - _match_processing.py : traitement des matchs (fetch, transform, insert)

    engine.py conserve l'orchestrateur, les connexions, le schéma et les helpers.

Usage:
    engine = DuckDBSyncEngine(
        player_db_path="data/players/SpartanB/stats.duckdb",
        xuid="123456789",
        gamertag="SpartanB",
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
import gc
import logging
import time
from collections.abc import Callable
from datetime import datetime, timezone
from pathlib import Path

import duckdb

from src.data.sync._aggregates import AggregatesMixin
from src.data.sync._career import CareerMixin
from src.data.sync._match_processing import MatchProcessingMixin
from src.data.sync._performance import PerformanceMixin
from src.data.sync._shared_writes import SharedWritesMixin
from src.data.sync._skill_rating import SkillRatingMixin
from src.data.sync.api_client import (
    SPNKrAPIClient,
    Tokens,
    get_tokens_for_player,
    get_tokens_from_env,
)
from src.data.sync.batch_insert import (
    HIGHLIGHT_EVENT_COLUMNS,
    PERSONAL_SCORE_COLUMNS,
    batch_insert_rows,
)
from src.data.sync.migrations import (
    add_spartan_id_to_career_progression,
    ensure_career_progression_autoincrement,
    ensure_match_registry_spnkr_version,
    ensure_player_performance_indexes,
)
from src.data.sync.models import (
    MatchStatsRow,
    SyncOptions,
    SyncResult,
)
from src.data.sync.transformers import (
    create_metadata_resolver,
)
from src.utils.paths import get_pve_db_path_from_player

logger = logging.getLogger(__name__)

# Re-exports pour compatibilité (utilisé par les tests et scripts externes)
__all__ = ["DuckDBSyncEngine", "get_tokens_for_player", "SPNKrAPIClient"]

# NOTE: post_sync_compute est appelé dans _aggregates.py (AggregatesMixin._refresh_aggregates_async)

# =============================================================================
# Schéma DuckDB pour les nouvelles tables
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


# =============================================================================
# DuckDBSyncEngine — Orchestrateur principal
# =============================================================================


class DuckDBSyncEngine(
    SharedWritesMixin,
    PerformanceMixin,
    SkillRatingMixin,
    CareerMixin,
    AggregatesMixin,
    MatchProcessingMixin,
):
    """Moteur de synchronisation API → DuckDB unifié.

    Gère tout le pipeline en une seule étape :
    1. Fetch depuis l'API SPNKr
    2. Transformation via transformers.py
    3. Upsert direct dans DuckDB
    4. Mise à jour des agrégats

    Thread-safe via lock asyncio pour les écritures DB.

    Mixins :
        SharedWritesMixin    — insertions dans shared_matches.duckdb
        PerformanceMixin     — calcul des scores de performance
        SkillRatingMixin     — CSR + LUSR (TrueSkill 2)
        CareerMixin          — rang carrière
        AggregatesMixin      — rafraîchissement agrégats
        MatchProcessingMixin — traitement des matchs
    """

    def __init__(  # noqa: PLR0913
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

        # Base PVE séparée (shared_pve.duckdb) — v5.2
        self._pve_db_path: Path = get_pve_db_path_from_player(self._player_db_path)
        self._pve_connection: duckdb.DuckDBPyConnection | None = None
        self._pve_db_lock = asyncio.Lock()

        # Cache des XUIDs amis (chargé lazily depuis friends_defaults.json)
        self._friends_xuids: frozenset[str] | None = None

        # Version SPNKr installée (trackée dans match_registry + sync_meta)
        self._spnkr_version: str | None = None
        try:
            from importlib.metadata import version as _pkg_version

            self._spnkr_version = _pkg_version("spnkr")
        except Exception:
            pass

        # Créer le resolver pour les métadonnées
        self._metadata_resolver = create_metadata_resolver(self._metadata_db_path)

    # =========================================================================
    # Connexions DuckDB
    # =========================================================================

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

    def _get_shared_connection(self) -> duckdb.DuckDBPyConnection | None:  # noqa: PLR0912
        """Retourne une connexion vers shared_matches.duckdb (R/W).

        Returns:
            Connexion DuckDB ou None si la base n'existe pas.
        """
        if self._shared_connection is not None:
            return self._shared_connection

        if self._shared_db_path is None or not self._shared_db_path.exists():
            logger.debug("shared_matches.duckdb absent, mode legacy v4")
            return None

        def _open_shared() -> duckdb.DuckDBPyConnection:
            return duckdb.connect(str(self._shared_db_path), read_only=False)

        try:
            self._shared_connection = _open_shared()
        except duckdb.IOException as e:
            err = str(e).lower()
            if "unique file handle conflict" in err or "already attached" in err:
                logger.warning(
                    "shared_matches.duckdb conflit de handle, libération et retry… (%s)", e
                )
                try:
                    from src.data.repositories.duckdb_repo import release_all_db_connections

                    release_all_db_connections()
                except Exception:
                    pass
                gc.collect()
                time.sleep(0.15)
                self._shared_connection = _open_shared()
            else:
                raise

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

        # Colonne sync_spnkr_version sur match_registry (migration v5.4)
        try:
            ensure_match_registry_spnkr_version(self._shared_connection)
        except Exception as e:
            logger.debug(f"Migration sync_spnkr_version shared: {e}")

        return self._shared_connection

    def _get_pve_connection(self) -> duckdb.DuckDBPyConnection:
        """Retourne (lazy) la connexion vers shared_pve.duckdb.

        Crée la base et applique le schéma PVE si elle n'existe pas encore.

        Returns:
            Connexion DuckDB R/W vers shared_pve.duckdb.
        """
        if self._pve_connection is not None:
            return self._pve_connection

        self._pve_db_path.parent.mkdir(parents=True, exist_ok=True)
        self._pve_connection = duckdb.connect(str(self._pve_db_path), read_only=False)
        self._pve_connection.execute("SET enable_object_cache = true")

        try:
            from src.data.sync.migrations import ensure_pve_schema

            ensure_pve_schema(self._pve_connection)
        except Exception as e:
            logger.warning(f"ensure_pve_schema: {e}")

        return self._pve_connection

    @property
    def shared_enabled(self) -> bool:
        """Indique si le mode shared_matches est activé."""
        return self._shared_db_path is not None and self._shared_db_path.exists()

    # =========================================================================
    # Schéma & migrations
    # =========================================================================

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
            with contextlib.suppress(Exception):
                conn.execute(f"DROP TABLE IF EXISTS {tbl} CASCADE")

        # Tables de sync (player-only)
        for stmt in SYNC_SCHEMA_DDL.split(";"):
            stmt = stmt.strip()
            if stmt:
                try:
                    conn.execute(stmt)
                except Exception as e:
                    if "already exists" not in str(e).lower():
                        logger.warning(f"Schema DDL warning: {e}")

        # S'assurer que la séquence pour career_progression existe (migration)
        self._ensure_career_progression_sequence()

        # Colonne spartan_id sur career_progression (migration v5.3+)
        self._ensure_career_progression_spartan_id()

        # Colonne had_bot_teammate sur player_match_enrichment (migration v5.5)
        from src.data.sync.migrations import ensure_bot_teammate_column

        ensure_bot_teammate_column(conn)

        # Index de performance sur tables locales (v5.1 Étape 2)
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
        ensure_career_progression_autoincrement(conn)

    def _ensure_career_progression_spartan_id(self) -> None:
        """Ajoute la colonne spartan_id à career_progression si absente."""
        conn = self._connection
        if conn is None:
            return
        add_spartan_id_to_career_progression(conn)

    # =========================================================================
    # Résolution XUID & matchs existants
    # =========================================================================

    def _resolve_xuid_from_db(self) -> str:
        """Résout le XUID depuis la DB joueur (fallback si non fourni).

        Stratégie :
        1. sync_meta (key='xuid')
        2. shared.xuid_aliases via gamertag
        """
        try:
            with duckdb.connect(str(self._player_db_path), read_only=True) as conn:
                # 1. sync_meta
                try:
                    r = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid'").fetchone()
                    if r and r[0] and str(r[0]).strip():
                        xuid = str(r[0]).strip()
                        logger.info(f"XUID résolu depuis sync_meta: {xuid}")
                        return xuid
                except Exception:
                    pass

                # 2. xuid_aliases via shared_matches.duckdb (v5.1)
                if self._gamertag:
                    try:
                        from src.utils.paths import get_shared_matches_path_from_player

                        shared_path = get_shared_matches_path_from_player(self._player_db_path)
                        if shared_path and shared_path.exists():
                            with duckdb.connect(str(shared_path), read_only=True) as shared_conn:
                                r = shared_conn.execute(
                                    "SELECT xuid FROM xuid_aliases WHERE gamertag = ? LIMIT 1",
                                    [self._gamertag],
                                ).fetchone()
                                if r and r[0] and str(r[0]).strip():
                                    xuid = str(r[0]).strip()
                                    logger.info(f"XUID résolu depuis shared.xuid_aliases: {xuid}")
                                    return xuid
                    except Exception:
                        pass

        except Exception as e:
            logger.debug(f"Impossible de résoudre le XUID depuis la DB: {e}")

        return ""

    def _load_existing_match_ids(self) -> set[str]:
        """Charge les IDs des matchs existants depuis shared + player DB (V5 finale).

        Un match est considéré « déjà traité » seulement si le joueur est présent
        dans shared.match_participants **ET** dans player_match_enrichment.
        """
        if self._existing_match_ids is not None:
            return self._existing_match_ids

        ids: set[str] = set()

        shared_conn = self._get_shared_connection()
        if shared_conn is not None and self._xuid:
            try:
                shared_ids_result = shared_conn.execute(
                    "SELECT DISTINCT match_id FROM match_participants WHERE xuid = ?", [self._xuid]
                ).fetchall()
                shared_ids = {str(r[0]) for r in shared_ids_result if r[0]}
                logger.debug(f"Chargé {len(shared_ids)} match IDs depuis shared.match_participants")

                enriched_ids: set[str] = set()
                conn = self._get_connection()
                try:
                    enr_result = conn.execute(
                        "SELECT DISTINCT match_id FROM player_match_enrichment"
                    ).fetchall()
                    enriched_ids = {str(r[0]) for r in enr_result if r[0]}
                except Exception:
                    pass

                ids = shared_ids & enriched_ids
                skipped = shared_ids - enriched_ids
                if skipped:
                    logger.info(
                        f"{len(skipped)} match(s) dans shared mais sans enrichment "
                        f"→ seront re-traités pour ce joueur"
                    )
            except Exception as e:
                logger.debug(f"Impossible de lire shared.match_participants: {e}")

        self._existing_match_ids = ids
        return ids

    # =========================================================================
    # Sync meta
    # =========================================================================

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

    # =========================================================================
    # Orchestration (sync_delta, sync_full, _sync_internal)
    # =========================================================================

    async def sync_delta(
        self,
        options: SyncOptions | None = None,
        *,
        progress_callback: Callable[[int, int], None] | None = None,
    ) -> SyncResult:
        """Synchronisation incrémentale (nouveaux matchs uniquement).

        S'arrête dès qu'un match déjà connu est rencontré.

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

    async def _sync_internal(  # noqa: C901, PLR0912, PLR0915
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

            # Sync du career rank (progression de rang) si activé
            if options.with_career_rank:
                try:
                    career_data = await self.sync_career_rank()
                    if career_data:
                        logger.info(
                            f"Career rank sync: {self._gamertag} → Rang {career_data.current_rank}"
                        )
                except Exception as e:
                    logger.warning(f"Career rank sync échoué (non bloquant): {e}")

            # Mettre à jour les métadonnées
            self._update_sync_meta("last_sync_at", datetime.now(timezone.utc).isoformat())
            self._update_sync_meta("last_sync_mode", "delta" if delta_mode else "full")
            self._update_sync_meta("last_sync_matches", str(result.matches_inserted))

            if self._xuid:
                self._update_sync_meta("xuid", self._xuid)
            if self._gamertag:
                self._update_sync_meta("gamertag", self._gamertag)
            if self._spnkr_version:
                self._update_sync_meta("spnkr_version", self._spnkr_version)

            # Commit final
            conn = self._get_connection()
            conn.commit()

            # Sprint 6 bis : Calcul batch des performance scores en tout dernier
            if result.matches_inserted > 0 and options.defer_performance_score:
                if self._shared_connection is not None:
                    with contextlib.suppress(Exception):
                        self._shared_connection.close()
                    self._shared_connection = None

                perf_count = self.batch_compute_performance_scores()
                logger.info(f"Performance scores calculés en batch : {perf_count}")

            # Recalculer les sessions dans player_match_enrichment après chaque sync.
            if result.matches_inserted > 0:
                try:
                    from src.data.sessions_backfill import backfill_sessions_for_player

                    sess_result = backfill_sessions_for_player(
                        db_path=self._player_db_path,
                        xuid=self._xuid,
                        conn=self._get_connection(),
                    )
                    updated = sess_result.get("sessions_updated", 0)
                    created = sess_result.get("sessions_created", 0)
                    logger.info(
                        f"Sessions recalculées post-sync : {created} créées, {updated} mises à jour"
                    )
                except Exception as e:
                    logger.warning(f"Erreur recalcul sessions post-sync : {e}")

            # Calculer les citations pour les nouveaux matchs (post-sync, comme les sessions).
            # La connexion shared est fermée (batch_compute_performance_scores l'a déjà fermée),
            # ce qui permet à citations_backfill de l'attacher en READ_ONLY sans conflit.
            if result.matches_inserted > 0:
                try:
                    from src.data.citations_backfill import backfill_citations_for_player

                    # Garde-fou : fermer shared si encore ouverte (cas defer_performance_score=False)
                    if self._shared_connection is not None:
                        with contextlib.suppress(Exception):
                            self._shared_connection.close()
                        self._shared_connection = None

                    cit_result = backfill_citations_for_player(
                        db_path=self._player_db_path,
                        xuid=self._xuid or "",
                        conn=self._get_connection(),
                    )
                    logger.info(
                        f"Citations post-sync : {cit_result['citations_computed']} matchs "
                        f"traités sur {cit_result['matches_processed']}"
                    )
                except Exception as e:
                    logger.warning(f"Erreur calcul citations post-sync : {e}")

        except Exception as e:
            result.errors.append(str(e))
            logger.error(f"Erreur sync: {e}")

        result.finished_at = datetime.now(timezone.utc)
        result.duration_seconds = time.time() - start_time

        logger.info(
            "Sync terminé [%s, mode=%s]: %d insérés, %d erreurs en %.1fs",
            self._gamertag,
            "delta" if delta_mode else "full",
            result.matches_inserted,
            len(result.errors),
            result.duration_seconds,
        )
        return result

    # =========================================================================
    # Player DB writes (enrichissements personnels)
    # =========================================================================

    def _insert_event_rows(self, rows: list) -> None:
        """Insère des lignes highlight_events en batch (Sprint 15)."""
        if not rows:
            return
        conn = self._get_connection()
        batch_insert_rows(conn, "highlight_events", rows, HIGHLIGHT_EVENT_COLUMNS)

    def _insert_personal_score_rows(self, rows: list) -> None:
        """Insère des lignes personal_score_awards en batch (Sprint 15)."""
        if not rows:
            return
        conn = self._get_connection()
        now = datetime.now(timezone.utc)
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

    def _load_friends_lazy(self) -> frozenset[str]:
        """Charge les XUIDs des amis depuis friends_defaults.json (cache interne).

        Returns:
            Set des XUIDs amis. Vide si pas de config disponible.
        """
        if self._friends_xuids is None:
            try:
                from src.data.sessions_backfill import get_friends_xuids_for_backfill

                conn = self._get_connection()
                self._friends_xuids = get_friends_xuids_for_backfill(
                    self._player_db_path,
                    self._xuid or "",
                    conn=conn,
                )
            except Exception:
                self._friends_xuids = frozenset()
        return self._friends_xuids

    def _insert_enrichment_row(self, match_id: str, match_row: MatchStatsRow) -> None:
        """Insère/met à jour une ligne dans player_match_enrichment (V5 finale).

        Args:
            match_id: ID du match
            match_row: Données du match (pour extraire teammates_signature, etc.)
        """
        conn = self._get_connection()
        now = datetime.now(timezone.utc)

        teammates_sig = getattr(match_row, "teammates_signature", None)

        try:
            from src.analysis.sessions import _parse_teammates_signature

            friends = self._load_friends_lazy()
            if friends:
                team_set = _parse_teammates_signature(teammates_sig)
                common = team_set & friends
                is_with_friends: bool | None = bool(common)
                known_teammates: int | None = len(common)
                friends_xuids: str | None = ",".join(sorted(common)) if common else None
            else:
                is_with_friends = None
                known_teammates = None
                friends_xuids = None
        except Exception:
            is_with_friends = None
            known_teammates = None
            friends_xuids = None

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

    # =========================================================================
    # Fermeture
    # =========================================================================

    def close(self) -> None:
        """Ferme les connexions DuckDB (player + shared + pve).

        Exécute un CHECKPOINT explicite sur chaque connexion R/W avant
        de la fermer. Sans cela, le WAL n'est pas flushed si d'autres
        connexions R/O (ex. ATTACH des sessions Streamlit) tiennent le
        fichier ouvert — le mtime ne changerait pas et les caches
        @st.cache_data ne seraient pas invalidés.
        """
        if self._connection:
            with contextlib.suppress(Exception):
                self._connection.execute("CHECKPOINT")
            with contextlib.suppress(Exception):
                self._connection.close()
            self._connection = None
        if self._shared_connection:
            with contextlib.suppress(Exception):
                self._shared_connection.execute("CHECKPOINT")
            with contextlib.suppress(Exception):
                self._shared_connection.close()
            self._shared_connection = None
        if self._pve_connection:
            with contextlib.suppress(Exception):
                self._pve_connection.execute("CHECKPOINT")
            with contextlib.suppress(Exception):
                self._pve_connection.close()
            self._pve_connection = None
