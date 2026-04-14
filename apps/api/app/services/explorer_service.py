"""Service Explorer — recherche gamertag, filtres matches, profil joueur.

Toutes les importations src.* sont lazy (dans le corps des fonctions)
pour permettre le mocking en tests.

Architecture :
  1. search_gamertags     : charge tous les gamertags depuis shared DB + fuzzy search
  2. get_explorer_matches : filtre global + filtres locaux + pagination
  3. get_explorer_player  : matchs communs + encounter summary
"""

from __future__ import annotations

import contextlib
import logging
from datetime import datetime
from pathlib import Path

from apps.api.app._db_helpers import (
    FMT_DATETIME_FR,
    OUTCOME_LABELS,
    Outcome,
    build_match_source_sql,
    has_mv_player_matches,
    resolve_xuid,
)
from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.common import PaginatedResponse, PaginationMeta, PaginationRequest
from apps.api.app.schemas.explorer import (
    ExplorerEncounterRow,
    ExplorerMatchesQueryRequest,
    ExplorerMatchesQueryResponse,
    ExplorerMatchesQuerySummary,
    ExplorerMatchFilters,
    ExplorerMatchRow,
    ExplorerPlayerQueryRequest,
    ExplorerPlayerQueryResponse,
    ExplorerPlayerSummary,
    ExplorerPlayerTarget,
    GamertagSearchResponse,
    GamertagSuggestion,
)
from apps.api.app.schemas.filters import FilterContextInput

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# 1. Recherche gamertags (endpoint global /directory)
# ---------------------------------------------------------------------------


def search_gamertags(
    query: str,
    limit: int,
    shared_db_path: str,
) -> GamertagSearchResponse:
    """Recherche des gamertags par requête floue."""
    all_gamertags = _load_all_gamertags(shared_db_path)
    if not query or len(query) < 2:
        return GamertagSearchResponse(query=query, items=[])

    matched = _fuzzy_search(query, all_gamertags, n=limit)
    q_lower = query.strip().casefold()

    suggestions: list[GamertagSuggestion] = []
    for gt in matched:
        exact = gt.casefold() == q_lower
        score = 1.0 if exact else _similarity_score(query, gt)
        suggestions.append(
            GamertagSuggestion(
                gamertag=gt,
                xuid=None,  # xuid résolu à la demande (player-query)
                score=round(score, 3),
                exact_match=exact,
            )
        )

    suggestions.sort(key=lambda s: (-s.score, not s.exact_match, s.gamertag))
    return GamertagSearchResponse(query=query, items=suggestions[:limit])


# ---------------------------------------------------------------------------
# 2. Query matches (endpoint player-scoped)
# ---------------------------------------------------------------------------


def get_explorer_matches(
    player: PlayerContext,
    request: ExplorerMatchesQueryRequest,
) -> ExplorerMatchesQueryResponse:
    """Retourne les matchs filtrés paginés pour la vue Explorer."""

    df_full = _load_matches_explorer(player)

    # Filtre global (FilterContextInput)
    dff = (
        _apply_global_filter(df_full, player, request.filters)
        if not df_full.is_empty()
        else df_full
    )

    # Filtres locaux Explorer
    dff = _apply_local_filters(dff, request.match_filters)

    # Enrichissement minimal
    if not dff.is_empty():
        dff = _enrich_for_explorer(dff)

    rows = _to_explorer_match_rows(dff)
    rows_sorted = sorted(rows, key=lambda r: r.start_time, reverse=True)

    page_meta, page_rows = _paginate(rows_sorted, request.pagination)

    selected_mid = request.match_filters.selected_match_id
    table: PaginatedResponse[ExplorerMatchRow] = PaginatedResponse(
        items=page_rows,
        pagination=page_meta,
    )
    return ExplorerMatchesQueryResponse(
        summary=ExplorerMatchesQuerySummary(
            total_matches=len(rows_sorted),
            selected_match_id=selected_mid,
        ),
        table=table,
    )


# ---------------------------------------------------------------------------
# 3. Player query (encounter + matchs communs)
# ---------------------------------------------------------------------------


