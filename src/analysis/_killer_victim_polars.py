"""Fonctions d'analyse killer→victim basées sur Polars.

Sprint 3 — Fonctions utilisant les données de killer_victim_pairs
stockées en DuckDB pour calculer antagonistes, timeseries, duels
et matrices.
"""

from __future__ import annotations

from dataclasses import dataclass

import polars as pl


@dataclass(frozen=True)
class AntagonistsResultPolars:
    """Résultat Némésis / Souffre-douleur calculé avec Polars.

    Version simplifiée pour les données provenant de killer_victim_pairs DuckDB.
    """

    nemesis_xuid: str | None
    nemesis_gamertag: str | None
    nemesis_times_killed_by: int
    victim_xuid: str | None
    victim_gamertag: str | None
    victim_times_killed: int
    total_deaths: int
    total_kills: int


_EMPTY_ANTAGONISTS = AntagonistsResultPolars(
    nemesis_xuid=None,
    nemesis_gamertag=None,
    nemesis_times_killed_by=0,
    victim_xuid=None,
    victim_gamertag=None,
    victim_times_killed=0,
    total_deaths=0,
    total_kills=0,
)


def _extract_top_row(
    df: pl.DataFrame, xuid_col: str, gt_col: str, count_col: str
) -> tuple[str | None, str | None, int]:
    """Extrait xuid, gamertag et count de la première ligne d'un DataFrame."""
    if len(df) == 0:
        return None, None, 0
    row = df.row(0, named=True)
    return row[xuid_col], row[gt_col], row[count_col]


def compute_personal_antagonists_from_pairs_polars(
    pairs_df: pl.DataFrame,
    me_xuid: str,
) -> AntagonistsResultPolars:
    """Calcule antagonistes (némésis/souffre-douleur) avec Polars.

    Args:
        pairs_df: DataFrame Polars avec colonnes killer_xuid, killer_gamertag,
                  victim_xuid, victim_gamertag, kill_count.
        me_xuid: XUID du joueur principal.

    Returns:
        AntagonistsResultPolars avec némésis et souffre-douleur.
    """
    if pairs_df.is_empty():
        return _EMPTY_ANTAGONISTS

    # Némésis : qui m'a le plus tué
    nemesis_df = (
        pairs_df.filter(pl.col("victim_xuid") == me_xuid)
        .group_by("killer_xuid", "killer_gamertag")
        .agg(pl.col("kill_count").sum().alias("times_killed_by"))
        .sort("times_killed_by", descending=True)
        .head(1)
    )

    # Souffre-douleur : qui j'ai le plus tué
    victim_df = (
        pairs_df.filter(pl.col("killer_xuid") == me_xuid)
        .group_by("victim_xuid", "victim_gamertag")
        .agg(pl.col("kill_count").sum().alias("times_killed"))
        .sort("times_killed", descending=True)
        .head(1)
    )

    total_deaths = (
        pairs_df.filter(pl.col("victim_xuid") == me_xuid).select(pl.col("kill_count").sum()).item()
    ) or 0

    total_kills = (
        pairs_df.filter(pl.col("killer_xuid") == me_xuid).select(pl.col("kill_count").sum()).item()
    ) or 0

    nem_xuid, nem_gt, nem_count = _extract_top_row(
        nemesis_df, "killer_xuid", "killer_gamertag", "times_killed_by"
    )
    vic_xuid, vic_gt, vic_count = _extract_top_row(
        victim_df, "victim_xuid", "victim_gamertag", "times_killed"
    )

    return AntagonistsResultPolars(
        nemesis_xuid=nem_xuid,
        nemesis_gamertag=nem_gt,
        nemesis_times_killed_by=nem_count,
        victim_xuid=vic_xuid,
        victim_gamertag=vic_gt,
        victim_times_killed=vic_count,
        total_deaths=total_deaths,
        total_kills=total_kills,
    )


def killer_victim_counts_long_polars(pairs_df: pl.DataFrame) -> pl.DataFrame:
    """Retourne un DataFrame Polars agrégé: killer, victim, count.

    Args:
        pairs_df: DataFrame Polars avec colonnes killer_xuid, killer_gamertag,
                  victim_xuid, victim_gamertag, kill_count.

    Returns:
        DataFrame Polars trié par count desc.
    """
    if pairs_df.is_empty():
        return pairs_df

    return (
        pairs_df.group_by("killer_xuid", "killer_gamertag", "victim_xuid", "victim_gamertag")
        .agg(pl.col("kill_count").sum().alias("count"))
        .sort(
            ["count", "killer_gamertag", "victim_gamertag"],
            descending=[True, False, False],
        )
    )


