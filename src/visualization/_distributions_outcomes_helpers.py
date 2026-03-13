"""Helpers internes pour distributions_outcomes.

Contient les fonctions utilitaires et de préparation de données
partagées par les graphiques de résultats (outcomes).
"""

from __future__ import annotations

from datetime import timedelta

import plotly.graph_objects as go
import polars as pl

from src.config import OUTCOME_CODES, SESSION_CONFIG
from src.ui.date_formats import FMT_DATE_ISO
from src.ui.i18n.viz import viz_t


def ensure_datetime(df: pl.DataFrame, col: str) -> pl.DataFrame:
    """Convertit *col* en Datetime si nécessaire (tolérance String)."""
    dtype = df.schema[col]
    if dtype == pl.String:
        return df.with_columns(pl.col(col).str.to_datetime(strict=False))
    return df


def safe_col(df: pl.DataFrame, col_name: str, default: int = 0) -> list:
    """Retourne la colonne *col_name* en liste, ou une liste de *default*."""
    if col_name in df.columns:
        return df[col_name].to_list()
    return [default] * df.height


def compute_outcome_buckets(
    d: pl.DataFrame, *, session_style: bool, lang: str = "fr"
) -> tuple[pl.DataFrame, str]:
    """Détermine le bucket temporel et retourne (df_avec_bucket, bucket_label)."""
    if session_style:
        d = d.sort("start_time")
        if d.height <= 20:
            d = d.with_row_index("bucket").with_columns((pl.col("bucket") + 1).cast(pl.Int64))
            return d, viz_t("bucket_match", lang)
        d = ensure_datetime(d, "start_time")
        d = d.with_columns(pl.col("start_time").dt.truncate("1h").alias("bucket"))
        return d, viz_t("bucket_hour", lang)

    d = ensure_datetime(d, "start_time")
    ts = d["start_time"].drop_nulls()
    tmin = ts.min() if ts.len() > 0 else None
    tmax = ts.max() if ts.len() > 0 else None

    dt_range = tmax - tmin if tmin is not None and tmax is not None else timedelta(days=999)
    days = dt_range.total_seconds() / 86400.0

    cfg = SESSION_CONFIG
    if days < cfg.bucket_threshold_hourly:
        d = d.sort("start_time")
        d = d.with_row_index("bucket").with_columns((pl.col("bucket") + 1).cast(pl.Int64))
        return d, viz_t("bucket_match", lang)
    if days <= cfg.bucket_threshold_daily:
        d = d.with_columns(pl.col("start_time").dt.truncate("1h").alias("bucket"))
        return d, viz_t("bucket_hour", lang)
    if days <= cfg.bucket_threshold_weekly:
        d = d.with_columns(pl.col("start_time").dt.strftime(FMT_DATE_ISO).alias("bucket"))
        return d, viz_t("bucket_day", lang)
    if days <= cfg.bucket_threshold_monthly:
        d = d.with_columns(
            pl.col("start_time").dt.truncate("1w").dt.strftime(FMT_DATE_ISO).alias("bucket")
        )
        return d, viz_t("bucket_week", lang)
    d = d.with_columns(pl.col("start_time").dt.strftime("%Y-%m").alias("bucket"))
    return d, viz_t("bucket_month", lang)


def build_outcome_pivot(
    d: pl.DataFrame,
    category_col: str,
    min_matches: int,
    sort_by: str,
    max_categories: int,
) -> pl.DataFrame | None:
    """Construit le pivot agrégé par catégorie. Retourne None si vide."""
    pivot = (
        d.group_by(category_col, "outcome")
        .agg(pl.col("match_id").count().alias("count"))
        .pivot(on="outcome", index=category_col, values="count")
        .fill_null(0)
    )

    # Mapper les codes outcome → noms lisibles
    outcome_map = {
        str(OUTCOME_CODES.WIN): "wins",
        str(OUTCOME_CODES.LOSS): "losses",
        str(OUTCOME_CODES.TIE): "ties",
        str(OUTCOME_CODES.NO_FINISH): "left",
    }
    for code_str, name in outcome_map.items():
        if code_str in pivot.columns:
            pivot = pivot.rename({code_str: name})
        else:
            pivot = pivot.with_columns(pl.lit(0).alias(name))

    # Garder uniquement les colonnes nécessaires
    keep = {category_col, "wins", "losses", "ties", "left"}
    pivot = pivot.select([c for c in pivot.columns if c in keep])

    pivot = pivot.with_columns(
        (pl.col("wins") + pl.col("losses") + pl.col("ties") + pl.col("left")).alias("total"),
    ).with_columns(
        (pl.col("wins").cast(pl.Float64) / pl.col("total")).fill_null(0.0).alias("win_rate"),
    )

    pivot = pivot.filter(pl.col("total") >= min_matches)
    if pivot.is_empty():
        return None
    if sort_by == "win_rate":
        pivot = pivot.sort("win_rate", descending=True)
    elif sort_by == "name":
        pivot = pivot.sort(category_col)
    else:
        pivot = pivot.sort("total", descending=True)
    return pivot.head(max_categories)


