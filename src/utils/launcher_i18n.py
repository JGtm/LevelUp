"""Traductions FR/EN pour launcher.py.

Usage:
    from src.utils.launcher_i18n import t as _t
    print(_t("setup_done", lang))
    print(_t("sync_new_matches", lang, n=5))
"""

from __future__ import annotations

# Clé → {lang: texte}. Utilise .format(**kwargs) si kwargs sont fournis.
STRINGS: dict[str, dict[str, str]] = {
    # ── Général ──────────────────────────────────────────────────────────────
    "yes": {
        "fr": "oui",
        "en": "yes",
    },
    "no": {
        "fr": "non",
        "en": "no",
    },
    # ── Signal handler ────────────────────────────────────────────────────────
    "shutdown_in_progress": {
        "fr": "\n⏹ Arrêt en cours (Ctrl+C à nouveau pour forcer)...",
        "en": "\n⏹ Shutting down (Ctrl+C again to force)...",
    },
    "shutdown_forced": {
        "fr": "\n⚠ Arrêt forcé.",
        "en": "\n⚠ Forced shutdown.",
    },
    # ── _require_module ───────────────────────────────────────────────────────
    "dep_missing": {
        "fr": "Dépendance manquante: {name}",
        "en": "Missing dependency: {name}",
    },
    "dep_detail": {
        "fr": "Détail:",
        "en": "Detail:",
    },
    "dep_install_hint": {
        "fr": "Installe-la puis relance:",
        "en": "Install it then retry:",
    },
    # ── _import_duckdb ────────────────────────────────────────────────────────
    "duckdb_not_installed": {
        "fr": "❌ DuckDB non installé. Exécute:",
        "en": "❌ DuckDB not installed. Run:",
    },
    # ── _list_players ─────────────────────────────────────────────────────────
    "db_unreadable": {
        "fr": "  ⚠ {gamertag}/stats.duckdb illisible (fichier corrompu ?)",
        "en": "  ⚠ {gamertag}/stats.duckdb unreadable (corrupted file?)",
    },
    # ── _classify_sync_error ──────────────────────────────────────────────────
    "sync_error_token": {
        "fr": (
            "  ⚠ Token OAuth expiré pour {gamertag}\n"
            "  → Relance LevelUp puis choisis « Ajouter un joueur » pour renouveler le token."
        ),
        "en": (
            "  ⚠ OAuth token expired for {gamertag}\n"
            "  → Relaunch LevelUp and choose 'Add Player' to renew the token."
        ),
    },
    "sync_error_locked": {
        "fr": (
            "  ⚠ Base de données verrouillée.\n"
            "  → Ferme le dashboard LevelUp (Streamlit) avant de synchroniser."
        ),
        "en": (
            "  ⚠ Database is locked.\n  → Close the LevelUp dashboard (Streamlit) before syncing."
        ),
    },
    "sync_error_network": {
        "fr": (
            "  ⚠ Impossible de joindre les serveurs Halo.\n"
            "  → Vérifie ta connexion internet et réessaie."
        ),
        "en": ("  ⚠ Cannot reach Halo servers.\n  → Check your internet connection and try again."),
    },
    "sync_error_unknown": {
        "fr": (
            "  ⚠ Erreur inattendue pour {gamertag} :\n"
            "  {err}\n"
            "  → Pour obtenir de l'aide, transmettre ce message complet."
        ),
        "en": (
            "  ⚠ Unexpected error for {gamertag}:\n"
            "  {err}\n"
            "  → Share this full message to get help."
        ),
    },
    # ── _fetch_profile_assets ─────────────────────────────────────────────────
    "fetch_profile_assets": {
        "fr": "  → Fetch assets profil...",
        "en": "  → Fetching profile assets...",
    },
    # ── _install_python_via_winget ────────────────────────────────────────────
    "winget_windows_only": {
        "fr": "  ⚠ Installation automatique uniquement disponible sur Windows.",
        "en": "  ⚠ Automatic installation only available on Windows.",
    },
    "python_manual_install": {
        "fr": "  Installez Python manuellement: https://www.python.org/downloads/",
        "en": "  Install Python manually: https://www.python.org/downloads/",
    },
    "winget_unavailable": {
        "fr": "  ⚠ winget non disponible sur ce système.",
        "en": "  ⚠ winget is not available on this system.",
    },
    "winget_installing": {
        "fr": "  → Installation de Python 3.12 via winget...",
        "en": "  → Installing Python 3.12 via winget...",
    },
    "winget_error": {
        "fr": "  ⚠ Erreur winget: {err}",
        "en": "  ⚠ winget error: {err}",
    },
    # ── _cmd_setup ────────────────────────────────────────────────────────────
    "setup_title": {
        "fr": "⚙️  LEVELUP — SETUP",
        "en": "⚙️  LEVELUP — SETUP",
    },
    "setup_title_update": {
        "fr": "⚙️  LEVELUP — SETUP (mise à jour)",
        "en": "⚙️  LEVELUP — SETUP (update)",
    },
    "setup_venv_exists": {
        "fr": "\n[1/3] Environnement .venv déjà présent ✓",
        "en": "\n[1/3] .venv environment already present ✓",
    },
    "setup_step1_searching": {
        "fr": "\n[1/3] Recherche de Python...",
        "en": "\n[1/3] Searching for Python...",
    },
    "setup_python_not_found": {
        "fr": "  Python 3.10+ non trouvé sur le système.",
        "en": "  Python 3.10+ not found on the system.",
    },
    "setup_python_impossible": {
        "fr": "\n  ❌ Impossible de trouver Python 3.10+.",
        "en": "\n  ❌ Unable to find Python 3.10+.",
    },
    "setup_python_install_url": {
        "fr": "  Installez-le depuis https://www.python.org/downloads/",
        "en": "  Install it from https://www.python.org/downloads/",
    },
    "setup_python_found": {
        "fr": "  Python trouvé: {py}",
        "en": "  Python found: {py}",
    },
    "setup_step2_creating_venv": {
        "fr": "\n[2/3] Création de l'environnement virtuel...",
        "en": "\n[2/3] Creating virtual environment...",
    },
    "setup_venv_create_failed": {
        "fr": "  ❌ Impossible de créer le venv.",
        "en": "  ❌ Unable to create the venv.",
    },
    "setup_venv_created": {
        "fr": "  ✓ .venv créé",
        "en": "  ✓ .venv created",
    },
    "setup_step2_venv_exists": {
        "fr": "\n[2/3] Environnement .venv existant ✓",
        "en": "\n[2/3] Existing .venv environment ✓",
    },
    "setup_step_deps": {
        "fr": "\n[{step}] Installation des dépendances...",
        "en": "\n[{step}] Installing dependencies...",
    },
    "setup_deps_failed": {
        "fr": "  ❌ L'installation des dépendances a échoué.",
        "en": "  ❌ Dependency installation failed.",
    },
    "setup_deps_causes": {
        "fr": "  Causes possibles :",
        "en": "  Possible causes:",
    },
    "setup_deps_no_internet": {
        "fr": "  - Pas de connexion internet",
        "en": "  - No internet connection",
    },
    "setup_deps_readonly": {
        "fr": "  - Dossier en lecture seule (déplace LevelUp dans Documents)",
        "en": "  - Folder is read-only (move LevelUp to Documents)",
    },
    "setup_deps_diskspace": {
        "fr": "  - Espace disque insuffisant",
        "en": "  - Insufficient disk space",
    },
    "setup_deps_ok": {
        "fr": "  ✓ Dépendances installées",
        "en": "  ✓ Dependencies installed",
    },
    "setup_critical_missing": {
        "fr": "  ⚠ Packages critiques manquants après installation.",
        "en": "  ⚠ Critical packages missing after installation.",
    },
    "setup_done": {
        "fr": "✅ SETUP TERMINÉ",
        "en": "✅ SETUP COMPLETE",
    },
    "setup_python_path": {
        "fr": "  Python: {py}",
        "en": "  Python: {py}",
    },
    "setup_useful_cmds": {
        "fr": "  Commandes utiles:",
        "en": "  Useful commands:",
    },
    "setup_cmd_run": {
        "fr": "    python launcher.py run     # Lancer le dashboard",
        "en": "    python launcher.py run     # Launch the dashboard",
    },
    "setup_cmd_doctor": {
        "fr": "    python launcher.py doctor  # Vérifier l'environnement",
        "en": "    python launcher.py doctor  # Check the environment",
    },
    # ── _cmd_doctor ───────────────────────────────────────────────────────────
    "doctor_title": {
        "fr": "🩺 LEVELUP — DOCTOR",
        "en": "🩺 LEVELUP — DOCTOR",
    },
    "doctor_os": {
        "fr": "  OS:     {system} {release}",
        "en": "  OS:     {system} {release}",
    },
    "doctor_python": {
        "fr": "  Python: {version}",
        "en": "  Python: {version}",
    },
    "doctor_exe": {
        "fr": "  Exe:    {exe}",
        "en": "  Exe:    {exe}",
    },
    "doctor_venv": {
        "fr": "  Venv:   {status}",
        "en": "  Venv:   {status}",
    },
    "doctor_wrong_interpreter": {
        "fr": "Mauvais interpréteur: {exe} (attendu {expected})",
        "en": "Wrong interpreter: {exe} (expected {expected})",
    },
    "doctor_no_venv": {
        "fr": "Dossier .venv introuvable — lancez: python launcher.py setup",
        "en": ".venv folder not found — run: python launcher.py setup",
    },
    "doctor_pkg_version_mismatch": {
        "fr": "{pkg}: {actual} (attendu {expected})",
        "en": "{pkg}: {actual} (expected {expected})",
    },
    "doctor_pkg_missing": {
        "fr": "Package manquant: {pkg}",
        "en": "Missing package: {pkg}",
    },
    "doctor_players_info": {
        "fr": "  📊 {count} joueur(s), {total} matchs",
        "en": "  📊 {count} player(s), {total} matches",
    },
    "doctor_no_players": {
        "fr": "Aucune donnée joueur trouvée dans data/players/",
        "en": "No player data found in data/players/",
    },
    "doctor_no_metadata": {
        "fr": "metadata.duckdb manquant",
        "en": "metadata.duckdb missing",
    },
    "doctor_warnings": {
        "fr": "\n⚠ Avertissements:",
        "en": "\n⚠ Warnings:",
    },
    "doctor_errors": {
        "fr": "\n❌ Erreurs:",
        "en": "\n❌ Errors:",
    },
    "doctor_fix_hint": {
        "fr": "\n  → Corrigez avec: python launcher.py setup",
        "en": "\n  → Fix with: python launcher.py setup",
    },
    "doctor_ok": {
        "fr": "\n✅ Environnement OK",
        "en": "\n✅ Environment OK",
    },
    # ── _run_migrations ───────────────────────────────────────────────────────
    "migrations_checking": {
        "fr": "\n🔧 Vérification du schéma de données…",
        "en": "\n🔧 Checking data schema…",
    },
    "migrations_up_to_date": {
        "fr": "   ✓ Schéma à jour",
        "en": "   ✓ Schema up to date",
    },
    "migrations_schemas_applied": {
        "fr": "   ✓ {n} migration(s) de schéma appliquée(s)",
        "en": "   ✓ {n} schema migration(s) applied",
    },
    "migrations_backfills_applied": {
        "fr": "   ✓ {n} backfill(s) exécuté(s)",
        "en": "   ✓ {n} backfill(s) executed",
    },
    "migrations_non_blocking_errors": {
        "fr": "   ⚠ {n} erreur(s) non-bloquante(s):",
        "en": "   ⚠ {n} non-blocking error(s):",
    },
    # ── _launch_streamlit ─────────────────────────────────────────────────────
    "launching_dashboard": {
        "fr": "\n🚀 Lancement du dashboard…",
        "en": "\n🚀 Launching dashboard…",
    },
    "launching_url": {
        "fr": "   URL: {url}",
        "en": "   URL: {url}",
    },
    "launching_arch": {
        "fr": "   Architecture: DuckDB v5",
        "en": "   Architecture: DuckDB v5",
    },
    "launching_data": {
        "fr": "   Données: {path}",
        "en": "   Data: {path}",
    },
    # ── _cmd_run ──────────────────────────────────────────────────────────────
    "run_no_data": {
        "fr": "❌ Aucune donnée joueur trouvée",
        "en": "❌ No player data found",
    },
    "run_configure_prompt": {
        "fr": "  Configurer un premier joueur maintenant ? [O/n] : ",
        "en": "  Configure a first player now? [Y/n]: ",
    },
    "run_no_tty_hint": {
        "fr": "   Lance : python launcher.py add-player --gamertag <gamertag>",
        "en": "   Run: python launcher.py add-player --gamertag <gamertag>",
    },
    "run_stats": {
        "fr": "\n📊 Architecture DuckDB v5: {count} joueur(s), {total} matchs",
        "en": "\n📊 DuckDB v5 Architecture: {count} player(s), {total} matches",
    },
    "run_player_row": {
        "fr": "   - {gamertag}: {matches} matchs",
        "en": "   - {gamertag}: {matches} matches",
    },
    # ── _cmd_sync ─────────────────────────────────────────────────────────────
    "sync_no_players": {
        "fr": "❌ Aucun joueur trouvé dans data/players/",
        "en": "❌ No players found in data/players/",
    },
    "sync_no_players_hint1": {
        "fr": "\n   Pour synchroniser un premier joueur :",
        "en": "\n   To sync a first player:",
    },
    "sync_no_players_hint3": {
        "fr": "\n   Ou directement en ligne de commande :",
        "en": "\n   Or directly via command line:",
    },
    "sync_title": {
        "fr": "🔄 SYNCHRONISATION (DuckDB v5)",
        "en": "🔄 SYNC (DuckDB v5)",
    },
    "sync_players_detected": {
        "fr": "\n   {count} joueur(s) détecté(s):",
        "en": "\n   {count} player(s) detected:",
    },
    "sync_player_row": {
        "fr": "   - {gamertag}: {matches} matchs",
        "en": "   - {gamertag}: {matches} matches",
    },
    "sync_in_progress": {
        "fr": "\n📥 Synchronisation en cours...",
        "en": "\n📥 Syncing...",
    },
    "sync_mode_delta": {
        "fr": "  → Sync delta...",
        "en": "  → Delta sync...",
    },
    "sync_mode_full": {
        "fr": "  → Sync complète...",
        "en": "  → Full sync...",
    },
    "sync_new_matches": {
        "fr": "  ✓ {n} nouveau(x) match(s)",
        "en": "  ✓ {n} new match(es)",
    },
    "sync_up_to_date": {
        "fr": "  ✓ À jour ({n} matchs)",
        "en": "  ✓ Up to date ({n} matches)",
    },
    "sync_done": {
        "fr": "✅ SYNCHRONISATION TERMINÉE",
        "en": "✅ SYNC COMPLETE",
    },
    "sync_summary_players": {
        "fr": "\n   Joueurs: {n}",
        "en": "\n   Players: {n}",
    },
    "sync_summary_total": {
        "fr": "   Total matchs: {n}",
        "en": "   Total matches: {n}",
    },
    "sync_summary_new": {
        "fr": "   Nouveaux: +{n}",
        "en": "   New: +{n}",
    },
    "sync_summary_failures": {
        "fr": "   ⚠ Échecs: {n}",
        "en": "   ⚠ Failures: {n}",
    },
    # ── _cmd_info ─────────────────────────────────────────────────────────────
    "info_title": {
        "fr": "📊 INFORMATIONS (DuckDB v5)",
        "en": "📊 INFORMATION (DuckDB v5)",
    },
    "info_dir": {
        "fr": "\n   Dossier: {path}",
        "en": "\n   Directory: {path}",
    },
    "info_players": {
        "fr": "   Joueurs: {n}",
        "en": "   Players: {n}",
    },
    "info_total_matches": {
        "fr": "   Total matchs: {n}",
        "en": "   Total matches: {n}",
    },
    "info_players_detail": {
        "fr": "\n   Détail par joueur:",
        "en": "\n   Per-player breakdown:",
    },
    "info_player_row": {
        "fr": "   - {gamertag}: {matches} matchs ({size:.1f} MB)",
        "en": "   - {gamertag}: {matches} matches ({size:.1f} MB)",
    },
    "info_metadata": {
        "fr": "\n   Métadonnées: {path} ({size:.1f} MB)",
        "en": "\n   Metadata: {path} ({size:.1f} MB)",
    },
    "info_no_metadata": {
        "fr": "\n   ⚠ Métadonnées non trouvées: {path}",
        "en": "\n   ⚠ Metadata not found: {path}",
    },
    # ── _ensure_warehouse_dbs ─────────────────────────────────────────────────
    "warehouse_shared_init": {
        "fr": "  ✓ shared_matches_v2.duckdb initialisé",
        "en": "  ✓ shared_matches_v2.duckdb initialized",
    },
    "warehouse_shared_init_fail": {
        "fr": "  ⚠ Impossible d'initialiser shared_matches_v2.duckdb : {err}",
        "en": "  ⚠ Unable to initialize shared_matches_v2.duckdb: {err}",
    },
    "warehouse_meta_init": {
        "fr": "  ✓ metadata.duckdb initialisé",
        "en": "  ✓ metadata.duckdb initialized",
    },
    "warehouse_meta_init_fail": {
        "fr": "  ⚠ Impossible d'initialiser metadata.duckdb : {err}",
        "en": "  ⚠ Unable to initialize metadata.duckdb: {err}",
    },
    # ── _print_device_code ────────────────────────────────────────────────────
    "dcf_box_title_line": {
        "fr": "  |         CODE A ENTRER SUR MICROSOFT                      |",
        "en": "  |           CODE TO ENTER ON MICROSOFT                      |",
    },
    "dcf_box_expires_label": {
        "fr": "Expire dans",
        "en": "Expires in",
    },
    "dcf_clipboard": {
        "fr": "  >> Code copie dans le presse-papiers (Ctrl+V pour coller).",
        "en": "  >> Code copied to clipboard (Ctrl+V to paste).",
    },
    # ── _wizard_oauth_token ───────────────────────────────────────────────────
    "wizard_title_text": {
        "fr": "Connexion Xbox Live — Device Code Flow",
        "en": "Xbox Live Sign-In — Device Code Flow",
    },
    "wizard_module_missing": {
        "fr": "  ❌ Module src.auth introuvable.",
        "en": "  ❌ Module src.auth not found.",
    },
    "wizard_dcf_init": {
        "fr": "  Initialisation du Device Code Flow…",
        "en": "  Initializing Device Code Flow…",
    },
    "wizard_dcf_error": {
        "fr": "  ❌ Erreur : {code} — {detail}",
        "en": "  ❌ Error: {code} — {detail}",
    },
    "wizard_dcf_reminder": {
        "fr": "  ↑ Revenez ici si besoin — Code : {code}",
        "en": "  ↑ Return here if needed — Code: {code}",
    },
    "wizard_dcf_waiting": {
        "fr": "  En attente de votre connexion Xbox… (ne fermez pas cette fenêtre)",
        "en": "  Waiting for your Xbox sign-in… (don't close this window)",
    },
    "wizard_dcf_connected": {
        "fr": "  ✅ Connecté : {gamertag} ({xuid})",
        "en": "  ✅ Signed in: {gamertag} ({xuid})",
    },
    "wizard_dcf_token_saved": {
        "fr": "     Token sauvegardé dans stats.duckdb (renouvellement automatique).",
        "en": "     Token saved in stats.duckdb (auto-renewal).",
    },
    "wizard_dcf_failed": {
        "fr": "  ❌ Échec : {code} — {detail}",
        "en": "  ❌ Failed: {code} — {detail}",
    },
    # ── _onboard_first_player ────────────────────────────────────────────────
    "onboard_title_text": {
        "fr": "LevelUp — Configuration du premier joueur",
        "en": "LevelUp — First Player Setup",
    },
    "onboard_non_tty": {
        "fr": "  Terminal non interactif — impossible de procéder.",
        "en": "  Non-interactive terminal — unable to proceed.",
    },
    "onboard_non_tty_hint": {
        "fr": "  Lance : python launcher.py add-player --gamertag <gamertag>",
        "en": "  Run: python launcher.py add-player --gamertag <gamertag>",
    },
    "onboard_msal_transfer_fail": {
        "fr": "  ⚠ Transfert cache MSAL : {err}",
        "en": "  ⚠ MSAL cache transfer failed: {err}",
    },
    "onboard_sync_how": {
        "fr": "  Comment veux-tu démarrer la synchronisation ?",
        "en": "  How would you like to start syncing?",
    },
    "onboard_sync_choice1": {
        "fr": "  1) Test rapide   — 10 matchs  (recommandé pour vérifier que tout fonctionne)",
        "en": "  1) Quick test    — 10 matches  (recommended to verify everything works)",
    },
    "onboard_sync_choice2": {
        "fr": "  2) Sync complet  — 200 matchs d'un coup",
        "en": "  2) Full sync     — 200 matches at once",
    },
    "onboard_sync_prompt": {
        "fr": "  Ton choix (1/2) [1] : ",
        "en": "  Your choice (1/2) [1]: ",
    },
    "onboard_sync_full_starting": {
        "fr": "  → Synchronisation de « {gamertag} » (200 matchs)…",
        "en": "  → Syncing '{gamertag}' (200 matches)…",
    },
    "onboard_sync_ok": {
        "fr": "\n  OK  {n} match(s) synchronise(s) pour {gamertag}",
        "en": "\n  OK  {n} match(es) synced for {gamertag}",
    },
    "onboard_sync_already": {
        "fr": "\n  OK  {n} match(s) deja presents pour {gamertag}",
        "en": "\n  OK  {n} match(es) already present for {gamertag}",
    },
    "onboard_sync_no_matches": {
        "fr": "\n  Aucun match recupere. Verifie ton token ou ta connexion.",
        "en": "\n  No matches retrieved. Check your token or connection.",
    },
    "onboard_test_starting": {
        "fr": "  → Test : synchronisation de 10 matchs pour « {gamertag} »…",
        "en": "  → Test: syncing 10 matches for '{gamertag}'…",
    },
    "onboard_test_failed": {
        "fr": "\n  Echec du test : {err}",
        "en": "\n  Test failed: {err}",
    },
    "onboard_test_what_now": {
        "fr": "  Que veux-tu faire ?",
        "en": "  What would you like to do?",
    },
    "onboard_test_retry": {
        "fr": "  1) Reessayer le test",
        "en": "  1) Retry the test",
    },
    "onboard_test_launch_anyway": {
        "fr": "  2) Lancer le dashboard quand meme (synchronisation plus tard)",
        "en": "  2) Launch the dashboard anyway (sync later)",
    },
    "onboard_quit": {
        "fr": "  Q) Quitter",
        "en": "  Q) Quit",
    },
    "onboard_test_prompt": {
        "fr": "  Ton choix (1/2/Q) [2] : ",
        "en": "  Your choice (1/2/Q) [2]: ",
    },
    "onboard_test_ok": {
        "fr": "\n  OK  {n} match(s) synchronise(s) — le test est concluant !",
        "en": "\n  OK  {n} match(es) synced — test successful!",
    },
    "onboard_test_existing": {
        "fr": "\n  OK  {n} match(s) deja presents (rien de nouveau sur 10 matchs)",
        "en": "\n  OK  {n} match(es) already present (nothing new in last 10 matches)",
    },
    "onboard_more_matches": {
        "fr": "  Veux-tu recuperer plus de matchs maintenant ?",
        "en": "  Would you like to retrieve more matches now?",
    },
    "onboard_more_continue": {
        "fr": "  1) Oui, continuer par batch de 200 matchs",
        "en": "  1) Yes, continue in batches of 200 matches",
    },
    "onboard_more_launch": {
        "fr": "  2) Non, lancer le dashboard avec les matchs actuels",
        "en": "  2) No, launch the dashboard with current matches",
    },
    "onboard_more_prompt": {
        "fr": "  Ton choix (1/2) [1] : ",
        "en": "  Your choice (1/2) [1]: ",
    },
    "onboard_batch_starting": {
        "fr": "  → Batch {n} : synchronisation de 200 matchs supplementaires…",
        "en": "  → Batch {n}: syncing 200 more matches…",
    },
    "onboard_batch_ok": {
        "fr": "\n  OK  {gained} nouveau(x) match(s) — total : {total} matchs",
        "en": "\n  OK  {gained} new match(es) — total: {total} matches",
    },
    "onboard_batch_done": {
        "fr": "  Tous les matchs disponibles ont ete recuperes.",
        "en": "  All available matches have been retrieved.",
    },
    "onboard_batch_continue": {
        "fr": "  Continuer avec un nouveau batch de 200 ?",
        "en": "  Continue with another batch of 200?",
    },
    "onboard_batch_yes": {
        "fr": "  1) Oui, continuer",
        "en": "  1) Yes, continue",
    },
    "onboard_batch_no": {
        "fr": "  2) Non, lancer le dashboard",
        "en": "  2) No, launch the dashboard",
    },
    "onboard_sync_total": {
        "fr": "\n  Synchronisation terminee — {n} match(s) recupere(s) au total.",
        "en": "\n  Sync complete — {n} match(es) retrieved in total.",
    },
    # ── _cmd_add_player ───────────────────────────────────────────────────────
    "add_player_sync_starting": {
        "fr": "  → Synchronisation de « {gamertag} »…",
        "en": "  → Syncing '{gamertag}'…",
    },
    "add_player_no_token": {
        "fr": "  ❌ Connexion Xbox manquante pour {gamertag}",
        "en": "  ❌ Xbox sign-in missing for {gamertag}",
    },
    "add_player_no_tty_hint": {
        "fr": "     Lance : python launcher.py add-player --gamertag {gamertag}",
        "en": "     Run: python launcher.py add-player --gamertag {gamertag}",
    },
    "add_player_new_matches": {
        "fr": "  ✅ {n} nouveau(x) match(s) pour {gamertag}",
        "en": "  ✅ {n} new match(es) for {gamertag}",
    },
    "add_player_up_to_date": {
        "fr": "  ✅ À jour ({n} matchs) pour {gamertag}",
        "en": "  ✅ Up to date ({n} matches) for {gamertag}",
    },
    # ── _cmd_reauth ───────────────────────────────────────────────────────────
    "reauth_starting": {
        "fr": "  Renouvellement du token OAuth pour « {gamertag} »…",
        "en": "  Renewing OAuth token for '{gamertag}'…",
    },
    "reauth_failed": {
        "fr": "  ❌ Échec du renouvellement.",
        "en": "  ❌ Renewal failed.",
    },
    "reauth_ok": {
        "fr": "  ✅ Token renouvelé pour {gamertag}",
        "en": "  ✅ Token renewed for {gamertag}",
    },
    # ── _recovery_menu ────────────────────────────────────────────────────────
    "recovery_title_text": {
        "fr": "⚠  Configuration incomplète détectée",
        "en": "⚠  Incomplete Configuration Detected",
    },
    "recovery_missing_token": {
        "fr": "  ✗ Accès Halo expiré ou manquant pour : {missing}",
        "en": "  ✗ Halo access expired or missing for: {missing}",
    },
    "recovery_non_tty_hint1": {
        "fr": "  → python launcher.py add-player   (reconfigurer)",
        "en": "  → python launcher.py add-player   (reconfigure)",
    },
    "recovery_non_tty_hint2": {
        "fr": "  → python launcher.py reauth --gamertag <GT>  (renouveler token)",
        "en": "  → python launcher.py reauth --gamertag <GT>  (renew token)",
    },
    "recovery_option_reauth": {
        "fr": "🔑 MSAL Device Code Flow  (code court sur microsoft.com/devicelogin pour {gt})",
        "en": "🔑 MSAL Device Code Flow  (short code on microsoft.com/devicelogin for {gt})",
    },
    "recovery_option_launch": {
        "fr": "🚀 Lancer quand même  (tu synchroniseras plus tard)",
        "en": "🚀 Launch anyway  (you'll sync later)",
    },
    "recovery_option_quit": {
        "fr": "Quitter",
        "en": "Quit",
    },
    "recovery_prompt": {
        "fr": "Ton choix ({keys}): ",
        "en": "Your choice ({keys}): ",
    },
    "recovery_invalid_choice": {
        "fr": "  Choix invalide.",
        "en": "  Invalid choice.",
    },
    "recovery_renewing": {
        "fr": "\n  🔄 Renouvellement du token pour « {gt} »…",
        "en": "\n  🔄 Renewing token for '{gt}'…",
    },
    # ── _interactive ──────────────────────────────────────────────────────────
    "interactive_title": {
        "fr": "        LevelUp - Dashboard Halo Infinite",
        "en": "        LevelUp - Halo Infinite Dashboard",
    },
    "interactive_arch": {
        "fr": "        Architecture DuckDB v5",
        "en": "        DuckDB v5 Architecture",
    },
    "interactive_state_header": {
        "fr": "\n📊 État actuel:",
        "en": "\n📊 Current status:",
    },
    "interactive_no_player": {
        "fr": "   ❌ Aucun joueur trouvé — premier démarrage",
        "en": "   ❌ No player found — first launch",
    },
    "interactive_choose_action": {
        "fr": "Choisis une action:\n",
        "en": "Choose an action:\n",
    },
    "interactive_add_player_option": {
        "fr": "  1) ➕ Ajouter un joueur               [premier démarrage]",
        "en": "  1) ➕ Add a player                    [first launch]",
    },
    "interactive_add_player_desc": {
        "fr": "     Configure et synchronise ton compte",
        "en": "     Set up and sync your account",
    },
    "interactive_quit_option": {
        "fr": "  Q) Quitter",
        "en": "  Q) Quit",
    },
    "interactive_non_tty": {
        "fr": "⚠ Terminal non interactif → python launcher.py add-player --gamertag <gt>",
        "en": "⚠ Non-interactive terminal → python launcher.py add-player --gamertag <gt>",
    },
    "interactive_choice_prompt": {
        "fr": "Ton choix (1/Q): ",
        "en": "Your choice (1/Q): ",
    },
    "interactive_launch_prompt": {
        "fr": "  Lancer le dashboard maintenant ? [O/n] : ",
        "en": "  Launch the dashboard now? [Y/n]: ",
    },
    "interactive_invalid_choice": {
        "fr": "Choix invalide.",
        "en": "Invalid choice.",
    },
    "interactive_storage": {
        "fr": "   Stockage: {path}",
        "en": "   Storage: {path}",
    },
    "interactive_players_count": {
        "fr": "   Joueurs: {n}",
        "en": "   Players: {n}",
    },
    "interactive_player_row": {
        "fr": "      - {gamertag}: {matches} matchs",
        "en": "      - {gamertag}: {matches} matches",
    },
    "interactive_total_matches": {
        "fr": "   Total: {n} matchs",
        "en": "   Total: {n} matches",
    },
    "interactive_metadata_ok": {
        "fr": "   Métadonnées: ✅",
        "en": "   Metadata: ✅",
    },
    "interactive_metadata_missing": {
        "fr": "   Métadonnées: ⚠ Non trouvées",
        "en": "   Metadata: ⚠ Not found",
    },
    "interactive_all_ok": {
        "fr": "\n  ✅ Configuration complète — lancement du dashboard…",
        "en": "\n  ✅ Configuration complete — launching dashboard…",
    },
}


def t(key: str, lang: str = "fr", **kwargs: object) -> str:
    """Retourne la chaîne traduite pour la clé et la langue données.

    Args:
        key: Clé de traduction.
        lang: Code langue ('fr' ou 'en'). Par défaut 'fr'.
        **kwargs: Variables à interpoler dans la chaîne via .format().

    Returns:
        Chaîne traduite, ou la clé elle-même si introuvable.
    """
    entry = STRINGS.get(key)
    if entry is None:
        return key
    text = entry.get(lang) or entry.get("fr") or key
    return text.format(**kwargs) if kwargs else text
