"""Helpers internes pour la progression Career Rank (SPNKr API)."""

from __future__ import annotations

from src.config import CAREER_RANK_MAX
from src.ui.career_ranks import format_career_rank_label_fr


async def _fetch_career_progress(session, xu: str, st_in: str, ct_in: str) -> dict | None:
    """Appelle l'API Economy pour récupérer la progression Career Rank du joueur."""
    economy_host = "https://economy.svc.halowaypoint.com"
    headers: dict[str, str] = {"Accept": "application/json"}
    if st_in:
        headers["X-343-Authorization-Spartan"] = st_in
    if ct_in:
        headers["343-Clearance"] = ct_in

    # Format 1 (den.dev): GET /hi/players/xuid({XUID})/rewardtracks/careerranks/careerrank1
    try:
        career_url = f"{economy_host}/hi/players/xuid({xu})/rewardtracks/careerranks/careerrank1"
        async with session.get(career_url, headers=headers) as resp:
            if resp.status == 200:
                data = await resp.json()
                return data.get("CurrentProgress", {})
    except Exception:
        pass

    # Format 2 (fallback Grunt): POST /hi/rewardtracks/careerRank1
    try:
        career_url = f"{economy_host}/hi/rewardtracks/careerRank1"
        body = {"Users": [f"xuid({xu})"]}
        async with session.post(career_url, headers=headers, json=body) as resp:
            if resp.status == 200:
                data = await resp.json()
                reward_tracks = data.get("RewardTracks", [])
                if not reward_tracks:
                    return None
                result = reward_tracks[0].get("Result", {})
                return result.get("CurrentProgress", {})
    except Exception:
        pass

    return None


def _build_career_rank_result(
    ranks_list, current_rank: int, partial_xp: int
) -> tuple[str | None, str | None, str | None, str | None]:
    """Construit le label, sous-titre et URLs d'icône pour un rang donné."""
    display_rank = current_rank if current_rank == CAREER_RANK_MAX else current_rank + 1
    current_stage = next((r for r in ranks_list if getattr(r, "rank", None) == display_rank), None)
    if current_stage is None:
        return f"Rang de carrière {display_rank}", f"XP {partial_xp}", None, None

    tier_type = getattr(current_stage, "tier_type", None)
    rank_title_obj = getattr(current_stage, "rank_title", None)
    rank_tier_obj = getattr(current_stage, "rank_tier", None)
    xp_required = getattr(current_stage, "xp_required_for_rank", None)
    rank_large_icon = getattr(current_stage, "rank_large_icon", None)
    rank_adornment_icon = getattr(current_stage, "rank_adornment_icon", None)
    rank_title = getattr(rank_title_obj, "value", None) if rank_title_obj else None
    rank_tier = getattr(rank_tier_obj, "value", None) if rank_tier_obj else None

    r_subtitle = f"XP {partial_xp}/{xp_required}" if xp_required else f"XP {partial_xp}"
    if current_rank == CAREER_RANK_MAX:
        r_label = format_career_rank_label_fr(tier=None, title=(rank_title or "Hero"), grade=None)
    else:
        r_label = format_career_rank_label_fr(tier=tier_type, title=rank_title, grade=rank_tier)
        if not r_label:
            r_label = f"Rang {display_rank}"

    host = "https://gamecms-hacs.svc.halowaypoint.com"
    r_icon = (
        f"{host}/hi/images/file/{str(rank_large_icon).lstrip('/')}" if rank_large_icon else None
    )
    r_adornment = (
        f"{host}/hi/images/file/{str(rank_adornment_icon).lstrip('/')}"
        if rank_adornment_icon
        else None
    )
    return r_label, r_subtitle, r_icon, r_adornment
