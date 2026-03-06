"""Fonctions utilitaires pures et constantes partagées entre les modules transformers.

Ce module contient :
- Constantes (XUID_RE, logger)
- Helpers de parsing sûrs (_safe_float, _safe_int, _safe_str, _parse_iso_utc)
- Helpers d'extraction JSON (_find_player, _find_core_stats_dict, etc.)
- Fonctions de détermination de mode (_is_ranked_playlist, _is_firefight_match, etc.)
"""

from __future__ import annotations

import json
import logging
import math
import re
from datetime import datetime
from typing import Any

from src.analysis.mode_categories import infer_custom_category_from_pair_name

logger = logging.getLogger(__name__)

# =============================================================================
# Regex et constantes
# =============================================================================

XUID_RE = re.compile(r"(\d{12,20})")

# Gamertags Xbox valides : alphanum + espace, 1-15 chars, pas de null bytes ni mojibake
_VALID_GAMERTAG_RE = re.compile(r"^[a-zA-Z0-9][a-zA-Z0-9 ]{0,14}$")


# =============================================================================
# Helpers de parsing
# =============================================================================


def _safe_float(v: Any) -> float | None:
    """Convertit une valeur en float, gérant NaN et None."""
    if v is None:
        return None
    try:
        f = float(v)
        if math.isnan(f) or math.isinf(f):
            return None
        return f
    except (TypeError, ValueError):
        return None


def _safe_int(v: Any) -> int | None:
    """Convertit une valeur en int, gérant NaN et None."""
    if v is None:
        return None
    try:
        f = float(v)
        if math.isnan(f) or math.isinf(f):
            return None
        return int(f)
    except (TypeError, ValueError):
        return None


def _safe_str(v: Any) -> str | None:
    """Convertit une valeur en str, gérant None."""
    if v is None:
        return None
    try:
        s = str(v)
        if s == "nan" or s == "None":
            return None
        return s
    except Exception:
        return None


def _parse_iso_utc(s: str | None) -> datetime | None:
    """Parse un timestamp ISO 8601 en datetime UTC."""
    if not s or not isinstance(s, str):
        return None
    try:
        # Gérer les formats avec ou sans 'Z'
        s = s.replace("Z", "+00:00")
        if "+" not in s and "-" not in s[10:]:
            s += "+00:00"
        return datetime.fromisoformat(s)
    except Exception:
        return None


def _find_player(players: list[dict[str, Any]], xuid: str) -> dict[str, Any] | None:
    """Trouve un joueur dans la liste par son XUID."""
    for pl in players:
        pid = pl.get("PlayerId")
        if pid is None:
            continue
        if xuid in json.dumps(pid):
            return pl
    return None


def _find_core_stats_dict(player_obj: dict[str, Any]) -> dict[str, Any] | None:
    """Trouve le dictionnaire contenant les stats Kills/Deaths/Assists.

    Parcourt récursivement PlayerTeamStats pour trouver le dict avec les stats.
    """
    targets = {"Kills", "Deaths", "Assists", "ShotsFired", "ShotsHit", "Accuracy"}

    def find_stats_dict(x: Any) -> dict[str, Any] | None:
        if isinstance(x, dict):
            if (
                "Kills" in x
                and "Deaths" in x
                and any(k in x for k in targets)
                and (
                    _safe_int(x.get("Kills")) is not None or _safe_int(x.get("Deaths")) is not None
                )
            ):
                return x
            for v in x.values():
                r = find_stats_dict(v)
                if r is not None:
                    return r
        elif isinstance(x, list):
            for v in x:
                r = find_stats_dict(v)
                if r is not None:
                    return r
        return None

    return find_stats_dict(player_obj.get("PlayerTeamStats"))


def _extract_player_stats(player_obj: dict[str, Any]) -> tuple[int, int, int, float | None]:
    """Extrait kills, deaths, assists, accuracy d'un joueur."""
    stats_dict = _find_core_stats_dict(player_obj)
    if stats_dict is None:
        return 0, 0, 0, None

    kills = _safe_int(stats_dict.get("Kills")) or 0
    deaths = _safe_int(stats_dict.get("Deaths")) or 0
    assists = _safe_int(stats_dict.get("Assists")) or 0
    accuracy = _safe_float(stats_dict.get("Accuracy"))
    return kills, deaths, assists, accuracy


