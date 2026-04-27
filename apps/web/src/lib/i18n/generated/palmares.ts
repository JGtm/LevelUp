// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/palmares.toml

export const palmaresManifest = {
  "palmares.empty.unavailable_description": { fr: "Aucune réponse exploitable n'a été renvoyée pour cette page.", en: "No exploitable response returned for this page." },
  "palmares.empty.unavailable_title": { fr: "Palmarès indisponible", en: "Palmares unavailable" },
  "palmares.errors.retry": { fr: "Réessayer", en: "Retry" },
  "palmares.page.subtitle": { fr: "Classements, relations de jeu, passes saisonniers et comparaison joueur à joueur regroupés dans un même hub.", en: "Leaderboards, gameplay relations, season passes and head-to-head comparison gathered in a single hub." },
  "palmares.page.title": { fr: "Palmarès", en: "Palmares" },
  "palmares.tabs.compare": { fr: "Comparaison", en: "Comparison" },
  "palmares.tabs.leaderboard": { fr: "Classements", en: "Leaderboards" },
  "palmares.tabs.relations": { fr: "Relations", en: "Relations" },
  "palmares.tabs.season_pass": { fr: "Pass saisonnier", en: "Season pass" },
} as const

export type PalmaresManifestKey = keyof typeof palmaresManifest
