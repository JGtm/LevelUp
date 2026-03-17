"""Mixin pour la résolution de gamertags depuis les sources DuckDB.

Extrait de ``_roster_loader.py`` pour respecter le seuil de 500 lignes.
"""

from __future__ import annotations

import logging
import re
from typing import TYPE_CHECKING, Any

from src.utils.xuid import lookup_xuid_for_gamertag

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
        match_id: str | None = None,  # noqa: ARG002 — conservé pour compatibilité API
    ) -> str | None:
        """Résout un XUID en gamertag via shared.v_gamertag_lookup.

        Priorité : xuid_aliases > match_participants (géré par la vue).
        La vue v_gamertag_lookup est garantie présente en architecture v6.

        Args:
            xuid: XUID du joueur à résoudre.
            match_id: Ignoré en v6 (conservé pour compatibilité de signature).

        Returns:
            Gamertag propre ou None si non trouvé.
        """
        if not xuid:
            return None

        conn = self._get_connection()
        xuid = str(xuid).strip()

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
            logger.warning("resolve_gamertag(%s): erreur v_gamertag_lookup", xuid, exc_info=True)

        logger.warning("resolve_gamertag(%s): aucune source", xuid)
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

    def resolve_xuid_from_gamertag(self, gamertag: str) -> str | None:
        """Résout un gamertag → XUID via v_gamertag_lookup puis xuid_aliases.

        Symétrique de resolve_gamertag() pour la direction inverse.

        Args:
            gamertag: Gamertag à résoudre (insensible à la casse).

        Returns:
            XUID en string, ou None si non trouvé.
        """
        conn = self._get_connection()
        return lookup_xuid_for_gamertag(conn, gamertag, view_prefix="shared.")

    def load_match_player_gamertags(self, match_id: str) -> dict[str, str]:
        """Retourne un mapping XUID → Gamertag pour un match via v_gamertag_lookup.

        Résolution v6 : COALESCE(v_gamertag_lookup.gamertag, match_participants.gamertag).
        La vue v_gamertag_lookup est garantie présente (architecture v6).

        Args:
            match_id: ID du match.

        Returns:
            Dict {xuid: gamertag}.
        """
        if not match_id:
            return {}

        conn = self._get_connection()
        try:
            return self._load_gamertags_via_view(conn, match_id)
        except Exception as e:
            logger.debug("Erreur load_match_player_gamertags: %s", e)
            return {}

    def _load_gamertags_via_view(self, conn: object, match_id: str) -> dict[str, str]:
        """Charge les gamertags via JOIN match_participants + v_gamertag_lookup.

        Architecture v6 : xuid_aliases peuplée depuis highlight_events.raw_json
        lors du sync (voir _shared_writes._upsert_event_aliases). La vue
        v_gamertag_lookup est la source unifiée de résolution.
        """
        result: dict[str, str] = {}
        rows = conn.execute(  # type: ignore[union-attr]
            """
            SELECT mp.xuid, COALESCE(vg.gamertag, mp.gamertag) AS gamertag
            FROM shared.match_participants mp
            LEFT JOIN shared.v_gamertag_lookup vg ON mp.xuid = vg.xuid
            WHERE mp.match_id = ? AND mp.xuid IS NOT NULL
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

    def get_all_gamertags(self) -> list[str]:
        """Retourne tous les gamertags connus depuis shared.v_gamertag_lookup.

        La vue v_gamertag_lookup est garantie présente en architecture v6.

        Returns:
            Liste triée de gamertags uniques. [] en cas d'erreur.
        """
        conn = self._get_connection()
        try:
            rows = conn.execute(
                "SELECT DISTINCT gamertag FROM shared.v_gamertag_lookup ORDER BY gamertag"
            ).fetchall()
            result = [str(r[0]) for r in rows if r[0]]
            logger.debug("%d gamertags chargés (v_gamertag_lookup)", len(result))
            return result
        except Exception:
            logger.error("get_all_gamertags: erreur", exc_info=True)
            return []