def get_explorer_player(
    player: PlayerContext,
    request: ExplorerPlayerQueryRequest,
) -> ExplorerPlayerQueryResponse:
    """Retourne le profil encounter + matchs communs avec un joueur cible."""
    target_gt = request.target_gamertag.strip()
    target_xuid = _resolve_xuid_for_gamertag(player, target_gt)

    common_df = _load_common_matches(player, target_xuid) if target_xuid else _empty_df()

    if not common_df.is_empty():
        common_df = _enrich_common_for_explorer(common_df)

    ally_df, enemy_df = _split_by_team(common_df)

    ally_encounter = _build_encounter_row(target_gt, target_xuid, ally_df, same_team=True)
    enemy_encounter = _build_encounter_row(target_gt, target_xuid, enemy_df, same_team=False)

    allies_table = [ally_encounter] if ally_encounter is not None else []
    enemies_table = [enemy_encounter] if enemy_encounter is not None else []

    summary = _build_player_summary(ally_df, enemy_df, common_df)
    common_match_rows = _to_explorer_match_rows(common_df)
    common_sorted = sorted(common_match_rows, key=lambda r: r.start_time, reverse=True)

    return ExplorerPlayerQueryResponse(
        target=ExplorerPlayerTarget(gamertag=target_gt, xuid=target_xuid),
        summary=summary,
        allies_table=allies_table,
        enemies_table=enemies_table,
        common_matches=common_sorted[:100],  # limité à 100 pour le rendu initial
    )


# ---------------------------------------------------------------------------
# Chargement DuckDB
# ---------------------------------------------------------------------------


def _load_all_gamertags(shared_db_path: str) -> list[str]:
    """Charge tous les gamertags depuis shared.v_gamertag_lookup."""
    from src.utils.db import duckdb_read_only

    db_path = Path(shared_db_path)
    if not db_path.exists():
        logger.debug("shared DB introuvable pour gamertag search: %s", db_path)
        return []

    try:
        with duckdb_read_only(str(db_path)) as conn:
            rows = conn.execute(
                "SELECT DISTINCT gamertag FROM v_gamertag_lookup ORDER BY gamertag"
            ).fetchall()
            return [str(r[0]) for r in rows if r[0]]
    except Exception:
        logger.debug("_load_all_gamertags: erreur", exc_info=True)
        return []


def _resolve_xuid_for_gamertag(player: PlayerContext, gamertag: str) -> str | None:
    """Résout un gamertag → xuid via shared.v_gamertag_lookup."""
    from src.utils.db import duckdb_read_only

    shared_path = Path(player.shared_db_path)
    if not shared_path.exists():
        return None

    try:
        with duckdb_read_only(str(Path(player.db_path))) as conn:
            with contextlib.suppress(Exception):
                conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")
            row = conn.execute(
                "SELECT xuid FROM shared.v_gamertag_lookup WHERE LOWER(gamertag) = LOWER(?) LIMIT 1",
                [gamertag],
            ).fetchone()
            return str(row[0]) if row else None
    except Exception:
        logger.debug("_resolve_xuid_for_gamertag(%s): erreur", gamertag, exc_info=True)
        return None


def _load_matches_explorer(player: PlayerContext):  # type: ignore[return]
    """Charge les matchs avec les colonnes nécessaires pour l'Explorer."""
    import polars as pl

    from src.utils.db import duckdb_read_only

    db_path = Path(player.db_path)
    shared_path = Path(player.shared_db_path)

    if not db_path.exists():
        return pl.DataFrame()

    try:
        with duckdb_read_only(str(db_path)) as conn:
            if shared_path.exists():
                with contextlib.suppress(Exception):
                    conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")

            xuid = resolve_xuid(conn)
            if not xuid:
                return pl.DataFrame()

            has_mv = has_mv_player_matches(conn)
            source_sql = build_match_source_sql(has_mv)

            sql = f"""
            SELECT
                ms.match_id,
                ms.start_time,
                ms.map_name,
                COALESCE(ms.map_name_fr, ms.map_name)               AS map_name_fr,
                ms.pair_name,
                COALESCE(ms.pair_name_fr, ms.pair_name)             AS pair_name_fr,
                ms.playlist_name,
                COALESCE(ms.playlist_name_fr, ms.playlist_name)     AS playlist_name_fr,
                COALESCE(ms.is_firefight, FALSE)                    AS is_firefight,
                COALESCE(ms.is_ranked, FALSE)                       AS is_ranked,
                pme.session_id,
                pme.session_label,
                COALESCE(pme.is_with_friends, FALSE)                AS is_with_friends,
                COALESCE(p.outcome, 0)                              AS outcome,
                p.my_team_score,
                p.enemy_team_score
            FROM {source_sql} ms
            LEFT JOIN shared.match_participants p
                ON ms.match_id = p.match_id AND p.xuid = ?
            LEFT JOIN player_match_enrichment pme
                ON ms.match_id = pme.match_id
            ORDER BY ms.start_time DESC
            """
            result = conn.execute(sql, [xuid, xuid] if "?" in source_sql else [xuid])
            columns = [d[0] for d in result.description]
            rows = result.fetchall()

        if not rows:
            return pl.DataFrame()

        return pl.DataFrame(rows, schema=columns, orient="row")

    except Exception:
        logger.exception("Erreur chargement matchs Explorer pour %s", player.player_slug)
        return pl.DataFrame()


