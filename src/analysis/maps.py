"""Analyse par carte (map)."""

import polars as pl

from src.analysis.stats import compute_global_ratio, compute_outcome_rates
from src.data.domain.models.stats import MapBreakdown
from src.utils.polars_compat import ensure_polars as _to_polars


def _empty_map_breakdown_schema() -> dict[str, pl.DataType]:
    return {
        "map_name": pl.Utf8,
        "matches": pl.Int64,
        "accuracy_avg": pl.Float64,
        "win_rate": pl.Float64,
        "loss_rate": pl.Float64,
        "ratio_global": pl.Float64,
        "performance_avg": pl.Float64,
    }


def _resolve_map_names(df: pl.DataFrame) -> pl.DataFrame:
    """Remplace les map_name égaux à map_id par un nom résolu disponible."""
    if "map_id" not in df.columns:
        return df

    resolved = (
        df.filter(
            pl.col("map_name").is_not_null()
            & (pl.col("map_name") != pl.col("map_id"))
            & (pl.col("map_name").str.strip_chars() != "")
        )
        .select(["map_id", "map_name"])
        .unique(subset=["map_id"])
    )
    if resolved.is_empty():
        return df

    return (
        df.join(
            resolved.rename({"map_name": "_resolved_name"}),
            on="map_id",
            how="left",
        )
        .with_columns(
            pl.when(pl.col("map_name") == pl.col("map_id"))
            .then(pl.col("_resolved_name"))
            .otherwise(pl.col("map_name"))
            .alias("map_name")
        )
        .drop("_resolved_name")
    )


def _performance_avg(g: pl.DataFrame) -> float | None:
    """Retourne la moyenne de performance_score, ou None si absente."""
    if "performance_score" not in g.columns:
        return None
    perf_vals = g["performance_score"].drop_nulls()
    return float(perf_vals.mean()) if len(perf_vals) > 0 else None


def compute_map_breakdown(df: pl.DataFrame, df_history: pl.DataFrame | None = None) -> pl.DataFrame:
    """Calcule les statistiques agrégées par carte.

    Args:
        df: DataFrame Polars de matchs.
        df_history: DataFrame complet Polars pour le calcul du score relatif.

    Returns:
        DataFrame Polars avec colonnes:
        - map_name
        - matches
        - accuracy_avg
        - win_rate
        - loss_rate
        - ratio_global
        - performance_avg
    """
    _ = df_history  # Compat API: le fallback basé history a été supprimé.
    df_pl = _to_polars(df)
    empty_schema = _empty_map_breakdown_schema()

    if df_pl.is_empty():
        return pl.DataFrame(schema=empty_schema)

    df_pl = _resolve_map_names(df_pl)

    # Filtrer les map_ui vides
    d = df_pl.with_columns(pl.col("map_ui").fill_null(""))
    d = d.filter(pl.col("map_ui").str.strip_chars() != "")

    if d.is_empty():
        return pl.DataFrame(schema=empty_schema)

    rows: list[dict] = []
    for map_name in d.select("map_ui").unique().to_series().to_list():
        g = d.filter(pl.col("map_ui") == map_name)

        rates = compute_outcome_rates(g)
        total_out = max(1, rates.total)

        acc: float | None = None
        if "accuracy" in g.columns:
            acc_val = g.select(pl.col("accuracy").cast(pl.Float64, strict=False).mean()).item()
            acc = float(acc_val) if acc_val is not None else None

        rows.append(
            {
                "map_name": map_name,
                "matches": int(len(g)),
                "accuracy_avg": acc,
                "win_rate": rates.wins / total_out if rates.total else None,
                "loss_rate": rates.losses / total_out if rates.total else None,
                "ratio_global": compute_global_ratio(g),
                "performance_avg": _performance_avg(g),
            }
        )

    out = pl.DataFrame(rows)
    out = out.sort(["matches", "ratio_global"], descending=[True, True])
    return out


def map_breakdown_to_models(df: pl.DataFrame) -> list[MapBreakdown]:
    """Convertit un DataFrame de breakdown en liste de MapBreakdown.

    Args:
        df: DataFrame Polars issu de compute_map_breakdown.

    Returns:
        Liste de MapBreakdown.
    """
    df_pl = _to_polars(df)

    return [
        MapBreakdown(
            map_name=row["map_name"],
            matches=int(row["matches"]),
            accuracy_avg=row["accuracy_avg"],
            win_rate=row["win_rate"],
            loss_rate=row["loss_rate"],
            ratio_global=row["ratio_global"],
        )
        for row in df_pl.iter_rows(named=True)
    ]
