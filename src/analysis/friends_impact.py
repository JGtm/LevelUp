"""Analyse d'impact des coéquipiers : First Blood, Clutch Finisher, Last Casualty."""

from __future__ import annotations

from dataclasses import dataclass

import polars as pl

# Constantes de scoring
SCORE_CLUTCH_FINISHER = 2  # Finisseur : +2 points
SCORE_FIRST_BLOOD = 1  # Premier sang : +1 point
SCORE_LAST_CASUALTY = -1  # Boulet : -1 point
SCORE_SILENT_HERO = 1  # Héros silencieux : +1 point
SCORE_FALSE_BROTHER = -1  # Faux-frère : -1 point

# Codes d'outcome
OUTCOME_WIN = 2
OUTCOME_LOSS = 3


@dataclass
class ImpactEvent:
    """Représente un événement d'impact dans un match."""

    match_id: str
    xuid: str
    gamertag: str
    time_ms: int
    event_type: str  # "first_blood", "clutch_finisher", "last_casualty"


def identify_first_blood(
    events_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> dict[str, ImpactEvent]:
    """Identifie le joueur avec le premier kill par match.

    Le First Blood est le kill avec le timestamp le plus bas (min time_ms)
    dans chaque match, indépendamment de l'outcome.

    Args:
        events_df: DataFrame Polars avec colonnes :
            - match_id, xuid, gamertag, event_type, time_ms
        friend_xuids: Set d'XUIDs à filtrer (optionnel). Si None, tous les joueurs.

    Returns:
        Dict {match_id: ImpactEvent} pour le premier kill de chaque match.
    """
    if events_df.is_empty():
        return {}

    # Filtrer les kills uniquement (event_type peut être "kill" ou "Kill")
    kills = events_df.filter(pl.col("event_type").str.to_lowercase() == "kill")

    if kills.is_empty():
        return {}

    # Filtrer par amis si spécifié
    if friend_xuids:
        # Normaliser en string pour comparaison
        friend_xuids_str = {str(x) for x in friend_xuids}
        kills = kills.filter(pl.col("xuid").cast(pl.Utf8).is_in(friend_xuids_str))

    if kills.is_empty():
        return {}

    # Trouver le premier kill par match (min time_ms)
    first_kills = (
        kills.group_by("match_id")
        .agg(pl.col("time_ms").min().alias("min_time"))
        .join(kills, on="match_id")
        .filter(pl.col("time_ms") == pl.col("min_time"))
        .unique(subset=["match_id"])  # Un seul FB par match
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

    # Filtrer les victoires
    wins = matches_df.filter(pl.col("outcome") == OUTCOME_WIN).select("match_id")
    win_match_ids = {str(m) for m in wins["match_id"].to_list()}

    if not win_match_ids:
        return {}

    # Filtrer les kills dans les matchs gagnés
    kills = events_df.filter(
        (pl.col("event_type").str.to_lowercase() == "kill")
        & (pl.col("match_id").cast(pl.Utf8).is_in(win_match_ids))
    )

    if kills.is_empty():
        return {}

    # Filtrer par amis si spécifié
    if friend_xuids:
        friend_xuids_str = {str(x) for x in friend_xuids}
        kills = kills.filter(pl.col("xuid").cast(pl.Utf8).is_in(friend_xuids_str))

    if kills.is_empty():
        return {}

    # Trouver le dernier kill par match (max time_ms)
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

    # Filtrer les défaites
    losses = matches_df.filter(pl.col("outcome") == OUTCOME_LOSS).select("match_id")
    loss_match_ids = {str(m) for m in losses["match_id"].to_list()}

    if not loss_match_ids:
        return {}

    # Filtrer les morts dans les matchs perdus
    deaths = events_df.filter(
        (pl.col("event_type").str.to_lowercase() == "death")
        & (pl.col("match_id").cast(pl.Utf8).is_in(loss_match_ids))
    )

    if deaths.is_empty():
        return {}

    # Filtrer par amis si spécifié
    if friend_xuids:
        friend_xuids_str = {str(x) for x in friend_xuids}
        deaths = deaths.filter(pl.col("xuid").cast(pl.Utf8).is_in(friend_xuids_str))

    if deaths.is_empty():
        return {}

    # Trouver la dernière mort par match (max time_ms)
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
    events_df: pl.DataFrame, friend_xuids: set[str] | None = None
) -> dict[str, ImpactEvent]:
    """Identifie le joueur le plus lent à obtenir son premier kill dans chaque match.

    Pour chaque match, trouve le joueur (parmi le groupe) dont le premier kill
    a le time_ms le plus élevé (le plus lent à "démarrer").

    Args:
        events_df: DataFrame des événements highlight (avec gamertag).
        friend_xuids: Set d'XUIDs des amis à filtrer (optionnel).

    Returns:
        Dict {match_id: ImpactEvent} du dernier à tuer dans chaque match.
    """
    if events_df.is_empty():
        return {}

    # Filtrer les kills
    kills = events_df.filter(pl.col("event_type") == "kill")

    if kills.is_empty():
        return {}

    # Filtrer par amis si spécifié
    if friend_xuids:
        friend_xuids_str = {str(x) for x in friend_xuids}
        kills = kills.filter(pl.col("xuid").cast(pl.Utf8).is_in(friend_xuids_str))

    if kills.is_empty():
        return {}

    # Pour chaque joueur dans chaque match, trouver son premier kill
    first_kills_per_player = kills.group_by(["match_id", "xuid", "gamertag"]).agg(
        pl.col("time_ms").min().alias("first_kill_time")
    )

    # Pour chaque match, trouver le joueur avec le temps le plus élevé
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
    events_df: pl.DataFrame, friend_xuids: set[str] | None = None
) -> dict[str, ImpactEvent]:
    """Identifie le premier joueur à mourir dans chaque match.

    Pour chaque match, trouve le joueur (parmi le groupe) avec la première mort
    (time_ms le plus bas).

    Args:
        events_df: DataFrame des événements highlight (avec gamertag).
        friend_xuids: Set d'XUIDs des amis à filtrer (optionnel).

    Returns:
        Dict {match_id: ImpactEvent} de la première victime dans chaque match.
    """
    if events_df.is_empty():
        return {}

    # Filtrer les morts
    deaths = events_df.filter(pl.col("event_type") == "death")

    if deaths.is_empty():
        return {}

    # Filtrer par amis si spécifié
    if friend_xuids:
        friend_xuids_str = {str(x) for x in friend_xuids}
        deaths = deaths.filter(pl.col("xuid").cast(pl.Utf8).is_in(friend_xuids_str))

    if deaths.is_empty():
        return {}

    # Trouver la première mort par match (min time_ms)
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


