"""Récupération automatique d'assets profil (opt-in).

Objectif:
- Récupérer automatiquement des éléments d'identité (service tag, emblem, backdrop, nameplate)
  depuis l'API Halo Waypoint via SPNKr.
- Mettre en cache sur disque pour éviter d'appeler l'API à chaque rerun Streamlit.

Contraintes:
- Accès réseau uniquement si l'option est activée.
- Tolérance aux erreurs (pas de crash UI si SPNKr n'est pas installé / tokens manquants).
"""

from __future__ import annotations

import contextlib
import os
import time

from src.config import CAREER_RANK_MAX
from src.ui.career_ranks import format_career_rank_label_fr

# Re-export des modules décomposés pour compatibilité
from src.ui.profile_api_cache import (
    ProfileAppearance,
    _safe_write_json,
    _xuid_cache_path,
    load_cached_appearance,
    load_cached_xuid_for_gamertag,
    save_cached_appearance,
)
from src.ui.profile_api_tokens import (
    _is_probable_auth_error,
)
from src.ui.profile_api_tokens import (
    get_tokens as _get_tokens,
)
from src.ui.profile_api_urls import (
    _inventory_backdrop_to_waypoint_png,
    _inventory_emblem_to_waypoint_png,
    _to_image_url,
    _waypoint_nameplate_png_from_emblem,
)
from src.ui.profile_api_urls import (
    resolve_inventory_png_via_api as _resolve_inventory_png_via_api,
)
from src.ui.profile_api_urls import (
    resolve_positive_emblem_cfg as _resolve_positive_emblem_cfg,
)
from src.utils.async_compat import run_sync as _run_sync_compat


async def _parse_spnkr_response(resp: object) -> object:
    """Parse une réponse SPNKr (data attribute ou parse() method)."""
    if hasattr(resp, "data"):
        return resp.data
    if hasattr(resp, "parse"):
        return await resp.parse()  # type: ignore[attr-defined]
    return resp


async def _call_customization_with_tokens(
    session, st_in: str, ct_in: str, *, xuid: str, requests_per_second: int
) -> tuple:
    """Récupère customization et career rank pour un joueur via SPNKr."""
    from src.data.sync._tokens import Tokens
    from src.data.sync.api_client import SPNKrAPIClient

    tokens = Tokens(spartan_token=st_in, clearance_token=ct_in)
    api = SPNKrAPIClient(tokens=tokens, requests_per_second=int(requests_per_second))
    await api.__aenter__()
    try:
        client = api.client
        resp = await client.economy.get_player_customization(str(xuid).strip(), view_type="public")
        customization = await _parse_spnkr_response(resp)
        (
            rank_label,
            rank_subtitle,
            rank_image_url,
            adornment_image_url,
        ) = await _get_career_rank_for_player(
            client, session, st_in, ct_in, xuid, _parse_spnkr_response
        )
        return customization, rank_label, rank_subtitle, rank_image_url, adornment_image_url
    finally:
        await api.__aexit__(None, None, None)


