"""
Repository DuckDB natif pour les données joueur.
(Native DuckDB repository for player data)

HOW IT WORKS:
Ce repository utilise exclusivement DuckDB :
1. data/warehouse/metadata.duckdb : Référentiels (playlists, maps, médailles)
2. data/players/{gamertag}/stats.duckdb : Données joueur (matchs, médailles, etc.)
3. data/players/{gamertag}/archive/*.parquet : Archives (cold storage)

Les jointures entre les deux DBs sont faites via ATTACH.
Les archives Parquet peuvent être lues via `load_matches_from_archives()`.
"""

from __future__ import annotations

import contextlib
import logging
import threading
import time
import weakref
from pathlib import Path
from typing import Any

import duckdb

from src.data.repositories._antagonists_repo import AntagonistsMixin
from src.data.repositories._archives_repo import ArchivesMixin
from src.data.repositories._arrow_bridge import result_to_polars
from src.data.repositories._events_repo import EventsMixin
from src.data.repositories._match_queries import MatchQueriesMixin
from src.data.repositories._materialized_views import MaterializedViewsMixin
from src.data.repositories._medals_repo import MedalsMixin
from src.data.repositories._roster_loader import RosterLoaderMixin

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Registre global des instances DuckDBRepository (weakref)
# Permet de fermer les connexions existantes lorsqu'un autre composant
# (ex. MediaIndexer) a besoin d'ouvrir la même DB avec un mode différent.
# ---------------------------------------------------------------------------
_instances: weakref.WeakSet[DuckDBRepository] = weakref.WeakSet()  # type: ignore[name-defined]
_instances_lock = threading.Lock()

# ---------------------------------------------------------------------------
# Flag global de mode sync
# Lorsqu'actif, DuckDBRepository ne tente PAS d'attacher shared_matches.duckdb.
# Le DuckDBSyncEngine ouvre cette base en R/W (connexion directe) ; DuckDB
# interdit qu'un même fichier soit ouvert sous deux noms différents dans le
# même processus. Le flag empêche tout conflit avec les threads Streamlit.
# ---------------------------------------------------------------------------
_sync_mode = threading.Event()


def begin_sync_mode() -> None:
    """Active le mode sync : les DuckDBRepository n'attacheront plus shared_matches.

    À appeler avant que le DuckDBSyncEngine ouvre shared_matches.duckdb en R/W.
    """
    _sync_mode.set()
    logger.debug("begin_sync_mode: ATTACH shared_matches suspendu")


def end_sync_mode() -> None:
    """Désactive le mode sync : les futurs DuckDBRepository peuvent attacher shared_matches."""
    _sync_mode.clear()
    logger.debug("end_sync_mode: ATTACH shared_matches rétabli")


def release_db_connections(db_path: str | Path) -> int:
    """Ferme toutes les connexions DuckDBRepository ouvertes vers *db_path*.

    Les repositories concernés se reconnecteront paresseusement lors du
    prochain appel à ``_get_connection()``.

    Returns:
        Nombre de connexions fermées.
    """
    target = str(Path(db_path).resolve())
    closed = 0
    with _instances_lock:
        for repo in list(_instances):
            try:
                if str(repo._player_db_path.resolve()) == target and repo._connection is not None:
                    with contextlib.suppress(Exception):
                        repo._connection.close()
                    repo._connection = None
                    repo._attached_dbs.clear()
                    closed += 1
            except Exception:
                pass
    if closed:
        logger.debug("release_db_connections: %d connexion(s) fermée(s) pour %s", closed, db_path)
    return closed


def release_all_db_connections() -> int:
    """Ferme toutes les connexions DuckDBRepository actives (tous joueurs confondus).

    Nécessaire avant d'ouvrir shared_matches.duckdb directement (ex. sync),
    car DuckDB interdit qu'un même fichier soit utilisé sous deux noms différents
    dans le même processus : les ATTACH 'shared_matches.duckdb' AS shared existants
    entreraient en conflit avec l'ouverture directe qui lui donne le nom 'shared_matches'.

    Les repositories se reconnecteront paresseusement au prochain accès.

    Returns:
        Nombre de connexions fermées.
    """
    closed = 0
    with _instances_lock:
        for repo in list(_instances):
            try:
                if repo._connection is not None:
                    with contextlib.suppress(Exception):
                        repo._connection.close()
                    repo._connection = None
                    repo._attached_dbs.clear()
                    closed += 1
            except Exception:
                pass
    if closed:
        logger.debug("release_all_db_connections: %d connexion(s) fermée(s)", closed)
    return closed


