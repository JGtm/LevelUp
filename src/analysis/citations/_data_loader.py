"""Mixin de chargement des données match pour CitationEngine.

Regroupe les méthodes ``load_match_*`` qui récupèrent les données brutes
d'un match depuis DuckDB (shared ou local) pour le calcul des citations.
"""

from __future__ import annotations

import logging
from typing import Any

import polars as pl

from src.utils.db import duckdb_read_only

logger = logging.getLogger(__name__)


class CitationDataLoaderMixin:
    """Fournit les méthodes de chargement de données match.

    Ce mixin attend que la classe hôte expose :
    - ``_db_path`` (``Path``)
    - ``_xuid`` (``str``)
    - ``_pve_db_path`` (``Path | None``)
    - ``_shared_conn`` (``DuckDBPyConnection | None``)
    - ``_read_conn()`` → context manager yielding ``conn``
    - ``_get_shared_alias(conn)`` → ``str | None``
    - ``_shared_has_table(conn, table)`` → ``bool``
    """

    # ------------------------------------------------------------------
    # Médailles
    # ------------------------------------------------------------------

    def load_match_medals(self, match_id: str) -> dict[int, int]:
        """Charge les médailles d'un match.

        En V5, lit depuis ``shared.medals_earned`` (filtré par xuid).
        Sinon, lit depuis la table locale ``medals_earned``.

        Returns:
            ``{medal_name_id: count}``.
        """
        if not self._db_path.exists() and self._shared_conn is None:
            return {}

        with self._read_conn() as conn:
            try:
                shared_alias = self._get_shared_alias(conn)
                if shared_alias and self._shared_has_table(conn, "medals_earned"):
                    rows = conn.execute(
                        f"SELECT medal_name_id, count FROM {shared_alias}.medals_earned "
                        "WHERE match_id = ? AND xuid = ?",
                        [match_id, self._xuid],
                    ).fetchall()
                    return {int(row[0]): int(row[1]) for row in rows}

                rows = conn.execute(
                    "SELECT medal_name_id, count FROM medals_earned WHERE match_id = ?",
                    [match_id],
                ).fetchall()
                return {int(row[0]): int(row[1]) for row in rows}
            except Exception:
                return {}

    # ------------------------------------------------------------------
    # Stats match (participants + registry)
    # ------------------------------------------------------------------

    def load_match_stats(self, match_id: str) -> dict[str, Any]:
        """Charge les stats d'un match.

        En V5, joint ``shared.match_participants`` et ``shared.match_registry``.
        Sinon, lit depuis la table locale ``match_stats``.

        Returns:
            Dict avec les colonnes de stats pour ce match.
        """
        if not self._db_path.exists() and self._shared_conn is None:
            return {}

        with self._read_conn() as conn:
            try:
                shared_alias = self._get_shared_alias(conn)
                if shared_alias and self._shared_has_table(conn, "match_participants"):
                    result = conn.execute(
                        f"SELECT p.*, r.map_name, r.playlist_name, r.game_variant_name, "
                        f"r.start_time "
                        f"FROM {shared_alias}.match_participants p "
                        f"LEFT JOIN {shared_alias}.match_registry r ON p.match_id = r.match_id "
                        f"WHERE p.match_id = ? AND p.xuid = ?",
                        [match_id, self._xuid],
                    )
                    row = result.fetchone()
                    if row is not None:
                        columns = [desc[0] for desc in result.description]
                        return dict(zip(columns, row, strict=False))

                result = conn.execute(
                    "SELECT * FROM match_stats WHERE match_id = ?",
                    [match_id],
                )
                row = result.fetchone()
                if row is None:
                    return {}
                columns = [desc[0] for desc in result.description]
                return dict(zip(columns, row, strict=False))
            except Exception:
                return {}

    # ------------------------------------------------------------------
    # Stats PVE (Firefight)
    # ------------------------------------------------------------------

    def load_match_pve_stats(self, match_id: str) -> dict[str, Any]:
        """Charge les stats PVE du joueur pour un match depuis ``shared_pve.duckdb`` (v5.2).

        Filtre par ``xuid`` pour retourner les stats du joueur courant uniquement.

        Returns:
            Dict ``{column_name: value}`` pour ce match ou ``{}`` si absent.
        """
        if self._pve_db_path is None:
            return {}
        try:
            with duckdb_read_only(self._pve_db_path) as pve_conn:
                result = pve_conn.execute(
                    "SELECT * FROM pve_match_stats WHERE match_id = ? AND xuid = ? LIMIT 1",
                    [match_id, self._xuid],
                )
                row = result.fetchone()
                if row is None:
                    return {}
                columns = [desc[0] for desc in result.description]
                return dict(zip(columns, row, strict=False))
        except Exception:
            return {}

    # ------------------------------------------------------------------
    # Awards (PersonalScores)
    # ------------------------------------------------------------------

    def load_match_awards(self, match_id: str) -> dict[str, int]:
        """Charge les awards d'un match depuis ``personal_score_awards``.

        Returns:
            ``{award_name: total_count}``.
        """
        if not self._db_path.exists() and self._shared_conn is None:
            return {}

        with self._read_conn() as conn:
            try:
                exists = conn.execute(
                    "SELECT COUNT(*) FROM information_schema.tables "
                    "WHERE table_name = 'personal_score_awards'"
                ).fetchone()[0]
                if not exists:
                    return {}

                rows = conn.execute(
                    "SELECT award_name, SUM(award_count) "
                    "FROM personal_score_awards "
                    "WHERE match_id = ? "
                    "GROUP BY award_name",
                    [match_id],
                ).fetchall()
                return {str(row[0]): int(row[1]) for row in rows}
            except Exception:
                return {}

    # ------------------------------------------------------------------
    # DataFrame match (pour fonctions custom)
    # ------------------------------------------------------------------

    def load_match_df(self, match_id: str) -> pl.DataFrame:  # noqa: PLR0912
        """Charge un match comme DataFrame Polars (1 ligne).

        En V5, joint ``shared.match_participants`` + ``shared.match_registry``.
        Utile pour les fonctions custom qui attendent un DataFrame.
        """
        if not self._db_path.exists() and self._shared_conn is None:
            return pl.DataFrame()

        with self._read_conn() as conn:
            try:
                shared_alias = self._get_shared_alias(conn)
                if shared_alias and self._shared_has_table(conn, "match_participants"):
                    result = conn.execute(
                        f"SELECT p.*, r.map_name, r.playlist_name, r.game_variant_name, "
                        f"r.start_time "
                        f"FROM {shared_alias}.match_participants p "
                        f"LEFT JOIN {shared_alias}.match_registry r ON p.match_id = r.match_id "
                        f"WHERE p.match_id = ? AND p.xuid = ?",
                        [match_id, self._xuid],
                    )
                    try:
                        df = result.pl()
                        if len(df) > 0:
                            return df
                    except Exception:
                        columns = [desc[0] for desc in result.description]
                        rows = result.fetchall()
                        if rows:
                            return pl.DataFrame(
                                {col: [row[i] for row in rows] for i, col in enumerate(columns)}
                            )

                result = conn.execute(
                    "SELECT * FROM match_stats WHERE match_id = ?",
                    [match_id],
                )
                try:
                    return result.pl()
                except Exception:
                    columns = [desc[0] for desc in result.description]
                    rows = result.fetchall()
                    if not rows:
                        return pl.DataFrame()
                    return pl.DataFrame(
                        {col: [row[i] for row in rows] for i, col in enumerate(columns)}
                    )
            except Exception:
                return pl.DataFrame()

    # ------------------------------------------------------------------
    # Highlight events
    # ------------------------------------------------------------------

    def load_match_highlight_events(self, match_id: str) -> list[tuple[int, str]]:
        """Charge les highlight_events (mode + death) pour un match.

        Retourne une liste de tuples ``(time_ms, event_type)`` triés par temps,
        filtré par le xuid du joueur.

        Returns:
            Liste de ``(time_ms, event_type)`` triée.
        """
        if not self._db_path.exists() and self._shared_conn is None:
            return []

        with self._read_conn() as conn:
            try:
                shared_alias = self._get_shared_alias(conn)
                if shared_alias and self._shared_has_table(conn, "highlight_events"):
                    rows = conn.execute(
                        f"SELECT time_ms, event_type "
                        f"FROM {shared_alias}.highlight_events "
                        f"WHERE match_id = ? AND xuid = ? "
                        f"  AND event_type IN ('mode', 'death') "
                        f"ORDER BY time_ms",
                        [match_id, self._xuid],
                    ).fetchall()
                    return [(int(r[0]), str(r[1])) for r in rows]
                return []
            except Exception:
                return []

    # ------------------------------------------------------------------
    # Weapon kills (v5.5)
    # ------------------------------------------------------------------

    def load_match_weapon_kills(self, match_id: str) -> dict[str, int]:
        """Charge les kills par arme du joueur courant pour un match.

        Interroge ``shared.weapon_kills`` (schéma per-kill v5.6) et agrège
        par ``weapon_name`` en excluant les kills non attribués.

        Returns:
            ``{weapon_name: kills}`` ou ``{}`` si absent.
        """
        with self._read_conn() as conn:
            try:
                shared_alias = self._get_shared_alias(conn)
                if not shared_alias:
                    return {}
                rows = conn.execute(
                    f"SELECT weapon_name, COUNT(*) AS kills "
                    f"FROM {shared_alias}.weapon_kills "
                    "WHERE match_id = ? AND xuid = ? "
                    "AND weapon_name NOT IN ('MELEE', 'GRENADE', 'UNKNOWN', 'NON TROUVE') "
                    "GROUP BY weapon_name",
                    [match_id, self._xuid],
                ).fetchall()
                # Appliquer les fusions de variantes avant de retourner
                from src.analysis._weapon_data import WEAPON_FUSION_MAP

                result: dict[str, int] = {}
                for weapon_name, kills in rows:
                    canonical = WEAPON_FUSION_MAP.get(str(weapon_name), str(weapon_name))
                    result[canonical] = result.get(canonical, 0) + int(kills)
                return result
            except Exception:
                return {}
