// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/first_blood.toml

export const firstBloodManifest = {
  "first_blood.empty": { fr: "Aucun premier frag ni première mort sur ce périmètre", en: "No first kill or first death in this scope" },
  "first_blood.label.advance": { fr: "{gap} d'avance", en: "{gap} ahead" },
  "first_blood.label.median_prefix": { fr: "méd.", en: "med." },
  "first_blood.title": { fr: "Premier frag / première mort", en: "First kill / first death" },
  "first_blood.tooltip.first_death": { fr: "{player} · {map} · {mode} · {date} · première mort {time}", en: "{player} · {map} · {mode} · {date} · first death {time}" },
  "first_blood.tooltip.first_kill": { fr: "{player} · {map} · {mode} · {date} · premier frag {time}", en: "{player} · {map} · {mode} · {date} · first kill {time}" },
  "first_blood.tooltip.gap": { fr: "fenêtre d'avance médiane : {gap}", en: "median advance window: {gap}" },
  "first_blood.tooltip.median_death": { fr: "médiane première mort {time} ({n}/{total, plural, one {# match} other {# matchs}})", en: "median first death {time} ({n}/{total, plural, one {# match} other {# matches}})" },
  "first_blood.tooltip.median_kill": { fr: "médiane premier frag {time} ({n}/{total, plural, one {# match} other {# matchs}})", en: "median first kill {time} ({n}/{total, plural, one {# match} other {# matches}})" },
} as const

export type FirstBloodManifestKey = keyof typeof firstBloodManifest
