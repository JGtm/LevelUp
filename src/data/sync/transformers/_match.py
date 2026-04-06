"""Transformation des matchs : match_stats, participants, registry.

Fonctions publiques :
- create_metadata_resolver : wrapper autour de MetadataResolver
- transform_match_stats    : JSON match → MatchStatsRow
- extract_xuids_from_match : JSON match → [xuid int]
- extract_participants      : JSON match → [MatchParticipantRow]
- extract_match_registry_data : JSON match → dict (colonnes match_registry)
"""

from __future__ import annotations

import json
import logging
from collections.abc import Callable
from datetime import timedelta
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

from src.data.domain.refdata import Outcome
from src.data.sync.metadata_resolver import create_metadata_resolver_function
from src.data.sync.models import (
    MatchParticipantRow,
    MatchStatsRow,
)
from src.data.sync.transformers._helpers import (
    XUID_RE,
    _determine_mode_category,
    _extract_asset_id,
    _extract_kda,
    _extract_life_time_stats,
    _extract_mmr_from_skill,
    _extract_player_outcome_team,
    _extract_player_rank,
    _extract_player_score,
    _extract_player_stats,
    _extract_public_name,
    _extract_spree_headshots,
    _extract_team_scores,
    _extract_team_scores_by_id,
    _find_core_stats_dict,
    _find_player,
    _is_firefight_match,
    _is_ranked_playlist,
    _is_uuid,
    _normalize_gamertag,
    _parse_iso_utc,
    _safe_float,
    _safe_int,
    compute_teammates_signature,
    extract_game_variant_category,
)


def create_metadata_resolver(
    metadata_db_path: Path | str | None = None,
) -> Callable[[str, str], str | None] | None:
    """Crée une fonction de résolution des noms depuis metadata.duckdb.

    Cette fonction est un wrapper autour de MetadataResolver pour maintenir
    la compatibilité avec le code existant.

    Args:
        metadata_db_path: Chemin vers metadata.duckdb (auto-détecté si None).

    Returns:
        Fonction resolver(asset_type, asset_id) -> name | None, ou None si metadata.duckdb n'existe pas.
    """
    return create_metadata_resolver_function(metadata_db_path)


