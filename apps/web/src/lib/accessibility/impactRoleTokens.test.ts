/**
 * impactRoleTokens.test.ts — garde-fou des deux rampes des rôles d'impact (`impact-gain-1..4`,
 * `impact-loss-1..3`, cf. `palettes/_impactRoleColors.ts`), pour chaque palette et chaque thème.
 *
 * Les segments d'une barre empilée se touchent : deux voisins d'empilement qui se rapprochent
 * fondent deux rôles en un. Propriétés vérifiées :
 *   1. Ordre — le pas 1 (plus fort barème, contre l'axe) est le plus contrasté sur la carte,
 *      le contraste décroît pas à pas vers le bout de la barre, et le dernier pas tient 2:1.
 *   2. Séparation — ΔE OKLab entre voisins d'empilement ≥ 15 en vision normale et sous
 *      protanopie / deutéranopie ; sous tritanopie aussi pour les palettes daltoniennes.
 *   3. Teinte — les gains suivent la teinte de `divergent-pos` de la palette, les pertes celle de
 *      `divergent-neg` ; une palette daltonienne ne reprend aucune valeur de la palette par défaut.
 *   4. Thème — `paletteForTheme` ne change que ces sept jetons, seulement en sombre, et y pose la
 *      jumelle sombre de la rampe claire que porte la palette.
 */
import { describe, expect, it } from 'vitest'

import { deltaE, oklab, simulateCvd, type CvdKind } from './colorDistance'
import { paletteForTheme, type PaletteTheme } from './paletteForTheme'
import {
  IMPACT_BLUE_VERMILLION,
  IMPACT_GREEN_RED,
  IMPACT_TOL_BLUE_RED,
  type ImpactRoleRamps,
  type ImpactRoleToken,
} from './palettes/_impactRoleColors'
import { cividisPalette } from './palettes/cividis'
import { defaultPalette } from './palettes/default'
import { okabePalette } from './palettes/okabe-ito'
import { tolBrightPalette } from './palettes/tol-bright'
import { ALL_TOKENS, type Palette, type SemanticToken } from './semantic-tokens'
import { contrastRatio } from './wcagContrast'

const GAINS: ImpactRoleToken[] = ['impact-gain-1', 'impact-gain-2', 'impact-gain-3', 'impact-gain-4']
const LOSSES: ImpactRoleToken[] = ['impact-loss-1', 'impact-loss-2', 'impact-loss-3']
const IMPACT: ImpactRoleToken[] = [...GAINS, ...LOSSES]

// Surfaces de la carte (`--card`) des thèmes clair et sombre, comme squadPlayerTokens.test.ts.
const SURFACE = { light: '#FCFDFF', dark: '#171717' } as const
const THEMES: PaletteTheme[] = ['light', 'dark']

/** Écart minimal entre voisins d'empilement (vision normale et simulations). */
const MIN_DELTA_E = 15
/** Contraste minimal de la nuance la plus pâle sur la carte. */
const MIN_PALE_CONTRAST = 2
/** Écart de teinte OKLCH maximal entre un pas et la teinte positif / négatif de sa palette. */
const MAX_HUE_DRIFT_DEG = 12
const CVD_BASE: CvdKind[] = ['protanopie', 'deuteranopie']
const CVD_FULL: CvdKind[] = [...CVD_BASE, 'tritanopie']

interface Case {
  palette: Palette
  ramps: ImpactRoleRamps
  cvd: CvdKind[]
}

const CASES: Record<string, Case> = {
  default: { palette: defaultPalette, ramps: IMPACT_GREEN_RED, cvd: CVD_BASE },
  'okabe-ito': { palette: okabePalette, ramps: IMPACT_BLUE_VERMILLION, cvd: CVD_FULL },
  cividis: { palette: cividisPalette, ramps: IMPACT_BLUE_VERMILLION, cvd: CVD_FULL },
  'tol-bright': { palette: tolBrightPalette, ramps: IMPACT_TOL_BLUE_RED, cvd: CVD_FULL },
}
const isImpact = (t: SemanticToken): t is ImpactRoleToken => (IMPACT as SemanticToken[]).includes(t)
const CVD_PALETTES = Object.entries(CASES).filter(([n]) => n !== 'default')

