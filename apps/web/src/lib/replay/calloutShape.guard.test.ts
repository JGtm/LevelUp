/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : UNE SEULE RÈGLE « LA ZONE A UNE FORME » (règle <= 2 copies, CLAUDE.md n° 6).
 *
 * Cinq endroits la recopiaient et divergeaient (le calque des tueurs acceptait une zone faite de
 * parties seules, la probabilité de réapparition et la Tactique exigeaient le contour). La règle et
 * l'énumération des contours vivent dans `lib/replay/calloutShape.ts` ; comparer le nombre de
 * sommets d'un contour (`polygon.length`) à trois ailleurs sous `src/` rouvre la divergence.
 * Plafond 0 hors du helper ; les tests ne comptent pas.
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

const HELPER = 'lib/replay/calloutShape.ts'
const COMPARATORS = new Set([
  ts.SyntaxKind.GreaterThanEqualsToken,
  ts.SyntaxKind.LessThanToken,
  ts.SyntaxKind.GreaterThanToken,
  ts.SyntaxKind.LessThanEqualsToken,
])

function isLiteralBetween2And3(node: ts.Node): boolean {
  return ts.isNumericLiteral(node) && (node.text === '3' || node.text === '2')
}

/** `x.polygon.length` (ou `polygon.length`). */
function isPolygonLength(node: ts.Node): boolean {
  if (!ts.isPropertyAccessExpression(node) || node.name.text !== 'length') return false
  const target = node.expression
  return (
    (ts.isIdentifier(target) && target.text === 'polygon') ||
    (ts.isPropertyAccessExpression(target) && target.name.text === 'polygon')
  )
}

/** Une comparaison de `polygon.length` à 2 ou 3, dans un sens ou dans l'autre. */
function polygonLengthChecks(source: ScannedSource): ScanHit[] {
  if (source.rel === HELPER) return []
  const hits: ScanHit[] = []
  visitAll(source, (node) => {
    if (!ts.isBinaryExpression(node) || !COMPARATORS.has(node.operatorToken.kind)) return
    const sides: [ts.Node, ts.Node][] = [
      [node.left, node.right],
      [node.right, node.left],
    ]
    if (sides.some(([a, b]) => isPolygonLength(a) && isLiteralBetween2And3(b))) {
      hits.push({ file: source.rel, line: lineOf(source, node), detail: node.getText(source.sf) })
    }
  })
  return hits
}

describe('garde-rail : la forme d’une zone se décide par zoneHasShape', () => {
  it('aucune comparaison de polygon.length à 3 hors du helper (plafond 0)', () => {
    const hits = scanSources().flatMap(polygonLengthChecks)
    expect(hits, formatHits(hits, 'Décider par zoneHasShape / zoneRings (@/lib/replay/calloutShape).')).toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit les formes de la comparaison, pas une autre longueur', () => {
    const n = (code: string): number => polygonLengthChecks(parseSource('features/x/a.ts', code)).length
    expect(n('const a = z.polygon.length >= 3')).toBe(1)
    expect(n('const a = c.big || c.polygon.length < 3')).toBe(1)
    expect(n('const a = 3 <= polygon.length')).toBe(1)
    expect(n('const a = z.polygon.length > 2')).toBe(1)
    expect(n('const a = z.parts.length >= 3')).toBe(0)
    expect(n('const a = history.length >= 3')).toBe(0)
    expect(n('const a = z.polygon.length >= 4')).toBe(0)
  })
})
