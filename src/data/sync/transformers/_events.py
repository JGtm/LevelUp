"""Transformation des événements highlight, aliases et paires killer→victim.

Fonctions publiques :
- transform_highlight_events                      : events + match_id → [HighlightEventRow]
- extract_aliases                                 : JSON match → [XuidAliasRow]
- build_players_lookup                            : JSON match → {xuid: gamertag}
- extract_killer_victim_pairs_from_highlight_events : events + match_id → [KillerVictimPairRow]
"""

from __future__ import annotations

import json
from datetime import datetime, timezone
from typing import Any

from src.data.sync.models import (
    HighlightEventRow,
    KillerVictimPairRow,
    XuidAliasRow,
)
from src.data.sync.transformers._helpers import (
    XUID_RE,
    _normalize_gamertag,
    _safe_int,
    _safe_str,
)


def transform_highlight_events(
    events: list[Any],
    match_id: str,
) -> list[HighlightEventRow]:
    """Transforme les highlight events en HighlightEventRows.

    Args:
        events: Liste des events (objets Pydantic ou dicts).
        match_id: ID du match.

    Returns:
        Liste de HighlightEventRow.
    """
    rows = []

    for event in events:
        # Convertir l'event en dict si nécessaire
        event_dict: dict[str, Any]
        if isinstance(event, dict):
            event_dict = event
        elif hasattr(event, "model_dump"):
            event_dict = event.model_dump()
        elif hasattr(event, "dict"):
            event_dict = event.dict()
        elif hasattr(event, "_asdict"):
            event_dict = event._asdict()
        else:
            continue

        event_type = event_dict.get("event_type")
        if not isinstance(event_type, str):
            continue

        time_ms = _safe_int(event_dict.get("time_ms")) or 0
        xuid = _safe_str(event_dict.get("xuid"))
        gamertag = _safe_str(event_dict.get("gamertag"))
        type_hint = _safe_int(event_dict.get("type_hint"))

        rows.append(
            HighlightEventRow(
                match_id=match_id,
                event_type=event_type,
                time_ms=time_ms,
                xuid=xuid,
                gamertag=gamertag,
                type_hint=type_hint,
                raw_json=json.dumps(event_dict, ensure_ascii=False),
            )
        )

    return rows


def extract_aliases(  # noqa: PLR0912
    match_json: dict[str, Any],
    *,
    source: str = "match_roster",
) -> list[XuidAliasRow]:
    """Extrait les paires XUID → Gamertag d'un match.

    Aligné avec le script legacy spnkr_import_db (_extract_gamertags_from_match_stats).
    Gère correctement l'encodage des gamertags (évite troncature/mojibake).

    Args:
        match_json: JSON brut du match.
        source: Source de l'alias (pour traçabilité).

    Returns:
        Liste de XuidAliasRow.
    """
    players = match_json.get("Players")
    if not isinstance(players, list):
        return []

    now = datetime.now(timezone.utc)
    aliases = []
    seen_xuids = set()

    for player in players:
        if not isinstance(player, dict):
            continue

        pid = player.get("PlayerId")
        gamertag_raw = player.get("PlayerGamertag") or player.get("Gamertag")

        # Extraire le XUID (aligné legacy: str ou json.dumps pour dict)
        xuid = None
        if isinstance(pid, str):
            m = XUID_RE.search(pid)
            if m:
                xuid = m.group(1)
        elif isinstance(pid, dict):
            xuid_val = pid.get("Xuid") or pid.get("xuid")
            if xuid_val is not None:
                xuid = str(xuid_val)
            else:
                try:
                    s = json.dumps(pid)
                    m = XUID_RE.search(s)
                    if m:
                        xuid = m.group(1)
                except (TypeError, ValueError):
                    pass

        if not xuid or xuid in seen_xuids:
            continue

        # Nettoyer et normaliser le gamertag (évite troncature/encodage)
        gt = _normalize_gamertag(gamertag_raw)
        if not gt:
            continue

        seen_xuids.add(xuid)
        aliases.append(
            XuidAliasRow(
                xuid=xuid,
                gamertag=gt,
                last_seen=now,
                source=source,
            )
        )

    return aliases


def build_players_lookup(match_json: dict[str, Any]) -> dict[str, str]:
    """Construit un dictionnaire XUID → Gamertag depuis le JSON du match.

    Args:
        match_json: JSON brut du match (clé Players).

    Returns:
        Dict mapping xuid (str) → gamertag (str).
    """
    aliases = extract_aliases(match_json)
    return {a.xuid: a.gamertag for a in aliases}


def extract_killer_victim_pairs_from_highlight_events(
    events: list[dict[str, Any]],
    match_id: str,
    *,
    players_lookup: dict[str, str] | None = None,
) -> list[KillerVictimPairRow]:
    """Extrait les paires killer→victim depuis les highlight events de type Kill.

    Args:
        events: Liste d'events (dicts avec event_type, xuid, victim_xuid, etc.).
        match_id: ID du match.
        players_lookup: Optionnel, mapping xuid → gamertag pour enrichir les rows.

    Returns:
        Liste de KillerVictimPairRow.
    """
    rows = []
    lookup = players_lookup or {}

    for event in events:
        if not isinstance(event, dict):
            continue
        if event.get("event_type") != "Kill":
            continue

        killer_xuid = _safe_str(event.get("xuid"))
        victim_xuid = _safe_str(event.get("victim_xuid"))
        if not killer_xuid or not victim_xuid:
            continue

        killer_gamertag = _safe_str(event.get("gamertag")) or lookup.get(killer_xuid)
        victim_gamertag = _safe_str(event.get("victim_gamertag")) or lookup.get(victim_xuid)
        time_ms = _safe_int(event.get("time_ms"))

        rows.append(
            KillerVictimPairRow(
                match_id=match_id,
                killer_xuid=killer_xuid,
                victim_xuid=victim_xuid,
                killer_gamertag=killer_gamertag,
                victim_gamertag=victim_gamertag,
                kill_count=1,
                time_ms=time_ms,
            )
        )

    return rows
