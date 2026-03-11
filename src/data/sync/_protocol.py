"""Protocol interne — contrat typé entre DuckDBSyncEngine et ses mixins.

Ce Protocol expose les attributs et méthodes de DuckDBSyncEngine que les
mixins utilisent via ``self``. Grâce à lui, mypy peut vérifier les accès
sans ``# type: ignore[attr-defined]``.

Usage dans un mixin :
    from src.data.sync._protocol import _SyncProtocol

    class SomeMixin:
        def _my_method(self: _SyncProtocol, ...) -> ...:
            conn = self._get_connection()   # résolu, pas de type: ignore
"""

from __future__ import annotations

import asyncio
from collections.abc import Callable
from pathlib import Path
from typing import TYPE_CHECKING, Protocol

import duckdb

if TYPE_CHECKING:
    from src.data.sync.models import MatchStatsRow


class _SyncProtocol(Protocol):
    """Contrat structurel de DuckDBSyncEngine vu par les mixins."""

    # ------------------------------------------------------------------
    # Attributs d'identité
    # ------------------------------------------------------------------
    _player_db_path: Path
    _xuid: str
    _gamertag: str
    _spnkr_version: str | None
    _existing_match_ids: set[str] | None
    _friends_xuids: frozenset[str] | None

    # Résolveur de métadonnées (maps, playlists)
    _metadata_resolver: Callable[[str, str | None], str | None] | None

    # ------------------------------------------------------------------
    # Connexions et verrous
    # ------------------------------------------------------------------
    _connection: duckdb.DuckDBPyConnection | None
    _shared_connection: duckdb.DuckDBPyConnection | None
    _pve_connection: duckdb.DuckDBPyConnection | None

    _db_lock: asyncio.Lock
    _shared_db_lock: asyncio.Lock
    _pve_db_lock: asyncio.Lock

    # ------------------------------------------------------------------
    # Méthodes d'accès aux connexions
    # ------------------------------------------------------------------
    def _get_connection(self) -> duckdb.DuckDBPyConnection: ...
    def _get_shared_connection(self) -> duckdb.DuckDBPyConnection | None: ...
    def _get_pve_connection(self) -> duckdb.DuckDBPyConnection: ...

    # ------------------------------------------------------------------
    # Métadonnées de synchronisation
    # ------------------------------------------------------------------
    def _get_sync_meta(self, key: str) -> str | None: ...
    def _update_sync_meta(self, key: str, value: str) -> None: ...

    # ------------------------------------------------------------------
    # Écriture en base
    # ------------------------------------------------------------------
    def _insert_personal_score_rows(self, rows: list) -> None: ...
    def _insert_enrichment_row(self, match_id: str, match_row: MatchStatsRow) -> None: ...
    def _insert_shared_killer_victim_pairs(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        match_id: str,
        highlight_events: list,
    ) -> int: ...