def _load_common_matches(player: PlayerContext, target_xuid: str):  # type: ignore[return]
    """Charge les matchs communs entre le joueur courant et target_xuid."""
    import polars as pl

    from src.utils.db import duckdb_read_only

    db_path = Path(player.db_path)
    shared_path = Path(player.shared_db_path)

    if not db_path.exists():
        return pl.DataFrame()

    try:
        with duckdb_read_only(str(db_path)) as conn:
            if shared_path.exists():
                with contextlib.suppress(Exception):
                    conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")

            xuid = resolve_xuid(conn)
            if not xuid:
                return pl.DataFrame()

            result = conn.execute(
                """
                SELECT
                    p.match_id,
                    r.start_time,
                    p.team_id   AS player_team_id,
                    t.team_id   AS target_team_id,
                    r.map_name,
                    r.playlist_name,
                    r.pair_name,
                    p.outcome,
                    p.my_team_score,
                    p.enemy_team_score
                FROM shared.match_participants p
                INNER JOIN shared.match_participants t
                    ON t.match_id = p.match_id AND t.xuid = ?
                INNER JOIN shared.match_registry r
                    ON r.match_id = p.match_id
                WHERE p.xuid = ?
                ORDER BY r.start_time DESC
                """,
                [target_xuid, xuid],
            )
            columns = [d[0] for d in result.description]
            rows = result.fetchall()

        if not rows:
            return pl.DataFrame()
        return pl.DataFrame(rows, schema=columns, orient="row")

    except Exception:
        logger.debug("_load_common_matches: erreur", exc_info=True)
        return pl.DataFrame()


def _empty_df():  # type: ignore[return]
    """Retourne un DataFrame vide Polars."""
    import polars as pl

    return pl.DataFrame()


# ---------------------------------------------------------------------------
# Filtrage
# ---------------------------------------------------------------------------


def _apply_global_filter(df_full, player: PlayerContext, filters: FilterContextInput):  # type: ignore[return]
    """Applique le filtre global (période ou sessions)."""
    try:
        from apps.api.app.services.filter_service import (
            _apply_cascade_filter,
            _apply_period_filter,
            _apply_session_filter,
            _normalize_filter_input,
        )

        effective = _normalize_filter_input(filters, df_full)
        if effective.filter_mode == "sessions":
            df_temporal = _apply_session_filter(df_full, effective.sessions)
        else:
            df_temporal = _apply_period_filter(df_full, effective.period)

        return _apply_cascade_filter(df_temporal, effective.cascade)
    except Exception:
        logger.debug("_apply_global_filter: erreur", exc_info=True)
        return df_full


def _apply_local_filters(dff, mf: ExplorerMatchFilters):  # type: ignore[return]
    """Applique les filtres locaux de l'Explorer (date, squad_scope, experience, etc.)."""
    try:
        import polars as pl

        if dff.is_empty():
            return dff

        # Filtre par date si demandé
        if mf.selected_date and "start_time" in dff.columns:
            col_dtype = dff["start_time"].dtype
            if col_dtype in (pl.Utf8, pl.String):
                dff = dff.with_columns(
                    pl.col("start_time").str.to_datetime(format="%Y-%m-%d %H:%M:%S", strict=False)
                )
            date_val = mf.selected_date
            dff = dff.filter(pl.col("start_time").cast(pl.Date) == date_val)

        # Filtre squad scope
        if mf.squad_scope != "all" and "is_with_friends" in dff.columns:
            if mf.squad_scope == "squad":
                dff = dff.filter(pl.col("is_with_friends").cast(pl.Boolean) == True)  # noqa: E712
            elif mf.squad_scope == "solo":
                dff = dff.filter(pl.col("is_with_friends").cast(pl.Boolean) == False)  # noqa: E712

        # Filtre playlist
        if mf.playlist and "playlist_name" in dff.columns:
            dff = dff.filter(
                pl.col("playlist_name").cast(pl.Utf8).str.contains(mf.playlist, literal=False)
            )

        # Filtre mode
        if mf.mode and "pair_name" in dff.columns:
            dff = dff.filter(pl.col("pair_name").cast(pl.Utf8).str.contains(mf.mode, literal=False))

        # Filtre carte
        if mf.map and "map_name" in dff.columns:
            dff = dff.filter(pl.col("map_name").cast(pl.Utf8).str.contains(mf.map, literal=False))

        # Filtre match_id unique
        if mf.selected_match_id and "match_id" in dff.columns:
            dff = dff.filter(pl.col("match_id").cast(pl.Utf8) == mf.selected_match_id)

        return dff
    except Exception:
        logger.debug("_apply_local_filters: erreur", exc_info=True)
        return dff