def transform_match_stats(  # noqa: C901, PLR0912, PLR0915
    match_json: dict[str, Any],
    xuid: str,
    *,
    skill_json: dict[str, Any] | None = None,
    metadata_resolver: Callable[[str, str | None], str | None] | None = None,
) -> MatchStatsRow | None:
    """Transforme le JSON API en MatchStatsRow pour DuckDB.

    Args:
        match_json: JSON brut de l'API SPNKr (MatchStats).
        xuid: XUID du joueur principal.
        skill_json: JSON skill optionnel (PlayerMatchStats) pour MMR.

    Returns:
        MatchStatsRow ou None si le parsing échoue.
    """
    match_id = match_json.get("MatchId")
    if not isinstance(match_id, str):
        return None

    match_info = match_json.get("MatchInfo")
    if not isinstance(match_info, dict):
        return None

    # Parse start_time
    start_time_raw = match_info.get("StartTime")
    start_time = _parse_iso_utc(start_time_raw)
    if start_time is None:
        return None

    # Trouver le joueur
    players = match_json.get("Players")
    if not isinstance(players, list):
        return None

    me = _find_player(players, xuid)
    if me is None:
        return None

    # Extraire les stats de base
    kills, deaths, assists, accuracy = _extract_player_stats(me)
    outcome, team_id = _extract_player_outcome_team(me)
    rank = _extract_player_rank(me)
    kda = _extract_kda(me)
    max_spree, headshots = _extract_spree_headshots(me)
    avg_life, time_played = _extract_life_time_stats(me, match_json)
    my_team_score, enemy_team_score = _extract_team_scores(match_json, team_id)

    # Extraire les identifiants d'assets
    playlist_id = _extract_asset_id(match_info, "Playlist")
    playlist_name = _extract_public_name(match_info, "Playlist")
    map_id = _extract_asset_id(match_info, "MapVariant")
    map_name = _extract_public_name(match_info, "MapVariant")
    pair_id = _extract_asset_id(match_info, "PlaylistMapModePair")
    pair_name = _extract_public_name(match_info, "PlaylistMapModePair")
    game_variant_id = _extract_asset_id(match_info, "UgcGameVariant")
    game_variant_name = _extract_public_name(match_info, "UgcGameVariant")

    # Résolution depuis les référentiels si les noms sont NULL mais les IDs sont présents
    # OU si les noms sont des UUIDs (fallback précédent qui a stocké l'ID)
    if metadata_resolver:
        # Vérifier si playlist_name est un UUID (format UUID standard)
        if playlist_id and (not playlist_name or _is_uuid(playlist_name)):
            resolved = metadata_resolver("playlist", playlist_id)
            if resolved:
                playlist_name = resolved
        # Vérifier si map_name est un UUID
        if map_id and (not map_name or _is_uuid(map_name)):
            resolved = metadata_resolver("map", map_id)
            if resolved:
                map_name = resolved
        # Vérifier si pair_name est un UUID
        if pair_id and (not pair_name or _is_uuid(pair_name)):
            resolved = metadata_resolver("pair", pair_id)
            if resolved:
                pair_name = resolved
        # Vérifier si game_variant_name est un UUID
        if game_variant_id and (not game_variant_name or _is_uuid(game_variant_name)):
            resolved = metadata_resolver("game_variant", game_variant_id)
            if resolved:
                game_variant_name = resolved

    # Fallback sur les IDs si les noms sont toujours NULL
    playlist_name = playlist_name or playlist_id
    map_name = map_name or map_id
    pair_name = pair_name or pair_id
    game_variant_name = game_variant_name or game_variant_id

    # Extraire MMR depuis skill_json si disponible
    team_mmr, enemy_mmr = None, None
    if skill_json:
        mmr_data = _extract_mmr_from_skill(skill_json, xuid, team_id)
        if mmr_data:
            team_mmr, enemy_mmr = mmr_data

    # Stats additionnelles depuis le dict de stats
    stats_dict = _find_core_stats_dict(me)
    damage_dealt = _safe_float(stats_dict.get("DamageDealt")) if stats_dict else None
    damage_taken = _safe_float(stats_dict.get("DamageTaken")) if stats_dict else None
    shots_fired = _safe_int(stats_dict.get("ShotsFired")) if stats_dict else None
    shots_hit = _safe_int(stats_dict.get("ShotsHit")) if stats_dict else None
    grenade_kills = _safe_int(stats_dict.get("GrenadeKills")) if stats_dict else None
    melee_kills = _safe_int(stats_dict.get("MeleeKills")) if stats_dict else None
    power_weapon_kills = _safe_int(stats_dict.get("PowerWeaponKills")) if stats_dict else None
    score = _safe_int(stats_dict.get("Score")) if stats_dict else None
    personal_score = _safe_int(stats_dict.get("PersonalScore")) if stats_dict else None

    # Déterminer les flags
    is_ranked = _is_ranked_playlist(match_info)
    is_firefight = _is_firefight_match(match_info)
    mode_category = _determine_mode_category(pair_name)

    # Sprint 2: Extraire GameVariantCategory (6=Slayer, 15=CTF, etc.)
    game_variant_category = extract_game_variant_category(match_json)

    # Déterminer si le joueur a quitté prématurément
    left_early = outcome == Outcome.DID_NOT_FINISH

    # Calculer la signature des coéquipiers
    teammates_signature = compute_teammates_signature(match_json, xuid, team_id)

    # Heure de fin du match : start_time + time_played_seconds
    end_time = None
    if start_time is not None and time_played is not None and time_played >= 0:
        end_time = start_time + timedelta(seconds=time_played)

    return MatchStatsRow(
        match_id=match_id,
        start_time=start_time,
        end_time=end_time,
        playlist_id=playlist_id,
        playlist_name=playlist_name,
        map_id=map_id,
        map_name=map_name,
        pair_id=pair_id,
        pair_name=pair_name,
        game_variant_id=game_variant_id,
        game_variant_name=game_variant_name,
        outcome=outcome,
        team_id=team_id,
        rank=rank,
        kills=kills,
        deaths=deaths,
        assists=assists,
        kda=kda,
        accuracy=accuracy,
        headshot_kills=headshots,
        max_killing_spree=max_spree,
        time_played_seconds=time_played,
        avg_life_seconds=avg_life,
        my_team_score=my_team_score,
        enemy_team_score=enemy_team_score,
        team_mmr=team_mmr,
        enemy_mmr=enemy_mmr,
        damage_dealt=damage_dealt,
        damage_taken=damage_taken,
        shots_fired=shots_fired,
        shots_hit=shots_hit,
        grenade_kills=grenade_kills,
        melee_kills=melee_kills,
        power_weapon_kills=power_weapon_kills,
        score=score,
        personal_score=personal_score,
        mode_category=mode_category,
        game_variant_category=game_variant_category,
        is_ranked=is_ranked,
        is_firefight=is_firefight,
        left_early=left_early,
        teammates_signature=teammates_signature,
    )


