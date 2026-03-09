"""Logique d'authentification Xbox pour LevelUp — Device Code Flow.

Ce module est SANS dépendance Streamlit — toutes les fonctions sont pures
ou async, testables unitairement avec des mocks HTTP.

Flux Device Code Flow (MSAL) :
  1. ``msal_device_flow.initiate_device_flow()`` → code de 8 caractères
  2. Utilisateur visite microsoft.com/devicelogin et entre le code
  3. ``msal_device_flow.acquire_token_blocking()`` → refresh_token
  4. ``complete_device_code_flow()`` → gamertag + XUID (via API Halo)
  5. ``store_refresh_token()`` → persisté dans stats.duckdb/sync_meta

Persistance multi-joueurs (sans .env par joueur) :
  - ``store_refresh_token(db_path, refresh_token)``
  - ``load_refresh_token(db_path) → str | None``
"""

from __future__ import annotations

import asyncio
import concurrent.futures
import logging
from pathlib import Path

logger = logging.getLogger(__name__)

# Durée max (secondes) pour les appels réseau OAuth (résolution identité)
_OAUTH_TIMEOUT_S = 20


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

    Supporte les clients publics (``client_secret=""``  pour le Device Code Flow)
    et confidentiels.

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

    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(path) as conn:
            row = conn.execute(
                "SELECT value FROM sync_meta WHERE key = ?",
                (_SYNC_META_OAUTH_KEY,),
            ).fetchone()
            if row and row[0] and str(row[0]).strip():
                return str(row[0]).strip()
    except Exception as exc:
        logger.debug("Impossible de lire oauth_refresh_token depuis %s: %s", db_path, exc)

    return None


# =============================================================================
# Résolution identité + completion du Device Code Flow
# =============================================================================


def complete_device_code_flow(refresh_token: str, client_id: str) -> dict:
    """Complète le flux Device Code : résout l'identité Xbox depuis le refresh_token.

    À appeler après ``msal_device_flow.acquire_token_blocking()``.
    Obtient spartan/clearance tokens puis résout le gamertag et le XUID.

    Args:
        refresh_token: Token obtenu via MSAL Device Code Flow.
        client_id: Application (client) ID Azure (public client, sans secret).

    Returns:
        Dict avec ``gamertag``, ``xuid``, ``refresh_token``, ``spartan_token``,
        ``clearance_token`` en cas de succès, ou ``error`` (str) en cas d'échec.
    """

    async def _run() -> dict:
        try:
            # Client public : client_secret et redirect_uri vides
            spartan_token, clearance_token = await get_spartan_tokens_from_refresh(
                None,
                client_id=client_id,
                client_secret="",
                redirect_uri="",
                refresh_token=refresh_token,
            )
            gamertag, xuid = await resolve_player_identity(spartan_token, clearance_token)
            logger.info("Device Code Flow complet : gamertag=%s xuid=%s", gamertag, xuid)
            return {
                "gamertag": gamertag,
                "xuid": xuid,
                "refresh_token": refresh_token,
                "spartan_token": spartan_token,
                "clearance_token": clearance_token,
            }
        except Exception as exc:
            logger.exception("Erreur lors de la résolution identity (Device Code Flow)")
            return {"error": str(exc)}

    try:
        return asyncio.run(_run())
    except RuntimeError:
        with concurrent.futures.ThreadPoolExecutor(max_workers=1) as ex:
            fut = ex.submit(lambda: asyncio.run(_run()))
            return fut.result(timeout=float(_OAUTH_TIMEOUT_S) + 30)
