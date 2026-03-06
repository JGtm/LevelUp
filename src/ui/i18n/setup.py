"""Traductions pour le wizard de configuration initiale."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str]] = {
    # ── Titres ──────────────────────────────────────────────────────────────
    "setup_title": {
        "fr": "⚙️ Configuration initiale",
        "en": "⚙️ Initial Setup",
    },
    "setup_welcome": {
        "fr": (
            "Bienvenue dans **LevelUp** ! Avant de pouvoir afficher vos stats "
            "Halo Infinite, quelques paramètres sont nécessaires."
        ),
        "en": (
            "Welcome to **LevelUp**! Before displaying your Halo Infinite "
            "stats, a few settings are needed."
        ),
    },
    # ── Étape 1 : Azure ────────────────────────────────────────────────────
    "setup_step1_title": {
        "fr": "1️⃣ Credentials Azure",
        "en": "1️⃣ Azure Credentials",
    },
    "setup_step1_help": {
        "fr": (
            "Pour accéder à l'API Halo Infinite, vous devez créer une application "
            "dans [Azure Portal](https://portal.azure.com/) → **App registrations**.\n\n"
            "📖 [Guide détaillé](docs/CONFIGURATION.md#azure-configuration)"
        ),
        "en": (
            "To access the Halo Infinite API, you need to create an application "
            "in [Azure Portal](https://portal.azure.com/) → **App registrations**.\n\n"
            "📖 [Detailed guide](docs/CONFIGURATION.md#azure-configuration)"
        ),
    },
    "setup_client_id": {
        "fr": "Application (Client) ID",
        "en": "Application (Client) ID",
    },
    "setup_client_secret": {
        "fr": "Client Secret (valeur)",
        "en": "Client Secret (value)",
    },
    "setup_redirect_uri": {
        "fr": "Redirect URI",
        "en": "Redirect URI",
    },
    "setup_save_credentials": {
        "fr": "💾 Sauvegarder les credentials",
        "en": "💾 Save credentials",
    },
    "setup_credentials_saved": {
        "fr": "✅ Credentials sauvegardées dans `.env.local`.",
        "en": "✅ Credentials saved to `.env.local`.",
    },
    "setup_credentials_ok": {
        "fr": "✅ Credentials Azure configurées.",
        "en": "✅ Azure credentials configured.",
    },
    # ── Étape 2 : Token ───────────────────────────────────────────────────
    "setup_step2_title": {
        "fr": "2️⃣ Token OAuth",
        "en": "2️⃣ OAuth Token",
    },
    "setup_step2_help": {
        "fr": (
            "Le refresh token permet à LevelUp d'accéder à l'API Halo "
            "sans redemander vos identifiants à chaque fois.\n\n"
            "Lancez la commande ci-dessous **dans un terminal**, "
            "suivez les instructions dans le navigateur, puis copiez le token affiché."
        ),
        "en": (
            "The refresh token allows LevelUp to access the Halo API "
            "without asking for your credentials each time.\n\n"
            "Run the command below **in a terminal**, "
            "follow the browser instructions, then copy the displayed token."
        ),
    },
    "setup_token_command": {
        "fr": "Commande à lancer dans un terminal :",
        "en": "Command to run in a terminal:",
    },
    "setup_token_paste": {
        "fr": "Coller le refresh token ici",
        "en": "Paste the refresh token here",
    },
    "setup_save_token": {
        "fr": "💾 Sauvegarder le token",
        "en": "💾 Save token",
    },
    "setup_token_saved": {
        "fr": "✅ Token sauvegardé dans `.env.local`.",
        "en": "✅ Token saved to `.env.local`.",
    },
    "setup_token_ok": {
        "fr": "✅ Token OAuth configuré.",
        "en": "✅ OAuth token configured.",
    },
    "setup_token_empty": {
        "fr": "⚠️ Le token est vide.",
        "en": "⚠️ The token is empty.",
    },
    # ── Étape 3 : Joueur ──────────────────────────────────────────────────
    "setup_step3_title": {
        "fr": "3️⃣ Ajouter un joueur",
        "en": "3️⃣ Add a player",
    },
    "setup_step3_help": {
        "fr": (
            "Entrez votre gamertag Xbox pour créer votre profil. "
            "La première synchronisation récupèrera votre historique de matchs."
        ),
        "en": (
            "Enter your Xbox gamertag to create your profile. "
            "The first sync will fetch your match history."
        ),
    },
    "setup_gamertag": {
        "fr": "Gamertag Xbox",
        "en": "Xbox Gamertag",
    },
    "setup_max_matches": {
        "fr": "Nombre de matchs à récupérer",
        "en": "Number of matches to fetch",
    },
    "setup_create_profile": {
        "fr": "👤 Créer le profil",
        "en": "👤 Create profile",
    },
    "setup_profile_created": {
        "fr": "✅ Profil créé pour **{gamertag}**.",
        "en": "✅ Profile created for **{gamertag}**.",
    },
    "setup_sync_instructions": {
        "fr": (
            "Profil créé ! Lancez la commande suivante dans un terminal "
            "pour récupérer vos matchs :"
        ),
        "en": (
            "Profile created! Run the following command in a terminal " "to fetch your matches:"
        ),
    },
    "setup_sync_done_hint": {
        "fr": (
            "Une fois la synchronisation terminée, **rechargez cette page** "
            "(F5) pour accéder au dashboard."
        ),
        "en": ("Once the sync is complete, **reload this page** " "(F5) to access the dashboard."),
    },
    # ── Général ───────────────────────────────────────────────────────────
    "setup_already_configured": {
        "fr": "✅ LevelUp est correctement configuré. Rechargez la page pour accéder au dashboard.",
        "en": "✅ LevelUp is properly configured. Reload the page to access the dashboard.",
    },
    "setup_reconfigure": {
        "fr": "🔧 Reconfigurer",
        "en": "🔧 Reconfigure",
    },
}
