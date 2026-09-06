// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/tactical.toml

export const tacticalManifest = {
  "tactical.filter.experience": { fr: "Expérience", en: "Experience" },
  "tactical.filter.experience_all": { fr: "Toutes", en: "All" },
  "tactical.filter.experience_ranked": { fr: "Classé", en: "Ranked" },
  "tactical.filter.experience_unranked": { fr: "Non classé", en: "Unranked" },
  "tactical.filter.modes": { fr: "Modes", en: "Modes" },
  "tactical.filter.playlists": { fr: "Playlists", en: "Playlists" },
  "tactical.filter.reset": { fr: "Réinitialiser les filtres", en: "Reset filters" },
  "tactical.filter.sessions": { fr: "Sessions", en: "Sessions" },
  "tactical.filter.sessions_off_list": { fr: "{n, plural, one {# session épinglée n'est pas dans la liste courante} other {# sessions épinglées ne sont pas dans la liste courante}} : {names}. Le filtre reste appliqué.", en: "{n, plural, one {# pinned session is not in the current list} other {# pinned sessions are not in the current list}}: {names}. The filter still applies." },
  "tactical.filter.squad_placeholder": { fr: "Coéquipiers ({n})", en: "Teammates ({n})" },
  "tactical.filter.unknown_teammate_description": { fr: "Impossible d'identifier {names} parmi tes coéquipiers. Retire ce nom de la composition pour voir les cartes.", en: "{names} could not be identified among your teammates. Remove that name from the squad to see the maps." },
  "tactical.filter.unknown_teammate_title": { fr: "Coéquipier introuvable", en: "Teammate not found" },
  "tactical.filter.view": { fr: "Vue", en: "View" },
  "tactical.filter.view_all": { fr: "Tous les matchs", en: "All matches" },
  "tactical.filter.view_solo": { fr: "Solo", en: "Solo" },
  "tactical.filter.view_squad": { fr: "En escouade", en: "With a squad" },
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
