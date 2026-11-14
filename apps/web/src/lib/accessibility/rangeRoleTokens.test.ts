/**
 * rangeRoleTokens.test.ts — garde-fou des trois jetons des rôles de portée (`range-role-front`,
 * `-mid`, `-far`), la rampe chaude -> neutre -> froide du ruban « Rôles de portée » (Escouade ›
 * Synergies), pour chaque palette et chaque thème.
 *
 * Les cases du ruban se touchent et se posent sur le fond de la carte (une case sans rôle reste
 * vide). Propriétés vérifiées :
 *   1. Contraste — chaque teinte tient 3:1 sur la carte, clair et sombre : une case se distingue
 *      du fond vide.
 *   2. Séparation — ΔE OKLab entre les trois teintes, deux à deux, en vision normale et sous
 *      protanopie / deutéranopie (et tritanopie pour les palettes daltoniennes).
 *   3. Ordre — le jeton « front » est plus chaud (teinte rouge-orange) que le « far » (bleu), le
 *      « mid » est le moins saturé des trois.
 */
import { describe, expect, it } from 'vitest'

import { deltaE, oklab, simulateCvd, type CvdKind } from './colorDistance'
import { cividisPalette } from './palettes/cividis'
import { defaultPalette } from './palettes/default'
import { okabePalette } from './palettes/okabe-ito'
import { tolBrightPalette } from './palettes/tol-bright'
import type { Palette, SemanticToken } from './semantic-tokens'
import { contrastRatio } from './wcagContrast'

const RAMP: readonly SemanticToken[] = ['range-role-front', 'range-role-mid', 'range-role-far']

// Surfaces de la carte (`--card`) des thèmes clair et sombre, comme impactRoleTokens.test.ts.
const SURFACES = { clair: '#FCFDFF', sombre: '#171717' } as const

const MIN_CONTRAST = 3
const MIN_DELTA_E = 10
const CVD_BASE: CvdKind[] = ['protanopie', 'deuteranopie']
const CVD_FULL: CvdKind[] = [...CVD_BASE, 'tritanopie']

const PALETTES: Record<string, { palette: Palette; cvd: CvdKind[] }> = {
  default: { palette: defaultPalette, cvd: CVD_BASE },
  'okabe-ito': { palette: okabePalette, cvd: CVD_FULL },
  cividis: { palette: cividisPalette, cvd: CVD_FULL },
  'tol-bright': { palette: tolBrightPalette, cvd: CVD_FULL },
}

const chroma = (hex: string): number => {
  const [, a, b] = oklab(hex)
  return Math.hypot(a, b)
}

describe.each(Object.entries(PALETTES))('palette « %s »', (_name, { palette, cvd }) => {
  it.each(Object.entries(SURFACES))('chaque teinte tient 3:1 sur la carte (%s)', (_theme, surface) => {
    for (const token of RAMP) {
      expect(contrastRatio(palette[token], surface), `${token} ${palette[token]}`).toBeGreaterThanOrEqual(MIN_CONTRAST)
    }
  })

  it('les trois teintes sont séparées deux à deux, en vision normale et sous daltonisme', () => {
    const pairs: [SemanticToken, SemanticToken][] = [
      [RAMP[0], RAMP[1]],
      [RAMP[1], RAMP[2]],
      [RAMP[0], RAMP[2]],
    ]
    for (const [a, b] of pairs) {
      expect(deltaE(palette[a], palette[b]), `${a} / ${b}`).toBeGreaterThanOrEqual(MIN_DELTA_E)
      for (const kind of cvd) {
        expect(deltaE(simulateCvd(palette[a], kind), simulateCvd(palette[b], kind)), `${a} / ${b} (${kind})`)
          .toBeGreaterThanOrEqual(MIN_DELTA_E)
      }
    }
  })

  it('front chaud, far froid, mid le plus neutre', () => {
    const [front, mid, far] = RAMP.map((t) => palette[t])
    // Chaud : composante b d'OKLab (jaune-bleu) positive ; froid : négative.
    expect(oklab(front)[2], 'front').toBeGreaterThan(0)
    expect(oklab(far)[2], 'far').toBeLessThan(0)
    expect(chroma(mid), 'mid').toBeLessThan(chroma(front))
    expect(chroma(mid), 'mid').toBeLessThan(chroma(far))
  })
})