def _add_sparse_bar_trace(  # noqa: PLR0913
    fig: go.Figure,
    cats: list,
    pivot: pl.DataFrame,
    col: str,
    name: str,
    color: str,
    opacity: float,
) -> None:
    """Ajoute une barre conditionnelle (si somme > 0) avec texte masqué pour zéros."""
    if pivot[col].sum() <= 0:
        return
    text_vals = (
        pivot.select(
            pl.when(pl.col(col) > 0).then(pl.col(col).cast(pl.String)).otherwise(pl.lit(""))
        )
        .to_series()
        .to_list()
    )
    fig.add_trace(
        go.Bar(
            x=cats,
            y=pivot[col].to_list(),
            name=name,
            marker_color=color,
            opacity=opacity,
            text=text_vals,
            textposition="inside",
            hovertemplate=f"%{{x}}<br>{name}: %{{y}}<extra></extra>",
        )
    )


def add_outcome_traces(
    fig: go.Figure,
    pivot: pl.DataFrame,
    colors: dict,
    *,
    category_col: str,
    lang: str = "fr",
) -> None:
    """Ajoute les traces Victoires / Défaites / Égalités / Non terminés."""
    cats = pivot[category_col].to_list()
    _wins_lbl = viz_t("trace_wins", lang)
    _losses_lbl = viz_t("trace_losses", lang)
    _ties_lbl = viz_t("trace_ties", lang)
    _unfinished_lbl = viz_t("trace_unfinished", lang)
    _win_rate_lbl = viz_t("hover_win_rate", lang)

    fig.add_trace(
        go.Bar(
            x=cats,
            y=pivot["wins"].to_list(),
            name=_wins_lbl,
            marker_color=colors["green"],
            opacity=0.85,
            text=pivot["wins"].to_list(),
            textposition="inside",
            hovertemplate=f"%{{x}}<br>{_wins_lbl}: %{{y}}<br>{_win_rate_lbl}: %{{customdata:.1%}}<extra></extra>",
            customdata=pivot["win_rate"].to_list(),
        )
    )
    fig.add_trace(
        go.Bar(
            x=cats,
            y=pivot["losses"].to_list(),
            name=_losses_lbl,
            marker_color=colors["red"],
            opacity=0.75,
            text=pivot["losses"].to_list(),
            textposition="inside",
            hovertemplate=f"%{{x}}<br>{_losses_lbl}: %{{y}}<extra></extra>",
        )
    )
    _add_sparse_bar_trace(fig, cats, pivot, "ties", _ties_lbl, colors["amber"], 0.70)
    _add_sparse_bar_trace(fig, cats, pivot, "left", _unfinished_lbl, colors["violet"], 0.60)


def determine_top_period(d: pl.DataFrame, lang: str = "fr") -> tuple[pl.DataFrame, str]:
    """Ajoute une colonne 'period' et retourne (df, period_label)."""
    d = ensure_datetime(d, "start_time")
    d = d.drop_nulls(subset=["start_time"])
    ts = d["start_time"]
    tmin = ts.min() if ts.len() > 0 else None
    tmax = ts.max() if ts.len() > 0 else None

    dt_range = tmax - tmin if tmin is not None and tmax is not None else timedelta(days=999)
    days = dt_range.total_seconds() / 86400.0

    if days < 2:
        d = d.sort("start_time")
        d = d.with_row_index("period").with_columns(pl.col("period").cast(pl.String))
        return d, viz_t("bucket_cap_match", lang)
    if days < 7:
        d = d.with_columns(pl.col("start_time").dt.strftime(FMT_DATE_ISO).alias("period"))
        return d, viz_t("bucket_cap_day", lang)
    d = d.with_columns(
        pl.col("start_time").dt.truncate("1w").dt.strftime(FMT_DATE_ISO).alias("period")
    )
    return d, viz_t("bucket_cap_week", lang)
