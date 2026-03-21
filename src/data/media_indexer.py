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
import os
from datetime import datetime
from pathlib import Path
from typing import Any

import duckdb
import polars as pl

from src.data.media_helpers import (
    IMAGE_EXTENSIONS,
    VIDEO_EXTENSIONS,
    ScanResult,
    compute_file_hash,
    get_all_player_dbs,
    get_existing_columns,
    get_file_metadata,
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
from src.data.media_loaders import load_media_for_match as _load_media_for_match
from src.data.media_loaders import load_media_for_ui as _load_media_for_ui
from src.data.media_thumbnails import generate_thumbnails_for_new as _generate_thumbnails
from src.utils.paths import PLAYER_DB_FILENAME, PLAYERS_DIR

logger = logging.getLogger(__name__)

# Re-exports pour rétrocompatibilité
_get_image_thumbnail_path = get_image_thumbnail_path
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

            has_mf = has_table(conn, "media_files")
            has_mma = has_table(conn, "media_match_associations")

            if has_mf:
                cols = get_existing_columns(conn, "media_files")
                migrations = []
                if "capture_start_utc" not in cols:
                    migrations.append(
                        (
                            "capture_start_utc",
                            "ALTER TABLE media_files ADD COLUMN capture_start_utc TIMESTAMP",
                        )
                    )
                if "capture_end_utc" not in cols:
                    migrations.append(
                        (
                            "capture_end_utc",
                            "ALTER TABLE media_files ADD COLUMN capture_end_utc TIMESTAMP",
                        )
                    )
                if "duration_seconds" not in cols:
                    migrations.append(
                        (
                            "duration_seconds",
                            "ALTER TABLE media_files ADD COLUMN duration_seconds DOUBLE",
                        )
                    )
                if "title" not in cols:
                    migrations.append(("title", "ALTER TABLE media_files ADD COLUMN title VARCHAR"))
                if "status" not in cols:
                    migrations.append(
                        (
                            "status",
                            "ALTER TABLE media_files ADD COLUMN status VARCHAR DEFAULT 'active'",
                        )
                    )
                if "mtime_paris_epoch" not in cols and "mtime" in cols:
                    migrations.append(
                        (
                            "mtime_paris_epoch",
                            "ALTER TABLE media_files ADD COLUMN mtime_paris_epoch DOUBLE",
                        )
                    )
                for _name, sql in migrations:
                    try:
                        conn.execute(sql)
                        conn.commit()
                    except Exception as e:
                        logger.warning("Migration %s: %s", _name, e)
                try:
                    conn.execute("UPDATE media_files SET status = 'active' WHERE status IS NULL")
                    conn.execute(
                        "UPDATE media_files SET mtime_paris_epoch = mtime WHERE mtime_paris_epoch IS NULL"
                    )
                    conn.commit()
                except Exception:
                    pass
            else:
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
                conn.execute(
                    "CREATE INDEX IF NOT EXISTS idx_media_mtime ON media_files(mtime DESC)"
                )
                conn.execute("CREATE INDEX IF NOT EXISTS idx_media_status ON media_files(status)")
                conn.execute("CREATE INDEX IF NOT EXISTS idx_media_kind ON media_files(kind)")
                conn.commit()

            if has_mma:
                cols = get_existing_columns(conn, "media_match_associations")
                if "map_id" not in cols:
                    try:
                        conn.execute(
                            "ALTER TABLE media_match_associations ADD COLUMN map_id VARCHAR"
                        )
                        conn.commit()
                    except Exception:
                        pass
                if "map_name" not in cols:
                    try:
                        conn.execute(
                            "ALTER TABLE media_match_associations ADD COLUMN map_name VARCHAR"
                        )
                        conn.commit()
                    except Exception:
                        pass
            else:
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
            existing = {}
            if not force_rescan:
                rows = conn.execute(
                    "SELECT file_path, file_hash, mtime FROM media_files WHERE status != 'deleted'"
                ).fetchall()
                existing = {row[0]: {"hash": row[1], "mtime": row[2]} for row in rows}

            paths_on_disk: set[str] = set()
            files_to_process: list[dict[str, Any]] = []

            if player_captures_dir and Path(player_captures_dir).exists():
                dirs_exts = [(player_captures_dir, VIDEO_EXTENSIONS | IMAGE_EXTENSIONS)]
            else:
                dirs_exts = [(videos_dir, VIDEO_EXTENSIONS), (screens_dir, IMAGE_EXTENSIONS)]

            for media_dir, exts in dirs_exts:
                if not media_dir or not Path(media_dir).exists():
                    continue
                try:
                    walk_iter = os.walk(media_dir)
                except OSError as e:
                    result.errors.append(f"Dossier inaccessible {media_dir}: {e}")
                    logger.warning("Scan dossier %s: %s", media_dir, e)
                    continue
                for root, _dirs, files in walk_iter:
                    for name in files:
                        fp = Path(root) / name
                        if fp.suffix.lower() not in exts:
                            continue
                        result.n_scanned += 1
                        try:
                            meta = get_file_metadata(fp)
                        except Exception as e:
                            result.errors.append(f"Métadonnées {fp}: {e}")
                            logger.debug("Métadonnées %s: %s", fp, e)
                            continue
                        if not meta:
                            continue
                        path_str = meta["file_path"]
                        paths_on_disk.add(path_str)
                        if path_str in existing:
                            ex = existing[path_str]
                            if not force_rescan and abs(meta["mtime"] - ex["mtime"]) < 1.0:
                                continue
                        h = compute_file_hash(fp)
                        if not h:
                            result.errors.append(f"Hash impossible: {path_str}")
                            continue
                        meta["file_hash"] = h
                        files_to_process.append(meta)

            has_owner_xuid = "owner_xuid" in get_existing_columns(conn, "media_files")
            owner_xuid_val = get_gamertag_from_db_path(self.db_path) or ""

            self._upsert_media(
                conn, files_to_process, existing, has_owner_xuid, owner_xuid_val, now, result
            )

            for path_str in existing:
                if path_str not in paths_on_disk:
                    try:
                        conn.execute(
                            "UPDATE media_files SET status = 'deleted' WHERE file_path = ?",
                            [path_str],
                        )
                        result.n_deleted += 1
                    except Exception as e:
                        result.errors.append(f"Delete {path_str}: {e}")

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
