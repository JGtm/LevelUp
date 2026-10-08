/**
 * impactRoleTokens.test.ts — garde-fou des deux rampes des rôles d'impact (`impact-gain-1..4`,
 * `impact-loss-1..3`, cf. `palettes/_impactRoleColors.ts`).
 *
 * Les segments d'une barre empilée se touchent : deux voisins d'empilement qui se rapprochent
 * fondent deux rôles en un. Propriétés vérifiées, dans chaque thème :
 *   1. Ordre — le pas 1 (plus fort barème, contre l'axe) est le plus contrasté sur la carte,
 *      et le contraste décroît pas à pas vers le bout de la barre.
 *   2. Séparation — ΔE OKLab entre voisins d'empilement ≥ 15 en vision normale, ≥ 14 sous
 *      protanopie / deutéranopie (la simulation de `colorDistance.ts` place deux voisins de la
 *      rampe sombre validée juste sous 15).
 *   3. Thème — `paletteForTheme` ne change que ces sept jetons, et seulement en sombre ; les
 *      quatre palettes portent la même rampe claire.
 */
import { describe, expect, it } from 'vitest'

import { deltaE, simulateCvd, type CvdKind } from './colorDistance'
import { paletteForTheme } from './paletteForTheme'
import { cividisPalette } from './palettes/cividis'
import { defaultPalette } from './palettes/default'
import { okabePalette } from './palettes/okabe-ito'
import { tolBrightPalette } from './palettes/tol-bright'
import { ALL_TOKENS, type Palette, type SemanticToken } from './semantic-tokens'
import { contrastRatio } from './wcagContrast'

const GAINS: SemanticToken[] = ['impact-gain-1', 'impact-gain-2', 'impact-gain-3', 'impact-gain-4']
const LOSSES: SemanticToken[] = ['impact-loss-1', 'impact-loss-2', 'impact-loss-3']
const IMPACT = new Set<SemanticToken>([...GAINS, ...LOSSES])

// Surfaces de la carte (`--card`) des thèmes clair et sombre, comme squadPlayerTokens.test.ts.
const SURFACE = { light: '#FCFDFF', dark: '#171717' } as const

/** Écart minimal entre voisins d'empilement, en vision normale puis sous daltonisme. */
const MIN_DELTA_E = 15
const MIN_DELTA_E_CVD = 14
const CVD: CvdKind[] = ['protanopie', 'deuteranopie']

const PALETTES: Record<string, Palette> = {
  default: defaultPalette,
  'okabe-ito': okabePalette,
  cividis: cividisPalette,
  'tol-bright': tolBrightPalette,
}

describe.each(['light', 'dark'] as const)('rampes des rôles d’impact, thème %s', (theme) => {
  const palette = paletteForTheme(defaultPalette, theme)

  it.each([['gains', GAINS], ['pertes', LOSSES]] as const)('%s : contraste décroissant du pas 1 au dernier', (_n, ramp) => {
    const contrasts = ramp.map((t) => contrastRatio(palette[t], SURFACE[theme]))
    for (let i = 1; i < contrasts.length; i++) {
      expect(contrasts[i], `${ramp[i]} (${contrasts[i].toFixed(2)}) vs ${ramp[i - 1]} (${contrasts[i - 1].toFixed(2)})`)
        .toBeLessThan(contrasts[i - 1])
    }
  })

  it.each([['gains', GAINS], ['pertes', LOSSES]] as const)('%s : voisins séparés (normal et daltonisme)', (_n, ramp) => {
    for (let i = 1; i < ramp.length; i++) {
      const a = palette[ramp[i - 1]]
      const b = palette[ramp[i]]
      expect(deltaE(a, b), `${ramp[i - 1]} / ${ramp[i]}`).toBeGreaterThanOrEqual(MIN_DELTA_E)
      for (const kind of CVD) {
        expect(deltaE(simulateCvd(a, kind), simulateCvd(b, kind)), `${ramp[i - 1]} / ${ramp[i]} (${kind})`)
          .toBeGreaterThanOrEqual(MIN_DELTA_E_CVD)
      }
    }
  })
})

describe('paletteForTheme', () => {
  it.each(Object.entries(PALETTES))('palette « %s » : même rampe claire, seuls les rôles d’impact changent en sombre', (_n, p) => {
    for (const t of IMPACT) expect(p[t]).toBe(defaultPalette[t])
    expect(paletteForTheme(p, 'light')).toBe(p)
    const dark = paletteForTheme(p, 'dark')
    for (const t of ALL_TOKENS) {
      if (IMPACT.has(t)) expect(dark[t], t).not.toBe(p[t])
      else expect(dark[t], t).toBe(p[t])
    }
  })
})