class DuckDBRepository(
    MatchQueriesMixin,
    RosterLoaderMixin,
    MaterializedViewsMixin,
    AntagonistsMixin,
    MedalsMixin,
    EventsMixin,
    ArchivesMixin,
):
    """
    Repository utilisant DuckDB natif exclusivement.
    (Repository using native DuckDB exclusively)

    Lit depuis :
    - metadata.duckdb : Tables de référence
    - stats.duckdb : Données du joueur (matchs, médailles, etc.)
    """

    def __init__(  # noqa: PLR0913
        self,
        player_db_path: str | Path,
        xuid: str,
        *,
        metadata_db_path: str | Path | None = None,
        shared_db_path: str | Path | None = None,
        gamertag: str | None = None,
        read_only: bool = True,
        memory_limit: str = "512MB",
    ) -> None:
        """
        Initialise le repository DuckDB.
        (Initialize DuckDB repository)

        Args:
            player_db_path: Chemin vers stats.duckdb du joueur
            xuid: XUID du joueur
            metadata_db_path: Chemin vers metadata.duckdb (auto-détecté si None)
            shared_db_path: Chemin vers shared_matches.duckdb (auto-détecté si None)
            gamertag: Gamertag du joueur (optionnel, pour logging)
            read_only: Si True, connexion en lecture seule
            memory_limit: Limite mémoire DuckDB
        """
        self._player_db_path = Path(player_db_path)
        self._xuid = xuid
        self._gamertag = gamertag
        self._read_only = read_only
        self._memory_limit = memory_limit

        # Auto-détection du chemin metadata.duckdb
        if metadata_db_path is None:
            # Cherche dans data/warehouse/metadata.duckdb
            data_dir = self._player_db_path.parent.parent.parent
            self._metadata_db_path = data_dir / "warehouse" / "metadata.duckdb"
        else:
            self._metadata_db_path = Path(metadata_db_path)

        # Auto-détection du chemin shared_matches.duckdb (v5)
        if shared_db_path is None:
            data_dir = self._player_db_path.parent.parent.parent
            self._shared_db_path = data_dir / "warehouse" / "shared_matches.duckdb"
        else:
            self._shared_db_path = Path(shared_db_path)

        # Connexion DuckDB (lazy loading)
        self._connection: duckdb.DuckDBPyConnection | None = None
        self._attached_dbs: set[str] = set()

        # Enregistrement dans le registre global (weakref)
        with _instances_lock:
            _instances.add(self)

        # Cache de schéma (v5.1 perf) — évite les requêtes information_schema répétées
        self._schema_cache: dict[str, bool] = {}
        self._table_cache: dict[str, bool] = {}
        self._view_cache: dict[str, bool] = {}

        # Cache des résolutions coûteuses (v5.1 perf — 1bis.3)
        # Le schéma metadata et les tables locales ne changent pas en session.
        self._metadata_resolution_cache: tuple[str, str, str, str] | None = None
        self._mmr_fallback_cache: tuple[str, str, str] | None = None

    @property
    def xuid(self) -> str:
        """XUID du joueur principal."""
        return self._xuid

    @property
    def db_path(self) -> str:
        """Chemin vers la base de données joueur."""
        return str(self._player_db_path)

    @property
    def has_shared(self) -> bool:
        """Indique si shared_matches.duckdb est attaché et disponible."""
        return "shared" in self._attached_dbs

    def _has_shared_table(self, table_name: str) -> bool:
        """Vérifie si une table existe dans shared_matches.duckdb (avec cache).

        Args:
            table_name: Nom de la table à vérifier (sans préfixe 'shared.').

        Returns:
            True si la table existe dans le schema shared.
        """
        cache_key = f"shared_table:{table_name}"
        if cache_key in self._table_cache:
            return self._table_cache[cache_key]

        conn = self._get_connection()
        if not self.has_shared:
            self._table_cache[cache_key] = False
            return False
        try:
            result = conn.execute(
                "SELECT table_name FROM information_schema.tables "
                "WHERE table_catalog = 'shared' AND table_name = ?",
                [table_name],
            ).fetchone()
            exists = result is not None
            self._table_cache[cache_key] = exists
            return exists
        except Exception:
            self._table_cache[cache_key] = False
            return False

    def _has_shared_view(self, view_name: str) -> bool:
        """Vérifie si une vue existe dans shared_matches.duckdb (avec cache).

        Utilise ``duckdb_views()`` qui fonctionne bien avec les bases attachées,
        contrairement à ``catalog.information_schema.views``.

        Args:
            view_name: Nom de la vue à vérifier (sans préfixe 'shared.').

        Returns:
            True si la vue existe dans le catalog shared.
        """
        cache_key = f"shared_view:{view_name}"
        if cache_key in self._view_cache:
            return self._view_cache[cache_key]

        conn = self._get_connection()
        if not self.has_shared:
            self._view_cache[cache_key] = False
            return False
        try:
            result = conn.execute(
                "SELECT view_name FROM duckdb_views() "
                "WHERE database_name = 'shared' AND view_name = ?",
                [view_name],
            ).fetchone()
            exists = result is not None
            self._view_cache[cache_key] = exists
            return exists
        except Exception:
            self._view_cache[cache_key] = False
            return False

    def _has_table_cached(self, conn: duckdb.DuckDBPyConnection, table_name: str) -> bool:
        """Vérifie si une table existe dans le schema main (avec cache).

        Args:
            conn: Connexion DuckDB.
            table_name: Nom de la table.

        Returns:
            True si la table existe.
        """
        cache_key = f"main_table:{table_name}"
        if cache_key in self._table_cache:
            return self._table_cache[cache_key]

        try:
            result = conn.execute(
                "SELECT COUNT(*) FROM information_schema.tables "
                "WHERE table_schema = 'main' AND table_name = ?",
                [table_name],
            ).fetchone()
            exists = bool(result and result[0] > 0)
            self._table_cache[cache_key] = exists
            return exists
        except Exception:
            self._table_cache[cache_key] = False
            return False

    def _get_connection(self) -> duckdb.DuckDBPyConnection:  # noqa: C901, PLR0912
        """
        Retourne une connexion DuckDB vers la DB joueur.
        (Returns DuckDB connection to player DB)

        Intègre un retry automatique si la DB est temporairement ouverte
        par un autre composant avec un mode d'accès différent (ex. MediaIndexer
        en read_write pendant que le repo est read_only).
        """
        if self._connection is None:
            if not self._player_db_path.exists():
                raise FileNotFoundError(
                    f"Base de données joueur non trouvée: {self._player_db_path}"
                )

            # Connexion à la DB joueur — retry si conflit de configuration
            max_retries = 3
            for attempt in range(max_retries):
                try:
                    self._connection = duckdb.connect(
                        str(self._player_db_path),
                        read_only=self._read_only,
                    )
                    break
                except Exception as e:
                    if "different configuration" in str(e).lower() and attempt < max_retries - 1:
                        logger.debug(
                            "DB %s occupée par un autre mode d'accès, retry %d/%d",
                            self._player_db_path.name,
                            attempt + 1,
                            max_retries,
                        )
                        time.sleep(1.0)
                        continue
                    raise

            # Configuration
            self._connection.execute(f"SET memory_limit = '{self._memory_limit}'")
            self._connection.execute("SET enable_object_cache = true")

            # Vérifier les DBs déjà attachées en consultant DuckDB directement
            # (au cas où la connexion est réutilisée par plusieurs instances de repo)
            try:
                attached_dbs = self._connection.execute(
                    "SELECT database_name FROM duckdb_databases()"
                ).fetchall()
                attached_db_names = {db[0].lower() for db in attached_dbs if db[0]}
            except Exception:
                attached_db_names = set()

            # Attacher la DB metadata si elle existe et pas déjà attachée
            if self._metadata_db_path.exists() and "meta" not in attached_db_names:
                try:
                    self._connection.execute(
                        f"ATTACH '{self._metadata_db_path}' AS meta (READ_ONLY)"
                    )
                    self._attached_dbs.add("meta")
                    logger.debug(f"Metadata DB attachée: {self._metadata_db_path}")
                except Exception as e:
                    err_str = str(e)
                    # DuckDB 1.4+ : le même fichier ne peut être attaché qu'à une seule connexion.
                    # Si déjà ouvert ailleurs (autre DuckDBRepository, MetadataResolver), ignorer silencieusement.
                    if (
                        "already exists" in err_str.lower()
                        or "unique file handle conflict" in err_str.lower()
                        or "already attached" in err_str.lower()
                    ):
                        self._attached_dbs.add("meta")  # Marquer comme attaché même si erreur
                        logger.debug(
                            "metadata.duckdb déjà ouvert par une autre connexion, "
                            "résolution métadonnées désactivée pour cette instance"
                        )
                    else:
                        logger.warning(f"Impossible d'attacher metadata.duckdb: {e}")
            elif "meta" in attached_db_names:
                self._attached_dbs.add("meta")  # Synchroniser le tracker local

            # Attacher shared_matches.duckdb en lecture seule (v5)
            # Skip si le mode sync est actif : le DuckDBSyncEngine détient shared_matches
            # en R/W (connexion directe) et DuckDB interdit deux noms différents pour le
            # même fichier dans un même processus.
            if self._shared_db_path.exists() and "shared" not in attached_db_names:
                if _sync_mode.is_set():
                    logger.debug(
                        "Sync en cours : ATTACH shared_matches.duckdb différé pour cette instance"
                    )
                else:
                    try:
                        self._connection.execute(
                            f"ATTACH '{self._shared_db_path}' AS shared (READ_ONLY)"
                        )
                        self._attached_dbs.add("shared")
                        logger.debug(f"Shared matches DB attachée: {self._shared_db_path}")
                    except Exception as e:
                        err_str = str(e)
                        if (
                            "already exists" in err_str.lower()
                            or "unique file handle conflict" in err_str.lower()
                            or "already attached" in err_str.lower()
                        ):
                            # Ne PAS marquer comme attaché : l'ATTACH a échoué, shared.* n'est
                            # pas accessible depuis cette connexion. L'appelant gérera l'absence.
                            logger.debug(
                                "shared_matches.duckdb conflit de handle lors de l'ATTACH "
                                "(sync probablement en cours) — shared non disponible pour cette instance"
                            )
                        else:
                            logger.warning(f"Impossible d'attacher shared_matches.duckdb: {e}")
            elif "shared" in attached_db_names:
                self._attached_dbs.add("shared")  # Synchroniser le tracker local

        return self._connection

    def _has_column(
        self, conn: duckdb.DuckDBPyConnection, table_name: str, column_name: str
    ) -> bool:
        """Retourne True si une colonne existe dans une table (avec cache).

        Utile pour supporter des schémas historiques (colonnes ajoutées en v4).
        """
        cache_key = f"{table_name}.{column_name}"
        if cache_key in self._schema_cache:
            return self._schema_cache[cache_key]

        try:
            exists = (
                conn.execute(
                    "SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?",
                    (table_name, column_name),
                ).fetchone()[0]
                > 0
            )
            self._schema_cache[cache_key] = exists
            return exists
        except Exception:
            self._schema_cache[cache_key] = False
            return False

    def _has_shared_mp_column(self, conn: duckdb.DuckDBPyConnection, column_name: str) -> bool:
        """Vérifie si match_participants (shared) possède une colonne (avec cache).

        Consulte le catalog ``shared`` attaché en priorité, puis fallback
        sur le catalog principal (utile en tests unitaires).
        """
        cache_key = f"shared_mp.{column_name}"
        if cache_key in self._schema_cache:
            return self._schema_cache[cache_key]

        for catalog in ("shared", "main"):
            try:
                result = conn.execute(
                    f"SELECT COUNT(*) FROM {catalog}.information_schema.columns "
                    "WHERE table_name = 'match_participants' AND column_name = ?",
                    (column_name,),
                ).fetchone()
                if result and result[0] > 0:
                    self._schema_cache[cache_key] = True
                    return True
            except Exception:
                continue
        self._schema_cache[cache_key] = False
        return False

    def _select_optional_column(  # noqa: PLR0913
        self,
        conn: duckdb.DuckDBPyConnection,
        *,
        table_name: str,
        table_alias: str,
        column_name: str,
        output_name: str | None = None,
    ) -> str:
        """Construit une expression SELECT tolérante si la colonne manque."""

        out = output_name or column_name
        if self._has_column(conn, table_name, column_name):
            return f"{table_alias}.{column_name} AS {out}"
        return f"NULL AS {out}"

    def _build_metadata_resolution(
        self, conn: duckdb.DuckDBPyConnection
    ) -> tuple[str, str, str, str]:
        """
        Construit les expressions SQL et les jointures pour résoudre les métadonnées.

        Le résultat est caché en instance car le schéma metadata ne change pas en session.

        Returns:
            Tuple (metadata_joins, map_name_expr, playlist_name_expr, pair_name_expr)
        """
        # Cache d'instance (v5.1 perf — 1bis.3)
        if self._metadata_resolution_cache is not None:
            return self._metadata_resolution_cache
        metadata_joins = ""
        map_name_expr = "match_stats.map_name"
        playlist_name_expr = "match_stats.playlist_name"
        pair_name_expr = "match_stats.pair_name"

        if "meta" not in self._attached_dbs:
            logger.debug("Metadata DB non attachée, pas de résolution des métadonnées")
            return metadata_joins, map_name_expr, playlist_name_expr, pair_name_expr

        try:
            # Vérifier si les tables de métadonnées existent
            # Utiliser une seule requête pour toutes les tables pour plus d'efficacité
            tables_check = conn.execute(
                "SELECT table_name FROM information_schema.tables "
                "WHERE table_schema = 'meta' AND table_name IN ('maps', 'playlists', 'map_mode_pairs', 'playlist_map_mode_pairs')"
            ).fetchall()
            existing_tables = {row[0] for row in tables_check}

            has_maps = "maps" in existing_tables
            has_playlists = "playlists" in existing_tables
            has_pairs_map_mode = "map_mode_pairs" in existing_tables
            has_pairs_playlist = "playlist_map_mode_pairs" in existing_tables
            has_pairs = has_pairs_map_mode or has_pairs_playlist
            pair_table_name = (
                "map_mode_pairs"
                if has_pairs_map_mode
                else ("playlist_map_mode_pairs" if has_pairs_playlist else None)
            )

            logger.debug(
                f"Résolution métadonnées: maps={has_maps}, playlists={has_playlists}, "
                f"pairs={has_pairs} (table={pair_table_name})"
            )

            if has_maps:
                metadata_joins += (
                    " LEFT JOIN meta.maps m_meta ON match_stats.map_id = m_meta.asset_id"
                )
                # Préférer le nom résolu depuis les métadonnées, même si map_name contient déjà une valeur
                # (car map_name peut contenir un UUID si la résolution a échoué lors de la sync)
                map_name_expr = "COALESCE(m_meta.public_name, match_stats.map_name)"

            if has_playlists:
                metadata_joins += (
                    " LEFT JOIN meta.playlists p_meta ON match_stats.playlist_id = p_meta.asset_id"
                )
                # Préférer le nom résolu depuis les métadonnées
                playlist_name_expr = "COALESCE(p_meta.public_name, match_stats.playlist_name)"

            if has_pairs and pair_table_name:
                metadata_joins += f" LEFT JOIN meta.{pair_table_name} pair_meta ON match_stats.pair_id = pair_meta.asset_id"
                # Préférer le nom résolu depuis les métadonnées
                pair_name_expr = "COALESCE(pair_meta.public_name, match_stats.pair_name)"
        except Exception as e:
            # Si erreur, utiliser les valeurs directes (déjà définies par défaut)
            logger.warning(f"Erreur lors de la construction des jointures métadonnées: {e}")

        result = (metadata_joins, map_name_expr, playlist_name_expr, pair_name_expr)
        self._metadata_resolution_cache = result
        return result

    def _build_mmr_fallback(self, conn) -> tuple[str, str, str]:
        """Construit la jointure et les expressions pour le fallback MMR.

        Si player_match_stats existe, utilise COALESCE pour récupérer les MMR
        depuis cette table si match_stats a des valeurs NULL.

        Le résultat est caché en instance car les tables locales ne changent pas en session.

        Returns:
            Tuple (pms_join, team_mmr_expr, enemy_mmr_expr)
        """
        # Cache d'instance (v5.1 perf — 1bis.3)
        if self._mmr_fallback_cache is not None:
            return self._mmr_fallback_cache
        pms_join = ""
        team_mmr_expr = "match_stats.team_mmr"
        enemy_mmr_expr = "match_stats.enemy_mmr"

        try:
            pms_tables = conn.execute(
                "SELECT table_name FROM information_schema.tables "
                "WHERE table_schema='main' AND table_name='player_match_stats'"
            ).fetchall()
            if pms_tables:
                pms_join = (
                    " LEFT JOIN player_match_stats pms ON match_stats.match_id = pms.match_id"
                )
                team_mmr_expr = "COALESCE(match_stats.team_mmr, pms.team_mmr)"
                enemy_mmr_expr = "COALESCE(match_stats.enemy_mmr, pms.enemy_mmr)"
        except Exception:
            pass

        result = (pms_join, team_mmr_expr, enemy_mmr_expr)
        self._mmr_fallback_cache = result
        return result

    # =========================================================================
    # Coéquipiers
    # =========================================================================

    def list_top_teammates(
        self,
        limit: int = 20,
    ) -> list[tuple[str, int]]:
        """Liste les coéquipiers les plus fréquents.

        Calcule dynamiquement depuis shared.match_participants :
        compte le nombre de matchs en commun avec chaque autre joueur.
        """
        conn = self._get_connection()

        # Utiliser shared.match_participants (v5) pour calculer dynamiquement
        if self._has_shared_table("match_participants"):
            try:
                result = conn.execute(
                    """
                    SELECT mp2.xuid AS teammate_xuid,
                           COUNT(DISTINCT mp2.match_id) AS matches_together
                    FROM shared.match_participants mp1
                    JOIN shared.match_participants mp2
                      ON mp1.match_id = mp2.match_id
                     AND mp1.xuid != mp2.xuid
                     AND mp1.team_id = mp2.team_id
                    WHERE mp1.xuid = ?
                    GROUP BY mp2.xuid
                    ORDER BY matches_together DESC
                    LIMIT ?
                    """,
                    [self._xuid, limit],
                )
                return [(row[0], row[1]) for row in result.fetchall()]
            except Exception:
                pass
        return []

    # =========================================================================
    # Métadonnées
    # =========================================================================

    def get_sync_metadata(self) -> dict[str, Any]:
        """Récupère les métadonnées de synchronisation."""
        conn = self._get_connection()

        # Essayer de lire depuis meta.sync_meta si attaché
        if "meta" in self._attached_dbs:
            try:
                result = conn.execute(
                    "SELECT last_sync_at FROM meta.sync_meta WHERE xuid = ?",
                    [self._xuid],
                ).fetchone()
                last_sync = result[0] if result else None
            except Exception:
                last_sync = None
        else:
            last_sync = None

        return {
            "last_sync_at": last_sync,
            "last_match_time": None,
            "total_matches": self.get_match_count(),
            "player_xuid": self._xuid,
            "storage_type": "duckdb",
        }

    # =========================================================================
    # Méthodes de diagnostic
    # =========================================================================

    def get_storage_info(self) -> dict[str, Any]:
        """Retourne des informations sur le stockage (local + shared)."""
        conn = self._get_connection()

        # Taille des tables locales
        tables_info = {}
        for table in ["match_stats", "medals_earned", "antagonists"]:
            try:
                count = conn.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
                tables_info[table] = count
            except Exception:
                tables_info[table] = 0

        # Counts shared
        shared_info = {}
        if self.has_shared:
            for table, xuid_col in [
                ("match_participants", "xuid"),
                ("medals_earned", "xuid"),
                ("highlight_events", None),
                ("match_registry", None),
            ]:
                try:
                    if xuid_col:
                        count = conn.execute(
                            f"SELECT COUNT(*) FROM shared.{table} WHERE {xuid_col} = ?",
                            [self._xuid],
                        ).fetchone()[0]
                    else:
                        count = conn.execute(f"SELECT COUNT(*) FROM shared.{table}").fetchone()[0]
                    shared_info[f"shared_{table}"] = count
                except Exception:
                    shared_info[f"shared_{table}"] = 0

        # Taille du fichier
        file_size_mb = 0
        if self._player_db_path.exists():
            file_size_mb = self._player_db_path.stat().st_size / (1024 * 1024)

        shared_size_mb = 0
        if self._shared_db_path.exists():
            shared_size_mb = self._shared_db_path.stat().st_size / (1024 * 1024)

        return {
            "type": "duckdb",
            "player_db_path": str(self._player_db_path),
            "metadata_db_path": str(self._metadata_db_path),
            "shared_db_path": str(self._shared_db_path),
            "xuid": self._xuid,
            "gamertag": self._gamertag,
            "file_size_mb": round(file_size_mb, 2),
            "shared_size_mb": round(shared_size_mb, 2),
            "tables": tables_info,
            "shared_tables": shared_info,
            "has_metadata": "meta" in self._attached_dbs,
            "has_shared": "shared" in self._attached_dbs,
        }

    def is_hybrid_available(self) -> bool:
        """Vérifie si les données sont disponibles."""
        return self._player_db_path.exists() and self.get_match_count() > 0

    # =========================================================================
    # Requêtes avancées
    # =========================================================================

    def query(self, sql: str, params: list | None = None) -> list[dict[str, Any]]:
        """
        Exécute une requête SQL arbitraire.
        (Execute arbitrary SQL query)
        """
        conn = self._get_connection()
        result = conn.execute(sql, params) if params else conn.execute(sql)
        columns = [desc[0] for desc in result.description]
        return [dict(zip(columns, row, strict=False)) for row in result.fetchall()]

    def query_df(self, sql: str, params: list | None = None):
        """
        Exécute une requête SQL et retourne un DataFrame Polars.
        (Execute SQL query and return Polars DataFrame)
        """
        conn = self._get_connection()
        result = conn.execute(sql, params) if params else conn.execute(sql)
        return result_to_polars(result)

    # =========================================================================
    # Gestion de la connexion
    # =========================================================================

    def close(self) -> None:
        """Ferme la connexion DuckDB et vide les caches."""
        if self._connection is not None:
            self._connection.close()
            self._connection = None
            self._attached_dbs.clear()
            self._schema_cache.clear()
            self._table_cache.clear()
            self._view_cache.clear()
            # Invalider les caches de résolution (v5.1 perf — 1bis.3)
            self._metadata_resolution_cache = None
            self._mmr_fallback_cache = None

    def __enter__(self) -> DuckDBRepository:
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        self.close()

    # =========================================================================
    # Citations (Sprint Citations)
    # =========================================================================

    def insert_citation(
        self,
        match_id: str,
        citation_name_norm: str,
        value: int,
    ) -> None:
        """Insère ou met à jour une citation pour un match.

        Args:
            match_id: Identifiant du match.
            citation_name_norm: Nom normalisé de la citation.
            value: Valeur de la citation.
        """
        conn = self._get_connection()
        conn.execute(
            "INSERT OR REPLACE INTO match_citations "
            "(match_id, citation_name_norm, value) VALUES (?, ?, ?)",
            [match_id, citation_name_norm, value],
        )

    def load_personal_score_awards_as_polars(
        self,
        *,
        match_id: str | None = None,
        match_ids: list[str] | None = None,
        category: str | None = None,
        limit: int | None = None,
    ):
        """Charge les PersonalScoreAwards en DataFrame Polars.

        Sprint 8.2 - Permet de visualiser la participation au match :
        kills, assists, objectifs, véhicules, pénalités.

        Args:
            match_id: Filtrer par un match spécifique.
            match_ids: Filtrer par une liste de matchs.
            category: Filtrer par catégorie (kill, assist, objective, etc.).
            limit: Limite du nombre de résultats.

        Returns:
            DataFrame Polars avec colonnes :
            match_id, xuid, award_name, award_category, award_count, award_score.
        """
        conn = self._get_connection()

        where_clauses = []
        params = []

        if match_id:
            where_clauses.append("match_id = ?")
            params.append(match_id)
        elif match_ids:
            placeholders = ", ".join(["?" for _ in match_ids])
            where_clauses.append(f"match_id IN ({placeholders})")
            params.extend(match_ids)

        if category:
            where_clauses.append("award_category = ?")
            params.append(category)

        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"
        limit_sql = f"LIMIT {int(limit)}" if limit else ""

        sql = f"""
            SELECT
                match_id,
                xuid,
                award_name,
                award_category,
                award_count,
                award_score,
                created_at
            FROM personal_score_awards
            WHERE {where_sql}
            ORDER BY award_score DESC
            {limit_sql}
        """

        try:
            result = conn.execute(sql, params) if params else conn.execute(sql)
            return result_to_polars(result)
        except Exception as e:
            logger.warning(f"Erreur chargement personal_score_awards Polars: {e}")
            import polars as pl

            return pl.DataFrame()

    def has_personal_score_awards(self) -> bool:
        """Vérifie si des PersonalScoreAwards existent dans la DB."""
        conn = self._get_connection()
        try:
            result = conn.execute("SELECT 1 FROM personal_score_awards LIMIT 1").fetchone()
            return result is not None
        except Exception:
            return False

    # =========================================================================
    # Sprint Gamertag Roster Fix : Helpers et résolution
    # =========================================================================

    def _has_table(self, table_name: str) -> bool:
        """Vérifie si une table existe dans la DB."""
        conn = self._get_connection()
        try:
            result = conn.execute(
                "SELECT table_name FROM information_schema.tables "
                "WHERE table_schema = 'main' AND table_name = ?",
                [table_name],
            ).fetchone()
            return result is not None
        except Exception:
            return False

    # =========================================================================
    # Méthodes legacy-compat (ajoutées pour migration src/db/)
    # =========================================================================

    def list_other_player_xuids(self, limit: int = 500) -> list[str]:
        """Liste les XUIDs des autres joueurs rencontrés.

        V5 : Utilise shared.match_participants si disponible (roster complet).
        Complète avec les sources locales.

        Args:
            limit: Nombre max de XUIDs à retourner.

        Returns:
            Liste de XUIDs uniques (hors le joueur principal).
        """
        conn = self._get_connection()
        xuids: set[str] = set()

        try:
            # V5 : shared.match_participants (tous les joueurs de tous les matchs)
            if self._has_shared_table("match_participants"):
                rows = conn.execute(
                    """
                    SELECT DISTINCT p2.xuid
                    FROM shared.match_participants p1
                    INNER JOIN shared.match_participants p2 ON p1.match_id = p2.match_id
                    WHERE p1.xuid = ? AND p2.xuid != ?
                    LIMIT ?
                    """,
                    [self._xuid, self._xuid, limit],
                ).fetchall()
                xuids.update(str(row[0]) for row in rows if row[0])

            # Depuis highlight_events locale
            if self._has_table("highlight_events"):
                rows = conn.execute(
                    """
                    SELECT DISTINCT xuid
                    FROM highlight_events
                    WHERE xuid IS NOT NULL AND xuid != ?
                    LIMIT ?
                    """,
                    [self._xuid, limit],
                ).fetchall()
                xuids.update(str(row[0]) for row in rows if row[0])

            # Depuis match_participants locale
            if self._has_table("match_participants"):
                rows = conn.execute(
                    """
                    SELECT DISTINCT xuid
                    FROM match_participants
                    WHERE xuid IS NOT NULL AND xuid != ?
                    LIMIT ?
                    """,
                    [self._xuid, limit],
                ).fetchall()
                xuids.update(str(row[0]) for row in rows if row[0])

            # Depuis antagonists
            if self._has_table("antagonists"):
                rows = conn.execute(
                    """
                    SELECT DISTINCT opponent_xuid
                    FROM antagonists
                    WHERE opponent_xuid IS NOT NULL
                    LIMIT ?
                    """,
                    [limit],
                ).fetchall()
                xuids.update(str(row[0]) for row in rows if row[0])

            return list(xuids)[:limit]
        except Exception as e:
            logger.debug(f"Erreur list_other_player_xuids: {e}")
            return []

    def get_match_session_info(self, match_id: str) -> dict[str, Any] | None:
        """Retourne les infos de session pour un match.

        Cherche dans match_stats (local) puis player_match_enrichment (local).
        session_id est une donnée d'enrichissement joueur qui reste locale.

        Args:
            match_id: ID du match.

        Returns:
            Dict avec session_id, label, etc. ou None.
        """
        if not match_id:
            return None

        conn = self._get_connection()

        # 1. match_stats (local)
        try:
            row = conn.execute(
                "SELECT session_id FROM match_stats WHERE match_id = ?",
                [match_id],
            ).fetchone()
            if row and row[0]:
                return {"session_id": row[0]}
        except Exception:
            pass

        # 2. player_match_enrichment (local)
        try:
            row = conn.execute(
                "SELECT session_id FROM player_match_enrichment WHERE match_id = ?",
                [match_id],
            ).fetchone()
            if row and row[0]:
                return {"session_id": row[0]}
        except Exception:
            pass

        return None

    def has_table(self, table_name: str) -> bool:
        """Vérifie si une table existe (alias public de _has_table).

        Args:
            table_name: Nom de la table.

        Returns:
            True si la table existe.
        """
        return self._has_table(table_name)
