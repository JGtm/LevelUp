"""Service Synthèse — KPIs solo vs escouade et métriques comparatives."""

from __future__ import annotations

import contextlib
import logging
from datetime import datetime, timedelta, timezone
from pathlib import Path

from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.synthesis import (
    ComparisonMetricItem,
    SynthesisKPIs,
    SynthesisPageResponse,
    SynthesisQueryRequest,
)

logger = logging.getLogger(__name__)

_OUTCOME_WIN = 2
_PERIOD_DAYS: dict[str, int] = {"1w": 7, "1m": 30, "1y": 365, "2y": 730}

# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def get_synthesis_page(
    player: PlayerContext, request: SynthesisQueryRequest
) -> SynthesisPageResponse:
    """Retourne la page de synthèse solo vs escouade."""
    df_all = _load_matches_synthesis(player)

    if df_all is None or (hasattr(df_all, "is_empty") and df_all.is_empty()):
        return SynthesisPageResponse(
            period=request.period,
            total_matches=0,
            solo_kpis=None,
            squad_kpis=None,
            comparison_metrics=[],
        )

    df_filtered = _apply_period_filter(df_all, request.period)
    solo_df, squad_df = _split_solo_squad(df_filtered)

    solo_kpis = _compute_synthesis_kpis(solo_df) if not _is_empty(solo_df) else None
    squad_kpis = _compute_synthesis_kpis(squad_df) if not _is_empty(squad_df) else None
    comparison = _build_comparison_metrics(solo_kpis, squad_kpis)

    return SynthesisPageResponse(
        period=request.period,
        total_matches=len(df_filtered),
        solo_kpis=solo_kpis,
        squad_kpis=squad_kpis,
        comparison_metrics=comparison,
    )


# ---------------------------------------------------------------------------
# Chargement DuckDB
# ---------------------------------------------------------------------------


def _load_matches_synthesis(player: PlayerContext):
    """Charge les matchs avec colonnes nécessaires à la synthèse."""
    try:
        import polars as pl

        from src.utils.db import duckdb_read_only

        db_path = Path(player.db_path)
        shared_path = Path(player.shared_db_path)
        if not db_path.exists():
            return pl.DataFrame()

        with duckdb_read_only(str(db_path)) as conn:
            if shared_path.exists():
                with contextlib.suppress(Exception):
                    conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")

            xuid = _resolve_xuid(conn)
            if not xuid:
                return pl.DataFrame()

            try:
                result = conn.execute(
                    """
                    SELECT
                        r.match_id,
                        r.start_time,
                        COALESCE(p.outcome, 0)                              AS outcome,
                        p.kills,
                        p.deaths,
                        p.accuracy,
                        p.time_played_seconds,
                        p.average_life_seconds,
                        pme.performance_score,
                        COALESCE(pme.is_with_friends, FALSE)                AS is_with_friends
                    FROM shared.match_registry r
                    JOIN shared.match_participants p
                        ON p.match_id = r.match_id AND p.xuid = ?
                    LEFT JOIN player_match_enrichment pme
                        ON pme.match_id = r.match_id
                    ORDER BY r.start_time DESC
                    """,
                    [xuid],
                )
            except Exception:
                result = conn.execute(
                    """
                    SELECT
                        pme.match_id,
                        NULL AS start_time,
                        0 AS outcome,
                        0 AS kills, 0 AS deaths, NULL AS accuracy,
                        NULL AS time_played_seconds, NULL AS average_life_seconds,
                        pme.performance_score,
                        COALESCE(pme.is_with_friends, FALSE) AS is_with_friends
                    FROM player_match_enrichment pme
                    """
                )

            columns = [d[0] for d in result.description]
            rows = result.fetchall()
            if not rows:
                return pl.DataFrame()
            return pl.DataFrame(rows, schema=columns, orient="row")
    except Exception:
        logger.exception("_load_matches_synthesis(%s)", player.player_slug)
        try:
            import polars as pl

            return pl.DataFrame()
        except ImportError:
            return None


# ---------------------------------------------------------------------------
# Filtrage temporel
# ---------------------------------------------------------------------------


def _apply_period_filter(df, period: str):
    """Filtre le DataFrame par période temporelle."""
    try:
        import polars as pl

        if period == "all" or period not in _PERIOD_DAYS:
            return df
        if "start_time" not in df.columns:
            return df

        days = _PERIOD_DAYS[period]
        cutoff = datetime.now(tz=timezone.utc) - timedelta(days=days)
        cutoff_naive = cutoff.replace(tzinfo=None)

        col = df["start_time"]
        if col.dtype == pl.Utf8 or col.dtype == pl.String:
            col = col.cast(pl.Datetime, strict=False)
        filtered = df.filter(col >= cutoff_naive)
        return filtered if not filtered.is_empty() else df
    except Exception:
        logger.debug("_apply_period_filter: erreur", exc_info=True)
        return df


