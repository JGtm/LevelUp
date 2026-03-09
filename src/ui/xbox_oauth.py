"""Logique d'authentification Xbox OAuth pour LevelUp.

Ce module est SANS dépendance Streamlit — toutes les fonctions sont pures
ou async, testables unitairement avec des mocks HTTP.

Flux OAuth Microsoft supporté :
  1. ``build_xbox_auth_url()`` → URL Microsoft → ouvert dans le navigateur
  2. Microsoft redirige vers ``<redirect_uri>?code=XXX&state=YYY``
  3. ``exchange_code_for_refresh_token()`` → refresh_token longue durée
  4. ``get_spartan_tokens_from_refresh()`` → spartan_token + clearance_token
  5. ``resolve_player_identity()`` → gamertag + XUID
  6. ``store_refresh_token()`` → persisté dans stats.duckdb/sync_meta

Gestion des tokens stockés en DB (multi-joueurs, sans .env par joueur) :
  - ``store_refresh_token(db_path, refresh_token)``
  - ``load_refresh_token(db_path) → str | None``
"""

from __future__ import annotations

import asyncio
import concurrent.futures
import logging
import secrets
from pathlib import Path
from urllib.parse import quote_plus

logger = logging.getLogger(__name__)

# Scopes Microsoft pour Xbox Live
_XBOX_SCOPES = "Xboxlive.signin Xboxlive.offline_access"

# Durée max (secondes) pour les appels réseau OAuth
_OAUTH_TIMEOUT_S = 20


# =============================================================================
# Construction de l'URL d'autorisation
# =============================================================================


def build_xbox_auth_url(client_id: str, redirect_uri: str, state: str) -> str:
    """Construit l'URL d'autorisation Microsoft/Xbox.

    L'utilisateur est redirigé vers cette URL pour se connecter avec son
    compte Microsoft. Après authentification, Microsoft le renvoie vers
    ``redirect_uri?code=XXX&state=YYY``.

    Args:
        client_id: Client ID de l'app Azure (SPNKR_AZURE_CLIENT_ID).
        redirect_uri: URI de redirection enregistrée dans Azure Portal.
        state: Valeur CSRF aléatoire générée par ``generate_oauth_state()``.

    Returns:
        URL complète à ouvrir dans le navigateur.
    """
    return (
        "https://login.live.com/oauth20_authorize.srf"
        f"?client_id={quote_plus(client_id)}"
        "&response_type=code"
        "&approval_prompt=auto"
        f"&scope={quote_plus(_XBOX_SCOPES)}"
        f"&redirect_uri={quote_plus(redirect_uri)}"
        f"&state={quote_plus(state)}"
    )


def generate_oauth_state() -> str:
    """Génère un token CSRF aléatoire pour la protection contre les attaques CSRF.

    Returns:
        Chaîne hexadécimale de 32 caractères.
    """
    return secrets.token_hex(16)


# =============================================================================
# Échange du code OAuth contre un refresh_token
# =============================================================================


async def exchange_code_for_refresh_token(
    session,
    *,
    client_id: str,
    client_secret: str,
    redirect_uri: str,
    code: str,
) -> str:
    """Échange le code OAuth contre un refresh_token longue durée.

    Utilise l'endpoint Microsoft identity platform v2 (consumers).

    Args:
        session: Session aiohttp active.
        client_id: Client ID Azure.
        client_secret: Client Secret Azure.
        redirect_uri: Doit correspondre exactement à la valeur enregistrée.
        code: Code reçu dans le query param ``code=`` du callback.

    Returns:
        Le refresh_token Microsoft.

    Raises:
        ValueError: Si la réponse ne contient pas de refresh_token ou en cas d'erreur.
    """
    url = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
    data = {
        "client_id": client_id,
        "client_secret": client_secret,
        "grant_type": "authorization_code",
        "code": code,
        "redirect_uri": redirect_uri,
        "scope": _XBOX_SCOPES,
    }

    resp = await session.post(url, data=data)
    payload = await resp.json(content_type=None)

    if resp.status >= 400:
        err = payload.get("error", "unknown_error")
        desc = payload.get("error_description", "")
        raise ValueError(f"Échec OAuth code→token (status={resp.status}): {err} — {desc}")

    refresh_token = payload.get("refresh_token")
    if not isinstance(refresh_token, str) or not refresh_token.strip():
        raise ValueError(
            "Réponse OAuth ne contient pas de refresh_token. "
            "Vérifiez que le scope 'Xboxlive.offline_access' est activé."
        )

    return refresh_token.strip()


