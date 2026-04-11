"""Chaînes i18n — page Paramètres."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    "settings_title": {
        "fr": "Paramètres",
        "en": "Settings",
    },
    "settings_backfill_data_label": {
        "fr": "Données à backfill :",
        "en": "Data to backfill:",
    },
    "backfill_all_data": {"fr": "Toutes les données", "en": "All data"},
    # ── Page Analyse Objectifs ───────────────────────────────────────────────
    "settings_save_ok": {
        "fr": "Paramètres sauvegardés.",
        "en": "Settings saved.",
    },
    "settings_save_error": {
        "fr": "Erreur lors de la sauvegarde : {error}",
        "en": "Error while saving: {error}",
    },
    # ── Last Match ────────────────────────────────────────────────────────────
    "settings_index_reset": {
        "fr": "Index médias réinitialisé (joueur courant).",
        "en": "Media index reset (current player).",
    },
    "set_db_mgmt": {"fr": "Gestion de la base de données", "en": "Database management"},
    "set_db_empty": {
        "fr": "La base sélectionnée est vide (0 octet). Lancez une synchronisation.",
        "en": "The selected database is empty (0 bytes). Run a sync.",
    },
    "set_sync_title": {"fr": "Synchronisation", "en": "Sync"},
    "set_sync_source": {"fr": "Source de données", "en": "Data source"},
    "set_sync_max_matches": {"fr": "Max matchs à récupérer", "en": "Max matches to fetch"},
    "set_sync_rate": {"fr": "Délai entre appels API (sec)", "en": "API rate limit (sec)"},
    "set_refresh_options": {"fr": "Backfill (avancé)", "en": "Backfill (advanced)"},
    "settings_backfill_warning": {
        "fr": "⚠️ Section expérimentale — le backfill via l'app n'a pas été pleinement testé. Préférer la CLI (`scripts/backfill_data.py`).",
        "en": "⚠️ Experimental section — backfill via the app has not been fully tested. Prefer the CLI (`scripts/backfill_data.py`).",
    },
    "settings_backfill_expand_label": {
        "fr": "Afficher les options de backfill",
        "en": "Show backfill options",
    },
    "set_backfill_medals": {"fr": "Médailles", "en": "Medals"},
    "set_backfill_events": {"fr": "Événements", "en": "Events"},
    "set_backfill_scores": {"fr": "Scores de performance", "en": "Performance scores"},
    "set_backfill_aliases": {"fr": "Alias joueurs", "en": "Player aliases"},
    "set_backfill_personal_scores": {"fr": "Personal Scores", "en": "Personal Scores"},
    "set_backfill_sessions": {"fr": "Sessions", "en": "Sessions"},
    "set_backfill_citations": {"fr": "Citations", "en": "Citations"},
    "set_backfill_antagonists": {"fr": "Antagonistes", "en": "Antagonists"},
    "set_backfill_skill": {"fr": "Skill/Rating", "en": "Skill/Rating"},
    "set_backfill_killer_victim": {"fr": "Killer/Victim", "en": "Killer/Victim"},
    "set_backfill_shots": {"fr": "Shots (tirs)", "en": "Shots"},
    "set_backfill_participants_shots": {"fr": "Shots participants", "en": "Participants shots"},
    "set_backfill_pve": {"fr": "Stats PvE", "en": "PvE stats"},
    "set_backfill_lusr": {"fr": "LUSR (rating local)", "en": "LUSR (local rating)"},
    "set_media_title": {"fr": "Médias", "en": "Media"},
    "set_media_enable": {"fr": "Activer les médias", "en": "Enable media"},
    "set_media_screenshots": {"fr": "Dossier captures", "en": "Screenshots folder"},
    "set_media_videos": {"fr": "Dossier vidéos", "en": "Videos folder"},
    "set_media_tolerance": {
        "fr": "Tolérance association (minutes)",
        "en": "Association tolerance (minutes)",
    },
    "set_experience_title": {"fr": "Expérience", "en": "Experience"},
    "set_lang_title": {"fr": "Langue", "en": "Language"},
    "set_clear_cache_title": {
        "fr": "Vider le cache à chaque actualisation",
        "en": "Clear cache on every refresh",
    },
    "set_timezone_label": {
        "fr": "Fuseau horaire (affichage des dates)",
        "en": "Timezone (date display)",
    },
    "set_timezone_help": {
        "fr": "Toutes les dates et heures affichées dans l'application utiliseront ce fuseau horaire.",
        "en": "All dates and times displayed in the application will use this timezone.",
    },
    "set_normalize_mode_labels": {
        "fr": "Normaliser les labels de modes",
        "en": "Normalize mode labels",
    },
    "set_normalize_mode_labels_help": {
        "fr": "Supprime les préfixes redondants selon le contexte (ex: BTB:Assassin dans une playlist BTB → Assassin).",
        "en": "Removes redundant prefixes based on context (e.g. BTB:Slayer in a BTB playlist → Slayer).",
    },
    # ── Match view helpers ──────────────────────────────────────────────────
    "set_arch_v5_info": {
        "fr": "ℹ️ **Architecture v5** : La synchronisation récupère automatiquement TOUTES les données :\n- Nouveaux matchs (matchmaking uniquement)\n- Highlight events (clips)\n- Médailles\n- Stats skill/MMR\n- Personal score awards\n- Aliases XUID\n\nCes options ne sont plus configurables - tout est récupéré à chaque sync.",
        "en": "ℹ️ **Architecture v5**: The sync automatically fetches ALL data:\n- New matches (matchmaking only)\n- Highlight events (clips)\n- Medals\n- Skill/MMR stats\n- Personal score awards\n- XUID aliases\n\nThese options are no longer configurable — everything is fetched on each sync.",
    },
    "settings_backfill_caption": {
        "fr": "Configurez ce que fait le bouton 🔄 Synchroniser dans la sidebar. Le backfill remplit les données manquantes pour les matchs existants.",
        "en": "Configure what the 🔄 Sync button does in the sidebar. Backfill fills missing data for existing matches.",
    },
    "settings_backfill_all_help": {
        "fr": "Cochez pour backfill toutes les données, ou choisissez individuellement ci-dessous",
        "en": "Check to backfill all data, or choose individually below",
    },
    "set_backfill_score_help": {
        "fr": "Calcule les scores de performance manquants (peut être activé même sans backfill général)",
        "en": "Computes missing performance scores (can be enabled even without general backfill)",
    },
    "set_backfill_lusr_help": {
        "fr": "Calcule le LUSR (LevelUp Skill Rank) pour les matchs non classés — calcul local, pas d'API requise",
        "en": "Computes LUSR (LevelUp Skill Rank) for unranked matches — local computation, no API required",
    },
    "set_backfill_events_help": {
        "fr": "Récupère les highlight events (premier frag, première mort, frags parfaits…) pour les matchs déjà en base. Peut être activé sans le toggle Backfill général.",
        "en": "Fetches highlight events (first kill, first death, perfect kills…) for matches already in the database. Can be enabled without the general Backfill toggle.",
    },
    "set_backfill_weapons": {
        "fr": "Kills par arme (films SPNKr)",
        "en": "Weapon kills (SPNKr films)",
    },
    "set_backfill_weapons_help": {
        "fr": "Télécharge les chunks film SPNKr pour extraire les kills par arme (POV uniquement, ~87% de couverture). Nécessite tokens API valides.",
        "en": "Downloads SPNKr film chunks to extract per-weapon kills (POV only, ~87% coverage). Requires valid API tokens.",
    },
    "set_media_arch_info": {
        "fr": "ℹ️ **Architecture v5** : La section Médias est toujours active. Configurez le dossier de base et la tolérance temporelle.",
        "en": "ℹ️ **Architecture v5**: The Media section is always active. Configure the base folder and time tolerance.",
    },
    "set_media_root_help": {
        "fr": "Racine des captures. Un sous-dossier par joueur, nommé comme le gamertag (ex: D:/Captures/PlayerA/, D:/Captures/PlayerB/). Images et vidéos dans le même dossier.",
        "en": "Capture root. One subfolder per player, named like the gamertag (e.g. D:/Captures/PlayerA/, D:/Captures/PlayerB/). Images and videos in the same folder.",
    },
    # ── Settings V2 — nouvelles sections ────────────────────────────────────
    "set_language_section": {"fr": "Langue & Région", "en": "Language & Region"},
    "set_lang_label": {"fr": "Langue de l'interface", "en": "Interface language"},
    "set_display_section": {"fr": "Affichage", "en": "Display"},
    "set_show_hints": {
        "fr": "Aides à la lecture",
        "en": "Reading hints",
    },
    "set_show_hints_help": {
        "fr": "Affiche les légendes, notes explicatives et conseils contextuels dans les graphiques.",
        "en": "Shows legends, explanatory notes and contextual tips in charts.",
    },
    "set_show_records": {
        "fr": "Afficher les records historiques sur les graphes Escouade",
        "en": "Show historical records on Squad charts",
    },
    "set_show_records_help": {
        "fr": "Affiche des barres hachurées indiquant le record personnel de chaque joueur sur les graphes de la page Escouade.",
        "en": "Displays hatched bars showing each player's personal record on Squad page charts.",
    },
    "set_career_exclude_btb": {
        "fr": "Exclure les matchs BTB du Top / Pire",
        "en": "Exclude BTB matches from Top / Worst",
    },
    "set_career_exclude_btb_help": {
        "fr": "Les matchs Big Team Battle sont exclus du classement Top 10 meilleurs/pires matchs.",
        "en": "Big Team Battle matches are excluded from the Top 10 best/worst matches ranking.",
    },
    "set_clear_cache_help": {
        "fr": "Vide automatiquement le cache Streamlit à chaque synchronisation. Utile si la DB change en dehors de l'app (NAS, import externe).",
        "en": "Automatically clears the Streamlit cache on each sync. Useful if the DB changes outside the app (NAS, external import).",
    },
    "set_discord_section": {"fr": "Notifications Discord", "en": "Discord Notifications"},
    "set_discord_enable": {
        "fr": "Activer les notifications Discord",
        "en": "Enable Discord notifications",
    },
    "set_discord_url": {"fr": "URL du webhook", "en": "Webhook URL"},
    "set_discord_url_env_hint": {
        "fr": "✅ Webhook configuré via `.env.local` (DISCORD_WEBHOOK_URL)",
        "en": "✅ Webhook configured via `.env.local` (DISCORD_WEBHOOK_URL)",
    },
    "set_discord_lang_label": {
        "fr": "Langue des messages Discord",
        "en": "Discord message language",
    },
    "set_discord_notify_types_label": {
        "fr": "Types de notifications",
        "en": "Notification types",
    },
    "set_discord_notify_sync": {
        "fr": "Après chaque sync",
        "en": "After each sync",
    },
    "set_discord_notify_backfill": {
        "fr": "Après chaque backfill",
        "en": "After each backfill",
    },
    "set_discord_notify_new_version": {
        "fr": "Nouvelle version majeure déployée",
        "en": "New major version deployed",
    },
    "set_discord_notify_new_version_help": {
        "fr": "Envoie un message Discord avec les nouveautés (README) quand X ou Y changent dans vX.Y.Z. Nécessite LEVELUP_NOTIFY_VERSIONS=1 dans l'environnement (opt-in prod).",
        "en": "Sends a Discord message with release notes (README) when X or Y change in vX.Y.Z. Requires LEVELUP_NOTIFY_VERSIONS=1 in the environment (prod opt-in).",
    },
    "set_media_watcher_label": {
        "fr": "Watcher automatique (Linux/inotify)",
        "en": "Automatic watcher (Linux/inotify)",
    },
    "set_media_watcher_help": {
        "fr": "Sur Linux, surveille le dossier captures en temps réel (inotify). Désactiver pour revenir au scan périodique.",
        "en": "On Linux, watches the captures folder in real time (inotify). Disable to fall back to periodic scanning.",
    },
    "set_media_debounce_label": {
        "fr": "Délai avant indexation (secondes)",
        "en": "Indexing delay (seconds)",
    },
    "set_media_debounce_help": {
        "fr": "Attend N secondes d'inactivité après le dernier fichier détecté avant d'indexer. Utile pour les copies de gros fichiers vidéo.",
        "en": "Waits N seconds of inactivity after the last detected file before indexing. Useful for large video file copies.",
    },
    # ── Avertissements de cohérence (Settings) ──────────────────────────────
    "warn_discord_no_webhook": {
        "fr": "Les notifications Discord sont activées mais aucun webhook n'est configuré (champ vide et variable DISCORD_WEBHOOK_URL absente).",
        "en": "Discord notifications are enabled but no webhook is configured (empty field and DISCORD_WEBHOOK_URL not set).",
    },
    "warn_media_no_dir": {
        "fr": "Le dossier de captures n'est pas configuré. Le watcher et la ré-indexation ne fonctionneront pas — les médias déjà indexés en base restent visibles.",
        "en": "No captures folder is configured. The watcher and re-indexing won't work — media already indexed in the database remains visible.",
    },
    # ── Match view helpers (Phase 3) ─────────────────────────────────────────
}
