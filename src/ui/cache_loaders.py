"""Fonctions de cache Streamlit — Chargement atomique depuis DuckDB.

Ce module regroupe les fonctions @st.cache_data qui effectuent
des lectures unitaires depuis la base DuckDB (matchs, médailles,
rosters, coéquipiers, highlights, etc.).

Extrait de cache.py lors du Sprint 17 (découpage <800L).
"""

from __future__ import annotations

import logging
import os
from typing import TYPE_CHECKING

import polars as pl
import streamlit as st

from src.utils.profiles import list_local_dbs

if TYPE_CHECKING:
    from src.data.repositories.duckdb_repo import DuckDBRepository

logger = logging.getLogger(__name__)

# Timezone Paris pour les conversions
PARIS_TZ_NAME = "Europe/Paris"


# ─── Cache Repository (v5.1 perf) ──────────────────────────────────────────
# Connexion persistante via @st.cache_resource pour éviter les reconnexions
# coûteuses (ATTACH × 3 = 50-100ms par instanciation).


@st.cache_resource(ttl=3600)
def get_cached_repository_st(
    db_path: str,
    xuid: str,
) -> DuckDBRepository:
    """Retourne un DuckDBRepository mis en cache avec connexion persistante.

    Le repository est réutilisé entre les pages Streamlit pour éviter
    les reconnexions coûteuses (3× ATTACH = 50-100ms).

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        xuid: XUID du joueur.

    Returns:
        Instance DuckDBRepository mise en cache.
    """
    from src.data.repositories.duckdb_repo import DuckDBRepository

    logger.info(f"Création d'un repository mis en cache pour {db_path}")
    repo = DuckDBRepository(db_path, xuid, read_only=True)
    # Warm-up : forcer la connexion + ATTACH immédiatement
    repo._get_connection()
    return repo


# ─── Constantes de projection par page (Sprint 19 — tâche 19.3) ────────────
# Colonnes effectivement utilisées par les pages principales.
# Permet de réduire la mémoire en ne chargeant que le nécessaire.
# Note : game_variant_id, game_variant_name, team_id ne sont utilisés par aucune
# page hot-path et sont exclus du set commun.

COLUMNS_COMMON: list[str] = [
    "match_id",
    "start_time",
    "map_id",
    "map_name",
    "playlist_id",
    "playlist_name",
    "pair_id",
    "pair_name",
    "outcome",
    "kda",
    "kills",
    "deaths",
    "assists",
    "accuracy",
    "average_life_seconds",
    "time_played_seconds",
    "max_killing_spree",
    "headshot_kills",
    "personal_score",
    "my_team_score",
    "enemy_team_score",
    "team_mmr",
    "enemy_mmr",
]

# Colonnes calculées ajoutées par _enrich_matches_df
COLUMNS_COMPUTED: list[str] = [
    "ratio",
    "date",
    "kills_per_min",
    "deaths_per_min",
    "assists_per_min",
]


def _to_polars(df: object) -> pl.DataFrame:
    """Convertit un DataFrame Pandas en Polars si nécessaire (pont de sécurité)."""
    if isinstance(df, pl.DataFrame):
        return df
    try:
        return pl.from_pandas(df)  # type: ignore[arg-type]
    except Exception:
        return pl.DataFrame()


