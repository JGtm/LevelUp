/**
 * relief/tint — LA TEINTE : une rampe à trois crans tirée d'UNE couleur de base.
 *
 * Principe `terrainStyle` de Den (conception, annexe teinte) : depuis la base `B`,
 * `bas = mélange(noir, B, 0,3)`, `milieu = mélange(noir, B, 0,78)`, `haut = mélange(B, blanc, 0,3)` ;
 * le gris 0 prend `bas`, 0,5 prend `milieu`, 1 prend `haut`, avec une interpolation linéaire en sRGB.
 * LES TROIS CRANS NE SE CALCULENT PAS ICI : la rampe est UNIQUE (`lib/accessibility/reliefRamp`), celle
 * que mesure le garde-fou de contraste ; ce module la lit et n'en tient aucune copie.
 *
 * La couleur de base est un PARAMETRE : le jeton `map-relief-base` (item D.5) la fournit, ce module
 * n'en connaît aucune. Il accepte une chaîne hexadécimale `#rgb` / `#rrggbb` (ce que rend
 * `resolveToken`) ou un triplet `[r, g, b]`.
 *
 * Les encres des traits (arête, courbe, mur) se tirent de la même base : mélange vers le blanc.
 */
import { reliefRampStops } from '@/lib/accessibility/reliefRamp'

import { FLAG_ON_MAP } from './constants'

/** Une couleur sRGB, trois entiers 0..255. */
export type Rgb = readonly [number, number, number]

/** La base d'une teinte : une chaîne hexadécimale ou un triplet. */
export type ReliefBaseColor = string | Rgb

/** La rampe et les encres de traits d'une teinte. */
export interface ReliefRamp {
  low: Rgb
  mid: Rgb
  high: Rgb
  /** Encre des ruptures de hauteur. */
  edge: Rgb
  /** Encre des courbes de niveau. */
  contour: Rgb
  /** Encre des murs du maillage. */
  wall: Rgb
}

/** Mélanges vers le blanc des encres de traits (les trois crans de la rampe viennent de `reliefRamp.ts`). */
const EDGE_LIGHTEN = 0.65
const CONTOUR_LIGHTEN = 0.45
const WALL_LIGHTEN = 0.8

const HEX = /^#?([0-9a-f]{3}|[0-9a-f]{6})$/i

/** parseHexColor lit `#rgb` ou `#rrggbb` ; `null` pour toute autre chaîne. */
export function parseHexColor(value: string): Rgb | null {
  const m = HEX.exec(value.trim())
  if (!m) return null
  const h = m[1].length === 3 ? [...m[1]].map((c) => c + c).join('') : m[1]
  return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
}

/** Un triplet valide : trois entiers 0..255. */
function isRgb(c: readonly number[]): boolean {
  return c.length === 3 && c.every((v) => Number.isInteger(v) && v >= 0 && v <= 255)
}

/** toRgb normalise une base ; `null` si elle n'est pas une couleur lisible. */
export function toRgb(base: ReliefBaseColor): Rgb | null {
  if (typeof base === 'string') return parseHexColor(base)
  return isRgb(base) ? base : null
}

/** La couleur `#rrggbb` d'un triplet, pour interroger la rampe unique. */
function toHex(c: Rgb): string {
  return '#' + c.map((v) => v.toString(16).padStart(2, '0')).join('')
}

/** Mélange de la base vers le blanc : `amount` de blanc. */
function towardWhite(c: Rgb, amount: number): Rgb {
  const lift = (v: number): number => Math.round(v + (255 - v) * amount)
  return [lift(c[0]), lift(c[1]), lift(c[2])]
}

/** Un cran de la rampe unique, toujours un `#rrggbb` valide. */
function parseStop(hex: string): Rgb {
  return parseHexColor(hex) as Rgb
}

/** reliefRamp tire la rampe et les encres de traits d'une base. Une base illisible rend `null`. */
export function reliefRamp(base: ReliefBaseColor): ReliefRamp | null {
  const c = toRgb(base)
  if (!c) return null
  const stops = reliefRampStops(toHex(c))
  return {
    low: parseStop(stops.low),
    mid: parseStop(stops.mid),
    high: parseStop(stops.high),
    edge: towardWhite(c, EDGE_LIGHTEN),
    contour: towardWhite(c, CONTOUR_LIGHTEN),
    wall: towardWhite(c, WALL_LIGHTEN),
  }
}

/** rgbToCss écrit une couleur pour un contexte 2D. */
export function rgbToCss(c: Rgb): string {
  return `rgb(${c[0]},${c[1]},${c[2]})` // color-allow(fonction-couleur): sérialise une teinte CALCULEE depuis le jeton, pas un littéral
}

/** buildTintTable fait la table de 256 couleurs (3 octets chacune) d'une rampe : gris -> couleur. */
export function buildTintTable(ramp: Pick<ReliefRamp, 'low' | 'mid' | 'high'>): Uint8ClampedArray {
  const table = new Uint8ClampedArray(256 * 3)
  for (let g = 0; g < 256; g++) {
    const f = g / 255
    const lower = f < 0.5
    const a = lower ? ramp.low : ramp.mid
    const b = lower ? ramp.mid : ramp.high
    const k = lower ? f * 2 : f * 2 - 1
    for (let i = 0; i < 3; i++) table[g * 3 + i] = a[i] + (b[i] - a[i]) * k
  }
  return table
}

/** Une image teintée prête pour `putImageData` : RGBA, 4 octets par pixel. */
export interface TintedImage {
  width: number
  height: number
  rgba: Uint8ClampedArray
}

/** Ce que la teinte lit : le gris, l'opacité de la coupe (0..255, 255 partout sans coupe) et les drapeaux. */
export interface TintSource {
  width: number
  height: number
  grey: Uint8Array
  alpha: Uint8Array
  flags: Uint8Array
}

/**
 * tintRelief teinte le gris : les pixels de la carte (`FLAG_ON_MAP`) prennent la couleur de la table à
 * leur gris et l'opacité de la coupe ; les autres restent transparents.
 */
export function tintRelief(source: TintSource, table: Uint8ClampedArray): TintedImage {
  const { width, height, grey, alpha, flags } = source
  const rgba = new Uint8ClampedArray(width * height * 4)
  for (let p = 0, q = 0; p < grey.length; p++, q += 4) {
    const a = alpha[p]
    if ((flags[p] & FLAG_ON_MAP) === 0 || a === 0) continue
    const t = grey[p] * 3
    rgba[q] = table[t]
    rgba[q + 1] = table[t + 1]
    rgba[q + 2] = table[t + 2]
    rgba[q + 3] = a
  }
  return { width, height, rgba }
}
