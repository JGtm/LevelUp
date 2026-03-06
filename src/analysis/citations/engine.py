"""Moteur de calcul et d'agrégation des citations H5G.

Ce module fournit ``CitationEngine``, classe responsable de :

- Charger les règles de mapping depuis ``citation_mappings`` (metadata.duckdb).
- Calculer les citations pour un match donné.
- Agréger les résultats depuis ``match_citations`` (player stats.duckdb).
"""

from __future__ import annotations

import contextlib
import json
import logging
from pathlib import Path
from typing import Any

import duckdb
import polars as pl

from src.analysis.citations._data_loader import CitationDataLoaderMixin
from src.analysis.citations.custom_rules import CUSTOM_FUNCTIONS
from src.utils.paths import get_pve_db_path_from_player

logger = logging.getLogger(__name__)


class CitationEngine(CitationDataLoaderMixin):
    """Moteur de calcul des citations stockées en DuckDB.

    Chaque instance travaille sur la DB d'un joueur donné et charge
    les référentiels depuis ``metadata.duckdb`` (ATTACHé en ``meta``).

    En mode V5, les tables ``medals_earned`` et ``match_stats`` (via
    ``match_participants`` / ``match_registry``) sont lues depuis
    ``shared_matches.duckdb`` (ATTACHé en ``shared``).

    Args:
        db_path: Chemin vers ``stats.duckdb`` du joueur.
        xuid: XUID du joueur.
        metadata_db_path: Chemin vers ``metadata.duckdb``.  Si ``None``,
            dérivé automatiquement de *db_path*.
        shared_db_path: Chemin vers ``shared_matches.duckdb``.  Si ``None``,
            auto-détecté. Passez ``False`` pour désactiver.
        conn: Connexion DuckDB partagée (réutilisée si fournie).
    """

    def __init__(  # noqa: PLR0913
        self,
        db_path: str | Path,
        xuid: str,
        *,
        metadata_db_path: str | Path | None = None,
        shared_db_path: str | Path | None | bool = None,
        conn: duckdb.DuckDBPyConnection | None = None,
    ) -> None:
        self._db_path = Path(db_path)
        self._xuid = xuid
        self._shared_conn = conn  # Connexion partagée (réutilisée si fournie)

        if metadata_db_path is not None:
            self._metadata_db_path = Path(metadata_db_path)
        else:
            # Convention: data/players/{gt}/stats.duckdb → data/warehouse/metadata.duckdb
            self._metadata_db_path = (
                self._db_path.parent.parent.parent / "warehouse" / "metadata.duckdb"
            )

        # Auto-détection shared_matches.duckdb (V5)
        if shared_db_path is False:
            self._shared_db_path: Path | None = None
        elif shared_db_path is not None:
            self._shared_db_path = Path(shared_db_path)  # type: ignore[arg-type]
        else:
            candidate = self._db_path.parent.parent.parent / "warehouse" / "shared_matches.duckdb"
            self._shared_db_path = candidate if candidate.exists() else None

        self._mappings: dict[str, dict[str, Any]] | None = None
        self._attached_shared: bool = False
        self._cached_shared_alias: str | None = None  # Cache pour l'alias shared

        # PVE DB (v5.2)
        pve_candidate = get_pve_db_path_from_player(self._db_path)
        self._pve_db_path: Path | None = pve_candidate if pve_candidate.exists() else None

    # ------------------------------------------------------------------
    # Connexion helper
    # ------------------------------------------------------------------

    def _read_conn(self) -> tuple[duckdb.DuckDBPyConnection, bool]:
        """Retourne une connexion lecture et indique si elle a été créée.

        Si une connexion partagée est disponible, la retourne (owned=False).
        Sinon, ouvre une nouvelle connexion read-only (owned=True) et
        ATTACH ``shared_matches.duckdb`` si disponible.
        """
        if self._shared_conn is not None:
            return self._shared_conn, False
        conn = duckdb.connect(str(self._db_path), read_only=True)
        # ATTACH shared_matches.duckdb pour lecture V5
        if self._shared_db_path is not None and self._shared_db_path.exists():
            try:
                conn.execute(f"ATTACH '{self._shared_db_path}' AS shared (READ_ONLY)")
            except Exception as e:
                err = str(e).lower()
                if "already" not in err and "conflict" not in err:
                    logger.debug("Impossible d'attacher shared: %s", e)
        return conn, True

    @property
    def has_shared(self) -> bool:
        """Indique si shared_matches.duckdb est configuré et existe."""
        return self._shared_db_path is not None and self._shared_db_path.exists()

    def _conn_has_shared(self, conn: duckdb.DuckDBPyConnection) -> bool:
        """Vérifie si la connexion a une base shared attachée (détection dynamique)."""
        return self._get_shared_alias(conn) is not None

    def _get_shared_alias(self, conn: duckdb.DuckDBPyConnection) -> str | None:
        """Retourne l'alias de la base shared attachée (avec cache)."""
        if self._cached_shared_alias is not None:
            return self._cached_shared_alias
        try:
            dbs = conn.execute("SELECT database_name, path FROM duckdb_databases()").fetchall()
            for db_name, db_path_val in dbs:
                if db_path_val and "shared_matches.duckdb" in str(db_path_val).lower():
                    self._cached_shared_alias = db_name
                    return db_name
                if db_name and "shared" in db_name.lower():
                    # Vérifier que cette DB a match_participants
                    try:
                        conn.execute(f"SELECT 1 FROM {db_name}.match_participants LIMIT 1")
                        self._cached_shared_alias = db_name
                        return db_name
                    except Exception:
                        continue
            return None
        except Exception:
            return None

    def _shared_has_table(self, conn: duckdb.DuckDBPyConnection, table_name: str) -> bool:
        """Vérifie si une table existe dans le catalog shared (alias dynamique)."""
        shared_alias = self._get_shared_alias(conn)
        if not shared_alias:
            return False
        try:
            conn.execute(f"SELECT 1 FROM {shared_alias}.{table_name} LIMIT 1")
            return True
        except Exception:
            return False

    # ------------------------------------------------------------------
    # Mappings
    # ------------------------------------------------------------------

    def load_mappings(self) -> dict[str, dict[str, Any]]:
        """Charge les mappings depuis ``citation_mappings`` dans metadata.duckdb.

        Returns:
            Dict ``{citation_name_norm: {mapping fields…}}``.
        """
        if self._mappings is not None:
            return self._mappings

        meta_path = self._metadata_db_path
        if not meta_path.exists():
            logger.warning("metadata.duckdb introuvable : %s", meta_path)
            self._mappings = {}
            return self._mappings

        conn = duckdb.connect(str(meta_path), read_only=True)
        try:
            # Vérifier que la table existe
            exists = conn.execute(
                "SELECT COUNT(*) FROM information_schema.tables "
                "WHERE table_name = 'citation_mappings'"
            ).fetchone()[0]
            if not exists:
                logger.warning("Table citation_mappings absente dans %s", meta_path)
                self._mappings = {}
                return self._mappings

            rows = conn.execute(
                "SELECT citation_name_norm, citation_name_display, mapping_type, "
                "medal_id, medal_ids, stat_name, award_name, award_category, "
                "custom_function, composite_children, confidence, notes, "
                "image_path, category, description, tier_targets "
                "FROM citation_mappings "
                "WHERE enabled IS NOT FALSE"
            ).fetchall()

            columns = [
                "citation_name_norm",
                "citation_name_display",
                "mapping_type",
                "medal_id",
                "medal_ids",
                "stat_name",
                "award_name",
                "award_category",
                "custom_function",
                "composite_children",
                "confidence",
                "notes",
                "image_path",
                "category",
                "description",
                "tier_targets",
            ]
            self._mappings = {}
            for row in rows:
                d = dict(zip(columns, row, strict=False))
                norm = d.pop("citation_name_norm")
                self._mappings[norm] = d
            return self._mappings
        finally:
            conn.close()

    # ------------------------------------------------------------------
    # Calcul par match
    # ------------------------------------------------------------------

    def compute_citation_for_match(  # noqa: C901, PLR0912, PLR0913
        self,
        mapping: dict[str, Any],
        *,
        match_medals: dict[int, int] | None = None,
        match_stats: dict[str, Any] | None = None,
        match_awards: dict[str, int] | None = None,
        df_match: pl.DataFrame | None = None,
        highlight_events: list[tuple[int, str]] | None = None,
    ) -> int:
        """Calcule la valeur d'une citation pour un match.

        Args:
            mapping: Dictionnaire de mapping (issu de ``load_mappings``).
            match_medals: ``{medal_name_id: count}`` pour ce match.
            match_stats: Dict des stats du match (kills, deaths, …).
            match_awards: ``{award_name: count}`` pour ce match.
            df_match: Ligne DataFrame du match (pour fonctions custom).

        Returns:
            Valeur calculée (0 si non applicable).
        """
        mtype = mapping.get("mapping_type", "")

        # Les composites ne sont pas calculées par match
        if mtype == "composite":
            return 0

        if mtype == "medal":
            # Support multi-médailles via medal_ids (comma-separated)
            medal_ids_str = mapping.get("medal_ids")
            if medal_ids_str and match_medals:
                total = 0
                for mid_str in str(medal_ids_str).split(","):
                    mid_str = mid_str.strip()
                    if mid_str:
                        with contextlib.suppress(ValueError):
                            total += match_medals.get(int(mid_str), 0)
                return total
            # Fallback: single medal_id
            medal_id = mapping.get("medal_id")
            if medal_id is not None and match_medals:
                return match_medals.get(int(medal_id), 0)
            return 0

        if mtype in ("stat", "pve_stat"):
            stat_name = mapping.get("stat_name", "")
            if stat_name and match_stats:
                try:
                    return int(match_stats.get(stat_name, 0) or 0)
                except (TypeError, ValueError):
                    return 0
            return 0

        if mtype == "award":
            award_name = mapping.get("award_name", "")
            if award_name and match_awards:
                return match_awards.get(award_name, 0)
            return 0

        if mtype == "custom":
            func_name = mapping.get("custom_function", "")
            func = CUSTOM_FUNCTIONS.get(func_name)
            if func is None:
                logger.warning("Fonction custom introuvable : %s", func_name)
                return 0
            try:
                return func(df=df_match, awards=match_awards, highlight_events=highlight_events)
            except TypeError:
                # Certaines fonctions n'acceptent que df
                try:
                    return func(df_match) if df_match is not None else 0
                except Exception:
                    return 0

        return 0

    def compute_all_for_match(  # noqa: PLR0913
        self,
        match_id: str,  # noqa: ARG002 — conservé pour cohérence API
        *,
        match_medals: dict[int, int] | None = None,
        match_stats: dict[str, Any] | None = None,
        match_awards: dict[str, int] | None = None,
        df_match: pl.DataFrame | None = None,
        highlight_events: list[tuple[int, str]] | None = None,
    ) -> dict[str, int]:
        """Calcule toutes les citations pour un match.

        Returns:
            Dict sparse ``{citation_name_norm: value}`` (valeurs > 0 uniquement).
        """
        mappings = self.load_mappings()
        results: dict[str, int] = {}

        for norm_name, mapping in mappings.items():
            value = self.compute_citation_for_match(
                mapping,
                match_medals=match_medals,
                match_stats=match_stats,
                match_awards=match_awards,
                df_match=df_match,
                highlight_events=highlight_events,
            )
            if value > 0:
                results[norm_name] = value

        return results

    # ------------------------------------------------------------------
    # Agrégation depuis match_citations
    # ------------------------------------------------------------------

    def aggregate_citations(
        self,
        citation_names: list[str] | None = None,
        match_ids: list[str] | None = None,
    ) -> dict[str, int]:
        """Agrège les valeurs depuis la table ``match_citations``.

        Args:
            citation_names: Noms normalisés à agréger. ``None`` = tous.
            match_ids: Filtrer par matchs. ``None`` = tous.

        Returns:
            ``{citation_name_norm: total}``.
        """
        if not self._db_path.exists() and self._shared_conn is None:
            return {}

        conn, owned = self._read_conn()
        try:
            # Vérifier que la table existe
            exists = conn.execute(
                "SELECT COUNT(*) FROM information_schema.tables "
                "WHERE table_name = 'match_citations'"
            ).fetchone()[0]
            if not exists:
                return {}

            # Construire la requête dynamiquement
            conditions: list[str] = []
            params: list[Any] = []

            if citation_names is not None:
                placeholders = ", ".join(["?"] * len(citation_names))
                conditions.append(f"citation_name_norm IN ({placeholders})")
                params.extend(citation_names)

            if match_ids is not None:
                placeholders = ", ".join(["?"] * len(match_ids))
                conditions.append(f"match_id IN ({placeholders})")
                params.extend(match_ids)

            where = ("WHERE " + " AND ".join(conditions)) if conditions else ""

            rows = conn.execute(
                f"SELECT citation_name_norm, SUM(value) as total "
                f"FROM match_citations "
                f"{where} "
                f"GROUP BY citation_name_norm",
                params,
            ).fetchall()

            return {row[0]: int(row[1]) for row in rows}
        finally:
            if owned:
                conn.close()

    # ------------------------------------------------------------------
    # Méthode haut-niveau : calcul complet pour un match
    # ------------------------------------------------------------------

    def compute_and_store_for_match(
        self,
        match_id: str,
        *,
        conn: duckdb.DuckDBPyConnection | None = None,
    ) -> int:
        """Charge les données, calcule et insère les citations pour 1 match.

        Args:
            match_id: Identifiant du match.
            conn: Connexion ouverte en écriture (optionnelle, en crée une sinon).

        Returns:
            Nombre de citations insérées (inclut le marqueur _processed).
        """
        # Charger les données nécessaires
        match_medals = self.load_match_medals(match_id)
        match_stats = self.load_match_stats(match_id)
        match_awards = self.load_match_awards(match_id)
        df_match = self.load_match_df(match_id)
        highlight_events = self.load_match_highlight_events(match_id)

        # Fusionner les stats PVE dans match_stats (v5.2) pour les citations pve_stat
        pve_stats = self.load_match_pve_stats(match_id)
        if pve_stats:
            match_stats = {**match_stats, **pve_stats}

        # Calculer les citations
        citations = self.compute_all_for_match(
            match_id,
            match_medals=match_medals,
            match_stats=match_stats,
            match_awards=match_awards,
            df_match=df_match,
            highlight_events=highlight_events,
        )

        # Insérer dans match_citations (même si vide, on marque comme traité)
        own_conn = conn is None
        if own_conn:
            if self._shared_conn is not None:
                conn = self._shared_conn
                own_conn = False
            else:
                conn = duckdb.connect(str(self._db_path))

        try:
            # Insérer les citations calculées
            for norm_name, value in citations.items():
                conn.execute(
                    "INSERT OR REPLACE INTO match_citations "
                    "(match_id, citation_name_norm, value) VALUES (?, ?, ?)",
                    [match_id, norm_name, value],
                )

            # Marquer le match comme traité (même si 0 citations)
            conn.execute(
                "INSERT OR REPLACE INTO match_citations "
                "(match_id, citation_name_norm, value) VALUES (?, '_processed', 1)",
                [match_id],
            )
            return len(citations) + 1  # +1 pour le marqueur
        finally:
            if own_conn:
                conn.close()

    # ------------------------------------------------------------------
    # Méthode haut-niveau : agrégation compatible UI
    # ------------------------------------------------------------------

    def aggregate_for_display(
        self,
        match_ids: list[str] | None = None,
    ) -> dict[str, int]:
        """Agrège toutes les citations pour affichage UI.

        Calcule aussi les citations composites (évaluation de la maîtrise
        des sous-citations).

        Args:
            match_ids: Filtrer par matchs. ``None`` = tous.

        Returns:
            ``{citation_name_norm: total}``.
        """
        result = self.aggregate_citations(citation_names=None, match_ids=match_ids)

        # Calculer les citations composites
        mappings = self.load_mappings()
        for norm_name, mapping in mappings.items():
            if mapping.get("mapping_type") != "composite":
                continue
            children_raw = mapping.get("composite_children")
            if not children_raw:
                continue
            try:
                children = (
                    json.loads(children_raw) if isinstance(children_raw, str) else children_raw
                )
            except (json.JSONDecodeError, TypeError):
                continue
            if not isinstance(children, list):
                continue

            # Compter les sous-citations qui ont atteint la maîtrise
            mastered_count = 0
            for child in children:
                if child not in mappings:
                    continue
                child_count = result.get(child, 0)
                if child_count <= 0:
                    continue
                child_tiers = mappings[child].get("tier_targets")
                if not child_tiers:
                    # Pas de tiers → considérer masterisé si > 0
                    mastered_count += 1
                    continue
                targets = sorted(
                    int(t.strip())
                    for t in str(child_tiers).split(",")
                    if t.strip().isdigit()
                )
                if targets and child_count >= targets[-1]:
                    mastered_count += 1
            if mastered_count > 0:
                result[norm_name] = mastered_count

        return result
