"""Échange access_token Microsoft → tokens Halo Infinite.

Ce module est SANS état (stateless) : aucune persistance, aucune logique de cache.
Reçoit un access_token Microsoft et retourne les tokens Halo prêts à l'emploi.

Chaîne d'échange :
    access_token (Microsoft)
    → User Token (XBL)
    → XSTS Token Xbox (audience Xbox)
    → XSTS Token Halo (audience Halo)
    → Spartan Token
    → Clearance Token
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)


async def exchange_access_token_for_halo(
    session: Any,
    access_token: str,
) -> tuple[str, str]:
    """Échange un access_token Microsoft contre des tokens Halo Infinite.

    Args:
        session: Session aiohttp active.
        access_token: Token d'accès Microsoft (obtenu via MSAL).

    Returns:
        Tuple ``(spartan_token, clearance_token)``.

    Raises:
        ImportError: Si SPNKr n'est pas installé.
        ValueError: Si un appel de la chaîne échoue.
    """
    try:
        from spnkr.auth.core import XSTS_V3_HALO_AUDIENCE
        from spnkr.auth.halo import request_clearance_token, request_spartan_token
        from spnkr.auth.xbox import request_user_token, request_xsts_token
    except ImportError as e:
        raise ImportError("SPNKr requis pour l'échange Halo. Installer : pip install spnkr") from e

    logger.debug("Échange access_token → tokens Halo (chaîne XBL/XSTS/Spartan/Clearance)")

    user_token = await request_user_token(session, access_token)
    halo_xsts_token = await request_xsts_token(session, user_token.token, XSTS_V3_HALO_AUDIENCE)
    spartan_token = await request_spartan_token(session, halo_xsts_token.token)
    clearance_token = await request_clearance_token(session, spartan_token.token)

    spartan = str(spartan_token.token)
    clearance = str(clearance_token.token)
    logger.debug(
        "Tokens Halo obtenus (spartan len=%d, clearance len=%d)", len(spartan), len(clearance)
    )
    return spartan, clearance


async def resolve_player_identity(
    spartan_token: str,
    clearance_token: str,
) -> tuple[str, str]:
    """Résout le gamertag et le XUID du joueur depuis ses tokens Halo.

    Appelle l'API Halo Waypoint (endpoint profil) pour obtenir l'identité
    du joueur authentifié.

    Args:
        spartan_token: Token Spartan (Halo Waypoint).
        clearance_token: Token Clearance (Halo Waypoint).

    Returns:
        Tuple ``(gamertag, xuid)``.

    Raises:
        ImportError: Si SPNKr n'est pas installé.
        ValueError: Si le profil ne peut pas être résolu.
    """
    try:
        import aiohttp
        from spnkr import HaloInfiniteClient
    except ImportError as e:
        raise ImportError("SPNKr requis : pip install spnkr") from e

    timeout = aiohttp.ClientTimeout(total=20.0)
    async with aiohttp.ClientSession(timeout=timeout) as session:
        client = HaloInfiniteClient(
            session=session,
            spartan_token=spartan_token,
            clearance_token=clearance_token,
            requests_per_second=5,
        )

        try:
            resp = await client.profile.get_current_user()
            data = await resp.json()
            xuid = str(data.get("xuid") or data.get("Xuid") or "").strip()
            gamertag = str(data.get("gamertag") or data.get("Gamertag") or "").strip()
            if xuid and gamertag:
                logger.debug("Identité résolue via get_current_user: %s / %s", gamertag, xuid)
                return gamertag, xuid
        except Exception as api_err:
            logger.debug("get_current_user échoué: %s", api_err)

        try:
            resp = await client.profile.get_current_player()
            data = await resp.json()
            xuid = str(data.get("xuid") or data.get("Xuid") or "").strip()
            gamertag = str(data.get("gamertag") or data.get("Gamertag") or "").strip()
            if xuid and gamertag:
                logger.debug("Identité résolue via get_current_player: %s / %s", gamertag, xuid)
                return gamertag, xuid
        except Exception as api_err2:
            logger.debug("get_current_player échoué: %s", api_err2)

    raise ValueError(
        "Impossible de résoudre le gamertag/XUID depuis l'API Halo. "
        "Tokens potentiellement invalides ou API indisponible."
    )