def fetch_appearance_via_spnkr(  # noqa: PLR0913
    *,
    xuid: str,
    gamertag: str | None = None,
    spartan_token: str | None = None,
    clearance_token: str | None = None,
    requests_per_second: int = 3,
    timeout_seconds: int = 12,
) -> ProfileAppearance:
    """Appelle l'API Halo Waypoint via SPNKr pour récupérer l'apparence.

    Si ``gamertag`` est fourni, utilise en priorité le refresh token propre
    au joueur (``SPNKR_OAUTH_REFRESH_TOKEN_<GT>`` dans ``.env.local``) pour
    les endpoints economy player-gated (customisation, career rank).
    """

    async def _run() -> ProfileAppearance:
        import aiohttp

        timeout = aiohttp.ClientTimeout(total=float(timeout_seconds))
        async with aiohttp.ClientSession(timeout=timeout) as session:
            st, ct = await _get_tokens(
                session,
                spartan_token=spartan_token,
                clearance_token=clearance_token,
                timeout_seconds=timeout_seconds,
                gamertag=gamertag,
            )

            try:
                (
                    customization,
                    rank_label,
                    rank_subtitle,
                    rank_image_url,
                    adornment_image_url,
                ) = await _call_customization_with_tokens(
                    session, st, ct, xuid=xuid, requests_per_second=requests_per_second
                )
            except Exception as e:
                if (spartan_token and clearance_token) and _is_probable_auth_error(e):
                    st, ct = await _get_tokens(
                        session,
                        spartan_token=None,
                        clearance_token=None,
                        timeout_seconds=timeout_seconds,
                        gamertag=gamertag,
                    )
                    (
                        customization,
                        rank_label,
                        rank_subtitle,
                        rank_image_url,
                        adornment_image_url,
                    ) = await _call_customization_with_tokens(
                        session, st, ct, xuid=xuid, requests_per_second=requests_per_second
                    )
                else:
                    raise

            appearance = customization.appearance
            emblem_url, backdrop_url, nameplate_url = await _resolve_appearance_urls(
                session, appearance, st, ct
            )

            return ProfileAppearance(
                service_tag=str(getattr(appearance, "service_tag", "") or "").strip() or None,
                emblem_image_url=emblem_url,
                backdrop_image_url=backdrop_url,
                nameplate_image_url=nameplate_url,
                rank_label=(str(rank_label or "").strip() or None),
                rank_subtitle=(str(rank_subtitle or "").strip() or None),
                rank_image_url=(str(rank_image_url or "").strip() or None),
                adornment_image_url=(str(adornment_image_url or "").strip() or None),
                # spartan_id : récupéré depuis la DB career_progression (sync player-gated)
                spartan_id=None,
            )

    return _run_sync_compat(_run(), timeout=float(timeout_seconds) + 20.0)


async def _get_career_rank_for_player(  # noqa: C901, PLR0912, PLR0913
    client, session, st_in: str, ct_in: str, xuid: str, parse_fn
) -> tuple[str | None, str | None, str | None, str | None]:
    """Récupère le Career Rank du joueur via les APIs Halo.

    1. Appelle gamecms_hacs.get_career_reward_track() pour les métadonnées des rangs
    2. Appelle l'endpoint Economy pour la progression du joueur

    Returns:
        (rank_label, rank_subtitle, rank_image_url, adornment_image_url)
    """
    xu = str(xuid).strip()
    try:
        gamecms = getattr(client, "gamecms_hacs", None)
        if gamecms is None:
            return None, None, None, None
        career_track_resp = await gamecms.get_career_reward_track()
        career_track = await parse_fn(career_track_resp)
        if career_track is None:
            return None, None, None, None
        ranks_list = getattr(career_track, "ranks", None)
        if not ranks_list:
            return None, None, None, None

        current_progress = await _fetch_career_progress(session, xu, st_in, ct_in)
        if current_progress is None:
            return None, None, None, None

        current_rank = current_progress.get("Rank")
        partial_xp = current_progress.get("PartialProgress", 0)
        if current_rank is None:
            return None, None, None, None

        return _build_career_rank_result(ranks_list, current_rank, partial_xp)
    except Exception:
        return None, None, None, None


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


async def _resolve_appearance_urls(
    session, appearance, st: str, ct: str
) -> tuple[str | None, str | None, str | None]:
    """Résout les URLs PNG finales pour l'emblem, le backdrop et la nameplate.

    Returns:
        (emblem_url, backdrop_url, nameplate_url)
    """
    emblem_path = getattr(appearance.emblem, "emblem_path", None)
    emblem_cfg = getattr(appearance.emblem, "configuration_id", None)

    # Emblem: pattern Waypoint rapide → API progression → fallback JSON
    emblem_url = _inventory_emblem_to_waypoint_png(emblem_path, emblem_cfg)
    if not emblem_url and emblem_path:
        emblem_url = await _resolve_inventory_png_via_api(
            session, emblem_path, spartan_token=st, clearance_token=ct
        )
    if not emblem_url:
        emblem_url = _to_image_url(emblem_path)

    # Backdrop: pattern Waypoint → API progression → fallback JSON
    backdrop_raw = getattr(appearance, "backdrop_image_path", None)
    backdrop_url = _inventory_backdrop_to_waypoint_png(backdrop_raw)
    if not backdrop_url and backdrop_raw:
        backdrop_url = await _resolve_inventory_png_via_api(
            session, backdrop_raw, spartan_token=st, clearance_token=ct
        )
    if not backdrop_url:
        backdrop_url = _to_image_url(backdrop_raw)

    # Nameplate: si cfg <= 0, résoudre le premier cfg positif disponible
    player_title_path = getattr(appearance, "player_title_path", None)
    if not _to_image_url(player_title_path):
        try:
            _raw_cfg = int(emblem_cfg) if emblem_cfg is not None else None
        except (TypeError, ValueError):
            _raw_cfg = None
        if _raw_cfg is not None and _raw_cfg <= 0:
            _nameplate_cfg = await _resolve_positive_emblem_cfg(
                session, emblem_path, emblem_cfg, spartan_token=st, clearance_token=ct
            )
        else:
            _nameplate_cfg = _raw_cfg
        nameplate_url = _waypoint_nameplate_png_from_emblem(emblem_path, _nameplate_cfg)
    else:
        nameplate_url = _to_image_url(player_title_path)

    return emblem_url, backdrop_url, nameplate_url


