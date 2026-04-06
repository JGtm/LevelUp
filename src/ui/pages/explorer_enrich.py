"""Enrichissement de DataFrames pour les tableaux de la page Explorer.

Prépare les colonnes dérivées (traductions, scores, performance)
nécessaires au rendu du tableau HTML partagé.
"""

from __future__ import annotations

import logging

import polars as pl

from src.visualization._compat import ensure_polars

logger = logging.getLogger(__name__)


def enrich_for_table(  # noqa: C901, PLR0912
    dff: pl.DataFrame,
    waypoint_player: str,
    df_full: pl.DataFrame | None,
) -> pl.DataFrame:
    """Enrichit un DataFrame filtré pour le rendu tableau (colonnes dérivées).

    Args:
        dff: DataFrame des matchs à afficher.
        waypoint_player: Gamertag Waypoint pour les liens externes.
        df_full: DataFrame complet pour le calcul du score relatif.

    Returns:
        DataFrame enrichi avec toutes les colonnes nécessaires au tableau.
    """
    from src.analysis.performance_score import compute_performance_series
    from src.ui.date_formats import FMT_DATETIME_FR
    from src.ui.i18n import get_lang, get_outcome_map
    from src.ui.translations import translate_pair_name
    from src.ui.vectorize_helpers import build_mapping

    result = dff.clone()

    if "playlist_fr" not in result.columns and "playlist_name" in result.columns:
        if "playlist_name_fr" in result.columns:
            result = result.with_columns(
                pl.coalesce(
                    [
                        pl.col("playlist_name_fr").cast(pl.Utf8),
                        pl.col("playlist_name").cast(pl.Utf8),
                    ]
                ).alias("playlist_fr")
            )
        else:
            result = result.with_columns(pl.col("playlist_name").alias("playlist_fr"))
    if "mode_ui" not in result.columns and "pair_name" in result.columns:
        if "pair_name_fr" in result.columns:
            result = result.with_columns(
                pl.coalesce(
                    [
                        pl.col("pair_name_fr").cast(pl.Utf8),
                        pl.col("pair_name").cast(pl.Utf8),
                    ]
                ).alias("mode_ui")
            )
        else:
            _ee_mode_map = build_mapping(
                result["pair_name"], lambda x: translate_pair_name(x, lang=get_lang())
            )
            result = result.with_columns(
                pl.col("pair_name")
                .cast(pl.Utf8)
                .replace_strict(
                    _ee_mode_map, default=pl.col("pair_name").cast(pl.Utf8), return_dtype=pl.Utf8
                )
                .alias("mode_ui")
            )
    if "outcome_label" not in result.columns and "outcome" in result.columns:
        result = result.with_columns(
            pl.col("outcome")
            .replace_strict(get_outcome_map(), default=pl.lit("-"))
            .alias("outcome_label")
        )
    if "start_time_fr" not in result.columns and "start_time" in result.columns:
        result = result.with_columns(
            pl.col("start_time").dt.strftime(FMT_DATETIME_FR).fill_null("-").alias("start_time_fr")
        )
    if "score" not in result.columns:
        result = _add_score_column(result)
    if "average_life_mmss" not in result.columns and "average_life_seconds" in result.columns:
        result = _add_avg_life_column(result)
    if "delta_mmr" not in result.columns:
        result = _add_delta_mmr(result)
    if "performance" not in result.columns:
        history = ensure_polars(df_full) if df_full is not None else result
        perf = compute_performance_series(result, history)
        if not isinstance(perf, pl.Series):
            perf = pl.Series("performance", perf.to_list())
        result = result.with_columns(perf.alias("performance"))

    logger.debug("enrich_for_table: %d matchs, colonnes ajoutées OK", len(result))

    # Lien Waypoint
    if "match_url" not in result.columns:
        wp = waypoint_player.strip()
        result = result.with_columns(
            (
                pl.lit(f"https://www.halowaypoint.com/halo-infinite/players/{wp}/matches/")
                + pl.col("match_id").cast(pl.Utf8)
            ).alias("match_url")
        )

    return result


