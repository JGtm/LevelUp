"""Transformation des données skill (MMR, CSR, StatPerformances).

Fonctions publiques :
- transform_skill_stats      : JSON skill + match_id + xuid → PlayerMatchStatsRow
- transform_all_skill_stats  : JSON skill + match_id → [SkillParticipantUpdate] (tous joueurs)
"""

from __future__ import annotations

from typing import Any

from src.data.sync.models import (
    PlayerMatchStatsRow,
    SkillParticipantUpdate,
)
from src.data.sync.transformers._helpers import (
    XUID_RE,
    _extract_mmr_from_skill,
    _safe_float,
    _safe_int,
)


def transform_skill_stats(  # noqa: C901, PLR0912
    skill_json: dict[str, Any],
    match_id: str,
    xuid: str,
) -> PlayerMatchStatsRow | None:
    """Transforme le JSON skill en PlayerMatchStatsRow.

    Args:
        skill_json: JSON de l'API skill.
        match_id: ID du match.
        xuid: XUID du joueur.

    Returns:
        PlayerMatchStatsRow ou None.
    """
    value = skill_json.get("Value")
    if not isinstance(value, list):
        return None

    # Trouver notre joueur
    for player in value:
        if not isinstance(player, dict):
            continue

        player_id = player.get("Id")
        if not isinstance(player_id, str):
            continue

        if xuid not in player_id:
            continue

        result = player.get("Result")
        if not isinstance(result, dict):
            continue

        team_id = _safe_int(result.get("TeamId"))

        # Utiliser _extract_mmr_from_skill() pour extraire team_mmr et enemy_mmr
        # Cela garantit la cohérence avec transform_match_stats()
        mmr_data = _extract_mmr_from_skill(skill_json, xuid, team_id)
        team_mmr = None
        enemy_mmr = None
        if mmr_data:
            team_mmr, enemy_mmr = mmr_data

        # Extraire expected/stddev (aligné legacy loaders.py et API : Kills, Deaths, Assists)
        stat_performances = result.get("StatPerformances")
        kills_expected = None
        kills_stddev = None
        deaths_expected = None
        deaths_stddev = None
        assists_expected = None
        assists_stddev = None

        def _perf_value(sp: dict | None, key: str, subkey: str) -> float | None:
            """Récupère StatPerformances[key][subkey] avec variantes de casse."""
            if not isinstance(sp, dict):
                return None
            for k, v in sp.items():
                if k and k.lower() == key.lower() and isinstance(v, dict):
                    return _safe_float(v.get(subkey))
            return None

        if isinstance(stat_performances, dict):
            # Accès direct (API retourne Kills, Deaths, Assists)
            kills_expected = _perf_value(stat_performances, "Kills", "Expected")
            kills_stddev = _perf_value(stat_performances, "Kills", "StdDev")
            deaths_expected = _perf_value(stat_performances, "Deaths", "Expected")
            deaths_stddev = _perf_value(stat_performances, "Deaths", "StdDev")
            assists_expected = _perf_value(stat_performances, "Assists", "Expected")
            assists_stddev = _perf_value(stat_performances, "Assists", "StdDev")

            # Fallback: itération si structure différente
            if kills_expected is None and deaths_expected is None and assists_expected is None:
                for stat_name, perf in stat_performances.items():
                    if not isinstance(perf, dict):
                        continue
                    expected = _safe_float(perf.get("Expected"))
                    stddev = _safe_float(perf.get("StdDev"))
                    if stat_name and stat_name.lower() == "kills":
                        kills_expected, kills_stddev = expected, stddev
                    elif stat_name and stat_name.lower() == "deaths":
                        deaths_expected, deaths_stddev = expected, stddev
                    elif stat_name and stat_name.lower() == "assists":
                        assists_expected, assists_stddev = expected, stddev

        return PlayerMatchStatsRow(
            match_id=match_id,
            xuid=xuid,
            team_id=team_id,
            team_mmr=team_mmr,
            enemy_mmr=enemy_mmr,
            kills_expected=kills_expected,
            kills_stddev=kills_stddev,
            deaths_expected=deaths_expected,
            deaths_stddev=deaths_stddev,
            assists_expected=assists_expected,
            assists_stddev=assists_stddev,
        )

    return None


