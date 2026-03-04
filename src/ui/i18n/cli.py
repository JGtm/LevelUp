"""Chaînes pour les scripts CLI et le Discord notifier.

Ce module ne dépend PAS de Streamlit — il peut être importé dans
n'importe quel contexte (cron, script CLI, tests).

La langue est lue via ``ct()`` depuis :
1. ``os.environ["LEVELUP_LANG"]``
2. ``app_settings.json`` → clé ``"cli_lang"``
3. Défaut : ``"fr"``

Pour Discord, la clé ``"discord_lang"`` dans ``app_settings.json`` prévaut.

⚠️ ChatGPT : remplir toutes les valeurs marquées "TODO" ci-dessous.
   Règles : voir le prompt de la Phase 1b dans le plan i18n.
"""

from __future__ import annotations

import contextlib
import os

STRINGS: dict[str, dict[str, str] | str] = {
    # ── Discord — Résultats de match ─────────────────────────────────────────
    "discord_outcome_draw": "outcome_draw",  # alias → common
    "discord_outcome_win": "outcome_win",  # alias → common
    "discord_outcome_loss": "outcome_loss",  # alias → common
    "discord_outcome_quit": {
        "fr": "Abandon",
        "en": "Quit",
    },
    # ── Discord — Labels d'opération ────────────────────────────────────────
    "discord_op_sync_delta": {
        "fr": "Sync delta",
        "en": "Delta sync",
    },
    "discord_op_sync_full": {
        "fr": "Sync complète",
        "en": "Full sync",
    },
    "discord_op_backfill": {
        "fr": "Backfill",
        "en": "Backfill",
    },
    # ── Discord — Corps du message ───────────────────────────────────────────
    "discord_completed_in": {
        "fr": "**{status}  {op}** terminée en **{duration}**",
        "en": "**{status}  {op}** completed in **{duration}**",
    },
    "discord_players_matches": {
        "fr": "👥  {players} joueur(s)  ·  {matches} match(s) traité(s)",
        "en": "👥  {players} player(s)  ·  {matches} match(es) processed",
    },
    "discord_matches_synced": {
        "fr": "**+{count}** match(s) synchronisé(s)",
        "en": "**+{count}** match(es) synced",
    },
    "discord_matches_processed": {
        "fr": "**{count}** match(s) retraité(s)",
        "en": "**{count}** match(es) reprocessed",
    },
    "discord_data_complete": {
        "fr": "✅  Données complètes",
        "en": "✅  Data complete",
    },
    "discord_data_incomplete": {
        "fr": "⚠️   **{count}** match(s) avec données incomplètes",
        "en": "⚠️   **{count}** match(es) with incomplete data",
    },
    "discord_error_field": {
        "fr": "⛔  Erreur : {error}",
        "en": "⛔  Error: {error}",
    },
    "discord_last_match": {
        "fr": "Dernier match",
        "en": "Last match",
    },
    "discord_ranked_tag": {
        "fr": "Classé",
        "en": "Ranked",
    },
    "discord_footer": {
        "fr": "LevelUp · Halo Infinite Stats",
        "en": "LevelUp · Halo Infinite Stats",
    },
    "discord_title": {
        "fr": "🎮  LevelUp — {op}",
        "en": "🎮  LevelUp — {op}",
    },
    "discord_time_range": {
        "fr": "🕐  `{t_start}`  →  `{t_end}`",
        "en": "🕐  `{t_start}`  →  `{t_end}`",
    },
    "discord_up_to_date_sync": {
        "fr": "Déjà à jour",
        "en": "Already up to date",
    },
    "discord_no_new_matches": {
        "fr": "Aucun nouveau match",
        "en": "No new matches",
    },
    "discord_no_matches_to_reprocess": {
        "fr": "Aucun match à retraiter",
        "en": "Nothing to reprocess",
    },
    "discord_all_up_to_date": {
        "fr": "Tout déjà à jour",
        "en": "All up to date",
    },
    "discord_player_count": {
        "fr": "👥  {count} joueur(s)",
        "en": "👥  {count} player(s)",
    },
    "discord_kda": {
        "fr": "{k}F / {d}D / {a}A",
        "en": "{k}K / {d}D / {a}A",
    },
    "discord_squad_match": {
        "fr": "🎮 Match en escouade",
        "en": "🎮 Squad match",
    },
    "discord_squad_friends": {
        "fr": "👥 Amis : {friends}",
        "en": "👥 Friends: {friends}",
    },
    "tailscale_discord_startup": {
        "fr": "🟢 LevelUp est disponible sur {url}",
        "en": "🟢 LevelUp is available at {url}",
    },
    # ── Scripts — Messages logger génériques ─────────────────────────────────
    "cli_no_players": {
        "fr": "Aucun joueur à synchroniser.",
        "en": "No players to sync.",
    },
    "cli_up_to_date": {
        "fr": "À jour ({count} matchs).",
        "en": "Up to date ({count} matches).",
    },
    "cli_timeout": {
        "fr": "Timeout après {n}s.",
        "en": "Timeout after {n}s.",
    },
    "cli_error": {
        "fr": "Erreur : {error}",
        "en": "Error: {error}",
    },
    "cli_dry_run": {
        "fr": "Aucune modification effectuée [DRY-RUN]",
        "en": "No changes made [DRY-RUN]",
    },
    "cli_players_available": {
        "fr": "Joueurs disponibles :",
        "en": "Available players:",
    },
    "cli_no_players_found": {
        "fr": "Aucun joueur trouvé",
        "en": "No players found",
    },
    "cli_operation_cancelled": {
        "fr": "Opération annulée",
        "en": "Operation cancelled",
    },
    "cli_summary_global": {
        "fr": "=== RÉSUMÉ GLOBAL ===",
        "en": "=== GLOBAL SUMMARY ===",
    },
    "cli_players_processed": {
        "fr": "Joueurs traités : {count}",
        "en": "Players processed: {count}",
    },
    "cli_matches_verified": {
        "fr": "Matchs vérifiés : {count}",
        "en": "Matches verified: {count}",
    },
    "cli_medals_inserted": {
        "fr": "Médailles insérées : {count}",
        "en": "Medals inserted: {count}",
    },
    "cli_events_inserted": {
        "fr": "Events insérés : {count}",
        "en": "Events inserted: {count}",
    },
    "cli_xuid_resolved_sync_meta": {
        "fr": "XUID résolu depuis sync_meta : {xuid}",
        "en": "XUID resolved from sync_meta: {xuid}",
    },
    "cli_xuid_resolved_shared": {
        "fr": "XUID résolu depuis shared.xuid_aliases : {xuid}",
        "en": "XUID resolved from shared.xuid_aliases: {xuid}",
    },
    "cli_archive_player": {
        "fr": "Joueur : {gamertag}",
        "en": "Player: {gamertag}",
    },
    "cli_archive_total_matches": {
        "fr": "Total matchs : {count}",
        "en": "Total matches: {count}",
    },
    "cli_archive_oldest": {
        "fr": "Plus ancien : {date}",
        "en": "Oldest: {date}",
    },
    "cli_archive_newest": {
        "fr": "Plus récent : {date}",
        "en": "Newest: {date}",
    },
    "cli_archive_by_year": {
        "fr": "Par année :",
        "en": "By year:",
    },
    "cli_archive_files": {
        "fr": "Fichiers : {count}",
        "en": "Files: {count}",
    },
    "cli_archive_total_size": {
        "fr": "Taille totale : {size:.2f} MB",
        "en": "Total size: {size:.2f} MB",
    },
    "cli_archive_no_archive": {
        "fr": "Aucune archive",
        "en": "No archive",
    },
    "cli_archive_recommendations": {
        "fr": "Recommandations :",
        "en": "Recommendations:",
    },
    "cli_archive_warn_5000": {
        "fr": "⚠ > 5000 matchs : archivage recommandé",
        "en": "⚠ > 5000 matches: archiving recommended",
    },
    "cli_archive_warn_years": {
        "fr": "⚠ Historique sur {years} années : archivage par année recommandé",
        "en": "⚠ History spanning {years} years: archiving by year recommended",
    },
    "cli_archive_ok": {
        "fr": "✓ < 1000 matchs : archivage non nécessaire",
        "en": "✓ < 1000 matches: archiving not necessary",
    },
    "cli_archive_delete_warning": {
        "fr": "ATTENTION : --delete va SUPPRIMER les matchs de la DB ! Cette opération est IRRÉVERSIBLE. Assurez-vous d'avoir un backup.",
        "en": "WARNING: --delete will DELETE matches from the DB! This operation is IRREVERSIBLE. Make sure you have a backup.",
    },
    "cli_env_ok": {
        "fr": "OK : environnement conforme.",
        "en": "OK: environment is compliant.",
    },
    "cli_env_warnings": {
        "fr": "Avertissements :",
        "en": "Warnings:",
    },
    "cli_env_errors": {
        "fr": "ERREURS :",
        "en": "ERRORS:",
    },
    "cli_env_suggested_actions": {
        "fr": "Actions suggérées :",
        "en": "Suggested actions:",
    },
}