def fetch_xuid_via_spnkr(
    *,
    gamertag: str,
    spartan_token: str | None = None,
    clearance_token: str | None = None,
    requests_per_second: int = 3,
    timeout_seconds: int = 12,
) -> tuple[str, str]:
    """Retourne (xuid_digits, canonical_gamertag) pour un gamertag."""

    def _run_sync(coro):
        return _run_sync_compat(coro, timeout=float(timeout_seconds) + 20.0)

    async def _run() -> tuple[str, str]:
        import aiohttp

        from src.data.sync._tokens import Tokens
        from src.data.sync.api_factory import create_api_client

        gt = str(gamertag or "").strip()
        if not gt:
            raise RuntimeError("Gamertag vide.")

        timeout = aiohttp.ClientTimeout(total=float(timeout_seconds))
        async with aiohttp.ClientSession(timeout=timeout) as session:

            async def _call_with_tokens(st_in: str, ct_in: str):
                tokens = Tokens(spartan_token=st_in, clearance_token=ct_in)
                async with create_api_client(
                    tokens=tokens,
                    requests_per_second=int(requests_per_second),
                ) as client:
                    result = await client.get_user_by_gamertag(gt)
                    if result is None:
                        raise RuntimeError(f"Gamertag introuvable : {gt}")
                    return result

            st, ct = await _get_tokens(
                session,
                spartan_token=spartan_token,
                clearance_token=clearance_token,
                timeout_seconds=timeout_seconds,
            )

            try:
                resp = await _call_with_tokens(st, ct)
            except Exception as e:
                if (spartan_token and clearance_token) and _is_probable_auth_error(e):
                    st, ct = await _get_tokens(
                        session,
                        spartan_token=None,
                        clearance_token=None,
                        timeout_seconds=timeout_seconds,
                    )
                    resp = await _call_with_tokens(st, ct)
                else:
                    raise

            xuid = resp.get("xuid", "").strip()
            if not xuid.isdigit():
                raise RuntimeError("Impossible de résoudre le XUID pour ce gamertag.")
            canonical_gt = resp.get("gamertag", "").strip() or gt
            return xuid, canonical_gt

    return _run_sync(_run())


