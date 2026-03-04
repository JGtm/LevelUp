"""
Mixin pour la gestion des archives Parquet (cold storage).

Regroupe les méthodes d'archives extraites de DuckDBRepository :
- _get_archive_dir
- get_archive_info
- load_matches_from_archives
- load_all_matches_unified
- get_total_match_count_with_archives
"""

from __future__ import annotations

import json
import logging
import re
from datetime import datetime
from pathlib import Path
from typing import Any

import duckdb

from src.data.domain.models.stats import MatchRow

logger = logging.getLogger(__name__)


class ArchivesMixin:
    """Méthodes DuckDBRepository relatives aux archives Parquet."""

    # =========================================================================
    # Archives (Sprint 4.5 - Partitionnement Temporel)
    # =========================================================================

    def _get_archive_dir(self) -> Path:
        """Retourne le chemin vers le dossier archive du joueur.

        Dans le layout standard, les archives sont stockées dans un dossier
        `archive/` à côté de `stats.duckdb`.

        Pour certains scénarios (notamment des fixtures de tests), le dossier
        joueur peut être suffixé par un identifiant (ex: `TestPlayer_ab12cd34`).
        Dans ce cas, on tente aussi de résoudre `players/<base>/archive`.
        """

        archive_dir = self._player_db_path.parent / "archive"
        if archive_dir.exists():
            return archive_dir

        player_dir_name = self._player_db_path.parent.name
        m = re.match(r"^(?P<base>.+)_[0-9a-f]{8}$", player_dir_name)
        if m:
            base_dir = m.group("base")
            alternative = self._player_db_path.parent.parent / base_dir / "archive"
            if alternative.exists():
                return alternative

        return archive_dir

    def get_archive_info(self) -> dict[str, Any]:
        """Retourne les informations sur les archives existantes.

        Returns:
            Dict avec:
                - has_archives: bool
                - archive_count: int
                - total_size_mb: float
                - archives: list[dict] avec détails de chaque fichier
                - last_updated: str (datetime ISO) ou None
        """
        archive_dir = self._get_archive_dir()

        if not archive_dir.exists():
            return {
                "has_archives": False,
                "archive_count": 0,
                "total_size_mb": 0.0,
                "archives": [],
                "last_updated": None,
            }

        parquet_files = list(archive_dir.glob("*.parquet"))

        if not parquet_files:
            return {
                "has_archives": False,
                "archive_count": 0,
                "total_size_mb": 0.0,
                "archives": [],
                "last_updated": None,
            }

        archives = []
        total_size = 0

        for pf in sorted(parquet_files):
            size = pf.stat().st_size
            total_size += size

            # Essayer de lire le nombre de lignes via DuckDB
            try:
                with duckdb.connect(":memory:") as conn:
                    count = conn.execute(f"SELECT COUNT(*) FROM read_parquet('{pf}')").fetchone()[0]
            except Exception:
                count = None

            archives.append(
                {
                    "name": pf.name,
                    "size_mb": round(size / (1024 * 1024), 2),
                    "row_count": count,
                    "modified_at": datetime.fromtimestamp(pf.stat().st_mtime).isoformat(),
                }
            )

        # Charger l'index si disponible
        index_file = archive_dir / "archive_index.json"
        last_updated = None
        if index_file.exists():
            try:
                with open(index_file, encoding="utf-8") as f:
                    index = json.load(f)
                    last_updated = index.get("last_updated")
            except Exception:
                pass

        return {
            "has_archives": True,
            "archive_count": len(archives),
            "total_size_mb": round(total_size / (1024 * 1024), 2),
            "archives": archives,
            "last_updated": last_updated,
        }

    def load_matches_from_archives(
        self,
        *,
        start_date: datetime | None = None,
        end_date: datetime | None = None,
    ) -> list[MatchRow]:
        """Charge les matchs depuis les fichiers Parquet archivés.

        Args:
            start_date: Date de début (incluse).
            end_date: Date de fin (exclue).

        Returns:
            Liste de MatchRow depuis les archives.
        """
        archive_dir = self._get_archive_dir()

        if not archive_dir.exists():
            return []

        parquet_files = list(archive_dir.glob("*.parquet"))

        if not parquet_files:
            return []

        # Construire la liste des fichiers à lire
        file_paths = [str(pf) for pf in sorted(parquet_files)]

        # Utiliser DuckDB pour lire les Parquet
        # Construire la clause WHERE
        where_clauses = []
        params = []

        if start_date:
            where_clauses.append("start_time >= ?")
            params.append(start_date)

        if end_date:
            where_clauses.append("start_time < ?")
            params.append(end_date)

        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"

        # Lire depuis tous les fichiers Parquet
        # DuckDB peut lire plusieurs fichiers avec read_parquet([...])
        file_list_sql = ", ".join([f"'{f}'" for f in file_paths])

        try:
            with duckdb.connect(":memory:") as conn:
                sql = f"""
                    SELECT
                        match_id, start_time, map_id, map_name,
                        playlist_id, playlist_name, pair_id, pair_name,
                        game_variant_id, game_variant_name,
                        outcome, team_id, kda, max_killing_spree, headshot_kills,
                        avg_life_seconds, time_played_seconds,
                        kills, deaths, assists, accuracy,
                        my_team_score, enemy_team_score, team_mmr, enemy_mmr
                    FROM read_parquet([{file_list_sql}])
                    WHERE {where_sql}
                    ORDER BY start_time ASC
                """

                result = conn.execute(sql, params) if params else conn.execute(sql)
                rows = result.fetchall()
                columns = [desc[0] for desc in result.description]

            return [
                MatchRow(
                    match_id=row[columns.index("match_id")],
                    start_time=row[columns.index("start_time")],
                    map_id=row[columns.index("map_id")],
                    map_name=row[columns.index("map_name")],
                    playlist_id=row[columns.index("playlist_id")],
                    playlist_name=row[columns.index("playlist_name")],
                    map_mode_pair_id=row[columns.index("pair_id")],
                    map_mode_pair_name=row[columns.index("pair_name")],
                    game_variant_id=row[columns.index("game_variant_id")],
                    game_variant_name=row[columns.index("game_variant_name")],
                    outcome=row[columns.index("outcome")],
                    last_team_id=row[columns.index("team_id")],
                    kda=row[columns.index("kda")],
                    max_killing_spree=row[columns.index("max_killing_spree")],
                    headshot_kills=row[columns.index("headshot_kills")],
                    average_life_seconds=row[columns.index("avg_life_seconds")],
                    time_played_seconds=row[columns.index("time_played_seconds")],
                    kills=row[columns.index("kills")] or 0,
                    deaths=row[columns.index("deaths")] or 0,
                    assists=row[columns.index("assists")] or 0,
                    accuracy=row[columns.index("accuracy")],
                    my_team_score=row[columns.index("my_team_score")],
                    enemy_team_score=row[columns.index("enemy_team_score")],
                    team_mmr=row[columns.index("team_mmr")],
                    enemy_mmr=row[columns.index("enemy_mmr")],
                    personal_score=row[columns.index("personal_score")]
                    if "personal_score" in columns
                    else None,
                )
                for row in rows
            ]
        except Exception as e:
            logger.warning(f"Erreur lecture archives: {e}")
            return []

    def load_all_matches_unified(
        self,
        *,
        include_archives: bool = True,
        start_date: datetime | None = None,
        end_date: datetime | None = None,
    ) -> list[MatchRow]:
        """Charge tous les matchs (DB principale + archives).

        Fournit une vue unifiée de l'historique complet du joueur,
        combinant la DB principale (données récentes) et les archives
        Parquet (données anciennes).

        Args:
            include_archives: Si True, inclut les matchs des archives.
            start_date: Date de début (incluse).
            end_date: Date de fin (exclue).

        Returns:
            Liste de MatchRow triée par start_time (chronologique).
        """
        matches: list[MatchRow] = []

        # 1. Charger depuis les archives (si demandé)
        if include_archives:
            archive_matches = self.load_matches_from_archives(
                start_date=start_date,
                end_date=end_date,
            )
            matches.extend(archive_matches)
            logger.debug(f"Archives: {len(archive_matches)} matchs chargés")

        # 2. Charger depuis la DB principale
        db_matches = self.load_matches()

        # Filtrer par dates si nécessaire
        if start_date or end_date:
            filtered = []
            for m in db_matches:
                if start_date and m.start_time < start_date:
                    continue
                if end_date and m.start_time >= end_date:
                    continue
                filtered.append(m)
            db_matches = filtered

        matches.extend(db_matches)
        logger.debug(f"DB principale: {len(db_matches)} matchs chargés")

        # 3. Dédupliquer (au cas où un match serait dans les deux)
        seen_ids: set[str] = set()
        unique_matches: list[MatchRow] = []

        for m in matches:
            if m.match_id not in seen_ids:
                seen_ids.add(m.match_id)
                unique_matches.append(m)

        # 4. Trier par date
        unique_matches.sort(key=lambda m: m.start_time)

        logger.debug(f"Total unifié: {len(unique_matches)} matchs")
        return unique_matches

    def get_total_match_count_with_archives(self) -> dict[str, int]:
        """Retourne le compte des matchs (DB + archives).

        Returns:
            Dict avec 'db_count', 'archive_count', 'total'.
        """
        db_count = self.get_match_count()

        archive_count = 0
        archive_info = self.get_archive_info()

        if archive_info["has_archives"]:
            for archive in archive_info["archives"]:
                if archive.get("row_count"):
                    archive_count += archive["row_count"]

        return {
            "db_count": db_count,
            "archive_count": archive_count,
            "total": db_count + archive_count,
        }
