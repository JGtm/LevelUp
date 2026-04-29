// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/asset_drawer.toml

export const assetDrawerManifest = {
  "asset_drawer.mini_tab": { fr: "Référentiel", en: "Reference" },
  "asset_drawer.tab.maps": { fr: "Cartes", en: "Maps" },
  "asset_drawer.tab.weapons": { fr: "Armes", en: "Weapons" },
  "asset_drawer.search.placeholder": { fr: "Rechercher…", en: "Search…" },
  "asset_drawer.empty.maps": { fr: "Aucune map trouvée.", en: "No map found." },
  "asset_drawer.empty.weapons": { fr: "Aucune arme trouvée.", en: "No weapon found." },
  "asset_drawer.empty.loading": { fr: "Chargement…", en: "Loading…" },
  "asset_drawer.empty.error": { fr: "Erreur de chargement.", en: "Loading error." },
  "asset_drawer.toggle.open": { fr: "Ouvrir le référentiel visuel", en: "Open visual reference" },
  "asset_drawer.toggle.close": { fr: "Fermer le référentiel visuel", en: "Close visual reference" },
} as const

export type AssetDrawerManifestKey = keyof typeof assetDrawerManifest
