/**
 * combatStatTokens.test.ts — garde-rail de la famille des stats de combat.
 *
 * `stat-kills`, `stat-deaths`, `stat-assists`, `assist-received`, `assist-given` disent
 * « c'est un frag / une mort / une assistance / ce sens d'assistance » dans toute l'app.
 * Deux d'entre eux trop proches, et un frag se lit comme une assistance, ou « il te
 * sert » comme une mort. Valeurs validées avec l'utilisateur le 2026-09-17
 * (`.ai/V7.5/chantiers/PLAN_COULEURS_STATS_COMBAT_2026-09-17.md`) ; mêmes seuils que
 * `squadPlayerTokens.test.ts` :
 *   1. Lisibilité — contraste WCAG ≥ 3:1 (objet graphique) sur les surfaces produit.
 *   2. Séparation — ΔE OKLab × 100 ≥ 15 entre toutes les paires, et ≥ 8 sous simulation
 *      protanopie / deutéranopie quand la palette l'exige.
 */
import { describe, it, expect } from 'vitest'
import type { Palette, SemanticToken } from './semantic-tokens'
import { defaultPalette } from './palettes/default'
import { okabePalette } from './palettes/okabe-ito'
import { cividisPalette } from './palettes/cividis'
import { tolBrightPalette } from './palettes/tol-bright'
import { contrastRatio } from './wcagContrast'
import { deltaE, simulateCvd, type CvdKind } from './colorDistance'

const COMBAT_STAT_TOKENS: SemanticToken[] = [
  'stat-kills',
  'stat-deaths',
  'stat-assists',
  'assist-received',
  'assist-given',
]

// Surfaces produit : `--card` clair et sombre (cf. squadPlayerTokens.test.ts).
const SURFACE_LIGHT = '#FCFDFF'
const SURFACE_DARK = '#171717'
const MIN_CONTRAST = 3
const MIN_DELTA_E_NORMAL = 15
const MIN_DELTA_E_CVD = 8

interface PaletteCase {
  name: string
  palette: Palette
  /** 'both' : contraste sur les DEUX surfaces ; 'any' : au moins une. */
  contrast: 'both' | 'any'
  cvd: boolean
}

const CASES: PaletteCase[] = [
  { name: 'default', palette: defaultPalette, contrast: 'both', cvd: true },
  // Exemption contraste 'any' — mesurée le 2026-09-17 par recherche exhaustive sur les
  // teintes Okabe-Ito et leurs versions assombries : aucun jeu ne tient à la fois 3:1 sur
  // fond clair pour le bleu ciel et l'orange ET la séparation sous daltonisme. La
  // palette daltonisme garde la séparation, qui est sa raison d'être.
  { name: 'okabe-ito', palette: okabePalette, contrast: 'any', cvd: true },
  // Cividis tient le contraste sur les DEUX surfaces depuis le 2026-09-17 : ses deux sens
  // d'assistance prenaient les extrémités de la rampe (1.65:1 sur fond clair, 1.39:1 sur
  // fond sombre) alors qu'une palette sert les deux thèmes. L'exemption 'any' masquait la
  // faille — cf. `palettes/cividis.ts`. Reste `cvd: false` : rampe séquentielle, schéma
  // validé daltonisme par son auteur.
  { name: 'cividis', palette: cividisPalette, contrast: 'both', cvd: false },
  { name: 'tol-bright', palette: tolBrightPalette, contrast: 'both', cvd: false },
]

describe.each(CASES)('famille stats de combat — palette "$name"', ({ palette, contrast, cvd }) => {
  const hexes = COMBAT_STAT_TOKENS.map((t) => palette[t])

  it.each(COMBAT_STAT_TOKENS)('%s est lisible sur les surfaces produit', (token) => {
    const hex = palette[token]
    const light = contrastRatio(hex, SURFACE_LIGHT)
    const dark = contrastRatio(hex, SURFACE_DARK)
    const detail = `${hex} : clair ${light.toFixed(2)}:1, sombre ${dark.toFixed(2)}:1`
    if (contrast === 'both') {
      expect(light, `${token} illisible sur la surface claire — ${detail}`).toBeGreaterThanOrEqual(MIN_CONTRAST)
      expect(dark, `${token} illisible sur la surface sombre — ${detail}`).toBeGreaterThanOrEqual(MIN_CONTRAST)
    } else {
      expect(Math.max(light, dark), `${token} illisible sur les deux surfaces — ${detail}`).toBeGreaterThanOrEqual(
        MIN_CONTRAST,
      )
    }
  })

  it('toutes les paires restent distinctes en vision normale', () => {
    for (let i = 0; i < hexes.length; i++) {
      for (let j = i + 1; j < hexes.length; j++) {
        const d = deltaE(hexes[i], hexes[j])
        expect(
          d,
          `${COMBAT_STAT_TOKENS[i]} (${hexes[i]}) et ${COMBAT_STAT_TOKENS[j]} (${hexes[j]}) trop proches : ΔE ${d.toFixed(1)}`,
        ).toBeGreaterThanOrEqual(MIN_DELTA_E_NORMAL)
      }
    }
  })

  const cvdKinds: CvdKind[] = cvd ? ['protanopie', 'deuteranopie'] : []
  it.each(cvdKinds)('toutes les paires restent distinctes sous %s', (kind) => {
    const simulated = hexes.map((h) => simulateCvd(h, kind))
    for (let i = 0; i < simulated.length; i++) {
      for (let j = i + 1; j < simulated.length; j++) {
        const d = deltaE(simulated[i], simulated[j])
        expect(
          d,
          `${COMBAT_STAT_TOKENS[i]} (${hexes[i]}) et ${COMBAT_STAT_TOKENS[j]} (${hexes[j]}) confondus en ${kind} : ΔE ${d.toFixed(1)}`,
        ).toBeGreaterThanOrEqual(MIN_DELTA_E_CVD)
      }
    }
  })
})
