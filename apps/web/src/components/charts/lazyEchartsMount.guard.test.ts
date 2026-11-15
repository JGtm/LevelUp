/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (plan architecture, étape 7.3) : `<LazyECharts` ne se monte que dans
 * `components/charts/`, et tout fichier qui le monte écoute la palette d'accessibilité.
 *
 * Pourquoi : un graphe monté en direct par une feature échappe à `ChartCard`, donc au rebuild
 * de l'option sur changement de thème ET de palette (la palette s'applique sur `:root` sans
 * toucher `data-theme`). Ses couleurs, lues à l'appel, restaient celles de l'ancienne palette
 * jusqu'au remontage (audit TEC-1 : `CumulativeFragGapChart`, `OutcomeSequenceTape`).
 *
 * Deux contrôles, plafond 0, sans allowlist :
 *  1. aucun `<LazyECharts` hors de `components/charts/` ;
 *  2. chaque fichier qui monte `<LazyECharts` appelle `useColorPaletteVersion`
 *     (`ChartCard` l'appelle pour tous les wrappers ; les montages directs le font eux-mêmes).
 */
import { describe, expect, it } from 'vitest'
import ts from 'typescript'
import {
  callsNamed,
  formatHits,
  isUnder,
  lineOf,
  parseSource,
  SCAN_TIMEOUT_MS,
  scanSources,
  visitAll,
  type Hit,
  type ScannedSource,
} from '@/lib/archScan.testutil'

const CHARTS_DIR = ['components/charts/']
const PALETTE_HOOK: ReadonlySet<string> = new Set(['useColorPaletteVersion'])

/** Montages JSX de `<LazyECharts ...>` (ouvrant ou auto-fermant), jamais le nom dans un commentaire ou une chaîne. */
function lazyEchartsMounts(source: ScannedSource): Hit[] {
  const hits: Hit[] = []
  visitAll(source, (node) => {
    const tag = ts.isJsxSelfClosingElement(node) ? node.tagName : ts.isJsxOpeningElement(node) ? node.tagName : null
    if (tag !== null && ts.isIdentifier(tag) && tag.text === 'LazyECharts') {
      hits.push({ file: source.rel, line: lineOf(source, node), detail: '<LazyECharts' })
    }
  })
  return hits
}

const HINT_LOCATION =
  'Monter le graphe par un wrapper de components/charts/ (ChartCard, ou un composant de ce dossier qui écoute la palette).'
const HINT_PALETTE =
  "Appeler `useColorPaletteVersion()` et mettre sa valeur dans les dépendances de l'option, sinon le graphe garde l'ancienne palette jusqu'au remontage."

describe('garde-rail : montage de LazyECharts', () => {
  const sources = scanSources()

  it('aucun <LazyECharts hors de components/charts/', () => {
    const hits = sources.filter((s) => !isUnder(s.rel, CHARTS_DIR)).flatMap(lazyEchartsMounts)
    expect(hits.length, formatHits(hits, HINT_LOCATION)).toBe(0)
  }, SCAN_TIMEOUT_MS)

  it('tout fichier qui monte <LazyECharts écoute la palette', () => {
    const hits = sources
      .filter((s) => lazyEchartsMounts(s).length > 0 && callsNamed(s, PALETTE_HOOK).length === 0)
      .map((s) => ({ file: s.rel, line: 1, detail: 'monte <LazyECharts sans useColorPaletteVersion()' }))
    expect(hits.length, formatHits(hits, HINT_PALETTE)).toBe(0)
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit les deux formes de montage et ignore commentaires et chaînes', () => {
    const mounts = (code: string): number => lazyEchartsMounts(parseSource('x/C.tsx', code)).length
    expect(mounts('const a = <LazyECharts option={o} />')).toBe(1)
    expect(mounts('const a = <LazyECharts option={o}>x</LazyECharts>')).toBe(1)
    expect(mounts('// <LazyECharts option={o} />\nconst a = 1')).toBe(0)
    expect(mounts("const s = '<LazyECharts'")).toBe(0)
    expect(mounts('export function LazyECharts() { return null }')).toBe(0)
  })

  it('le garde n’est pas fantôme : ChartCard et les deux montages directs sont vus', () => {
    const mounting = sources.filter((s) => lazyEchartsMounts(s).length > 0).map((s) => s.rel)
    for (const expected of [
      'components/charts/ChartCard.tsx',
      'components/charts/CumulativeFragGapChart.tsx',
      'components/charts/OutcomeSequenceTape.tsx',
    ]) {
      expect(mounting).toContain(expected)
    }
  }, SCAN_TIMEOUT_MS)
})