def db_cache_key(db_path: str) -> tuple[int, int, int, int] | None:
    """Retourne une signature stable des DBs pour invalider les caches.

    Surveille à la fois *stats.duckdb* (player) ET *shared_matches.duckdb*
    (matchs partagés v5) : les nouveaux matchs sont écrits dans shared, pas
    dans stats. Sans ce second composant, le cache @st.cache_data ne voit
    pas les matchs ajoutés après la dernière lecture.

    Returns:
        (mtime_ns_player, size_player, mtime_ns_shared, size_shared) ou None.
    """
    try:
        st_ = os.stat(db_path)
    except OSError:
        return None

    mtime_player = int(getattr(st_, "st_mtime_ns", int(st_.st_mtime * 1e9)))
    size_player = int(st_.st_size)

    # Chemin shared_matches.duckdb déduit du chemin joueur
    mtime_shared = 0
    size_shared = 0
    try:
        from src.utils.paths import get_shared_matches_path_from_player

        shared_path = get_shared_matches_path_from_player(db_path)
        if shared_path and shared_path.exists():
            st_shared = os.stat(shared_path)
            mtime_shared = int(getattr(st_shared, "st_mtime_ns", int(st_shared.st_mtime * 1e9)))
            size_shared = int(st_shared.st_size)
    except Exception:
        pass

    return mtime_player, size_player, mtime_shared, size_shared


def _is_duckdb_v4_path(db_path: str) -> bool:
    """Détecte si le chemin est une DB joueur DuckDB v4."""
    if not db_path:
        return False
    return db_path.endswith(".duckdb") or db_path.endswith("stats.duckdb")


def _resolve_player_xuid(db_path: str) -> str:
    """Résout le XUID du joueur depuis sa DB.

    Stratégie de fallback :
    1. sync_meta (key='xuid') — source canonique v5
    2. player_match_stats.xuid — source legacy v3/v4 (toujours présente)
    3. shared.xuid_aliases via gamertag — dernier recours

    Returns:
        XUID en string, ou "" si introuvable.
    """
    from src.utils.db import duckdb_read_only

    try:
        with duckdb_read_only(db_path) as conn:
            # Stratégie 1 : sync_meta (source canonique v5)
            try:
                result = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid'").fetchone()
                if result and result[0] and str(result[0]).strip():
                    return str(result[0]).strip()
            except Exception:
                pass

            # Stratégie 2 : player_match_stats.xuid (legacy v3/v4)
            try:
                result = conn.execute(
                    "SELECT DISTINCT xuid FROM player_match_stats WHERE xuid IS NOT NULL LIMIT 1"
                ).fetchone()
                if result and result[0] and str(result[0]).strip():
                    return str(result[0]).strip()
            except Exception:
                pass

        # Stratégie 3 : xuid_aliases via shared_matches.duckdb (v5.1)
        try:
            from pathlib import Path

            from src.utils.paths import get_shared_matches_path_from_player

            gamertag = Path(db_path).parent.name
            shared_path = get_shared_matches_path_from_player(db_path)
            if shared_path and shared_path.exists():
                with duckdb_read_only(shared_path) as shared_con:
                    result = shared_con.execute(
                        "SELECT xuid FROM xuid_aliases WHERE gamertag = ? LIMIT 1", [gamertag]
                    ).fetchone()
                    if result and result[0] and str(result[0]).strip():
                        return str(result[0]).strip()
        except Exception:
            pass

    except Exception:
        pass

    return ""


def _load_matches_duckdb_v4(db_path: str, include_firefight: bool = True) -> list:
    """Charge les matchs depuis une DB DuckDB v4 (legacy — retourne MatchRow).

    Préférer _load_matches_duckdb_v4_polars() pour le chemin optimisé.
    Utilise le repository caché (v5.1 perf) pour éviter les reconnexions.
    """
    try:
        player_xuid = _resolve_player_xuid(db_path)
        repo = get_cached_repository_st(db_path, player_xuid)
        return repo.load_matches(include_firefight=include_firefight)
    except Exception:
        return []


