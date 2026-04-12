"""Données live pour l'accueil V7 — battlepass et défis actifs via SPNKr ext.

Ce module est responsable des appels API asynchrones (avec cache session)
et des dataclasses légères utilisées par home_mission_control.py.
"""

from __future__ import annotations

import asyncio
import logging
import threading
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

_CACHE_TTL_S: int = 300  # 5 minutes

# Cache process-level (survit aux reruns Streamlit, partagé entre sessions)
# Clé : xuid → {ts: float, loading: bool, bp: ..., ch: ...}
_process_cache: dict[str, dict] = {}
_cache_lock = threading.Lock()


# =============================================================================
# Dataclasses
# =============================================================================


@dataclass(frozen=True)
class HomeBattlepassInfo:
    """Info d'opération/saison courante pour l'accueil.

    track_name        : identifiant de l'opération (ex: "S13Op01")
    op_rank           : rang dans l'opération actuelle (ex: 8)
    is_owned          : pass premium acheté
    career_rank       : rang carrière global (ex: 174)
    career_rank_label : label lisible (ex: "Caporal d'élite IV")
    track_image_bytes : artwork de l'opération (BattlePassImage)
    """

    track_name: str
    op_rank: int = 0
    is_owned: bool = False
    career_rank: int = 0
    career_rank_label: str | None = None
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


