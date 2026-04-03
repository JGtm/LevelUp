"""Service Coéquipiers — agrégats pour la page teammates.

Encapsule les calculs lourds : chargement stats coéquipiers, enrichissement
perfect kills, profils radar de complémentarité, et impact/taquinerie.

Contrat : les pages UI appellent ces fonctions, jamais de calcul inline.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import polars as pl

from src.data._score_sql import NORM_ENEMY_TEAM_SCORE_SQL, NORM_MY_TEAM_SCORE_SQL
from src.data.services._teammates_impact_queries import _collect_impact_data
from src.data.services._teammates_perf_queries import load_perf_enrichment_with_session

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class TeammateStats:
    """Stats agrégées d'un coéquipier pour un ensemble de matchs."""

    gamertag: str
    """Gamertag du coéquipier."""
    df: pl.DataFrame
    """DataFrame Polars des stats sur les matchs communs."""
    is_empty: bool
    """True si aucune donnée trouvée."""


@dataclass(frozen=True)
class EnrichedSeries:
    """Série (nom, DataFrame) enrichie avec perfect_kills."""

    series: list[tuple[str, pl.DataFrame]]
    """Liste de (gamertag, df Polars) avec colonne perfect_kills ajoutée."""


@dataclass(frozen=True)
class ParticipationProfile:
    """Profil de participation radar (6 axes) pour un joueur."""

    name: str
    color: str
    values: dict[str, float]
    """Mapping axe → valeur normalisée."""


@dataclass(frozen=True)
class ImpactData:
    """Données d'impact et taquinerie pour un groupe de joueurs."""

    first_bloods: dict[str, list[str]]
    clutch_finishers: dict[str, list[str]]
    last_casualties: dict[str, list[str]]
    scores: dict[str, float]
    gamertags: list[str]
    match_ids: list[str]
    available: bool
    """True si des événements d'impact ont été trouvés."""


def _resolve_xuid_from_shared(conn: object, gamertag: str) -> str | None:
    """Résout gamertag → xuid via shared.v_gamertag_lookup (vue centralisée v6)."""
    try:
        row = conn.execute(  # type: ignore[union-attr]
            "SELECT xuid FROM shared.v_gamertag_lookup WHERE LOWER(gamertag) = LOWER(?) LIMIT 1",
            [gamertag],
        ).fetchone()
        if row:
            return str(row[0]).strip()
    except Exception:
        pass
    # Fallback pré-vue : xuid_aliases puis match_participants
    for query, params in [
        (
            "SELECT xuid FROM shared.xuid_aliases WHERE LOWER(gamertag) = LOWER(?) LIMIT 1",
            [gamertag],
        ),
        (
            "SELECT DISTINCT xuid FROM shared.match_participants WHERE LOWER(gamertag) = LOWER(?) LIMIT 1",
            [gamertag],
        ),
    ]:
        try:
            row = conn.execute(query, params).fetchone()  # type: ignore[union-attr]
            if row:
                return str(row[0]).strip()
        except Exception:
            pass
    return None


def _query_teammate_shared_stats(
    conn: object, xuid: str, match_ids_list: list[str]
) -> pl.DataFrame:
    """Charge les stats d'un coéquipier depuis shared.match_participants + v_match_full."""
    from src.data.repositories._arrow_bridge import result_to_polars

    placeholders = ", ".join(["?" for _ in match_ids_list])
    query = f"""
        SELECT
            p.match_id, r.start_time, r.map_id,
            COALESCE(r.map_name, '') AS map_name,
            COALESCE(r.map_name_fr, r.map_name, '') AS map_name_fr,
            r.playlist_id, COALESCE(r.playlist_name, '') AS playlist_name,
            r.pair_id, COALESCE(r.pair_name, '') AS pair_name,
            r.game_variant_id, COALESCE(r.game_variant_name, '') AS game_variant_name,
            p.outcome, p.team_id,
            p.kda AS ratio,
            COALESCE(p.max_killing_spree, 0) AS max_killing_spree,
            COALESCE(p.headshot_kills, 0) AS headshot_kills,
            COALESCE(p.avg_life_seconds, 0) AS average_life_seconds,
            COALESCE(r.duration_seconds, 0) AS time_played_seconds,
            COALESCE(p.kills, 0) AS kills, COALESCE(p.deaths, 0) AS deaths,
            COALESCE(p.assists, 0) AS assists,
            CASE WHEN p.shots_fired > 0
                THEN CAST(p.shots_hit AS FLOAT) * 100.0 / CAST(p.shots_fired AS FLOAT)
                ELSE NULL
            END AS accuracy,
            {NORM_MY_TEAM_SCORE_SQL} AS my_team_score,
            {NORM_ENEMY_TEAM_SCORE_SQL} AS enemy_team_score,
            p.score AS personal_score,
            COALESCE(r.is_firefight, FALSE) AS is_firefight,
            COALESCE(r.is_ranked, FALSE) AS is_ranked,
            p.xuid,
            p.team_mmr AS team_mmr,
            p.enemy_mmr AS enemy_mmr,
            CASE WHEN COALESCE(r.duration_seconds, 0) > 0
                THEN CAST(COALESCE(p.kills, 0) AS FLOAT) * 60.0 / CAST(r.duration_seconds AS FLOAT)
                ELSE NULL
            END AS kills_per_min
        FROM shared.match_participants p
        JOIN shared.v_match_full r ON p.match_id = r.match_id
        WHERE p.xuid = ?
          AND p.match_id IN ({placeholders})
        ORDER BY r.start_time DESC
    """
    result = conn.execute(query, [xuid, *match_ids_list])
    return result_to_polars(result)