# =============================================================================
# Obtention des tokens Halo depuis le refresh_token
# =============================================================================


async def get_spartan_tokens_from_refresh(
    session,
    *,
    client_id: str,
    client_secret: str,
    redirect_uri: str,
    refresh_token: str,
) -> tuple[str, str]:
    """Obtient spartan_token + clearance_token depuis un refresh_token.

    Essaie d'abord via ``spnkr.refresh_player_tokens``, puis fallback
    sur le flux OAuth v2 manuel (nécessaire avec certaines versions).

    Args:
        session: Session aiohttp active.
        client_id: Client ID Azure.
        client_secret: Client Secret Azure.
        redirect_uri: URI de redirection enregistrée.
        refresh_token: Token de rafraîchissement OAuth.

    Returns:
        Tuple ``(spartan_token, clearance_token)``.

    Raises:
        ImportError: Si les dépendances auth sont manquantes.
        ValueError: En cas d'échec d'authentification.
    """
    from src.data.sync._auth import refresh_halo_tokens

    tokens = await refresh_halo_tokens(
        session,
        client_id=client_id,
        client_secret=client_secret,
        redirect_uri=redirect_uri,
        refresh_token=refresh_token,
    )
    return tokens.spartan_token, tokens.clearance_token


# =============================================================================
# Résolution de l'identité joueur (gamertag + XUID)
# =============================================================================


async def resolve_player_identity(
    spartan_token: str,
    clearance_token: str,
) -> tuple[str, str]:
    """Résout le gamertag et le XUID du joueur depuis ses tokens Halo.

    Appelle l'API Halo Waypoint (endpoint ``/hi/profiles``) via SPNKr
    pour obtenir le profil du joueur authentifié.

    Args:
        spartan_token: Token Spartan (Halo Waypoint).
        clearance_token: Token Clearance (Halo Waypoint).

    Returns:
        Tuple ``(gamertag, xuid)``.

    Raises:
        ValueError: Si le profil ne peut pas être résolu.
    """
    try:
        import aiohttp
        from spnkr import HaloInfiniteClient
    except ImportError as e:
        raise ImportError("La librairie SPNKr est requise : pip install spnkr") from e

    timeout = aiohttp.ClientTimeout(total=float(_OAUTH_TIMEOUT_S))
    async with aiohttp.ClientSession(timeout=timeout) as session:
        client = HaloInfiniteClient(
            session=session,
            spartan_token=spartan_token,
            clearance_token=clearance_token,
            requests_per_second=5,
        )

        # Endpoint profil (gamertag + xuid du joueur courant)
        try:
            resp = await client.profile.get_current_user()
            data = await resp.json()
            xuid = str(data.get("xuid") or data.get("Xuid") or "").strip()
            gamertag = str(data.get("gamertag") or data.get("Gamertag") or "").strip()
            if xuid and gamertag:
                return gamertag, xuid
        except Exception as api_err:
            logger.debug("Échec get_current_user: %s", api_err)

        # Fallback : endpoint GET /hi/profiles/gamertags
        try:
            resp = await client.profile.get_current_player()
            data = await resp.json()
            xuid = str(data.get("xuid") or data.get("Xuid") or "").strip()
            gamertag = str(data.get("gamertag") or data.get("Gamertag") or "").strip()
            if xuid and gamertag:
                return gamertag, xuid
        except Exception as api_err2:
            logger.debug("Échec get_current_player: %s", api_err2)

        raise ValueError(
            "Impossible de résoudre le gamertag/XUID depuis l'API Halo. "
            "Tokens potentiellement invalides ou API indisponible."
        )


# =============================================================================
# Persistance du refresh_token dans stats.duckdb/sync_meta
# =============================================================================

_SYNC_META_OAUTH_KEY = "oauth_refresh_token"  # pragma: allowlist secret

# Cache process-level : évite N ouvertures DuckDB par session (185×/session observé).
# Invalidé dès qu'un nouveau token est écrit via store_refresh_token().
_rt_cache: dict[str, str | None] = {}

_SYNC_META_DDL = """
CREATE TABLE IF NOT EXISTS sync_meta (
    key VARCHAR PRIMARY KEY,
    value VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
"""


