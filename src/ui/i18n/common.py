"""Chaînes génériques partagées entre toutes les pages.

Ces chaînes apparaissent dans plusieurs fichiers src/ui/pages/ et src/app/.
En centralisant ici, on évite la duplication.

⚠️ ChatGPT : remplir toutes les valeurs marquées "TODO" ci-dessous.
   Règles : voir le prompt de la Phase 1b dans le plan i18n.
"""
from __future__ import annotations

STRINGS: dict[str, dict[str, str]] = {
    # ── Données manquantes / vides ──────────────────────────────────────────
    "no_matches": {
        "fr": "Aucun match à afficher. Vérifiez vos filtres ou synchronisez les données.",
        "en": "No matches to display. Check your filters or sync your data.",
    },
    "no_data": {
        "fr": "Aucune donnée disponible.",
        "en": "No data available.",
    },
    "no_data_filter": {
        "fr": "Aucune donnée disponible pour ce filtre.",
        "en": "No data available for this filter.",
    },
    "insufficient_data": {
        "fr": "Données insuffisantes.",
        "en": "Not enough data.",
    },
    "insufficient_data_chart": {
        "fr": "Données insuffisantes pour afficher ce graphique.",
        "en": "Not enough data to display this chart.",
    },
    "no_events": {
        "fr": "Aucun événement trouvé.",
        "en": "No events found.",
    },
    "no_media": {
        "fr": "Aucun média à afficher avec ces filtres.",
        "en": "No media to display with these filters.",
    },
    "no_match_found": {
        "fr": "Aucun match trouvé.",
        "en": "No match found.",
    },
    "no_medals": {
        "fr": "Aucune médaille.",
        "en": "No medals.",
    },
    "match_unknown": {
        "fr": "Match inconnu",
        "en": "Unknown match",
    },
    # ── Erreurs d'affichage ─────────────────────────────────────────────────
    "cannot_display": {
        "fr": "Impossible d'afficher ce graphique.",
        "en": "Unable to display this chart.",
    },
    "cannot_generate": {
        "fr": "Impossible de générer ce graphique.",
        "en": "Unable to generate this chart.",
    },
    "error_chart": {
        "fr": "Erreur lors de l'affichage du graphique : {error}",
        "en": "Error while displaying the chart: {error}",
    },
    "error_generic": {
        "fr": "Erreur : {error}",
        "en": "Error: {error}",
    },
    "error_loading": {
        "fr": "Erreur lors du chargement : {error}",
        "en": "Error while loading: {error}",
    },
    # ── Actions ─────────────────────────────────────────────────────────────
    "loading": {
        "fr": "Chargement…",
        "en": "Loading…",
    },
    "computing": {
        "fr": "Calcul en cours…",
        "en": "Computing…",
    },
    "syncing": {
        "fr": "Synchronisation en cours…",
        "en": "Sync in progress…",
    },
    "cache_cleared": {
        "fr": "Caches vidés.",
        "en": "Caches cleared.",
    },
    "not_associated": {
        "fr": "Match: non associé",
        "en": "Match: unassociated",
    },
    "no_thumbnail": {
        "fr": "(pas de miniature générée)",
        "en": "(no thumbnail generated)",
    },
    # ── Données temporelles ─────────────────────────────────────────────────
    "missing_time_data": {
        "fr": "Données temporelles manquantes.",
        "en": "Missing time data.",
    },
    "missing_outcome_data": {
        "fr": "Données de résultat manquantes.",
        "en": "Missing outcome data.",
    },
    # ── Minimum de matchs ───────────────────────────────────────────────────
    "not_enough_matches": {
        "fr": "Pas assez de données ({count} matchs). Il en faut au moins {min} pour afficher ce graphique.",
        "en": "Not enough data ({count} matches). Need at least {min} to display this chart.",
    },
    "at_least_n_matches": {
        "fr": "Au moins {n} matchs requis.",
        "en": "At least {n} matches required.",
    },
    # ── Médias ──────────────────────────────────────────────────────────────
    "file_missing": {
        "fr": "Fichier absent : {name}",
        "en": "Missing file: {name}",
    },
    "thumbnail_too_large": {
        "fr": "Miniature trop volumineuse : {name}",
        "en": "Thumbnail too large: {name}",
    },
    "media_disabled": {
        "fr": "Les médias sont désactivés dans Paramètres → Médias.",
        "en": "Media are disabled in Settings → Media.",
    },
    "media_no_folder": {
        "fr": "Configure au moins un dossier dans Paramètres → Médias (captures et/ou vidéos).",
        "en": "Configure at least one folder in Settings → Media (screenshots and/or videos).",
    },
    "media_not_found": {
        "fr": "Aucun média trouvé.",
        "en": "No media found.",
    },
    "indexing_error": {
        "fr": "Erreur lors de l'indexation : {error}",
        "en": "Indexing error: {error}",
    },
}