def enrich_common_matches(
    common: pl.DataFrame,
    df_full: pl.DataFrame | None,
) -> pl.DataFrame:
    """Enrichit les matchs communs (recherche joueur) pour le tableau.

    Les matchs communs proviennent de ``load_common_matches`` et n'ont
    pas toutes les colonnes standard. Les colonnes manquantes sont
    remplies avec des valeurs par défaut.

    Args:
        common: DataFrame des matchs communs.
        df_full: DataFrame complet (non utilisé ici, prévu pour extension).

    Returns:
        DataFrame enrichi compatible avec le tableau HTML.
    """
    from src.ui.date_formats import FMT_DATETIME_FR
    from src.ui.i18n import get_lang, get_outcome_map
    from src.ui.translations import translate_pair_name
    from src.ui.vectorize_helpers import build_mapping

    result = common.clone()

    if "outcome_label" not in result.columns and "outcome" in result.columns:
        result = result.with_columns(
            pl.col("outcome")
            .replace_strict(get_outcome_map(), default=pl.lit("-"))
            .alias("outcome_label")
        )
    if "start_time_fr" not in result.columns and "start_time" in result.columns:
        result = result.with_columns(
            pl.col("start_time").dt.strftime(FMT_DATETIME_FR).fill_null("-").alias("start_time_fr")
        )
    # Colonnes manquantes → alias avec priorité FR
    if "playlist_fr" not in result.columns and "playlist_name" in result.columns:
        if "playlist_name_fr" in result.columns:
            result = result.with_columns(
                pl.coalesce(
                    [
                        pl.col("playlist_name_fr").cast(pl.Utf8),
                        pl.col("playlist_name").cast(pl.Utf8),
                    ]
                ).alias("playlist_fr")
            )
        else:
            result = result.with_columns(pl.col("playlist_name").alias("playlist_fr"))
    if "mode_ui" not in result.columns and "pair_name" in result.columns:
        if "pair_name_fr" in result.columns:
            result = result.with_columns(
                pl.coalesce(
                    [
                        pl.col("pair_name_fr").cast(pl.Utf8),
                        pl.col("pair_name").cast(pl.Utf8),
                    ]
                ).alias("mode_ui")
            )
        else:
            _ec_mode_map = build_mapping(
                result["pair_name"], lambda x: translate_pair_name(x, lang=get_lang())
            )
            result = result.with_columns(
                pl.col("pair_name")
                .cast(pl.Utf8)
                .replace_strict(
                    _ec_mode_map, default=pl.col("pair_name").cast(pl.Utf8), return_dtype=pl.Utf8
                )
                .alias("mode_ui")
            )

    # Colonnes absentes → null
    for col_name in (
        "score",
        "team_mmr",
        "enemy_mmr",
        "delta_mmr",
        "max_killing_spree",
        "headshot_kills",
        "average_life_mmss",
        "accuracy",
        "ratio",
        "performance",
        "match_url",
    ):
        if col_name not in result.columns:
            result = result.with_columns(pl.lit(None).alias(col_name))

    return result


# ---------------------------------------------------------------------------
# Helpers colonnes
# ---------------------------------------------------------------------------


def _add_score_column(df: pl.DataFrame) -> pl.DataFrame:
    """Ajoute la colonne score concaténée ``'X - Y'``."""
    if "my_team_score" not in df.columns or "enemy_team_score" not in df.columns:
        return df.with_columns(pl.lit("-").alias("score"))
    return df.with_columns(
        pl.concat_str(
            [
                pl.col("my_team_score")
                .cast(pl.Float64, strict=False)
                .round(0)
                .cast(pl.Int64, strict=False)
                .fill_null(pl.lit("-"))
                .cast(pl.Utf8),
                pl.lit(" - "),
                pl.col("enemy_team_score")
                .cast(pl.Float64, strict=False)
                .round(0)
                .cast(pl.Int64, strict=False)
                .fill_null(pl.lit("-"))
                .cast(pl.Utf8),
            ]
        ).alias("score")
    )


def _add_avg_life_column(df: pl.DataFrame) -> pl.DataFrame:
    """Ajoute la colonne ``average_life_mmss`` formatée ``'M:SS'``."""
    return df.with_columns(
        pl.concat_str(
            [
                (pl.col("average_life_seconds").cast(pl.Int64, strict=False) // 60)
                .fill_null(0)
                .cast(pl.Utf8),
                pl.lit(":"),
                (pl.col("average_life_seconds").cast(pl.Int64, strict=False) % 60)
                .fill_null(0)
                .cast(pl.Utf8)
                .str.zfill(2),
            ]
        ).alias("average_life_mmss")
    )


def _add_delta_mmr(df: pl.DataFrame) -> pl.DataFrame:
    """Ajoute la colonne ``delta_mmr`` (team_mmr − enemy_mmr)."""
    if "team_mmr" not in df.columns:
        return df.with_columns(pl.lit(None).cast(pl.Float64).alias("delta_mmr"))
    return df.with_columns(
        (
            pl.col("team_mmr").cast(pl.Float64, strict=False)
            - pl.col("enemy_mmr").cast(pl.Float64, strict=False)
        ).alias("delta_mmr")
    )
