"""Helpers de vectorisation — remplace map_elements() par des opérations Polars natives.

Sprint 8ter : Éradication des map_elements() dans src/.

Stratégie : pour les fonctions de traduction/normalisation complexes
(translate_pair_name, normalize_mode_label, etc.), on pré-calcule un
mapping exhaustif sur les valeurs distinctes de la colonne, puis on
applique `replace_strict`.
"""

from __future__ import annotations

from collections.abc import Callable

import polars as pl


def vectorized_apply(
    series_or_expr: pl.Series | pl.Expr,
    fn: Callable[[str | None], str | None],
    *,
    default: str | None = None,
    column_name: str | None = None,
) -> pl.Expr:
    """Vectorise une fonction Python sur une colonne Polars via replace_strict.

    Extrait les valeurs distinctes, calcule le mapping une fois,
    puis applique `replace_strict`.

    Pour utiliser sur un DataFrame::

        df = df.with_columns(
            vectorized_apply(df["pair_name"], translate_pair_name).alias("pair_fr")
        )

    Args:
        series_or_expr: La Series source (pour extraire les valeurs distinctes).
        fn: Fonction Python à appliquer (str | None → str | None).
        default: Valeur par défaut si pas de mapping.
        column_name: Nom de colonne si series_or_expr est une Expr (non recommandé).
    """
    if isinstance(series_or_expr, pl.Expr):
        raise TypeError(
            "vectorized_apply requiert une pl.Series, pas une Expr. "
            "Utiliser build_mapping() + replace_strict() pour les Expr."
        )

    distinct_vals = series_or_expr.drop_nulls().unique().to_list()
    mapping: dict[str, str] = {}
    for v in distinct_vals:
        result = fn(v)
        if result is not None:
            mapping[str(v)] = str(result)

    col_ref = pl.col(series_or_expr.name)
    expr = col_ref.cast(pl.Utf8).replace_strict(mapping, default=default, return_dtype=pl.Utf8)
    return expr


def build_mapping(
    series: pl.Series,
    fn: Callable[[str | None], str | None],
) -> dict[str, str]:
    """Construit un dict de mapping à partir des valeurs distinctes d'une Series.

    Utile quand on a besoin du mapping pour plusieurs colonnes ou pour
    des paramètres callbacks.

    Args:
        series: Colonne source.
        fn: Fonction de transformation.

    Returns:
        Dict {valeur_source: valeur_traduite} (sans les None).
    """
    distinct_vals = series.drop_nulls().unique().to_list()
    mapping: dict[str, str] = {}
    for v in distinct_vals:
        result = fn(v)
        if result is not None:
            mapping[str(v)] = str(result)
    return mapping


# ---------------------------------------------------------------------------
#  Expressions réutilisables
# ---------------------------------------------------------------------------

_UUID_PATTERN = r"^[a-f0-9]{8}(-[a-f0-9]{4}){0,3}(-[a-f0-9]{1,12})?$"


def is_uuid_expr(col: str = "value") -> pl.Expr:
    """Retourne une expression booléenne True si la valeur ressemble à un UUID."""
    return pl.col(col).cast(pl.Utf8).str.to_lowercase().str.contains(_UUID_PATTERN)


def safe_int_format(col: str) -> pl.Expr:
    """Formate une colonne numérique comme entier, '-' si null/NaN."""
    return (
        pl.when(pl.col(col).is_null() | pl.col(col).is_nan())
        .then(pl.lit("-"))
        .otherwise(pl.col(col).round(0).cast(pl.Int64).cast(pl.Utf8))
    )


def format_score_pair(my_col: str, enemy_col: str) -> pl.Expr:
    """Formate 'my_score - enemy_score' avec gestion null/NaN."""
    return safe_int_format(my_col) + pl.lit(" - ") + safe_int_format(enemy_col)