def _load_xuid_from_db(db_path: str) -> str:
    """Charge le xuid depuis la table sync_meta de la DB."""
    from src.utils.db import duckdb_read_only

    try:
        with duckdb_read_only(db_path) as conn:
            result = conn.execute(
                "SELECT value FROM sync_meta WHERE key = 'xuid' LIMIT 1"
            ).fetchone()
        if result:
            return str(result[0]).strip()
    except Exception as e:
        logger.debug("Impossible de charger le xuid depuis %s: %s", db_path, e)
    return ""


def _resolve_xuid_for_enrichment(
    name: str, idx: int, df: pl.DataFrame, db_path: str, base_dir: Path
) -> str:
    """Résout le xuid d'un joueur pour l'enrichissement perfect_kills."""
    # 1. Depuis le DataFrame (colonne xuid)
    if "xuid" in df.columns:
        xuid_values = df["xuid"].unique()
        if len(xuid_values) > 0:
            return str(xuid_values[0])
    # 2. Joueur principal : depuis sync_meta
    if idx == 0:
        return _load_xuid_from_db(db_path)
    # 3. Coéquipier : depuis sa DB
    player_db = base_dir / name / "stats.duckdb"
    if player_db.exists():
        return _load_xuid_from_db(str(player_db))
    return ""


def _extract_match_row_from_df(df_player: Any) -> dict[str, Any]:
    """Extrait deaths/time_played_seconds/pair_name depuis un df Polars ou Pandas."""
    if isinstance(df_player, pl.DataFrame):
        deaths = int(df_player["deaths"].sum()) if "deaths" in df_player.columns else 0
        tps = (
            float(df_player["time_played_seconds"].sum())
            if "time_played_seconds" in df_player.columns
            else 600.0 * len(df_player)
        )
        pair = (
            df_player["pair_name"][0]
            if "pair_name" in df_player.columns and len(df_player) > 0
            else None
        )
    else:
        deaths = int(df_player["deaths"].sum()) if "deaths" in df_player.columns else 0
        tps = (
            float(df_player["time_played_seconds"].sum())
            if "time_played_seconds" in df_player.columns
            else 600.0 * len(df_player)
        )
        pair = (
            df_player["pair_name"].iloc[0]
            if "pair_name" in df_player.columns and len(df_player) > 0
            else None
        )
    return {"deaths": deaths, "time_played_seconds": tps, "pair_name": pair}


