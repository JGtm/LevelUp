"""Additions to spnkr/services/economy.py — challenges, reward tracks, Spartan Points.

These four methods extend the existing EconomyService.
In the actual PR they are appended directly into EconomyService in economy.py.
"""

from typing import Literal

from spnkr.responses import JsonResponse
from spnkr.services.base import BaseService
from spnkr.xuid import wrap_xuid

# Import the new models defined in this PR.
# In the actual PR: from spnkr.models.economy import (...)
from spnkr_pr.models.economy_additions import (
    PlayerChallenges,
    PlayerRewardTrackProgression,
    PlayerRewardTracksSummary,
    SpartanPoints,
)

_HOST = "https://economy.svc.halowaypoint.com:443"


class EconomyServiceExtension(BaseService):
    """Extension of EconomyService adding progression and challenge endpoints.

    In the actual PR these methods are merged directly into EconomyService.
    """

    async def get_player_challenges(
        self,
        xuid: str | int,
    ) -> JsonResponse[PlayerChallenges]:
        """Get the active challenge deck for a player.

        Note: Only works when ``xuid`` matches the authenticated player.

        Args:
            xuid: Xbox Live ID of the authenticated player.

        Returns:
            The player's active challenges grouped by category (daily, weekly…).
        """
        url = f"{_HOST}/hi/players/{wrap_xuid(xuid)}/challenges"
        resp = await self._get(url)
        return JsonResponse(resp, lambda data: PlayerChallenges(**data))

    async def get_player_reward_tracks(
        self,
        xuid: str | int,
    ) -> JsonResponse[PlayerRewardTracksSummary]:
        """Get a summary of all reward track progressions for a player.

        Returns progress on all operations (battlepass), events, and career rank.

        Note: Only works when ``xuid`` matches the authenticated player.

        Args:
            xuid: Xbox Live ID of the authenticated player.

        Returns:
            Summary of operation, event, and career rank progressions.
        """
        url = f"{_HOST}/hi/players/{wrap_xuid(xuid)}/rewardtracks"
        resp = await self._get(url)
        return JsonResponse(resp, lambda data: PlayerRewardTracksSummary(**data))

    async def get_player_reward_track(
        self,
        xuid: str | int,
        track_type: Literal["operations", "events", "career_rank"],
        track_id: str,
    ) -> JsonResponse[PlayerRewardTrackProgression]:
        """Get a player's progress on a specific reward track.

        ``current_progress`` semantics differ by track type:
        - ``"operations"`` / ``"events"``: raw XP accumulated.
        - ``"career_rank"``: integer rank number (1–272).

        Note: Only works when ``xuid`` matches the authenticated player.

        Args:
            xuid: Xbox Live ID of the authenticated player.
            track_type: One of ``"operations"``, ``"events"``, ``"career_rank"``.
            track_id: The reward track identifier, e.g. ``"Season6Operations-1"``,
                ``"WinterContingency1"``, or ``"CareerRank1"``.

        Returns:
            The player's progression on the specified reward track.
        """
        url = f"{_HOST}/hi/players/{wrap_xuid(xuid)}/rewardtracks/{track_type}/{track_id}"
        resp = await self._get(url)
        return JsonResponse(resp, lambda data: PlayerRewardTrackProgression(**data))

    async def get_player_spartan_points(
        self,
        xuid: str | int,
    ) -> JsonResponse[SpartanPoints]:
        """Get the Spartan Points balance for a player.

        Note: Only works when ``xuid`` matches the authenticated player.

        Args:
            xuid: Xbox Live ID of the authenticated player.

        Returns:
            The player's current Spartan Points balance.
        """
        url = f"{_HOST}/hi/players/{wrap_xuid(xuid)}/currency/spartan-points"
        resp = await self._get(url)
        return JsonResponse(resp, lambda data: SpartanPoints(**data))
