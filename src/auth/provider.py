"""Point d'entrée unique de la couche auth LevelUp.

API publique — tout le code applicatif importe uniquement depuis ce module :

    from src.auth.provider import get_halo_tokens, AuthRequiredError

Algorithme de ``get_halo_tokens(db_path)`` :
    1. Cache process-level valide → retour immédiat (pas d'appel réseau)
    2. MSAL silent (refresh_token caché en DuckDB → nouvel access_token)
    3. access_token → chaîne XBL/XSTS/Spartan/Clearance (spnkr.auth)
    4. Mise à jour cache process + cache MSAL DuckDB si tourné
    5. Si silent échoue → raise AuthRequiredError (côté Streamlit)
       ou lance Device Code Flow interactif (côté CLI)

Séparation CLI / Streamlit :
    - CLI  : ``get_halo_tokens_cli(db_path)`` — bloquant, affiche le code en console
    - UI   : ``get_halo_tokens_or_raise(db_path)`` + ``start_device_flow()``
               / ``complete_device_flow()``
"""

from __future__ import annotations

import asyncio
import logging
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from src.auth._constants import HALO_TOKEN_CACHE_TTL_S
from src.auth._halo_exchange import exchange_access_token_for_halo, resolve_player_identity
from src.auth._msal import (
    DeviceCodeInfo,
    MsalUnavailableError,  # noqa: F401 — re-exporté via __init__
    acquire_token_by_device_flow,
    acquire_token_silent,
    build_msal_app,
    initiate_device_flow,
    load_msal_cache,
    save_msal_cache_if_changed,
)

logger = logging.getLogger(__name__)

# Méthode d'auth préférée (configurable via AppSettings.auth_method)
# "refresh_token" : OAuth v2 direct (skip Device Code Flow)
# "msal"          : MSAL Device Code Flow (comportement historique)
_preferred_auth_method: str = "refresh_token"


def set_preferred_auth_method(method: str) -> None:
    """Définit la méthode d'auth préférée (appelé au démarrage depuis AppSettings)."""
    global _preferred_auth_method
    _preferred_auth_method = method if method in ("refresh_token", "msal") else "refresh_token"
    logger.debug("Auth method: %s", _preferred_auth_method)


# =============================================================================
# Types internes (évite l'import circulaire avec src.data.sync._tokens)
# =============================================================================


def _make_tokens(spartan_token: str, clearance_token: str) -> Any:
    """Construit un objet Tokens en important lazily depuis _tokens.py.

    Import retardé pour éviter le cycle d'import :
    provider → _tokens → sync.__init__ → api_client → provider.
    """
    from src.data.sync._tokens import Tokens

    return Tokens(spartan_token=spartan_token, clearance_token=clearance_token)


# =============================================================================
# Cache process-level des Halo tokens
# =============================================================================

# Clé: chemin DB résolu en str → valeur: (Tokens, timestamp expiry)
_halo_token_cache: dict[str, tuple[Any, float]] = {}


def _get_cached_tokens(db_path: Path) -> Any | None:
    """Retourne les tokens depuis le cache process si encore valides."""
    key = str(db_path.resolve())
    entry = _halo_token_cache.get(key)
    if entry is None:
        return None
    tokens, expiry = entry
    if time.monotonic() < expiry:
        logger.debug("_get_cached_tokens: hit cache pour %s", db_path.name)
        return tokens
    del _halo_token_cache[key]
    return None


def _set_cached_tokens(db_path: Path, tokens: Any) -> None:
    """Met en cache les Halo tokens pour la durée TTL configurée."""
    key = str(db_path.resolve())
    _halo_token_cache[key] = (tokens, time.monotonic() + HALO_TOKEN_CACHE_TTL_S)


def invalidate_token_cache(db_path: Path | str) -> None:
    """Invalide le cache process pour forcer un renouvellement au prochain appel."""
    key = str(Path(db_path).resolve())
    _halo_token_cache.pop(key, None)


