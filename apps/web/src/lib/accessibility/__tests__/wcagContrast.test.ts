/**
 * wcagContrast.test.ts — Vérifie WCAG 2.0 AA (≥ 4.5:1) sur toutes les
 * paires `narrative-{bg}` / `narrative-{text}` de chaque palette.
 *
 * S'inspire de afonsolopez/colorblind (port Go) — réimplémenté en TS
 * pour rester sans dépendance.
 *
 * Si un test échoue, le plan d'action est dans
 * .ai/PLAN_ACCESSIBILITY_PALETTES_V2.md § C.5 :
 *   1. ajuster la couleur de texte (noir ↔ blanc) ;
 *   2. ajuster la couleur de fond ;
 *   3. en dernier recours : `.skip` + commentaire `// FIXME: WCAG-fail-by-design`.
 */
import { describe, it, expect } from 'vitest'
import type { SemanticToken } from '../semantic-tokens'
import { defaultPalette } from '../palettes/default'
import { okabePalette } from '../palettes/okabe-ito'
import { cividisPalette } from '../palettes/cividis'
import { tolBrightPalette } from '../palettes/tol-bright'
import { contrastRatio } from '../wcagContrast'

const PALETTES = {
  default: defaultPalette,
  'okabe-ito': okabePalette,
  cividis: cividisPalette,
  'tol-bright': tolBrightPalette,
}

const NARRATIVE_PAIRS: Array<[SemanticToken, SemanticToken]> = [
  ['narrative-dominant', 'narrative-dominant-text'],
  ['narrative-humiliation', 'narrative-humiliation-text'],
  ['narrative-remontada', 'narrative-remontada-text'],
  ['narrative-debacle', 'narrative-debacle-text'],
  ['narrative-contre-remontada', 'narrative-contre-remontada-text'],
]

const WCAG_AA_NORMAL = 4.5

describe.each(Object.entries(PALETTES))('palette "%s" — WCAG AA narratif', (_name, palette) => {
  it.each(NARRATIVE_PAIRS)(
    'paire %s / %s ≥ 4.5:1',
    (bg, text) => {
      const ratio = contrastRatio(palette[bg], palette[text])
      expect(
        ratio,
        `contraste insuffisant pour ${bg}/${text}: ${ratio.toFixed(2)} < ${WCAG_AA_NORMAL}`,
      ).toBeGreaterThanOrEqual(WCAG_AA_NORMAL)
    },
  )
})
