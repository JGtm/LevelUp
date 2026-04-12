"""Module d'indexation des médias – Onglet Médias.

Gère :
- Scan delta des dossiers configurés
- Schéma media_files (capture_start_utc, capture_end_utc, duration_seconds, title, status)
- Association média ↔ match
- Thumbnails (délégué à ``media_thumbnails``)

Chaque joueur a sa propre BDD : data/players/{gamertag}/stats.duckdb

Sous-modules extraits :
- ``media_helpers`` : fonctions libres, ScanResult, constantes
- ``media_loaders`` : load_media_for_ui, load_media_for_match
- ``media_thumbnails`` : génération de miniatures
"""

from __future__ import annotations

import logging
from datetime import datetime
from pathlib import Path
from typing import Any

import duckdb
import polars as pl

from src.data.media_helpers import (
    ScanResult,
    get_all_player_dbs,
    get_existing_columns,
    get_gamertag_from_db_path,
    get_image_thumbnail_path,
    insert_new_media,
    match_start_to_epoch,
    update_existing_media,
)
from src.data.media_indexer_matchers import (  # noqa: F401
    _associate_single_media,
    _load_matches_by_xuid,
)
from src.data.media_indexer_scan import (
    load_existing_media,
    mark_deleted_media,
    resolve_scan_targets,
    scan_media_dirs,
)
from src.data.media_loaders import load_media_for_match as _load_media_for_match
from src.data.media_loaders import load_media_for_ui as _load_media_for_ui
from src.data.media_thumbnails import generate_thumbnails_for_new as _generate_thumbnails
from src.utils.paths import PLAYER_DB_FILENAME, PLAYERS_DIR

logger = logging.getLogger(__name__)

# Re-exports pour retrocompatibilite
_get_image_thumbnail_path = get_image_thumbnail_path
# Deprecated: conserver uniquement pour d'eventuels imports legacy externes.
_match_start_to_epoch = match_start_to_epoch

__all__ = [
    "MediaIndexer",
    "ScanResult",
    "_get_image_thumbnail_path",
]