def _extract_player_outcome_team(player_obj: dict[str, Any]) -> tuple[int | None, int | None]:
    """Extrait outcome et team_id d'un joueur."""
    outcome = player_obj.get("Outcome")
    last_team_id = player_obj.get("LastTeamId")
    outcome_i = int(outcome) if isinstance(outcome, int) else None
    team_i = int(last_team_id) if isinstance(last_team_id, int) else None
    return outcome_i, team_i


def _extract_player_rank(player_obj: dict[str, Any]) -> int | None:
    """Extrait le rang du joueur dans le match."""
    rank = player_obj.get("Rank")
    return int(rank) if isinstance(rank, int) else None


def _extract_kda(player_obj: dict[str, Any]) -> float | None:
    """Extrait le KDA d'un joueur."""
    stats_dict = _find_core_stats_dict(player_obj)
    if stats_dict is not None:
        v = _safe_float(stats_dict.get("KDA"))
        if v is not None:
            return v
    return None


def _extract_spree_headshots(player_obj: dict[str, Any]) -> tuple[int | None, int | None]:
    """Extrait max_killing_spree et headshot_kills."""
    stats_dict = _find_core_stats_dict(player_obj)
    if stats_dict is None:
        return None, None

    max_spree = _safe_int(stats_dict.get("MaxKillingSpree"))
    headshots = _safe_int(stats_dict.get("HeadshotKills"))
    return max_spree, headshots


def _parse_duration_to_seconds(duration_str: str) -> int | None:
    """Parse une durée ISO 8601 (PT1H30M45S) en secondes."""
    if not duration_str or not isinstance(duration_str, str):
        return None

    # Format: PT{hours}H{minutes}M{seconds}S ou variations
    pattern = r"PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?"
    match = re.match(pattern, duration_str, re.IGNORECASE)
    if not match:
        return None

    hours = int(match.group(1) or 0)
    minutes = int(match.group(2) or 0)
    seconds = float(match.group(3) or 0)

    return int(hours * 3600 + minutes * 60 + seconds)


def _extract_life_time_stats(
    player_obj: dict[str, Any],
    match_obj: dict[str, Any] | None = None,
) -> tuple[float | None, int | None]:
    """Extrait avg_life_seconds et time_played_seconds.

    Args:
        player_obj: Objet joueur avec PlayerTeamStats.
        match_obj: Objet match complet (pour extraire Duration depuis MatchInfo).

    Returns:
        (avg_life_seconds, time_played_seconds)
    """
    stats_dict = _find_core_stats_dict(player_obj)

    # Extraire avg_life_seconds
    avg_life = None
    if stats_dict:
        # Essayer d'abord AverageLifeSeconds (format numérique)
        avg_life = _safe_float(stats_dict.get("AverageLifeSeconds"))

        # Si non trouvé, essayer AverageLifeDuration (format ISO: "PT49.3S")
        if avg_life is None:
            avg_life_duration = stats_dict.get("AverageLifeDuration")
            if isinstance(avg_life_duration, str):
                avg_life_secs = _parse_duration_to_seconds(avg_life_duration)
                avg_life = float(avg_life_secs) if avg_life_secs else None

    # Extraire time_played_seconds
    time_played = None

    # 1. Essayer depuis CoreStats
    if stats_dict:
        if "TimePlayed" in stats_dict:
            tp = stats_dict.get("TimePlayed")
            if isinstance(tp, str):
                time_played = _parse_duration_to_seconds(tp)
            elif isinstance(tp, int | float):
                time_played = _safe_int(tp)
        elif "TimePlayedSeconds" in stats_dict:
            time_played = _safe_int(stats_dict.get("TimePlayedSeconds"))

    # 2. Fallback: extraire depuis MatchInfo.Duration
    if time_played is None and match_obj:
        match_info = match_obj.get("MatchInfo")
        if isinstance(match_info, dict):
            duration = match_info.get("Duration")
            if isinstance(duration, str):
                time_played = _parse_duration_to_seconds(duration)

    return avg_life, time_played


def _extract_team_score_value(team: dict[str, Any]) -> int | None:
    """Extrait le score d'une entrée team du JSON Halo Infinite.

    L'API peut stocker le score à plusieurs endroits selon la version :
    - team["Score"] ou team["TotalPoints"] (format simplifié/legacy)
    - team["Stats"]["CoreStats"]["Score"] (format réel de l'API)
    """
    # Format direct (fixtures de test ou anciennes réponses)
    v = _safe_int(team.get("TotalPoints"))
    if v is not None:
        return v
    v = _safe_int(team.get("Score"))
    if v is not None:
        return v
    # Format réel : Stats.CoreStats.Score
    stats = team.get("Stats")
    if isinstance(stats, dict):
        core = stats.get("CoreStats")
        if isinstance(core, dict):
            v = _safe_int(core.get("Score"))
            if v is not None:
                return v
    return None


