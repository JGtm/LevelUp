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

from src.data.challenges import (
    collect_challenge_paths_from_decks,
    load_challenge_metadata_map,
    persist_challenge_catalog,
    persist_challenge_snapshots,
)
from src.ui.pages.home_mission_control_challenges import (
    build_challenge_summary_from_decks,
    enrich_active_challenges,
    fetch_challenge_definitions,
)

logger = logging.getLogger(__name__)

_CACHE_TTL_S: int = 300  # 5 minutes

# Cache process-level (survit aux reruns Streamlit, partagé entre sessions)
# Clé : xuid:lang → {ts: float, loading: bool, bp: ..., ch: ...}
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
    title: str | None = None
    description: str | None = None
    badge_bytes: bytes | None = None
    progress_current: int | None = None
    progress_target: int | None = None


@dataclass(frozen=True)
class HomeChallengeFetchContext:
    """Contexte de récupération live des défis home."""

    db_path: Path
    tokens: Any
    xuid_clean: str
    session: Any
    gamecms_host: str
    lang: str


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


async def _fetch_challenge_summary(
    context: HomeChallengeFetchContext,
) -> HomeChallengeSummary | None:
    """Récupère les défis actifs via halostats /decks (Spartan token seul)."""
    from aiohttp import ClientSession, ClientTimeout

    decks_host = "https://halostats.svc.halowaypoint.com"
    deck_headers = {
        "Accept": "application/json",
        "x-343-authorization-spartan": context.tokens.spartan_token,
    }
    decks_url = f"{decks_host}/hi/players/xuid({context.xuid_clean})/decks"

    async with (
        ClientSession(
            timeout=ClientTimeout(total=15),
            headers=deck_headers,
        ) as deck_session,
        deck_session.get(decks_url) as resp,
    ):
        if resp.status != 200:
            logger.debug("home_api: decks indisponible (%s)", resp.status)
            return None
        decks_data = await resp.json(content_type=None)

    summary_data, active_challenges = build_challenge_summary_from_decks(decks_data)
    if summary_data is None:
        return None
    all_challenge_paths = collect_challenge_paths_from_decks(decks_data)
    if not active_challenges:
        persist_challenge_snapshots(context.db_path, context.xuid_clean, decks_data)
        return HomeChallengeSummary(
            total=summary_data["total"],
            completed=summary_data["completed"],
            xp_available=0,
            next_expiry=summary_data["next_expiry"],
        )

    enrichment = await enrich_active_challenges(
        context.session,
        context.gamecms_host,
        active_challenges,
        context.lang,
        challenge_paths=all_challenge_paths,
    )
    definitions = dict(enrichment.get("definitions") or {})
    stored_metadata = load_challenge_metadata_map(all_challenge_paths, context.lang)
    missing_paths = [
        path
        for path in all_challenge_paths
        if path not in definitions and path not in stored_metadata
    ]
    if missing_paths:
        extra_definitions = await fetch_challenge_definitions(
            context.session,
            context.gamecms_host,
            missing_paths,
        )
        definitions.update(extra_definitions)
    definition_hashes = persist_challenge_catalog(definitions)
    persist_challenge_snapshots(
        context.db_path,
        context.xuid_clean,
        decks_data,
        definitions=definitions,
        definition_hashes=definition_hashes,
    )
    return HomeChallengeSummary(
        total=summary_data["total"],
        completed=summary_data["completed"],
        xp_available=enrichment["xp_available"],
        next_expiry=summary_data["next_expiry"],
        title=enrichment["title"],
        description=enrichment["description"],
        badge_bytes=enrichment["badge_bytes"],
        progress_current=enrichment["progress_current"],
        progress_target=enrichment["progress_target"],
    )