# ---------------------------------------------------------------------------
# Enrichissement
# ---------------------------------------------------------------------------


def _enrich_for_explorer(dff):  # type: ignore[return]
    """Enrich le DataFrame avec les colonnes UI nécessaires."""
    import polars as pl

    # display columns
    if "map_ui" not in dff.columns and "map_name" in dff.columns:
        src = (
            pl.coalesce([pl.col("map_name_fr").cast(pl.Utf8), pl.col("map_name").cast(pl.Utf8)])
            if "map_name_fr" in dff.columns
            else pl.col("map_name").cast(pl.Utf8)
        )
        dff = dff.with_columns(src.alias("map_ui"))

    if "mode_ui" not in dff.columns and "pair_name" in dff.columns:
        src = (
            pl.coalesce([pl.col("pair_name_fr").cast(pl.Utf8), pl.col("pair_name").cast(pl.Utf8)])
            if "pair_name_fr" in dff.columns
            else pl.col("pair_name").cast(pl.Utf8)
        )
        dff = dff.with_columns(src.alias("mode_ui"))

    if "playlist_ui" not in dff.columns and "playlist_name" in dff.columns:
        src = (
            pl.coalesce(
                [pl.col("playlist_name_fr").cast(pl.Utf8), pl.col("playlist_name").cast(pl.Utf8)]
            )
            if "playlist_name_fr" in dff.columns
            else pl.col("playlist_name").cast(pl.Utf8)
        )
        dff = dff.with_columns(src.alias("playlist_ui"))

    # outcome label
    if "outcome_label" not in dff.columns and "outcome" in dff.columns:
        dff = dff.with_columns(
            pl.col("outcome")
            .cast(pl.Int64, strict=False)
            .replace_strict(OUTCOME_LABELS, default=pl.lit("-"))
            .alias("outcome_label")
        )

    # score label
    if "score_label" not in dff.columns:
        my_score = (
            pl.col("my_team_score")
            .cast(pl.Float64, strict=False)
            .round(0)
            .cast(pl.Int64, strict=False)
            .fill_null(0)
            .cast(pl.Utf8)
            if "my_team_score" in dff.columns
            else pl.lit("-")
        )
        enemy_score = (
            pl.col("enemy_team_score")
            .cast(pl.Float64, strict=False)
            .round(0)
            .cast(pl.Int64, strict=False)
            .fill_null(0)
            .cast(pl.Utf8)
            if "enemy_team_score" in dff.columns
            else pl.lit("-")
        )
        dff = dff.with_columns(
            pl.concat_str([my_score, pl.lit(" - "), enemy_score]).alias("score_label")
        )

    # start_time_label
    col_dtype = dff["start_time"].dtype if "start_time" in dff.columns else None
    if col_dtype in (pl.Utf8, pl.String):
        dff = dff.with_columns(
            pl.col("start_time").str.to_datetime(format="%Y-%m-%d %H:%M:%S", strict=False)
        )
    if "start_time_label" not in dff.columns and "start_time" in dff.columns:
        dff = dff.with_columns(
            pl.col("start_time")
            .dt.strftime(FMT_DATETIME_FR)
            .fill_null("-")
            .alias("start_time_label")
        )

    # experience_type_label
    if "experience_type_label" not in dff.columns:
        if "is_ranked" in dff.columns:
            dff = dff.with_columns(
                pl.when(
                    pl.col("is_firefight").cast(pl.Boolean)
                    if "is_firefight" in dff.columns
                    else pl.lit(False)
                )
                .then(pl.lit("PvE"))
                .when(pl.col("is_ranked").cast(pl.Boolean))
                .then(pl.lit("Classé"))
                .otherwise(pl.lit("Non classé"))
                .alias("experience_type_label")
            )
        else:
            dff = dff.with_columns(pl.lit("Non classé").alias("experience_type_label"))

    return dff


