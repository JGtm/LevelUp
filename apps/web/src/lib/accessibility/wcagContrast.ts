/**
 * wcagContrast.ts — Helpers de contraste WCAG 2.0.
 *
 * Implémente la formule officielle de contraste relatif WCAG 2.0
 * (https://www.w3.org/TR/WCAG20/#contrast-ratiodef) pour vérifier que
 * les paires fond/texte des badges narratifs atteignent AA (≥ 4.5:1).
 *
 * S'inspire de afonsolopez/colorblind (port Go) — réimplémenté en TS
 * sans dépendance.
 *
 * Seuils :
 *   AA texte normal   : ≥ 4.5
 *   AA texte large    : ≥ 3.0  (≥ 18pt ou ≥ 14pt bold)
 *   AAA               : ≥ 7.0
 */

export type WcagGrade = 'fail' | 'AA-large' | 'AA' | 'AAA'

const HEX_RE = /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/

function expandShortHex(hex: string): string {
  // #abc → #aabbcc
  if (hex.length === 4) {
    return '#' + hex[1]! + hex[1]! + hex[2]! + hex[2]! + hex[3]! + hex[3]!
  }
  return hex
}

function parseHex(hex: string): [number, number, number] {
  if (!HEX_RE.test(hex)) {
    throw new Error(`Hex invalide: "${hex}"`)
  }
  const expanded = expandShortHex(hex)
  const r = parseInt(expanded.slice(1, 3), 16)
  const g = parseInt(expanded.slice(3, 5), 16)
  const b = parseInt(expanded.slice(5, 7), 16)
  return [r, g, b]
}

function channelLuminance(c: number): number {
  const v = c / 255
  return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)
}

/** Luminance relative WCAG 2.0 d'une couleur hex (0..1, 0=noir, 1=blanc). */
export function relLuminance(hex: string): number {
  const [r, g, b] = parseHex(hex)
  const [rl, gl, bl] = [channelLuminance(r), channelLuminance(g), channelLuminance(b)]
  return 0.2126 * rl + 0.7152 * gl + 0.0722 * bl
}

/** Ratio de contraste WCAG 2.0 entre deux couleurs hex (1..21). */
export function contrastRatio(hex1: string, hex2: string): number {
  const l1 = relLuminance(hex1)
  const l2 = relLuminance(hex2)
  const [hi, lo] = l1 >= l2 ? [l1, l2] : [l2, l1]
  return (hi + 0.05) / (lo + 0.05)
}

/** Note WCAG d'un ratio donné. */
export function wcagGrade(ratio: number): WcagGrade {
  if (ratio >= 7) return 'AAA'
  if (ratio >= 4.5) return 'AA'
  if (ratio >= 3) return 'AA-large'
  return 'fail'
}
