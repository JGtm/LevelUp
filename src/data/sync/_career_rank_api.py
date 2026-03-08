"""Logique Career Rank (API economy, parsing, adornments).

Fonctions extraites de ``SPNKrAPIClient`` pour réduire le volume du module
principal ``api_client.py``.
"""

from __future__ import annotations

import logging
from dataclasses import replace
from typing import Any

from src.config import CAREER_RANK_MAX
from src.data.sync._tokens import Tokens, build_halo_headers, clean_xuid
from src.data.sync.models import CareerRankData

logger = logging.getLogger(__name__)


async def get_career_rank_progression(
    session: Any,
    tokens: Tokens,
    client: Any,
    xuid: str,
) -> CareerRankData | None:
    """Récupère la progression du rang carrière d'un joueur.

    Args:
        session: aiohttp.ClientSession active.
        tokens: Tokens SPNKr.
        client: HaloInfiniteClient (pour gamecms).
        xuid: XUID du joueur (format numérique).

    Returns:
        CareerRankData ou None si indisponible.
    """
    from src.data.sync.api_client import request_with_retries

    xuid_clean = clean_xuid(xuid)

    async def _fetch() -> dict[str, Any] | None:
        url = (
            f"https://economy.svc.halowaypoint.com/hi/players/"
            f"xuid({xuid_clean})/rewardtracks/careerranks/careerrank1"
        )
        if session is None:
            raise RuntimeError("Session non initialisée")
        headers = build_halo_headers(tokens)
        async with session.get(url, headers=headers) as resp:
            if resp.status == 404:
                return None
            resp.raise_for_status()
            return await resp.json()

    try:
        json_data = await request_with_retries(_fetch)
        if json_data is None:
            return None

        career_data = parse_career_rank(xuid_clean, json_data)

        if career_data and not career_data.adornment_path:
            adornment_url = await resolve_adornment_url(client, career_data.current_rank)
            if adornment_url:
                career_data = replace(career_data, adornment_path=adornment_url)

        return career_data
    except Exception as e:
        logger.warning("Erreur get_career_rank_progression(%s): %s", xuid, e)
        return None


def parse_career_rank(xuid: str, data: dict[str, Any]) -> CareerRankData:
    """Parse les données brutes de career rank en CareerRankData.

    Note : l'API Halo retourne CurrentProgress.Rank = dernier rang *complété*
    (pas le rang affiché). Le rang affiché = API.Rank + 1, sauf au rang max.
    """
    current = data.get("CurrentProgress", {})
    rank = current.get("Rank", 0)
    partial_xp = current.get("PartialProgress", 0)
    is_max = rank >= CAREER_RANK_MAX

    # Le rang affiché est le suivant (celui en cours), sauf si on est au max
    display_rank = rank if is_max else rank + 1
    rank_info = get_rank_info(display_rank)

    # xp_for_next_rank : utiliser les vraies valeurs depuis metadata.duckdb
    xp_for_next = rank_info.get("xp_required", 0)
    try:
        from src.ui.career_ranks import get_rank_info as _get_meta

        meta = _get_meta(display_rank)
        if meta is not None:
            xp_for_next = meta.xp_required
    except Exception:
        pass

    return CareerRankData(
        xuid=xuid,
        current_rank=display_rank,
        current_rank_name=rank_info.get("name", f"Rank {display_rank}"),
        current_rank_tier=rank_info.get("tier", ""),
        current_xp=partial_xp,
        xp_for_next_rank=xp_for_next,
        xp_total=compute_total_xp(display_rank, partial_xp),
        is_max_rank=is_max,
        adornment_path=None,
        raw_json=data,
    )


async def resolve_adornment_url(client: Any, rank: int) -> str | None:
    """Résout l'URL d'adornment via gamecms pour un rang donné."""
    try:
        if client is None:
            return None
        gamecms = getattr(client, "gamecms_hacs", None)
        if gamecms is None:
            return None

        career_track_resp = await gamecms.get_career_reward_track()
        if hasattr(career_track_resp, "data"):
            career_track = career_track_resp.data
        elif hasattr(career_track_resp, "parse"):
            career_track = await career_track_resp.parse()
        else:
            career_track = career_track_resp

        if career_track is None:
            return None

        ranks_list = getattr(career_track, "ranks", None)
        if not ranks_list:
            return None

        display_rank = rank if rank == CAREER_RANK_MAX else rank + 1
        for rank_obj in ranks_list:
            r = getattr(rank_obj, "rank", None)
            if r == display_rank:
                adornment_icon = getattr(rank_obj, "rank_adornment_icon", None)
                if adornment_icon:
                    host = "https://gamecms-hacs.svc.halowaypoint.com"
                    adorn_path = str(adornment_icon).lstrip("/")
                    return f"{host}/hi/images/file/{adorn_path}"
                break
        return None
    except Exception as e:
        logger.debug("Résolution adornment gamecms échouée: %s", e)
        return None


def get_rank_info(rank: int) -> dict[str, Any]:
    """Retourne les infos d'un rang carrière (nom, tier, XP requis)."""
    base_tiers = ["Bronze", "Silver", "Gold", "Platinum", "Diamond", "Onyx"]
    hero_ranks = {271: "Hero", 272: "Hero Legend"}

    if rank in hero_ranks:
        return {"name": hero_ranks[rank], "tier": "Hero", "xp_required": 0}

    if rank <= 30:
        tier_idx = (rank - 1) // 5
        level = ((rank - 1) % 5) + 1
        tier = base_tiers[tier_idx] if tier_idx < len(base_tiers) else "Unknown"
        return {
            "name": f"{tier} {level}",
            "tier": tier,
            "xp_required": 10000 + (rank * 500),
        }

    cycle = (rank - 31) // 30
    pos_in_cycle = (rank - 31) % 30
    tier_idx = pos_in_cycle // 5
    level = (pos_in_cycle % 5) + 1
    tier = base_tiers[tier_idx] if tier_idx < len(base_tiers) else "Unknown"
    grade = ["I", "II", "III", "IV", "V", "VI", "VII", "VIII"][min(cycle, 7)]

    return {
        "name": f"{tier} {level} ({grade})",
        "tier": tier,
        "xp_required": 15000 + (cycle * 2000) + (rank * 100),
    }


def compute_total_xp(rank: int, partial_xp: int) -> int:
    """Calcule l'XP total basé sur le rang et l'XP partiel.

    Utilise les vraies valeurs depuis metadata.duckdb (table career_ranks)
    pour une précision maximale. Fallback sur la formule approx si metadata
    indisponible (ex: premier lancement, DB absente).
    """
    try:
        from src.ui.career_ranks import get_rank_info as _get_meta

        base_xp = 0
        for r in range(1, rank):
            info = _get_meta(r)
            base_xp += info.xp_required if info is not None else 0
        return base_xp + partial_xp
    except Exception:
        # Fallback si metadata.duckdb est indisponible
        base_xp = 0
        for r in range(1, rank):
            info = get_rank_info(r)
            base_xp += info.get("xp_required", 10000)
        return base_xp + partial_xp
