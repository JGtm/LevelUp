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

# Sentinel time_ms pour les événements basés sur des stats (pas de timestamp)
_STATS_SENTINEL: int = -1


@dataclass(frozen=True)
class MatchImpactEvent:
    """Événement d'impact identifié dans un match unique."""

    event_type: (
        str  # "first_blood", "clutch_finisher", "last_group_kill", "first_group_death", "top_gun"
        # "silent_hero", "false_brother"
    )
    xuid: str
    gamertag: str
    time_ms: int  # _STATS_SENTINEL (-1) si pas de timestamp (badge basé sur stats)
    is_me: bool  # True si c'est le joueur principal
    extra_label: str = ""  # Légende stats à afficher à la place du temps (si rempli)


def get_impact_labels(lang: str = "fr") -> dict[str, tuple[str, str]]:
    """Retourne le mapping événement → (icône, label traduit)."""
    return {
        "first_blood": ("⚡", viz_t("impact_first_blood", lang)),
        "clutch_finisher": ("🎯", viz_t("impact_clutch_finisher", lang)),
        "last_casualty": ("💀", viz_t("impact_last_casualty", lang)),
        "last_group_kill": ("🐌", viz_t("impact_last_group_kill", lang)),
        "first_group_death": ("🪦", viz_t("impact_first_group_death", lang)),
        "top_gun": ("🔫", viz_t("impact_top_gun", lang)),
        "silent_hero": ("🛡️", viz_t("impact_silent_hero", lang)),
        "false_brother": ("🤡", viz_t("impact_false_brother", lang)),
    }


def _find_silent_hero_event(
    participants: list[dict[str, Any]],
    team_xuids: set[str],
    me_xuid: str,
    lang: str = "fr",
) -> MatchImpactEvent | None:
    """Victoire : joueur de l'équipe avec le meilleur ratio assists−deaths (≥1 assist)."""
    team = [p for p in participants if str(p.get("xuid", "")).strip() in team_xuids]
    eligible = [p for p in team if int(p.get("assists", 0)) >= 1]
    if len(team) < 2 or not eligible:
        return None
    best = max(eligible, key=lambda p: int(p.get("assists", 0)) - int(p.get("deaths", 0)))
    xu = str(best.get("xuid", "")).strip()
    assists = int(best.get("assists", 0))
    deaths = int(best.get("deaths", 0))
    if lang == "en":
        extra = f"{assists} ast · {deaths} death" + ("s" if deaths != 1 else "")
    else:
        extra = f"{assists} pass. · {deaths} mort" + ("s" if deaths != 1 else "")
    return MatchImpactEvent(
        event_type="silent_hero",
        xuid=xu,
        gamertag=best.get("gamertag") or "?",
        time_ms=_STATS_SENTINEL,
        is_me=(xu == me_xuid),
        extra_label=extra,
    )


def _find_false_brother_event(
    participants: list[dict[str, Any]],
    team_xuids: set[str],
    me_xuid: str,
    lang: str = "fr",
) -> MatchImpactEvent | None:
    """Défaite : joueur de l'équipe avec le pire ratio deaths−assists (≥1 death)."""
    team = [p for p in participants if str(p.get("xuid", "")).strip() in team_xuids]
    eligible = [p for p in team if int(p.get("deaths", 0)) >= 1]
    if len(team) < 2 or not eligible:
        return None
    worst = max(eligible, key=lambda p: int(p.get("deaths", 0)) - int(p.get("assists", 0)))
    xu = str(worst.get("xuid", "")).strip()
    deaths = int(worst.get("deaths", 0))
    assists = int(worst.get("assists", 0))
    if lang == "en":
        extra = f"{deaths} death" + ("s" if deaths != 1 else "") + f" · {assists} ast"
    else:
        extra = f"{deaths} mort" + ("s" if deaths != 1 else "") + f" · {assists} pass."
    return MatchImpactEvent(
        event_type="false_brother",
        xuid=xu,
        gamertag=worst.get("gamertag") or "?",
        time_ms=_STATS_SENTINEL,
        is_me=(xu == me_xuid),
        extra_label=extra,
    )


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
    xu = str(top_gun_kill.get("xuid", "")).strip()
    return MatchImpactEvent(
        event_type="top_gun",
        xuid=xu,
        gamertag=top_gun_kill.get("gamertag") or "?",
        time_ms=int(top_gun_kill["time_ms"]),
        is_me=(xu == me_xuid),
    )