def extract_xuids_from_match(match_json: dict[str, Any]) -> list[int]:
    """Extrait tous les XUIDs d'un match.

    Args:
        match_json: JSON brut du match.

    Returns:
        Liste d'entiers XUID.
    """
    players = match_json.get("Players")
    if not isinstance(players, list):
        return []

    xuids = []
    seen = set()

    for player in players:
        if not isinstance(player, dict):
            continue

        pid = player.get("PlayerId")
        s = pid if isinstance(pid, str) else json.dumps(pid) if pid else None
        if not s:
            continue

        m = XUID_RE.search(s)
        if not m:
            continue

        try:
            xuid = int(m.group(1))
            if xuid not in seen:
                seen.add(xuid)
                xuids.append(xuid)
        except ValueError:
            continue

    return xuids


def extract_participants(match_json: dict[str, Any]) -> list[MatchParticipantRow]:  # noqa: C901, PLR0912
    """Extrait tous les participants d'un match (xuid, team_id, outcome, gamertag, score, rank, k/d/a).

    Source : MatchStats.Players[] (JSON API propre, pas les films corrompus).

    - Score : priorité API (PersonalScore ou Score depuis CoreStats).
    - Rang : priorité API (Players[].Rank) si présent, sinon calculé par tri score décroissant.
    - Kills, deaths, assists : depuis CoreStats (API).
    """
    match_id = match_json.get("MatchId")
    if not isinstance(match_id, str):
        return []

    players = match_json.get("Players")
    if not isinstance(players, list):
        return []

    rows: list[MatchParticipantRow] = []
    seen_xuids: set[str] = set()

    for player in players:
        if not isinstance(player, dict):
            continue

        # Extraire le XUID
        pid = player.get("PlayerId")
        xuid = None
        if isinstance(pid, str):
            m = XUID_RE.search(pid)
            if m:
                xuid = m.group(1)
            elif pid.strip().lower().startswith("bid("):
                # Bot Halo Infinite : PlayerId direct de la forme "bid(0.0)"
                xuid = pid.strip()
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

        seen_xuids.add(xuid)

        # Extraire team_id et outcome
        team_id = _safe_int(player.get("LastTeamId"))
        outcome = _safe_int(player.get("Outcome"))

        # Extraire et normaliser le gamertag
        gamertag_raw = player.get("PlayerGamertag") or player.get("Gamertag")
        gamertag = _normalize_gamertag(gamertag_raw)

        # Score (API prioritaire)
        score = _extract_player_score(player)

        # Rang API si présent (sinon sera assigné après tri)
        rank_from_api = _extract_player_rank(player)

        # K/D/A depuis CoreStats (API) — 0 est une valeur valide
        kills_val, deaths_val, assists_val, _ = _extract_player_stats(player)

        # ShotsFired / ShotsHit depuis CoreStats (API)
        stats_dict = _find_core_stats_dict(player)
        shots_fired_val = _safe_int(stats_dict.get("ShotsFired")) if stats_dict else None
        shots_hit_val = _safe_int(stats_dict.get("ShotsHit")) if stats_dict else None

        # DamageDealt / DamageTaken depuis CoreStats (API)
        damage_dealt_val = _safe_float(stats_dict.get("DamageDealt")) if stats_dict else None
        damage_taken_val = _safe_float(stats_dict.get("DamageTaken")) if stats_dict else None

        # AverageLifeSeconds et TimePlayedSeconds depuis CoreStats (API)
        avg_life_val, time_played_val = _extract_life_time_stats(player, match_json)

        # KDA depuis CoreStats (API)
        kda_val = _extract_kda(player)

        # MaxKillingSpree et HeadshotKills depuis CoreStats (API)
        max_spree_val, headshots_val = _extract_spree_headshots(player)

        # GrenadeKills, MeleeKills, PowerWeaponKills depuis CoreStats (API)
        grenade_kills_val = _safe_int(stats_dict.get("GrenadeKills")) if stats_dict else None
        melee_kills_val = _safe_int(stats_dict.get("MeleeKills")) if stats_dict else None
        power_weapon_kills_val = (
            _safe_int(stats_dict.get("PowerWeaponKills")) if stats_dict else None
        )

        # PersonalScore depuis CoreStats (API) - distinct du Score
        personal_score_val = _safe_int(stats_dict.get("PersonalScore")) if stats_dict else None

        # Accuracy calculée (shots_hit / shots_fired * 100)
        accuracy_val = None
        if shots_fired_val and shots_fired_val > 0 and shots_hit_val is not None:
            accuracy_val = round(shots_hit_val * 100.0 / shots_fired_val, 2)

        rows.append(
            MatchParticipantRow(
                match_id=match_id,
                xuid=xuid,
                team_id=team_id,
                outcome=outcome,
                gamertag=gamertag,
                rank=rank_from_api,
                score=score,
                kills=kills_val,
                deaths=deaths_val,
                assists=assists_val,
                shots_fired=shots_fired_val,
                shots_hit=shots_hit_val,
                damage_dealt=damage_dealt_val,
                damage_taken=damage_taken_val,
                avg_life_seconds=avg_life_val,
                headshot_kills=headshots_val,
                max_killing_spree=max_spree_val,
                kda=kda_val,
                accuracy=accuracy_val,
                time_played_seconds=time_played_val,
                grenade_kills=grenade_kills_val,
                melee_kills=melee_kills_val,
                power_weapon_kills=power_weapon_kills_val,
                personal_score=personal_score_val,
            )
        )

    # Trier par score décroissant (None en dernier) et assigner le rang si pas fourni par l'API
    def sort_key(r: MatchParticipantRow) -> tuple[int, int]:
        s = r.score if r.score is not None else -1
        return (-s, 0)

    rows.sort(key=sort_key)
    for rank_idx, r in enumerate(rows, start=1):
        if r.rank is None:
            r.rank = rank_idx

    return rows


