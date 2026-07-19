/**
 * Garde-rail fragClass (P1.3 + P1.7 du PLAN_FRAG_DISTRIBUTION_V2).
 *
 * 1. Anti-collision : chaque classe du sunburst a un hex DISTINCT (corrige le
 *    doublon mêlée=grenade de l'ancien donut). Une régression ici re-fusionnerait
 *    deux classes en une couleur.
 * 2. Indépendance de la palette active : les couleurs de frags viennent d'un jeu
 *    de hex FIXES (fragClassColors.ts, teintes Okabe-Ito), PAS de la palette active
 *    — sous la palette DÉFAUT, chart-series-1..5 sont une rampe indigo (ΔE ~5.4,
 *    SOUS le plancher CVD). fragClassColor renvoie donc un hex littéral, jamais un
 *    token résolu au runtime : la distinction est garantie quelle que soit la palette.
 * 3. CVD : les 6 classes de combat restent séparées sous simulation protanope/
 *    deutéranope. Cible : min all-pairs ΔE (OKLab ×100) >= 8 (cible du validateur
 *    dataviz ; cible plan >=12 non atteignable sans casser le floor normal-vision —
 *    plafond structurel des 6 teintes Okabe). Mesuré : 11.0 (deutan), 15.6 (normal).
 *
 * Math CVD = OKLab + transforme Machado-Oliveira-Fernandes (2009) severité 1.0,
 * portée du validateur dataviz (scripts/validate_palette.js) — self-contained.
 */
import { describe, it, expect } from 'vitest'
import { fragClassColor, FRAG_CLASS_ORDER } from './fragClass'
import { FRAG_CLASS_HEX, FRAG_CLASS_NEUTRAL_HEX } from './fragClassColors'

// ── conversions couleur (extrait du validateur dataviz) ─────────────────────────
const s2lin = (c: number) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4)
function lin(hex: string): [number, number, number] {
  const h = hex.replace(/^#/, '')
  return [0, 2, 4].map((i) => s2lin(parseInt(h.slice(i, i + 2), 16) / 255)) as [number, number, number]
}
function oklab([r, g, b]: [number, number, number]): [number, number, number] {
  const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b)
  const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b)
  const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b)
  return [
    0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s,
    1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s,
    0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s,
  ]
}
const MACHADO: Record<string, number[][]> = {
  protan: [
    [0.152286, 1.052583, -0.204868],
    [0.114503, 0.786281, 0.099216],
    [-0.003882, -0.048116, 1.051998],
  ],
  deutan: [
    [0.367322, 0.860646, -0.227968],
    [0.280085, 0.672501, 0.047413],
    [-0.01182, 0.04294, 0.968881],
  ],
}
function simulate(hex: string, kind: string): [number, number, number] {
  const [r, g, b] = lin(hex)
  const M = MACHADO[kind]
  const clamp = (c: number) => Math.max(0, Math.min(1, c))
  return [
    clamp(M[0][0] * r + M[0][1] * g + M[0][2] * b),
    clamp(M[1][0] * r + M[1][1] * g + M[1][2] * b),
    clamp(M[2][0] * r + M[2][1] * g + M[2][2] * b),
  ]
}
function deltaE(h1: string, h2: string, kind?: string): number {
  const a = oklab(kind ? simulate(h1, kind) : lin(h1))
  const b = oklab(kind ? simulate(h2, kind) : lin(h2))
  return 100 * Math.hypot(a[0] - b[0], a[1] - b[1], a[2] - b[2])
}

const HEX_LITERAL = /^#[0-9a-f]{6}$/i

describe('fragClass — garde-rail couleur des classes de frags', () => {
  it('mappe chaque classe sur un hex DISTINCT (anti-collision)', () => {
    const hexes = FRAG_CLASS_ORDER.map((c) => fragClassColor(c))
    expect(new Set(hexes.map((h) => h.toLowerCase())).size).toBe(FRAG_CLASS_ORDER.length)
  })

  it('pin le jeu de hex FIXES validé CVD (Okabe-Ito, indépendant de la palette)', () => {
    expect(FRAG_CLASS_HEX).toEqual({
      shoulder: '#0072B2',
      sidearm: '#E69F00',
      heavy: '#56B4E9',
      melee: '#D55E00',
      grenade: '#009E73',
      spartan_ability: '#F0E442',
      unattributed: '#888888',
    })
  })

  it('la distinction NE dépend PAS de la palette active : hex littéral, jamais un token résolu', () => {
    // fragClassColor renvoie un hex direct (pas de resolveToken/CSS var) → même
    // valeur quelle que soit la palette appliquée au DOM (défaut/Okabe-Ito/…).
    for (const c of FRAG_CLASS_ORDER) {
      expect(fragClassColor(c)).toMatch(HEX_LITERAL)
      expect(fragClassColor(c)).toBe(FRAG_CLASS_HEX[c])
    }
  })

  it('clé inconnue → neutre (pas de couleur de combat empruntée)', () => {
    expect(fragClassColor('inexistant')).toBe(FRAG_CLASS_NEUTRAL_HEX)
    expect(fragClassColor(null)).toBe(FRAG_CLASS_NEUTRAL_HEX)
  })

  it('6 classes combat CVD-safe : min all-pairs ΔE >= 8 (protan+deutan)', () => {
    const combat = FRAG_CLASS_ORDER.filter((c) => c !== 'unattributed')
    const hexes = combat.map((c) => fragClassColor(c))
    let worst = Infinity
    for (let i = 0; i < hexes.length; i++) {
      for (let j = i + 1; j < hexes.length; j++) {
        for (const kind of ['protan', 'deutan']) {
          worst = Math.min(worst, deltaE(hexes[i], hexes[j], kind))
        }
      }
    }
    // Mesure de référence : 11.0 (deutan). Seuil 8 = cible du validateur dataviz.
    expect(worst).toBeGreaterThanOrEqual(8)
  })
})