def compute_single_match_impact(  # noqa: C901, PLR0912, PLR0913 — complexité inhérente au scoring
    highlight_events: list[dict[str, Any]],
    me_xuid: str,
    outcome: int | None = None,
    team_xuids: set[str] | None = None,
    participants_stats: list[dict[str, Any]] | None = None,
    lang: str = "fr",
) -> list[MatchImpactEvent]:
    """Identifie les événements d'impact pour un match unique.

    Args:
        highlight_events: Liste de dicts {event_type, time_ms, xuid, gamertag, ...}.
        me_xuid: XUID du joueur principal.
        outcome: Code outcome (2=win, 3=loss) pour le clutch_finisher / badges stats.
        team_xuids: Si fourni, filtre les events pour ne garder que l'équipe alliée.
        participants_stats: Stats de tous les joueurs du match (kills, deaths, assists, team_id).
            Nécessaire pour les badges «Héros silencieux» et «Faux-frère».
        lang: Code langue pour les labels stats ("fr" ou "en").

    Returns:
        Liste de MatchImpactEvent identifiés.
    """
    if not highlight_events:
        return []

    me_xuid = str(me_xuid).strip()
    events: list[MatchImpactEvent] = []

    # Filtrer par équipe si team_xuids est fourni
    if team_xuids:
        highlight_events = [
            e for e in highlight_events if str(e.get("xuid", "")).strip() in team_xuids
        ]

    # Séparer kills et deaths
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

    # --- Premier sang : premier kill du match (tous joueurs) ---
    if kills:
        first_kill = min(kills, key=lambda e: int(e["time_ms"]))
        xu = str(first_kill.get("xuid", "")).strip()
        events.append(
            MatchImpactEvent(
                event_type="first_blood",
                xuid=xu,
                gamertag=first_kill.get("gamertag") or "?",
                time_ms=int(first_kill["time_ms"]),
                is_me=(xu == me_xuid),
            )
        )

    # --- Finisseur : dernier kill d'une victoire ---
    if kills and outcome == Outcome.WIN:
        last_kill = max(kills, key=lambda e: int(e["time_ms"]))
        xu = str(last_kill.get("xuid", "")).strip()
        events.append(
            MatchImpactEvent(
                event_type="clutch_finisher",
                xuid=xu,
                gamertag=last_kill.get("gamertag") or "?",
                time_ms=int(last_kill["time_ms"]),
                is_me=(xu == me_xuid),
            )
        )

    # --- Boulet : dernière mort d'une défaite ---
    if deaths and outcome == Outcome.LOSS:
        last_death = max(deaths, key=lambda e: int(e["time_ms"]))
        xu = str(last_death.get("xuid", "")).strip()
        events.append(
            MatchImpactEvent(
                event_type="last_casualty",
                xuid=xu,
                gamertag=last_death.get("gamertag") or "?",
                time_ms=int(last_death["time_ms"]),
                is_me=(xu == me_xuid),
            )
        )

    # --- Touriste : joueur le plus lent à obtenir son 1er kill ---
    if kills:
        # Grouper par xuid → premier kill de chaque joueur
        first_kill_by_player: dict[str, dict[str, Any]] = {}
        for k in kills:
            xu = str(k.get("xuid", "")).strip()
            if not xu:
                continue
            if xu not in first_kill_by_player or int(k["time_ms"]) < int(
                first_kill_by_player[xu]["time_ms"]
            ):
                first_kill_by_player[xu] = k

        if len(first_kill_by_player) >= 2:
            # Le Touriste = celui avec le max de time_ms pour son premier kill
            slowest = max(first_kill_by_player.values(), key=lambda e: int(e["time_ms"]))
            xu = str(slowest.get("xuid", "")).strip()
            events.append(
                MatchImpactEvent(
                    event_type="last_group_kill",
                    xuid=xu,
                    gamertag=slowest.get("gamertag") or "?",
                    time_ms=int(slowest["time_ms"]),
                    is_me=(xu == me_xuid),
                )
            )

    # --- Première victime : première mort du match (tous joueurs) ---
    if deaths:
        first_death = min(deaths, key=lambda e: int(e["time_ms"]))
        xu = str(first_death.get("xuid", "")).strip()
        events.append(
            MatchImpactEvent(
                event_type="first_group_death",
                xuid=xu,
                gamertag=first_death.get("gamertag") or "?",
                time_ms=int(first_death["time_ms"]),
                is_me=(xu == me_xuid),
            )
        )

    # --- Top Gun : premier joueur de l'équipe à atteindre le seuil de kills ---
    if kills:
        top_gun = _find_top_gun_event(kills, me_xuid)
        if top_gun is not None:
            events.append(top_gun)

    # --- Héros silencieux (victoire) / Faux-frère (défaite) ---
    if participants_stats and team_xuids:
        if outcome == Outcome.WIN:
            hero = _find_silent_hero_event(participants_stats, team_xuids, me_xuid, lang)
            if hero is not None:
                events.append(hero)
        elif outcome == Outcome.LOSS:
            traitor = _find_false_brother_event(participants_stats, team_xuids, me_xuid, lang)
            if traitor is not None:
                events.append(traitor)

    return events
