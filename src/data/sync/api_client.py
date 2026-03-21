"""Client API SPNKr asynchrone.

Ce module encapsule HaloInfiniteClient de SPNKr avec :
- Gestion automatique des tokens (env ou refresh OAuth)
- Rate limiting configurable
- Retry avec backoff exponentiel
- Support des highlight events via spnkr.film

Usage:
    async with SPNKrAPIClient() as client:
        history = await client.get_match_history("SpartanB")
        for item in history:
            data = await client.get_match_data(item.match_id, xuids)
"""

from __future__ import annotations

import asyncio
import logging
from collections.abc import Callable
from typing import Any

from src.data.sync._tokens import (
    Tokens,
    build_halo_headers,
    clean_xuid,
    get_player_token_env_key,
    get_tokens_for_player,
    get_tokens_from_env,  # LEGACY — utiliser src.auth.provider.get_halo_tokens
)
from src.data.sync.models import CareerRankData, MatchData, MatchHistoryItem
from src.utils.env import normalize_gamertag_for_env as _normalize_gamertag_for_env  # noqa: F401

logger = logging.getLogger(__name__)

# Re-exports pour compatibilité descendante  (ne pas supprimer)
__all__ = [
    "SPNKrAPIClient",
    "Tokens",
    "enrich_match_info_with_assets",
    "get_player_token_env_key",
    "get_tokens_for_player",
    "get_tokens_from_env",
    "request_with_retries",
]


# =============================================================================
# Retry helper
# =============================================================================


async def request_with_retries(
    coro_factory: Callable[[], Any],
    *,
    tries: int = 4,
    base_sleep: float = 0.8,
) -> Any:
    """Exécute une coroutine avec retry et backoff exponentiel.

    Args:
        coro_factory: Factory qui retourne la coroutine à exécuter.
        tries: Nombre maximum de tentatives.
        base_sleep: Délai de base entre les tentatives (secondes).

    Returns:
        Résultat de la coroutine.

    Raises:
        Exception: Dernière exception après épuisement des tentatives.
    """
    last_err: Exception | None = None

    for i in range(tries):
        try:
            return await coro_factory()
        except Exception as e:
            # Auth invalide: inutile de retry
            try:
                from aiohttp.client_exceptions import ClientResponseError

                if isinstance(e, ClientResponseError):
                    if e.status in (401, 403):
                        raise ValueError(
                            "Requête non autorisée (401/403). "
                            "Tokens probablement invalides/expirés."
                        ) from e
                    # Assets manquants: pas de retry
                    if e.status in (400, 404, 410):
                        raise
            except ImportError:
                pass

            last_err = e
            await asyncio.sleep(base_sleep * (2**i))

    assert last_err is not None
    raise last_err


# =============================================================================
# SPNKrAPIClient
# =============================================================================


