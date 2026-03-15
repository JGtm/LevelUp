"""Mixin pour l'accès aux données de la médiathèque.

Regroupe :
- Fenêtres temporelles des matchs (shared.match_registry)
- Fichiers médias et associations (media_files + media_match_associations)
"""

from __future__ import annotations

import logging

logger = logging.getLogger(__name__)


class MediaLibraryMixin:
    """Mixin fournissant les données médiathèque pour DuckDBRepository."""

    def load_match_registry_raw(self) -> list[tuple]:
        """Charge match_id, start_time, duration_seconds depuis shared.match_registry.

        Retourne toutes les lignes sans filtre xuid (usage : fenêtres temporelles médias).

        Returns:
            Liste de tuples (match_id, start_time, duration_seconds). [] si indisponible.
        """
        if not self._has_shared_table("match_registry"):
            return []
        conn = self._get_connection()
        try:
            return conn.execute(
                """
                SELECT match_id, start_time, duration_seconds
                FROM shared.match_registry
                WHERE start_time IS NOT NULL
                """
            ).fetchall()
        except Exception:
            logger.debug("load_match_registry_raw: erreur", exc_info=True)
            return []

    def load_media_files_raw(
        self,
        xuid: str | None,
        gamertag: str | None,
    ) -> list[tuple] | None:
        """Charge media_files + media_match_associations depuis la DB joueur.

        Args:
            xuid: XUID pour filtrer media_match_associations (optionnel).
            gamertag: Gamertag fallback pour le filtre (optionnel).

        Returns:
            Liste de tuples (path, mtime, mtime_paris_epoch, ext, kind, basename,
            thumbnail_path, match_id, match_start_time, association_confidence, xuid),
            ou None si la table media_files est absente.
        """
        if not self._has_table("media_files"):
            return None
        conn = self._get_connection()

        uids = [u for u in (xuid, gamertag) if u]
        uids = list(dict.fromkeys(uids))

        _select = """
            SELECT DISTINCT
                mf.file_path,
                mf.mtime,
                mf.mtime_paris_epoch,
                mf.file_ext,
                mf.kind,
                mf.file_name,
                mf.thumbnail_path,
                mma.match_id,
                mma.match_start_time,
                mma.association_confidence,
                mma.xuid
            FROM media_files mf
            LEFT JOIN media_match_associations mma ON mf.file_path = mma.media_path
        """

        if uids:
            if len(uids) == 1:
                uid_filter = "mma.xuid = ?"
                params: list[str] = [uids[0]]
            else:
                uid_filter = "(mma.xuid = ? OR mma.xuid = ?)"
                params = list(uids[:2])
            sql = f"{_select} WHERE {uid_filter} ORDER BY mf.mtime_paris_epoch DESC"  # noqa: S608
        else:
            sql = f"{_select} ORDER BY mf.mtime_paris_epoch DESC"  # noqa: S608
            params = []

        try:
            return conn.execute(sql, params).fetchall()
        except Exception:
            logger.debug("load_media_files_raw: erreur", exc_info=True)
            return None
