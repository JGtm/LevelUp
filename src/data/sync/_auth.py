"""Facade d'authentification Halo — API agnostique.

Ce module expose des fonctions d'authentification que l'UI et les scripts
peuvent appeler **sans importer directement spnkr**. L'implémentation
interne délègue à ``_tokens.py`` (source de vérité OAuth).

Les consommateurs UI importent :
    from src.data.sync._auth import refresh_halo_tokens
au lieu de :
    from spnkr import AzureApp, refresh_player_tokens  # interdit dans l'UI
"""

from __future__ import annotations

import os
from typing import Any

from src.data.sync._tokens import Tokens
from src.utils.env import load_dotenv_if_present


async def refresh_halo_tokens(
    session: Any,
    *,
    client_id: str,
    client_secret: str,
    redirect_uri: str = "https://localhost",
    refresh_token: str,
) -> Tokens:
    """Obtient des tokens Spartan + Clearance via OAuth Azure.

    Encapsule la logique SPNKr OAuth (avec fallback v2) sans exposer les
    types SPNKr aux appelants.

    Args:
        session: Session aiohttp active.
        client_id: Azure Application (client) ID.
        client_secret: Azure client secret.
        redirect_uri: URI de redirection OAuth.
        refresh_token: Refresh token OAuth.

    Returns:
        Tokens (spartan_token, clearance_token).

    Raises:
        ImportError: Si les dépendances SPNKr/aiohttp sont manquantes.
        ValueError: Si le refresh échoue.
    """
    from src.data.sync._tokens import _get_tokens_via_oauth

    return await _get_tokens_via_oauth(client_id, client_secret, redirect_uri, refresh_token)


async def refresh_halo_tokens_from_env(
    session: Any,
    *,
    gamertag: str | None = None,
) -> Tokens:
    """Obtient des tokens en résolvant les credentials depuis l'environnement.

    Cherche (dans l'ordre) :
    1. Token per-player ``SPNKR_OAUTH_REFRESH_TOKEN_<GT>`` si gamertag fourni
    2. Token global ``SPNKR_OAUTH_REFRESH_TOKEN``
    3. Tokens manuels ``SPNKR_SPARTAN_TOKEN`` + ``SPNKR_CLEARANCE_TOKEN``

    Args:
        session: Session aiohttp active.
        gamertag: Gamertag du joueur (optionnel, pour token per-player).

    Returns:
        Tokens valides.

    Raises:
        RuntimeError: Si aucun moyen d'authentification n'est configuré.
    """
    load_dotenv_if_present()

    # Mode tokens manuels
    st = (os.environ.get("SPNKR_SPARTAN_TOKEN") or "").strip()
    ct = (os.environ.get("SPNKR_CLEARANCE_TOKEN") or "").strip()
    if st and ct:
        return Tokens(spartan_token=st, clearance_token=ct)

    # Mode OAuth
    client_id = (os.environ.get("SPNKR_AZURE_CLIENT_ID") or "").strip()
    client_secret = (os.environ.get("SPNKR_AZURE_CLIENT_SECRET") or "").strip()
    redirect_uri = (os.environ.get("SPNKR_AZURE_REDIRECT_URI") or "").strip() or "https://localhost"

    # Chercher le refresh token
    rt = ""
    if gamertag:
        from src.utils.env import normalize_gamertag_for_env

        player_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{normalize_gamertag_for_env(gamertag)}"
        rt = (os.environ.get(player_key) or "").strip()
    if not rt:
        rt = (os.environ.get("SPNKR_OAUTH_REFRESH_TOKEN") or "").strip()

    if not (client_id and client_secret and rt):
        raise RuntimeError(
            "Tokens manquants : configurer SPNKR_SPARTAN_TOKEN + SPNKR_CLEARANCE_TOKEN, "
            "ou SPNKR_AZURE_CLIENT_ID + SPNKR_AZURE_CLIENT_SECRET + SPNKR_OAUTH_REFRESH_TOKEN."
        )

    tokens = await refresh_halo_tokens(
        session,
        client_id=client_id,
        client_secret=client_secret,
        redirect_uri=redirect_uri,
        refresh_token=rt,
    )

    # Rendre les tokens accessibles aux helpers (téléchargement d'assets, etc.)
    os.environ["SPNKR_SPARTAN_TOKEN"] = tokens.spartan_token
    os.environ["SPNKR_CLEARANCE_TOKEN"] = tokens.clearance_token

    return tokens