# =============================================================================
# Erreur de re-authentification
# =============================================================================


@dataclass
class DeviceCodePending:
    """État du Device Code Flow en cours — retourné par ``start_device_flow()``.

    Attributes:
        user_code: Code court à entrer sur la page de vérification.
        verification_url: URL de vérification (``https://microsoft.com/devicelogin``).
        expires_in: Secondes avant expiration.
        message: Message MSAL verbatim (localisé).
        _info: Objet interne DeviceCodeInfo (opaque pour les appelants).
    """

    user_code: str
    verification_url: str
    expires_in: int
    message: str
    _info: DeviceCodeInfo = field(repr=False)


class AuthRequiredError(Exception):
    """Levée quand re-authentification interactive est requise.

    Côté Streamlit : attraper cette exception et appeler ``start_device_flow()``
    pour afficher le code à l'utilisateur.
    """

    def __init__(self, detail: str = "Re-authentification requise.") -> None:
        super().__init__(detail)


# =============================================================================
# Obtention des tokens Halo (interface principale)
# =============================================================================


async def get_halo_tokens(db_path: Path | str) -> Any:
    """Obtient les tokens Halo Infinite pour un joueur.

    Algorithme : cache process → MSAL silent → échange Halo.
    Lance un Device Code Flow interactif (bloquant, console) si MSAL silent échoue.

    Adapté pour la CLI et les scripts où une interaction console est acceptable.
    Pour Streamlit, utiliser ``get_halo_tokens_or_raise()``.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.

    Returns:
        Tokens (spartan_token, clearance_token).

    Raises:
        DeviceFlowError: Si l'utilisateur refuse ou si le code expire.
        MsalUnavailableError: Si MSAL n'est pas installé.
    """
    path = Path(db_path)

    # 1. Cache process
    cached = _get_cached_tokens(path)
    if cached is not None:
        return cached

    # 2. MSAL silent ou Device Code Flow interactif
    access_token = await _get_access_token_interactive(path)

    # 3. Échange → tokens Halo
    tokens = await _exchange_and_cache(path, access_token)
    return tokens


async def get_halo_tokens_or_raise(db_path: Path | str) -> Any:
    """Obtient les tokens Halo pour un joueur — version Streamlit.

    Si MSAL silent réussit → retourne les tokens directement.
    Si re-auth est nécessaire → raise ``AuthRequiredError``.
    L'appelant doit attraper cette exception et appeler ``start_device_flow()``
    + ``complete_device_flow()`` pour relancer le flux interactif.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.

    Returns:
        Tokens (spartan_token, clearance_token).

    Raises:
        AuthRequiredError: Si re-authentification interactive requise.
        MsalUnavailableError: Si MSAL n'est pas installé.
    """
    path = Path(db_path)

    # 1. Cache process
    cached = _get_cached_tokens(path)
    if cached is not None:
        return cached

    # 2. Tentative silencieuse uniquement
    access_token = await _try_silent_access_token(path)
    if access_token is None:
        raise AuthRequiredError(
            "Aucun token valide dans le cache. Veuillez vous reconnecter via le Device Code Flow."
        )

    # 3. Échange → tokens Halo
    tokens = await _exchange_and_cache(path, access_token)
    return tokens


# =============================================================================
# Fonctions Device Code Flow (côté UI)
# =============================================================================


def start_device_flow(db_path: Path | str) -> DeviceCodePending:
    """Démarre le Device Code Flow pour un joueur.

    Non-bloquant : retourne immédiatement avec les informations à afficher.
    Appeler ensuite ``complete_device_flow()`` dans un thread séparé.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.

    Returns:
        ``DeviceCodePending`` avec le code et l'URL à afficher.

    Raises:
        DeviceFlowError: Si l'initiation échoue.
        MsalUnavailableError: Si MSAL n'est pas installé.
    """
    path = Path(db_path)
    cache = load_msal_cache(path)
    app = build_msal_app(cache)
    info = initiate_device_flow(app)

    return DeviceCodePending(
        user_code=info.user_code,
        verification_url=info.verification_url,
        expires_in=info.expires_in,
        message=info.message,
        _info=info,
    )


