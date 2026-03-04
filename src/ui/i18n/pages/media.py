"""Chaînes i18n — page Médias et bibliothèque."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    "media_library_title": {
        "fr": "Bibliothèque médias",
        "en": "Media library",
    },
    "media_unassociated": {
        "fr": "Non associés",
        "en": "Unassociated",
    },
    "media_rescan": {
        "fr": "Re-scanner les dossiers",
        "en": "Rescan folders",
    },
    "media_error_indexing": "indexing_error",  # alias → common
    "media_configure_video": {
        "fr": "Configure un dossier vidéos dans Paramètres → Médias.",
        "en": "Configure a videos folder in Settings → Media.",
    },
    # ── Page Paramètres ──────────────────────────────────────────────────────
    "media_options_expander": {
        "fr": "Options",
        "en": "Options",
    },
    "media_scanning": {
        "fr": "Indexation en cours…",
        "en": "Scanning…",
    },
    "media_generating_thumbnails": {
        "fr": "Génération des thumbnails…",
        "en": "Generating thumbnails…",
    },
    # ── Citations ─────────────────────────────────────────────────────────────
    "media_no_files": "media_not_found",  # alias → common
    "media_no_filter_result": "no_media",  # alias → common
    "media_unknown_match": "match_unknown",  # alias → common
    "media_no_thumbnail": "no_thumbnail",  # alias → common
    "media_unassociated_match": "not_associated",  # alias → common
    # ── Settings ─────────────────────────────────────────────────────────────
    "media_view_full": {"fr": "Voir en grand", "en": "View full size"},
    "media_file_missing": {"fr": "Fichier absent : {file_name}", "en": "File missing: {file_name}"},
    "media_dialog_title": {"fr": "Média", "en": "Media"},
    "media_no_indexed": {
        "fr": "Aucun média indexé. Configure les dossiers dans Paramètres → Médias et lance un scan.",
        "en": "No media indexed. Configure folders in Settings → Media and run a scan.",
    },
    "media_filters": {"fr": "Filtres", "en": "Filters"},
    "media_type": {"fr": "Type", "en": "Type"},
    "media_filename": {"fr": "Nom de fichier", "en": "File name"},
    "media_columns": {"fr": "Colonnes", "en": "Columns"},
    "media_my_captures": {"fr": "Mes captures", "en": "My captures"},
    "media_captures_of": {"fr": "Captures de {gamertag}", "en": "Captures by {gamertag}"},
    "media_unmatched": {"fr": "Sans correspondance", "en": "Unmatched"},
    # ── Media library ───────────────────────────────────────────────────────
    "ml_open_match": {"fr": "Ouvrir le match", "en": "Open match"},
    "ml_click_thumbnail": {
        "fr": "Cliquer pour afficher la miniature",
        "en": "Click to show thumbnail",
    },
    "ml_hide_thumbnail": {"fr": "Masquer miniature", "en": "Hide thumbnail"},
    "ml_show_thumbnail": {"fr": "Afficher miniature", "en": "Show thumbnail"},
    "ml_preview": {"fr": "Aperçu", "en": "Preview"},
    "ml_options": {"fr": "Options", "en": "Options"},
    "ml_group_by_match": {"fr": "Grouper par match", "en": "Group by match"},
    "ml_show_unassociated": {"fr": "Afficher non associés", "en": "Show unassociated"},
    "ml_max_media": {"fr": "Max médias", "en": "Max media"},
    "ml_types": {"fr": "Types", "en": "Types"},
    "ml_filter_filename": {"fr": "Filtre nom de fichier", "en": "Filter filename"},
    "ml_rescan": {"fr": "Re-scanner les dossiers", "en": "Re-scan folders"},
    "ml_indexing_done": {
        "fr": "Indexation terminée : {n_new} nouveaux, {n_updated} mis à jour, {n_associated} association(s).",
        "en": "Indexing done: {n_new} new, {n_updated} updated, {n_associated} association(s).",
    },
    "ml_indexing_thumbnails": {
        "fr": " — {n_gen} miniature(s), {n_err} erreur(s)",
        "en": " — {n_gen} thumbnail(s), {n_err} error(s)",
    },
    "ml_generate_thumbnails": {"fr": "Générer les thumbnails", "en": "Generate thumbnails"},
    "ml_thumbnails_generated": {
        "fr": "{n_gen} thumbnail(s) généré(s)",
        "en": "{n_gen} thumbnail(s) generated",
    },
    "ml_unassigned_from_db": {
        "fr": "ℹ️ {count} média(s) non associé(s) depuis la BDD. Cliquez sur 'Re-scanner les dossiers' pour forcer l'association.",
        "en": "ℹ️ {count} unassociated media from DB. Click 'Re-scan folders' to force association.",
    },
    "ml_disk_fallback": {
        "fr": "ℹ️ Les médias sont chargés depuis le scan disque (pas encore indexés en BDD). Cliquez sur 'Re-scanner les dossiers' pour indexer.",
        "en": "ℹ️ Media loaded from disk scan (not yet indexed in DB). Click 'Re-scan folders' to index.",
    },
    "ml_no_match_windows": {
        "fr": "⚠️ Aucune fenêtre temporelle de match disponible pour l'association automatique.",
        "en": "⚠️ No match time windows available for automatic association.",
    },
    "ml_no_associations": {
        "fr": "⚠️ Aucun média associé à un match depuis la BDD. Tolérance actuelle : {tol} min.",
        "en": "⚠️ No media could be associated with a match from DB. Current tolerance: {tol} min.",
    },
    "ml_generate_help": {
        "fr": "Génère les miniatures pour les vidéos. Nécessite ffmpeg.",
        "en": "Generate thumbnails for videos. Requires ffmpeg.",
    },
    "ml_rescan_prompt": {
        "fr": "Cliquez sur 'Re-scanner les dossiers' pour forcer l'indexation.",
        "en": "Click 'Re-scan folders' to force indexing.",
    },
    "ml_rescan_index": {
        "fr": "Cliquez sur 'Re-scanner les dossiers' pour indexer en BDD.",
        "en": "Click 'Re-scan folders' to index into DB.",
    },
    "ml_tolerance": {
        "fr": "Tolérance actuelle : {tol} min",
        "en": "Current tolerance: {tol} min",
    },
    "ml_window": {"fr": "Fenêtre : {t0} → {t1}", "en": "Window: {t0} → {t1}"},
    "ml_captures": {"fr": "Captures", "en": "Captures"},
    "ml_video": {"fr": "Vidéo", "en": "Video"},
    # ── Settings ────────────────────────────────────────────────────────────
}