def _extract_team_scores(
    match_obj: dict[str, Any], team_id: int | None
) -> tuple[int | None, int | None]:
    """Extrait my_team_score et enemy_team_score."""
    teams = match_obj.get("Teams")
    if not isinstance(teams, list) or team_id is None:
        return None, None

    my_score = None
    enemy_scores = []

    for team in teams:
        if not isinstance(team, dict):
            continue
        tid = team.get("TeamId")
        score = _extract_team_score_value(team)

        if tid == team_id:
            my_score = score
        elif score is not None:
            enemy_scores.append(score)

    # Pour les modes FFA ou multi-équipes, prendre le max des ennemis
    enemy_score = max(enemy_scores) if enemy_scores else None

    return my_score, enemy_score


def _extract_team_scores_by_id(
    match_obj: dict[str, Any],
) -> tuple[int | None, int | None]:
    """Extrait les scores d'équipe par ID (team_0 et team_1).

    Contrairement à _extract_team_scores() qui est relative au joueur,
    cette fonction retourne les scores absolus par TeamId.

    Returns:
        (team_0_score, team_1_score)
    """
    teams = match_obj.get("Teams")
    if not isinstance(teams, list):
        return None, None

    team_scores: dict[int, int | None] = {}
    for team in teams:
        if not isinstance(team, dict):
            continue
        tid = team.get("TeamId")
        if tid is not None:
            team_scores[tid] = _extract_team_score_value(team)

    return team_scores.get(0), team_scores.get(1)


def _extract_asset_id(match_info: dict[str, Any], key: str) -> str | None:
    """Extrait l'AssetId d'un objet (Playlist, MapVariant, etc.)."""
    obj = match_info.get(key)
    if isinstance(obj, dict):
        asset_id = obj.get("AssetId")
        if isinstance(asset_id, str):
            return asset_id
    return None


def _extract_public_name(match_info: dict[str, Any], key: str) -> str | None:
    """Extrait le PublicName d'un objet si disponible."""
    obj = match_info.get(key)
    if isinstance(obj, dict):
        name = obj.get("PublicName")
        if isinstance(name, str):
            return name
    return None


def _is_uuid(value: str | None) -> bool:
    """Vérifie si une chaîne est un UUID (format standard)."""
    if not value or not isinstance(value, str):
        return False
    # Format UUID standard: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 caractères)
    uuid_pattern = re.compile(
        r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
        re.IGNORECASE,
    )
    return bool(uuid_pattern.match(value.strip()))


def _is_ranked_playlist(match_info: dict[str, Any]) -> bool:
    """Détermine si le match est ranked."""
    playlist = match_info.get("Playlist")
    if isinstance(playlist, dict):
        # Vérifier les tags ou le nom
        tags = playlist.get("Tags")
        if isinstance(tags, list) and "ranked" in [t.lower() for t in tags if isinstance(t, str)]:
            return True
        name = playlist.get("PublicName", "")
        if isinstance(name, str) and "ranked" in name.lower():
            return True
    return False


def _determine_mode_category(pair_name: str | None) -> str:
    """Détermine la catégorie custom de mode de jeu.

    Utilise la logique de mode_categories.py pour aligner avec les filtres UI.
    Retourne une des catégories : Assassin, Fiesta, BTB, Ranked, Firefight, Other.

    Args:
        pair_name: Nom du PlaylistMapModePair (ex: "Arena:Slayer on Aquarius").

    Returns:
        Catégorie custom (jamais None, "Other" par défaut).
    """
    return infer_custom_category_from_pair_name(pair_name)


def _extract_player_score(player_obj: dict[str, Any]) -> int | None:
    """Extrait le score personnel d'un joueur (PersonalScore ou Score)."""
    stats_dict = _find_core_stats_dict(player_obj)
    if stats_dict is None:
        return None
    score = _safe_int(stats_dict.get("PersonalScore"))
    if score is None:
        score = _safe_int(stats_dict.get("Score"))
    return score