function hueDeg(hex: string): number {
  const [, a, b] = oklab(hex)
  return ((Math.atan2(b, a) * 180) / Math.PI + 360) % 360
}

function hueGap(a: number, b: number): number {
  const d = Math.abs(a - b) % 360
  return d > 180 ? 360 - d : d
}

const RAMPS = [['gains', GAINS, 'divergent-pos'], ['pertes', LOSSES, 'divergent-neg']] as const

describe.each(Object.entries(CASES))('palette « %s »', (_name, { palette: base, cvd }) => {
  describe.each(THEMES)('thème %s', (theme) => {
    const palette = paletteForTheme(base, theme)

    it.each(RAMPS)('%s : contraste décroissant du pas 1 au dernier, dernier ≥ 2:1', (_n, ramp) => {
      const contrasts = ramp.map((t) => contrastRatio(palette[t], SURFACE[theme]))
      for (let i = 1; i < contrasts.length; i++) {
        expect(contrasts[i], `${ramp[i]} (${contrasts[i].toFixed(2)}) vs ${ramp[i - 1]} (${contrasts[i - 1].toFixed(2)})`)
          .toBeLessThan(contrasts[i - 1])
      }
      expect(contrasts[contrasts.length - 1], ramp[ramp.length - 1]).toBeGreaterThanOrEqual(MIN_PALE_CONTRAST)
    })

    it.each(RAMPS)('%s : voisins séparés (normal et daltonisme)', (_n, ramp) => {
      for (let i = 1; i < ramp.length; i++) {
        const a = palette[ramp[i - 1]]
        const b = palette[ramp[i]]
        expect(deltaE(a, b), `${ramp[i - 1]} / ${ramp[i]}`).toBeGreaterThanOrEqual(MIN_DELTA_E)
        for (const kind of cvd) {
          expect(deltaE(simulateCvd(a, kind), simulateCvd(b, kind)), `${ramp[i - 1]} / ${ramp[i]} (${kind})`)
            .toBeGreaterThanOrEqual(MIN_DELTA_E)
        }
      }
    })

    it.each(RAMPS)('%s : teinte du jeton %s de la palette', (_n, ramp, anchor) => {
      const target = hueDeg(base[anchor])
      for (const t of ramp) {
        expect(hueGap(hueDeg(palette[t]), target), `${t} ${palette[t]} vs ${anchor} ${base[anchor]}`)
          .toBeLessThanOrEqual(MAX_HUE_DRIFT_DEG)
      }
    })
  })
})

describe('valeurs propres à chaque palette', () => {
  it.each(CVD_PALETTES)('palette « %s » : aucune valeur reprise de la palette par défaut', (_n, { palette }) => {
    for (const theme of THEMES) {
      const own = paletteForTheme(palette, theme)
      const def = paletteForTheme(defaultPalette, theme)
      for (const t of IMPACT) expect(own[t], `${t} (${theme})`).not.toBe(def[t])
    }
  })
})

describe('paletteForTheme', () => {
  it.each(Object.entries(CASES))('palette « %s » : rampe claire portée, jumelle sombre posée, rien d’autre ne bouge', (_n, { palette, ramps }) => {
    for (const t of IMPACT) expect(palette[t], t).toBe(ramps.light[t])
    expect(paletteForTheme(palette, 'light')).toBe(palette)
    const dark = paletteForTheme(palette, 'dark')
    for (const t of ALL_TOKENS) {
      if (isImpact(t)) expect(dark[t], t).toBe(ramps.dark[t])
      else expect(dark[t], t).toBe(palette[t])
    }
  })

  it('rampe claire inconnue : palette rendue telle quelle en sombre', () => {
    const odd: Palette = { ...defaultPalette, 'impact-gain-1': '#123456' }
    expect(paletteForTheme(odd, 'dark')).toBe(odd)
  })
})
