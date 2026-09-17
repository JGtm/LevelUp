/**
 * colorDistance.ts — distance perceptuelle et simulation daltonisme, SOURCE UNIQUE.
 *
 * Centralisé le 2026-09-17 (plan PLAN_COULEURS_STATS_COMBAT) : la conversion OKLab et
 * la distance ΔE vivaient en copie dans `squadPlayerTokens.test.ts` et
 * `scales/fragClass.guard.test.ts` ; le garde-fou des stats de combat en aurait fait une
 * troisième. Les trois garde-fous importent d'ici, et `colorDistance.guard.test.ts`
 * interdit toute nouvelle copie de la matrice OKLab.
 *
 * - `deltaE` : distance euclidienne OKLab × 100 (Björn Ottosson, 2020).
 * - `simulateCvd` : Machado, Oliveira & Fernandes (2009), sévérité 1.0, en RGB linéaire.
 */

function parseHex(hex: string): [number, number, number] {
  const h = hex.replace(/^#/, '')
  return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
}

const toLinear = (c: number): number => {
  const v = c / 255
  return v <= 0.04045 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)
}

const toSrgb = (v: number): number => {
  const c = v <= 0.0031308 ? 12.92 * v : 1.055 * Math.pow(v, 1 / 2.4) - 0.055
  return Math.max(0, Math.min(255, Math.round(c * 255)))
}

/** sRGB (hex #RRGGBB) → OKLab. */
export function oklab(hex: string): [number, number, number] {
  const [r, g, b] = parseHex(hex).map(toLinear)
  const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b)
  const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b)
  const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b)
  return [
    0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s,
    1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s,
    0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s,
  ]
}

/** Distance euclidienne OKLab × 100 (≈ 1 unité = 1 « just noticeable step »). */
export function deltaE(hexA: string, hexB: string): number {
  const a = oklab(hexA)
  const b = oklab(hexB)
  return 100 * Math.hypot(a[0] - b[0], a[1] - b[1], a[2] - b[2])
}

/** Machado et al. (2009), sévérité 1.0 — matrices en RGB linéaire, ligne par ligne. */
const CVD_MATRICES = {
  protanopie: [0.152286, 1.052583, -0.204868, 0.114503, 0.786281, 0.099216, -0.003882, -0.048116, 1.051998],
  deuteranopie: [0.367322, 0.860646, -0.227968, 0.280085, 0.672501, 0.047413, -0.01182, 0.04294, 0.968881],
} as const

export type CvdKind = keyof typeof CVD_MATRICES

/** Couleur perçue sous protanopie / deutéranopie (hex #rrggbb). */
export function simulateCvd(hex: string, kind: CvdKind): string {
  const m = CVD_MATRICES[kind]
  const [r, g, b] = parseHex(hex).map(toLinear)
  const out = [m[0] * r + m[1] * g + m[2] * b, m[3] * r + m[4] * g + m[5] * b, m[6] * r + m[7] * g + m[8] * b]
  return '#' + out.map((v) => toSrgb(v).toString(16).padStart(2, '0')).join('')
}