class MediaIndexer:
    """Gère l'indexation des médias (scan delta, schéma v2)."""

    def __init__(self, db_path: Path | None = None):
        if db_path:
            self.db_path = Path(db_path)
        else:
            if not PLAYERS_DIR.exists():
                raise ValueError("Aucune DB joueur trouvée dans data/players/")
            for player_dir in PLAYERS_DIR.iterdir():
                if player_dir.is_dir():
                    db_file = player_dir / PLAYER_DB_FILENAME
                    if db_file.exists():
                        self.db_path = db_file
                        break
            if not hasattr(self, "db_path") or not self.db_path.exists():
                raise ValueError("Aucune DB joueur valide trouvée")

    # ------------------------------------------------------------------
    #  Schéma
    # ------------------------------------------------------------------

    def reset_media_tables(self) -> None:
        """Vide les tables media_files et media_match_associations (schéma conservé)."""
        with duckdb.connect(str(self.db_path), read_only=False) as conn:
            self.ensure_schema()
            conn.execute("DELETE FROM media_match_associations")
            conn.execute("DELETE FROM media_files")
            conn.commit()

    def ensure_schema(self) -> None:  # noqa: C901, PLR0912, PLR0915
        """Crée ou migre le schéma media_files et media_match_associations."""
        try:
            conn = duckdb.connect(str(self.db_path), read_only=False)
        except duckdb.IOException as e:
            logger.warning(
                "Impossible d'ouvrir %s en écriture (verrouillé ?): %s",
                self.db_path,
                e,
            )
            raise
        with conn:
            from src.utils.db import has_table

            self._ensure_media_files_schema(conn, has_table(conn, "media_files"))
            self._ensure_media_match_associations_schema(
                conn,
                has_table(conn, "media_match_associations"),
            )

    @staticmethod
    def _ensure_media_files_schema(conn: duckdb.DuckDBPyConnection, exists: bool) -> None:
        """Crée ou migre la table media_files."""
        if exists:
            cols = get_existing_columns(conn, "media_files")
            MediaIndexer._apply_media_files_migrations(conn, cols)
            MediaIndexer._normalize_media_files_rows(conn)
            return
        MediaIndexer._create_media_files_table(conn)

    @staticmethod
    def _apply_media_files_migrations(
        conn: duckdb.DuckDBPyConnection,
        cols: set[str],
    ) -> None:
        """Applique les migrations de colonnes manquantes pour media_files."""
        for name, sql in MediaIndexer._build_media_files_migrations(cols):
            try:
                conn.execute(sql)
                conn.commit()
            except Exception as exc:
                logger.warning("Migration %s: %s", name, exc)

    @staticmethod
    def _build_media_files_migrations(cols: set[str]) -> list[tuple[str, str]]:
        """Construit la liste des migrations media_files à exécuter."""
        migrations: list[tuple[str, str]] = []
        column_sql = {
            "capture_start_utc": "ALTER TABLE media_files ADD COLUMN capture_start_utc TIMESTAMP",
            "capture_end_utc": "ALTER TABLE media_files ADD COLUMN capture_end_utc TIMESTAMP",
            "duration_seconds": "ALTER TABLE media_files ADD COLUMN duration_seconds DOUBLE",
            "title": "ALTER TABLE media_files ADD COLUMN title VARCHAR",
            "status": "ALTER TABLE media_files ADD COLUMN status VARCHAR DEFAULT 'active'",
        }
        for column_name, sql in column_sql.items():
            if column_name not in cols:
                migrations.append((column_name, sql))
        if "mtime_paris_epoch" not in cols and "mtime" in cols:
            migrations.append(
                (
                    "mtime_paris_epoch",
                    "ALTER TABLE media_files ADD COLUMN mtime_paris_epoch DOUBLE",
                )
            )
        return migrations

    @staticmethod
    def _normalize_media_files_rows(conn: duckdb.DuckDBPyConnection) -> None:
        """Backfill les colonnes media_files ajoutées après création initiale."""
        try:
            conn.execute("UPDATE media_files SET status = 'active' WHERE status IS NULL")
            conn.execute(
                "UPDATE media_files SET mtime_paris_epoch = mtime WHERE mtime_paris_epoch IS NULL"
            )
            conn.commit()
        except Exception:
            pass

    @staticmethod
    def _create_media_files_table(conn: duckdb.DuckDBPyConnection) -> None:
        """Crée la table media_files et ses index."""
        conn.execute("""
            CREATE TABLE media_files (
                file_path VARCHAR PRIMARY KEY,
                file_hash VARCHAR NOT NULL,
                file_name VARCHAR NOT NULL,
                file_size BIGINT NOT NULL,
                file_ext VARCHAR NOT NULL,
                kind VARCHAR NOT NULL,
                capture_start_utc TIMESTAMP,
                capture_end_utc TIMESTAMP NOT NULL,
                duration_seconds DOUBLE,
                title VARCHAR,
                thumbnail_path VARCHAR,
                mtime DOUBLE NOT NULL,
                mtime_paris_epoch DOUBLE,
                status VARCHAR NOT NULL DEFAULT 'active',
                first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                last_scan_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                scan_version INTEGER DEFAULT 2
            )
        """)
        conn.execute("CREATE INDEX IF NOT EXISTS idx_media_mtime ON media_files(mtime DESC)")
        conn.execute("CREATE INDEX IF NOT EXISTS idx_media_status ON media_files(status)")
        conn.execute("CREATE INDEX IF NOT EXISTS idx_media_kind ON media_files(kind)")
        conn.commit()

    @staticmethod
    def _ensure_media_match_associations_schema(
        conn: duckdb.DuckDBPyConnection,
        exists: bool,
    ) -> None:
        """Crée ou migre la table media_match_associations."""
        if exists:
            MediaIndexer._ensure_media_match_association_columns(conn)
            return
        MediaIndexer._create_media_match_associations_table(conn)

    @staticmethod
    def _ensure_media_match_association_columns(conn: duckdb.DuckDBPyConnection) -> None:
        """Ajoute les colonnes manquantes sur media_match_associations."""
        cols = get_existing_columns(conn, "media_match_associations")
        for column_name in ("map_id", "map_name"):
            if column_name in cols:
                continue
            try:
                conn.execute(
                    f"ALTER TABLE media_match_associations ADD COLUMN {column_name} VARCHAR"
                )
                conn.commit()
            except Exception:
                pass

    @staticmethod
    def _create_media_match_associations_table(conn: duckdb.DuckDBPyConnection) -> None:
        """Crée la table media_match_associations et ses index."""
        conn.execute("""
            CREATE TABLE media_match_associations (
                media_path VARCHAR NOT NULL,
                match_id VARCHAR NOT NULL,
                xuid VARCHAR NOT NULL,
                match_start_time TIMESTAMP NOT NULL,
                map_id VARCHAR,
                map_name VARCHAR,
                association_confidence DOUBLE DEFAULT 1.0,
                associated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                PRIMARY KEY (media_path, match_id, xuid)
            )
        """)
        conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_assoc_media ON media_match_associations(media_path)"
        )
        conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_assoc_match ON media_match_associations(match_id, xuid)"
        )
        conn.commit()

    # ------------------------------------------------------------------
    #  Scan
    # ------------------------------------------------------------------

    def scan_and_index(  # noqa: C901, PLR0912, PLR0915
        self,
        videos_dir: Path | None = None,
        screens_dir: Path | None = None,
        *,
        player_captures_dir: Path | None = None,
        force_rescan: bool = False,
    ) -> ScanResult:
        """Scan delta : nouveaux, modifiés, absents → status='deleted'."""
        self.ensure_schema()
        result = ScanResult()
        now = datetime.now()

        with duckdb.connect(str(self.db_path), read_only=False) as conn:
            existing = load_existing_media(conn)
            dirs_exts = resolve_scan_targets(player_captures_dir, videos_dir, screens_dir)
            paths_on_disk, files_to_process = scan_media_dirs(
                dirs_exts,
                existing,
                force_rescan,
                result,
            )

            has_owner_xuid = "owner_xuid" in get_existing_columns(conn, "media_files")
            owner_xuid_val = get_gamertag_from_db_path(self.db_path) or ""

            self._upsert_media(
                conn, files_to_process, existing, has_owner_xuid, owner_xuid_val, now, result
            )

            mark_deleted_media(conn, existing, paths_on_disk, result)

            conn.commit()
            logger.info(
                "Scan: %d scannés, %d nouveaux, %d modifiés, %d supprimés",
                result.n_scanned,
                result.n_new,
                result.n_updated,
                result.n_deleted,
            )
        return result

    @staticmethod
    def _upsert_media(  # noqa: PLR0913
        conn: duckdb.DuckDBPyConnection,
        files_to_process: list[dict[str, Any]],
        existing: dict[str, Any],
        has_owner_xuid: bool,
        owner_xuid_val: str,
        now: datetime,
        result: ScanResult,
    ) -> None:
        """Insère ou met à jour les médias dans la DB."""
        for meta in files_to_process:
            path_str = meta["file_path"]
            is_new = path_str not in existing
            try:
                if is_new:
                    insert_new_media(conn, meta, has_owner_xuid, owner_xuid_val, now)
                    result.n_new += 1
                else:
                    update_existing_media(conn, meta, now)
                    result.n_updated += 1
            except Exception as e:
                result.errors.append(f"Insert {path_str}: {e}")
                logger.exception("Insert média: %s", e)

    # ------------------------------------------------------------------
    #  Association
    # ------------------------------------------------------------------

    def associate_with_matches(self, tolerance_minutes: int = 1) -> int:  # noqa: C901, PLR0912
        """Associe les médias actifs avec les matchs (multi-joueurs).

        Returns:
            Nombre de nouvelles associations insérées.
        """
        self.ensure_schema()
        with duckdb.connect(str(self.db_path), read_only=False) as conn:
            try:
                media_rows = conn.execute(
                    "SELECT mf.file_path, "
                    "COALESCE(epoch(mf.capture_end_utc), mf.mtime_paris_epoch, mf.mtime) "
                    "FROM media_files mf "
                    "WHERE mf.status = 'active' ORDER BY mf.mtime DESC"
                ).fetchall()
            except Exception:
                return 0
            if not media_rows:
                return 0

            player_dbs = self._get_all_player_dbs_current_first()
            if not player_dbs:
                player_dbs = [(self.db_path, get_gamertag_from_db_path(self.db_path) or "")]

            matches_by_xuid = _load_matches_by_xuid(self.db_path, player_dbs)
            tol_seconds = tolerance_minutes * 60
            before = conn.execute("SELECT COUNT(*) FROM media_match_associations").fetchone()[0]

            for media_path, mtime_epoch in media_rows:
                _associate_single_media(conn, media_path, mtime_epoch, matches_by_xuid, tol_seconds)

            conn.commit()
            after = conn.execute("SELECT COUNT(*) FROM media_match_associations").fetchone()[0]
            return int(after - before)

    # ------------------------------------------------------------------
    #  Thumbnails (délégation)
    # ------------------------------------------------------------------

    def generate_thumbnails_for_new(
        self,
        videos_dir: Path | None = None,
        screens_dir: Path | None = None,
        *,
        max_concurrent: int = 2,  # noqa: ARG002
    ) -> tuple[int, int]:
        """Génère les thumbnails vidéo (GIF) et image (miniatures)."""
        return _generate_thumbnails(
            self.db_path,
            videos_dir,
            screens_dir,
            ensure_schema_fn=self.ensure_schema,
        )

    # ------------------------------------------------------------------
    #  Loaders (délégation — rétrocompatibilité)
    # ------------------------------------------------------------------

    @staticmethod
    def load_media_for_ui(db_path: Path | str, current_xuid: str | None) -> pl.DataFrame:
        """Charge les médias actifs pour l'onglet Médias."""
        return _load_media_for_ui(db_path, current_xuid)

    @staticmethod
    def load_media_for_match(
        db_path: Path | str,
        match_id: str,
        current_xuid: str | None = None,
    ) -> pl.DataFrame:
        """Charge tous les médias associés à un match spécifique."""
        return _load_media_for_match(db_path, match_id, current_xuid)

    # ------------------------------------------------------------------
    #  Helpers internes (délégation)
    # ------------------------------------------------------------------

    @staticmethod
    def _get_all_player_dbs() -> list[tuple[Path, str]]:
        """Retourne les (db_path, xuid) de tous les joueurs."""
        return get_all_player_dbs()

    def _get_all_player_dbs_current_first(self) -> list[tuple[Path, str]]:
        """Liste des (db_path, xuid) avec la DB courante en premier."""
        all_dbs = self._get_all_player_dbs()
        current = self.db_path.resolve()
        current_first = [(p, x) for p, x in all_dbs if p.resolve() == current]
        others = [(p, x) for p, x in all_dbs if p.resolve() != current]
        return current_first + others
