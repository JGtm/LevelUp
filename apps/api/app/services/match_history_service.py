"""Service Historique des parties — extraction, enrichissement, pagination.

Toutes les importations src.* sont lazy (dans le corps des fonctions)
pour permettre le mocking en tests.

Architecture :
  1. Charger TOUS les matchs du joueur (colonnes enrichies)
  2. Appliquer le filtre → dff (scoped) + conserver df_full pour win_rate et perf
  3. Enrichir les colonnes : scores, MMR, win_rate_hist, performance, temps
  4. Trier + paginer → MatchHistoryPageResponse
"""

from __future__ import annotations

import contextlib
import logging
from datetime import datetime
from pathlib import Path

from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.common import PaginatedResponse, PaginationMeta, PaginationRequest
from apps.api.app.schemas.filters import FilterContextInput
from apps.api.app.schemas.match_history import (
    ExportHint,
    FileTokenResponse,
    MatchHistoryExportRequest,
    MatchHistoryPageResponse,
    MatchHistoryQueryRequest,
    MatchHistoryQuerySummary,
    MatchHistoryRow,
)

logger = logging.getLogger(__name__)

_OUTCOME_LABELS: dict[int, str] = {2: "Victoire", 3: "Défaite", 1: "Égalité", 4: "Abandon"}
_FMT_DATETIME_FR = "%d/%m/%Y %H:%M"
_DEFAULT_SORT_FIELD = "start_time"
_DEFAULT_SORT_DIR = "desc"
_AVAILABLE_SORT_FIELDS = [
    "start_time",
    "outcome_code",
    "performance_score_relative",
    "team_mmr",
    "delta_mmr",
    "win_rate_hist",
]


# ---------------------------------------------------------------------------
# Points d'entrée publics
# ---------------------------------------------------------------------------


def get_match_history_page(
    player: PlayerContext,
    request: MatchHistoryQueryRequest,
) -> MatchHistoryPageResponse:
    """Construit la réponse paginée pour la page Historique des parties."""

    df_full = _load_matches_full(player)
    total_unfiltered = len(df_full)

    dff = _apply_filter(df_full, player, request.filters) if not df_full.is_empty() else df_full
    total_scoped = len(dff)

    if not dff.is_empty():
        dff = _enrich(dff, df_full, player.waypoint_player)

    rows = _to_rows(dff)
    rows = _sort_rows(rows, _DEFAULT_SORT_FIELD, _DEFAULT_SORT_DIR)

    page_meta, page_rows = _paginate(rows, request.pagination)

    export_hint = None
    if request.include_export_hint and total_scoped > 0:
        export_hint = ExportHint(
            file_name="levelup_matches.csv",
            estimated_rows=total_scoped,
            token=None,
        )

    table: PaginatedResponse[MatchHistoryRow] = PaginatedResponse(
        items=page_rows,
        pagination=page_meta,
    )

    period_label = _build_period_label(request.filters)

    return MatchHistoryPageResponse(
        summary=MatchHistoryQuerySummary(
            total_matches_scoped=total_scoped,
            total_matches_unfiltered=total_unfiltered,
            period_label=period_label,
            active_filter_mode=request.filters.filter_mode,
        ),
        table=table,
        available_sort_fields=_AVAILABLE_SORT_FIELDS,
        export_hint=export_hint,
    )


def get_match_history_export(
    player: PlayerContext,
    request: MatchHistoryExportRequest,
) -> FileTokenResponse:
    """Génère un export CSV (jeton de téléchargement immédiat)."""
    import secrets
    from datetime import timezone

    df_full = _load_matches_full(player)
    dff = _apply_filter(df_full, player, request.filters) if not df_full.is_empty() else df_full
    if not dff.is_empty():
        dff = _enrich(dff, df_full, player.waypoint_player)

    estimated = len(dff)
    token = secrets.token_urlsafe(24)
    expires_at = datetime.now(tz=timezone.utc).replace(second=0, microsecond=0)

    return FileTokenResponse(
        file_token=token,
        file_name="levelup_matches.csv",
        content_type="text/csv",
        download_path=f"/api/v1/downloads/{token}",
        expires_at=expires_at,
        estimated_rows=estimated,
    )


