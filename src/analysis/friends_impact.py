"""Analyse d'impact des coéquipiers : orchestration + badges participants + matrice."""

from __future__ import annotations

import logging
from dataclasses import dataclass, field

import polars as pl

from src.analysis._impact_event_badges import (
    identify_clutch_finisher,
    identify_first_blood,
    identify_first_group_death,
    identify_last_casualty,
    identify_last_group_kill,
)
from src.analysis._impact_types import (
    OUTCOME_LOSS,
    OUTCOME_WIN,
    SCORE_CLUTCH_FINISHER,
    SCORE_FALSE_BROTHER,
    SCORE_FIRST_BLOOD,
    SCORE_FIRST_GROUP_DEATH,
    SCORE_LAST_CASUALTY,
    SCORE_LAST_GROUP_KILL,
    SCORE_SILENT_HERO,
    SCORE_TOP_KILLER,
    ImpactEvent,
)

logger = logging.getLogger(__name__)

# Re-exports pour la compatibilité des importeurs existants
__all__ = [
    "ImpactEvent",
    "ImpactEventSets",
    "SCORE_CLUTCH_FINISHER",
    "SCORE_FIRST_BLOOD",
    "SCORE_LAST_CASUALTY",
    "SCORE_SILENT_HERO",
    "SCORE_FALSE_BROTHER",
    "SCORE_LAST_GROUP_KILL",
    "SCORE_FIRST_GROUP_DEATH",
    "SCORE_TOP_KILLER",
    "OUTCOME_WIN",
    "OUTCOME_LOSS",
    "build_impact_matrix",
    "compute_impact_scores",
    "get_all_impact_events",
    "identify_clutch_finisher",
    "identify_false_brother_multi",
    "identify_first_blood",
    "identify_first_group_death",
    "identify_last_casualty",
    "identify_last_group_kill",
    "identify_silent_hero_multi",
    "identify_top_killer_multi",
]


@dataclass
class ImpactEventSets:
    """Regroupe tous les dicts d'événements d'impact pour un ensemble de matchs."""

    first_bloods: dict[str, ImpactEvent] = field(default_factory=dict)
    clutch_finishers: dict[str, ImpactEvent] = field(default_factory=dict)
    last_casualties: dict[str, ImpactEvent] = field(default_factory=dict)
    last_group_kills: dict[str, ImpactEvent] = field(default_factory=dict)
    first_group_deaths: dict[str, ImpactEvent] = field(default_factory=dict)
    silent_heroes: dict[str, ImpactEvent] = field(default_factory=dict)
    false_brothers: dict[str, ImpactEvent] = field(default_factory=dict)
    top_killers: dict[str, ImpactEvent] = field(default_factory=dict)