class SPNKrAPIClient:
    """Client API SPNKr asynchrone.

    Encapsule HaloInfiniteClient avec :
    - Gestion automatique des tokens
    - Rate limiting configurable
    - Support des highlight events

    Usage:
        async with SPNKrAPIClient() as client:
            history = await client.get_match_history("SpartanB")
    """

    def __init__(
        self,
        *,
        tokens: Tokens | None = None,
        requests_per_second: int = 5,
    ) -> None:
        """
        Args:
            tokens: Tokens pré-fournis (sinon récupérés depuis env).
            requests_per_second: Rate limiting par service.
        """
        self._tokens = tokens
        self._requests_per_second = requests_per_second
        self._session = None
        self._client = None
        self._film_mod = None

    async def __aenter__(self) -> SPNKrAPIClient:
        """Initialise la session et le client."""
        if self._tokens is None:
            self._tokens = await get_tokens_from_env()

        try:
            from aiohttp import ClientSession, ClientTimeout
            from spnkr import HaloInfiniteClient
        except ImportError as e:
            raise ImportError(
                "Dépendances SPNKr manquantes. Installer: pip install spnkr aiohttp"
            ) from e

        self._session = ClientSession(timeout=ClientTimeout(total=45))
        self._client = HaloInfiniteClient(
            session=self._session,
            spartan_token=self._tokens.spartan_token,
            clearance_token=self._tokens.clearance_token,
            requests_per_second=self._requests_per_second,
        )

        # Charger le module film pour les highlight events
        try:
            from spnkr import film as film_mod

            self._film_mod = film_mod
        except ImportError:
            self._film_mod = None

        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Ferme la session."""
        if self._session:
            await self._session.close()
            self._session = None
        self._client = None

    @property
    def client(self) -> Any:
        """Retourne le HaloInfiniteClient sous-jacent."""
        if self._client is None:
            raise RuntimeError("Client non initialisé. Utiliser 'async with'.")
        return self._client

    async def get_match_history(
        self,
        player: str,
        *,
        match_type: str = "matchmaking",
        start: int = 0,
        count: int = 25,
    ) -> list[MatchHistoryItem]:
        """Récupère l'historique des matchs d'un joueur.

        Args:
            player: Gamertag ou XUID du joueur.
            match_type: Type de matchs (all, matchmaking, custom, local).
            start: Offset dans l'historique (0 = plus récent).
            count: Nombre de matchs (max 25).

        Returns:
            Liste de MatchHistoryItem.
        """

        async def _fetch():
            resp = await self.client.stats.get_match_history(
                player, start=start, count=count, match_type=match_type
            )
            return await resp.parse()

        history = await request_with_retries(_fetch)

        if not hasattr(history, "results") or not history.results:
            return []

        items = []
        for r in history.results:
            items.append(
                MatchHistoryItem(
                    match_id=str(r.match_id),
                    start_time=str(r.match_info.start_time) if hasattr(r, "match_info") else "",
                    match_type=match_type,
                )
            )

        return items

    async def get_match_stats(self, match_id: str) -> dict[str, Any] | None:
        """Récupère les stats d'un match."""

        async def _fetch():
            resp = await self.client.stats.get_match_stats(match_id)
            return await resp.json()

        try:
            result = await request_with_retries(_fetch)
            return result if isinstance(result, dict) else None
        except Exception as e:
            logger.warning("Erreur get_match_stats(%s): %s", match_id, e)
            return None

    async def get_skill_stats(
        self,
        match_id: str,
        xuids: list[int],
    ) -> dict[str, Any] | None:
        """Récupère les stats skill (MMR) d'un match."""
        if not xuids:
            return None

        async def _fetch():
            resp = await self.client.skill.get_match_skill(match_id, xuids)
            return await resp.json()

        try:
            result = await request_with_retries(_fetch)
            return result if isinstance(result, dict) else None
        except Exception:
            return None

    async def get_highlight_events(self, match_id: str) -> list[Any]:
        """Récupère les highlight events (film) d'un match."""
        if self._film_mod is None:
            return []

        async def _fetch():
            return await self._film_mod.read_highlight_events(self.client, match_id=match_id)

        try:
            events = await request_with_retries(_fetch)
            return events if events else []
        except Exception:
            return []

    async def get_match_data(
        self,
        match_id: str,
        xuids: list[int],
        *,
        with_skill: bool = True,
        with_highlight_events: bool = True,
    ) -> MatchData | None:
        """Récupère les données complètes d'un match (stats + skill + events)."""
        stats_json = await self.get_match_stats(match_id)
        if stats_json is None:
            return None

        skill_task = self.get_skill_stats(match_id, xuids) if with_skill else asyncio.sleep(0)
        events_task = (
            self.get_highlight_events(match_id) if with_highlight_events else asyncio.sleep(0)
        )

        skill_result, events_result = await asyncio.gather(
            skill_task,
            events_task,
            return_exceptions=True,
        )

        skill_json = skill_result if isinstance(skill_result, dict) else None
        highlight_events = events_result if isinstance(events_result, list) else []

        return MatchData(
            match_id=match_id,
            stats_json=stats_json,
            skill_json=skill_json,
            highlight_events=highlight_events,
        )

    async def get_asset(
        self,
        asset_type: str,
        asset_id: str,
        version_id: str,
    ) -> dict[str, Any] | None:
        """Récupère un asset (map, playlist, variant)."""

        async def _fetch():
            if asset_type == "Maps":
                resp = await self.client.discovery_ugc.get_map(asset_id, version_id)
            elif asset_type == "Playlists":
                resp = await self.client.discovery_ugc.get_playlist(asset_id, version_id)
            elif asset_type == "PlaylistMapModePairs":
                resp = await self.client.discovery_ugc.get_map_mode_pair(asset_id, version_id)
            elif asset_type == "GameVariants":
                resp = await self.client.discovery_ugc.get_ugc_game_variant(asset_id, version_id)
            else:
                return None
            return await resp.json()

        try:
            result = await request_with_retries(_fetch)
            return result if isinstance(result, dict) else None
        except Exception:
            return None

    # =========================================================================
    # Career Rank (délégation vers _career_rank_api)
    # =========================================================================

    async def get_career_rank_progression(self, xuid: str) -> CareerRankData | None:
        """Récupère la progression du rang carrière d'un joueur."""
        from src.data.sync._career_rank_api import get_career_rank_progression

        return await get_career_rank_progression(self._session, self._tokens, self._client, xuid)

    # =========================================================================
    # Endpoints supplémentaires
    # =========================================================================

    async def get_match_count(self, xuid: str) -> dict[str, int] | None:
        """Récupère le nombre total de matchs d'un joueur."""
        xuid_clean = clean_xuid(xuid)

        async def _fetch():
            url = (
                f"https://halostats.svc.halowaypoint.com/hi/players/"
                f"xuid({xuid_clean})/matches/count"
            )
            if self._session is None:
                raise RuntimeError("Session non initialisée")
            headers = build_halo_headers(self._tokens)
            async with self._session.get(url, headers=headers) as resp:
                if resp.status == 404:
                    return None
                resp.raise_for_status()
                return await resp.json()

        try:
            json_data = await request_with_retries(_fetch)
            if json_data is None:
                return None
            return {
                "matchmaking": json_data.get("MatchmadeMatchesPlayedCount", 0),
                "custom": json_data.get("CustomMatchesPlayedCount", 0),
                "local": json_data.get("LocalMatchesPlayedCount", 0),
                "total": json_data.get("MatchesPlayedCount", 0),
            }
        except Exception as e:
            logger.warning("Erreur get_match_count(%s): %s", xuid, e)
            return None

    async def get_player_customization(self, xuid: str) -> dict[str, Any] | None:
        """Récupère les données de personnalisation Spartan."""
        xuid_clean = clean_xuid(xuid)

        async def _fetch():
            resp = await self.client.economy.get_player_customization(f"xuid({xuid_clean})")
            return await resp.json()

        try:
            result = await request_with_retries(_fetch)
            return result if isinstance(result, dict) else None
        except Exception as e:
            logger.warning("Erreur get_player_customization(%s): %s", xuid, e)
            return None

    async def get_user_by_gamertag(self, gamertag: str) -> dict[str, Any] | None:
        """Résout un gamertag vers un profil Xbox (xuid, gamertag canonique)."""
        gt = str(gamertag or "").strip()
        if not gt:
            return None
        try:
            resp = await self.client.profile.get_user_by_gamertag(gt)
            if hasattr(resp, "data"):
                user = resp.data
            else:
                user = await resp.parse()
            if user is None:
                return None
            return {
                "xuid": str(getattr(user, "xuid", "") or "").strip(),
                "gamertag": str(getattr(user, "gamertag", "") or "").strip(),
            }
        except Exception as e:
            logger.warning("Erreur get_user_by_gamertag(%s): %s", gamertag, e)
            return None

    async def get_users_by_id(self, xuids: list[str]) -> list[dict[str, Any]]:
        """Résout une liste de XUIDs vers des profils Xbox."""
        if not xuids:
            return []
        try:
            resp = await self.client.profile.get_users_by_id(xuids)
            users = resp.data if hasattr(resp, "data") else await resp.parse()
            return [
                {
                    "xuid": str(getattr(u, "xuid", "") or "").strip(),
                    "gamertag": str(getattr(u, "gamertag", "") or "").strip(),
                }
                for u in (users or [])
            ]
        except Exception as e:
            logger.warning("Erreur get_users_by_id(%s XUIDs): %s", len(xuids), e)
            return []

    # =========================================================================
    # Film (weapon extraction)
    # =========================================================================

    async def get_film_by_match_id(self, match_id: str) -> Any | None:
        """Récupère les métadonnées film d'un match (chunks, blob prefix)."""

        async def _fetch():
            resp = await self.client.discovery_ugc.get_film_by_match_id(match_id)
            return await resp.parse()

        try:
            return await request_with_retries(_fetch)
        except Exception as e:
            logger.warning("Erreur get_film_by_match_id(%s): %s", match_id[:8], e)
            return None

    async def download_film_chunk(self, url: str) -> bytes | None:
        """Télécharge et décompresse un blob Azure (chunk film)."""
        import zlib

        if self._session is None:
            raise RuntimeError("Session non initialisée")
        try:
            resp = await self._session.get(url)
            resp.raise_for_status()
            raw = await resp.read()
            return zlib.decompress(raw)
        except Exception as e:
            logger.warning("Erreur download_film_chunk: %s", e)
            return None

    async def get_spartan_token_xuid(self) -> str | None:
        """Récupère le XUID du joueur authentifié depuis le token Spartan."""
        if self._tokens is None:
            return None
        try:
            import base64
            import json as json_module

            parts = self._tokens.spartan_token.split(".")
            if len(parts) >= 2:
                payload_b64 = parts[1]
                padding = 4 - len(payload_b64) % 4
                if padding != 4:
                    payload_b64 += "=" * padding
                payload = base64.urlsafe_b64decode(payload_b64)
                data = json_module.loads(payload)
                xuid = data.get("xid") or data.get("xuid") or data.get("sub")
                if xuid:
                    return str(xuid)
        except Exception:
            logger.debug("Échec extraction XUID du token JWT", exc_info=True)
        return None


async def enrich_match_info_with_assets(
    client: SPNKrAPIClient,
    stats_json: dict[str, Any],
) -> None:
    """Enrichit MatchInfo avec les PublicName depuis Discovery UGC (in-place)."""
    match_info = stats_json.get("MatchInfo")
    if not isinstance(match_info, dict):
        return

    asset_keys = [
        ("Playlist", "Playlists"),
        ("MapVariant", "Maps"),
        ("PlaylistMapModePair", "PlaylistMapModePairs"),
        ("UgcGameVariant", "GameVariants"),
    ]

    for json_key, api_type in asset_keys:
        ref = match_info.get(json_key)
        if not isinstance(ref, dict):
            continue
        aid = ref.get("AssetId")
        vid = ref.get("VersionId")
        if not aid or not vid:
            continue

        try:
            asset = await client.get_asset(api_type, aid, vid)
            if isinstance(asset, dict):
                name = asset.get("PublicName")
                if isinstance(name, str) and name.strip():
                    ref["PublicName"] = name.strip()
        except Exception as e:
            logger.debug("Asset %s %s: %s", api_type, aid, e)
