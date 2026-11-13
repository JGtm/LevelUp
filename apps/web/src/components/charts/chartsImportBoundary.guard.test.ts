/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (plan architecture, étape 7.5) : la bibliothèque de graphes (`echarts`,
 * `echarts/core`, `echarts-for-react`…) ne s'importe que depuis `components/charts/`.
 *
 * Le type d'option, seul import qu'une feature avait besoin de faire (74 fichiers), vient de
 * `components/charts/chartOption.ts` (`ChartOption`) ; un test qui veut le rendu réel passe par
 * `svgRender.testutil.ts`. Changer de bibliothèque ne touche ainsi que ce dossier. Types compris :
 * `import type` et `import('echarts')` comptent comme un import.
 *
 * Plafond 0, sans allowlist. Périmètre : tout `src/` tests compris, sauf `components/charts/`.
 */
import { describe, expect, it } from 'vitest'
import {
  formatHits,
  isUnder,
  moduleSpecifiers,
  parseSource,
  SCAN_TIMEOUT_MS,
  scanSources,
  type Hit,
  type ScannedSource,
} from '@/lib/archScan.testutil'

const CHARTS_DIR = ['components/charts/']
const CHART_LIBRARY = /^echarts(?:$|\/)|^echarts-for-react(?:$|\/)/

function libraryImports(source: ScannedSource): Hit[] {
  return moduleSpecifiers(source)
    .filter((m) => CHART_LIBRARY.test(m.specifier))
    .map((m) => ({ file: source.rel, line: m.line, detail: `importe « ${m.specifier} »` }))
}

const HINT =
  "Le type d'option vient de '@/components/charts/chartOption' (ChartOption) ; un graphe se monte par un wrapper de components/charts/ ; un test qui veut le rendu réel passe par '@/components/charts/svgRender.testutil'."

describe('garde-rail : la bibliothèque de graphes reste dans components/charts/', () => {
  it('aucun import de echarts hors de components/charts/', () => {
    const hits = scanSources({ includeTests: true })
      .filter((s) => !isUnder(s.rel, CHARTS_DIR))
      .flatMap(libraryImports)
    expect(hits.length, formatHits(hits, HINT)).toBe(0)
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit import, import type, réexport et import dynamique, pas un commentaire', () => {
    const seen = (code: string): number => libraryImports(parseSource('features/x/a.ts', code)).length
    expect(seen("import * as echarts from 'echarts'")).toBe(1)
    expect(seen("import type { EChartsCoreOption } from 'echarts/core'")).toBe(1)
    expect(seen("export type { EChartsType } from 'echarts/core'")).toBe(1)
    expect(seen("const m = await import('echarts-for-react')")).toBe(1)
    expect(seen("type T = import('echarts/core').EChartsType")).toBe(1)
    expect(seen("// import * as echarts from 'echarts'\nconst a = 1")).toBe(0)
    expect(seen("import { x } from '@/lib/echarts/themeColors'")).toBe(0)
    expect(seen("import { y } from 'echarts-like'")).toBe(0)
  })

  it('le garde n’est pas fantôme : le dossier des graphes importe bien la bibliothèque', () => {
    const inside = scanSources({ under: 'components/charts' }).flatMap(libraryImports)
    expect(inside.length).toBeGreaterThan(0)
  })
})