def _enrich_common_for_explorer(dff):  # type: ignore[return]
    """Enrich les matchs communs pour le rendu Explorer."""
    import polars as pl

    if "map_ui" not in dff.columns and "map_name" in dff.columns:
        dff = dff.with_columns(pl.col("map_name").cast(pl.Utf8).alias("map_ui"))
    if "mode_ui" not in dff.columns and "pair_name" in dff.columns:
        dff = dff.with_columns(pl.col("pair_name").cast(pl.Utf8).alias("mode_ui"))
    if "playlist_ui" not in dff.columns and "playlist_name" in dff.columns:
        dff = dff.with_columns(pl.col("playlist_name").cast(pl.Utf8).alias("playlist_ui"))
    if "outcome_label" not in dff.columns and "outcome" in dff.columns:
        dff = dff.with_columns(
            pl.col("outcome")
            .cast(pl.Int64, strict=False)
            .replace_strict(OUTCOME_LABELS, default=pl.lit("-"))
            .alias("outcome_label")
        )
    if "score_label" not in dff.columns:
        my_score = (
            pl.col("my_team_score")
            .cast(pl.Float64, strict=False)
            .round(0)
            .cast(pl.Int64, strict=False)
            .fill_null(0)
            .cast(pl.Utf8)
            if "my_team_score" in dff.columns
            else pl.lit("-")
        )
        enemy_score = (
            pl.col("enemy_team_score")
            .cast(pl.Float64, strict=False)
            .round(0)
            .cast(pl.Int64, strict=False)
            .fill_null(0)
            .cast(pl.Utf8)
            if "enemy_team_score" in dff.columns
            else pl.lit("-")
        )
        dff = dff.with_columns(
            pl.concat_str([my_score, pl.lit(" - "), enemy_score]).alias("score_label")
        )
    col_dtype = dff["start_time"].dtype if "start_time" in dff.columns else None
    if col_dtype in (pl.Utf8, pl.String):
        dff = dff.with_columns(
            pl.col("start_time").str.to_datetime(format="%Y-%m-%d %H:%M:%S", strict=False)
        )
    if "start_time_label" not in dff.columns and "start_time" in dff.columns:
        dff = dff.with_columns(
            pl.col("start_time")
            .dt.strftime(FMT_DATETIME_FR)
            .fill_null("-")
            .alias("start_time_label")
        )
    if "is_with_friends" not in dff.columns:
        dff = dff.with_columns(pl.lit(False).alias("is_with_friends"))
    if "experience_type_label" not in dff.columns:
        dff = dff.with_columns(pl.lit("Non classé").alias("experience_type_label"))
    return dff


# ---------------------------------------------------------------------------
# Conversion → ExplorerMatchRow
# ---------------------------------------------------------------------------


def _to_explorer_match_rows(dff) -> list[ExplorerMatchRow]:
    """Convertit le DataFrame en liste d'ExplorerMatchRow."""
    if hasattr(dff, "is_empty") and dff.is_empty():
        return []

    rows: list[ExplorerMatchRow] = []
    for r in dff.iter_rows(named=True):
        rows.append(
            ExplorerMatchRow(
                match_id=str(r.get("match_id") or ""),
                start_time=r.get("start_time") or datetime.min,
                start_time_label=str(r.get("start_time_label") or "-"),
                map_ui=str(r.get("map_ui") or r.get("map_name") or "-"),
                mode_ui=str(r.get("mode_ui") or r.get("pair_name") or "-"),
                playlist_label=str(r.get("playlist_ui") or r.get("playlist_name") or "-"),
                outcome_label=str(r.get("outcome_label") or "-"),
                score_label=str(r.get("score_label") or "-"),
                is_with_friends=bool(r.get("is_with_friends") or False),
                experience_type_label=str(r.get("experience_type_label") or "Non classé"),
            )
        )
    return rows


# ---------------------------------------------------------------------------
# Encounter logic
# ---------------------------------------------------------------------------


