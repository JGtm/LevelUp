"""
Repository DuckDB natif pour les données joueur.
(Native DuckDB repository for player data)

HOW IT WORKS:
Ce repository utilise exclusivement DuckDB :
1. data/warehouse/metadata.duckdb : Référentiels (playlists, maps, médailles)
2. data/players/{gamertag}/stats.duckdb : Données joueur (matchs, médailles, etc.)
3. data/players/{gamertag}/archive/*.parquet : Archives (cold storage)

Les jointures entre les deux DBs sont faites via ATTACH.
Les archives Parquet peuvent être lues via ``load_matches_from_archives()``.
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

from src.data.repositories._archives_repo import ArchivesMixin
from src.data.repositories._arrow_bridge import result_to_polars
from src.data.repositories._awards_repo import AwardsMixin
from src.data.repositories._career_encounters_repo import EncounterCareerMixin
from src.data.repositories._career_repo import CareerMixin
from src.data.repositories._diagnostic_repo import DiagnosticMixin
from src.data.repositories._events_repo import EventsMixin
from src.data.repositories._killer_victim_repo import KillerVictimMixin
from src.data.repositories._legacy_compat import LegacyCompatMixin
from src.data.repositories._match_queries import MatchQueriesMixin
from src.data.repositories._match_relations import MatchRelationsMixin
from src.data.repositories._materialized_views import MaterializedViewsMixin
from src.data.repositories._medals_repo import MedalsMixin
from src.data.repositories._media_repo import MediaLibraryMixin
from src.data.repositories._metadata_resolution import MetadataResolutionMixin
from src.data.repositories._roster_loader import RosterLoaderMixin
from src.data.repositories._schema_introspection import SchemaIntrospectionMixin
from src.data.repositories._weapon_kills_repo import WeaponKillsMixin
from src.data.repositories._write_lease import (
    db_write_lease,  # noqa: F401 — re-export pour les consommateurs
    wait_for_write_leases_cleared,
)

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


def _release_repos(filter_fn=None) -> int:
    """Ferme les connexions des repos correspondant au filtre (DRY helper).

    Args:
        filter_fn: Prédicat ``(repo) -> bool``. Si None, ferme tout.

    Returns:
        Nombre de connexions fermées.
    """
    closed = 0
    with _instances_lock:
        for repo in list(_instances):
            try:
                _lock = getattr(repo, "_connection_lock", None)
                _ctx = _lock if _lock is not None else contextlib.nullcontext()
                with _ctx:
                    if repo._connection is not None and (filter_fn is None or filter_fn(repo)):
                        with contextlib.suppress(Exception):
                            repo._connection.close()
                        repo._connection = None
                        repo._attached_dbs.clear()
                        closed += 1
            except Exception:
                pass
    return closed


def release_db_connections(db_path: str | Path) -> int:
    """Ferme toutes les connexions DuckDBRepository ouvertes vers *db_path*.

    Les repositories concernés se reconnecteront paresseusement lors du
    prochain appel à ``_get_connection()``.

    Returns:
        Nombre de connexions fermées.
    """
    target = str(Path(db_path).resolve())
    closed = _release_repos(lambda repo: str(repo._player_db_path.resolve()) == target)
    if closed:
        logger.debug(
            "release_db_connections: %d connexion(s) fermée(s) pour %s",
            closed,
            db_path,
        )
    return closed


def release_all_db_connections() -> int:
    """Ferme toutes les connexions DuckDBRepository actives (tous joueurs).

    Nécessaire avant d'ouvrir shared_matches.duckdb directement (ex. sync),
    car DuckDB interdit qu'un même fichier soit utilisé sous deux noms différents
    dans le même processus.

    Returns:
        Nombre de connexions fermées.
    """
    closed = _release_repos()
    if closed:
        logger.debug("release_all_db_connections: %d connexion(s) fermée(s)", closed)
    return closed


class DuckDBRepository(
    MatchQueriesMixin,
    RosterLoaderMixin,
    MatchRelationsMixin,
    MaterializedViewsMixin,
    KillerVictimMixin,
    MedalsMixin,
    EventsMixin,
    ArchivesMixin,
    SchemaIntrospectionMixin,
    MetadataResolutionMixin,
    CareerMixin,
    EncounterCareerMixin,
    DiagnosticMixin,
    AwardsMixin,
    MediaLibraryMixin,
    LegacyCompatMixin,
    WeaponKillsMixin,
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
            data_dir = self._player_db_path.parent.parent.parent
            self._metadata_db_path = data_dir / "warehouse" / "metadata.duckdb"
        else:
            self._metadata_db_path = Path(metadata_db_path)

        # Auto-détection du chemin shared_matches.duckdb (v5/v6)
        if shared_db_path is None:
            from src.utils.paths import get_shared_matches_path, get_shared_matches_path_from_player

            detected = get_shared_matches_path_from_player(self._player_db_path)
            if detected is not None:
                self._shared_db_path = detected
            else:
                # Fallback : chemin canonique dérivé du player, même si le fichier n'existe pas encore
                data_dir = self._player_db_path.parent.parent.parent
                self._shared_db_path = data_dir / "warehouse" / get_shared_matches_path().name
        else:
            self._shared_db_path = Path(shared_db_path)

        # Connexion DuckDB (lazy loading)
        self._connection: duckdb.DuckDBPyConnection | None = None
        self._attached_dbs: set[str] = set()
        # Verrou par instance pour protéger l'initialisation contre les races
        # (ex. release_all_db_connections appelé depuis thread media-indexer
        # pendant que _get_connection est en cours dans le thread Streamlit).
        self._connection_lock = threading.Lock()

        # Enregistrement dans le registre global (weakref)
        with _instances_lock:
            _instances.add(self)

        # Cache de schéma (v5.1 perf) — évite les requêtes information_schema répétées
        self._schema_cache: dict[str, bool] = {}
        self._table_cache: dict[str, bool] = {}
        self._view_cache: dict[str, bool] = {}

        # Cache des résolutions coûteuses (v5.1 perf — 1bis.3)
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

    def _get_connection(self) -> duckdb.DuckDBPyConnection:  # noqa: C901, PLR0912
        """
        Retourne une connexion DuckDB vers la DB joueur.
        (Returns DuckDB connection to player DB)

        Intègre un retry automatique si la DB est temporairement ouverte
        par un autre composant avec un mode d'accès différent (ex. MediaIndexer
        en read_write pendant que le repo est read_only), ou verrouillée au
        niveau OS sur Windows.
        Thread-safe via ``_connection_lock`` (pattern DCL) pour éviter la race
        condition avec ``release_all_db_connections`` (thread media-indexer).
        """
        # Fast-path sans verrou (optimiste)
        if self._connection is not None:
            # Lazy re-ATTACH shared si désactivé pendant sync_mode
            if (
                "shared" not in self._attached_dbs
                and self._shared_db_path.exists()
                and not _sync_mode.is_set()
            ):
                self._try_reattach_shared()
            return self._connection
        with self._connection_lock:
            # Double-check sous verrou (pattern DCL)
            if self._connection is not None:
                return self._connection
            if not self._player_db_path.exists():
                raise FileNotFoundError(
                    f"Base de données joueur non trouvée: {self._player_db_path}"
                )

            # Attendre que le MediaIndexer (ou tout autre writer) ait libéré sa connexion
            # read_write avant d'ouvrir en read_only — évite le conflit DuckDB
            # « different configuration than existing connections ».
            wait_for_write_leases_cleared(self._player_db_path)

            # Connexion à la DB joueur — retry si conflit de configuration ou lock OS (Windows)
            max_retries = 3
            for attempt in range(max_retries):
                try:
                    self._connection = duckdb.connect(
                        str(self._player_db_path),
                        read_only=self._read_only,
                    )
                    break
                except Exception as e:
                    err_str = str(e).lower()
                    is_transient = (
                        "different configuration" in err_str
                        or "cannot open file" in err_str
                        or "utilisé par un autre processus" in err_str
                        or "process cannot access" in err_str
                    )
                    if is_transient and attempt < max_retries - 1:
                        logger.debug(
                            "DB %s temporairement verrouillée, retry %d/%d",
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
            try:
                attached_dbs = self._connection.execute(
                    "SELECT database_name FROM duckdb_databases()"
                ).fetchall()
                attached_db_names = {db[0].lower() for db in attached_dbs if db[0]}
            except Exception:
                attached_db_names = set()

            self._attach_metadata(attached_db_names)
            self._attach_shared(attached_db_names)

            return self._connection

    def _attach_metadata(self, attached_db_names: set[str]) -> None:
        """Attache metadata.duckdb si disponible et pas déjà attaché."""
        if self._metadata_db_path.exists() and "meta" not in attached_db_names:
            try:
                self._connection.execute(f"ATTACH '{self._metadata_db_path}' AS meta (READ_ONLY)")
                self._attached_dbs.add("meta")
                logger.debug("Metadata DB attachée: %s", self._metadata_db_path)
            except Exception as e:
                err_str = str(e).lower()
                if any(
                    k in err_str
                    for k in ("already exists", "unique file handle conflict", "already attached")
                ):
                    self._attached_dbs.add("meta")
                    logger.debug(
                        "metadata.duckdb déjà ouvert par une autre connexion, "
                        "résolution métadonnées désactivée pour cette instance"
                    )
                else:
                    logger.warning("Impossible d'attacher metadata.duckdb: %s", e)
        elif "meta" in attached_db_names:
            self._attached_dbs.add("meta")

    def _attach_shared(self, attached_db_names: set[str]) -> None:
        """Attache shared_matches.duckdb en lecture seule si possible."""
        if self._shared_db_path.exists() and "shared" not in attached_db_names:
            if _sync_mode.is_set():
                logger.debug(
                    "Sync en cours : ATTACH shared_matches.duckdb différé pour cette instance"
                )
                return
            try:
                self._connection.execute(f"ATTACH '{self._shared_db_path}' AS shared (READ_ONLY)")
                self._attached_dbs.add("shared")
                logger.debug("Shared matches DB attachée: %s", self._shared_db_path)
            except Exception as e:
                err_str = str(e).lower()
                if any(
                    k in err_str
                    for k in ("already exists", "unique file handle conflict", "already attached")
                ):
                    logger.debug(
                        "shared_matches.duckdb conflit de handle lors de l'ATTACH "
                        "(sync probablement en cours) — shared non disponible"
                    )
                else:
                    logger.warning("Impossible d'attacher shared_matches.duckdb: %s", e)
        elif "shared" in attached_db_names:
            self._attached_dbs.add("shared")

    def _try_reattach_shared(self) -> None:
        """Tente de ré-attacher shared_matches.duckdb après une période sync_mode."""
        try:
            self._connection.execute(f"ATTACH '{self._shared_db_path}' AS shared (READ_ONLY)")
            self._attached_dbs.add("shared")
            logger.info("Shared matches DB ré-attachée après sync: %s", self._shared_db_path)
        except Exception as e:
            err_str = str(e).lower()
            if "already" in err_str:
                self._attached_dbs.add("shared")
            else:
                logger.debug("Ré-ATTACH shared échoué: %s", e)

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

        if self.has_shared:
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
                      AND NOT LOWER(CAST(mp2.xuid AS VARCHAR)) LIKE 'bid(%'
                      AND NOT (
                          COALESCE(mp2.kills, 0) = 0
                          AND COALESCE(mp2.deaths, 0) = 0
                          AND COALESCE(mp2.assists, 0) = 0
                          AND COALESCE(mp2.score, 0) = 0
                          AND (mp2.kills IS NOT NULL OR mp2.deaths IS NOT NULL
                               OR mp2.assists IS NOT NULL OR mp2.score IS NOT NULL)
                      )
                    GROUP BY mp2.xuid
                    ORDER BY matches_together DESC
                    LIMIT ?
                    """,
                    [self._xuid, limit],
                )
                return [(row[0], row[1]) for row in result.fetchall()]
            except Exception:
                logger.debug("Erreur chargement coéquipiers fréquents", exc_info=True)
        return []

    # =========================================================================
    # Requêtes avancées
    # =========================================================================

    def query(self, sql: str, params: list | None = None) -> list[dict[str, Any]]:
        """Exécute une requête SQL arbitraire."""
        conn = self._get_connection()
        result = conn.execute(sql, params) if params else conn.execute(sql)
        columns = [desc[0] for desc in result.description]
        return [dict(zip(columns, row, strict=False)) for row in result.fetchall()]

    def query_df(self, sql: str, params: list | None = None):
        """Exécute une requête SQL et retourne un DataFrame Polars."""
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
            self._metadata_resolution_cache = None
            self._mmr_fallback_cache = None

    def __enter__(self) -> DuckDBRepository:
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        self.close()