def _load_matches_duckdb_v4_polars(
    db_path: str,
    include_firefight: bool = True,
    columns: list[str] | None = None,
) -> pl.DataFrame:
    """Charge les matchs depuis une DB DuckDB v4 en Polars via Arrow zero-copy.

    Chemin optimisé Sprint 19 : DuckDB → Arrow → Polars sans intermédiaire
    MatchRow. ~3× plus rapide que _load_matches_duckdb_v4 + reconstruction.
    Utilise le repository caché (v5.1 perf) pour éviter les reconnexions.

    Args:
        db_path: Chemin vers la DB DuckDB.
        include_firefight: Inclure les matchs PvE.
        columns: Liste de colonnes à projeter (None = toutes).

    Returns:
        DataFrame Polars. Vide en cas d'erreur.
    """
    try:
        player_xuid = _resolve_player_xuid(db_path)
        repo = get_cached_repository_st(db_path, player_xuid)
        return repo.load_matches_as_polars(
            include_firefight=include_firefight,
            columns=columns,
        )
    except Exception:
        logger.debug("load_matches_as_polars échoué, fallback MatchRow", exc_info=True)
        return pl.DataFrame()


# 8bis.A4 : TTL augmenté de 30s à 300s (le filesystem ne change pas en navigation)
@st.cache_data(show_spinner=False, ttl=300)
def cached_list_local_dbs(_refresh_token: int = 0) -> list[str]:
    """Liste des DB locales (TTL court pour éviter un scan disque trop fréquent)."""
    return list_local_dbs()


@st.cache_data(show_spinner=False)
def cached_same_team_match_ids_with_friend(
    db_path: str,
    self_xuid: str,
    friend_xuid: str,
    db_key: tuple[int, int] | None,
) -> tuple[str, ...]:
    """Retourne les match_id (str) joués dans la même équipe avec un ami (cache).

    Utilise DuckDBRepository pour DuckDB v4, sinon fallback legacy.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(self_xuid).strip())
            match_ids = repo.load_same_team_match_ids(str(friend_xuid).strip())
            return tuple(sorted(match_ids))
        except Exception:
            return ()
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return ()


@st.cache_data(show_spinner=False)
def cached_query_matches_with_friend(
    db_path: str,
    self_xuid: str,
    friend_xuid: str,
    db_key: tuple[int, int] | None,
):
    """Requête les matchs joués avec un ami (cache).

    Utilise DuckDBRepository pour DuckDB v4, sinon fallback legacy.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(self_xuid).strip())
            match_ids = repo.load_matches_with_teammate(str(friend_xuid).strip())
            return match_ids
        except Exception:
            return []
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return []


@st.cache_data(show_spinner=False)
def cached_load_player_match_result(
    db_path: str,
    match_id: str,
    xuid: str,
    db_key: tuple[int, int] | None,
):
    """Charge le résultat d'un match pour un joueur (cache).

    Utilise DuckDBRepository pour .duckdb, sinon fallback legacy.

    Pipeline de lecture v5.1 :
        1. repo.load_match_skill_data(match_id) — charge team_mmr, enemy_mmr,
           kills/deaths/assists expected/stddev depuis shared.match_participants.
        2. Fallback : repo.load_match_mmr_batch() — uniquement team_mmr/enemy_mmr.

    ⚠️ assists expected/stddev : toujours NULL (limitation API Halo Infinite).
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(xuid).strip())
            # Charger skill data complet (MMR + expected/stddev)
            skill_data = repo.load_match_skill_data(match_id)
            if skill_data:
                skill_data["team_mmrs"] = None  # Non disponible dans DuckDB v4
                return skill_data
            # Fallback: load_match_mmr_batch si load_match_skill_data ne retourne rien
            mmr_data = repo.load_match_mmr_batch([match_id])
            team_mmr = None
            enemy_mmr = None
            if match_id in mmr_data:
                team_mmr, enemy_mmr = mmr_data[match_id]
            return {
                "team_id": None,
                "team_mmr": team_mmr,
                "enemy_mmr": enemy_mmr,
                "team_mmrs": None,
                "kills": {"count": None, "expected": None, "stddev": None},
                "deaths": {"count": None, "expected": None, "stddev": None},
                "assists": {"count": None, "expected": None, "stddev": None},
            }
        except Exception:
            return None
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return None


@st.cache_data(show_spinner=False)
def cached_load_match_medals_for_player(
    db_path: str,
    match_id: str,
    xuid: str,
    db_key: tuple[int, int] | None,
):
    """Charge les médailles d'un match pour un joueur (cache).

    Utilise DuckDBRepository pour .duckdb, sinon fallback legacy.
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(xuid).strip())
            return repo.load_match_medals(match_id)
        except Exception:
            return []
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return []


