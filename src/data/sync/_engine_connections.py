"""Mixin de gestion des connexions DuckDB pour le DuckDBSyncEngine.

Gère les connexions player, shared et PVE, ainsi que la résolution XUID
et le chargement des match IDs existants.
"""

from __future__ import annotations

import gc
import logging
import time
from pathlib import Path

import duckdb

logger = logging.getLogger(__name__)

# Guard process-level : évite de relancer les migrations shared à chaque SyncEngine
_SHARED_MIGRATIONS_DONE: set[str] = set()


# =============================================================================
# Bootstrap shared_matches_v2.duckdb
# =============================================================================

_SHARED_SCHEMA_SQL = """
CREATE TABLE IF NOT EXISTS match_registry (
    match_id VARCHAR PRIMARY KEY,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    playlist_id VARCHAR, playlist_name VARCHAR,
    map_id VARCHAR, map_name VARCHAR,
    pair_id VARCHAR, pair_name VARCHAR,
    game_variant_id VARCHAR, game_variant_name VARCHAR,
    mode_category VARCHAR,
    is_ranked BOOLEAN DEFAULT FALSE,
    is_firefight BOOLEAN DEFAULT FALSE,
    duration_seconds INTEGER,
    playable_duration_seconds INTEGER,
    real_start_time TIMESTAMP,
    team_0_score SMALLINT, team_1_score SMALLINT,
    team_0_ps_score INTEGER, team_1_ps_score INTEGER,
    backfill_completed INTEGER DEFAULT 0,
    participants_loaded BOOLEAN DEFAULT FALSE,
    events_loaded BOOLEAN DEFAULT FALSE,
    medals_loaded BOOLEAN DEFAULT FALSE,
    first_sync_by VARCHAR, first_sync_at TIMESTAMP,
    last_updated_at TIMESTAMP, player_count SMALLINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS match_participants (
    match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
    gamertag VARCHAR, team_id INTEGER, outcome INTEGER,
    rank SMALLINT, score INTEGER,
    kills SMALLINT, deaths SMALLINT, assists SMALLINT,
    shots_fired INTEGER, shots_hit INTEGER,
    damage_dealt FLOAT, damage_taken FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (match_id, xuid)
);
CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq;
CREATE TABLE IF NOT EXISTS highlight_events (
    id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
    match_id VARCHAR NOT NULL, event_type VARCHAR NOT NULL,
    time_ms INTEGER,
    xuid VARCHAR,
    type_hint INTEGER, raw_json VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS killer_victim_pairs (
    match_id VARCHAR NOT NULL,
    killer_xuid VARCHAR NOT NULL, killer_gamertag VARCHAR,
    victim_xuid VARCHAR NOT NULL, victim_gamertag VARCHAR,
    kill_count INTEGER DEFAULT 1, time_ms INTEGER,
    is_validated BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS medals_earned (
    match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
    medal_name_id BIGINT NOT NULL, count SMALLINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (match_id, xuid, medal_name_id)
);
CREATE TABLE IF NOT EXISTS xuid_aliases (
    xuid VARCHAR PRIMARY KEY, gamertag VARCHAR NOT NULL,
    last_seen TIMESTAMP, source VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    description VARCHAR NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
"""