class TeammatesService:
    """Service d'agrégation pour la page Coéquipiers.

    Encapsule le chargement multi-DB, l'enrichissement des données,
    et les calculs de profils de participation.
    """

    @staticmethod
    def _load_xuid_from_db(db_path: str) -> str:
        """Charge le xuid depuis la table sync_meta de la DB (wrapper module-level)."""
        return _load_xuid_from_db(db_path)

    @staticmethod
    def load_teammate_stats(
        teammate_gamertag: str,
        match_ids: set[str],
        reference_db_path: str,
    ) -> TeammateStats:
        """Charge les stats d'un coéquipier depuis shared.match_participants.

        Architecture V5 : lit directement depuis shared_matches.duckdb
        au lieu de charger la DB individuelle du coéquipier.

        Returns:
            TeammateStats avec le DataFrame filtré ou vide.
        """
        from src.data.repositories.duckdb_repo import DuckDBRepository

        _empty = TeammateStats(gamertag=teammate_gamertag, df=pl.DataFrame(), is_empty=True)
        if not match_ids or not reference_db_path:
            return _empty

        try:
            repo = DuckDBRepository(reference_db_path, "")
            conn = repo._get_connection()

            xuid = _resolve_xuid_from_shared(conn, teammate_gamertag)
            if not xuid:
                logger.warning(
                    "Coéquipier '%s' introuvable dans shared "
                    "(ni xuid_aliases ni match_participants)",
                    teammate_gamertag,
                )
                return _empty

            match_ids_list = [str(mid) for mid in match_ids]
            df_filtered = _query_teammate_shared_stats(conn, xuid, match_ids_list)
            return TeammateStats(
                gamertag=teammate_gamertag,
                df=df_filtered,
                is_empty=df_filtered.is_empty(),
            )
        except Exception as e:
            logger.debug("Erreur load_teammate_stats shared: %s", e)
            return _empty

    @staticmethod
    def load_all_teammate_stats(
        teammate_gamertag: str,
        reference_db_path: str,
    ) -> TeammateStats:
        """Charge l'historique complet d'un coéquipier (tous matchs, sans filtre session).

        Utilisé pour calculer les records historiques. Retourne toutes les parties
        du joueur avec pair_name, pour permettre le filtre par catégorie de mode.

        Returns:
            TeammateStats avec le DataFrame complet ou vide.
        """
        from src.data.repositories.duckdb_repo import DuckDBRepository
        from src.data.services._teammates_history_queries import query_teammate_full_history

        _empty = TeammateStats(gamertag=teammate_gamertag, df=pl.DataFrame(), is_empty=True)
        if not reference_db_path:
            return _empty
        try:
            repo = DuckDBRepository(reference_db_path, "")
            conn = repo._get_connection()
            xuid = _resolve_xuid_from_shared(conn, teammate_gamertag)
            if not xuid:
                logger.debug("load_all_teammate_stats: xuid introuvable pour '%s'", teammate_gamertag)
                return _empty
            df_all = query_teammate_full_history(conn, xuid)
            return TeammateStats(gamertag=teammate_gamertag, df=df_all, is_empty=df_all.is_empty())
        except Exception as e:
            logger.debug("Erreur load_all_teammate_stats pour '%s': %s", teammate_gamertag, e)
            return _empty

    @staticmethod
    def enrich_with_performance_score(
        df: pl.DataFrame,
        gamertag: str,
        reference_db_path: str,
        *,
        is_main: bool = False,
    ) -> pl.DataFrame:
        """Ajoute performance_score, session_id et session_label depuis player_match_enrichment."""
        if df.is_empty() or "match_id" not in df.columns:
            return df
        base_dir = Path(reference_db_path).parent.parent
        player_db = reference_db_path if is_main else str(base_dir / gamertag / "stats.duckdb")
        m_ids = df["match_id"].cast(pl.Utf8)
        match_ids = m_ids.to_list()
        enrichment = load_perf_enrichment_with_session(player_db, match_ids)
        if not enrichment:
            return df
        perf = m_ids.map_elements(
            lambda m: enrichment.get(m, {}).get("perf"), return_dtype=pl.Float64
        )
        session_id_col = m_ids.map_elements(
            lambda m: enrichment.get(m, {}).get("session_id"), return_dtype=pl.Utf8
        )
        session_label_col = m_ids.map_elements(
            lambda m: enrichment.get(m, {}).get("session_label"), return_dtype=pl.Utf8
        )
        updates = {"performance_score": perf}
        if "session_id" not in df.columns:
            updates["session_id"] = session_id_col
        if "session_label" not in df.columns:
            updates["session_label"] = session_label_col
        return df.with_columns(**updates)

    @staticmethod
    def enrich_series_with_perfect_kills(
        series: list[tuple[str, pl.DataFrame]],
        db_path: str,
    ) -> EnrichedSeries:
        """Ajoute la colonne perfect_kills à chaque DataFrame de la série.

        Le 1er élément (idx=0) = joueur principal (utilise db_path).
        Les suivants = coéquipiers (utilise base_dir / gamertag / stats.duckdb).

        Returns:
            EnrichedSeries avec la colonne perfect_kills ajoutée.
        """
        from src.data.repositories.duckdb_repo import DuckDBRepository

        if not db_path or not str(db_path).endswith(".duckdb"):
            return EnrichedSeries(series=series)

        base_dir = Path(db_path).parent.parent
        enriched: list[tuple[str, pl.DataFrame]] = []

        for idx, (name, df) in enumerate(series):
            df = df.clone()
            if not ("match_id" in df.columns and not df.is_empty()):
                df = df.with_columns(pl.lit(0).alias("perfect_kills"))
                enriched.append((name, df))
                continue

            match_ids = df["match_id"].cast(pl.Utf8).to_list()
            try:
                player_xuid = _resolve_xuid_for_enrichment(name, idx, df, db_path, base_dir)
                if not player_xuid:
                    logger.warning(
                        "Impossible de trouver le xuid pour %s, skip perfect_kills", name
                    )
                    df = df.with_columns(pl.lit(0).alias("perfect_kills"))
                    enriched.append((name, df))
                    continue

                use_path = db_path if idx == 0 else str(base_dir / name / "stats.duckdb")
                if idx != 0 and not Path(use_path).exists():
                    use_path = db_path

                repo = DuckDBRepository(use_path, player_xuid)
                counts = repo.count_perfect_kills_by_match(match_ids)
                if not counts:
                    df = df.with_columns(pl.lit(0).alias("perfect_kills"))
                else:
                    _lookup = pl.DataFrame(
                        {
                            "_mid": pl.Series(list(counts.keys()), dtype=pl.Utf8),
                            "_pk": pl.Series(list(counts.values()), dtype=pl.Int64),
                        }
                    )
                    df = df.with_columns(pl.col("match_id").cast(pl.Utf8).alias("_join_mid"))
                    df = df.join(_lookup, left_on="_join_mid", right_on="_mid", how="left")
                    df = df.with_columns(pl.col("_pk").fill_null(0).alias("perfect_kills")).drop(
                        ["_join_mid", "_pk"], strict=False
                    )
            except Exception as e:
                logger.debug("Erreur enrichissement perfect_kills pour %s: %s", name, e)
                df = df.with_columns(pl.lit(0).alias("perfect_kills"))
            enriched.append((name, df))

        return EnrichedSeries(series=enriched)

    @staticmethod
    def compute_participation_profiles(
        players_data: list[dict[str, Any]],
        db_path: str,
        shared_match_ids: list[str],
    ) -> list[dict[str, Any]]:
        """Calcule les profils de participation radar pour N joueurs.

        Returns:
            Liste de profils (dicts) compatibles avec create_participation_profile_radar.
        """
        from src.analysis.participation_radar import (
            ProfileOptions,
            compute_participation_profile,
            get_radar_thresholds,
        )
        from src.data.repositories import DuckDBRepository  # noqa: PLC0415

        thresholds = get_radar_thresholds(db_path) if db_path else None
        base_dir = Path(db_path).parent.parent
        profiles: list[dict[str, Any]] = []

        for player in players_data:
            name = player["name"]
            df_player = player["df"]
            color = player["color"]
            is_main = player.get("is_main", False)

            is_empty = (
                hasattr(df_player, "is_empty")
                and df_player.is_empty()
                or hasattr(df_player, "empty")
                and df_player.empty
            )
            if is_empty:
                continue

            player_db = db_path if is_main else str(base_dir / name / "stats.duckdb")
            if not Path(player_db).exists():
                continue

            try:
                repo = DuckDBRepository(player_db, player.get("xuid", ""))
                if not repo.has_personal_score_awards():
                    continue
                ps = repo.load_personal_score_awards_as_polars(match_ids=shared_match_ids)
                if ps.is_empty():
                    continue
                match_row = _extract_match_row_from_df(df_player)
                profile = compute_participation_profile(
                    ps,
                    match_row=match_row,
                    options=ProfileOptions(
                        name=name,
                        color=color,
                        pair_name=match_row.get("pair_name"),
                        thresholds=thresholds,
                    ),
                )
                profiles.append(profile)
            except Exception:
                logger.warning("Échec construction profil coéquipier %s", name, exc_info=True)

        return profiles

    @staticmethod
    def load_impact_data(
        db_path: str,
        xuid: str,
        match_ids: list[str],
        friend_xuids: list[str],
    ) -> ImpactData:
        """Charge les données d'impact et taquinerie depuis shared.

        V5 : Lit shared.highlight_events et shared.match_participants (outcome).

        Returns:
            ImpactData avec événements ou available=False.
        """
        from src.data.repositories import DuckDBRepository

        _empty = ImpactData(
            first_bloods={},
            clutch_finishers={},
            last_casualties={},
            scores={},
            gamertags=[],
            match_ids=[],
            available=False,
        )

        if len(friend_xuids) < 2 or not match_ids:
            return _empty

        try:
            repo = DuckDBRepository(db_path, xuid.strip())
            conn = repo._get_connection()
            placeholders = ", ".join(["?" for _ in match_ids])
            all_friend_xuids = {str(x) for x in friend_xuids}
            all_friend_xuids.add(str(xuid).strip())

            result = _collect_impact_data(conn, match_ids, xuid, all_friend_xuids, placeholders)
            if result is None:
                return _empty

            (
                first_bloods,
                clutch_finishers,
                last_casualties,
                last_group_kills,
                first_group_deaths,
                scores,
            ) = result
            if not scores:
                return _empty

            involved_keys = (
                set(first_bloods)
                | set(clutch_finishers)
                | set(last_casualties)
                | set(last_group_kills)
                | set(first_group_deaths)
            )
            return ImpactData(
                first_bloods=first_bloods,
                clutch_finishers=clutch_finishers,
                last_casualties=last_casualties,
                scores=scores,
                gamertags=list(scores.keys()),
                match_ids=sorted({m for m in match_ids if m in involved_keys}),
                available=True,
            )

        except Exception:
            return _empty