def store_refresh_token(db_path: str | Path, refresh_token: str) -> None:
    """Persiste le refresh_token OAuth dans la table sync_meta du joueur.

    Crée la table sync_meta si elle n'existe pas (DB à peine créée).

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        refresh_token: Token de rafraîchissement à stocker.
    """
    from src.utils.db import duckdb_read_write

    with duckdb_read_write(db_path) as conn:
        conn.execute(_SYNC_META_DDL)
        conn.execute(
            """INSERT OR REPLACE INTO sync_meta (key, value, updated_at)
               VALUES (?, ?, CURRENT_TIMESTAMP)""",
            (_SYNC_META_OAUTH_KEY, refresh_token),
        )
    _rt_cache[str(Path(db_path).resolve())] = refresh_token.strip()


def load_refresh_token(db_path: str | Path) -> str | None:
    """Lit le refresh_token OAuth depuis la table sync_meta du joueur.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.

    Returns:
        Le refresh_token si présent, ``None`` sinon.
    """
    path = Path(db_path)
    if not path.exists():
        return None

    cache_key = str(path.resolve())
    if cache_key in _rt_cache:
        return _rt_cache[cache_key]

    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(path) as conn:
            row = conn.execute(
                "SELECT value FROM sync_meta WHERE key = ?",
                (_SYNC_META_OAUTH_KEY,),
            ).fetchone()
            if row and row[0] and str(row[0]).strip():
                result = str(row[0]).strip()
                _rt_cache[cache_key] = result
                return result
    except Exception as exc:
        logger.debug("Impossible de lire oauth_refresh_token depuis %s: %s", db_path, exc)

    _rt_cache[cache_key] = None
    return None


# =============================================================================
# Point d'entrée synchrone (compatible Streamlit ThreadPoolExecutor)
# =============================================================================


def run_xbox_oauth_callback(
    code: str,
    *,
    client_id: str,
    client_secret: str,
    redirect_uri: str,
) -> dict:
    """Exécute le flux OAuth de façon synchrone (compatible Streamlit).

    Appelé depuis ``ThreadPoolExecutor`` dans ``streamlit_app.py``
    après réception du callback ``?code=XXX``.

    Args:
        code: Code OAuth reçu dans ``st.query_params["code"]``.
        client_id: SPNKR_AZURE_CLIENT_ID.
        client_secret: SPNKR_AZURE_CLIENT_SECRET.
        redirect_uri: SPNKR_AZURE_REDIRECT_URI.

    Returns:
        Dict avec les clés ``gamertag``, ``xuid``, ``refresh_token``
        en cas de succès, ou ``error`` en cas d'échec.
    """

    async def _run() -> dict:
        try:
            import aiohttp

            timeout = aiohttp.ClientTimeout(total=float(_OAUTH_TIMEOUT_S))
            async with aiohttp.ClientSession(timeout=timeout) as session:
                # 1. Échanger le code contre un refresh_token
                refresh_token = await exchange_code_for_refresh_token(
                    session,
                    client_id=client_id,
                    client_secret=client_secret,
                    redirect_uri=redirect_uri,
                    code=code,
                )

                # 2. Obtenir les tokens Halo
                spartan_token, clearance_token = await get_spartan_tokens_from_refresh(
                    session,
                    client_id=client_id,
                    client_secret=client_secret,
                    redirect_uri=redirect_uri,
                    refresh_token=refresh_token,
                )

            # 3. Résoudre l'identité (session séparée pour le client Halo)
            gamertag, xuid = await resolve_player_identity(spartan_token, clearance_token)

            return {
                "gamertag": gamertag,
                "xuid": xuid,
                "refresh_token": refresh_token,
                "spartan_token": spartan_token,
                "clearance_token": clearance_token,
            }
        except Exception as exc:
            logger.exception("Erreur lors du callback Xbox OAuth")
            return {"error": str(exc)}

    # Exécution compatible avec une boucle asyncio déjà en cours
    try:
        return asyncio.run(_run())
    except RuntimeError:
        with concurrent.futures.ThreadPoolExecutor(max_workers=1) as ex:
            fut = ex.submit(lambda: asyncio.run(_run()))
            return fut.result(timeout=float(_OAUTH_TIMEOUT_S) + 30)