# ---------------------------------------------------------------------------
# Chargement DuckDB
# ---------------------------------------------------------------------------


def _load_matches_full(player: PlayerContext):  # type: ignore[return]
    """Charge tous les matchs du joueur avec les colonnes enrichies nécessaires.

    Retourne un ``pl.DataFrame`` vide en cas d'erreur.
    """
    try:
        import polars as pl

        from src.utils.db import duckdb_read_only
    except ImportError:
        logger.warning("src.utils.db non disponible — match history désactivé")
        try:
            import polars as pl
        except ImportError:
            return []
        return pl.DataFrame()

    db_path = Path(player.db_path)
    shared_path = Path(player.shared_db_path)

    if not db_path.exists():
        logger.warning("DB player introuvable : %s", db_path)
        return pl.DataFrame()

    try:
        with duckdb_read_only(str(db_path)) as conn:
            if shared_path.exists():
                with contextlib.suppress(Exception):
                    conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")

            xuid = _resolve_xuid(conn)
            if not xuid:
                return pl.DataFrame()

            has_mv = _has_mv_player_matches(conn)
            source_sql = _build_source_sql(has_mv)

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
                p.enemy_team_score,
                p.team_mmr,
                p.enemy_mmr,
                COALESCE(p.kills, 0)                                AS kills,
                COALESCE(p.deaths, 0)                               AS deaths,
                COALESCE(p.assists, 0)                              AS assists,
                p.kda,
                p.accuracy,
                p.personal_score,
                p.average_life_seconds,
                p.time_played_seconds
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

        df = pl.DataFrame(rows, schema=columns, orient="row")
        return _add_display_columns(df)

    except Exception:
        logger.exception("Erreur chargement matchs historique pour %s", player.player_slug)
        try:
            import polars as pl

            return pl.DataFrame()
        except ImportError:
            return []


def _resolve_xuid(conn) -> str:  # type: ignore[no-untyped-def]
    """Extrait le xuid depuis sync_meta."""
    try:
        row = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid'").fetchone()
        return str(row[0]).strip() if row else ""
    except Exception:
        return ""


def _has_mv_player_matches(conn) -> bool:  # type: ignore[no-untyped-def]
    """Vérifie si shared.mv_player_matches est disponible."""
    try:
        conn.execute("SELECT 1 FROM shared.mv_player_matches LIMIT 0")
        return True
    except Exception:
        return False


def _build_source_sql(has_mv: bool) -> str:
    """Retourne le sous-SELECT approprié selon la disponibilité de la vue matérialisée."""
    if has_mv:
        return """(
            SELECT match_id, start_time, map_id, map_name, map_name_fr,
                   pair_name, pair_name_fr, playlist_name, playlist_name_fr,
                   is_firefight, is_ranked
            FROM shared.mv_player_matches
            WHERE xuid = ?
        ) AS ms"""
    return """(
        SELECT r.match_id, r.start_time, r.map_id, r.map_name,
               NULL AS map_name_fr,
               r.pair_name,
               NULL AS pair_name_fr,
               r.playlist_name,
               NULL AS playlist_name_fr,
               COALESCE(r.is_firefight, FALSE) AS is_firefight,
               COALESCE(r.is_ranked, FALSE)    AS is_ranked
        FROM shared.match_registry r
        JOIN shared.match_participants p ON r.match_id = p.match_id
        WHERE p.xuid = ?
    ) AS ms"""


