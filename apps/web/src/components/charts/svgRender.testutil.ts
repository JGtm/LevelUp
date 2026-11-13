/**
 * Rendu SVG d'une option de graphe côté serveur, sans canevas : de quoi vérifier ce qu'ECharts
 * écrit réellement (libellés d'axe, textes) dans un test de feature, sans que la feature importe
 * la bibliothèque (`chartsImportBoundary.guard.test.ts`).
 */
import * as echarts from 'echarts'

import type { ChartOption } from './chartOption'

export function renderOptionToSvg(option: ChartOption, width: number, height: number): string {
  const chart = echarts.init(null, null, { renderer: 'svg', ssr: true, width, height })
  chart.setOption(option)
  const svg = chart.renderToSVGString()
  chart.dispose()
  return svg
}
