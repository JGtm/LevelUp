"""Gestion des tokens d'authentification SPNKr.

Regroupe la logique d'obtention des tokens (manuels ou OAuth Azure)
et les helpers de normalisation.
"""

from __future__ import annotations

import logging
import os
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from src.utils.env import load_dotenv_if_present, normalize_gamertag_for_env

logger = logging.getLogger(__name__)


# =============================================================================
# Constantes
# =============================================================================

XUID_RE = re.compile(r"(\d{12,20})")
CLEARANCE_COOKIE_RE = re.compile(r"(?:^|[;\s])343-clearance=([^;\s]+)", re.IGNORECASE)

#: Headers communs pour les appels directs à l'API Halo Waypoint.
HALO_API_HEADERS_KEYS = (
    "x-343-authorization-spartan",
    "343-clearance",
    "Accept",
)


# =============================================================================
# Helpers
# =============================================================================


def _normalize_token_value(raw: Any) -> str | None:
    """Normalise une valeur de token (enlève préfixes headers, etc.)."""
    if raw is None:
        return None
    s = str(raw).strip()
    if not s:
        return None

    # Autorise un copier-coller depuis l'onglet réseau
    if ":" in s:
        _, after = s.split(":", 1)
        s = after.strip()

    # Autorise un copier-coller depuis un header Cookie
    m = CLEARANCE_COOKIE_RE.search(s)
    if m:
        return m.group(1).strip().strip('"').strip("'") or None

    return s or None


def clean_xuid(xuid: str) -> str:
    """Normalise un XUID en retirant le préfixe ``xuid()`` éventuel."""
    s = str(xuid).strip()
    if s.startswith("xuid(") and s.endswith(")"):
        return s[5:-1]
    return s


def build_halo_headers(tokens: Tokens) -> dict[str, str]:
    """Construit les headers pour les appels directs à l'API Halo."""
    return {
        "x-343-authorization-spartan": tokens.spartan_token,
        "343-clearance": tokens.clearance_token,
        "Accept": "application/json",
    }


# =============================================================================
# Tokens dataclass
# =============================================================================


@dataclass(frozen=True)
class Tokens:
    """Paire de tokens SPNKr."""

    spartan_token: str
    clearance_token: str


# =============================================================================
# Fonctions publiques
# =============================================================================


async def get_tokens_from_env() -> Tokens:
    """Récupère les tokens depuis les variables d'environnement.

    Supporte deux modes :
    1. Tokens manuels : SPNKR_SPARTAN_TOKEN + SPNKR_CLEARANCE_TOKEN
    2. OAuth Azure : SPNKR_AZURE_CLIENT_ID + SPNKR_AZURE_CLIENT_SECRET + SPNKR_OAUTH_REFRESH_TOKEN

    Raises:
        SystemExit: Si les tokens sont manquants ou invalides.

    Returns:
        Tokens validés.
    """
    load_dotenv_if_present()

    # Mode OAuth Azure
    azure_client_id = os.environ.get("SPNKR_AZURE_CLIENT_ID")
    azure_client_secret = os.environ.get("SPNKR_AZURE_CLIENT_SECRET")
    azure_redirect_uri = os.environ.get("SPNKR_AZURE_REDIRECT_URI", "https://localhost")
    oauth_refresh_token = os.environ.get("SPNKR_OAUTH_REFRESH_TOKEN")

    # Fallback : si SPNKR_OAUTH_REFRESH_TOKEN absent, chercher SPNKR_OAUTH_REFRESH_TOKEN_<GT>
    if not oauth_refresh_token and azure_client_id and azure_client_secret:
        for key, value in os.environ.items():
            if key.startswith("SPNKR_OAUTH_REFRESH_TOKEN_") and value.strip():
                logger.debug(
                    "SPNKR_OAUTH_REFRESH_TOKEN absent — utilisation de %s comme token par défaut.",
                    key,
                )
                oauth_refresh_token = value.strip()
                break

    # Fallback DB : chercher le token dans sync_meta du premier joueur configuré
    if not oauth_refresh_token and azure_client_id and azure_client_secret:
        try:
            from src.ui.xbox_oauth import load_refresh_token as _load_rt
            from src.utils.profiles import load_profiles as _load_profiles

            _profiles = _load_profiles()
            for _gt, _prof in _profiles.items():
                _db = _prof.get("db_path", "")
                if _db and Path(_db).exists():
                    _rt = _load_rt(_db)
                    if _rt:
                        oauth_refresh_token = _rt
                        logger.debug(
                            "SPNKR_OAUTH_REFRESH_TOKEN chargé depuis sync_meta de '%s'.", _gt
                        )
                        break
        except Exception:
            pass

    if azure_client_id and azure_client_secret and oauth_refresh_token:
        return await _get_tokens_via_oauth(
            azure_client_id,
            azure_client_secret,
            azure_redirect_uri,
            oauth_refresh_token,
        )

    # Mode tokens manuels
    spartan = _normalize_token_value(os.environ.get("SPNKR_SPARTAN_TOKEN"))
    clearance = _normalize_token_value(os.environ.get("SPNKR_CLEARANCE_TOKEN"))

    if not spartan or not clearance:
        raise ValueError(
            "Tokens SPNKr manquants. Définir soit:\n"
            "- SPNKR_SPARTAN_TOKEN + SPNKR_CLEARANCE_TOKEN,\n"
            "- ou SPNKR_AZURE_CLIENT_ID + SPNKR_AZURE_CLIENT_SECRET + SPNKR_OAUTH_REFRESH_TOKEN"
        )

    return Tokens(spartan_token=spartan, clearance_token=clearance)