@st.cache_data(show_spinner=False)
def cached_load_match_rosters(
    db_path: str,
    match_id: str,
    xuid: str,
    db_key: tuple[int, int] | None,
):
    """Charge les rosters d'un match (cache).

    Utilise DuckDBRepository pour DuckDB v4, sinon fallback legacy.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(xuid).strip())
            return repo.load_match_rosters(match_id)
        except Exception:
            return None
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return None


@st.cache_data(show_spinner=False)
def cached_load_highlight_events_for_match(
    db_path: str,
    match_id: str,
    *,
    db_key: tuple[int, int] | None = None,
):
    """Charge les événements highlight d'un match (cache).

    Utilise DuckDBRepository caché pour .duckdb, sinon fallback legacy.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            player_xuid = _resolve_player_xuid(db_path)
            repo = get_cached_repository_st(db_path, player_xuid)
            return repo.load_highlight_events(match_id)
        except Exception:
            return []
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return []


@st.cache_data(show_spinner=False)
def cached_load_match_player_gamertags(
    db_path: str,
    match_id: str,
    *,
    db_key: tuple[int, int] | None = None,
):
    """Charge les gamertags des joueurs d'un match (cache).

    Sprint Gamertag Roster Fix : Utilise DuckDBRepository.resolve_gamertags_batch
    pour obtenir des gamertags propres depuis match_participants/xuid_aliases.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository pour résolution centralisée
    if _is_duckdb_v4_path(db_path):
        try:
            # Utiliser le repo caché pour récupérer les XUIDs et résoudre les gamertags
            # Résolution XUID : on utilise un xuid temporaire, resolve_gamertags_batch
            # ne dépend pas du xuid du repo
            repo = get_cached_repository_st(db_path, "")

            # Récupérer tous les XUIDs du match via le repo (highlight_events)
            events = repo.load_highlight_events(match_id)
            xuids = list({str(e["xuid"]) for e in events if e.get("xuid")})
            if not xuids:
                return {}

            return {
                xuid: gt
                for xuid, gt in repo.resolve_gamertags_batch(xuids, match_id=match_id).items()
                if gt
            }
        except Exception:
            return {}
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return {}


@st.cache_data(show_spinner=False)
def cached_load_top_medals(
    db_path: str,
    xuid: str,
    match_ids: tuple[str, ...],
    top_n: int | None,
    db_key: tuple[int, int] | None,
):
    """Charge les top médailles (cache).

    Utilise DuckDBRepository pour les bases .duckdb, sinon fallback legacy.
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(xuid).strip())
            return repo.load_top_medals(
                list(match_ids),
                top_n=(int(top_n) if top_n is not None else None),
            )
        except Exception:
            return []
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return []