def extract_match_registry_data(  # noqa: C901, PLR0912, PLR0915
    match_json: dict[str, Any],
    *,
    metadata_resolver: Callable[[str, str | None], str | None] | None = None,
) -> dict[str, Any] | None:
    """Extrait les données communes d'un match pour match_registry (v5).

    Données indépendantes du joueur : map, playlist, scores d'équipe,
    durée, mode, etc.

    Args:
        match_json: JSON brut du match depuis l'API SPNKr.
        metadata_resolver: Résolveur de noms depuis metadata.duckdb.

    Returns:
        Dict avec toutes les colonnes de match_registry, ou None si parsing échoue.
    """
    match_id = match_json.get("MatchId")
    if not isinstance(match_id, str):
        return None

    match_info = match_json.get("MatchInfo")
    if not isinstance(match_info, dict):
        return None

    # Parse start_time
    start_time_raw = match_info.get("StartTime")
    start_time = _parse_iso_utc(start_time_raw)
    if start_time is None:
        return None

    # Extraire les identifiants d'assets
    playlist_id = _extract_asset_id(match_info, "Playlist")
    playlist_name = _extract_public_name(match_info, "Playlist")
    map_id = _extract_asset_id(match_info, "MapVariant")
    map_name = _extract_public_name(match_info, "MapVariant")
    pair_id = _extract_asset_id(match_info, "PlaylistMapModePair")
    pair_name = _extract_public_name(match_info, "PlaylistMapModePair")
    game_variant_id = _extract_asset_id(match_info, "UgcGameVariant")
    game_variant_name = _extract_public_name(match_info, "UgcGameVariant")

    # Résolution depuis les référentiels
    if metadata_resolver:
        if playlist_id and (not playlist_name or _is_uuid(playlist_name)):
            resolved = metadata_resolver("playlist", playlist_id)
            if resolved:
                playlist_name = resolved
        if map_id and (not map_name or _is_uuid(map_name)):
            resolved = metadata_resolver("map", map_id)
            if resolved:
                map_name = resolved
        if pair_id and (not pair_name or _is_uuid(pair_name)):
            resolved = metadata_resolver("pair", pair_id)
            if resolved:
                pair_name = resolved
        if game_variant_id and (not game_variant_name or _is_uuid(game_variant_name)):
            resolved = metadata_resolver("game_variant", game_variant_id)
            if resolved:
                game_variant_name = resolved

    # Fallback sur les IDs si les noms sont toujours NULL
    playlist_name = playlist_name or playlist_id
    map_name = map_name or map_id
    pair_name = pair_name or pair_id
    game_variant_name = game_variant_name or game_variant_id

    # Flags
    is_ranked = _is_ranked_playlist(match_info)
    is_firefight = _is_firefight_match(match_info)
    mode_category = _determine_mode_category(pair_name)

    # Durée : depuis MatchInfo.Duration (durée globale du match)
    from src.data.sync.transformers._helpers import _parse_duration_to_seconds

    duration_seconds: int | None = None
    duration_raw = match_info.get("Duration")
    if isinstance(duration_raw, str):
        duration_seconds = _parse_duration_to_seconds(duration_raw)

    # Durée jouable réelle (sans countdown/lobby) — depuis PlayableDuration
    playable_duration_seconds: int | None = None
    playable_raw = match_info.get("PlayableDuration")
    if isinstance(playable_raw, str):
        playable_duration_seconds = _parse_duration_to_seconds(playable_raw)
        if playable_duration_seconds is None:
            logger.warning("PlayableDuration non parsé pour match %s: %r", match_id, playable_raw)
    else:
        logger.debug("PlayableDuration absent du JSON pour match %s", match_id)

    # Heure de fin — EndTime API en priorité, calcul en fallback
    end_time_raw = match_info.get("EndTime")
    end_time = _parse_iso_utc(end_time_raw) if isinstance(end_time_raw, str) else None
    if (
        end_time is None
        and start_time is not None
        and duration_seconds is not None
        and duration_seconds >= 0
    ):
        end_time = start_time + timedelta(seconds=duration_seconds)

    # Début réel du gameplay (après countdown/lobby)
    real_start_time = None
    if (
        start_time is not None
        and duration_seconds is not None
        and playable_duration_seconds is not None
    ):
        countdown_s = duration_seconds - playable_duration_seconds
        if countdown_s >= 0:
            real_start_time = start_time + timedelta(seconds=countdown_s)
            logger.debug(
                "Match %s : countdown=%ds, real_start=%s",
                match_id,
                countdown_s,
                real_start_time,
            )
        else:
            logger.warning(
                "Match %s : playable_duration_seconds (%d) > duration_seconds (%d) — "
                "real_start_time ignoré",
                match_id,
                playable_duration_seconds,
                duration_seconds,
            )

    # Scores des équipes (team_0 et team_1, indépendants du joueur)
    team_0_score, team_1_score = _extract_team_scores_by_id(match_json)

    return {
        "match_id": match_id,
        "start_time": start_time,
        "end_time": end_time,
        "playlist_id": playlist_id,
        "playlist_name": playlist_name,
        "map_id": map_id,
        "map_name": map_name,
        "pair_id": pair_id,
        "pair_name": pair_name,
        "game_variant_id": game_variant_id,
        "game_variant_name": game_variant_name,
        "mode_category": mode_category,
        "is_ranked": is_ranked,
        "is_firefight": is_firefight,
        "duration_seconds": duration_seconds,
        "playable_duration_seconds": playable_duration_seconds,
        "real_start_time": real_start_time,
        "team_0_score": team_0_score,
        "team_1_score": team_1_score,
    }
