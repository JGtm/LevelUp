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

    def resolve_gamertag(  # noqa: C901, PLR0912
        self,
        xuid: str,
        *,
        match_id: str | None = None,
    ) -> str | None:
        """Résout un XUID en gamertag avec cascade de sources.

        Sprint Gamertag Roster Fix : Fonction centralisée pour obtenir un
        gamertag propre à partir d'un XUID, en utilisant plusieurs sources.

        Priorité des sources (v5 = shared d'abord):
        1. shared.match_participants (roster complet, v5)
        2. shared.xuid_aliases (source officielle API, v5)
        3. match_participants locale (fallback v4)
        4. xuid_aliases locale
        5. highlight_events (nettoyé avec extraction ASCII)

        Args:
            xuid: XUID du joueur à résoudre.
            match_id: ID du match (optionnel, améliore la résolution contextuelle).

        Returns:
            Gamertag propre ou None si non trouvé.
        """
        conn = self._get_connection()
        xuid = str(xuid).strip()

        # 1. shared.match_participants (v5 — roster complet)
        if match_id and self._has_shared_table("match_participants"):
            try:
                result = conn.execute(
                    "SELECT gamertag FROM shared.match_participants WHERE match_id = ? AND xuid = ?",
                    [match_id, xuid],
                ).fetchone()
                if result and result[0]:
                    cleaned = _clean_gamertag_static(result[0])
                    if cleaned:
                        return cleaned
            except Exception:
                pass

        # 2. shared.xuid_aliases (v5)
        if self._has_shared_table("xuid_aliases"):
            try:
                result = conn.execute(
                    "SELECT gamertag FROM shared.xuid_aliases WHERE xuid = ?",
                    [xuid],
                ).fetchone()
                if result and result[0]:
                    cleaned = _clean_gamertag_static(result[0])
                    if cleaned:
                        return cleaned
            except Exception:
                pass

        # 3. match_participants locale (si match_id fourni et table existe)
        if match_id and self._has_table("match_participants"):
            try:
                result = conn.execute(
                    "SELECT gamertag FROM match_participants WHERE match_id = ? AND xuid = ?",
                    [match_id, xuid],
                ).fetchone()
                if result and result[0]:
                    cleaned = _clean_gamertag_static(result[0])
                    if cleaned:
                        return cleaned
            except Exception:
                pass

        # NOTE v5.1 : xuid_aliases locale supprimé — shared.xuid_aliases utilisé ci-dessus

        # 4. highlight_events avec extraction ASCII (shared d'abord, puis local)
        if match_id:
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
            # Extraire tous les tokens alphanumériques
            parts = re.findall(r"[A-Za-z0-9]+", str(value or ""))
            if not parts:
                return None

            # Prendre le plus long (probablement le gamertag)
            parts.sort(key=len, reverse=True)
            token = parts[0]

            # Minimum 3 caractères pour être un gamertag valide
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

    def load_match_player_gamertags(self, match_id: str) -> dict[str, str]:  # noqa: C901, PLR0912
        """Retourne un mapping XUID → Gamertag pour un match.

        Utilise shared.match_participants (v5) si disponible, sinon la table locale.
        Complète avec shared.xuid_aliases et les sources locales.

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
            # 1. Depuis shared.match_participants (v5, prioritaire — roster complet)
            if self._has_shared_table("match_participants"):
                rows = conn.execute(
                    """
                    SELECT DISTINCT xuid, gamertag
                    FROM shared.match_participants
                    WHERE match_id = ? AND xuid IS NOT NULL AND gamertag IS NOT NULL
                    """,
                    [match_id],
                ).fetchall()
                for xuid, gt in rows:
                    if xuid and gt:
                        result[str(xuid)] = str(gt)

            # 2. Compléter depuis match_participants locale (fallback v4)
            if self._has_table("match_participants"):
                rows = conn.execute(
                    """
                    SELECT DISTINCT xuid, gamertag
                    FROM match_participants
                    WHERE match_id = ? AND xuid IS NOT NULL AND gamertag IS NOT NULL
                    """,
                    [match_id],
                ).fetchall()
                for xuid, gt in rows:
                    if xuid and gt and str(xuid) not in result:
                        result[str(xuid)] = str(gt)

            # 3. Compléter depuis highlight_events (shared d'abord, puis local)
            if self._has_shared_table("highlight_events"):
                rows = conn.execute(
                    """
                    SELECT DISTINCT xuid, gamertag
                    FROM shared.highlight_events
                    WHERE match_id = ? AND xuid IS NOT NULL AND gamertag IS NOT NULL
                    """,
                    [match_id],
                ).fetchall()
                for xuid, gt in rows:
                    if xuid and gt and str(xuid) not in result:
                        result[str(xuid)] = str(gt)

            if self._has_table("highlight_events"):
                rows = conn.execute(
                    """
                    SELECT DISTINCT xuid, gamertag
                    FROM highlight_events
                    WHERE match_id = ? AND xuid IS NOT NULL AND gamertag IS NOT NULL
                    """,
                    [match_id],
                ).fetchall()
                for xuid, gt in rows:
                    if xuid and gt and str(xuid) not in result:
                        result[str(xuid)] = str(gt)

            # 4. Compléter depuis shared.xuid_aliases (v5.1 — source unique)
            all_xuids_in_result = list(result.keys())
            missing = [x for x in all_xuids_in_result if not result.get(x)]

            if missing and self._has_shared_table("xuid_aliases"):
                placeholders = ", ".join(["?" for _ in missing])
                rows = conn.execute(
                    f"SELECT xuid, gamertag FROM shared.xuid_aliases WHERE xuid IN ({placeholders})",
                    missing,
                ).fetchall()
                for xuid, gt in rows:
                    if xuid and gt:
                        result[str(xuid)] = str(gt)

            return result
        except Exception as e:
            logger.debug(f"Erreur load_match_player_gamertags: {e}")
            return result
