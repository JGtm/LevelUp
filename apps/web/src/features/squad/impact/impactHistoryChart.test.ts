import { describe, expect, it } from 'vitest'

import type { SemanticToken } from '@/lib/accessibility'
import type { EChartsThemeColors } from '@/lib/echarts/themeColors'

import { impactHistory3 } from './impactHistory.fixtures'
import { buildImpactHistoryView, type ImpactHistoryView } from './impactHistory.logic'
import { buildImpactHistoryOption, type ImpactChartColors } from './impactHistoryChart'
import { IMPACT_HISTORY_TEXT } from './impactHistoryStrings'

const t = IMPACT_HISTORY_TEXT.fr

// Couleurs factices lisibles (le rendu réel les résout depuis les jetons).
const theme = {
  axisLabel: 'c-axis', axisLine: 'c-line', splitLine: 'c-split', splitAreaA: 'c-area-a', splitAreaB: 'c-area-b',
  text: 'c-text', tooltipBg: 'c-tip', tooltipBorder: 'c-tip-border', card: 'c-card', isDark: false,
} satisfies EChartsThemeColors
const colors: ImpactChartColors = {
  players: ['c-p1', 'c-p2', 'c-p3'],
  role: (token: SemanticToken) => `c-${token}`,
  theme,
}

function view(): ImpactHistoryView {
  const v = buildImpactHistoryView(impactHistory3, 'fr', t)
  if (!v) throw new Error('vue attendue')
  return v
}

interface Series {
  data: number[][]
  renderItem: (params: unknown, api: unknown) => { children: Record<string, unknown>[] } | null
}

/** Une API de rendu factice : 100 px par soirée, 10 px par point, zéro à y = 200. */
const api = (values: number[]) => ({
  value: (i: number) => values[i],
  coord: ([x, y]: number[]) => [50 + 100 * x, 200 - 10 * y],
  size: () => [100, 0],
})

function option(narrow = false) {
  return buildImpactHistoryOption(view(), colors, t, narrow) as {
    series: Series[]
    tooltip: { formatter: (p: unknown) => string }
    xAxis: { axisLabel: { interval: (i: number) => boolean } }
    grid: { bottom: number }
  }
}

describe('buildImpactHistoryOption', () => {
  it('une donnée par barre, étendue de l’axe = gains et pertes', () => {
    const bars = option().series[1]
    expect(bars.data).toHaveLength(9)
    expect(bars.data[0]).toEqual([0, 0, 3, -12, -9])
  })

  it('barre : cible de survol, segments aux couleurs des rôles, pastille du joueur', () => {
    const bars = option().series[1]
    const out = bars.renderItem({ dataIndex: 0, coordSys: { x: 0, y: 0, height: 300 } }, api(bars.data[0]))
    const fills = out?.children.map((c) => (c.style as { fill: string }).fill)
    expect(fills).toEqual([
      'transparent',
      'c-impact-gain-1', 'c-impact-gain-4',
      'c-impact-loss-1', 'c-impact-loss-2', 'c-impact-loss-3',
      'c-p1',
    ])
    // Le segment du bout porte les coins arrondis, pas celui contre l'axe.
    expect((out?.children[1].shape as { r: unknown }).r).toBe(0)
    expect((out?.children[2].shape as { r: number[] }).r).toEqual([3, 3, 0, 0])
  })

  it('courbes : halo puis trait à la couleur du joueur, un losange par soirée, les nets extrêmes écrits', () => {
    const nets = option().series[2]
    const jgtm = nets.renderItem({ coordSys: { x: 0 } }, api(nets.data[0]))?.children ?? []
    expect(jgtm.filter((c) => c.type === 'polyline').map((c) => (c.style as { stroke: string }).stroke)).toEqual(['c-card', 'c-p1'])
    expect(jgtm.filter((c) => c.type === 'polygon')).toHaveLength(3)
    expect(jgtm.filter((c) => c.type === 'text').map((c) => (c.style as { text: string }).text)).toEqual(['−9'])
    const madina = nets.renderItem({ coordSys: { x: 0 } }, api(nets.data[2]))?.children ?? []
    expect(madina.filter((c) => c.type === 'text')).toHaveLength(0)
  })

  it('infobulle de la barre, données échappées', () => {
    const html = option().tooltip.formatter({ seriesIndex: 1, dataIndex: 7 })
    expect(html).toContain('Chocoboflor')
    expect(html).toContain('1 match, aucune victoire')
    expect(html).toContain('Boulet ×1')
    expect(html).toContain('Net <b>−4</b>')
    expect(option().tooltip.formatter({ seriesIndex: 2, dataIndex: 0 })).toBe('')
  })

  it('sur téléphone, une date sur deux ; deux lignes d’axe quand une date est partagée', () => {
    const o = option(true)
    expect([0, 1, 2].map(o.xAxis.axisLabel.interval)).toEqual([true, false, true])
    expect(o.grid.bottom).toBe(58)
  })
})