def _build_kd_by_minute(pairs_df: pl.DataFrame, me_xuid: str) -> tuple[pl.DataFrame, pl.DataFrame]:
    """Agrège kills et deaths par minute pour un joueur."""
    my_kills = (
        pairs_df.filter(pl.col("killer_xuid") == me_xuid)
        .with_columns((pl.col("time_ms") // 60000).alias("minute"))
        .group_by("minute")
        .agg(pl.col("kill_count").sum().alias("kills"))
    )
    my_deaths = (
        pairs_df.filter(pl.col("victim_xuid") == me_xuid)
        .with_columns((pl.col("time_ms") // 60000).alias("minute"))
        .group_by("minute")
        .agg(pl.col("kill_count").sum().alias("deaths"))
    )
    return my_kills, my_deaths


_EMPTY_KD_TS = pl.DataFrame(
    {
        "minute": [],
        "kills": [],
        "deaths": [],
        "net_kd": [],
        "cumulative_net_kd": [],
    }
)


def compute_kd_timeseries_by_minute_polars(
    pairs_df: pl.DataFrame,
    me_xuid: str,
    *,
    match_duration_ms: int | None = None,
) -> pl.DataFrame:
    """Calcule le K/D cumulé par minute avec Polars.

    Args:
        pairs_df: DataFrame Polars avec colonnes killer_xuid, victim_xuid, time_ms.
        me_xuid: XUID du joueur principal.
        match_duration_ms: Durée du match en ms (optionnel).

    Returns:
        DataFrame Polars avec colonnes minute, kills, deaths, net_kd,
        cumulative_net_kd.
    """
    if pairs_df.is_empty():
        return _EMPTY_KD_TS

    my_kills, my_deaths = _build_kd_by_minute(pairs_df, me_xuid)

    all_minutes: set[int] = set()
    if len(my_kills) > 0:
        all_minutes.update(my_kills["minute"].to_list())
    if len(my_deaths) > 0:
        all_minutes.update(my_deaths["minute"].to_list())
    if match_duration_ms:
        all_minutes.update(range(match_duration_ms // 60000 + 1))
    if not all_minutes:
        return _EMPTY_KD_TS

    minutes_df = pl.DataFrame({"minute": sorted(all_minutes)})

    return (
        minutes_df.join(my_kills, on="minute", how="left")
        .join(my_deaths, on="minute", how="left")
        .with_columns(
            pl.col("kills").fill_null(0),
            pl.col("deaths").fill_null(0),
        )
        .with_columns((pl.col("kills") - pl.col("deaths")).alias("net_kd"))
        .with_columns(pl.col("net_kd").cum_sum().alias("cumulative_net_kd"))
        .sort("minute")
    )


_EMPTY_DUEL = pl.DataFrame({"match_id": [], "my_kills": [], "opponent_kills": [], "net": []})


def compute_duel_history_polars(
    pairs_df: pl.DataFrame,
    me_xuid: str,
    opponent_xuid: str,
) -> pl.DataFrame:
    """Calcule l'historique des duels entre deux joueurs avec Polars.

    Args:
        pairs_df: DataFrame Polars avec colonnes match_id, killer_xuid,
                  victim_xuid, time_ms.
        me_xuid: XUID du joueur principal.
        opponent_xuid: XUID de l'adversaire.

    Returns:
        DataFrame Polars avec colonnes match_id, my_kills, opponent_kills, net.
    """
    if pairs_df.is_empty():
        return _EMPTY_DUEL

    my_kills = (
        pairs_df.filter(
            (pl.col("killer_xuid") == me_xuid) & (pl.col("victim_xuid") == opponent_xuid)
        )
        .group_by("match_id")
        .agg(pl.col("kill_count").sum().alias("my_kills"))
    )

    opponent_kills = (
        pairs_df.filter(
            (pl.col("killer_xuid") == opponent_xuid) & (pl.col("victim_xuid") == me_xuid)
        )
        .group_by("match_id")
        .agg(pl.col("kill_count").sum().alias("opponent_kills"))
    )

    all_matches: set[str] = set()
    if len(my_kills) > 0:
        all_matches.update(my_kills["match_id"].to_list())
    if len(opponent_kills) > 0:
        all_matches.update(opponent_kills["match_id"].to_list())
    if not all_matches:
        return _EMPTY_DUEL

    matches_df = pl.DataFrame({"match_id": list(all_matches)})

    return (
        matches_df.join(my_kills, on="match_id", how="left")
        .join(opponent_kills, on="match_id", how="left")
        .with_columns(
            pl.col("my_kills").fill_null(0),
            pl.col("opponent_kills").fill_null(0),
        )
        .with_columns((pl.col("my_kills") - pl.col("opponent_kills")).alias("net"))
    )


def killer_victim_matrix_polars(pairs_df: pl.DataFrame) -> pl.DataFrame:
    """Retourne une matrice killer/victim en format pivot avec Polars.

    Args:
        pairs_df: DataFrame Polars avec colonnes killer_gamertag,
                  victim_gamertag, kill_count.

    Returns:
        DataFrame Polars pivotée avec gamertags en lignes et colonnes.
    """
    if pairs_df.is_empty():
        return pairs_df

    aggregated = pairs_df.group_by("killer_gamertag", "victim_gamertag").agg(
        pl.col("kill_count").sum().alias("count")
    )

    return aggregated.pivot(
        values="count",
        index="killer_gamertag",
        on="victim_gamertag",
        aggregate_function="sum",
    ).fill_null(0)