def _add_display_columns(df):  # type: ignore[return]
    """Ajoute map_ui, mode_ui, playlist_ui au DataFrame si absentes."""
    try:
        import re

        import polars as pl

        exprs = []
        if "map_ui" not in df.columns and "map_name" in df.columns:
            src = (
                pl.coalesce([pl.col("map_name_fr").cast(pl.Utf8), pl.col("map_name").cast(pl.Utf8)])
                if "map_name_fr" in df.columns
                else pl.col("map_name").cast(pl.Utf8)
            )
            exprs.append(src.alias("map_ui"))

        if "mode_ui" not in df.columns and "pair_name" in df.columns:
            src = (
                pl.coalesce(
                    [pl.col("pair_name_fr").cast(pl.Utf8), pl.col("pair_name").cast(pl.Utf8)]
                )
                if "pair_name_fr" in df.columns
                else pl.col("pair_name").cast(pl.Utf8)
            )

            def _strip_suffix(s: str | None) -> str | None:
                if not s:
                    return None
                s = str(s).strip()
                if " on " in s:
                    s = s.split(" on ", 1)[0].strip()
                s = re.sub(r"\s*-\s*Forge\b", "", s, flags=re.IGNORECASE).strip()
                s = re.sub(r"\s*-\s*Ranked\b", "", s, flags=re.IGNORECASE).strip()
                return s or None

            exprs.append(src.map_elements(_strip_suffix, return_dtype=pl.Utf8).alias("mode_ui"))

        if "playlist_ui" not in df.columns and "playlist_name" in df.columns:
            src = (
                pl.coalesce(
                    [
                        pl.col("playlist_name_fr").cast(pl.Utf8),
                        pl.col("playlist_name").cast(pl.Utf8),
                    ]
                )
                if "playlist_name_fr" in df.columns
                else pl.col("playlist_name").cast(pl.Utf8)
            )
            exprs.append(src.alias("playlist_ui"))

        return df.with_columns(exprs) if exprs else df
    except Exception:
        return df


# ---------------------------------------------------------------------------
# Filtrage
# ---------------------------------------------------------------------------


def _apply_filter(df_full, player: PlayerContext, filters: FilterContextInput):  # type: ignore[return]
    """Applique le contexte de filtre sur le DataFrame complet."""
    try:
        import polars as pl

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

        # Cascade filter works on map_ui / mode_ui / playlist_ui columns
        # Remap playlist_ui if needed
        if "playlist_ui" not in df_temporal.columns and "playlist_name" in df_temporal.columns:
            df_temporal = df_temporal.with_columns(
                pl.col("playlist_name").cast(pl.Utf8).alias("playlist_ui")
            )

        return _apply_cascade_filter(df_temporal, effective.cascade)
    except Exception:
        logger.exception("Erreur lors du filtrage de l'historique")
        return df_full


# ---------------------------------------------------------------------------
# Enrichissement
# ---------------------------------------------------------------------------


def _enrich(dff, df_full, waypoint_player: str):  # type: ignore[return]
    """Enrichit le DataFrame filtré avec les colonnes calculées."""

    dff = _add_score_columns(dff)
    dff = _add_time_columns(dff)
    dff = _add_win_rate_column(dff, df_full)
    dff = _add_performance_column(dff, df_full)
    dff = _add_match_url(dff, waypoint_player)
    return dff


def _add_score_columns(dff):  # type: ignore[return]
    """Ajoute outcome_label, score_label, delta_mmr."""
    import polars as pl

    if "outcome" not in dff.columns:
        dff = dff.with_columns(pl.lit(0).alias("outcome"))

    outcome_map = dict(_OUTCOME_LABELS)
    dff = dff.with_columns(
        pl.col("outcome")
        .cast(pl.Int64, strict=False)
        .replace_strict(outcome_map, default=pl.lit("-"))
        .alias("outcome_label")
    )

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

    if "team_mmr" not in dff.columns:
        dff = dff.with_columns(pl.lit(None).cast(pl.Float64).alias("team_mmr"))
    if "enemy_mmr" not in dff.columns:
        dff = dff.with_columns(pl.lit(None).cast(pl.Float64).alias("enemy_mmr"))

    dff = dff.with_columns(
        (
            pl.col("team_mmr").cast(pl.Float64, strict=False)
            - pl.col("enemy_mmr").cast(pl.Float64, strict=False)
        ).alias("delta_mmr")
    )
    return dff


