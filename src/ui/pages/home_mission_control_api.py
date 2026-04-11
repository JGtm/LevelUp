"""Données live pour l'accueil V7 — battlepass et défis actifs via SPNKr ext.

Ce module est responsable des appels API asynchrones (avec cache session)
et des dataclasses légères utilisées par home_mission_control.py.
"""

from __future__ import annotations

import asyncio
import logging
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

_CACHE_TTL_S: int = 300  # 5 minutes


# =============================================================================
# Dataclasses
# =============================================================================


@dataclass(frozen=True)
class HomeBattlepassInfo:
    """Info de progression battlepass pour l'accueil."""

    track_name: str
    current_progress: int
    is_owned: bool
    track_image_bytes: bytes | None = None


@dataclass(frozen=True)
class HomeChallengeSummary:
    """Résumé des défis actifs pour l'accueil."""

    total: int
    completed: int
    xp_available: int
    next_expiry: str | None


# =============================================================================
# Builders (API data → dataclasses)
# =============================================================================


def _build_battlepass_info(tracks_data: Any) -> tuple[HomeBattlepassInfo | None, str | None]:
    """Construit HomeBattlepassInfo + reward_track_path depuis PlayerRewardTracksSummary.

    Retourne ``(None, None)`` si les données sont absentes ou invalides.
    """
    try:
        from spnkr_pr.models.economy_additions import PlayerRewardTracksSummary
    except ImportError:
        return None, None

    if not isinstance(tracks_data, PlayerRewardTracksSummary):
        return None, None

    ops = tracks_data.operation_reward_tracks
    if not ops:
        return None, None

    # Opération la plus récemment mise à jour = saison/opération en cours
    current_op = max(ops, key=lambda t: t.date_last_updated.value)
    raw_path = current_op.reward_track_path
    #  "Operations/Season6Operations-1.json" → "Season6Operations-1"
    track_name = raw_path.split("/")[-1].replace(".json", "")

    return HomeBattlepassInfo(
        track_name=track_name,
        current_progress=current_op.current_progress,
        is_owned=current_op.is_owned,
    ), current_op.reward_track_path


def _build_challenge_summary(challenges_data: Any) -> HomeChallengeSummary | None:
    """Construit HomeChallengeSummary depuis un PlayerChallenges parsé."""
    try:
        from spnkr_pr.models.economy_additions import PlayerChallenges
    except ImportError:
        return None

    if not isinstance(challenges_data, PlayerChallenges):
        return None

    all_challenges = [ch for cat in challenges_data.categories for ch in cat.challenges]
    if not all_challenges:
        return HomeChallengeSummary(total=0, completed=0, xp_available=0, next_expiry=None)

    completed = sum(1 for ch in all_challenges if ch.is_completed)
    xp_available = sum(ch.xp_reward for ch in all_challenges if not ch.is_completed)

    expiring = [
        ch for ch in all_challenges if not ch.is_completed and ch.expiration_time is not None
    ]
    next_expiry: str | None = None
    if expiring:
        soonest = min(expiring, key=lambda c: c.expiration_time.value)
        next_expiry = soonest.expiration_time.value.strftime("%d/%m %H:%M")

    return HomeChallengeSummary(
        total=len(all_challenges),
        completed=completed,
        xp_available=xp_available,
        next_expiry=next_expiry,
    )


# =============================================================================
# Fetch asynchrone
# =============================================================================


async def _resolve_tokens(
    db_path: Path,
    gamertag: str | None,
) -> Any | None:
    """Résout les tokens Halo : MSAL silent d'abord, puis refresh token OAuth env."""
    # 1. Tentative MSAL (cache process + silent)
    try:
        from src.auth.provider import get_halo_tokens_or_raise

        return await get_halo_tokens_or_raise(db_path)
    except ImportError as exc:
        logger.debug("home_api: auth module indisponible: %s", exc)
    except Exception:
        pass  # On tente le fallback refresh token ci-dessous

    # 2. Fallback : refresh token OAuth par joueur (SPNKR_OAUTH_REFRESH_TOKEN_<GT>)
    if gamertag:
        try:
            from src.data.sync._tokens import get_tokens_for_player

            player_tokens = await get_tokens_for_player(gamertag)
            if player_tokens is not None:
                logger.debug("home_api: tokens obtenus via refresh token env (%s)", gamertag)
                return player_tokens
        except Exception as exc:
            logger.debug("home_api: fallback refresh token échoué (%s): %s", gamertag, exc)

    return None