def top_medals_smart(
    db_path: str,
    xuid: str,
    match_ids: list[str],
    *,
    top_n: int | None,
    db_key: tuple[int, int] | None,
):
    """Charge les top médailles avec gestion intelligente du cache.

    Évite de stocker d'immenses tuples en cache pour les grandes listes.
    Utilise DuckDBRepository pour les bases .duckdb.
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        if len(match_ids) > 1500:
            try:
                repo = get_cached_repository_st(db_path, str(xuid).strip())
                return repo.load_top_medals(match_ids, top_n=top_n)
            except Exception:
                return []
        return cached_load_top_medals(db_path, xuid, tuple(match_ids), top_n, db_key=db_key)
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return []


def clear_app_caches() -> None:
    """Vide les caches Streamlit (utile si DB/alias/csv changent en dehors de l'app).

    Invalide aussi le cache repository (v5.1) pour forcer une reconnexion
    avec les données fraîches.
    """
    import contextlib

    with contextlib.suppress(Exception):
        st.cache_data.clear()
    with contextlib.suppress(Exception):
        st.cache_resource.clear()


@st.cache_data(show_spinner=False)
def cached_list_other_xuids(
    db_path: str, self_xuid: str, db_key: tuple[int, int] | None = None, limit: int = 500
) -> list[str]:
    """Version cachée de list_other_player_xuids.

    DuckDB v4 utilise xuid_aliases. En v5, shared.match_participants.
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(self_xuid).strip())
            return repo.list_other_player_xuids(limit=limit)
        except Exception:
            return []
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return []


@st.cache_data(show_spinner=False)
def cached_list_top_teammates(
    db_path: str, self_xuid: str, db_key: tuple[int, int] | None = None, limit: int = 20
) -> list[tuple[str, int]]:
    """Version cachée de list_top_teammates.

    Utilise DuckDBRepository pour .duckdb, sinon TeammatesAggregate (cache DB),
    sinon fallback sur la requête JSON lente (list_top_teammates).
    """
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(self_xuid).strip())
            return repo.list_top_teammates(limit=limit)
        except Exception:
            return []
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return []


# =============================================================================
# Nouvelles fonctions utilisant les tables de cache optimisées
# =============================================================================


@st.cache_data(show_spinner=False)
def cached_has_cache_tables(db_path: str, db_key: tuple[int, int] | None = None) -> bool:
    """Vérifie si les tables de cache existent.

    DuckDB v4 considéré comme ayant toujours les tables de cache.
    """
    _ = db_key
    # DuckDB v4 : toujours considéré comme ayant le cache
    if _is_duckdb_v4_path(db_path):
        return True
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return False


@st.cache_data(show_spinner=False, ttl=300)
def cached_get_match_skill_rank(
    db_path: str,
    match_id: str,
    db_key: tuple[int, int] | None = None,
) -> tuple | None:
    """Retourne le rating LUSR/CSR d'un match depuis match_skill_rank (read-only, mis en cache).

    N'exécute aucun DDL : si la table n'existe pas, retourne None silencieusement.
    Le ``rating_delta`` est calculé dynamiquement via ``LAG`` sur les ``rating_value``
    stockés pour garantir la cohérence avec les valeurs affichés (corrige l'incohérence
    du delta stocké lors d'un recalcul avec seed différent).

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        match_id: Identifiant du match.
        db_key: Clé d'invalidation (mtime, size) de la DB.

    Returns:
        Tuple (rating_type, rating_value, rating_deviation, tier_label,
               sub_tier, tier, tier_fr, rating_delta, playlist_group)
        ou None si absent.
    """
    _ = db_key
    import duckdb

    try:
        with duckdb.connect(str(db_path), read_only=True) as conn:
            row = conn.execute(
                """
                WITH cte AS (
                    SELECT
                        msr.match_id,
                        msr.rating_type, msr.rating_value, msr.rating_deviation,
                        msr.tier_label, msr.sub_tier, msr.tier, msr.tier_fr,
                        msr.playlist_group,
                        msr.rating_value - LAG(msr.rating_value) OVER (
                            PARTITION BY msr.playlist_group
                            ORDER BY COALESCE(msr.start_time, msr.updated_at)
                        ) AS computed_delta
                    FROM match_skill_rank msr
                )
                SELECT rating_type, rating_value, rating_deviation, tier_label,
                       sub_tier, tier, tier_fr, computed_delta, playlist_group
                FROM cte
                WHERE match_id = ?
                """,
                [match_id],
            ).fetchone()
        return row
    except duckdb.CatalogException:
        # Table match_skill_rank absente (joueur sans LUSR/CSR calculé)
        return None
    except Exception:
        return None