def _add_time_columns(dff):  # type: ignore[return]
    """Ajoute start_time_label et average_life_mmss."""
    import polars as pl

    col_dtype = dff["start_time"].dtype if "start_time" in dff.columns else None
    if col_dtype in (pl.Utf8, pl.String):
        dff = dff.with_columns(
            pl.col("start_time").str.to_datetime(format="%Y-%m-%d %H:%M:%S", strict=False)
        )

    dff = dff.with_columns(
        pl.col("start_time").dt.strftime(_FMT_DATETIME_FR).fill_null("-").alias("start_time_label")
    )

    avg_life_col = "average_life_seconds" if "average_life_seconds" in dff.columns else None
    if avg_life_col:
        dff = dff.with_columns(
            pl.concat_str(
                [
                    (pl.col(avg_life_col).cast(pl.Int64, strict=False) // 60)
                    .fill_null(0)
                    .cast(pl.Utf8),
                    pl.lit(":"),
                    (pl.col(avg_life_col).cast(pl.Int64, strict=False) % 60)
                    .fill_null(0)
                    .cast(pl.Utf8)
                    .str.zfill(2),
                ]
            ).alias("average_life_mmss")
        )
    else:
        dff = dff.with_columns(pl.lit("0:00").alias("average_life_mmss"))

    return dff


def _add_win_rate_column(dff, df_full):  # type: ignore[return]
    """Calcule win_rate_hist par carte sur l'historique complet."""
    import polars as pl

    if "outcome" not in dff.columns or "map_name" not in dff.columns:
        return dff.with_columns(
            pl.lit(None).cast(pl.Float64).alias("win_rate_hist"),
            pl.lit(None).cast(pl.Int64).alias("win_rate_hist_total"),
        )

    base = df_full if (df_full is not None and not df_full.is_empty()) else dff
    if "map_name" not in base.columns or "outcome" not in base.columns:
        return dff.with_columns(
            pl.lit(None).cast(pl.Float64).alias("win_rate_hist"),
            pl.lit(None).cast(pl.Int64).alias("win_rate_hist_total"),
        )

    map_wr = (
        base.group_by("map_name")
        .agg(
            pl.col("outcome")
            .cast(pl.Int64, strict=False)
            .eq(2)
            .cast(pl.Float64)
            .sum()
            .alias("_wins"),
            pl.len().alias("_total"),
        )
        .with_columns((pl.col("_wins") / pl.col("_total") * 100).round(1).alias("win_rate_hist"))
        .rename({"_total": "win_rate_hist_total"})
        .select(["map_name", "win_rate_hist", "win_rate_hist_total"])
    )
    return dff.join(map_wr, on="map_name", how="left")


def _add_performance_column(dff, df_full):  # type: ignore[return]
    """Ajoute performance_score_relative (entier arrondi)."""
    import polars as pl

    try:
        from src.analysis.performance_score import compute_performance_series

        history_df = df_full if (df_full is not None and not df_full.is_empty()) else dff
        perf_series = compute_performance_series(dff, history_df)
        if not isinstance(perf_series, pl.Series):
            perf_series = pl.Series("performance", perf_series.to_list())
        dff = dff.with_columns(perf_series.alias("_perf_raw"))
        return dff.with_columns(
            pl.col("_perf_raw")
            .round(0)
            .cast(pl.Int64, strict=False)
            .alias("performance_score_relative")
        ).drop("_perf_raw")
    except Exception:
        logger.debug("performance_score indisponible — valeur null", exc_info=True)
        return dff.with_columns(pl.lit(None).cast(pl.Int64).alias("performance_score_relative"))


def _add_match_url(dff, waypoint_player: str):  # type: ignore[return]
    """Ajoute l'URL Waypoint pour chaque match."""
    import polars as pl

    base = "https://www.halowaypoint.com/halo-infinite/players/"
    return dff.with_columns(
        (
            pl.lit(base)
            + pl.lit(waypoint_player.strip())
            + pl.lit("/matches/")
            + pl.col("match_id").cast(pl.Utf8)
        ).alias("match_url")
    )


# ---------------------------------------------------------------------------
# Conversion → MatchHistoryRow
# ---------------------------------------------------------------------------


def _to_rows(dff) -> list[MatchHistoryRow]:
    """Convertit le DataFrame en liste de MatchHistoryRow."""
    if hasattr(dff, "is_empty") and dff.is_empty():
        return []

    rows: list[MatchHistoryRow] = []
    for r in dff.iter_rows(named=True):
        outcome_code = _safe_int(r.get("outcome"))
        rows.append(
            MatchHistoryRow(
                match_id=str(r.get("match_id") or ""),
                start_time=r.get("start_time") or datetime.min,
                start_time_label=str(r.get("start_time_label") or "-"),
                outcome_code=outcome_code,
                outcome_label=str(r.get("outcome_label") or "-"),
                score_label=str(r.get("score_label") or "-"),
                map_ui=str(r.get("map_ui") or r.get("map_name") or "-"),
                mode_ui=str(r.get("mode_ui") or r.get("pair_name") or "-"),
                playlist_label=str(
                    r.get("playlist_ui")
                    or r.get("playlist_name_fr")
                    or r.get("playlist_name")
                    or "-"
                ),
                team_mmr=_safe_float(r.get("team_mmr")),
                enemy_mmr=_safe_float(r.get("enemy_mmr")),
                delta_mmr=_safe_float(r.get("delta_mmr")),
                win_rate_hist=_safe_float(r.get("win_rate_hist")),
                win_rate_hist_total=_safe_int(r.get("win_rate_hist_total")),
                performance_score_relative=_safe_int(r.get("performance_score_relative")),
                average_life_mmss=str(r.get("average_life_mmss") or "0:00"),
                match_url=str(r.get("match_url") or ""),
            )
        )
    return rows


def _sort_rows(
    rows: list[MatchHistoryRow],
    sort_field: str = _DEFAULT_SORT_FIELD,
    direction: str = _DEFAULT_SORT_DIR,
) -> list[MatchHistoryRow]:
    """Trie la liste de MatchHistoryRow selon le champ spécifié."""
    reverse = direction == "desc"
    try:
        return sorted(
            rows,
            key=lambda r: (getattr(r, sort_field) is None, getattr(r, sort_field)),
            reverse=reverse,
        )
    except Exception:
        return rows


# ---------------------------------------------------------------------------
# Pagination
# ---------------------------------------------------------------------------


def _paginate(
    rows: list[MatchHistoryRow],
    pagination: PaginationRequest,
) -> tuple[PaginationMeta, list[MatchHistoryRow]]:
    """Applique la pagination sur la liste de lignes."""
    total = len(rows)
    page_size = pagination.page_size
    page = pagination.page

    if total == 0:
        meta = PaginationMeta(total=0, page=1, page_size=page_size, has_next=False, has_prev=False)
        return meta, []

    total_pages = (total + page_size - 1) // page_size
    page = max(1, min(page, total_pages))
    start = (page - 1) * page_size
    end = start + page_size

    meta = PaginationMeta(
        total=total,
        page=page,
        page_size=page_size,
        has_next=page < total_pages,
        has_prev=page > 1,
    )
    return meta, rows[start:end]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _build_period_label(filters: FilterContextInput) -> str | None:
    """Construit un label lisible pour la période active."""
    if filters.filter_mode == "sessions":
        labels = filters.sessions.picked_sessions or []
        if filters.sessions.picked_session_label:
            labels = [filters.sessions.picked_session_label]
        return f"{len(labels)} session(s)" if labels else None

    p = filters.period
    if p.start_date and p.end_date:
        return f"{p.start_date.strftime('%d/%m/%Y')} → {p.end_date.strftime('%d/%m/%Y')}"
    if p.start_date:
        return f"depuis le {p.start_date.strftime('%d/%m/%Y')}"
    if p.end_date:
        return f"jusqu'au {p.end_date.strftime('%d/%m/%Y')}"
    return None


def _safe_float(val) -> float | None:
    """Convertit en float ou retourne None."""
    if val is None:
        return None
    try:
        return float(val)
    except (ValueError, TypeError):
        return None


def _safe_int(val) -> int | None:
    """Convertit en int ou retourne None."""
    if val is None:
        return None
    try:
        return int(val)
    except (ValueError, TypeError):
        return None