def _bootstrap_shared_matches_db(db_path) -> None:  # type: ignore[no-untyped-def]  # noqa: C901, PLR0915
    """Crée shared_matches_v2.duckdb avec le schéma v5 de base (idempotent)."""
    import pathlib

    path = pathlib.Path(db_path)
    path.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(path), read_only=False)
    try:
        for stmt in _SHARED_SCHEMA_SQL.split(";"):
            s = stmt.strip()
            if s:
                conn.execute(s)
        # Insérer la version de schéma seulement si absente
        conn.execute(
            "INSERT OR IGNORE INTO schema_version (version, description) "
            "VALUES (1, 'v5.0 — Création initiale du schéma shared_matches')"
        )
        logger.info("shared_matches_v2.duckdb initialisé : %s", path)
    finally:
        conn.close()


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

    def _get_shared_connection(self) -> duckdb.DuckDBPyConnection | None:  # noqa: PLR0912, C901
        """Retourne une connexion vers shared_matches_v2.duckdb.

        Mode R/O si self._shared_read_only=True (fanout), R/W sinon (sync principal).
        Returns None si la base n'existe pas — l'initialisation est assurée
        par _ensure_warehouse_dbs() dans le launcher (jamais en mode silencieux ici).

        Returns:
            Connexion DuckDB ou None si la base est absente ou inaccessible.
        """
        if self._shared_connection is not None:
            return self._shared_connection

        if self._shared_db_path is None:
            return None

        # Retourner None si absent — pas de bootstrap silencieux ici
        if not self._shared_db_path.exists():
            return None

        read_only: bool = getattr(self, "_shared_read_only", False)

        def _open_shared() -> duckdb.DuckDBPyConnection:
            return duckdb.connect(str(self._shared_db_path), read_only=read_only)

        try:
            self._shared_connection = _open_shared()
        except duckdb.Error as e:
            err = str(e).lower()
            if "unique file handle conflict" in err or "already attached" in err:
                logger.debug(
                    "shared_matches_v2.duckdb conflit de handle, libération et retry… (%s)", e
                )
                try:
                    from src.data.repositories.duckdb_repo import release_all_db_connections

                    release_all_db_connections()
                except Exception:
                    pass
                gc.collect()
                time.sleep(0.5)
                try:
                    self._shared_connection = _open_shared()
                except duckdb.Error as e2:
                    # 2ème tentative aussi échouée : log + return None
                    # pour éviter de faire remonter le Binder Error dans result.errors.
                    logger.debug(
                        "shared_matches_v2.duckdb : retry échoué (%s) — shared non disponible", e2
                    )
                    return None
            else:
                raise

        self._shared_connection.execute("SET enable_object_cache = true")

        self._run_shared_migrations(self._shared_connection, self._shared_db_path)

        return self._shared_connection

    @staticmethod
    def _run_shared_migrations(  # noqa: PLR0912
        conn: duckdb.DuckDBPyConnection,
        db_path: Path | None = None,
    ) -> None:
        """Applique les migrations sur la connexion shared."""
        # Guard process-level : clé = chemin résolu pour éviter collision entre
        # fichiers différents portant le même nom (ex. fixtures de test)
        db_key: str | None = None
        if db_path is not None:
            try:
                db_key = str(Path(db_path).resolve())
            except Exception as e:
                logger.debug("event=db_key_resolve_failed error=%s", e)
        if not db_key:
            try:
                db_key = conn.execute("SELECT current_database()").fetchone()[0]  # type: ignore[index]
            except Exception as e:
                logger.debug("event=db_key_current_database_failed error=%s", e)
        if db_key and db_key in _SHARED_MIGRATIONS_DONE:
            return
        if db_key:
            _SHARED_MIGRATIONS_DONE.add(db_key)
        try:
            from src.data.sync.migrations import ensure_match_participants_columns

            ensure_match_participants_columns(conn)
        except Exception as e:
            logger.debug("Migration match_participants shared: %s", e)

        try:
            from src.data.sync.migrations import ensure_performance_indexes

            ensure_performance_indexes(conn)
        except Exception as e:
            logger.debug("Index performance shared: %s", e)

        try:
            from src.data.sync.migrations import ensure_match_registry_spnkr_version

            ensure_match_registry_spnkr_version(conn)
        except Exception as e:
            logger.debug("Migration sync_spnkr_version shared: %s", e)

        try:
            from src.data.sync.migrations import ensure_weapon_kills_table

            ensure_weapon_kills_table(conn)
        except Exception as e:
            logger.debug("Migration weapon_kills shared: %s", e)

        try:
            from src.data.sync.migrations import ensure_resolution_views

            ensure_resolution_views(conn)
        except Exception as e:
            logger.debug("ensure_resolution_views shared: %s", e)

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

                # 2. v_gamertag_lookup via shared_matches_v2.duckdb (v5.8)
                if self._gamertag:
                    try:
                        from src.utils.paths import get_shared_matches_path_from_player
                        from src.utils.xuid import lookup_xuid_for_gamertag

                        shared_path = get_shared_matches_path_from_player(self._player_db_path)
                        if shared_path and shared_path.exists():
                            with duckdb.connect(str(shared_path), read_only=True) as shared_conn:
                                resolved = lookup_xuid_for_gamertag(shared_conn, self._gamertag)
                                if resolved:
                                    xuid = resolved
                                    logger.info("XUID résolu depuis v_gamertag_lookup: %s", xuid)
                                    return xuid
                    except Exception:
                        pass

        except Exception as e:
            logger.debug("Impossible de résoudre le XUID depuis la DB: %s", e)

        # Normal au premier lancement (sync_meta et shared_matches non encore peuplés).
        # Le XUID sera résolu et persisté lors du premier sync via l'API.
        logger.debug(
            "XUID non résolu après tous les fallbacks (sync_meta + v_gamertag_lookup) pour %s",
            self._player_db_path,
        )
        return ""

    def _load_existing_match_ids(self) -> set[str]:
        """Charge les IDs des matchs existants depuis shared + player DB (V5 finale).

        Un match est considéré « déjà traité » seulement si le joueur est présent
        dans shared.match_participants **ET** dans player_match_enrichment **ET** soit :
        - a des personal_score_awards dans la player DB, OU
        - a personal_score = 0 dans shared.match_participants (légitimement vide).

        Les matchs avec enrichment mais sans personal_score_awards alors que
        personal_score > 0 (ex : sync partiel dû à un bug antérieur) sont exclus
        pour être retraités via _process_known_match().
        """
        if self._existing_match_ids is not None:
            return self._existing_match_ids

        ids: set[str] = set()

        shared_conn = self._get_shared_connection()
        if shared_conn is not None and self._xuid:
            try:
                # Matchs de ce joueur dans shared, avec leur personal_score
                shared_rows = shared_conn.execute(
                    "SELECT match_id, COALESCE(personal_score, 0) FROM match_participants WHERE xuid = ?",
                    [self._xuid],
                ).fetchall()
                shared_ids = {str(r[0]) for r in shared_rows if r[0]}
                shared_score_zero = {str(r[0]) for r in shared_rows if r[0] and (r[1] or 0) == 0}
                logger.debug(
                    "Chargé %d match IDs depuis shared.match_participants (%d avec score=0)",
                    len(shared_ids),
                    len(shared_score_zero),
                )

                # Q2 (consolidée): player — enrichment + awards en une seule requête
                # LEFT JOIN : évite 2 requêtes + 2 intersections Python séparées.
                enriched_ids: set[str] = set()
                scored_ids: set[str] = set()
                conn = self._get_connection()
                try:
                    player_rows = conn.execute(
                        "SELECT pme.match_id, "
                        "  MAX(CASE WHEN psa.match_id IS NOT NULL THEN 1 ELSE 0 END)"
                        "    AS has_awards "
                        "FROM player_match_enrichment pme "
                        "LEFT JOIN personal_score_awards psa"
                        "  ON pme.match_id = psa.match_id "
                        "GROUP BY pme.match_id"
                    ).fetchall()
                    enriched_ids = {str(r[0]) for r in player_rows if r[0]}
                    scored_ids = {str(r[0]) for r in player_rows if r[0] and r[1] == 1}
                except Exception as e:
                    logger.debug("_load_existing_match_ids: lecture player DB: %s", e)

                # Un match est "terminé" s'il a enrichment ET (personal_score_awards OU score=0)
                # Note : OR COALESCE(personal_score, 0) = 0 est intentionnel — les matchs
                # légitimement sans awards (score nul) ne doivent pas être re-traités.
                enriched_and_done = enriched_ids & (scored_ids | shared_score_zero)
                ids = shared_ids & enriched_and_done

                missing_personal_scores = (shared_ids & enriched_ids) - enriched_and_done
                skipped = shared_ids - enriched_ids
                if skipped:
                    logger.info(
                        "%d match(s) dans shared mais sans enrichment → seront re-traités",
                        len(skipped),
                    )
                if missing_personal_scores:
                    logger.info(
                        "%d match(s) avec enrichment mais sans personal_score_awards "
                        "(personal_score > 0) → seront re-traités",
                        len(missing_personal_scores),
                    )
                logger.debug(
                    "event=existing_ids_loaded gamertag=%s count=%d source=sql_consolidated",
                    getattr(self, "_gamertag", "?"),
                    len(ids),
                )
            except Exception as e:
                logger.debug("Impossible de lire shared.match_participants: %s", e)

        self._existing_match_ids = ids
        return ids

    def _get_latest_match_id_in_db(self) -> str | None:
        """Retourne le match_id le plus récent du joueur en DB (sans charger tous les IDs)."""
        shared_conn = self._get_shared_connection()
        if shared_conn is None or not self._xuid:
            return None
        try:
            row = shared_conn.execute(
                "SELECT mr.match_id FROM match_registry mr "
                "JOIN match_participants mp ON mr.match_id = mp.match_id "
                "WHERE mp.xuid = ? ORDER BY mr.start_time DESC LIMIT 1",
                [self._xuid],
            ).fetchone()
            return str(row[0]) if row else None
        except Exception as e:
            logger.debug("event=latest_match_id_failed error=%s", e)
            return None

    def _is_player_fully_synced_for_match(self, match_id: str) -> bool:
        """Vérifie que le joueur a des données complètes pour ce match (enrichment + PSA).

        Utilisé par le HEAD check delta pour éviter un court-circuit prématuré quand
        un coéquipier a inséré le match dans shared_matches (via son propre sync) mais
        que le joueur courant n'a jamais collecté ses PSA via l'API.

        Un match est considéré « complet » si :
        - le joueur a une ligne dans player_match_enrichment, ET
        - soit des personal_score_awards existent, soit personal_score = 0 dans shared.

        Returns:
            True si le match est pleinement synchronisé pour ce joueur, False sinon.
            En cas d'erreur, retourne False (conservateur : ne pas court-circuiter).
        """
        try:
            conn = self._get_connection()
            has_enrichment = (
                conn.execute(
                    "SELECT COUNT(*) FROM player_match_enrichment WHERE match_id = ?",
                    [match_id],
                ).fetchone()[0]
                > 0
            )
            if not has_enrichment:
                return False

            has_psa = (
                conn.execute(
                    "SELECT COUNT(*) FROM personal_score_awards WHERE match_id = ?",
                    [match_id],
                ).fetchone()[0]
                > 0
            )
            if has_psa:
                return True

            # Match légitimement sans PSA si personal_score = 0 dans shared
            shared_conn = self._get_shared_connection()
            if shared_conn and self._xuid:
                score_row = shared_conn.execute(
                    "SELECT COALESCE(personal_score, 0) FROM match_participants "
                    "WHERE match_id = ? AND xuid = ?",
                    [match_id, self._xuid],
                ).fetchone()
                if score_row and score_row[0] == 0:
                    return True

            return False
        except Exception as e:
            logger.debug("event=is_fully_synced_check_failed match=%s error=%s", match_id, e)
            return False

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
                try:
                    conn.execute("CHECKPOINT")
                except Exception as e:
                    logger.warning("event=checkpoint_failed conn=%s error=%s", attr, e)
                try:
                    conn.close()
                except Exception as e:
                    logger.debug("event=conn_close_failed conn=%s error=%s", attr, e)
                setattr(self, attr, None)
