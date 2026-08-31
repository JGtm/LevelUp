/**
 * Garde-rail fragClass (P1.3/P1.7 du PLAN_FRAG_DISTRIBUTION_V2, révisé « famille
 * dédiée frag-* » le 2026-08-29).
 *
 * Historique : le mapping empruntait des tokens d'autres gammes, contrôlés sur la
 * SEULE palette défaut — les palettes daltoniennes repliaient plusieurs classes sur
 * la même teinte (Okabe-Ito : lourde ≡ grenade ≡ équipement, épaule ≡ environnement ;
 * Cividis : lourde ≡ capacité spartan, épaule ≡ environnement — ΔE = 0). La famille
 * dédiée `frag-*` (semantic-tokens.ts) permet d'accorder chaque palette, et CE test
 * contrôle désormais LES TROIS :
 *
 * 1. Anti-collision : chaque classe a un TOKEN DISTINCT.
 * 2. Tokens valides : chaque token existe dans le contrat sémantique (ALL_TOKENS).
 * 3. Pin du mapping validé (famille dédiée, actée avec l'utilisateur le 2026-08-29).
 * 4. Distinction PAR PALETTE : hex tous DISTINCTS et pire ΔE OKLab ×100 all-pairs
 *    ≥ SEUIL — 8 sur défaut et Okabe-Ito ; 5 sur Cividis, seuil DOCUMENTÉ : rampe
 *    séquentielle par construction (l'identité par teinte y est impossible — cf.
 *    doctrine squad-player-* de cividis.ts), l'identité vient de la clarté et le
 *    DOUBLE ENCODAGE (labels + lignes de rappel + légende + position d'anneau, P1.2)
 *    porte le sens. La paire héritée spartan_ability/unattributed (6,89, exemptée le
 *    2026-08-29 matin) est RÉSOLUE par la famille dédiée — plus aucune exception.
 */
import { describe, it, expect } from 'vitest'
import { FRAG_CLASS_ORDER, FRAG_CLASS_TOKENS, fragClassToken } from './fragClass'
import { ALL_TOKENS, type Palette } from '../semantic-tokens'
import { defaultPalette } from '../palettes/default'
import { okabePalette } from '../palettes/okabe-ito'
import { cividisPalette } from '../palettes/cividis'
import { tolBrightPalette } from '../palettes/tol-bright'

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

describe('fragClass — garde-rail famille dédiée frag-* (tokens)', () => {
  it('mappe chaque classe sur un TOKEN distinct (anti-collision)', () => {
    const tokens = FRAG_CLASS_ORDER.map((c) => FRAG_CLASS_TOKENS[c])
    expect(new Set(tokens).size).toBe(FRAG_CLASS_ORDER.length)
  })

  it('chaque token est un SemanticToken valide du contrat', () => {
    for (const c of FRAG_CLASS_ORDER) {
      expect(ALL_TOKENS).toContain(FRAG_CLASS_TOKENS[c])
    }
  })

  it('pin le mapping validé (famille dédiée frag-*)', () => {
    expect(FRAG_CLASS_TOKENS).toEqual({
      shoulder: 'frag-shoulder',
      sidearm: 'frag-sidearm',
      heavy: 'frag-heavy',
      melee: 'frag-melee',
      grenade: 'frag-grenade',
      spartan_ability: 'frag-spartan-ability',
      vehicle: 'frag-vehicle',
      turret: 'frag-turret',
      equipment: 'frag-equipment',
      environmental: 'frag-environmental',
      unattributed: 'frag-unattributed',
    })
  })

  it('clé inconnue → token neutre (jamais une couleur de combat empruntée)', () => {
    expect(fragClassToken('inexistant')).toBe('divergent-neutral')
    expect(fragClassToken(null)).toBe('divergent-neutral')
  })

  // Seuils par palette — 8 = distinction normale-vision standard du dépôt ; 5 pour
  // Cividis : rampe séquentielle, identité par clarté + double encodage (cf. en-tête).
  const PALETTES: ReadonlyArray<{ name: string; palette: Palette; minDeltaE: number }> = [
    { name: 'défaut', palette: defaultPalette, minDeltaE: 8 },
    { name: 'okabe-ito', palette: okabePalette, minDeltaE: 8 },
    { name: 'cividis', palette: cividisPalette, minDeltaE: 5 },
    { name: 'tol-bright', palette: tolBrightPalette, minDeltaE: 8 },
  ]

  for (const { name, palette, minDeltaE } of PALETTES) {
    it(`palette ${name} : hex DISTINCTS + pire ΔE all-pairs ≥ ${minDeltaE} (résidu inclus)`, () => {
      const hexes = FRAG_CLASS_ORDER.map((c) => palette[FRAG_CLASS_TOKENS[c]])
      // Aucune classe sans définition dans la palette (le typage Palette le garantit
      // à la compilation ; ce contrôle attrape un hex vide ou malformé).
      for (const h of hexes) expect(h).toMatch(/^#[0-9a-fA-F]{6}$/)
      // Hex distincts (pas deux classes sur la même teinte).
      expect(new Set(hexes.map((h) => h.toLowerCase())).size).toBe(FRAG_CLASS_ORDER.length)
      // Distance perceptuelle toutes paires.
      let worst = Infinity
      let worstPair = ''
      for (let i = 0; i < hexes.length; i++) {
        for (let j = i + 1; j < hexes.length; j++) {
          const d = deltaE(hexes[i], hexes[j])
          if (d < worst) {
            worst = d
            worstPair = `${FRAG_CLASS_ORDER[i]}(${hexes[i]})|${FRAG_CLASS_ORDER[j]}(${hexes[j]})`
          }
        }
      }
      expect(worst, `palette ${name}, paire la plus proche : ${worstPair}`).toBeGreaterThanOrEqual(minDeltaE)
    })
  }
})
