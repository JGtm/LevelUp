/**
 * reliefRamp.ts — LA RAMPE DE CLARTÉ du fond « Relief », tirée du jeton `map-relief-base`.
 *
 * C'EST LA RAMPE DU FOND RELIEF, ET LA SEULE : tout code qui teinte le relief (table de teinte du
 * rendu, garde-fou de contraste) la lit ici, jamais une seconde copie des trois mélanges.
 *
 * Une seule teinte par palette (jeton), et la rampe qui en sort : sombre en bas, la base au
 * milieu, claire en haut. Elle est calculée ici, pas portée par des jetons de plus : un jeton par
 * palier aurait multiplié les valeurs à tenir pour une seule information (la teinte).
 *
 *     bas   = mélange(noir, base, 0,30)
 *     milieu = mélange(noir, base, 0,78)
 *     haut  = mélange(base, blanc, 0,30)
 *     gris 0 -> bas, 0,5 -> milieu, 1 -> haut, interpolation linéaire en sRGB entre ces trois points
 *
 * Les coefficients et l'interpolation sont ceux de la mesure de teinte (mesure de teinte, d'après
 * le terrain de Den) ; les garde-fous de `mapReliefTokens.test.ts` mesurent cette rampe-là.
 */

type Rgb = readonly [number, number, number]

const BLACK: Rgb = [0, 0, 0]
const WHITE: Rgb = [255, 255, 255]

/** Part de la base dans le bas de la rampe (mélangée à du noir). */
export const RELIEF_LOW_SHARE = 0.3
/** Part de la base au milieu de la rampe (mélangée à du noir). */
export const RELIEF_MID_SHARE = 0.78
/** Part du blanc dans le haut de la rampe (mélangé à la base). */
export const RELIEF_HIGH_WHITE_SHARE = 0.3

function parseHex(hex: string): Rgb {
  const h = hex.replace(/^#/, '')
  return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
}

function toHex(c: Rgb): string {
  return '#' + c.map((v) => Math.round(Math.max(0, Math.min(255, v))).toString(16).padStart(2, '0')).join('')
}

function mix(a: Rgb, b: Rgb, t: number): Rgb {
  return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t]
}

/** Les trois points de la rampe (#rrggbb) : bas sombre, milieu, haut clair. */
export interface ReliefRampStops {
  low: string
  mid: string
  high: string
}

function stopsRgb(base: string): { low: Rgb; mid: Rgb; high: Rgb } {
  const b = parseHex(base)
  return {
    low: mix(BLACK, b, RELIEF_LOW_SHARE),
    mid: mix(BLACK, b, RELIEF_MID_SHARE),
    high: mix(b, WHITE, RELIEF_HIGH_WHITE_SHARE),
  }
}

/** reliefRampStops rend les trois points de la rampe pour la base `base` (#rrggbb). */
export function reliefRampStops(base: string): ReliefRampStops {
  const { low, mid, high } = stopsRgb(base)
  return { low: toHex(low), mid: toHex(mid), high: toHex(high) }
}

/**
 * reliefRampHex rend la couleur (#rrggbb) du palier de gris `gray` (0 = bas, 1 = haut ; hors
 * bornes, ramené dans [0, 1]) pour la base `base` (#rrggbb).
 */
export function reliefRampHex(base: string, gray: number): string {
  const { low, mid, high } = stopsRgb(base)
  const g = Math.max(0, Math.min(1, gray))
  return toHex(g < 0.5 ? mix(low, mid, g * 2) : mix(mid, high, g * 2 - 1))
}
