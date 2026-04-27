// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/citations.toml

export const citationsManifest = {
  "citations.commendations.col_name": { fr: "Nom", en: "Name" },
  "citations.commendations.col_progress": { fr: "Progression", en: "Progress" },
  "citations.commendations.col_tier": { fr: "Palier", en: "Tier" },
  "citations.commendations.no_data": { fr: "Aucune commendation à afficher pour cette sélection.", en: "No commendation to display for this selection." },
  "citations.deltas.delta": { fr: "Delta", en: "Delta" },
  "citations.deltas.filtered_total": { fr: "Total filtré", en: "Filtered total" },
  "citations.deltas.unfiltered_total": { fr: "Total complet", en: "Full total" },
  "citations.distribution.axis_count": { fr: "Nombre", en: "Count" },
  "citations.distribution.title": { fr: "Distribution des médailles", en: "Medals distribution" },
  "citations.distribution.unavailable_description": { fr: "Aucun graphique de distribution n'a été généré pour la sélection actuelle.", en: "No distribution chart generated for the current selection." },
  "citations.distribution.unavailable_title": { fr: "Distribution indisponible", en: "Distribution unavailable" },
  "citations.empty.no_data": { fr: "Citations indisponibles", en: "Citations unavailable" },
  "citations.empty.no_data_description": { fr: "Aucune réponse exploitable n'a été renvoyée pour cette page.", en: "No exploitable response returned for this page." },
  "citations.errors.load_failed": { fr: "Erreur lors du chargement des citations", en: "Failed to load citations" },
  "citations.errors.retry": { fr: "Réessayer", en: "Retry" },
  "citations.medals.col_count_filtered": { fr: "Filtré", en: "Filtered" },
  "citations.medals.col_count_total": { fr: "Total", en: "Total" },
  "citations.medals.col_name": { fr: "Médaille", en: "Medal" },
  "citations.section.commendations": { fr: "Commendations", en: "Commendations" },
  "citations.section.distribution": { fr: "Distribution", en: "Distribution" },
  "citations.section.medals_summary": { fr: "Médailles", en: "Medals" },
} as const

export type CitationsManifestKey = keyof typeof citationsManifest
