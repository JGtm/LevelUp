"""Moteur de synchronisation DuckDB unifié.

Ce module contient le DuckDBSyncEngine qui orchestre tout le pipeline :
API SPNKr → Transformation → DuckDB (direct, sans intermédiaire)

Architecture mixin :
    Le code métier est réparti dans 8 modules mixin :
    - _engine_connections.py : gestion des connexions DuckDB + résolution XUID
    - _engine_schema.py     : DDL et migrations du schéma
    - _shared_writes.py     : insertions dans shared_matches.duckdb
    - _performance.py       : calcul des scores de performance
    - _skill_rating.py      : CSR (ranked) + LUSR (TrueSkill 2)
    - _career.py            : synchronisation du rang carrière
    - _aggregates.py        : rafraîchissement agrégats post-sync
    - _match_processing.py  : traitement des matchs (fetch, transform, insert)

    engine.py conserve l'orchestrateur, __init__, sync_meta et les helpers d'écriture.

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
import logging
import time
from collections.abc import Callable
from datetime import datetime, timezone
from pathlib import Path

from src.data.sync._aggregates import AggregatesMixin
from src.data.sync._career import CareerMixin
from src.data.sync._engine_connections import ConnectionMixin
from src.data.sync._engine_fanout import FanoutEnrichmentMixin
from src.data.sync._engine_schema import SchemaMixin
from src.data.sync._engine_weapon_kills import WeaponKillsEngineMixin
from src.data.sync._engine_writes import EnrichedWritesMixin
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
from src.data.sync.api_factory import create_api_client
from src.data.sync.models import (
    SyncOptions,
    SyncResult,
)
from src.data.sync.transformers import (
    create_metadata_resolver,
)
from src.utils.paths import (
    get_pve_db_path_from_player,
    get_shared_matches_path,
    get_shared_matches_path_from_player,
)

logger = logging.getLogger(__name__)

# Re-exports pour compatibilité (utilisé par les tests et scripts externes)
__all__ = ["DuckDBSyncEngine", "get_tokens_for_player", "SPNKrAPIClient"]


def _resolve_token_scope(gamertag: str) -> str:
    """Détermine le scope de token disponible pour un joueur sans déclencher OAuth.

    Vérifie uniquement la présence d'une env var ou d'un token DB — pas d'échange.

    Returns:
        "player" si un token per-joueur est configuré, "any" sinon.
    """
    import os

    from src.utils.env import normalize_gamertag_for_env

    env_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{normalize_gamertag_for_env(gamertag)}"
    if os.environ.get(env_key, "").strip():
        return "player"
    # Vérification DB cache (Xbox OAuth UI) — lecture uniquement, sans échange
    try:
        from src.utils.paths import PLAYERS_DIR

        _db = PLAYERS_DIR / gamertag / "stats.duckdb"
        if _db.exists():
            from src.ui.xbox_oauth import load_refresh_token as _lrt

            if _lrt(_db):
                return "player"
    except Exception:
        pass
    return "any"


# =============================================================================
# DuckDBSyncEngine — Orchestrateur principal
# =============================================================================


class DuckDBSyncEngine(
    ConnectionMixin,
    SchemaMixin,
    SharedWritesMixin,
    PerformanceMixin,
    SkillRatingMixin,
    CareerMixin,
    AggregatesMixin,
    WeaponKillsEngineMixin,
    MatchProcessingMixin,
    EnrichedWritesMixin,
    FanoutEnrichmentMixin,
):
    """Moteur de synchronisation API → DuckDB unifié.

    Gère tout le pipeline en une seule étape :
    1. Fetch depuis l'API SPNKr
    2. Transformation via transformers.py
    3. Upsert direct dans DuckDB
    4. Mise à jour des agrégats

    Thread-safe via lock asyncio pour les écritures DB.

    Mixins :
        ConnectionMixin       — connexions DuckDB + résolution XUID
        SchemaMixin           — schéma DDL + migrations
        SharedWritesMixin     — insertions dans shared_matches.duckdb
        PerformanceMixin      — calcul des scores de performance
        SkillRatingMixin      — CSR + LUSR (TrueSkill 2)
        CareerMixin           — rang carrière
        AggregatesMixin       — rafraîchissement agrégats
        MatchProcessingMixin  — traitement des matchs
        FanoutEnrichmentMixin — enrichissement des coéquipiers post-sync
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
        shared_read_only: bool = False,
    ) -> None:
        """
        Args:
            player_db_path: Chemin vers stats.duckdb du joueur.
            xuid: XUID du joueur.
            gamertag: Gamertag pour l'identification API.
            metadata_db_path: Chemin vers metadata.duckdb (auto-détecté si None).
            shared_db_path: Chemin vers shared_matches.duckdb (auto-détecté si None).
            tokens: Tokens SPNKr pré-fournis (sinon récupérés depuis env).
            shared_read_only: Si True, ouvre shared_matches.duckdb en lecture seule.
                Utiliser True pour le fanout (ne fait que lire shared), False (défaut)
                pour le sync principal (écrit dans shared).
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

        # Auto-détection du chemin shared_matches (v5/v6)
        if shared_db_path is None:
            self._shared_db_path: Path | None = (
                get_shared_matches_path_from_player(self._player_db_path)
                or self._player_db_path.parent.parent.parent
                / "warehouse"
                / get_shared_matches_path().name
            )
            logger.debug("SyncEngine: shared_db_path auto-détecté → %s", self._shared_db_path)
        else:
            self._shared_db_path = Path(shared_db_path)
            logger.debug("SyncEngine: shared_db_path explicit → %s", self._shared_db_path)

        self._connection = None
        self._shared_connection = None
        self._shared_read_only = shared_read_only
        self._db_lock = asyncio.Lock()
        self._shared_db_lock = asyncio.Lock()
        self._existing_match_ids: set[str] | None = None

        # Base PVE séparée (shared_pve.duckdb) — v5.2
        self._pve_db_path: Path = get_pve_db_path_from_player(self._player_db_path)
        self._pve_connection = None
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

    def _save_sync_metadata(self, delta_mode: bool, matches_inserted: int) -> None:
        """Sauvegarde toutes les métadonnées de sync en une fois."""
        self._update_sync_meta("last_sync_at", datetime.now(timezone.utc).isoformat())
        self._update_sync_meta("last_sync_mode", "delta" if delta_mode else "full")
        self._update_sync_meta("last_sync_matches", str(matches_inserted))
        if self._xuid:
            self._update_sync_meta("xuid", self._xuid)
        if self._gamertag:
            self._update_sync_meta("gamertag", self._gamertag)
        if self._spnkr_version:
            self._update_sync_meta("spnkr_version", self._spnkr_version)

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

        options = options.with_resolved_batch_size()

        try:
            if self._tokens is None:
                self._tokens = await get_tokens_from_env()

            # C: log token scope résolu — déterminé par présence d'un token per-joueur
            # (sans déclencher d'échange OAuth — lecture env/DB uniquement).
            _token_scope = _resolve_token_scope(self._gamertag)
            logger.info(
                "event=token_resolved gamertag=%s scope=%s mode=%s",
                self._gamertag,
                _token_scope,
                "delta" if delta_mode else "full",
            )

            async with create_api_client(
                tokens=self._tokens,
                requests_per_second=options.requests_per_second,
            ) as client:
                # B.1: HEAD-first short-circuit en mode delta — évite de charger
                # existing_ids si le joueur est déjà à jour (1 appel API suffisant).
                if delta_mode:
                    _latest_db = self._get_latest_match_id_in_db()
                    try:
                        _head = await client.get_match_history(
                            self._gamertag,
                            match_type=options.match_type,
                            start=0,
                            count=1,
                        )
                        if _head and _latest_db and _head[0].match_id == _latest_db:
                            logger.info(
                                "event=delta_head_check gamertag=%s short_circuit=true latest=%s",
                                self._gamertag,
                                _latest_db[:8],
                            )
                            # Toujours enregistrer la date du sync même si aucun nouveau match
                            # (sinon l'indicateur affiche la date du dernier sync avec insertions)
                            self._save_sync_metadata(delta_mode=True, matches_inserted=0)
                            self._get_connection().commit()
                            # Réparer les PME manquants chez les coéquipiers
                            # (cas : session jouée ensemble non encore enrichie côté coéquipier)
                            self.fanout_repair_missing_scores()
                            result.finished_at = datetime.now(timezone.utc)
                            result.duration_seconds = time.time() - start_time
                            return result
                        logger.debug(
                            "event=delta_head_check gamertag=%s short_circuit=false latest_db=%s",
                            self._gamertag,
                            (_latest_db or "")[:8] or "none",
                        )
                    except Exception as _hce:
                        logger.warning(
                            "event=delta_head_check_failed gamertag=%s error=%s — pipeline normal",
                            self._gamertag,
                            _hce,
                        )

                existing_ids = self._load_existing_match_ids()
                logger.info("Matchs existants en DB: %d", len(existing_ids))

                if delta_mode and not existing_ids:
                    logger.warning("Mode delta mais aucun match existant!")

                result = await self._process_matches(
                    client,
                    options,
                    existing_ids,
                    delta_mode=delta_mode,
                    progress_callback=progress_callback,
                )

            await self._run_post_sync_pipeline(options, result, delta_mode=delta_mode)

        except Exception as e:
            result.errors.append(str(e))
            logger.error("Erreur sync: %s", e)

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

    async def _run_post_sync_pipeline(
        self,
        options: SyncOptions,
        result: SyncResult,
        *,
        delta_mode: bool,
    ) -> None:
        """Pipeline post-traitement après insertion des matchs.

        Ordre contraint (dépendances de données) :
        1. Agrégats (shared → aggregates)
        2. Career rank (player-gated, non bloquant)
        3. Sync metadata + commit player DB
        4. Perf scores, sessions, citations (lecture shared R/O si nouveaux matchs)
        5. Detach shared + LUSR (toujours recalculé pour rattraper les ratings manquants)
        6. Fan-out enrichissement autres joueurs (après detach shared)
        """
        if result.matches_inserted > 0:
            await self._refresh_aggregates_async(new_ids=result.inserted_match_ids)

        await self._run_career_rank_if_needed(options)

        # 3. Métadonnées + commit player DB
        self._save_sync_metadata(delta_mode, result.matches_inserted)
        self._get_connection().commit()

        # 4. Post-sync : performance scores, sessions, citations (si nouveaux matchs)
        if result.matches_inserted > 0:
            await self._run_post_sync_compute(options)

        # 5. LUSR toujours recalculé (rattrapage des ratings manquants)
        self._detach_shared_from_player_conn()
        self._run_lusr_post_sync()

        # 6. Fan-out après detach — shared libéré (voir _engine_fanout.py)
        if result.inserted_match_ids:
            self._enrich_other_registered_players(result.inserted_match_ids)

    async def _run_career_rank_if_needed(self, options: SyncOptions) -> None:
        """Synchronise le rang carrière si demandé dans les options (non bloquant)."""
        if not options.with_career_rank:
            return
        try:
            career_data = await self.sync_career_rank()
            if career_data:
                logger.info(
                    "Career rank sync: %s → Rang %s", self._gamertag, career_data.current_rank
                )
        except Exception as e:
            logger.warning("Career rank sync échoué (non bloquant): %s", e)

    def _run_lusr_post_sync(self) -> None:
        """Calcule les ratings LUSR manquants — appelé inconditionnellement après tout sync.

        Contrairement aux autres traitements post-sync, le LUSR est recalculé
        même si aucun nouveau match n'a été inséré, afin de rattraper les ratings
        manquants suite à un sync partiel ou une erreur précédente.

        Ferme `_shared_connection` si elle est encore ouverte, puis délègue à
        `batch_compute_lusr` qui la rouvrira et absorbera tout Binder Error éventuel.
        Appelé que `matches_inserted` soit 0 ou >0 (dans ce dernier cas
        `_run_post_sync_compute` avait déjà fermé `_shared_connection`).
        """
        try:
            # Fermer la connexion shared si elle est encore ouverte,
            # pour qu'elle puisse être rouverte proprement dans batch_compute_lusr.
            if self._shared_connection is not None:
                try:
                    self._shared_connection.close()
                except Exception as e:
                    logger.debug("event=shared_conn_close_failed step=lusr_post_sync error=%s", e)
                self._shared_connection = None
            lusr_count = self.batch_compute_lusr(force=False)
            if lusr_count > 0:
                logger.info("LUSR calculés post-sync : %d matchs", lusr_count)
        except Exception as e:
            logger.warning("Erreur calcul LUSR post-sync (non bloquant) : %s", e)

    def _detach_shared_from_player_conn(self) -> None:
        """Détache shared_matches.duckdb de la connexion joueur si attaché.

        Nécessaire après des opérations qui ATTACH shared en READ_ONLY
        sur la connexion joueur (ex: citations_backfill), pour libérer
        le file handle et permettre à batch_compute_lusr d'ouvrir
        shared_matches.duckdb en R/W.
        """
        try:
            conn = self._get_connection()
            dbs = conn.execute("SELECT database_name, path FROM duckdb_databases()").fetchall()
            for db_name, db_path_val in dbs:
                if (
                    db_path_val
                    and "shared_matches" in str(db_path_val).lower()
                    and db_name != "memory"
                ):
                    conn.execute(f"DETACH {db_name}")
                    logger.debug("Détaché %s de la connexion joueur", db_name)
        except Exception as e:
            logger.debug("_detach_shared_from_player_conn (non bloquant): %s", e)

    def _post_sync_citations_sync(self) -> dict[str, int]:
        """Wrapper citations pour run_in_executor — ouvre sa propre connexion R/W.

        Thread-safe via DuckDB MVCC : écrit dans ``match_citations`` uniquement.
        """
        try:
            from src.data.citations_backfill import backfill_citations_for_player

            return backfill_citations_for_player(
                db_path=self._player_db_path,
                xuid=self._xuid or "",
            )
        except Exception as e:
            logger.warning("Citations post-sync (thread) : %s", e)
            return {"matches_processed": 0, "citations_computed": 0}

    async def _run_post_sync_compute(self, options: SyncOptions) -> None:
        """Axe 1 : citations en thread parallèle, perf→sessions→dominance sérialisés.

        Note : le LUSR est calculé séparément dans _run_lusr_post_sync().
        """
        import time as _time

        _t0 = _time.perf_counter()
        logger.info("post_sync: démarrage (citations en thread, writes player DB sérialisées)")

        loop = asyncio.get_running_loop()
        if self._shared_connection is not None:
            try:
                self._shared_connection.close()
            except Exception as e:
                logger.debug("event=shared_conn_close_failed step=post_sync_compute error=%s", e)
            self._shared_connection = None
        cit_future = loop.run_in_executor(None, self._post_sync_citations_sync)

        perf_count = 0
        if options.defer_performance_score:
            perf_count = self.batch_compute_performance_scores()
            logger.info("Performance scores calculés en batch : %d", perf_count)

        sess_created = sess_updated = 0
        try:
            from src.data.sessions_backfill import backfill_sessions_for_player

            sess = backfill_sessions_for_player(
                db_path=self._player_db_path,
                xuid=self._xuid,
                conn=self._get_connection(),
            )
            sess_created = sess.get("sessions_created", 0)
            sess_updated = sess.get("sessions_updated", 0)
            logger.info(
                "Sessions post-sync : %d créées, %d màj",
                sess_created,
                sess_updated,
            )
        except Exception as e:
            logger.warning("Sessions post-sync : %s", e)

        self._compute_dominance_post_sync()

        cit_result = await cit_future
        elapsed = _time.perf_counter() - _t0
        logger.info(
            "post_sync: terminé en %.1fs (perf=%d sessions=%d/%d citations=%d/%d)",
            elapsed,
            perf_count,
            sess_created,
            sess_updated,
            cit_result["citations_computed"],
            cit_result["matches_processed"],
        )

    def _compute_dominance_post_sync(self) -> None:
        """Calcule les dominance flags pour les matchs nouvellement synchronisés."""
        try:
            from src.data.dominance_backfill import compute_dominance_for_player

            if self._shared_connection is not None:
                try:
                    self._shared_connection.close()
                except Exception as e:
                    logger.debug("event=shared_conn_close_failed step=dominance error=%s", e)
                self._shared_connection = None

            import duckdb as _ddb

            shared_path = self._shared_db_path
            if shared_path and shared_path.exists():
                with _ddb.connect(str(shared_path), read_only=True) as _sconn:
                    dom_result = compute_dominance_for_player(
                        self._get_connection(), _sconn, self._xuid or ""
                    )
                    logger.info(
                        "Dominance flags post-sync : %d traités (domination: %d, humiliation: %d)",
                        dom_result["processed"],
                        dom_result["domination"],
                        dom_result["humiliation"],
                    )
        except Exception as e:
            logger.warning("Erreur calcul dominance post-sync : %s", e)

    def _verify_enrichment_completeness(self, inserted_ids: list[str]) -> None:
        """Vérification post-pipeline : détecte les matchs sans enrichissement.

        Ne relance pas les calculs — signale uniquement via les logs pour
        permettre l'investigation sans masquer un bug de connexion sous-jacent.
        """
        if not inserted_ids:
            return
        conn = self._get_connection()
        placeholders = ", ".join(["?" for _ in inserted_ids])
        try:
            missing_perf = conn.execute(
                f"SELECT COUNT(*) FROM player_match_enrichment "
                f"WHERE match_id IN ({placeholders}) AND performance_score IS NULL",
                inserted_ids,
            ).fetchone()[0]
            missing_sess = conn.execute(
                f"SELECT COUNT(*) FROM player_match_enrichment "
                f"WHERE match_id IN ({placeholders}) AND session_id IS NULL",
                inserted_ids,
            ).fetchone()[0]
            if missing_perf > 0 or missing_sess > 0:
                logger.warning(
                    "event=enrichment_incomplete perf_missing=%d sessions_missing=%d total=%d",
                    missing_perf,
                    missing_sess,
                    len(inserted_ids),
                )
            else:
                logger.info("event=enrichment_complete total=%d", len(inserted_ids))
        except Exception as e:
            logger.debug("_verify_enrichment_completeness: %s", e)
