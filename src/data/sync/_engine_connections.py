"""Mixin de gestion des connexions DuckDB pour le DuckDBSyncEngine.

Gère les connexions player, shared et PVE, ainsi que la résolution XUID
et le chargement des match IDs existants.
"""

from __future__ import annotations

import contextlib
import gc
import logging
import time

import duckdb

logger = logging.getLogger(__name__)


class ConnectionMixin:
    """Mixin de gestion des connexions DuckDB pour DuckDBSyncEngine."""

    # =========================================================================
    # Connexions DuckDB
    # =========================================================================

    def _get_connection(self) -> duckdb.DuckDBPyConnection:
        """Retourne une connexion DuckDB (lecture/écriture)."""
        if self._connection is None:
            self._player_db_path.parent.mkdir(parents=True, exist_ok=True)

            self._connection = duckdb.connect(
                str(self._player_db_path),
                read_only=False,
            )

            self._connection.execute("SET memory_limit = '512MB'")
            self._connection.execute("SET enable_object_cache = true")

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

        # Migrations shared
        try:
            from src.data.sync.migrations import ensure_match_participants_columns

            ensure_match_participants_columns(self._shared_connection)
        except Exception as e:
            logger.debug("Migration match_participants shared: %s", e)

        try:
            from src.data.sync.migrations import ensure_performance_indexes

            ensure_performance_indexes(self._shared_connection)
        except Exception as e:
            logger.debug("Index performance shared: %s", e)

        try:
            from src.data.sync.migrations import ensure_match_registry_spnkr_version

            ensure_match_registry_spnkr_version(self._shared_connection)
        except Exception as e:
            logger.debug("Migration sync_spnkr_version shared: %s", e)

        return self._shared_connection

    def _get_pve_connection(self) -> duckdb.DuckDBPyConnection:
        """Retourne (lazy) la connexion vers shared_pve.duckdb."""
        if self._pve_connection is not None:
            return self._pve_connection

        self._pve_db_path.parent.mkdir(parents=True, exist_ok=True)
        self._pve_connection = duckdb.connect(str(self._pve_db_path), read_only=False)
        self._pve_connection.execute("SET enable_object_cache = true")

        try:
            from src.data.sync.migrations import ensure_pve_schema

            ensure_pve_schema(self._pve_connection)
        except Exception as e:
            logger.warning("ensure_pve_schema: %s", e)

        return self._pve_connection

    @property
    def shared_enabled(self) -> bool:
        """Indique si le mode shared_matches est activé."""
        return self._shared_db_path is not None and self._shared_db_path.exists()

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
                        logger.info("XUID résolu depuis sync_meta: %s", xuid)
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
                                    logger.info("XUID résolu depuis shared.xuid_aliases: %s", xuid)
                                    return xuid
                    except Exception:
                        pass

        except Exception as e:
            logger.debug("Impossible de résoudre le XUID depuis la DB: %s", e)

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
                logger.debug(
                    "Chargé %d match IDs depuis shared.match_participants", len(shared_ids)
                )

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
                        "%d match(s) dans shared mais sans enrichment → seront re-traités",
                        len(skipped),
                    )
            except Exception as e:
                logger.debug("Impossible de lire shared.match_participants: %s", e)

        self._existing_match_ids = ids
        return ids

    # =========================================================================
    # Fermeture
    # =========================================================================

    def close(self) -> None:
        """Ferme les connexions DuckDB (player + shared + pve).

        Exécute un CHECKPOINT explicite sur chaque connexion R/W avant
        de la fermer pour flusher le WAL.
        """
        for attr in ("_connection", "_shared_connection", "_pve_connection"):
            conn = getattr(self, attr, None)
            if conn:
                with contextlib.suppress(Exception):
                    conn.execute("CHECKPOINT")
                with contextlib.suppress(Exception):
                    conn.close()
                setattr(self, attr, None)
