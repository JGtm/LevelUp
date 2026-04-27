// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/match_view.toml

export const match_viewManifest = {
  "match_view.empty.no_data": { fr: "Aucune donnée disponible pour ce match", en: "No data available for this match" },
  "match_view.error.load_failed": { fr: "Impossible de charger le détail du match", en: "Failed to load match detail" },
  "match_view.header.duration_label": { fr: "Durée", en: "Duration" },
  "match_view.header.outcome_label": { fr: "Résultat", en: "Outcome" },
  "match_view.header.performance_score_label": { fr: "Score de performance", en: "Performance score" },
  "match_view.header.title": { fr: "Détail du match", en: "Match detail" },
  "match_view.header.waypoint_link": { fr: "Voir sur Halo Waypoint", en: "View on Halo Waypoint" },
  "match_view.tabs.citations": { fr: "Citations", en: "Citations" },
  "match_view.tabs.combat": { fr: "Combat", en: "Combat" },
  "match_view.tabs.media": { fr: "Médias", en: "Media" },
  "match_view.tabs.summary": { fr: "Résumé", en: "Summary" },
  "match_view.tabs.team": { fr: "Équipe", en: "Team" },
  "narrative.dominance.contre_remontada": { fr: "Contre-remontada", en: "Counter-comeback" },
  "narrative.dominance.debandade": { fr: "Débandade", en: "Collapse" },
  "narrative.dominance.domination": { fr: "Domination", en: "Domination" },
  "narrative.dominance.humiliation": { fr: "Humiliation", en: "Humiliation" },
  "narrative.dominance.remontada": { fr: "Remontada", en: "Comeback" },
} as const

export type Match_viewManifestKey = keyof typeof match_viewManifest
