/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (plan architecture, étape 7.7) : aucun `toggleSort` local dans `features/`.
 *
 * Le tri par clic sur un en-tête (sens du premier clic, bascule, valeurs absentes en dernier, égaux
 * dans l'ordre reçu) est écrit une fois, dans `lib/table/useColumnSort` (testé), et ordonné par
 * TanStack Table. Un tableau qui garde son propre `toggleSort` recopie cette règle et diverge
 * (11 copies relevées le 2026-11-13). Plafond 0, sans allowlist.
 */
import ts from 'typescript'
import { describe, expect, it } from 'vitest'
import { formatHits, parseSource, SCAN_TIMEOUT_MS, scanSources, visitAll, lineOf, type Hit, type ScannedSource } from '@/lib/archScan.testutil'

function toggleSortNames(source: ScannedSource): Hit[] {
  const hits: Hit[] = []
  visitAll(source, (node) => {
    if (ts.isIdentifier(node) && node.text === 'toggleSort') {
      hits.push({ file: source.rel, line: lineOf(source, node), detail: 'toggleSort' })
    }
  })
  return hits
}

describe('garde-rail : pas de toggleSort local dans features/', () => {
  it('aucun identifiant toggleSort dans features/', () => {
    const hits = scanSources({ under: 'features' }).flatMap(toggleSortNames)
    expect(
      hits.length,
      formatHits(hits, 'Utiliser useColumnSort (lib/table/useColumnSort) : sort.header(clé) fournit active, dir et onClick à SortableTh.'),
    ).toBe(0)
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit la déclaration et l’appel, pas un commentaire ni une chaîne', () => {
    const seen = (code: string): number => toggleSortNames(parseSource('features/x/T.tsx', code)).length
    expect(seen('function toggleSort(key: string) {}')).toBe(1)
    expect(seen('const x = () => toggleSort("a")')).toBe(1)
    expect(seen('// toggleSort\nconst a = 1')).toBe(0)
    expect(seen("const s = 'toggleSort'")).toBe(0)
    expect(seen('const toggleSortKey = 1')).toBe(0)
  })
})