def identify_silent_hero_multi(
    participants_df: pl.DataFrame,
    matches_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> dict[str, ImpactEvent]:
    """Par match en victoire : joueur avec simultanément max assists ET min deaths.

    Nécessite >=1 assist et >=2 joueurs. Le top killer de l'équipe est exclu.
    Si moins de 2 joueurs restent après exclusion, le badge n'est pas attribué.
    """
    if participants_df.is_empty():
        return {}
    win_ids = set(
        matches_df.filter(pl.col("outcome") == OUTCOME_WIN)["match_id"].cast(pl.Utf8).to_list()
    )
    if not win_ids:
        return {}
    df = participants_df.with_columns(pl.col("xuid").cast(pl.Utf8))
    df = df.filter(pl.col("match_id").cast(pl.Utf8).is_in(win_ids))
    if friend_xuids:
        df = df.filter(pl.col("xuid").is_in({str(x) for x in friend_xuids}))
    if df.is_empty():
        return {}
    result: dict[str, ImpactEvent] = {}
    for match_id in df["match_id"].unique().to_list():
        rows = df.filter(pl.col("match_id") == match_id).to_dicts()
        if len(rows) < 2:
            continue
        max_kills = max(r.get("kills", 0) for r in rows)
        if max_kills > 0:
            eligible = [r for r in rows if r.get("kills", 0) < max_kills]
            if len(eligible) < 2:
                continue
        else:
            eligible = rows
        max_assists = max(r["assists"] for r in eligible)
        if max_assists == 0:
            continue
        min_deaths = min(r["deaths"] for r in eligible)
        candidates = [r for r in eligible if r["assists"] == max_assists and r["deaths"] == min_deaths]
        if not candidates:
            continue
        hero = candidates[0]
        mid = str(match_id)
        result[mid] = ImpactEvent(
            match_id=mid,
            xuid=str(hero["xuid"]),
            gamertag=str(hero.get("gamertag", "Unknown")),
            time_ms=0,
            event_type="silent_hero",
        )
    logger.debug("identify_silent_hero_multi : %d badge(s) attribue(s)", len(result))
    return result


def identify_false_brother_multi(
    participants_df: pl.DataFrame,
    matches_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> dict[str, ImpactEvent]:
    """Par match en défaite : joueur avec simultanément max deaths ET min assists.

    Nécessite >=1 mort et >=2 joueurs. Le top killer de l'équipe est exclu.
    Si moins de 2 joueurs restent après exclusion, le badge n'est pas attribué.
    """
    if participants_df.is_empty():
        return {}
    loss_ids = set(
        matches_df.filter(pl.col("outcome") == OUTCOME_LOSS)["match_id"].cast(pl.Utf8).to_list()
    )
    if not loss_ids:
        return {}
    df = participants_df.with_columns(pl.col("xuid").cast(pl.Utf8))
    df = df.filter(pl.col("match_id").cast(pl.Utf8).is_in(loss_ids))
    if friend_xuids:
        df = df.filter(pl.col("xuid").is_in({str(x) for x in friend_xuids}))
    if df.is_empty():
        return {}
    result: dict[str, ImpactEvent] = {}
    for match_id in df["match_id"].unique().to_list():
        rows = df.filter(pl.col("match_id") == match_id).to_dicts()
        if len(rows) < 2:
            continue
        max_kills = max(r.get("kills", 0) for r in rows)
        if max_kills > 0:
            eligible = [r for r in rows if r.get("kills", 0) < max_kills]
            if len(eligible) < 2:
                continue
        else:
            eligible = rows
        max_deaths = max(r["deaths"] for r in eligible)
        if max_deaths == 0:
            continue
        min_assists = min(r["assists"] for r in eligible)
        candidates = [r for r in eligible if r["deaths"] == max_deaths and r["assists"] == min_assists]
        if not candidates:
            continue
        traitor = candidates[0]
        mid = str(match_id)
        result[mid] = ImpactEvent(
            match_id=mid,
            xuid=str(traitor["xuid"]),
            gamertag=str(traitor.get("gamertag", "Unknown")),
            time_ms=0,
            event_type="false_brother",
        )
    logger.debug("identify_false_brother_multi : %d badge(s) attribue(s)", len(result))
    return result


def identify_top_killer_multi(
    participants_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> dict[str, ImpactEvent]:
    """Par match : joueur avec le plus de kills dans l'équipe, quelle que soit l'issue.

    Nécessite la colonne `kills`. Minimum 1 kill et 2 joueurs pour attribuer le badge.
    """
    if participants_df.is_empty() or "kills" not in participants_df.columns:
        return {}
    df = participants_df.with_columns(pl.col("xuid").cast(pl.Utf8))
    if friend_xuids:
        df = df.filter(pl.col("xuid").is_in({str(x) for x in friend_xuids}))
    if df.is_empty():
        return {}
    result: dict[str, ImpactEvent] = {}
    for match_id in df["match_id"].unique().to_list():
        rows = df.filter(pl.col("match_id") == match_id).to_dicts()
        if len(rows) < 2:
            continue
        max_kills = max(r.get("kills", 0) for r in rows)
        if max_kills == 0:
            continue
        candidates = [r for r in rows if r.get("kills", 0) == max_kills]
        if not candidates:
            continue
        top = candidates[0]
        mid = str(match_id)
        result[mid] = ImpactEvent(
            match_id=mid,
            xuid=str(top["xuid"]),
            gamertag=str(top.get("gamertag", "Unknown")),
            time_ms=0,
            event_type="top_killer",
        )
    logger.debug("identify_top_killer_multi : %d badge(s) attribue(s)", len(result))
    return result


def compute_impact_scores(events: ImpactEventSets) -> dict[str, float]:
    """Calcule les scores d'impact par joueur à partir d'un ImpactEventSets.

    Returns:
        Dict {gamertag: score} trié par score décroissant.
    """
    scores: dict[str, float] = {}
    _apply_score(scores, events.first_bloods, SCORE_FIRST_BLOOD)
    _apply_score(scores, events.clutch_finishers, SCORE_CLUTCH_FINISHER)
    _apply_score(scores, events.last_casualties, SCORE_LAST_CASUALTY)
    _apply_score(scores, events.last_group_kills, SCORE_LAST_GROUP_KILL)
    _apply_score(scores, events.first_group_deaths, SCORE_FIRST_GROUP_DEATH)
    _apply_score(scores, events.silent_heroes, SCORE_SILENT_HERO)
    _apply_score(scores, events.false_brothers, SCORE_FALSE_BROTHER)
    _apply_score(scores, events.top_killers, SCORE_TOP_KILLER)
    return dict(sorted(scores.items(), key=lambda x: x[1], reverse=True))


def _apply_score(
    scores: dict[str, float],
    event_dict: dict[str, ImpactEvent],
    value: float,
) -> None:
    """Ajoute `value` au score du gamertag de chaque événement."""
    for event in event_dict.values():
        scores[event.gamertag] = scores.get(event.gamertag, 0.0) + value


def get_all_impact_events(
    events_df: pl.DataFrame,
    matches_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> tuple[
    dict[str, ImpactEvent],
    dict[str, ImpactEvent],
    dict[str, ImpactEvent],
    dict[str, ImpactEvent],
    dict[str, ImpactEvent],
    dict[str, float],
]:
    """Récupère les 5 événements highlight et calcule les scores.

    Returns:
        Tuple (first_bloods, clutch_finishers, last_casualties,
               last_group_kills, first_group_deaths, scores).
    """
    ev = ImpactEventSets(
        first_bloods=identify_first_blood(events_df, friend_xuids),
        clutch_finishers=identify_clutch_finisher(events_df, matches_df, friend_xuids),
        last_casualties=identify_last_casualty(events_df, matches_df, friend_xuids),
        last_group_kills=identify_last_group_kill(events_df, friend_xuids),
        first_group_deaths=identify_first_group_death(events_df, friend_xuids),
    )
    scores = compute_impact_scores(ev)
    return (
        ev.first_bloods,
        ev.clutch_finishers,
        ev.last_casualties,
        ev.last_group_kills,
        ev.first_group_deaths,
        scores,
    )


_EVENT_DEFS: list[tuple[str, int | float]] = [
    ("first_blood", SCORE_FIRST_BLOOD),
    ("clutch_finisher", SCORE_CLUTCH_FINISHER),
    ("last_casualty", SCORE_LAST_CASUALTY),
    ("last_group_kill", SCORE_LAST_GROUP_KILL),
    ("first_group_death", SCORE_FIRST_GROUP_DEATH),
    ("silent_hero", SCORE_SILENT_HERO),
    ("false_brother", SCORE_FALSE_BROTHER),
    ("top_killer", SCORE_TOP_KILLER),
]

_EMPTY_IMPACT_SCHEMA = {
    "match_id": pl.Utf8,
    "gamertag": pl.Utf8,
    "events": pl.List(pl.Struct([pl.Field("event", pl.Utf8), pl.Field("value", pl.Float64)])),
    "outcome": pl.Int64,
}


def _populate_events_map(
    match_ids: list[str],
    gamertags: list[str],
    event_dicts: list[dict[str, ImpactEvent]],
) -> dict[tuple[str, str], list[dict[str, str | int | float]]]:
    """Initialise et remplit le mapping (match_id, gamertag) -> events."""
    events_map: dict[tuple[str, str], list[dict[str, str | int | float]]] = {
        (mid, gt): [] for mid in match_ids for gt in gamertags
    }
    for src, (event_name, value) in zip(event_dicts, _EVENT_DEFS, strict=True):
        for _match_id, event in src.items():
            key = (_match_id, event.gamertag)
            if key in events_map:
                events_map[key].append({"event": event_name, "value": float(value)})
    return events_map


def _build_impact_rows(
    events_map: dict[tuple[str, str], list[dict[str, str | int]]],
    match_ids: list[str],
    match_outcomes: dict[str, int] | None,
) -> list[dict]:
    """Construit les lignes du DataFrame d'impact."""
    rows: list[dict] = []
    if match_outcomes:
        for match_id in match_ids:
            outcome = match_outcomes.get(match_id, 0)
            rows.append(
                {"match_id": match_id, "gamertag": "Resultat", "events": [], "outcome": outcome}
            )
    for (match_id, gamertag), events in events_map.items():
        rows.append({"match_id": match_id, "gamertag": gamertag, "events": events, "outcome": None})
    return rows


def build_impact_matrix(
    events: ImpactEventSets,
    match_ids: list[str],
    gamertags: list[str],
    match_outcomes: dict[str, int] | None = None,
) -> pl.DataFrame:
    """Construit une matrice d'impact pour la heatmap."""
    event_dicts = [
        events.first_bloods,
        events.clutch_finishers,
        events.last_casualties,
        events.last_group_kills,
        events.first_group_deaths,
        events.silent_heroes,
        events.false_brothers,
        events.top_killers,
    ]
    events_map = _populate_events_map(match_ids, gamertags, event_dicts)
    rows = _build_impact_rows(events_map, match_ids, match_outcomes)
    if not rows:
        return pl.DataFrame([], schema=_EMPTY_IMPACT_SCHEMA)
    return pl.DataFrame(rows)