def identify_silent_hero_multi(
    participants_df: pl.DataFrame,
    matches_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> dict[str, ImpactEvent]:
    """Par match en victoire : joueur avec simultanément max assists ET min deaths, ≥1 assist, ≥2 joueurs."""
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
        max_assists = max(r["assists"] for r in rows)
        if max_assists == 0:
            continue
        min_deaths = min(r["deaths"] for r in rows)
        candidates = [r for r in rows if r["assists"] == max_assists and r["deaths"] == min_deaths]
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
    return result


def identify_false_brother_multi(
    participants_df: pl.DataFrame,
    matches_df: pl.DataFrame,
    friend_xuids: set[str] | None = None,
) -> dict[str, ImpactEvent]:
    """Par match en défaite : joueur avec simultanément max deaths ET min assists, ≥1 mort, ≥2 joueurs."""
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
        max_deaths = max(r["deaths"] for r in rows)
        if max_deaths == 0:
            continue
        min_assists = min(r["assists"] for r in rows)
        candidates = [r for r in rows if r["deaths"] == max_deaths and r["assists"] == min_assists]
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
    return result


def compute_impact_scores(
    first_bloods: dict[str, ImpactEvent],
    clutch_finishers: dict[str, ImpactEvent],
    last_casualties: dict[str, ImpactEvent],
) -> dict[str, int]:
    """Calcule les scores d'impact par joueur.

    Scoring :
    - Clutch Finisher : +2 points
    - First Blood : +1 point
    - Last Casualty : -1 point

    Args:
        first_bloods: Dict {match_id: ImpactEvent} des premiers kills.
        clutch_finishers: Dict {match_id: ImpactEvent} des derniers kills victorieux.
        last_casualties: Dict {match_id: ImpactEvent} des dernières morts en défaite.

    Returns:
        Dict {gamertag: score} trié par score décroissant.
    """
    scores: dict[str, int] = {}

    # +1 pour First Blood
    for event in first_bloods.values():
        gamertag = event.gamertag
        scores[gamertag] = scores.get(gamertag, 0) + SCORE_FIRST_BLOOD

    # +2 pour Clutch Finisher
    for event in clutch_finishers.values():
        gamertag = event.gamertag
        scores[gamertag] = scores.get(gamertag, 0) + SCORE_CLUTCH_FINISHER

    # -1 pour Last Casualty
    for event in last_casualties.values():
        gamertag = event.gamertag
        scores[gamertag] = scores.get(gamertag, 0) + SCORE_LAST_CASUALTY

    # Trier par score décroissant
    return dict(sorted(scores.items(), key=lambda x: x[1], reverse=True))


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
    dict[str, int],
]:
    """Récupère tous les événements d'impact et calcule les scores.

    Fonction de convenance qui appelle les 5 fonctions d'identification
    puis calcule les scores.

    Args:
        events_df: DataFrame des événements highlight.
        matches_df: DataFrame des matchs avec outcome.
        friend_xuids: Set d'XUIDs des amis à filtrer.

    Returns:
        Tuple (first_bloods, clutch_finishers, last_casualties,
               last_group_kills, first_group_deaths, scores).
    """
    first_bloods = identify_first_blood(events_df, friend_xuids)
    clutch_finishers = identify_clutch_finisher(events_df, matches_df, friend_xuids)
    last_casualties = identify_last_casualty(events_df, matches_df, friend_xuids)
    last_group_kills = identify_last_group_kill(events_df, friend_xuids)
    first_group_deaths = identify_first_group_death(events_df, friend_xuids)
    scores = compute_impact_scores(first_bloods, clutch_finishers, last_casualties)

    return (
        first_bloods,
        clutch_finishers,
        last_casualties,
        last_group_kills,
        first_group_deaths,
        scores,
    )


