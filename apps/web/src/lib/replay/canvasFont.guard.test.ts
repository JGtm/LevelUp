/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : UNE SEULE ÉCRITURE DE LA POLICE DU CANVAS (règle <= 2 copies, CLAUDE.md n° 6).
 *
 * La pile `ui-sans-serif, system-ui, sans-serif` était écrite sept fois dans les peintres du rejeu.
 * `CANVAS_FONT_FAMILY` (`lib/replay/canvasFont.ts`) est la seule à la porter : l'écrire dans un
 * littéral ailleurs sous `src/` rouvre la divergence. Plafond 0 hors du module ; les tests et les
 * commentaires ne comptent pas.
 */
import ts from 'typescript'
import { describe, expect, it } from 'vitest'
import {
  formatHits,
  lineOf,
  parseSource,
  scanSources,
  SCAN_TIMEOUT_MS,
  visitAll,
  type ScanHit,
  type ScannedSource,
} from '@/lib/archScan.testutil'

const HELPER = 'lib/replay/canvasFont.ts'
const PILE = 'ui-sans-serif'

/** Les littéraux (chaîne, gabarit) qui portent la pile système. */
function pileLiterals(source: ScannedSource): ScanHit[] {
  if (source.rel === HELPER) return []
  const hits: ScanHit[] = []
  visitAll(source, (node) => {
    const isText =
      ts.isStringLiteral(node) ||
      ts.isNoSubstitutionTemplateLiteral(node) ||
      ts.isTemplateHead(node) ||
      ts.isTemplateMiddle(node) ||
      ts.isTemplateTail(node)
    if (isText && node.text.includes(PILE)) hits.push({ file: source.rel, line: lineOf(source, node), detail: node.text })
  })
  return hits
}

describe('garde-rail : la police du canvas vient de CANVAS_FONT_FAMILY', () => {
  it('aucun littéral « ui-sans-serif » hors du module (plafond 0)', () => {
    const hits = scanSources().flatMap(pileLiterals)
    expect(hits, formatHits(hits, 'Écrire `${CANVAS_FONT_FAMILY}` (@/lib/replay/canvasFont).')).toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit la chaîne et le gabarit, pas un commentaire ni un autre nom', () => {
    const n = (code: string): number => pileLiterals(parseSource('features/x/a.ts', code)).length
    expect(n("const f = 'ui-sans-serif, system-ui'")).toBe(1)
    expect(n('const f = `600 ${px}px ui-sans-serif, system-ui`')).toBe(1)
    expect(n('const f = `${a} ui-sans-serif ${b}`')).toBe(1)
    expect(n('// ui-sans-serif')).toBe(0)
    expect(n("const f = 'Roboto, sans-serif'")).toBe(0)
  })
})