async def complete_device_flow(
    db_path: Path | str,
    pending: DeviceCodePending,
) -> tuple[str, str]:
    """Complète le Device Code Flow et retourne l'identité du joueur.

    Bloquant jusqu'à validation par l'utilisateur ou expiration.
    Doit être appelé dans un thread dédié côté Streamlit.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        pending: ``DeviceCodePending`` retourné par ``start_device_flow()``.

    Returns:
        Tuple ``(gamertag, xuid)``.

    Raises:
        DeviceFlowError: Si l'utilisateur refuse ou si le code expire.
    """
    path = Path(db_path)
    cache = load_msal_cache(path)
    app = build_msal_app(cache)

    # Bloquer jusqu'à validation (polling MSAL interne)
    loop = asyncio.get_event_loop()
    access_token = await loop.run_in_executor(
        None, acquire_token_by_device_flow, app, pending._info._flow
    )

    # Persister le cache MSAL (contient le refresh_token tourné)
    save_msal_cache_if_changed(path, cache)

    # Échange → Halo tokens + cache process
    tokens = await _exchange_and_cache(path, access_token)

    # Résoudre l'identité
    gamertag, xuid = await resolve_player_identity(tokens.spartan_token, tokens.clearance_token)
    logger.info("complete_device_flow: authentifié — %s (%s)", gamertag, xuid)
    return gamertag, xuid


# =============================================================================
# Helpers internes
# =============================================================================


async def _try_silent_access_token(db_path: Path) -> str | None:
    """Tentative silencieuse d'access_token via MSAL. None si échec."""
    try:
        cache = load_msal_cache(db_path)
        app = build_msal_app(cache)
        access_token = acquire_token_silent(app)
        if access_token:
            save_msal_cache_if_changed(db_path, cache)
        return access_token
    except MsalUnavailableError:
        raise
    except Exception as exc:
        logger.debug("_try_silent_access_token: échec (%s)", exc)
        return None


async def _get_access_token_interactive(db_path: Path) -> str:
    """MSAL silent ou Device Code Flow interactif (CLI). Retourne access_token."""
    cache = load_msal_cache(db_path)
    app = build_msal_app(cache)

    # Tentative silencieuse
    access_token = acquire_token_silent(app)
    if access_token:
        save_msal_cache_if_changed(db_path, cache)
        return access_token

    # Si auth_method=refresh_token, ne pas lancer le Device Code Flow interactif
    # (le caller doit utiliser les refresh tokens OAuth v2 via get_tokens_for_player)
    if _preferred_auth_method == "refresh_token":
        from src.auth._msal import DeviceFlowError

        raise DeviceFlowError(
            "refresh_token_mode",
            "auth_method=refresh_token : Device Code Flow interactif désactivé. "
            "Configurer SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> dans .env.local.",
        )

    # Device Code Flow interactif
    info = initiate_device_flow(app)
    print(f"\n{info.message}")
    print(f"  Code   : {info.user_code}")
    print(f"  URL    : {info.verification_url}")
    print(f"  Expire dans {info.expires_in}s\n")

    loop = asyncio.get_event_loop()
    access_token = await loop.run_in_executor(None, acquire_token_by_device_flow, app, info._flow)
    save_msal_cache_if_changed(db_path, cache)
    return access_token


async def _exchange_and_cache(db_path: Path, access_token: str) -> Any:
    """Échange access_token → Halo tokens + mise en cache process."""
    try:
        from aiohttp import ClientSession, ClientTimeout
    except ImportError as e:
        raise ImportError("aiohttp requis : pip install aiohttp") from e

    async with ClientSession(timeout=ClientTimeout(total=45)) as session:
        spartan, clearance = await exchange_access_token_for_halo(session, access_token)

    tokens = _make_tokens(spartan, clearance)
    _set_cached_tokens(db_path, tokens)
    return tokens
