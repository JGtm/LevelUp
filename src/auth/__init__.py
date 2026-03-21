"""Couche d'authentification LevelUp — API publique.

Interface unique pour obtenir les tokens Halo Infinite d'un joueur.
Tout le code applicatif importe uniquement depuis ce package.

Usage rapide :
    from src.auth import get_halo_tokens, AuthRequiredError

    # CLI / scripts (Device Code Flow interactif si besoin)
    tokens = await get_halo_tokens(db_path)

    # Streamlit (raise si re-auth requise)
    try:
        tokens = await get_halo_tokens_or_raise(db_path)
    except AuthRequiredError:
        pending = start_device_flow(db_path)
        # afficher pending.user_code + pending.verification_url dans l'UI
        gamertag, xuid = await complete_device_flow(db_path, pending)
"""

from src.auth._constants import LEVELUP_CLIENT_ID
from src.auth._msal import DeviceCodeInfo, DeviceFlowError, MsalUnavailableError
from src.auth.provider import (
    AuthRequiredError,
    DeviceCodePending,
    complete_device_flow,
    get_halo_tokens,
    get_halo_tokens_or_raise,
    invalidate_token_cache,
    set_preferred_auth_method,
    start_device_flow,
)

__all__ = [
    # Interface principale
    "get_halo_tokens",
    "get_halo_tokens_or_raise",
    # Device Code Flow UI
    "start_device_flow",
    "complete_device_flow",
    "DeviceCodePending",
    # Gestion du cache
    "invalidate_token_cache",
    # Configuration auth
    "set_preferred_auth_method",
    # Exceptions
    "AuthRequiredError",
    "DeviceFlowError",
    "MsalUnavailableError",
    # Infos MSAL (lecture seule)
    "DeviceCodeInfo",
    # Constantes (accès direct si nécessaire)
    "LEVELUP_CLIENT_ID",
]
