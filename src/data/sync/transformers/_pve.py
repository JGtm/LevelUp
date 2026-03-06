"""Extraction des stats PvE / Firefight (v5.2).

Fonctions publiques :
- extract_pve_stats : JSON match → [PveMatchStatsRow]

Fonctions et constantes privées (internes) :
- _FIREFIGHT_CATEGORY_IDS   : frozenset des IDs GameVariantCategory Firefight
- _find_pve_stats_dict      : trouve le bloc de stats PvE dans PlayerTeamStats
- _extract_enemy_kills_by_type : extrait les kills par type d'ennemi

Note : _FIREFIGHT_CATEGORY_IDS et _is_firefight_match sont définis dans _helpers.py
pour être partagés avec d'autres modules. Ce module les importe depuis _helpers.
"""

from __future__ import annotations

from typing import Any

from src.data.sync.models import PveMatchStatsRow
from src.data.sync.transformers._helpers import (
    _FIREFIGHT_CATEGORY_IDS,
    XUID_RE,
    _is_firefight_match,
    _safe_int,
)


def _find_pve_stats_dict(player_obj: dict[str, Any]) -> dict[str, Any] | None:  # noqa: C901
    """Trouve le dictionnaire contenant les stats PvE dans PlayerTeamStats.

    Parcourt récursivement PlayerTeamStats pour trouver le bloc de stats PvE.
    Cherche les clés connues (PveStats, EliminationStats, FirefightStats) en
    priorité, puis détecte par présence de clés caractéristiques de l'API
    PveStats (Kills, BossKills, GruntKills, etc.).

    Args:
        player_obj: Objet joueur du JSON API (Players[]).

    Returns:
        Dict des stats PvE ou None si non trouvé.
    """
    # Clés confirmées de l'interface PveStats de l'API
    _pve_keys = {"BossKills", "GruntKills", "EliteKills", "SentinelKills", "MarineKills"}
    # Liste ordonnée — PveStats en premier pour éviter de retourner EliminationStats
    # quand les deux coexistent dans Stats (confirmé sur matchs Firefight 2025).
    _known_block_names = ["PveStats", "FirefightStats", "SurvivalStats", "EliminationStats"]

    def _find(x: Any, depth: int = 0) -> dict[str, Any] | None:
        if depth > 10:
            return None  # Protection contre la récursion infinie sur JSON malformé
        if isinstance(x, dict):
            # Chemin direct par nom de bloc connu
            for key in _known_block_names:
                if key in x:
                    val = x[key]
                    if isinstance(val, dict):
                        return val
            # Détection par clés PvE caractéristiques
            if any(k in x for k in _pve_keys):
                return x
            # Recherche récursive
            for v in x.values():
                r = _find(v, depth + 1)
                if r is not None:
                    return r
        elif isinstance(x, list):
            for v in x:
                r = _find(v, depth + 1)
                if r is not None:
                    return r
        return None

    return _find(player_obj.get("PlayerTeamStats"))


def _extract_enemy_kills_by_type(pve_dict: dict[str, Any]) -> dict[str, int]:
    """Extrait les kills par type d'ennemi depuis le bloc PvE.

    Champs validés depuis l'interface PveStats de l'API Halo Infinite :
    GruntKills, EliteKills, JackalKills, BruteKills, HunterKills,
    SkimmerKills, SentinelKills, MarineKills.

    Args:
        pve_dict: Dictionnaire des stats PvE.

    Returns:
        Dict {enemy_type: kill_count}.
    """
    # Mapping interne → clé(s) API possibles
    direct_mappings: dict[str, list[str]] = {
        "grunt": ["GruntKills", "Grunts"],
        "elite": ["EliteKills", "Elites"],
        "jackal": ["JackalKills", "Jackals"],
        "brute": ["BruteKills", "Brutes"],
        "hunter": ["HunterKills", "Hunters"],
        "skimmer": ["SkimmerKills", "Skimmers"],
        "sentinel": ["SentinelKills", "Sentinels"],
        "marine": ["MarineKills", "Marines"],
    }

    result: dict[str, int] = {}
    for enemy_type, api_keys in direct_mappings.items():
        for key in api_keys:
            val = pve_dict.get(key)
            if val is not None:
                result[enemy_type] = _safe_int(val) or 0
                break

    return result


def extract_pve_stats(match_json: dict[str, Any]) -> list[PveMatchStatsRow]:
    """Extrait les stats PvE de tous les joueurs d'un match Firefight.

    Ne retourne des données que si le match est identifié comme Firefight.
    Extrait les stats pour TOUS les joueurs du match (partage cohérent).

    Args:
        match_json: JSON brut du match (MatchStats API).

    Returns:
        Liste de PveMatchStatsRow pour chaque joueur, vide si pas un match PvE.
    """
    match_info = match_json.get("MatchInfo") or {}
    if not _is_firefight_match(match_info):
        return []

    match_id = match_json.get("MatchId")
    if not isinstance(match_id, str) or not match_id:
        return []

    players = match_json.get("Players")
    if not isinstance(players, list):
        return []

    rows: list[PveMatchStatsRow] = []

    for player in players:
        if not isinstance(player, dict):
            continue

        # Extraire le XUID depuis PlayerId (format: "xuid(123456789)")
        pid = player.get("PlayerId")
        xuid: str | None = None
        if isinstance(pid, str):
            m = XUID_RE.search(pid)
            if m:
                xuid = m.group(1)

        if not xuid:
            continue

        # Trouver le bloc de stats PvE
        pve_dict = _find_pve_stats_dict(player)
        if pve_dict is None:
            # Pas de stats PvE (bot, spectateur, ou structure non reconnue)
            continue

        enemy_kills = _extract_enemy_kills_by_type(pve_dict)

        rows.append(
            PveMatchStatsRow(
                match_id=match_id,
                xuid=xuid,
                total_enemy_kills=_safe_int(pve_dict.get("Kills")),
                boss_kills=_safe_int(pve_dict.get("BossKills")),
                grunt_kills=enemy_kills.get("grunt", 0),
                elite_kills=enemy_kills.get("elite", 0),
                jackal_kills=enemy_kills.get("jackal", 0),
                brute_kills=enemy_kills.get("brute", 0),
                hunter_kills=enemy_kills.get("hunter", 0),
                skimmer_kills=enemy_kills.get("skimmer", 0),
                sentinel_kills=enemy_kills.get("sentinel", 0),
                marine_kills=enemy_kills.get("marine", 0),
            )
        )

    return rows


# Ré-export pour compatibilité avec du code qui accèderait directement à ce module
__all__ = [
    "_FIREFIGHT_CATEGORY_IDS",
    "_extract_enemy_kills_by_type",
    "_find_pve_stats_dict",
    "extract_pve_stats",
]
