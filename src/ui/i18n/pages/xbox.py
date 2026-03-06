"""Chaînes i18n — Xbox OAuth (connexion compte Xbox).

Ce module est un sous-module du package ``src.ui.i18n.pages``.
Il sera enregistré dans ``__init__.py`` lors du merge dans refactor/cleanup-all :

    from src.ui.i18n.pages import xbox
    # dans la boucle :
    for _mod in (..., xbox):
        STRINGS.update(_mod.STRINGS)
"""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    # ── Xbox OAuth (connexion compte Xbox) ────────────────────────────────────
    "xbox_connect_section_title": {
        "fr": "🎮 Connexion Xbox",
        "en": "🎮 Xbox Login",
    },
    "xbox_connect_btn": {
        "fr": "🎮 Se connecter avec Xbox",
        "en": "🎮 Sign in with Xbox",
    },
    "xbox_connect_help": {
        "fr": "Ouvre la page de connexion Microsoft dans le navigateur. Vous serez redirigé vers l'app après authentification.",
        "en": "Opens the Microsoft login page in your browser. You will be redirected back to the app after authentication.",
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
    "xbox_redirect_notice": {
        "fr": "Après connexion, Microsoft vous redirigera vers `{redirect_uri}`. Assurez-vous que cette URL est bien configurée dans Azure Portal.",
        "en": "After login, Microsoft will redirect you to `{redirect_uri}`. Make sure this URL is registered in Azure Portal.",
    },
    "xbox_auth_missing_config": {
        "fr": "⚠️ Configuration Azure manquante. Définissez `SPNKR_AZURE_CLIENT_ID` et `SPNKR_AZURE_CLIENT_SECRET` dans `.env.local`.",
        "en": "⚠️ Azure configuration missing. Set `SPNKR_AZURE_CLIENT_ID` and `SPNKR_AZURE_CLIENT_SECRET` in `.env.local`.",
    },
    "xbox_auth_setup_help_title": {
        "fr": "Comment configurer Azure ?",
        "en": "How to configure Azure?",
    },
    "xbox_auth_setup_help_body": {
        "fr": (
            "1. Créer une **App Registration** sur [Azure Portal](https://portal.azure.com)\n"
            "2. Dans **Authentication** → ajouter `http://localhost:8501` (et l'URL de votre NAS si applicable) comme **Redirect URI**\n"
            "3. Copier le **Client ID** → `SPNKR_AZURE_CLIENT_ID=...` dans `.env.local`\n"
            "4. Créer un **Client Secret** → `SPNKR_AZURE_CLIENT_SECRET=...` dans `.env.local`\n"
            "5. Définir `SPNKR_AZURE_REDIRECT_URI=http://localhost:8501` dans `.env.local`"
        ),
        "en": (
            "1. Create an **App Registration** on [Azure Portal](https://portal.azure.com)\n"
            "2. Under **Authentication** → add `http://localhost:8501` (and your NAS URL if applicable) as **Redirect URI**\n"
            "3. Copy the **Client ID** → `SPNKR_AZURE_CLIENT_ID=...` in `.env.local`\n"
            "4. Create a **Client Secret** → `SPNKR_AZURE_CLIENT_SECRET=...` in `.env.local`\n"
            "5. Set `SPNKR_AZURE_REDIRECT_URI=http://localhost:8501` in `.env.local`"
        ),
    },
}

__all__ = ["STRINGS"]
