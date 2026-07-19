// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/frags.toml

export const fragsManifest = {
  "frags.authority.estimated": { fr: "estimé", en: "estimated" },
  "frags.authority.exact": { fr: "exact", en: "exact" },
  "frags.charts.center_total_label": { fr: "Frags", en: "Kills" },
  "frags.charts.sunburst_title": { fr: "Répartition des frags", en: "Frag distribution" },
  "frags.charts.weapon_breakdown_title": { fr: "Frags par arme", en: "Kills by weapon" },
  "frags.class.grenade": { fr: "Grenade", en: "Grenade" },
  "frags.class.heavy": { fr: "Lourde", en: "Heavy" },
  "frags.class.melee": { fr: "Mêlée", en: "Melee" },
  "frags.class.shoulder": { fr: "Épaule", en: "Shoulder" },
  "frags.class.sidearm": { fr: "Poing", en: "Sidearm" },
  "frags.class.spartan_ability": { fr: "Capacités spartanes", en: "Spartan abilities" },
  "frags.class.unattributed": { fr: "Non attribué", en: "Unattributed" },
  "frags.empty.no_data": { fr: "Aucun frag sur ce périmètre", en: "No kills in this scope" },
  "frags.role.assassination": { fr: "Assassinat", en: "Assassination" },
  "frags.role.automatic": { fr: "Automatique", en: "Automatic" },
  "frags.role.direct_melee": { fr: "Corps-à-corps direct", en: "Direct melee" },
  "frags.role.ground_pound": { fr: "Frappe au sol", en: "Ground pound" },
  "frags.role.power": { fr: "Arme Puissante", en: "Power weapon" },
  "frags.role.precision": { fr: "Précision", en: "Precision" },
  "frags.role.shotgun": { fr: "Fusil à pompe", en: "Shotgun" },
  "frags.role.shoulder_bash": { fr: "Charge d'épaule", en: "Shoulder bash" },
  "frags.role.sidearm": { fr: "Arme de poing", en: "Sidearm" },
  "frags.role.sniper": { fr: "Sniper", en: "Sniper" },
  "frags.role.special": { fr: "Spéciale", en: "Special" },
  "frags.tooltip.share_class": { fr: "{pct} % de {className}", en: "{pct}% of {className}" },
  "frags.tooltip.share_total": { fr: "{pct} % du total", en: "{pct}% of total" },
} as const

export type FragsManifestKey = keyof typeof fragsManifest
