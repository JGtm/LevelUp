"""Modèle d'événement d'impact et logique d'extraction pour un match unique.

Identifie les événements remarquables (premier sang, finisseur, etc.)
à partir des highlight_events d'un match Halo Infinite.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from src.data.domain.refdata import Outcome
from src.ui.i18n.viz import viz_t

# Seuil de kills pour le badge Top Gun
TOP_GUN_KILL_THRESHOLD = 10


@dataclass(frozen=True)
class MatchImpactEvent:
    """Événement d'impact identifié dans un match unique."""

    event_type: (
        str  # "first_blood", "clutch_finisher", "last_group_kill", "first_group_death", "top_gun"
    )
    xuid: str
    gamertag: str
    time_ms: int
    is_me: bool  # True si c'est le joueur principal


def get_impact_labels(lang: str = "fr") -> dict[str, tuple[str, str]]:
    """Retourne le mapping événement → (icône, label traduit)."""
    return {
        "first_blood": ("⚡", viz_t("impact_first_blood", lang)),
        "clutch_finisher": ("🎯", viz_t("impact_clutch_finisher", lang)),
        "last_casualty": ("💀", viz_t("impact_last_casualty", lang)),
        "last_group_kill": ("🐌", viz_t("impact_last_group_kill", lang)),
        "first_group_death": ("🪦", viz_t("impact_first_group_death", lang)),
        "top_gun": ("🔫", viz_t("impact_top_gun", lang)),
    }


def _make_event(event_type: str, raw: dict[str, Any], me_xuid: str) -> MatchImpactEvent:
    """Construit un MatchImpactEvent depuis un dict brut."""
    xu = str(raw.get("xuid", "")).strip()
    return MatchImpactEvent(
        event_type=event_type,
        xuid=xu,
        gamertag=raw.get("gamertag") or "?",
        time_ms=int(raw["time_ms"]),
        is_me=(xu == me_xuid),
    )


def _find_slowest_killer_event(
    kills: list[dict[str, Any]],
    me_xuid: str,
) -> MatchImpactEvent | None:
    """Retourne l'événement Touriste (joueur le plus lent à obtenir son 1er kill), ou None."""
    first_kill_by_player: dict[str, dict[str, Any]] = {}
    for k in kills:
        xu = str(k.get("xuid", "")).strip()
        if not xu:
            continue
        if xu not in first_kill_by_player or int(k["time_ms"]) < int(
            first_kill_by_player[xu]["time_ms"]
        ):
            first_kill_by_player[xu] = k
    if len(first_kill_by_player) < 2:
        return None
    slowest = max(first_kill_by_player.values(), key=lambda e: int(e["time_ms"]))
    return _make_event("last_group_kill", slowest, me_xuid)


def _find_top_gun_event(
    kills: list[dict[str, Any]],
    me_xuid: str,
) -> MatchImpactEvent | None:
    """Retourne l'événement Top Gun (premier à atteindre TOP_GUN_KILL_THRESHOLD kills), ou None."""
    kills_sorted = sorted(kills, key=lambda e: int(e["time_ms"]))
    kill_count_by_player: dict[str, int] = {}
    threshold_kill_by_player: dict[str, dict[str, Any]] = {}
    for k in kills_sorted:
        xu = str(k.get("xuid", "")).strip()
        if not xu:
            continue
        kill_count_by_player[xu] = kill_count_by_player.get(xu, 0) + 1
        if kill_count_by_player[xu] == TOP_GUN_KILL_THRESHOLD:
            threshold_kill_by_player[xu] = k
    if not threshold_kill_by_player:
        return None
    top_gun_kill = min(threshold_kill_by_player.values(), key=lambda e: int(e["time_ms"]))
    return _make_event("top_gun", top_gun_kill, me_xuid)


def compute_single_match_impact(
    highlight_events: list[dict[str, Any]],
    me_xuid: str,
    outcome: int | None = None,
    team_xuids: set[str] | None = None,
) -> list[MatchImpactEvent]:
    """Identifie les événements d'impact pour un match unique.

    Args:
        highlight_events: Liste de dicts {event_type, time_ms, xuid, gamertag, ...}.
        me_xuid: XUID du joueur principal.
        outcome: Code outcome (2=win, 3=loss) pour le clutch_finisher.
        team_xuids: Si fourni, filtre les events pour ne garder que l'équipe alliée.

    Returns:
        Liste de MatchImpactEvent identifiés.
    """
    if not highlight_events:
        return []

    me_xuid = str(me_xuid).strip()

    if team_xuids:
        highlight_events = [
            e for e in highlight_events if str(e.get("xuid", "")).strip() in team_xuids
        ]

    kills = [
        e
        for e in highlight_events
        if str(e.get("event_type", "")).lower() == "kill" and e.get("time_ms") is not None
    ]
    deaths = [
        e
        for e in highlight_events
        if str(e.get("event_type", "")).lower() == "death" and e.get("time_ms") is not None
    ]

    if not kills and not deaths:
        return []

    events: list[MatchImpactEvent] = []

    if kills:
        events.append(
            _make_event("first_blood", min(kills, key=lambda e: int(e["time_ms"])), me_xuid)
        )
    if kills and outcome == Outcome.WIN:
        events.append(
            _make_event("clutch_finisher", max(kills, key=lambda e: int(e["time_ms"])), me_xuid)
        )
    if deaths and outcome == Outcome.LOSS:
        events.append(
            _make_event("last_casualty", max(deaths, key=lambda e: int(e["time_ms"])), me_xuid)
        )
    if kills:
        slowest = _find_slowest_killer_event(kills, me_xuid)
        if slowest is not None:
            events.append(slowest)
    if deaths:
        events.append(
            _make_event("first_group_death", min(deaths, key=lambda e: int(e["time_ms"])), me_xuid)
        )
    if kills:
        top_gun = _find_top_gun_event(kills, me_xuid)
        if top_gun is not None:
            events.append(top_gun)

    return events