def _split_solo_squad(df):
    """Sépare le DataFrame en sous-ensembles solo et escouade."""
    try:
        import polars as pl

        if "is_with_friends" not in df.columns:
            return df, pl.DataFrame(schema=df.schema)
        solo = df.filter(pl.col("is_with_friends").cast(pl.Boolean, strict=False) == False)  # noqa: E712
        squad = df.filter(pl.col("is_with_friends").cast(pl.Boolean, strict=False) == True)  # noqa: E712
        return solo, squad
    except Exception:
        return df, df


# ---------------------------------------------------------------------------
# Calcul des KPIs
# ---------------------------------------------------------------------------


def _compute_synthesis_kpis(df) -> SynthesisKPIs:
    """Calcule les KPIs de synthèse depuis un DataFrame."""
    try:
        import polars as pl

        total = len(df)
        wins = 0
        if "outcome" in df.columns:
            wins = int(df["outcome"].cast(pl.Int64, strict=False).eq(_OUTCOME_WIN).sum())

        kd_ratio = None
        if "kills" in df.columns and "deaths" in df.columns:
            k = int(df["kills"].cast(pl.Int64, strict=False).fill_null(0).sum())
            d = int(df["deaths"].cast(pl.Int64, strict=False).fill_null(0).sum())
            kd_ratio = round(k / d, 3) if d > 0 else float(k)

        win_rate = round(wins / total, 4) if total else 0.0

        accuracy = None
        if "accuracy" in df.columns:
            vals = df["accuracy"].cast(pl.Float64, strict=False).drop_nulls()
            if not vals.is_empty():
                accuracy = round(float(vals.mean()), 1)

        kills_per_min = None
        if "kills" in df.columns and "time_played_seconds" in df.columns:
            total_k = float(df["kills"].cast(pl.Float64, strict=False).fill_null(0).sum())
            total_s = float(
                df["time_played_seconds"].cast(pl.Float64, strict=False).fill_null(0).sum()
            )
            if total_s > 0:
                kills_per_min = round(total_k / (total_s / 60.0), 3)

        avg_life = None
        if "average_life_seconds" in df.columns:
            vals = df["average_life_seconds"].cast(pl.Float64, strict=False).drop_nulls()
            if not vals.is_empty():
                avg_life = round(float(vals.mean()), 1)

        perf_score = None
        if "performance_score" in df.columns:
            vals = df["performance_score"].cast(pl.Float64, strict=False).drop_nulls()
            if not vals.is_empty():
                perf_score = round(float(vals.mean()), 1)

        return SynthesisKPIs(
            match_count=total,
            wins=wins,
            kd_ratio=kd_ratio,
            win_rate=win_rate,
            accuracy=accuracy,
            kills_per_min=kills_per_min,
            avg_life_seconds=avg_life,
            performance_score=perf_score,
        )
    except Exception:
        return SynthesisKPIs(match_count=len(df), wins=0, win_rate=0.0)


# ---------------------------------------------------------------------------
# Métriques comparatives
# ---------------------------------------------------------------------------


def _build_comparison_metrics(
    solo: SynthesisKPIs | None,
    squad: SynthesisKPIs | None,
) -> list[ComparisonMetricItem]:
    """Construit la liste des métriques comparatives solo vs escouade."""
    metrics: list[ComparisonMetricItem] = []

    def _add(label: str, sv, qv, fmt: str = ".2f") -> None:
        if sv is None and qv is None:
            return
        st = f"{sv:{fmt}}" if sv is not None else "-"
        qt = f"{qv:{fmt}}" if qv is not None else "-"
        metrics.append(
            ComparisonMetricItem(
                label=label,
                solo_value=sv,
                squad_value=qv,
                solo_text=st,
                squad_text=qt,
            )
        )

    _add("K/D", getattr(solo, "kd_ratio", None), getattr(squad, "kd_ratio", None), ".2f")
    _add(
        "Win Rate",
        round(getattr(solo, "win_rate", 0) * 100, 1) if solo else None,
        round(getattr(squad, "win_rate", 0) * 100, 1) if squad else None,
        ".1f",
    )
    _add(
        "Précision (%)",
        getattr(solo, "accuracy", None),
        getattr(squad, "accuracy", None),
        ".1f",
    )
    _add(
        "Kills/min",
        getattr(solo, "kills_per_min", None),
        getattr(squad, "kills_per_min", None),
        ".2f",
    )
    _add(
        "Vie moy. (s)",
        getattr(solo, "avg_life_seconds", None),
        getattr(squad, "avg_life_seconds", None),
        ".0f",
    )
    _add(
        "Perf. Score",
        getattr(solo, "performance_score", None),
        getattr(squad, "performance_score", None),
        ".1f",
    )
    return metrics


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _resolve_xuid(conn) -> str:
    try:
        row = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid'").fetchone()
        return str(row[0]).strip() if row else ""
    except Exception:
        return ""


def _is_empty(df) -> bool:
    if df is None:
        return True
    if hasattr(df, "is_empty"):
        return df.is_empty()
    return len(df) == 0
