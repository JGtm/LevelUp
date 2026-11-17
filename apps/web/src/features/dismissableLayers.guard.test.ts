/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : la fermeture d'une couche (Échap, clic dehors) s'écrit UNE fois, dans
 * `lib/hooks/useDismissable`, jamais dans un composant.
 *
 * Motifs interdits dans un `.tsx` de `features/` : un écouteur `pointerdown` ou `mousedown` posé par
 * `addEventListener`, et un test `key === 'Escape'`. Les menus du rejeu les ont rendus par le hook ;
 * leur plafond est 0. Les autres features n'y sont pas migrées : leur compte (en fichiers) est gelé à
 * sa valeur mesurée et ne fait que descendre — il échoue aussi quand il baisse sans que
 * `PLAFOND_HORS_REJEU` le suive (cliquet).
 */
import ts from 'typescript'
import { describe, expect, it } from 'vitest'
import {
  lineOf,
  parseSource,
  scanSources,
  SCAN_TIMEOUT_MS,
  visitAll,
  type ScanHit,
  type ScannedSource,
} from '@/lib/archScan.testutil'

const REPLAY = 'features/match-replay/'

/** Le compte mesuré (fichiers `.tsx` de `features/` hors rejeu) à la clôture de E.1. */
const PLAFOND_HORS_REJEU = 17

const GESTES_DEHORS = new Set(['pointerdown', 'mousedown'])

function isLiteral(node: ts.Node, value: string): boolean {
  return ts.isStringLiteralLike(node) && node.text === value
}

/** Un `addEventListener('pointerdown' | 'mousedown', ...)` ou un `x === 'Escape'` / `'Escape' === x`. */
function dismissPatterns(source: ScannedSource): ScanHit[] {
  const hits: ScanHit[] = []
  visitAll(source, (node) => {
    if (
      ts.isCallExpression(node) &&
      ts.isPropertyAccessExpression(node.expression) &&
      node.expression.name.text === 'addEventListener' &&
      node.arguments[0] !== undefined &&
      ts.isStringLiteralLike(node.arguments[0]) &&
      GESTES_DEHORS.has(node.arguments[0].text)
    ) {
      hits.push({ file: source.rel, line: lineOf(source, node), detail: `addEventListener('${node.arguments[0].text}')` })
    }
    if (
      ts.isBinaryExpression(node) &&
      (node.operatorToken.kind === ts.SyntaxKind.EqualsEqualsEqualsToken ||
        node.operatorToken.kind === ts.SyntaxKind.EqualsEqualsToken) &&
      (isLiteral(node.left, 'Escape') || isLiteral(node.right, 'Escape'))
    ) {
      hits.push({ file: source.rel, line: lineOf(source, node), detail: "=== 'Escape'" })
    }
  })
  return hits
}

const tsxDeFeatures = () => scanSources({ under: 'features', tsxOnly: true })

describe('garde-rail : la fermeture d’une couche passe par useDismissable', () => {
  it('plafond 0 : aucun .tsx de features/match-replay ne l’écrit', () => {
    const hits = tsxDeFeatures()
      .filter((s) => s.rel.startsWith(REPLAY))
      .flatMap(dismissPatterns)
    expect(
      hits.map((h) => `${h.file}:${h.line} ${h.detail}`),
      'Fermer une couche : useDismissable (@/lib/hooks/useDismissable).',
    ).toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('le reste de features/ est gelé à sa valeur mesurée (cliquet : ne fait que descendre)', () => {
    const files = new Set(
      tsxDeFeatures()
        .filter((s) => !s.rel.startsWith(REPLAY))
        .flatMap(dismissPatterns)
        .map((h) => h.file),
    )
    expect(
      files.size,
      `${files.size} fichier(s) hors rejeu, plafond ${PLAFOND_HORS_REJEU} : migrer vers useDismissable, ` +
        `et abaisser PLAFOND_HORS_REJEU à ${files.size} quand le compte baisse.\n${[...files].sort().join('\n')}`,
    ).toBe(PLAFOND_HORS_REJEU)
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit les trois motifs, pas un autre événement ni un autre nom', () => {
    const n = (code: string): number => dismissPatterns(parseSource('features/x/a.tsx', code)).length
    expect(n("document.addEventListener('pointerdown', f)")).toBe(1)
    expect(n("el.addEventListener('mousedown', f)")).toBe(1)
    expect(n("if (e.key === 'Escape') close()")).toBe(1)
    expect(n("if ('Escape' === e.key) close()")).toBe(1)
    expect(n("document.addEventListener('keydown', f)")).toBe(0)
    expect(n("if (e.key !== 'Escape') return")).toBe(0)
    expect(n("const s = \"key === 'Escape'\"")).toBe(0)
  })
})
