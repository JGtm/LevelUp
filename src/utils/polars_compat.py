"""
Utilitaires de compatibilité Polars.
(Polars compatibility utilities)

Pont de sécurité pour convertir en pl.DataFrame tout objet compatible
(Pandas DataFrame, etc.). Centralise la logique auparavant dupliquée
dans 9 modules.
"""

from __future__ import annotations

from collections.abc import Callable

import polars as pl


def ensure_polars(df: object) -> pl.DataFrame:
    """Convertit un DataFrame Pandas en Polars si nécessaire (pont de sécurité).

    - Si *df* est déjà un ``pl.DataFrame``, il est retourné tel quel.
    - Sinon, tente ``pl.from_pandas(df)``.
    - En cas d'échec, retourne un ``pl.DataFrame`` vide.

    Args:
        df: Objet à convertir (idéalement un DataFrame Pandas ou Polars).

    Returns:
        ``pl.DataFrame`` — garanti non-None.
    """
    if isinstance(df, pl.DataFrame):
        return df
    try:
        return pl.from_pandas(df)  # type: ignore[arg-type]
    except Exception:
        return pl.DataFrame()


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
