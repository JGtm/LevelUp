"""Protocol abstrait pour tout client API Halo Infinite.

Définit le contrat que doit respecter tout backend API (SPNKr, Grunt, etc.).
Les consommateurs (engine, backfill, UI) typent sur ce Protocol au lieu
du type concret ``SPNKrAPIClient``.

Architecture : Ports & Adapters (hexagonal).
"""

from __future__ import annotations

from typing import Any, Protocol, runtime_checkable

from src.data.sync.models import CareerRankData, MatchData, MatchHistoryItem


@runtime_checkable
class HaloAPIPort(Protocol):
    """Contrat structurel pour un client API Halo Infinite."""

    async def __aenter__(self) -> HaloAPIPort: ...

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: Any,
    ) -> None: ...

    # ------------------------------------------------------------------
    # Historique et stats de matchs
    # ------------------------------------------------------------------

    async def get_match_history(
        self,
        player: str,
        *,
        match_type: str = "matchmaking",
        start: int = 0,
        count: int = 25,
    ) -> list[MatchHistoryItem]: ...

    async def get_match_stats(self, match_id: str) -> dict[str, Any] | None: ...

    async def get_skill_stats(
        self,
        match_id: str,
        xuids: list[int],
    ) -> dict[str, Any] | None: ...

    async def get_highlight_events(self, match_id: str) -> list[Any]: ...

    async def get_match_data(
        self,
        match_id: str,
        xuids: list[int],
        *,
        with_skill: bool = True,
        with_highlight_events: bool = True,
    ) -> MatchData | None: ...

    # ------------------------------------------------------------------
    # Assets (maps, playlists, variants)
    # ------------------------------------------------------------------

    async def get_asset(
        self,
        asset_type: str,
        asset_id: str,
        version_id: str,
    ) -> dict[str, Any] | None: ...

    # ------------------------------------------------------------------
    # Profil joueur
    # ------------------------------------------------------------------

    async def get_career_rank_progression(self, xuid: str) -> CareerRankData | None: ...

    async def get_match_count(self, xuid: str) -> dict[str, int] | None: ...

    async def get_player_customization(self, xuid: str) -> dict[str, Any] | None: ...

    async def get_user_by_gamertag(self, gamertag: str) -> dict[str, Any] | None: ...

    async def get_users_by_id(self, xuids: list[str]) -> list[dict[str, Any]]: ...
