// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/tactical.toml

export const tacticalManifest = {
  "tactical.maps.coverage": { fr: "{maps, plural, one {# carte jouée} other {# cartes jouées}}, {matches, plural, one {# match} other {# matchs}}", en: "{maps, plural, one {# map played} other {# maps played}}, {matches, plural, one {# match} other {# matches}}" },
  "tactical.maps.empty_description": { fr: "Aucun match ne correspond aux filtres en cours. Élargis la période ou retire un filtre.", en: "No match matches the current filters. Widen the period or remove a filter." },
  "tactical.maps.empty_title": { fr: "Aucune carte jouée", en: "No map played" },
  "tactical.maps.error": { fr: "Les cartes n'ont pas pu être chargées.", en: "Maps could not be loaded." },
  "tactical.maps.floor_note": { fr: "Une carte s'ouvre à partir de {floor} matchs : en dessous, une lecture de placement ne mesure que du bruit.", en: "A map opens from {floor} matches on: below that, a positioning reading measures nothing but noise." },
  "tactical.maps.floor_reason": { fr: "{n, plural, one {# match} other {# matchs}} sur {floor} requis", en: "{n, plural, one {# match} other {# matches}} of {floor} required" },
  "tactical.maps.label": { fr: "Cartes jouées", en: "Maps played" },
  "tactical.maps.loading": { fr: "Chargement des cartes…", en: "Loading maps…" },
  "tactical.maps.matches": { fr: "{n, plural, one {# match} other {# matchs}}", en: "{n, plural, one {# match} other {# matches}}" },
  "tactical.maps.record": { fr: "{wins, plural, one {# victoire} other {# victoires}} · {losses, plural, one {# défaite} other {# défaites}}", en: "{wins, plural, one {# win} other {# wins}} · {losses, plural, one {# loss} other {# losses}}" },
  "tactical.maps.record_label": { fr: "{wins, plural, one {# victoire} other {# victoires}} et {losses, plural, one {# défaite} other {# défaites}} sur {n, plural, one {# match} other {# matchs}}", en: "{wins, plural, one {# win} other {# wins}} and {losses, plural, one {# loss} other {# losses}} out of {n, plural, one {# match} other {# matches}}" },
  "tactical.maps.select": { fr: "Sélectionner {map}", en: "Select {map}" },
  "tactical.maps.selected": { fr: "Carte sélectionnée", en: "Selected map" },
  "tactical.maps.title": { fr: "Cartes jouées", en: "Maps played" },
} as const

export type TacticalManifestKey = keyof typeof tacticalManifest
