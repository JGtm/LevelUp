"""
Mixin pour le chargement des rosters et la résolution des gamertags.

Regroupe les méthodes d'analyse des compositions d'équipes et
de résolution XUID → Gamertag extraites de DuckDBRepository :
- load_match_rosters
- load_matches_with_teammate
- load_same_team_match_ids
- has_match_participants
- resolve_gamertag
- _extract_ascii_token
- resolve_gamertags_batch
- load_match_player_gamertags
- load_match_players_stats
"""

from __future__ import annotations

import contextlib
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


class RosterLoaderMixin:
    """Mixin fournissant le chargement des rosters et la résolution des gamertags pour DuckDBRepository."""

    def load_match_rosters(
        self,
        match_id: str,
    ) -> dict[str, Any] | None:
        """Charge les rosters d'un match depuis killer_victim_pairs ou highlight_events.

        Utilise killer_victim_pairs si disponible (source fiable), sinon
        analyse les patterns de kills dans highlight_events.

        Returns:
            None si le match n'existe pas ou si les données sont insuffisantes.
            Sinon un dict avec la structure:
            {
                "my_team_id": int,
                "my_team": [{"xuid": str, "gamertag": str|None, "team_id": int|None, "is_me": bool}],
                "enemy_team": [...],
            }
        """
        conn = self._get_connection()

        try:
            # V5.1 : shared.match_participants est la source canonique du team_id
            my_xuid_str = str(self._xuid).strip()
            match_info = None

            if self._has_shared_table("match_participants"):
                with contextlib.suppress(Exception):
                    match_info = conn.execute(
                        "SELECT team_id FROM shared.match_participants WHERE match_id = ? AND xuid = ?",
                        [match_id, my_xuid_str],
                    ).fetchone()

            # NOTE v5.1 : match_stats supprimée, pas de fallback legacy
            if not match_info:
                return None

            my_team_id = match_info[0]
            if my_team_id is None:
                return None

            # Alias local pour la fonction de nettoyage module-level
            _clean_gamertag = _clean_gamertag_static

            # ======================================================================
            # MÉTHODE 0 (PRIORITAIRE) : Utiliser match_participants.team_id
            # C'est la source la plus fiable car elle vient directement de l'API
            # ======================================================================
            team_by_xuid: dict[str, int | None] = {}
            gamertag_by_xuid: dict[str, str | None] = {}
            mp_success = False

            if self._has_shared_table("match_participants"):
                try:
                    # Requête de base : match_participants seul (toujours présent en v5.1)
                    mp_result = conn.execute(
                        """
                        SELECT xuid, team_id, gamertag
                        FROM shared.match_participants
                        WHERE match_id = ?
                        """,
                        [match_id],
                    ).fetchall()

                    if mp_result and len(mp_result) > 0:
                        for xuid, team_id, mp_gt in mp_result:
                            xu = str(xuid).strip()
                            if not xu:
                                continue

                            # Stocker le team_id (source fiable)
                            team_by_xuid[xu] = team_id

                            # Gamertag depuis match_participants
                            gt = _clean_gamertag(mp_gt)
                            if gt:
                                gamertag_by_xuid[xu] = gt

                        # Enrichir les gamertags depuis xuid_aliases (optionnel)
                        if self._has_shared_table("xuid_aliases"):
                            try:
                                alias_result = conn.execute(
                                    """
                                    SELECT xuid, gamertag
                                    FROM shared.xuid_aliases
                                    WHERE xuid IN (SELECT xuid FROM shared.match_participants WHERE match_id = ?)
                                    """,
                                    [match_id],
                                ).fetchall()
                                for xuid, alias_gt in alias_result:
                                    xu = str(xuid).strip()
                                    gt = _clean_gamertag(alias_gt)
                                    if gt and (
                                        xu not in gamertag_by_xuid
                                        or len(gt) > len(gamertag_by_xuid.get(xu) or "")
                                    ):
                                        gamertag_by_xuid[xu] = gt
                            except Exception:
                                pass

                        # Enrichir les gamertags depuis highlight_events (optionnel)
                        if self._has_shared_table("highlight_events"):
                            try:
                                he_result = conn.execute(
                                    """
                                    SELECT xuid, gamertag
                                    FROM (
                                        SELECT xuid, gamertag,
                                               ROW_NUMBER() OVER (
                                                   PARTITION BY xuid
                                                   ORDER BY LENGTH(COALESCE(gamertag, '')) DESC
                                               ) as rn
                                        FROM shared.highlight_events
                                        WHERE match_id = ? AND xuid IS NOT NULL AND xuid != ''
                                    ) sub
                                    WHERE rn = 1
                                    """,
                                    [match_id],
                                ).fetchall()
                                for xuid, he_gt in he_result:
                                    xu = str(xuid).strip()
                                    gt = _clean_gamertag(he_gt)
                                    if gt and (
                                        xu not in gamertag_by_xuid
                                        or len(gt) > len(gamertag_by_xuid.get(xu) or "")
                                    ):
                                        gamertag_by_xuid[xu] = gt
                            except Exception:
                                pass

                        mp_success = len(team_by_xuid) >= 1
                        logger.debug(
                            f"MÉTHODE 0: {len(team_by_xuid)} participants chargés depuis match_participants"
                        )

                except Exception as e:
                    logger.debug(f"Erreur lecture match_participants: {e}")

            # ======================================================================
            # V5.1 : match_participants.team_id est la source canonique.
            # Pas de fallback killer_victim_pairs/highlight_events (obsolète).
            # ======================================================================
            if not mp_success:
                # En v5.1, match_participants devrait toujours être disponible
                logger.warning(f"match_participants manquant pour {match_id}")
                return None

            all_xuids: set[str] = set(team_by_xuid.keys())

            # ======================================================================
            # Construire les listes d'équipes
            # ======================================================================
            # Sprint Gamertag Roster Fix : Utiliser resolve_gamertags_batch pour
            # obtenir des gamertags propres depuis match_participants/xuid_aliases
            resolved_gamertags = self.resolve_gamertags_batch(list(all_xuids), match_id=match_id)

            my_team = []
            enemy_team = []

            for xuid_str in all_xuids:
                is_me = xuid_str == my_xuid_str
                # Priorité : gamertag résolu > gamertag extrait > XUID
                cleaned_gamertag = resolved_gamertags.get(xuid_str) or gamertag_by_xuid.get(
                    xuid_str
                )
                display_name = cleaned_gamertag if cleaned_gamertag else xuid_str
                player_team_id = team_by_xuid.get(xuid_str, None if not is_me else my_team_id)

                player_data = {
                    "xuid": xuid_str,
                    "gamertag": cleaned_gamertag,
                    "team_id": player_team_id,
                    "is_me": is_me,
                    "is_bot": False,
                    "display_name": display_name,
                }

                if player_team_id == my_team_id or is_me:
                    my_team.append(player_data)
                else:
                    enemy_team.append(player_data)

            # Trier: moi en premier, puis alphabétique
            def _sort_key(r: dict[str, Any]) -> tuple[int, str]:
                me_rank = 0 if r.get("is_me") else 1
                name = str(r.get("gamertag") or r.get("xuid") or "").strip().lower()
                return (me_rank, name)

            my_team.sort(key=_sort_key)
            enemy_team.sort(key=_sort_key)

            return {
                "my_team_id": int(my_team_id),
                "my_team_name": None,
                "my_team": my_team,
                "enemy_team": enemy_team,
                "enemy_team_ids": [],
                "enemy_team_names": [],
            }
        except Exception as e:
            logger.warning(f"Erreur lors du chargement des rosters pour {match_id}: {e}")
            return None

    def load_matches_with_teammate(
        self,
        teammate_xuid: str,
    ) -> list[str]:
        """Retourne les match_id joués avec un coéquipier depuis shared.

        Args:
            teammate_xuid: XUID du coéquipier.

        Returns:
            Liste des match_id où les deux joueurs apparaissent.
        """
        conn = self._get_connection()

        try:
            result = conn.execute(
                """
                SELECT DISTINCT me.match_id
                FROM shared.match_participants me
                INNER JOIN shared.match_participants tm
                    ON me.match_id = tm.match_id
                WHERE me.xuid = ? AND tm.xuid = ?
                ORDER BY me.match_id DESC
                """,
                [self._xuid, teammate_xuid],
            )
            return [row[0] for row in result.fetchall()]
        except Exception as e:
            logger.debug(f"Erreur load_matches_with_teammate shared: {e}")
            return []

    def load_same_team_match_ids(
        self,
        teammate_xuid: str,
    ) -> list[str]:
        """Retourne les match_id où les deux joueurs étaient dans la même équipe.

        Utilise shared.match_participants (team_id fiable).

        Args:
            teammate_xuid: XUID du coéquipier.

        Returns:
            Liste des match_id où les deux joueurs étaient dans la même équipe.
        """
        conn = self._get_connection()

        try:
            result = conn.execute(
                """
                SELECT DISTINCT me.match_id
                FROM shared.match_participants me
                INNER JOIN shared.match_participants tm
                    ON me.match_id = tm.match_id
                    AND me.team_id = tm.team_id
                WHERE me.xuid = ? AND tm.xuid = ?
                ORDER BY me.match_id DESC
                """,
                [self._xuid, teammate_xuid],
            )
            return [row[0] for row in result.fetchall()]
        except Exception as e:
            logger.debug(f"Erreur load_same_team_match_ids shared: {e}")
            return []

    def has_match_participants(self) -> bool:
        """Vérifie si des données match_participants existent dans shared."""
        conn = self._get_connection()
        try:
            count = conn.execute(
                "SELECT COUNT(*) FROM shared.match_participants WHERE xuid = ?",
                [self._xuid],
            ).fetchone()[0]
            return count > 0
        except Exception:
            return False

    def resolve_gamertag(
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

    def load_match_player_gamertags(self, match_id: str) -> dict[str, str]:
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

    def load_match_players_stats(self, match_id: str) -> list[dict[str, Any]]:
        """Charge les statistiques officielles de tous les joueurs d'un match.

        Utilise shared.match_participants (roster complet, toutes colonnes).

        Args:
            match_id: ID du match.

        Returns:
            Liste de dicts avec: xuid, gamertag, kills, deaths, assists, team_id, rank, score
        """
        if not match_id:
            return []

        conn = self._get_connection()

        try:
            rows = conn.execute(
                """
                SELECT
                    p.xuid,
                    COALESCE(p.gamertag, a.gamertag, p.xuid) AS gamertag,
                    p.team_id,
                    p.rank,
                    p.score,
                    p.kills,
                    p.deaths,
                    p.assists
                FROM shared.match_participants p
                LEFT JOIN shared.xuid_aliases a ON a.xuid = p.xuid
                WHERE p.match_id = ?
                ORDER BY p.rank ASC NULLS LAST
                """,
                [match_id],
            ).fetchall()

            result = []
            for idx, row in enumerate(rows):
                result.append(
                    {
                        "xuid": str(row[0] or "").strip(),
                        "gamertag": str(row[1] or row[0] or "").strip(),
                        "team_id": int(row[2]) if row[2] is not None else None,
                        "rank": int(row[3]) if row[3] is not None else idx + 1,
                        "score": int(row[4]) if row[4] is not None else None,
                        "kills": int(row[5]) if row[5] is not None else 0,
                        "deaths": int(row[6]) if row[6] is not None else 0,
                        "assists": int(row[7]) if row[7] is not None else 0,
                    }
                )
            return result
        except Exception as e:
            logger.debug(f"Erreur load_match_players_stats shared: {e}")
            return []

    def load_match_scoreboard(self, match_id: str) -> list[dict[str, Any]]:
        """Charge le tableau de bord complet de tous les joueurs d'un match.

        Récupère toutes les colonnes de match_participants + le compte de médailles
        Perfect Kill (ID 1512363953) par joueur.

        Args:
            match_id: ID du match.

        Returns:
            Liste de dicts triée par (team_id, rank) avec les champs :
            xuid, gamertag, team_id, rank, score, kills, deaths, assists,
            kda, max_killing_spree, headshot_kills, shots_fired, shots_hit,
            accuracy, melee_kills, power_weapon_kills, damage_dealt,
            damage_taken, avg_life_seconds, perfect_kills.
        """
        if not match_id:
            return []

        conn = self._get_connection()

        try:
            rows = conn.execute(
                """
                SELECT
                    p.xuid,
                    COALESCE(p.gamertag, a.gamertag, p.xuid) AS gamertag,
                    p.team_id,
                    p.rank,
                    p.score,
                    p.kills,
                    p.deaths,
                    p.assists,
                    p.kda,
                    p.max_killing_spree,
                    p.headshot_kills,
                    p.shots_fired,
                    p.shots_hit,
                    p.accuracy,
                    p.melee_kills,
                    p.power_weapon_kills,
                    p.damage_dealt,
                    p.damage_taken,
                    p.avg_life_seconds,
                    COALESCE(pk.perfect_kills, 0) AS perfect_kills
                FROM shared.match_participants p
                LEFT JOIN shared.xuid_aliases a ON a.xuid = p.xuid
                LEFT JOIN (
                    SELECT xuid, SUM(count) AS perfect_kills
                    FROM shared.medals_earned
                    WHERE match_id = ? AND medal_name_id = 1512363953
                    GROUP BY xuid
                ) pk ON pk.xuid = p.xuid
                WHERE p.match_id = ?
                ORDER BY p.team_id ASC NULLS LAST, p.rank ASC NULLS LAST
                """,
                [match_id, match_id],
            ).fetchall()

            result = []
            for idx, row in enumerate(rows):
                result.append(
                    {
                        "xuid": str(row[0] or "").strip(),
                        "gamertag": str(row[1] or row[0] or "").strip(),
                        "team_id": int(row[2]) if row[2] is not None else None,
                        "rank": int(row[3]) if row[3] is not None else idx + 1,
                        "score": int(row[4]) if row[4] is not None else None,
                        "kills": int(row[5]) if row[5] is not None else None,
                        "deaths": int(row[6]) if row[6] is not None else None,
                        "assists": int(row[7]) if row[7] is not None else None,
                        "kda": float(row[8]) if row[8] is not None else None,
                        "max_killing_spree": int(row[9]) if row[9] is not None else None,
                        "headshot_kills": int(row[10]) if row[10] is not None else None,
                        "shots_fired": int(row[11]) if row[11] is not None else None,
                        "shots_hit": int(row[12]) if row[12] is not None else None,
                        "accuracy": float(row[13]) if row[13] is not None else None,
                        "melee_kills": int(row[14]) if row[14] is not None else None,
                        "power_weapon_kills": int(row[15]) if row[15] is not None else None,
                        "damage_dealt": float(row[16]) if row[16] is not None else None,
                        "damage_taken": float(row[17]) if row[17] is not None else None,
                        "avg_life_seconds": float(row[18]) if row[18] is not None else None,
                        "perfect_kills": int(row[19]) if row[19] is not None else 0,
                    }
                )
            return result
        except Exception as e:
            logger.debug(f"Erreur load_match_scoreboard shared: {e}")
            return []
