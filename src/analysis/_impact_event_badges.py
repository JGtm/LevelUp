"""Badges basés sur les événements highlight : First Blood, Clutch Finisher, etc.

Ces 5 fonctions identifient des événements d'impact à partir des données
highlight_events (kills / deaths horodatés).
"""

from __future__ import annotations

import logging

import polars as pl

from src.analysis._impact_types import OUTCOME_LOSS, OUTCOME_WIN, ImpactEvent

logger = logging.getLogger(__name__)


def identify_first_blood(
    events_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> dict[str, ImpactEvent]:
    """Identifie le joueur avec le premier kill par match.

    Le First Blood est le kill avec le timestamp le plus bas (min time_ms)
    dans chaque match, toutes équipes confondues.
    Si friend_xuids est fourni, le badge n'est retenu que si le killer est un ami.

    Args:
        events_df: DataFrame Polars avec colonnes :
            - match_id, xuid, gamertag, event_type, time_ms
        friend_xuids: Set d'XUIDs à filtrer (optionnel). Si fourni, seuls les matchs
            où un ami a obtenu le premier kill sont retournés.

    Returns:
        Dict {match_id: ImpactEvent} pour le premier kill de chaque match.
    """
    if events_df.is_empty():
        return {}

    kills = events_df.filter(pl.col("event_type").str.to_lowercase() == "kill")
    if kills.is_empty():
        return {}

    first_kills = (
        kills.group_by("match_id")
        .agg(pl.col("time_ms").min().alias("min_time"))
        .join(kills, on="match_id")
        .filter(pl.col("time_ms") == pl.col("min_time"))
        .unique(subset=["match_id"])
    )

    result = {}
    for row in first_kills.iter_rows(named=True):
        match_id = str(row["match_id"])
        result[match_id] = ImpactEvent(
            match_id=match_id,
            xuid=str(row["xuid"]),
            gamertag=str(row.get("gamertag", "Unknown")),
            time_ms=int(row["time_ms"]),
            event_type="first_blood",
        )

    if friend_xuids:
        friend_xuids_str = {str(x) for x in friend_xuids}
        result = {mid: ev for mid, ev in result.items() if ev.xuid in friend_xuids_str}

    return result


def identify_clutch_finisher(
    events_df: pl.DataFrame,
    matches_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> dict[str, ImpactEvent]:
    """Identifie le joueur avec le dernier kill d'une victoire.

    Le Clutch Finisher est le kill avec le timestamp le plus haut (max time_ms)
    dans un match où l'outcome = 2 (victoire).

    Args:
        events_df: DataFrame Polars des événements (match_id, xuid, gamertag, event_type, time_ms).
        matches_df: DataFrame Polars des matchs avec (match_id, outcome).
        friend_xuids: Set d'XUIDs à filtrer (optionnel).

    Returns:
        Dict {match_id: ImpactEvent} pour le dernier kill de chaque victoire.
    """
    if events_df.is_empty() or matches_df.is_empty():
        return {}

    wins = matches_df.filter(pl.col("outcome") == OUTCOME_WIN).select("match_id")
    win_match_ids = {str(m) for m in wins["match_id"].to_list()}
    if not win_match_ids:
        return {}

    kills = events_df.filter(
        (pl.col("event_type").str.to_lowercase() == "kill")
        & (pl.col("match_id").cast(pl.Utf8).is_in(win_match_ids))
    )
    if kills.is_empty():
        return {}

    if friend_xuids:
        friend_xuids_str = {str(x) for x in friend_xuids}
        kills = kills.filter(pl.col("xuid").cast(pl.Utf8).is_in(friend_xuids_str))
    if kills.is_empty():
        return {}

    last_kills = (
        kills.group_by("match_id")
        .agg(pl.col("time_ms").max().alias("max_time"))
        .join(kills, on="match_id")
        .filter(pl.col("time_ms") == pl.col("max_time"))
        .unique(subset=["match_id"])
    )

    result = {}
    for row in last_kills.iter_rows(named=True):
        match_id = str(row["match_id"])
        result[match_id] = ImpactEvent(
            match_id=match_id,
            xuid=str(row["xuid"]),
            gamertag=str(row.get("gamertag", "Unknown")),
            time_ms=int(row["time_ms"]),
            event_type="clutch_finisher",
        )

    return result


def identify_last_casualty(
    events_df: pl.DataFrame,
    matches_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> dict[str, ImpactEvent]:
    """Identifie le joueur avec la dernière mort d'une défaite.

    Le Last Casualty (Boulet) est la mort avec le timestamp le plus haut
    dans un match où l'outcome = 3 (défaite).

    Args:
        events_df: DataFrame Polars des événements.
        matches_df: DataFrame Polars des matchs avec (match_id, outcome).
        friend_xuids: Set d'XUIDs à filtrer (optionnel).

    Returns:
        Dict {match_id: ImpactEvent} pour la dernière mort de chaque défaite.
    """
    if events_df.is_empty() or matches_df.is_empty():
        return {}

    losses = matches_df.filter(pl.col("outcome") == OUTCOME_LOSS).select("match_id")
    loss_match_ids = {str(m) for m in losses["match_id"].to_list()}
    if not loss_match_ids:
        return {}

    deaths = events_df.filter(
        (pl.col("event_type").str.to_lowercase() == "death")
        & (pl.col("match_id").cast(pl.Utf8).is_in(loss_match_ids))
    )
    if deaths.is_empty():
        return {}

    if friend_xuids:
        friend_xuids_str = {str(x) for x in friend_xuids}
        deaths = deaths.filter(pl.col("xuid").cast(pl.Utf8).is_in(friend_xuids_str))
    if deaths.is_empty():
        return {}

    last_deaths = (
        deaths.group_by("match_id")
        .agg(pl.col("time_ms").max().alias("max_time"))
        .join(deaths, on="match_id")
        .filter(pl.col("time_ms") == pl.col("max_time"))
        .unique(subset=["match_id"])
    )

    result = {}
    for row in last_deaths.iter_rows(named=True):
        match_id = str(row["match_id"])
        result[match_id] = ImpactEvent(
            match_id=match_id,
            xuid=str(row["xuid"]),
            gamertag=str(row.get("gamertag", "Unknown")),
            time_ms=int(row["time_ms"]),
            event_type="last_casualty",
        )

    return result


def identify_last_group_kill(
    events_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> dict[str, ImpactEvent]:
    """Identifie le joueur le plus lent à obtenir son premier kill dans chaque match.

    Pour chaque match, trouve le joueur de l'équipe dont le premier kill
    a le time_ms le plus élevé (le plus lent à "démarrer").
    Si friend_xuids est fourni, la recherche est restreinte à ces joueurs.

    Args:
        events_df: DataFrame des événements highlight (avec gamertag).
        friend_xuids: Set d'XUIDs des amis/équipe à filtrer (optionnel).

    Returns:
        Dict {match_id: ImpactEvent} du dernier à tuer dans l'équipe pour chaque match.
    """
    if events_df.is_empty():
        return {}

    kills = events_df.filter(pl.col("event_type") == "kill")
    if kills.is_empty():
        return {}

    if friend_xuids:
        friend_xuids_str = {str(x) for x in friend_xuids}
        kills = kills.filter(pl.col("xuid").cast(pl.Utf8).is_in(friend_xuids_str))
    if kills.is_empty():
        return {}

    first_kills_per_player = kills.group_by(["match_id", "xuid", "gamertag"]).agg(
        pl.col("time_ms").min().alias("first_kill_time")
    )
    slowest_kills = (
        first_kills_per_player.group_by("match_id")
        .agg(pl.col("first_kill_time").max().alias("max_time"))
        .join(first_kills_per_player, on="match_id")
        .filter(pl.col("first_kill_time") == pl.col("max_time"))
        .unique(subset=["match_id"])
    )

    result = {}
    for row in slowest_kills.iter_rows(named=True):
        match_id = str(row["match_id"])
        result[match_id] = ImpactEvent(
            match_id=match_id,
            xuid=str(row["xuid"]),
            gamertag=str(row.get("gamertag", "Unknown")),
            time_ms=int(row["first_kill_time"]),
            event_type="last_group_kill",
        )

    return result


def identify_first_group_death(
    events_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> dict[str, ImpactEvent]:
    """Identifie le premier joueur de l'équipe à mourir dans chaque match.

    Pour chaque match, trouve le joueur (parmi le groupe / l'équipe) avec la première mort
    (time_ms le plus bas). Si friend_xuids est fourni, seules les morts de ces joueurs
    sont considérées.

    Args:
        events_df: DataFrame des événements highlight (avec gamertag).
        friend_xuids: Set d'XUIDs des amis à filtrer (optionnel).

    Returns:
        Dict {match_id: ImpactEvent} de la première victime dans l'équipe pour chaque match.
    """
    if events_df.is_empty():
        return {}

    deaths = events_df.filter(pl.col("event_type") == "death")
    if deaths.is_empty():
        return {}

    if friend_xuids:
        friend_xuids_str = {str(x) for x in friend_xuids}
        deaths = deaths.filter(pl.col("xuid").cast(pl.Utf8).is_in(friend_xuids_str))
    if deaths.is_empty():
        return {}

    first_deaths = (
        deaths.group_by("match_id")
        .agg(pl.col("time_ms").min().alias("min_time"))
        .join(deaths, on="match_id")
        .filter(pl.col("time_ms") == pl.col("min_time"))
        .unique(subset=["match_id"])
    )

    result = {}
    for row in first_deaths.iter_rows(named=True):
        match_id = str(row["match_id"])
        result[match_id] = ImpactEvent(
            match_id=match_id,
            xuid=str(row["xuid"]),
            gamertag=str(row.get("gamertag", "Unknown")),
            time_ms=int(row["time_ms"]),
            event_type="first_group_death",
        )

    return result