@st.cache_data(show_spinner=False)
def cached_get_cache_stats(db_path: str, xuid: str, db_key: tuple[int, int] | None = None) -> dict:
    """Retourne les stats du cache DB pour un joueur.

    DuckDB v4 retourne des stats depuis le repository.
    """
    _ = db_key
    # DuckDB v4 : utiliser le repository caché (v5.1 perf)
    if _is_duckdb_v4_path(db_path):
        try:
            repo = get_cached_repository_st(db_path, str(xuid).strip())
            storage = repo.get_storage_info()
            return {
                "has_cache": True,
                "match_count": storage.get("total_matches", 0),
                "sessions_count": storage.get("sessions_count", 0),
            }
        except Exception:
            return {"has_cache": True}
    # Legacy SQLite non supporté depuis v4.8
    logger.warning(f"DB legacy SQLite non supportée: {db_path}")
    return {}


def _enrich_matches_df(df: pl.DataFrame) -> pl.DataFrame:
    """Enrichit un DataFrame Polars de matchs avec timezone et colonnes calculées.

    Applique les transformations standard :
    - Conversion timezone UTC → Paris → naïf
    - Extraction colonne date
    - Calcul kills/deaths/assists par minute

    Args:
        df: DataFrame Polars brut avec au minimum start_time, kills, deaths, assists,
            time_played_seconds.

    Returns:
        DataFrame enrichi.
    """
    if df.is_empty():
        return df

    # Conversion timezone start_time
    if "start_time" in df.columns:
        start_time_dtype = df.schema.get("start_time")
        if start_time_dtype in (
            pl.Datetime,
            pl.Datetime("us"),
            pl.Datetime("ns"),
            pl.Datetime("ms"),
        ):
            try:
                df = df.with_columns(
                    pl.col("start_time")
                    .dt.convert_time_zone(PARIS_TZ_NAME)
                    .dt.replace_time_zone(None)
                    .alias("start_time")
                )
            except Exception:
                df = df.with_columns(
                    pl.col("start_time")
                    .dt.replace_time_zone("UTC")
                    .dt.convert_time_zone(PARIS_TZ_NAME)
                    .dt.replace_time_zone(None)
                    .alias("start_time")
                )
        elif start_time_dtype == pl.Utf8:
            df = df.with_columns(
                pl.col("start_time")
                .str.to_datetime(time_zone="UTC")
                .dt.convert_time_zone(PARIS_TZ_NAME)
                .dt.replace_time_zone(None)
                .alias("start_time")
            )

        # Extraire la date
        df = df.with_columns(pl.col("start_time").dt.date().alias("date"))

    # Stats par minute
    if "time_played_seconds" in df.columns:
        df = df.with_columns(
            (pl.col("time_played_seconds").cast(pl.Float64) / 60.0)
            .clip(lower_bound=0.0)
            .alias("minutes")
        )
        per_min_cols = []
        if "kills" in df.columns:
            per_min_cols.append(
                (pl.col("kills").cast(pl.Float64) / pl.col("minutes")).alias("kills_per_min")
            )
        if "deaths" in df.columns:
            per_min_cols.append(
                (pl.col("deaths").cast(pl.Float64) / pl.col("minutes")).alias("deaths_per_min")
            )
        if "assists" in df.columns:
            per_min_cols.append(
                (pl.col("assists").cast(pl.Float64) / pl.col("minutes")).alias("assists_per_min")
            )
        if per_min_cols:
            df = df.with_columns(per_min_cols)
        df = df.drop("minutes")

    return df


