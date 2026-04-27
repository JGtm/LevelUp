// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/squad.toml

export const squadManifest = {
  "squad.header.solo_section_title": { fr: "Mes stats sur cette session", en: "My session stats" },
  "squad.header.squad_grade_label": { fr: "Grade", en: "Grade" },
  "squad.header.squad_score_base": { fr: "Base", en: "Base" },
  "squad.header.squad_score_bonus": { fr: "{base, number} (+{bonus, number})", en: "{base, number} (+{bonus, number})" },
  "squad.header.squad_section_title": { fr: "Score d'équipe", en: "Team score" },
  "squad.kpi.assists_per_match": { fr: "Assistances par match", en: "Assists per match" },
  "squad.kpi.avg_accuracy": { fr: "Précision moyenne", en: "Average accuracy" },
  "squad.kpi.avg_lifespan": { fr: "Vie moyenne", en: "Average lifespan" },
  "squad.kpi.deaths_per_match": { fr: "Morts par match", en: "Deaths per match" },
  "squad.kpi.kills_per_match": { fr: "Frags par match", en: "Kills per match" },
  "squad.kpi.kills_per_min_suffix": { fr: "/min", en: "/min" },
  "squad.kpi.matches": { fr: "Matchs sélectionnés", en: "Selected matches" },
  "squad.kpi.matches_count": { fr: "{n, plural, one {# match} other {# matchs}}", en: "{n, plural, one {# match} other {# matches}}" },
  "squad.kpi.results_summary": { fr: "Résultats", en: "Results" },
  "squad.kpi.total_duration": { fr: "Durée totale", en: "Total duration" },
  "squad.score.label_average": { fr: "Moyen", en: "Average" },
  "squad.score.label_bad": { fr: "Insuffisant", en: "Bad" },
  "squad.score.label_excellent": { fr: "Excellent", en: "Excellent" },
  "squad.score.label_good": { fr: "Bon", en: "Good" },
  "squad.score.label_poor": { fr: "Faible", en: "Poor" },
} as const

export type SquadManifestKey = keyof typeof squadManifest
