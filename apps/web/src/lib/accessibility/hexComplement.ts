/**
 * hexComplement.ts — Couleur complémentaire (rotation HSL +180°) d'un hex.
 *
 * Usage : barres "négatives" des charts escouade (morts, FDA…).
 * La couleur complémentaire est perceptuellement distincte de la couleur source
 * tout en restant dérivée de la palette active — l'accessibilité est maintenue
 * sans introduire de nouveau token ni de hex codé en dur dans les features.
 *
 * Quand la palette change (ex. Okabe-Ito), colorByPlayer change aussi, donc
 * les compléments calculés ici changent de concert : aucune intervention manuelle.
 */

function hexToRgb(hex: string): [number, number, number] | null {
  // Accepte #RGB, #RRGGBB (avec ou sans #)
  const h = hex.replace(/^#/, '')
  const expanded = h.length === 3
    ? h.split('').map((c) => c + c).join('')
    : h
  const m = /^([0-9a-fA-F]{2})([0-9a-fA-F]{2})([0-9a-fA-F]{2})$/.exec(expanded)
  if (!m) return null
  return [parseInt(m[1], 16), parseInt(m[2], 16), parseInt(m[3], 16)]
}

function rgbToHsl(r: number, g: number, b: number): [number, number, number] {
  const rn = r / 255, gn = g / 255, bn = b / 255
  const max = Math.max(rn, gn, bn)
  const min = Math.min(rn, gn, bn)
  const l = (max + min) / 2
  if (max === min) return [0, 0, l]
  const d = max - min
  const s = l > 0.5 ? d / (2 - max - min) : d / (max + min)
  let h: number
  if (max === rn)      h = ((gn - bn) / d + (gn < bn ? 6 : 0)) / 6
  else if (max === gn) h = ((bn - rn) / d + 2) / 6
  else                 h = ((rn - gn) / d + 4) / 6
  return [h * 360, s, l]
}

function hue2rgb(p: number, q: number, t: number): number {
  const tc = t < 0 ? t + 1 : t > 1 ? t - 1 : t
  if (tc < 1 / 6) return p + (q - p) * 6 * tc
  if (tc < 1 / 2) return q
  if (tc < 2 / 3) return p + (q - p) * (2 / 3 - tc) * 6
  return p
}

function hslToRgb(h: number, s: number, l: number): [number, number, number] {
  const hn = h / 360
  if (s === 0) {
    const v = Math.round(l * 255)
    return [v, v, v]
  }
  const q = l < 0.5 ? l * (1 + s) : l + s - l * s
  const p = 2 * l - q
  return [
    Math.round(hue2rgb(p, q, hn + 1 / 3) * 255),
    Math.round(hue2rgb(p, q, hn) * 255),
    Math.round(hue2rgb(p, q, hn - 1 / 3) * 255),
  ]
}

function toHex(r: number, g: number, b: number): string {
  return '#' + [r, g, b].map((v) => v.toString(16).padStart(2, '0')).join('')
}

/**
 * Retourne la couleur complémentaire (hue +180°, même saturation et luminosité)
 * d'un hex. Résultat toujours opaque (#RRGGBB sans canal alpha).
 *
 * Fallback : '#888888' si le hex est invalide.
 */
export function hexComplement(hex: string): string {
  const rgb = hexToRgb(hex)
  if (!rgb) return '#888888'
  const [h, s, l] = rgbToHsl(...rgb)
  const [r2, g2, b2] = hslToRgb((h + 180) % 360, s, l)
  return toHex(r2, g2, b2)
}
