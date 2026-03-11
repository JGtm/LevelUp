"""Chaînes i18n — Xbox Device Code Flow.

Ce module est un sous-module du package ``src.ui.i18n.pages``.
"""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    # ── Connexion Xbox (Device Code Flow) ────────────────────────────────────
    "xbox_connect_section_title": {
        "fr": "🎮 Connexion Xbox",
        "en": "🎮 Xbox Login",
    },
    "xbox_connected_as": {
        "fr": "Connecté en tant que {gamertag}",
        "en": "Signed in as {gamertag}",
    },
    "xbox_disconnect": {
        "fr": "Déconnecter",
        "en": "Disconnect",
    },
    "xbox_token_stored": {
        "fr": "Token stocké localement (déconnexion supprime uniquement le token local).",
        "en": "Token stored locally (disconnect only removes the local token).",
    },
    "xbox_auth_success": {
        "fr": "✅ Connexion Xbox réussie ! Bienvenue, {gamertag}. Profil créé.",
        "en": "✅ Xbox login successful! Welcome, {gamertag}. Profile created.",
    },
    "xbox_auth_error": {
        "fr": "❌ Erreur lors de la connexion Xbox :",
        "en": "❌ Xbox login error:",
    },
    "xbox_auth_intro": {
        "fr": "Connectez votre compte Xbox pour synchroniser vos statistiques Halo automatiquement.",
        "en": "Connect your Xbox account to automatically sync your Halo stats.",
    },
    # ── Formulaire Device Code (widget Settings) ──────────────────────────────
    "xbox_dc_client_id_label": {
        "fr": "Application (client) ID Azure",
        "en": "Azure Application (client) ID",
    },
    "xbox_dc_client_id_help": {
        "fr": "Client Public Azure : uniquement le Client ID est requis (pas de secret, pas de Redirect URI).",
        "en": "Azure Public Client: only the Client ID is required (no secret, no Redirect URI).",
    },
    "xbox_dc_client_id_empty": {
        "fr": "Veuillez saisir l'Application (client) ID.",
        "en": "Please enter the Application (client) ID.",
    },
    "xbox_dc_start_btn": {
        "fr": "🔑 Générer le code",
        "en": "🔑 Generate code",
    },
    "xbox_dc_code_title": {
        "fr": "Code à entrer sur",
        "en": "Code to enter at",
    },
    "xbox_dc_verify_btn": {
        "fr": "✅ Vérifier la connexion",
        "en": "✅ Check connection",
    },
    "xbox_dc_waiting": {
        "fr": "Code pas encore confirmé. Retournez sur microsoft.com/devicelogin.",
        "en": "Code not confirmed yet. Go back to microsoft.com/devicelogin.",
    },
    "xbox_dc_cancel_btn": {
        "fr": "Annuler",
        "en": "Cancel",
    },
    "xbox_dc_token_ready": {
        "fr": "✅ Connexion confirmée ! Token obtenu.",
        "en": "✅ Connection confirmed! Token obtained.",
    },
}

__all__ = ["STRINGS"]