def transform_all_skill_stats(  # noqa: C901, PLR0912, PLR0915
    skill_json: dict[str, Any],
    match_id: str,
) -> list[SkillParticipantUpdate]:
    """Extrait les données skill de TOUS les joueurs (pas juste le joueur courant).

    Pipeline v5.1 :
        API Skill JSON → transform_all_skill_stats() → [SkillParticipantUpdate]
        → engine.py : UPDATE shared.match_participants (COALESCE)

    Corrigé v5.1 : enemy_mmr était ignoré (bug `team_mmr, _ = mmr_data`).
    Désormais `team_mmr, enemy_mmr = mmr_data` extrait les deux MMR.

    ⚠️ Limitation API Halo Infinite :
        StatPerformances ne fournit Expected/StdDev que pour Kills et Deaths.
        Les données Assists (assists_expected, assists_stddev) ne sont jamais
        retournées par l'API et restent donc NULL en base.

    Args:
        skill_json: JSON de l'API skill.
        match_id: ID du match.

    Returns:
        Liste de SkillParticipantUpdate pour tous les joueurs.
    """
    results: list[SkillParticipantUpdate] = []

    value = skill_json.get("Value") or skill_json.get("value") or []
    if not isinstance(value, list):
        return results

    for player in value:
        if not isinstance(player, dict):
            continue

        # Extraire le XUID du joueur
        player_id = player.get("Id") or player.get("id")
        if not isinstance(player_id, str):
            continue

        # Extraire le XUID (format: xuid(1234567890123456))
        xuid = None
        m = XUID_RE.search(player_id)
        if m:
            xuid = m.group(1)
        if not xuid:
            continue

        result = player.get("Result") or player.get("result")
        if not isinstance(result, dict):
            continue

        team_id = _safe_int(result.get("TeamId") or result.get("teamId"))

        # Extraire team_mmr et enemy_mmr via la fonction existante
        mmr_data = _extract_mmr_from_skill(skill_json, xuid, team_id)
        team_mmr = None
        enemy_mmr = None
        if mmr_data:
            team_mmr, enemy_mmr = mmr_data

        # Extraire StatPerformances
        stat_performances = result.get("StatPerformances") or result.get("statPerformances")
        kills_expected = None
        kills_stddev = None
        deaths_expected = None
        deaths_stddev = None
        assists_expected = None
        assists_stddev = None

        def _perf_value(sp: dict | None, key: str, subkey: str) -> float | None:
            """Récupère StatPerformances[key][subkey] avec variantes de casse."""
            if not isinstance(sp, dict):
                return None
            for k, v in sp.items():
                if k and k.lower() == key.lower() and isinstance(v, dict):
                    return _safe_float(v.get(subkey))
            return None

        if isinstance(stat_performances, dict):
            # Accès direct (API retourne Kills, Deaths, Assists)
            kills_expected = _perf_value(stat_performances, "Kills", "Expected")
            kills_stddev = _perf_value(stat_performances, "Kills", "StdDev")
            deaths_expected = _perf_value(stat_performances, "Deaths", "Expected")
            deaths_stddev = _perf_value(stat_performances, "Deaths", "StdDev")
            assists_expected = _perf_value(stat_performances, "Assists", "Expected")
            assists_stddev = _perf_value(stat_performances, "Assists", "StdDev")

            # Fallback: itération si structure différente
            if kills_expected is None and deaths_expected is None and assists_expected is None:
                for stat_name, perf in stat_performances.items():
                    if not isinstance(perf, dict):
                        continue
                    expected = _safe_float(perf.get("Expected"))
                    stddev = _safe_float(perf.get("StdDev"))
                    if stat_name and stat_name.lower() == "kills":
                        kills_expected, kills_stddev = expected, stddev
                    elif stat_name and stat_name.lower() == "deaths":
                        deaths_expected, deaths_stddev = expected, stddev
                    elif stat_name and stat_name.lower() == "assists":
                        assists_expected, assists_stddev = expected, stddev

        # ── CSR (Competitive Skill Rating) depuis RankRecap — matchs classés ──
        rank_recap = result.get("RankRecap") or result.get("rankRecap")
        pre_match_csr: float | None = None
        post_match_csr: float | None = None
        csr_tier: str | None = None
        csr_sub_tier: int | None = None
        if isinstance(rank_recap, dict):
            pre = rank_recap.get("PreMatchCsr") or rank_recap.get("preMatchCsr") or {}
            post = rank_recap.get("PostMatchCsr") or rank_recap.get("postMatchCsr") or {}
            if isinstance(pre, dict):
                pre_match_csr = _safe_float(pre.get("Value") or pre.get("value"))
            if isinstance(post, dict):
                post_match_csr = _safe_float(post.get("Value") or post.get("value"))
                csr_tier = post.get("Tier") or post.get("tier")
                if csr_tier is None and isinstance(pre, dict):
                    csr_tier = pre.get("Tier") or pre.get("tier")
                raw_sub = post.get("SubTier") or post.get("subTier")
                if raw_sub is None and isinstance(pre, dict):
                    raw_sub = pre.get("SubTier") or pre.get("subTier")
                csr_sub_tier = _safe_int(raw_sub)

        results.append(
            SkillParticipantUpdate(
                match_id=match_id,
                xuid=xuid,
                team_mmr=team_mmr,
                enemy_mmr=enemy_mmr,
                kills_expected=kills_expected,
                kills_stddev=kills_stddev,
                deaths_expected=deaths_expected,
                deaths_stddev=deaths_stddev,
                assists_expected=assists_expected,
                assists_stddev=assists_stddev,
                pre_match_csr=pre_match_csr,
                post_match_csr=post_match_csr,
                csr_tier=csr_tier,
                csr_sub_tier=csr_sub_tier,
            )
        )

    return results
