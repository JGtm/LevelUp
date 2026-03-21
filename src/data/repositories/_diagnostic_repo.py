"""Mixin de diagnostic et métadonnées sync pour DuckDBRepository.

Méthodes utilitaires : informations de stockage, métadonnées de synchronisation,
et vérification de disponibilité.
"""

from __future__ import annotations

import logging
from typing import Any

logger = logging.getLogger(__name__)


class DiagnosticMixin:
    """Diagnostic de stockage et métadonnées de synchronisation."""

    def get_sync_metadata(self) -> dict[str, Any]:
        """Récupère les métadonnées de synchronisation."""
        conn = self._get_connection()

        last_sync = None
        if "meta" in self._attached_dbs:
            try:
                result = conn.execute(
                    "SELECT last_sync_at FROM meta.sync_meta WHERE xuid = ?",
                    [self._xuid],
                ).fetchone()
                last_sync = result[0] if result else None
            except Exception:
                pass

        return {
            "last_sync_at": last_sync,
            "last_match_time": None,
            "total_matches": self.get_match_count(),
            "player_xuid": self._xuid,
            "storage_type": "duckdb",
        }

    def get_storage_info(self) -> dict[str, Any]:
        """Retourne des informations sur le stockage (local + shared)."""
        conn = self._get_connection()

        # Taille des tables locales
        tables_info = {}
        for table in ["player_match_enrichment", "personal_score_awards", "match_citations"]:
            try:
                count = conn.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
                tables_info[table] = count
            except Exception as e:
                logger.debug("get_storage_info: COUNT(*) échoué pour %s: %s", table, e)
                tables_info[table] = 0

        # Counts shared
        shared_info = self._collect_shared_counts(conn)

        # Taille des fichiers
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

    def _collect_shared_counts(self, conn) -> dict[str, int]:
        """Collecte les counts des tables shared pour le diagnostic."""
        shared_info: dict[str, int] = {}
        if not self.has_shared:
            return shared_info

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
            except Exception as e:
                logger.debug("_collect_shared_counts: COUNT(*) échoué pour shared.%s: %s", table, e)
                shared_info[f"shared_{table}"] = 0
        return shared_info

    def is_hybrid_available(self) -> bool:
        """Vérifie si les données sont disponibles."""
        return self._player_db_path.exists() and self.get_match_count() > 0