def get_xuid_for_gamertag(  # noqa: PLR0913
    *,
    gamertag: str,
    enabled: bool,
    refresh_hours: int,
    force_refresh: bool = False,
    requests_per_second: int = 3,
    timeout_seconds: int = 12,
) -> tuple[str | None, str | None]:
    """Retourne (xuid, error_message)."""

    gt = str(gamertag or "").strip()
    if not gt:
        return None, None

    if not force_refresh:
        cached = load_cached_xuid_for_gamertag(gt, refresh_hours=refresh_hours)
        if cached is not None:
            return cached, None

    if not enabled:
        return None, None

    try:
        xuid, canonical_gt = fetch_xuid_via_spnkr(
            gamertag=gt,
            spartan_token=str(os.environ.get("SPNKR_SPARTAN_TOKEN") or "").strip() or None,
            clearance_token=str(os.environ.get("SPNKR_CLEARANCE_TOKEN") or "").strip() or None,
            requests_per_second=requests_per_second,
            timeout_seconds=timeout_seconds,
        )
    except ImportError as e:
        # Erreur de dépendances manquantes
        return (
            None,
            f"Échec résolution XUID via SPNKr: {e}",
        )
    except Exception as e:
        # Autres erreurs (tokens manquants, erreur API, etc.)
        error_msg = str(e)
        if "Module" in error_msg and "manquant" in error_msg:
            # C'est déjà un message d'erreur de dépendances
            return None, f"Échec résolution XUID via SPNKr: {error_msg}"
        return (
            None,
            (
                "Échec résolution XUID via SPNKr: "
                f"{error_msg} (attendu: SPNKR_AZURE_CLIENT_ID + SPNKR_AZURE_CLIENT_SECRET + SPNKR_OAUTH_REFRESH_TOKEN, "
                "ou SPNKR_SPARTAN_TOKEN + SPNKR_CLEARANCE_TOKEN)"
            ),
        )

    with contextlib.suppress(Exception):
        _safe_write_json(
            _xuid_cache_path(gt),
            {
                "fetched_at": time.time(),
                "gamertag": canonical_gt,
                "xuid": xuid,
            },
        )

    return xuid, None


def get_profile_appearance(  # noqa: PLR0913
    *,
    xuid: str,
    enabled: bool,
    refresh_hours: int,
    requests_per_second: int = 3,
    timeout_seconds: int = 12,
    force_refresh: bool = False,
    gamertag: str | None = None,
) -> tuple[ProfileAppearance | None, str | None]:
    """Retourne (appearance, error_message).

    Stratégie cache :
    1. Cache frais (< refresh_hours) → retourné immédiatement.
    2. API activée et cache périmé → appel API.
       - Succès : met à jour le cache, retourne la nouvelle donnée.
       - Échec (token expiré, réseau…) : retombe sur le cache périmé si disponible
         ("stale-on-error") pour conserver les assets déjà téléchargés.
    3. API désactivée et cache absent → retourne None.

    Args:
        gamertag: Gamertag du joueur. Si fourni, utilise en priorité le token
            joueur (``SPNKR_OAUTH_REFRESH_TOKEN_<GT>``) pour les endpoints
            economy player-gated (customisation, career rank).
    """

    xu = str(xuid or "").strip()
    if not xu or not xu.isdigit():
        return None, None

    if not force_refresh:
        cached = load_cached_appearance(xu, refresh_hours=refresh_hours)
        if cached is not None:
            return cached, None

    if not enabled:
        return None, None

    try:
        appearance = fetch_appearance_via_spnkr(
            xuid=xu,
            gamertag=gamertag,
            spartan_token=str(os.environ.get("SPNKR_SPARTAN_TOKEN") or "").strip() or None,
            clearance_token=str(os.environ.get("SPNKR_CLEARANCE_TOKEN") or "").strip() or None,
            requests_per_second=requests_per_second,
            timeout_seconds=timeout_seconds,
        )
    except ImportError as e:
        # Erreur de dépendances manquantes — pas de fallback cache (dépend du libelle)
        return (
            None,
            f"Échec récupération profil via SPNKr: {e}",
        )
    except Exception as e:
        # Erreur API (token expiré, réseau, 403…) — on retourne le cache périmé si disponible
        # plutôt que de perdre les assets déjà récupérés.
        stale = load_cached_appearance(xu, refresh_hours=999_999)
        if stale is not None:
            return stale, None
        error_msg = str(e)
        if "Module" in error_msg and "manquant" in error_msg:
            return None, f"Échec récupération profil via SPNKr: {error_msg}"
        return (
            None,
            (
                "Échec récupération profil via SPNKr: "
                f"{error_msg} (attendu: SPNKR_AZURE_CLIENT_ID + SPNKR_AZURE_CLIENT_SECRET + SPNKR_OAUTH_REFRESH_TOKEN, "
                "ou SPNKR_SPARTAN_TOKEN + SPNKR_CLEARANCE_TOKEN)"
            ),
        )

    with contextlib.suppress(Exception):
        save_cached_appearance(xu, appearance)

    return appearance, None