def get_player_token_env_key(gamertag: str) -> str:
    """Retourne le nom de la variable d'env du refresh token propre au joueur.

    Args:
        gamertag: Gamertag du joueur (ex: « SpartanC »).

    Returns:
        Nom de la variable (ex: « SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC »).
    """
    return f"SPNKR_OAUTH_REFRESH_TOKEN_{normalize_gamertag_for_env(gamertag)}"


async def get_tokens_for_player(gamertag: str) -> Tokens | None:
    """Récupère les tokens SPNKr propres à un joueur depuis l'env.

    Args:
        gamertag: Gamertag du joueur (``SpartanC``, ``Mon GT``, …).

    Returns:
        ``Tokens`` si le token joueur est trouvé et valide, ``None`` sinon.
    """
    load_dotenv_if_present()

    key = get_player_token_env_key(gamertag)
    player_refresh_token = (os.environ.get(key) or "").strip()

    # Fallback DB : token dans sync_meta (connexion Xbox OAuth via UI)
    if not player_refresh_token:
        try:
            from src.ui.xbox_oauth import load_refresh_token as _load_rt

            _repo_root = Path(__file__).resolve().parents[3]
            _player_db = _repo_root / "data" / "players" / gamertag / "stats.duckdb"
            if _player_db.exists():
                _rt = _load_rt(_player_db)
                if _rt:
                    player_refresh_token = _rt
                    logger.debug(
                        "Token joueur '%s' chargé depuis sync_meta (Xbox OAuth UI).", gamertag
                    )
        except Exception:
            pass

    if not player_refresh_token:
        return None

    azure_client_id = (os.environ.get("SPNKR_AZURE_CLIENT_ID") or "").strip()
    azure_client_secret = (os.environ.get("SPNKR_AZURE_CLIENT_SECRET") or "").strip()
    azure_redirect_uri = os.environ.get("SPNKR_AZURE_REDIRECT_URI") or "https://localhost"

    if not azure_client_id or not azure_client_secret:
        logger.warning(
            "Token joueur '%s' trouvé (%s) mais SPNKR_AZURE_CLIENT_ID/SECRET "
            "manquants dans .env.local — token ignoré.",
            gamertag,
            key,
        )
        return None

    logger.debug("Utilisation du token joueur '%s' (%s)", gamertag, key)
    return await _get_tokens_via_oauth(
        azure_client_id,
        azure_client_secret,
        azure_redirect_uri,
        player_refresh_token,
    )


# =============================================================================
# OAuth interne
# =============================================================================


async def _get_tokens_via_oauth(
    client_id: str,
    client_secret: str,
    redirect_uri: str,
    refresh_token: str,
) -> Tokens:
    """Récupère les tokens via OAuth Azure."""
    try:
        from aiohttp import ClientSession, ClientTimeout
        from spnkr import AzureApp, refresh_player_tokens
    except ImportError as e:
        raise ImportError(
            "Dépendances SPNKr manquantes. Installer: pip install spnkr aiohttp"
        ) from e

    app = AzureApp(client_id, client_secret, redirect_uri)

    async with ClientSession(timeout=ClientTimeout(total=45)) as session:
        try:
            player = await refresh_player_tokens(session, app, refresh_token)
            return Tokens(
                spartan_token=str(player.spartan_token.token),
                clearance_token=str(player.clearance_token.token),
            )
        except Exception as e:
            msg = str(e)
            if "invalid_client" not in msg:
                raise
            return await _get_tokens_oauth_v2_fallback(session, app, refresh_token)


async def _get_tokens_oauth_v2_fallback(
    session: Any,
    app: Any,
    refresh_token: str,
) -> Tokens:
    """Fallback OAuth v2 pour les versions anciennes de SPNKr."""
    from spnkr.auth.core import XSTS_V3_HALO_AUDIENCE, XSTS_V3_XBOX_AUDIENCE
    from spnkr.auth.halo import request_clearance_token, request_spartan_token
    from spnkr.auth.xbox import request_user_token, request_xsts_token

    url = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
    data = {
        "client_id": app.client_id,
        "client_secret": app.client_secret,
        "grant_type": "refresh_token",
        "refresh_token": refresh_token,
        "scope": "Xboxlive.signin Xboxlive.offline_access",
    }
    resp = await session.post(url, data=data)
    payload = await resp.json()

    if resp.status >= 400:
        raise ValueError(
            f"Échec refresh OAuth v2: status={resp.status} error={payload.get('error')}"
        )

    access_token = payload.get("access_token")
    if not access_token:
        raise ValueError("OAuth v2: pas de access_token")

    user_token = await request_user_token(session, access_token)
    _ = await request_xsts_token(session, user_token.token, XSTS_V3_XBOX_AUDIENCE)
    halo_xsts_token = await request_xsts_token(session, user_token.token, XSTS_V3_HALO_AUDIENCE)
    spartan_token = await request_spartan_token(session, halo_xsts_token.token)
    clearance_token = await request_clearance_token(session, spartan_token.token)

    return Tokens(
        spartan_token=str(spartan_token.token),
        clearance_token=str(clearance_token.token),
    )