def _build_challenge_summary_from_json(data: dict) -> HomeChallengeSummary | None:
    """Construit HomeChallengeSummary depuis le JSON brut de l'API /challenges."""
    cats = data.get("CategoryProgress") or data.get("ChallengeDecks") or []
    if not cats:
        return None
    all_ch: list[dict] = []
    for cat in cats:
        for ch in cat.get("ActiveChallenges") or cat.get("Challenges") or []:
            all_ch.append(ch)
    if not all_ch:
        return HomeChallengeSummary(total=0, completed=0, xp_available=0, next_expiry=None)
    completed = sum(
        1
        for ch in all_ch
        if ch.get("CompletionThreshold")
        and ch.get("CurrentProgress", 0) >= ch["CompletionThreshold"]
    )
    xp_available = sum(
        ch.get(
            "XpReward",
            ch.get("Reward", {}).get("InventoryRewards", [{}])[0].get("Quantity", 0)
            if isinstance(ch.get("Reward"), dict)
            else 0,
        )
        for ch in all_ch
    )
    next_expiry = None
    return HomeChallengeSummary(
        total=len(all_ch),
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


async def _fetch_current_operation_track_path(session: Any) -> str | None:
    """Retourne le reward_track_path de l'opération en cours via GameCMS."""
    from datetime import datetime, timezone

    from spnkr.services.gamecms_hacs import GameCmsHacsService

    svc = GameCmsHacsService(session)
    try:
        cal_resp = await svc.get_season_calendar()
        cal = await cal_resp.parse()
        now = datetime.now(timezone.utc)
        current = next(
            (s for s in cal.seasons if s.start_date.value <= now <= s.end_date.value),
            None,
        )
        if current is None and cal.seasons:
            current = cal.seasons[-1]
        return getattr(current, "operation_track_path", None) if current else None
    except Exception as exc:
        logger.debug("home_api: calendrier saison indisponible: %s", exc)
        return None


async def _parse_career_rank(resp: Any) -> int:
    """Extrait le career rank depuis la réponse /rewardtracks/careerranks/careerrank1."""
    if isinstance(resp, BaseException) or resp.status != 200:
        return 0
    data = await resp.json()
    return data.get("CurrentProgress", {}).get("Rank", 0)


async def _parse_op_progression(resp: Any | None) -> tuple[int, bool]:
    """Extrait (op_rank, is_owned) depuis la réponse /rewardtracks/operations/{id}."""
    if resp is None or isinstance(resp, BaseException) or resp.status != 200:
        return 0, False
    data = await resp.json()
    prog = data.get("CurrentProgress", {})
    op_rank = prog.get("Rank", 0)
    is_owned = data.get("IsOwned", False) or prog.get("IsOwned", False)
    return op_rank, is_owned


async def _fetch_economy_data(
    tokens: Any,
    xuid: str,
) -> tuple[HomeBattlepassInfo | None, HomeChallengeSummary | None]:
    """Appels réseau Economy + GameCMS avec des endpoints valides.

    - Career rank    : /hi/players/xuid({xuid})/rewardtracks/careerranks/careerrank1
    - Op progression : /hi/players/xuid({xuid})/rewardtracks/operations/{season_id}
    - Opération      : GameCMS season_calendar + fichier JSON de l'opération
    - Artwork        : GameCMS /hi/images/file/{BattlePassImage}

    Note : l'endpoint /challenges retourne 404 — non disponible en V6 API.
    """
    try:
        from aiohttp import ClientSession, ClientTimeout  # noqa: I001
    except ImportError as exc:
        logger.debug("home_api: aiohttp manquant: %s", exc)
        return None, None

    economy_host = "https://economy.svc.halowaypoint.com"
    gamecms_host = "https://gamecms-hacs.svc.halowaypoint.com"
    xuid_clean = xuid.lstrip("xuid(").rstrip(")")
    hdr = {
        "Accept": "application/json",
        "x-343-authorization-spartan": tokens.spartan_token,
        "343-clearance": tokens.clearance_token,
    }

    try:
        async with ClientSession(timeout=ClientTimeout(total=15), headers=hdr) as session:
            op_path = await _fetch_current_operation_track_path(session)
            season_id = op_path.split("/")[-1].replace(".json", "") if op_path else None

            career_url = (
                f"{economy_host}/hi/players/xuid({xuid_clean})/rewardtracks/careerranks/careerrank1"
            )
            reqs: list[Any] = [session.get(career_url)]
            if season_id:
                op_url = (
                    f"{economy_host}/hi/players/xuid({xuid_clean})"
                    f"/rewardtracks/operations/{season_id}"
                )
                reqs.append(session.get(op_url))

            responses = await asyncio.gather(*reqs, return_exceptions=True)
            career_rank = await _parse_career_rank(responses[0])
            op_rank, is_owned = await _parse_op_progression(
                responses[1] if len(responses) > 1 else None
            )

            bp_info = None
            if season_id:
                image_bytes = await _fetch_operation_artwork(session, gamecms_host, op_path)  # type: ignore[arg-type]
                bp_info = HomeBattlepassInfo(
                    track_name=season_id,
                    op_rank=op_rank,
                    is_owned=is_owned,
                    career_rank=career_rank,
                    track_image_bytes=image_bytes,
                )

        return bp_info, None  # challenges non disponibles (endpoint 404 en V6)
    except Exception as exc:
        logger.debug("home_api: erreur fetch economy: %s", exc)
        return None, None


async def _fetch_operation_artwork(session: Any, gamecms_host: str, op_path: str) -> bytes | None:
    """Télécharge l'artwork (BattlePassImage) d'une opération GameCMS."""
    try:
        meta_url = f"{gamecms_host}/hi/Progression/file/{op_path}"
        async with session.get(meta_url) as r:
            if r.status != 200:
                return None
            meta = await r.json(content_type=None)
        img_path = meta.get("BattlePassImage") or meta.get("BackgroundImagePath")
        if not img_path:
            return None
        img_url = f"{gamecms_host}/hi/images/file/{img_path.lstrip('/')}"
        async with session.get(img_url) as r:
            if r.status != 200:
                return None
            return await r.read()
    except Exception as exc:
        logger.debug("home_api: artwork opération indisponible: %s", exc)
        return None


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
# API publique (synchrone, cachée + prefetch background)
# =============================================================================


def prefetch_home_progressions(
    db_path: str | Path,
    xuid: str,
    gamertag: str | None = None,
) -> None:
    """Déclenche le fetch battlepass/défis en arrière-plan (thread daemon).

    Sans effet si le cache est encore frais ou si un fetch est déjà en cours.
    Appeler le plus tôt possible dans le cycle de rendu Streamlit pour que
    les données soient prêtes quand l'utilisateur arrive sur l'accueil.
    """
    now = time.monotonic()
    with _cache_lock:
        cached = _process_cache.get(xuid)
        if cached is not None and (cached.get("loading") or now - cached["ts"] < _CACHE_TTL_S):
            return
        # Réserver l'entrée pour éviter les lancements concurrents
        _process_cache[xuid] = {"ts": now, "loading": True, "bp": None, "ch": None}

    _db_path = Path(db_path)

    def _run() -> None:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            bp, ch = loop.run_until_complete(_async_fetch_progressions(_db_path, xuid, gamertag))
        except Exception as exc:
            logger.debug("prefetch: erreur (%s): %s", gamertag or xuid[:8], exc)
            bp, ch = None, None
        finally:
            loop.close()
            asyncio.set_event_loop(None)
        with _cache_lock:
            _process_cache[xuid] = {
                "ts": time.monotonic(),
                "loading": False,
                "bp": bp,
                "ch": ch,
            }
        logger.debug("prefetch: terminé pour %s (bp=%s)", gamertag or xuid[:8], bp is not None)

    threading.Thread(target=_run, daemon=True, name=f"home_prefetch_{xuid[:8]}").start()


def fetch_home_progressions(
    db_path: str | Path,
    xuid: str,
    gamertag: str | None = None,
) -> tuple[HomeBattlepassInfo | None, HomeChallengeSummary | None]:
    """Récupère battlepass + défis — retourne immédiatement si prefetch dispo.

    Dégradation gracieuse : retourne ``(None, None)`` si l'API est indisponible
    ou si l'authentification est requise.
    """
    now = time.monotonic()
    with _cache_lock:
        cached = _process_cache.get(xuid)

    if cached is not None and not cached.get("loading") and now - cached["ts"] < _CACHE_TTL_S:
        return cached["bp"], cached["ch"]

    # Pas de cache frais : fetch synchrone (prefetch non terminé ou non déclenché)
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

    with _cache_lock:
        _process_cache[xuid] = {"ts": time.monotonic(), "loading": False, "bp": bp, "ch": ch}
    return bp, ch
