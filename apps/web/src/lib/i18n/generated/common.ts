// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/common.toml

export const commonManifest = {
  "common.empty.error": { fr: "Erreur de chargement", en: "Loading error" },
  "common.empty.loading": { fr: "Chargement…", en: "Loading…" },
  "common.empty.no_data": { fr: "Aucune donnée à afficher", en: "No data to display" },
  "common.kpi.accuracy": { fr: "Précision", en: "Accuracy" },
  "common.kpi.assists": { fr: "Assistances", en: "Assists" },
  "common.kpi.deaths": { fr: "Morts", en: "Deaths" },
  "common.kpi.kda": { fr: "K/D/A", en: "K/D/A" },
  "common.kpi.kills": { fr: "Frags", en: "Kills" },
  "common.kpi.matches_count": { fr: "{n, plural, one {# match} other {# matchs}}", en: "{n, plural, one {# match} other {# matches}}" },
  "common.outcome.dnf": { fr: "Abandon", en: "DNF" },
  "common.outcome.loss": { fr: "Défaite", en: "Loss" },
  "common.outcome.tie": { fr: "Égalité", en: "Tie" },
  "common.outcome.win": { fr: "Victoire", en: "Win" },
  "common.period.all": { fr: "Toutes les périodes", en: "All periods" },
  "common.period.last_1m": { fr: "Dernier mois", en: "Last month" },
  "common.period.last_1w": { fr: "Dernière semaine", en: "Last week" },
  "common.period.last_1y": { fr: "Dernière année", en: "Last year" },
  "common.period.last_2y": { fr: "2 dernières années", en: "Last 2 years" },
} as const

export type CommonManifestKey = keyof typeof commonManifest
