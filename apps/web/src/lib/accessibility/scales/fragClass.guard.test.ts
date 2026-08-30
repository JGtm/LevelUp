/**
 * Garde-rail fragClass (P1.3/P1.7 du PLAN_FRAG_DISTRIBUTION_V2, révisé « gamme
 * Antagonistes »).
 *
 * L'ancienne approche « hex Okabe FIXES » est REMPLACÉE par un mapping classe →
 * TOKEN de la gamme « Antagonistes » (réactive à la palette, comme
 * MatchAntagonistChart). Le garde-rail vérifie donc désormais :
 *
 * 1. Anti-collision : chaque classe a un TOKEN DISTINCT (aucune classe n'en partage
 *    un) — corrige le doublon mêlée=grenade de l'ancien donut.
 * 2. Tokens valides : chaque token existe dans le contrat sémantique (ALL_TOKENS).
 * 3. Pin du mapping validé (la gamme Antagonistes actée avec l'utilisateur).
 * 4. Distinction : résolus sur la palette DÉFAUT, TOUTES les classes — le résidu
 *    `unattributed` inclus depuis le 2026-08-29, il en était exclu et masquait une
 *    collision — donnent autant d'hex DISTINCTS et une distance perceptuelle
 *    normale-vision suffisante (min all-pairs ΔE OKLab ×100 ≥ 8), à l'exception datée
 *    ci-dessous. La robustesse daltonisme est
 *    portée par l'ENCODAGE SECONDAIRE (labels + lignes de rappel + légende +
 *    position d'anneau), jamais par la seule teinte — cf. double encodage P1.2.
 */
import { describe, it, expect } from 'vitest'
import { FRAG_CLASS_ORDER, FRAG_CLASS_TOKENS, fragClassToken } from './fragClass'
import { ALL_TOKENS } from '../semantic-tokens'
import { defaultPalette } from '../palettes/default'

// ── ΔE normal-vision (OKLab, extrait du validateur dataviz) ─────────────────────
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
function deltaE(h1: string, h2: string): number {
  const a = oklab(lin(h1))
  const b = oklab(lin(h2))
  return 100 * Math.hypot(a[0] - b[0], a[1] - b[1], a[2] - b[2])
}

describe('fragClass — garde-rail gamme Antagonistes (tokens)', () => {
  it('mappe chaque classe sur un TOKEN distinct (anti-collision)', () => {
    const tokens = FRAG_CLASS_ORDER.map((c) => FRAG_CLASS_TOKENS[c])
    expect(new Set(tokens).size).toBe(FRAG_CLASS_ORDER.length)
  })

  it('chaque token est un SemanticToken valide du contrat', () => {
    for (const c of FRAG_CLASS_ORDER) {
      expect(ALL_TOKENS).toContain(FRAG_CLASS_TOKENS[c])
    }
  })

  it('pin le mapping validé (gamme Antagonistes)', () => {
    expect(FRAG_CLASS_TOKENS).toEqual({
      shoulder: 'perf-tier-2',
      sidearm: 'chart-series-6',
      heavy: 'narrative-humiliation',
      melee: 'chart-series-8',
      grenade: 'chart-series-7',
      spartan_ability: 'compare-a',
      vehicle: 'chart-series-5',
      turret: 'narrative-debacle',
      equipment: 'extreme',
      environmental: 'narrative-remontada',
      unattributed: 'divergent-neutral',
    })
  })

  it('clé inconnue → token neutre (jamais une couleur de combat empruntée)', () => {
    expect(fragClassToken('inexistant')).toBe('divergent-neutral')
    expect(fragClassToken(null)).toBe('divergent-neutral')
  })

  // PAIRE HÉRITÉE, constatée le 2026-08-29 en incluant `unattributed` au contrôle ΔE
  // (il en était exclu jusque-là, ce qui masquait la collision) : spartan_ability
  // (compare-a #818CF8) et unattributed (divergent-neutral #60A5FA) valent 6,89 — deux
  // bleus clairs. Exception CIBLÉE à cette seule paire, à re-trancher si refonte palette :
  // les deux tokens servent ailleurs (page Compare, Résistance défensive), les rebasculer
  // dépasse le lot. Le seuil global reste 8 et aucune autre paire n'est exemptée.
  const LEGACY_UNDER_THRESHOLD = new Set(['spartan_ability|unattributed'])

  it('classes (résidu INCLUS) : hex DISTINCTS + distinction normale-vision (ΔE ≥ 8) sur la palette défaut', () => {
    const hexes = FRAG_CLASS_ORDER.map((c) => defaultPalette[FRAG_CLASS_TOKENS[c]])
    // Hex distincts (pas deux classes sur la même teinte de palette).
    expect(new Set(hexes.map((h) => h.toLowerCase())).size).toBe(FRAG_CLASS_ORDER.length)
    // Distance perceptuelle normale-vision, toutes paires sauf l'exception héritée.
    let worst = Infinity
    let worstPair = ''
    for (let i = 0; i < hexes.length; i++) {
      for (let j = i + 1; j < hexes.length; j++) {
        const pair = [FRAG_CLASS_ORDER[i], FRAG_CLASS_ORDER[j]].sort().join('|')
        if (LEGACY_UNDER_THRESHOLD.has(pair)) continue
        const d = deltaE(hexes[i], hexes[j])
        if (d < worst) {
          worst = d
          worstPair = pair
        }
      }
    }
    expect(worst, `paire la plus proche : ${worstPair}`).toBeGreaterThanOrEqual(8)
  })

  it('l\'exception héritée est toujours JUSTIFIÉE (sinon la retirer)', () => {
    // Ratchet anti « compatibility guard forever » : le jour où la palette rend cette
    // paire conforme, ce test casse et l'exception doit DISPARAÎTRE du fichier.
    for (const pair of LEGACY_UNDER_THRESHOLD) {
      const [a, b] = pair.split('|') as [(typeof FRAG_CLASS_ORDER)[number], (typeof FRAG_CLASS_ORDER)[number]]
      const d = deltaE(defaultPalette[FRAG_CLASS_TOKENS[a]], defaultPalette[FRAG_CLASS_TOKENS[b]])
      expect(d, `${pair} n'est plus sous le seuil : retirer l'exception`).toBeLessThan(8)
    }
  })
})
