"""Profils d'intensité normalisés pour heatmap cross-matchs et tempo d'escouade.

Module d'analyse pure — pas d'accès DB ni Streamlit.
"""

from __future__ import annotations

from dataclasses import dataclass

import polars as pl


@dataclass(frozen=True)
class IntensityProfile:
    """Profils d'intensité : colonnes match_id + n_buckets colonnes de kills."""

    df: pl.DataFrame
    n_buckets: int


def compute_match_intensity_profiles(
    events_df: pl.DataFrame,
    n_buckets: int = 10,
) -> IntensityProfile:
    """Calcule le profil d'intensité normalisé pour chaque match.

    Chaque match est découpé en *n_buckets* phases égales (0-10%, 10-20%…).
    La valeur = nombre de kills du joueur dans cette phase.

    Args:
        events_df: DataFrame avec colonnes ``match_id``, ``time_ms``.
                   Pré-filtré : uniquement les kills du joueur.
        n_buckets: Nombre de tranches (défaut 10).

    Returns:
        IntensityProfile avec un DataFrame : une ligne par match,
        colonnes ``match_id`` + ``phase_0`` … ``phase_{n-1}``.
    """
    if events_df.is_empty():
        schema = {"match_id": pl.Utf8}
        schema.update({f"phase_{i}": pl.Int32 for i in range(n_buckets)})
        return IntensityProfile(df=pl.DataFrame(schema=schema), n_buckets=n_buckets)

    # Durée max par match pour normaliser
    durations = events_df.group_by("match_id").agg(
        pl.col("time_ms").max().alias("max_ms")
    )

    enriched = events_df.join(durations, on="match_id")

    # Phase = floor(time_ms / max_ms * n_buckets), clippé à n_buckets - 1
    enriched = enriched.with_columns(
        pl.when(pl.col("max_ms") > 0)
        .then(
            (pl.col("time_ms").cast(pl.Float64) / pl.col("max_ms") * n_buckets)
            .floor()
            .cast(pl.Int32)
            .clip(0, n_buckets - 1)
        )
        .otherwise(pl.lit(0))
        .alias("phase")
    )

    # Pivot : count par match × phase
    counts = enriched.group_by(["match_id", "phase"]).len().rename({"len": "kills"})

    pivoted = counts.pivot(on="phase", index="match_id", values="kills").fill_null(0)

    # Renommer les colonnes numériques en phase_0, phase_1, …
    rename_map = {}
    for col in pivoted.columns:
        if col != "match_id":
            rename_map[col] = f"phase_{col}"
    pivoted = pivoted.rename(rename_map)

    # Ajouter les colonnes manquantes (phases sans kills)
    for i in range(n_buckets):
        col_name = f"phase_{i}"
        if col_name not in pivoted.columns:
            pivoted = pivoted.with_columns(pl.lit(0).cast(pl.Int32).alias(col_name))

    # Réordonner : match_id, phase_0, phase_1, …
    ordered_cols = ["match_id"] + [f"phase_{i}" for i in range(n_buckets)]
    pivoted = pivoted.select(ordered_cols)

    return IntensityProfile(df=pivoted, n_buckets=n_buckets)


def compute_squad_cadence_profiles(
    events_df: pl.DataFrame,
    xuid_name_map: dict[str, str],
    n_buckets: int = 10,
) -> pl.DataFrame:
    """Calcule le profil de tempo moyen par joueur sur les matchs communs.

    Args:
        events_df: DataFrame avec colonnes ``match_id``, ``time_ms``, ``xuid``.
                   Pré-filtré : event_type='kill' uniquement.
        xuid_name_map: Mapping {xuid: display_name} des joueurs à analyser.
        n_buckets: Nombre de tranches normalisées.

    Returns:
        DataFrame avec colonnes ``phase`` (0 à n_buckets-1) + une colonne
        par joueur (display_name) contenant la moyenne de kills par phase.
    """
    if events_df.is_empty() or not xuid_name_map:
        return pl.DataFrame()

    xuids = list(xuid_name_map.keys())
    filtered = events_df.filter(pl.col("xuid").is_in(xuids))
    if filtered.is_empty():
        return pl.DataFrame()

    # Durée max par match (tous joueurs confondus pour l'axe temps partagé)
    durations = events_df.group_by("match_id").agg(
        pl.col("time_ms").max().alias("max_ms")
    )

    enriched = filtered.join(durations, on="match_id")
    enriched = enriched.with_columns(
        pl.when(pl.col("max_ms") > 0)
        .then(
            (pl.col("time_ms").cast(pl.Float64) / pl.col("max_ms") * n_buckets)
            .floor()
            .cast(pl.Int32)
            .clip(0, n_buckets - 1)
        )
        .otherwise(pl.lit(0))
        .alias("phase")
    )

    # Comptage par joueur × match × phase
    per_match = enriched.group_by(["xuid", "match_id", "phase"]).len().rename({"len": "kills"})

    # Moyenne par joueur × phase
    avg_by_phase = per_match.group_by(["xuid", "phase"]).agg(
        pl.col("kills").mean().alias("avg_kills")
    )

    # Pivot par joueur
    result = pl.DataFrame({"phase": list(range(n_buckets))}).cast({"phase": pl.Int32})

    for xuid, name in xuid_name_map.items():
        player_data = avg_by_phase.filter(pl.col("xuid") == xuid).select(["phase", "avg_kills"])
        player_data = player_data.rename({"avg_kills": name}).cast({"phase": pl.Int32})
        result = result.join(player_data, on="phase", how="left")

    # Fill nulls after all joins to avoid type promotion issues
    for name in xuid_name_map.values():
        if name in result.columns:
            result = result.with_columns(pl.col(name).fill_null(0.0))

    return result.sort("phase")