async def _fetch_economy_data(
    tokens: Any,
    xuid: str,
) -> tuple[HomeBattlepassInfo | None, HomeChallengeSummary | None]:
    """Appels réseau Economy + GameCMS avec un token déjà résolu."""
    try:
        from aiohttp import ClientSession, ClientTimeout  # noqa: I001
        from spnkr_pr.services.economy_additions import EconomyServiceExtension
        from spnkr_pr.services.gamecms_hacs_additions import GameCmsHacsServiceExtension
    except ImportError as exc:
        logger.debug("home_api: dépendances manquantes: %s", exc)
        return None, None

    try:
        async with ClientSession(timeout=ClientTimeout(total=10)) as session:
            session.headers.update(
                {
                    "Accept": "application/json",
                    "x-343-authorization-spartan": tokens.spartan_token,
                    "343-clearance": tokens.clearance_token,
                }
            )
            economy_svc = EconomyServiceExtension(session)
            tracks_resp, chal_resp = await asyncio.gather(
                economy_svc.get_player_reward_tracks(xuid),
                economy_svc.get_player_challenges(xuid),
                return_exceptions=True,
            )
            tracks_data = (
                await tracks_resp.parse() if not isinstance(tracks_resp, BaseException) else None
            )
            chal_data = (
                await chal_resp.parse() if not isinstance(chal_resp, BaseException) else None
            )
            bp_info, reward_track_path = _build_battlepass_info(tracks_data)

            if bp_info is not None and reward_track_path is not None:
                gamecms_svc = GameCmsHacsServiceExtension(session)
                try:
                    track_meta_resp = await gamecms_svc.get_operation_reward_track(
                        reward_track_path
                    )
                    track_meta = await track_meta_resp.parse()
                    image_bytes = await gamecms_svc.get_image_bytes(track_meta.summary_image_path)
                    bp_info = HomeBattlepassInfo(
                        track_name=bp_info.track_name,
                        current_progress=bp_info.current_progress,
                        is_owned=bp_info.is_owned,
                        track_image_bytes=image_bytes,
                    )
                except Exception as exc:
                    logger.debug("home_api: artwork opération indisponible: %s", exc)

        return bp_info, _build_challenge_summary(chal_data)
    except Exception as exc:
        logger.debug("home_api: erreur fetch progressions: %s", exc)
        return None, None


async def _async_fetch_progressions(
    db_path: Path,
    xuid: str,
    gamertag: str | None = None,
) -> tuple[HomeBattlepassInfo | None, HomeChallengeSummary | None]:
    """Orchestre résolution des tokens puis appels API Economy/GameCMS."""
    tokens = await _resolve_tokens(db_path, gamertag)
    if tokens is None:
        logger.debug("home_api: aucun token disponible pour %s", gamertag or db_path.name)
        return None, None
    return await _fetch_economy_data(tokens, xuid)


# =============================================================================
# API publique (synchrone, cachée)
# =============================================================================


def fetch_home_progressions(
    db_path: str | Path,
    xuid: str,
    gamertag: str | None = None,
) -> tuple[HomeBattlepassInfo | None, HomeChallengeSummary | None]:
    """Récupère battlepass + défis avec cache session de 5 min.

    Dégradation gracieuse : retourne ``(None, None)`` si l'API est indisponible
    ou si l'authentification est requise.
    """
    import streamlit as st

    cache_key = f"_home_prog_{xuid}"
    cached = st.session_state.get(cache_key)
    now = time.monotonic()
    if cached is not None and now - cached["ts"] < _CACHE_TTL_S:
        return cached["bp"], cached["ch"]

    db_path = Path(db_path)
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    try:
        bp, ch = loop.run_until_complete(_async_fetch_progressions(db_path, xuid, gamertag))
    except Exception as exc:
        logger.debug("home_api: fetch_home_progressions échoué: %s", exc)
        bp, ch = None, None
    finally:
        loop.close()
        asyncio.set_event_loop(None)

    st.session_state[cache_key] = {"ts": now, "bp": bp, "ch": ch}
    return bp, ch