def _extract_xuid(player: dict[str, Any] | str) -> str | None:
    """Extrait le XUID d'un joueur depuis son PlayerId.

    Args:
        player: Dict joueur ou PlayerId (str).

    Returns:
        XUID (str) ou None.
    """
    pid = player if isinstance(player, str) else player.get("PlayerId")

    if not pid:
        return None

    if isinstance(pid, str):
        m = XUID_RE.search(pid)
        if m:
            return m.group(1)
    elif isinstance(pid, dict):
        xuid_val = pid.get("Xuid") or pid.get("xuid")
        if xuid_val is not None:
            return str(xuid_val)
        else:
            try:
                s = json.dumps(pid)
                m = XUID_RE.search(s)
                if m:
                    return m.group(1)
            except (TypeError, ValueError):
                pass

    return None


def _normalize_gamertag(raw: str | bytes | Any) -> str | None:
    """Normalise un gamertag pour éviter troncature et problèmes d'encodage.

    Rejette les valeurs corrompues (null bytes, mojibake, chaînes purement
    numériques trop courtes). Les gamertags Xbox valides contiennent uniquement
    des caractères alphanumériques et espaces (1-15 chars).
    """
    if raw is None:
        return None
    if isinstance(raw, bytes):
        try:
            raw = raw.decode("utf-8", errors="replace")
        except Exception:
            return None
    s = str(raw).strip() if raw else ""
    if "\x00" in s:
        s = s.split("\x00")[0].strip()
        logger.debug("Gamertag tronqué (null bytes): %r → %r", raw, s)
    if not s or not _VALID_GAMERTAG_RE.match(s):
        if s:
            logger.debug("Gamertag rejeté (format invalide): %r", s)
        return None
    return s


def _extract_mmr_from_skill(  # noqa: C901, PLR0912
    skill_json: dict[str, Any],
    xuid: str,
    team_id: int | None,
) -> tuple[float | None, float | None] | None:
    """Extrait team_mmr et enemy_mmr depuis le JSON skill.

    Args:
        skill_json: JSON de l'API skill (PlayerMatchStats).
        xuid: XUID du joueur.
        team_id: ID de l'équipe du joueur (utilisé comme fallback si non trouvé dans Result).

    Returns:
        Tuple (team_mmr, enemy_mmr) où chaque valeur peut être None, ou None si le joueur n'est pas trouvé.
        Retourne None uniquement si le joueur n'est pas trouvé dans le JSON.
        Sinon, retourne toujours un tuple, même si une seule valeur est disponible.
    """
    value = skill_json.get("Value")
    if not isinstance(value, list):
        return None

    # Trouver notre joueur et extraire TeamMmrs
    my_result = None
    my_team_id = None

    for player in value:
        if not isinstance(player, dict):
            continue

        player_id = player.get("Id")
        player_xuid = None
        if isinstance(player_id, str):
            m = XUID_RE.search(player_id)
            if m:
                player_xuid = m.group(1)

        if player_xuid == xuid:
            my_result = player.get("Result")
            if isinstance(my_result, dict):
                my_team_id = _safe_int(my_result.get("TeamId"))
                break

    # Si le joueur n'est pas trouvé, retourner None
    if not my_result:
        return None

    # Utiliser team_id du paramètre si non trouvé dans Result
    if my_team_id is None:
        my_team_id = team_id

    # Extraire team_mmr depuis TeamMmr du joueur
    team_mmr = _safe_float(my_result.get("TeamMmr"))

    # Extraire enemy_mmr depuis TeamMmrs (recommandé)
    # TeamMmrs contient les MMR de toutes les équipes : {"0": 1200.5, "1": 1150.3}
    enemy_mmr = None
    team_mmrs_raw = my_result.get("TeamMmrs")
    if isinstance(team_mmrs_raw, dict) and my_team_id is not None:
        my_key = str(my_team_id)
        for k, v in team_mmrs_raw.items():
            if k != my_key:
                enemy_mmr = _safe_float(v)
                break

    # Fallback : utiliser TeamMmr d'un adversaire si TeamMmrs n'est pas disponible
    if enemy_mmr is None and my_team_id is not None:
        enemy_team_mmrs = []
        for player in value:
            if not isinstance(player, dict):
                continue
            result = player.get("Result")
            if not isinstance(result, dict):
                continue
            player_team = result.get("TeamId")
            player_team_mmr = _safe_float(result.get("TeamMmr"))
            if (
                player_team is not None
                and player_team != my_team_id
                and player_team_mmr is not None
            ):
                enemy_team_mmrs.append(player_team_mmr)

        if enemy_team_mmrs:
            enemy_mmr = sum(enemy_team_mmrs) / len(enemy_team_mmrs)

    # Retourner les valeurs même si une seule est disponible
    return (team_mmr, enemy_mmr)


