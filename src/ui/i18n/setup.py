"""Traductions pour le wizard de configuration initiale."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str]] = {
    # ── En-tête ─────────────────────────────────────────────────────────────
    "setup_title": {
        "fr": "⚙️ Configuration initiale",
        "en": "⚙️ Initial Setup",
    },
    "setup_title_clean": {
        "fr": "Bienvenue dans LevelUp",
        "en": "Welcome to LevelUp",
    },
    "setup_subtitle": {
        "fr": "Analysez vos stats Halo Infinite. Choisissez une méthode de connexion pour commencer.",
        "en": "Analyze your Halo Infinite stats. Choose a login method to get started.",
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
    # ── Cartes de sélection de mode ────────────────────────────────────────
    "setup_xbox_card_title": {
        "fr": "Connexion Xbox",
        "en": "Xbox Login",
    },
    "setup_xbox_card_desc": {
        "fr": "Code à 8 chiffres sur microsoft.com/devicelogin. Pas de secret Azure, pas de Redirect URI.",
        "en": "8-character code at microsoft.com/devicelogin. No Azure secret, no Redirect URI.",
    },
    "setup_xbox_card_badge": {
        "fr": "✨ Recommandé — Le plus simple",
        "en": "✨ Recommended — Simplest",
    },
    "setup_xbox_card_btn": {
        "fr": "🎮 Choisir Xbox",
        "en": "🎮 Choose Xbox",
    },
    "setup_azure_card_title": {
        "fr": "Configuration Azure",
        "en": "Azure Setup",
    },
    "setup_azure_card_desc": {
        "fr": "Configurez manuellement votre App Azure et votre token. Pour les utilisateurs avancés.",
        "en": "Manually configure your Azure App and token. For advanced users.",
    },
    "setup_azure_card_badge": {
        "fr": "🔧 Avancé",
        "en": "🔧 Advanced",
    },
    "setup_azure_card_btn": {
        "fr": "☁️ Choisir Azure",
        "en": "☁️ Choose Azure",
    },
    "setup_footer_note": {
        "fr": "Les deux méthodes nécessitent une App Azure. La méthode Xbox (Device Code) est de loin la plus simple : pas de secret ni de Redirect URI.",
        "en": "Both methods require an Azure App. The Xbox (Device Code) method is by far the simplest: no secret or Redirect URI needed.",
    },
    # ── Parcours Xbox ──────────────────────────────────────────────────────
    "setup_xbox_step1_title": {
        "fr": "App Azure",
        "en": "Azure App",
    },
    "setup_xbox_step1_help": {
        "fr": (
            "Même avec la connexion Xbox, une **App Azure** est nécessaire (elle sert de passerelle OAuth).\n\n"
            "**Important** : dans Azure Portal → Authentication, ajoutez `http://localhost:8501` "
            "comme **Redirect URI** (type Web).\n\n"
            "📖 [Guide pas à pas avec captures d'écran](docs/CONFIGURATION.md#azure-configuration)"
        ),
        "en": (
            "Even with Xbox login, an **Azure App** is needed (it serves as the OAuth gateway).\n\n"
            "**Important**: in Azure Portal → Authentication, add `http://localhost:8501` "
            "as a **Redirect URI** (Web type).\n\n"
            "📖 [Step-by-step guide with screenshots](docs/CONFIGURATION.md#azure-configuration)"
        ),
    },
    "setup_xbox_step2_title": {
        "fr": "Connexion Xbox",
        "en": "Xbox Login",
    },
    "setup_xbox_step2_help": {
        "fr": (
            "Cliquez ci-dessous pour vous connecter avec votre compte Xbox/Microsoft.\n\n"
            "Votre gamertag et votre token seront récupérés **automatiquement** — "
            "pas besoin de copier/coller quoi que ce soit."
        ),
        "en": (
            "Click below to sign in with your Xbox/Microsoft account.\n\n"
            "Your gamertag and token will be retrieved **automatically** — "
            "no copy/paste needed."
        ),
    },
    "setup_xbox_connect_btn": {
        "fr": "🎮 Se connecter avec Xbox",
        "en": "🎮 Sign in with Xbox",
    },
    "setup_xbox_redirect_note": {
        "fr": "Après connexion, Microsoft vous redirigera vers `{redirect_uri}`.",
        "en": "After login, Microsoft will redirect you to `{redirect_uri}`.",
    },
    "setup_xbox_provisioned": {
        "fr": "🎉 Connexion réussie ! Profil créé pour **{gamertag}**.",
        "en": "🎉 Login successful! Profile created for **{gamertag}**.",
    },
    "setup_xbox_sync_hint": {
        "fr": (
            "Lancez cette commande dans un terminal pour récupérer vos matchs, "
            "puis rechargez la page (F5) :"
        ),
        "en": (
            "Run this command in a terminal to fetch your matches, " "then reload the page (F5):"
        ),
    },
    "setup_credentials_missing": {
        "fr": "⚠️ Complétez l'étape précédente d'abord.",
        "en": "⚠️ Complete the previous step first.",
    },
    "setup_back_btn": {
        "fr": "← Retour",
        "en": "← Back",
    },
    # ── Étape 1 : Azure (parcours classique) ───────────────────────────────
    "setup_step1_title": {
        "fr": "Credentials Azure",
        "en": "Azure Credentials",
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
        "fr": "💾 Sauvegarder",
        "en": "💾 Save",
    },
    "setup_credentials_saved": {
        "fr": "✅ Credentials sauvegardées dans `.env.local`.",
        "en": "✅ Credentials saved to `.env.local`.",
    },
    "setup_credentials_ok": {
        "fr": "✅ Credentials Azure configurées.",
        "en": "✅ Azure credentials configured.",
    },
    # ── Étape 2 : Token (parcours classique) ──────────────────────────────
    "setup_step2_title": {
        "fr": "Token OAuth",
        "en": "OAuth Token",
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
    # ── Étape 3 : Joueur (parcours classique) ─────────────────────────────
    "setup_step3_title": {
        "fr": "Ajouter un joueur",
        "en": "Add a player",
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
    # ── Smoke test post-installation ──────────────────────────────────────
    "smoke_title": {
        "fr": "Vérification de l'installation",
        "en": "Installation Verification",
    },
    "smoke_subtitle": {
        "fr": (
            "On va synchroniser **{count} matchs** pour **{gamertag}** "
            "et vérifier que tout fonctionne."
        ),
        "en": ("We'll sync **{count} matches** for **{gamertag}** " "and verify everything works."),
    },
    "smoke_start_btn": {
        "fr": "🚀 Lancer le test de vérification",
        "en": "🚀 Start verification test",
    },
    "smoke_info_before_start": {
        "fr": (
            "Ce test va :\n"
            "1. Synchroniser quelques matchs depuis l'API Halo\n"
            "2. Enrichir les données (scores, citations, ratings...)\n"
            "3. Vérifier l'intégrité de toutes les tables\n\n"
            "⏱️ Durée estimée : 1-2 minutes"
        ),
        "en": (
            "This test will:\n"
            "1. Sync a few matches from the Halo API\n"
            "2. Enrich data (scores, citations, ratings...)\n"
            "3. Verify all tables integrity\n\n"
            "⏱️ Estimated time: 1-2 minutes"
        ),
    },
    # Phase 1 : Sync
    "smoke_phase1_title": {
        "fr": "📡 Phase 1 — Synchronisation",
        "en": "📡 Phase 1 — Synchronization",
    },
    "smoke_phase1_connecting": {
        "fr": "Connexion à l'API Halo Infinite...",
        "en": "Connecting to Halo Infinite API...",
    },
    "smoke_phase1_fetching": {
        "fr": "Récupération de {count} matchs...",
        "en": "Fetching {count} matches...",
    },
    "smoke_phase1_done": {
        "fr": "Sync terminée en {seconds}s",
        "en": "Sync completed in {seconds}s",
    },
    "smoke_phase1_failed": {
        "fr": "Échec de la synchronisation",
        "en": "Synchronization failed",
    },
    # Phase 2 : Backfill
    "smoke_phase2_title": {
        "fr": "⚙️ Phase 2 — Enrichissement des données",
        "en": "⚙️ Phase 2 — Data enrichment",
    },
    "smoke_phase2_running": {
        "fr": "Calcul des scores, sessions, citations, ratings...",
        "en": "Computing scores, sessions, citations, ratings...",
    },
    "smoke_phase2_done": {
        "fr": "Enrichissements terminés",
        "en": "Enrichments completed",
    },
    "smoke_phase2_partial": {
        "fr": "Enrichissements partiels",
        "en": "Partial enrichments",
    },
    # Phase 3 : Vérification
    "smoke_phase3_title": {
        "fr": "🔍 Phase 3 — Vérification d'intégrité",
        "en": "🔍 Phase 3 — Integrity verification",
    },
    "smoke_phase3_checking": {
        "fr": "Vérification de toutes les tables...",
        "en": "Checking all tables...",
    },
    "smoke_phase3_done": {
        "fr": "Vérification terminée",
        "en": "Verification completed",
    },
    # Résultats
    "smoke_results_title": {
        "fr": "Rapport de vérification",
        "en": "Verification Report",
    },
    "smoke_result_sync_failed": {
        "fr": (
            "La synchronisation a échoué : **{message}**\n\n"
            "Vérifiez vos credentials Azure et votre token OAuth."
        ),
        "en": (
            "Synchronization failed: **{message}**\n\n"
            "Check your Azure credentials and OAuth token."
        ),
    },
    "smoke_result_all_ok": {
        "fr": "🎉 Tout est parfait ! **{passed}/{total}** vérifications réussies.",
        "en": "🎉 Everything is perfect! **{passed}/{total}** checks passed.",
    },
    "smoke_result_ok_with_warnings": {
        "fr": (
            "✅ Installation fonctionnelle — **{passed}/{total}** vérifications réussies, "
            "**{warnings}** avertissement(s) non bloquant(s)."
        ),
        "en": (
            "✅ Installation functional — **{passed}/{total}** checks passed, "
            "**{warnings}** non-blocking warning(s)."
        ),
    },
    "smoke_result_issues": {
        "fr": (
            "⚠️ **{failed}** vérification(s) en échec sur **{total}**. "
            "Consultez le détail ci-dessous."
        ),
        "en": ("⚠️ **{failed}** check(s) failed out of **{total}**. " "See details below."),
    },
    # Actions post-test
    "smoke_next_steps": {
        "fr": "Prochaines étapes",
        "en": "Next steps",
    },
    "smoke_next_steps_info": {
        "fr": (
            "L'installation est validée sur 20 matchs test !\n\n"
            "- **Sync complète (recommandé)** → récupère tout votre historique depuis l'API Halo "
            "(peut prendre plusieurs minutes)\n"
            "- **Dashboard** → consulter vos 20 matchs déjà synchronisés"
        ),
        "en": (
            "Installation validated on 20 test matches!\n\n"
            "- **Full sync (recommended)** → fetch your complete match history from the Halo API "
            "(may take several minutes)\n"
            "- **Dashboard** → browse your 20 already synced matches"
        ),
    },
    "smoke_btn_full_sync": {
        "fr": "⚙️ Sync complète — récupérer tout l'historique",
        "en": "⚙️ Full sync — fetch complete history",
    },
    "smoke_btn_dashboard": {
        "fr": "📊 Dashboard (20 matchs seulement)",
        "en": "📊 Dashboard (20 matches only)",
    },
    "smoke_btn_retry": {
        "fr": "🔄 Relancer le test",
        "en": "🔄 Retry test",
    },
    # ── Parcours Xbox — Device Code Flow ────────────────────────────────────────────
    "setup_dc_step1_title": {
        "fr": "Application (client) ID Azure",
        "en": "Azure Application (client) ID",
    },
    "setup_dc_step1_help": {
        "fr": (
            "Créez une **App Azure Public Client** (à faire une seule fois) :\n\n"
            "1. [portal.azure.com](https://portal.azure.com) → **App registrations** → **New registration**\n"
            "2. **Name** : LevelUp (ou autre)\n"
            "3. **Supported account types** : *Personal Microsoft accounts only*\n"
            "4. **Redirect URI** : laisser vide\n"
            "5. Cliquer **Register**\n"
            "6. Aller dans **Authentication** → *Allow public client flows* → **Yes** → **Save**\n"
            "7. Copier l'**Application (client) ID** ci-dessous."
        ),
        "en": (
            "Create an **Azure Public Client App** (one time only):\n\n"
            "1. [portal.azure.com](https://portal.azure.com) → **App registrations** → **New registration**\n"
            "2. **Name**: LevelUp (or any name)\n"
            "3. **Supported account types**: *Personal Microsoft accounts only*\n"
            "4. **Redirect URI**: leave blank\n"
            "5. Click **Register**\n"
            "6. Go to **Authentication** → *Allow public client flows* → **Yes** → **Save**\n"
            "7. Copy the **Application (client) ID** below."
        ),
    },
    "setup_dc_client_id_label": {
        "fr": "Application (client) ID",
        "en": "Application (client) ID",
    },
    "setup_dc_start_btn": {
        "fr": "🔑 Générer le code de connexion",
        "en": "🔑 Generate login code",
    },
    "setup_dc_client_id_empty": {
        "fr": "Veuillez saisir l'Application (client) ID.",
        "en": "Please enter the Application (client) ID.",
    },
    "setup_dc_code_title": {
        "fr": "Entrez ce code sur [microsoft.com/devicelogin](https://microsoft.com/devicelogin)",
        "en": "Enter this code at [microsoft.com/devicelogin](https://microsoft.com/devicelogin)",
    },
    "setup_dc_verify_btn": {
        "fr": "✅ J'ai confirmé — Continuer",
        "en": "✅ I confirmed — Continue",
    },
    "setup_dc_waiting": {
        "fr": "⏳ Code pas encore confirmé. Retournez sur microsoft.com/devicelogin.",
        "en": "⏳ Code not confirmed yet. Go back to microsoft.com/devicelogin.",
    },
    "setup_dc_cancel_btn": {
        "fr": "← Recommencer",
        "en": "← Start over",
    },
    "setup_dc_getting_profile": {
        "fr": "Récupération du profil Xbox…",
        "en": "Fetching Xbox profile…",
    },
}