_EVENT_DEFS: list[tuple[str, int]] = [
    ("first_blood", 1),
    ("clutch_finisher", 2),
    ("last_casualty", -1),
    ("last_group_kill", 3),
    ("first_group_death", -2),
    ("silent_hero", 1),
    ("false_brother", -1),
]

_EMPTY_IMPACT_SCHEMA = {
    "match_id": pl.Utf8,
    "gamertag": pl.Utf8,
    "events": pl.List(pl.Struct([pl.Field("event", pl.Utf8), pl.Field("value", pl.Int64)])),
    "outcome": pl.Int64,
}


def _populate_events_map(
    match_ids: list[str],
    gamertags: list[str],
    event_dicts: list[dict[str, ImpactEvent]],
) -> dict[tuple[str, str], list[dict[str, str | int]]]:
    """Initialise et remplit le mapping (match_id, gamertag) → events."""
    events_map: dict[tuple[str, str], list[dict[str, str | int]]] = {
        (mid, gt): [] for mid in match_ids for gt in gamertags
    }
    for src, (event_name, value) in zip(event_dicts, _EVENT_DEFS, strict=True):
        for _match_id, event in src.items():
            key = (_match_id, event.gamertag)
            if key in events_map:
                events_map[key].append({"event": event_name, "value": value})
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
                {"match_id": match_id, "gamertag": "Résultat", "events": [], "outcome": outcome}
            )
    for (match_id, gamertag), events in events_map.items():
        rows.append({"match_id": match_id, "gamertag": gamertag, "events": events, "outcome": None})
    return rows


def build_impact_matrix(  # noqa: PLR0913
    first_bloods: dict[str, ImpactEvent],
    clutch_finishers: dict[str, ImpactEvent],
    last_casualties: dict[str, ImpactEvent],
    last_group_kills: dict[str, ImpactEvent],
    first_group_deaths: dict[str, ImpactEvent],
    match_ids: list[str],
    gamertags: list[str],
    match_outcomes: dict[str, int] | None = None,
    silent_heroes: dict[str, ImpactEvent] | None = None,
    false_brothers: dict[str, ImpactEvent] | None = None,
) -> pl.DataFrame:
    """Construit une matrice d'impact pour la heatmap."""
    event_dicts = [
        first_bloods,
        clutch_finishers,
        last_casualties,
        last_group_kills,
        first_group_deaths,
        silent_heroes if silent_heroes is not None else {},
        false_brothers if false_brothers is not None else {},
    ]
    events_map = _populate_events_map(match_ids, gamertags, event_dicts)
    rows = _build_impact_rows(events_map, match_ids, match_outcomes)
    if not rows:
        return pl.DataFrame([], schema=_EMPTY_IMPACT_SCHEMA)
    return pl.DataFrame(rows)