def _split_by_team(common_df):  # type: ignore[return]
    """Sépare les matchs communs en alliés et adversaires."""
    try:
        import polars as pl

        if common_df.is_empty():
            return pl.DataFrame(), pl.DataFrame()
        if "player_team_id" not in common_df.columns or "target_team_id" not in common_df.columns:
            return common_df, pl.DataFrame()
        ally_df = common_df.filter(pl.col("player_team_id") == pl.col("target_team_id"))
        enemy_df = common_df.filter(pl.col("player_team_id") != pl.col("target_team_id"))
        return ally_df, enemy_df
    except Exception:
        return _empty_df(), _empty_df()


def _build_encounter_row(
    gamertag: str,
    xuid: str | None,
    df: pl.DataFrame,  # type: ignore[name-defined]  # noqa: F821
    *,
    same_team: bool,
) -> ExplorerEncounterRow | None:
    """Construit un ExplorerEncounterRow depuis un sous-DataFrame."""
    try:
        if df.is_empty():
            return None

        import polars as pl

        count = len(df)
        wins = 0
        losses = 0
        last_seen: datetime | None = None

        if "outcome" in df.columns:
            wins = int(df["outcome"].cast(pl.Int64, strict=False).eq(Outcome.WIN).sum())
            losses = count - wins

        if "start_time" in df.columns:
            col_dtype = df["start_time"].dtype
            if col_dtype not in (pl.Utf8, pl.String):
                ts = df["start_time"].drop_nulls()
                if not ts.is_empty():
                    last_seen = ts.max()

        return ExplorerEncounterRow(
            gamertag=gamertag,
            xuid=xuid,
            count_matches=count,
            wins=wins,
            losses=losses,
            last_seen_at=last_seen,
            same_team=same_team,
        )
    except Exception:
        logger.debug("_build_encounter_row: erreur", exc_info=True)
        return None


def _build_player_summary(ally_df, enemy_df, common_df) -> ExplorerPlayerSummary:
    """Construit le résumé global de l'encounter."""
    try:
        import polars as pl

        total = len(common_df) if hasattr(common_df, "__len__") else 0
        wins = 0
        losses = 0
        last_seen: datetime | None = None

        if hasattr(common_df, "is_empty") and not common_df.is_empty():
            if "outcome" in common_df.columns:
                wins = int(common_df["outcome"].cast(pl.Int64, strict=False).eq(Outcome.WIN).sum())
                losses = total - wins

            if "start_time" in common_df.columns:
                col_dtype = common_df["start_time"].dtype
                if col_dtype not in (pl.Utf8, pl.String):
                    ts = common_df["start_time"].drop_nulls()
                    if not ts.is_empty():
                        last_seen = ts.max()

        return ExplorerPlayerSummary(
            matches_together=total,
            wins_together=wins,
            losses_together=losses,
            last_seen_at=last_seen,
        )
    except Exception:
        return ExplorerPlayerSummary(
            matches_together=0,
            wins_together=0,
            losses_together=0,
            last_seen_at=None,
        )


# ---------------------------------------------------------------------------
# Fuzzy search
# ---------------------------------------------------------------------------


def _fuzzy_search(query: str, all_gamertags: list[str], n: int = 8) -> list[str]:
    """Recherche floue de gamertags (difflib + fallback substring)."""
    import difflib

    q_lower = query.strip().casefold()
    close = difflib.get_close_matches(query, all_gamertags, n=n, cutoff=0.4)
    substring_matches = [gt for gt in all_gamertags if q_lower in gt.casefold()]

    seen: set[str] = set()
    result: list[str] = []
    for gt in close + substring_matches:
        if gt not in seen:
            seen.add(gt)
            result.append(gt)
        if len(result) >= n:
            break
    return result


def _similarity_score(query: str, candidate: str) -> float:
    """Score de similarité SequenceMatcher."""
    import difflib

    return difflib.SequenceMatcher(None, query.casefold(), candidate.casefold()).ratio()


# ---------------------------------------------------------------------------
# Pagination
# ---------------------------------------------------------------------------


def _paginate(
    rows: list[ExplorerMatchRow],
    pagination: PaginationRequest,
) -> tuple[PaginationMeta, list[ExplorerMatchRow]]:
    total = len(rows)
    page_size = pagination.page_size
    page = pagination.page

    if total == 0:
        return PaginationMeta(
            total=0, page=1, page_size=page_size, has_next=False, has_prev=False
        ), []

    total_pages = (total + page_size - 1) // page_size
    page = max(1, min(page, total_pages))
    start = (page - 1) * page_size

    meta = PaginationMeta(
        total=total,
        page=page,
        page_size=page_size,
        has_next=page < total_pages,
        has_prev=page > 1,
    )
    return meta, rows[start : start + page_size]