# IDs GameVariantCategory pour Firefight — VALIDÉS sur JSON API réels.
# 42 = Firefight (confirmé sur match 8c12fd58, Gruntpocalypse:Heroic KOTH, avr 2025).
# Les anciens IDs {9, 24} ont été retirés car hypothétiques et causaient des
# faux positifs (Fiesta, BTB Flood Gulch marqués Firefight).
_FIREFIGHT_CATEGORY_IDS: frozenset[int] = frozenset(
    {
        41,  # Firefight (Battle of the Academy, confirmé match edc5daf6 Nov 2025)
        42,  # Gruntpocalypse / Firefight KOTH (confirmé match 8c12fd58 Apr 2025)
    }
)


def _is_firefight_match(match_info: dict[str, Any]) -> bool:
    """Retourne True si le match est un mode Firefight/PvE.

    Vérifie dans l'ordre :
    1. GameVariantCategory (int, le plus fiable si l'ID est confirmé)
    2. UgcGameVariant.PublicName (nom du mode de jeu)
    3. Playlist.PublicName (nom de la playlist)

    Args:
        match_info: Dict MatchInfo du JSON API (ou dict racine).

    Returns:
        True si détecté comme match Firefight/PvE.
    """
    # 1. GameVariantCategory (le plus fiable)
    category = match_info.get("GameVariantCategory")
    if isinstance(category, int) and category in _FIREFIGHT_CATEGORY_IDS:
        return True

    # 2. UgcGameVariant.PublicName (rétrocompat avec l'ancienne implémentation)
    game_variant = match_info.get("UgcGameVariant", {})
    if isinstance(game_variant, dict):
        name = game_variant.get("PublicName", "")
        if isinstance(name, str) and "firefight" in name.lower():
            return True

    # 3. Playlist.PublicName
    playlist_name = (match_info.get("Playlist") or {}).get("PublicName", "") or ""
    if not playlist_name:
        playlist_name = str(match_info.get("PlaylistName", "") or "")
    name_lower = playlist_name.lower()
    return "firefight" in name_lower or "baptême" in name_lower or "survive" in name_lower


def extract_game_variant_category(match_json: dict[str, Any]) -> int | None:
    """Extrait le GameVariantCategory depuis le JSON du match.

    Le GameVariantCategory est un entier qui identifie le type de mode
    (Slayer=6, CTF=15, Oddball=18, etc.).

    Args:
        match_json: JSON brut du match (MatchStats).

    Returns:
        GameVariantCategory (int) ou None si non disponible.
    """
    match_info = match_json.get("MatchInfo")
    if not isinstance(match_info, dict):
        return None

    # Chemin direct : MatchInfo.GameVariantCategory
    category = match_info.get("GameVariantCategory")
    if isinstance(category, int):
        return category

    # Fallback : UgcGameVariant.Category
    ugc = match_info.get("UgcGameVariant")
    if isinstance(ugc, dict):
        cat = ugc.get("Category") or ugc.get("GameVariantCategory")
        if isinstance(cat, int):
            return cat

    return None


def compute_teammates_signature(
    match_json: dict[str, Any],
    my_xuid: str,
    my_team_id: int | None,
) -> str | None:
    """Calcule la signature des coéquipiers pour un match.

    Args:
        match_json: JSON du match depuis l'API.
        my_xuid: XUID du joueur principal.
        my_team_id: ID de l'équipe du joueur.

    Returns:
        Signature (XUIDs triés séparés par virgule) ou None.
    """
    players = match_json.get("Players")
    if not players or my_team_id is None:
        return None

    # Extraire les XUIDs des coéquipiers (même équipe, excluant moi)
    teammate_xuids = []
    for player in players:
        if not isinstance(player, dict):
            continue

        xuid = _extract_xuid(player)
        team_id = _safe_int(player.get("LastTeamId"))

        if xuid and team_id == my_team_id and xuid != my_xuid:
            teammate_xuids.append(xuid)

    if not teammate_xuids:
        return None

    # Trier et joindre pour créer une signature stable
    teammate_xuids.sort()
    return ",".join(teammate_xuids)