def _read_cli_lang() -> str:
    """Lit la langue CLI depuis l'environnement ou app_settings.json."""
    env_lang = os.environ.get("LEVELUP_LANG")
    if env_lang:
        return env_lang
    try:
        import json
        from pathlib import Path

        settings_path = Path(__file__).parents[4] / "app_settings.json"
        if settings_path.exists():
            data = json.loads(settings_path.read_text(encoding="utf-8"))
            return data.get("cli_lang", "fr")
    except Exception:
        pass
    return "fr"


def _resolve_entry(key: str) -> dict[str, str] | None:
    """Résout une entrée STRINGS en suivant les alias (str → str → dict).

    Les alias peuvent pointer vers des clés du registre global i18n
    (``common.py``, ``pages/``). Import paresseux pour éviter la circularité.
    """
    seen: set[str] = set()
    current = key
    while current not in seen:
        seen.add(current)
        entry = STRINGS.get(current)
        if entry is None:
            # Clé absente du STRINGS local → chercher dans le registre global
            try:
                import src.ui.i18n as _i18n_pkg

                if _i18n_pkg._REGISTRY is None:
                    _i18n_pkg._REGISTRY = _i18n_pkg._build_registry()
                global_entry = _i18n_pkg._REGISTRY.get(current)
            except Exception:
                return None
            if global_entry is None:
                return None
            # Le registre global a déjà résolu ses alias → toujours un dict
            return global_entry if isinstance(global_entry, dict) else None
        if isinstance(entry, dict):
            return entry
        # entry est un alias (str) → suivre vers la clé cible
        current = entry
    return None  # boucle d'alias détectée


