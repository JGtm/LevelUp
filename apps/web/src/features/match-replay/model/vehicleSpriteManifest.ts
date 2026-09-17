/**
 * vehicleSpriteManifest.ts — LECTURE PURE du manifeste des sprites véhicules (`index.json`,
 * `static/vehicles-assets/{slug}/replay/`) et règle de la BOÎTE DU VÉHICULE dans l'image.
 *
 * POURQUOI CE FICHIER EXISTE (2026-09-16). Les sprites redessinés portent une BORDURE en couche
 * séparée (`{famille}_outline.png` : anneau blanc + liseré noir, jamais teinté). Pour la loger,
 * le canevas des deux images a grandi de `pad` pixels de CHAQUE côté, symétriquement, sans
 * rééchantillonnage (10 mm/px inchangé). Tout ce qui mesure le véhicule — longueur à l'écran,
 * ancres d'armes en fractions du sprite, rayon d'explosion, place des noms — doit donc raisonner
 * sur la boîte du véhicule (`naturel − 2·pad`), pas sur l'image : sans cette soustraction, le
 * véhicule grandirait de la bordure et les ancres d'armes glisseraient vers l'extérieur.
 *
 * AUCUN CANVAS ICI : le chargement et la cuisson restent dans `useReplayVehicles`.
 */

/** Une entrée brute de `index.json` : seuls les champs utiles au rejeu sont lus. */
interface VehicleManifestRawEntry {
  famille?: unknown
  scale_mm_per_px?: unknown
  outline?: unknown
  pad?: unknown
}

/** Ce que le rejeu retient d'une famille dimensionnée. */
export interface VehicleManifestEntry {
  mmPerPx: number
  /** Fichier de la bordure (ex. `warthog_outline.png`), ou `null` : sprite sans bordure. */
  outline: string | null
  /** Marge ajoutée de chaque côté de l'image autour de la boîte du véhicule (px source). */
  pad: number
}

/**
 * parseVehicleManifest — `index.json` -> famille -> entrée. Une famille sans `scale_mm_per_px`
 * (élément de carte sans asset) n'est pas retenue : sans échelle, rien ne se dessine. Un `pad`
 * absent ou invalide vaut 0 (sprite historique sans marge), jamais une valeur inventée.
 */
export function parseVehicleManifest(raw: unknown): Map<string, VehicleManifestEntry> {
  const map = new Map<string, VehicleManifestEntry>()
  if (!Array.isArray(raw)) return map
  for (const entry of raw as VehicleManifestRawEntry[]) {
    if (typeof entry?.famille !== 'string' || typeof entry.scale_mm_per_px !== 'number') continue
    const outline = typeof entry.outline === 'string' && entry.outline !== '' ? entry.outline : null
    const pad = typeof entry.pad === 'number' && entry.pad > 0 ? entry.pad : 0
    map.set(entry.famille, { mmPerPx: entry.scale_mm_per_px, outline, pad })
  }
  return map
}

/**
 * vehicleBodyPx — la boîte du véhicule dans une image de `naturalPx` pixels, marge retirée des
 * deux côtés. Une marge qui mangerait toute l'image est ignorée (image incohérente avec son
 * manifeste) : la dimension naturelle est rendue telle quelle plutôt qu'une taille nulle.
 */
export function vehicleBodyPx(naturalPx: number, pad: number): number {
  const body = naturalPx - 2 * pad
  return body > 0 ? body : naturalPx
}

/** Identifiant d'asset (sans extension) d'un fichier du manifeste, pour `staticAssetURL`. */
export function vehicleManifestAssetId(file: string): string {
  return `replay/${file.replace(/\.png$/i, '')}`
}
