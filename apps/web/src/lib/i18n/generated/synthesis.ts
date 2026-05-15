// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/synthesis.toml

export const synthesisManifest = {
  "synthesis.empty.activity_description": { fr: "Aucune activité enregistrée sur la période sélectionnée.", en: "No activity recorded for the selected period." },
  "synthesis.empty.activity_unavailable": { fr: "Activité indisponible", en: "Activity unavailable" },
  "synthesis.empty.no_comparison": { fr: "Pas assez de données pour une comparaison Solo / Escouade", en: "Not enough data for Solo / Squad comparison" },
  "synthesis.empty.no_data": { fr: "Aucune donnée disponible pour cette période", en: "No data available for this period" },
  "synthesis.errors.load_failed": { fr: "Erreur lors du chargement de la synthèse", en: "Failed to load synthesis" },
  "synthesis.errors.retry": { fr: "Réessayer", en: "Retry" },
  "synthesis.filters.experience": { fr: "Expérience", en: "Experience" },
  "synthesis.filters.experience_all": { fr: "Toutes", en: "All" },
  "synthesis.filters.experience_ranked": { fr: "Classé", en: "Ranked" },
  "synthesis.filters.experience_unranked": { fr: "Non classé", en: "Unranked" },
  "synthesis.filters.modes": { fr: "Modes", en: "Modes" },
  "synthesis.filters.playlists": { fr: "Playlists", en: "Playlists" },
  "synthesis.filters.reset": { fr: "Réinitialiser", en: "Reset" },
  "synthesis.highlights.best_matches": { fr: "Meilleurs matchs", en: "Best matches" },
  "synthesis.highlights.no_remarkable_week": { fr: "Aucune semaine remarquable", en: "No remarkable week" },
  "synthesis.highlights.tough_matches": { fr: "Matchs difficiles", en: "Tough matches" },
  "synthesis.overview.losses": { fr: "Défaites", en: "Losses" },
  "synthesis.overview.matches_total": { fr: "Matchs joués", en: "Matches played" },
  "synthesis.overview.victories": { fr: "Victoires", en: "Wins" },
  "synthesis.overview.win_rate": { fr: "Taux de victoire", en: "Win rate" },
  "synthesis.period.all": { fr: "Tout", en: "All" },
  "synthesis.period.label": { fr: "Période", en: "Period" },
  "synthesis.scope.matches_count": { fr: "{n, plural, one {# match} other {# matchs}}", en: "{n, plural, one {# match} other {# matches}}" },
  "synthesis.scope.solo_label": { fr: "Solo", en: "Solo" },
  "synthesis.scope.squad_label": { fr: "Escouade", en: "Squad" },
  "synthesis.section.activity": { fr: "Activité par jour et heure", en: "Activity by day and hour" },
  "synthesis.section.breakdown_map": { fr: "Par carte", en: "By map" },
  "synthesis.section.breakdown_mode": { fr: "Par mode", en: "By mode" },
  "synthesis.section.comparison": { fr: "Comparaison Solo / Escouade", en: "Solo / Squad comparison" },
  "synthesis.section.highlights": { fr: "Meilleurs matchs", en: "Best matches" },
  "synthesis.section.overview": { fr: "Vue d'ensemble", en: "Overview" },
  "synthesis.section.relations": { fr: "Relations", en: "Relations" },
} as const

export type SynthesisManifestKey = keyof typeof synthesisManifest