def ct(key: str, **kwargs: object) -> str:
    """Retourne la chaîne CLI traduite pour la clé donnée.

    Lit la langue depuis ``LEVELUP_LANG`` ou ``app_settings.json:cli_lang``.

    Args:
        key:     Clé dans STRINGS.
        **kwargs: Variables pour str.format().

    Returns:
        La chaîne traduite, ou la clé entre crochets si introuvable.
    """
    lang = _read_cli_lang()
    entry = _resolve_entry(key)
    if entry is None:
        return f"[{key}]"
    text = entry.get(lang) or entry.get("fr") or f"[{key}]"
    if kwargs:
        with contextlib.suppress(KeyError, ValueError):
            text = text.format(**kwargs)
    return text


def discord_t(key: str, **kwargs: object) -> str:
    """Retourne la chaîne Discord traduite.

    Lit la langue depuis ``app_settings.json:discord_lang`` (défaut ``"fr"``).

    Args:
        key:     Clé dans STRINGS.
        **kwargs: Variables pour str.format().

    Returns:
        La chaîne traduite, ou la clé entre crochets si introuvable.
    """
    try:
        import json
        from pathlib import Path

        settings_path = Path(__file__).parents[4] / "app_settings.json"
        if settings_path.exists():
            data = json.loads(settings_path.read_text(encoding="utf-8"))
            lang = data.get("discord_lang", "fr")
        else:
            lang = "fr"
    except Exception:
        lang = "fr"

    entry = _resolve_entry(key)
    if entry is None:
        return f"[{key}]"
    text = entry.get(lang) or entry.get("fr") or f"[{key}]"
    if kwargs:
        with contextlib.suppress(KeyError, ValueError):
            text = text.format(**kwargs)
    return text