@st.cache_data(show_spinner=False)
def load_df_optimized(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
    include_firefight: bool = True,
    cache_buster: int = 0,
) -> pl.DataFrame:
    """Charge les matchs avec fallback intelligent.

    Supporte:
    - DuckDB v4: data/players/{gamertag}/stats.duckdb
    - Legacy SQLite: MatchCache puis MatchStats

    Mécanisme d'invalidation du cache (Sprint 19 — tâche 19.4) :
    - db_key (mtime_ns, size) : détecte les modifications du fichier DB
      (sync externe, modification directe). Invalidation automatique.
    - cache_buster (int) : incrémenté dans session_state après un sync réussi.
      Force le rechargement même si db_key n'a pas encore changé (race condition).
    Les deux paramètres sont passés à @st.cache_data comme clés de hash,
    et ne sont pas lus dans le corps de la fonction.

    Args:
        db_path: Chemin vers la DB.
        xuid: XUID du joueur (ignoré pour DuckDB v4).
        db_key: Clé de cache (mtime, size) — None si fichier inexistant.
        include_firefight: Inclure les matchs PvE.
        cache_buster: Token pour forcer l'invalidation du cache après sync.

    Returns:
        DataFrame Polars enrichi avec toutes les colonnes calculées.
    """
    _ = db_key  # Utilisé pour invalidation du cache Streamlit
    _ = cache_buster  # Utilisé pour forcer le rechargement après sync

    # Détecter le type de DB
    if _is_duckdb_v4_path(db_path):
        # Sprint 19 : chemin optimisé DuckDB → Arrow → Polars (zero-copy)
        df = _load_matches_duckdb_v4_polars(db_path, include_firefight=include_firefight)
        if not df.is_empty():
            # Enrichissement standard (timezone, colonnes calculées)
            df = _enrich_matches_df(df)
            return df

        # Fallback legacy : MatchRow → reconstruction DataFrame
        matches = _load_matches_duckdb_v4(db_path, include_firefight=include_firefight)
    else:
        # Legacy SQLite non supporté depuis v4.8
        logger.warning(f"DB legacy SQLite non supportée: {db_path}")
        matches = []

    if not matches:
        return pl.DataFrame()

    # Construire le DataFrame Polars depuis MatchRow (fallback legacy)
    df = pl.DataFrame(
        {
            "match_id": [m.match_id for m in matches],
            "start_time": [m.start_time for m in matches],
            "map_id": [m.map_id for m in matches],
            "map_name": [m.map_name for m in matches],
            "playlist_id": [m.playlist_id for m in matches],
            "playlist_name": [m.playlist_name for m in matches],
            "pair_id": [m.map_mode_pair_id for m in matches],
            "pair_name": [m.map_mode_pair_name for m in matches],
            "game_variant_id": [m.game_variant_id for m in matches],
            "game_variant_name": [m.game_variant_name for m in matches],
            "outcome": [m.outcome for m in matches],
            "kda": [m.kda for m in matches],
            "my_team_score": [m.my_team_score for m in matches],
            "enemy_team_score": [m.enemy_team_score for m in matches],
            "max_killing_spree": [m.max_killing_spree for m in matches],
            "headshot_kills": [m.headshot_kills for m in matches],
            "average_life_seconds": [m.average_life_seconds for m in matches],
            "time_played_seconds": [m.time_played_seconds for m in matches],
            "kills": [m.kills for m in matches],
            "deaths": [m.deaths for m in matches],
            "assists": [m.assists for m in matches],
            "accuracy": [m.accuracy for m in matches],
            "ratio": [m.ratio for m in matches],
            "team_mmr": [m.team_mmr for m in matches],
            "enemy_mmr": [m.enemy_mmr for m in matches],
        }
    )

    return _enrich_matches_df(df)


# ─── Fonctions sociales réexportées depuis cache_social.py ─────────────────
from src.ui.cache_social import (  # noqa: E402
    cached_get_match_session_info,
    cached_load_friends,
    cached_load_top_teammates_optimized,
)

__all__ = [
    "cached_get_match_session_info",
    "cached_load_friends",
    "cached_load_top_teammates_optimized",
]
