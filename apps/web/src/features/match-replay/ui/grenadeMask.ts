/**
 * grenadeMask — la vignette d'un type de grenade, lue dans le catalogue du document.
 */
import type { CatalogLabel } from '../i18n/catalogLabel'

/** Ce que la puce a besoin de savoir pour peindre la vignette : l'URL, et son mode. */
export interface GrenadeIconRef {
  url: string
  /** Vrai = masque à teindre (currentColor) ; faux = image finie. */
  tinted: boolean
}

/**
 * grenadeMaskOf — la vignette d'un rang : le MASQUE DE HUD cuit dans l'artefact, la
 * version PLEINE de la maquette (option 2a du handoff 2026-08-27 — elle remplace l'image
 * versionnée de la planche du 16/08, dont le module `grenadeIcon.ts` est parti avec elle).
 * Teint par currentColor, le masque suit le thème ET l'encre `warning` du type équipé.
 * Sans vignette : null — l'appelant garde le libellé, jamais l'image d'un autre type.
 */
export function grenadeMaskOf(label: CatalogLabel | undefined): GrenadeIconRef | null {
  return label?.img ? { url: label.img, tinted: !!label.tinted } : null
}