async def _fetch_economy_data(
    db_path: Path,
    tokens: Any,
    xuid: str,
    lang: str,
) -> tuple[HomeBattlepassInfo | None, HomeChallengeSummary | None]:
    """Appels réseau Economy + GameCMS avec des endpoints valides.

    - Career rank    : /hi/players/xuid({xuid})/rewardtracks/careerranks/careerrank1
    - Op progression : /hi/players/xuid({xuid})/rewardtracks/operations/{season_id}
    - Défis actifs   : halostats /hi/players/xuid({xuid})/decks (sans clearance)
    - Opération      : GameCMS season_calendar + fichier JSON de l'opération
    - Artwork        : GameCMS /hi/images/file/{BattlePassImage}
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
            challenge_summary = await _fetch_challenge_summary(
                HomeChallengeFetchContext(
                    db_path=db_path,
                    tokens=tokens,
                    xuid_clean=xuid_clean,
                    session=session,
                    gamecms_host=gamecms_host,
                    lang=lang,
                )
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

        return bp_info, challenge_summary
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
    lang: str = "fr",
) -> tuple[HomeBattlepassInfo | None, HomeChallengeSummary | None]:
    """Orchestre résolution des tokens puis appels API Economy/GameCMS."""
    tokens = await _resolve_tokens(db_path, gamertag)
    if tokens is None:
        logger.debug("home_api: aucun token disponible pour %s", gamertag or db_path.name)
        return None, None
    return await _fetch_economy_data(db_path, tokens, xuid, lang)


# =============================================================================
# API publique (synchrone, cachée + prefetch background)
# =============================================================================


def prefetch_home_progressions(
    db_path: str | Path,
    xuid: str,
    gamertag: str | None = None,
    lang: str = "fr",
) -> None:
    """Déclenche le fetch battlepass/défis en arrière-plan (thread daemon).

    Sans effet si le cache est encore frais ou si un fetch est déjà en cours.
    Appeler le plus tôt possible dans le cycle de rendu Streamlit pour que
    les données soient prêtes quand l'utilisateur arrive sur l'accueil.
    """
    now = time.monotonic()
    cache_key = _make_cache_key(xuid, lang)
    with _cache_lock:
        cached = _process_cache.get(cache_key)
        if cached is not None and (cached.get("loading") or now - cached["ts"] < _CACHE_TTL_S):
            return
        # Réserver l'entrée pour éviter les lancements concurrents
        _process_cache[cache_key] = {"ts": now, "loading": True, "bp": None, "ch": None}

    _db_path = Path(db_path)

    def _run() -> None:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            bp, ch = loop.run_until_complete(
                _async_fetch_progressions(_db_path, xuid, gamertag, lang)
            )
        except Exception as exc:
            logger.debug("prefetch: erreur (%s): %s", gamertag or xuid[:8], exc)
            bp, ch = None, None
        finally:
            loop.close()
            asyncio.set_event_loop(None)
        with _cache_lock:
            _process_cache[cache_key] = {
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
    lang: str = "fr",
) -> tuple[HomeBattlepassInfo | None, HomeChallengeSummary | None]:
    """Récupère battlepass + défis — retourne immédiatement si prefetch dispo.

    Dégradation gracieuse : retourne ``(None, None)`` si l'API est indisponible
    ou si l'authentification est requise.
    """
    now = time.monotonic()
    cache_key = _make_cache_key(xuid, lang)
    with _cache_lock:
        cached = _process_cache.get(cache_key)

    if cached is not None and not cached.get("loading") and now - cached["ts"] < _CACHE_TTL_S:
        return cached["bp"], cached["ch"]

    # Pas de cache frais : fetch synchrone (prefetch non terminé ou non déclenché)
    db_path = Path(db_path)
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    try:
        bp, ch = loop.run_until_complete(_async_fetch_progressions(db_path, xuid, gamertag, lang))
    except Exception as exc:
        logger.debug("home_api: fetch_home_progressions échoué: %s", exc)
        bp, ch = None, None
    finally:
        loop.close()
        asyncio.set_event_loop(None)

    with _cache_lock:
        _process_cache[cache_key] = {
            "ts": time.monotonic(),
            "loading": False,
            "bp": bp,
            "ch": ch,
        }
    return bp, ch


def _make_cache_key(xuid: str, lang: str) -> str:
    """Construit une clé de cache sensible à la langue UI."""
    normalized_lang = str(lang or "fr").strip().lower() or "fr"
    return f"{xuid}:{normalized_lang}"
