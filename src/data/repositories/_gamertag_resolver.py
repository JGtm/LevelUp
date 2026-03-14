"""Mixin pour la résolution de gamertags depuis les sources DuckDB.

Extrait de ``_roster_loader.py`` pour respecter le seuil de 500 lignes.
"""

from __future__ import annotations

import logging
import re
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)


def _clean_gamertag_static(value: Any) -> str | None:
    """Nettoie un gamertag en supprimant les caractères invalides.

    Fonction utilitaire au niveau du module pour éviter les duplications.
    """
    if value is None:
        return None
    try:
        s = str(value)
        s = s.encode("utf-8", errors="ignore").decode("utf-8", errors="ignore")
        s = s.replace("\ufffd", "")
        s = re.sub(r"[\x00-\x1f\x7f-\x9f]", "", s)
        s = re.sub(r"[\ufffe\uffff]", "", s)
        s = re.sub(r"[\s\t]+", " ", s).strip()
        s = s.strip("\u200b\u200c\u200d\ufeff")
        if not s or s == "?" or s.isdigit() or s.lower().startswith("xuid("):
            return None
        if not any(c.isprintable() for c in s):
            return None
        return s
    except Exception:
        return None


class GamertagResolverMixin:
    """Mixin fournissant la résolution XUID → Gamertag pour DuckDBRepository."""

    def resolve_gamertag(
        self,
        xuid: str,
        *,
        match_id: str | None = None,
    ) -> str | None:
        """Résout un XUID en gamertag via v_gamertag_lookup.

        Priorité : xuid_aliases > match_participants (géré par la vue).
        Fallback transitoire : highlight_events avec extraction ASCII.

        Args:
            xuid: XUID du joueur à résoudre.
            match_id: ID du match (optionnel, utilisé uniquement pour le fallback).

        Returns:
            Gamertag propre ou None si non trouvé.
        """
        if not xuid:
            return None

        conn = self._get_connection()
        xuid = str(xuid).strip()

        # Source unique : vue v_gamertag_lookup (xuid_aliases > match_participants)
        if self._has_shared_view("v_gamertag_lookup"):
            try:
                result = conn.execute(
                    "SELECT gamertag FROM shared.v_gamertag_lookup WHERE xuid = ?",
                    [xuid],
                ).fetchone()
                if result and result[0]:
                    cleaned = _clean_gamertag_static(result[0])
                    if cleaned:
                        logger.debug(
                            "resolve_gamertag(%s): source=v_gamertag_lookup → %s", xuid, cleaned
                        )
                        return cleaned
            except Exception:
                pass
        else:
            # Fallback pré-vue : shared.xuid_aliases puis shared.match_participants
            resolved = self._resolve_gamertag_without_view(conn, xuid, match_id)
            if resolved:
                return resolved

        # Fallback transitoire : highlight_events (sera supprimé en Commit 8)
        gt = self._resolve_from_highlight_events(xuid, match_id)
        if gt:
            logger.debug("resolve_gamertag(%s): source=highlight_events → %s", xuid, gt)
            return gt

        logger.warning("resolve_gamertag(%s): aucune source", xuid)
        return None

    def _resolve_gamertag_without_view(
        self,
        conn: object,
        xuid: str,
        match_id: str | None,
    ) -> str | None:
        """Fallback quand v_gamertag_lookup n'est pas encore disponible.

        Ordre : shared.xuid_aliases, shared.match_participants (si match_id fourni).
        """
        if self._has_shared_table("xuid_aliases"):
            try:
                result = conn.execute(  # type: ignore[union-attr]
                    "SELECT gamertag FROM shared.xuid_aliases WHERE xuid = ?",
                    [xuid],
                ).fetchone()
                if result and result[0]:
                    cleaned = _clean_gamertag_static(result[0])
                    if cleaned:
                        return cleaned
            except Exception:
                pass
        if match_id and self._has_shared_table("match_participants"):
            try:
                result = conn.execute(  # type: ignore[union-attr]
                    "SELECT gamertag FROM shared.match_participants WHERE match_id = ? AND xuid = ?",
                    [match_id, xuid],
                ).fetchone()
                if result and result[0]:
                    cleaned = _clean_gamertag_static(result[0])
                    if cleaned:
                        return cleaned
            except Exception:
                pass
        return None

    def _resolve_from_highlight_events(
        self,
        xuid: str,
        match_id: str | None,
    ) -> str | None:
        """Fallback transitoire : résolution gamertag depuis highlight_events.

        Données corrompues (NUL bytes) — elles seront supprimées en Commit 8.
        Méthode extraite pour que resolve_gamertag() reste < 80 L.
        """
        if not match_id:
            return None

        conn = self._get_connection()
        he_sources = []
        if self._has_shared_table("highlight_events"):
            he_sources.append("shared.highlight_events")
        if self._has_table("highlight_events"):
            he_sources.append("highlight_events")

        for he_src in he_sources:
            try:
                result = conn.execute(
                    f"SELECT gamertag FROM {he_src} WHERE match_id = ? AND xuid = ? LIMIT 1",
                    [match_id, xuid],
                ).fetchone()
                if result and result[0]:
                    cleaned = self._extract_ascii_token(result[0])
                    if cleaned:
                        return cleaned
            except Exception:
                pass

        return None

    def _extract_ascii_token(self, value: str | None) -> str | None:
        """Extrait un token ASCII plausible depuis un gamertag corrompu.

        Les gamertags provenant de highlight_events peuvent contenir des
        caractères NUL et de contrôle (ex: 'juan1\\x00\\x00\\x00\\x00').
        Cette fonction extrait la partie lisible.

        Args:
            value: Gamertag potentiellement corrompu.

        Returns:
            Token ASCII nettoyé ou None si rien de plausible.
        """
        if value is None:
            return None

        try:
            parts = re.findall(r"[A-Za-z0-9]+", str(value or ""))
            if not parts:
                return None
            parts.sort(key=len, reverse=True)
            token = parts[0]
            return token if len(token) >= 3 else None
        except Exception:
            return None

    def resolve_gamertags_batch(
        self,
        xuids: list[str],
        *,
        match_id: str | None = None,
    ) -> dict[str, str | None]:
        """Résout plusieurs XUIDs en gamertags en batch.

        Args:
            xuids: Liste des XUIDs à résoudre.
            match_id: ID du match (optionnel).

        Returns:
            Dict {xuid: gamertag} pour chaque XUID.
        """
        return {xuid: self.resolve_gamertag(xuid, match_id=match_id) for xuid in xuids}

    def load_match_player_gamertags(self, match_id: str) -> dict[str, str]:
        """Retourne un mapping XUID → Gamertag pour un match via v_gamertag_lookup.

        Résolution : COALESCE(v_gamertag_lookup.gamertag, match_participants.gamertag).
        Fallback si la vue n'est pas disponible : match_participants seul.

        Args:
            match_id: ID du match.

        Returns:
            Dict {xuid: gamertag}.
        """
        if not match_id:
            return {}

        conn = self._get_connection()
        result: dict[str, str] = {}

        try:
            if self._has_shared_view("v_gamertag_lookup") and self._has_shared_table(
                "match_participants"
            ):
                return self._load_gamertags_via_view(conn, match_id)

            # Fallback : match_participants seul (sans vue)
            return self._load_gamertags_fallback(conn, match_id, result)
        except Exception as e:
            logger.debug("Erreur load_match_player_gamertags: %s", e)
            return result

    def _load_gamertags_via_view(self, conn: object, match_id: str) -> dict[str, str]:
        """Charge les gamertags via JOIN match_participants + v_gamertag_lookup."""
        result: dict[str, str] = {}
        rows = conn.execute(  # type: ignore[union-attr]
            """
            SELECT mp.xuid, COALESCE(vg.gamertag, mp.gamertag) AS gamertag
            FROM shared.match_participants mp
            LEFT JOIN shared.v_gamertag_lookup vg ON mp.xuid = vg.xuid
            WHERE mp.match_id = ?
              AND mp.xuid IS NOT NULL
            """,
            [match_id],
        ).fetchall()
        for xuid, gt in rows:
            cleaned = _clean_gamertag_static(gt)
            if xuid and cleaned:
                result[str(xuid)] = cleaned
        if not result:
            logger.debug("load_match_player_gamertags(%s): aucun joueur résolu", match_id)
        return result

    def _load_gamertags_fallback(
        self,
        conn: object,
        match_id: str,
        result: dict[str, str],
    ) -> dict[str, str]:
        """Fallback : charge les gamertags depuis les tables brutes (sans vue)."""
        for src in ("shared.match_participants", "match_participants"):
            table = src.split(".")[-1]
            if (src.startswith("shared.") and not self._has_shared_table(table)) or (
                not src.startswith("shared.") and not self._has_table(table)
            ):
                continue
            try:
                rows = conn.execute(  # type: ignore[union-attr]
                    f"SELECT DISTINCT xuid, gamertag FROM {src} "
                    "WHERE match_id = ? AND xuid IS NOT NULL AND gamertag IS NOT NULL",
                    [match_id],
                ).fetchall()
                for xuid, gt in rows:
                    if xuid and gt and str(xuid) not in result:
                        result[str(xuid)] = str(gt)
            except Exception:
                pass
        return result
