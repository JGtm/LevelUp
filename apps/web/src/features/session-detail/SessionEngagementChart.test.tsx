/**
 * Tests buildSessionEngagementOption — barres d'engagement par match + markLine moyenne.
 */
import { describe, expect, it } from 'vitest'

import { buildSessionEngagementOption } from './SessionEngagementChart'

interface OptShape {
  series?: Array<{
    type: string
    data: Array<{ value: number; itemStyle: { color: string } }>
    markLine: { data: Array<{ yAxis: number }> }
    markArea?: unknown
  }>
  yAxis?: { name?: string }
  xAxis?: { data: string[] }
}

describe('buildSessionEngagementOption', () => {
  it('barres par match (colorées par signe) + markLine moyenne, style aligné sur Perf', () => {
    const opt = buildSessionEngagementOption(
      [
        {
          key: 'e',
          datapoints: [
            { label: '#1 Map', value: 2 },
            { label: '#2 Map', value: -4 },
          ],
        },
      ],
      { meanLabel: 'Moyenne', axisName: 'évén./min vs attendu' },
    ) as unknown as OptShape

    const bar = opt.series![0]
    expect(bar.type).toBe('bar')
    expect(bar.data).toHaveLength(2)
    expect(bar.data.every((d) => 'color' in d.itemStyle)).toBe(true)
    expect(bar.markLine.data[0].yAxis).toBe(-1) // (2 + -4) / 2
    // Style "propre" aligné sur Score de performance : pas de bande markArea ni de nom d'axe Y.
    expect(bar.markArea).toBeUndefined()
    expect(opt.yAxis!.name).toBeUndefined()
    expect(opt.xAxis!.data).toEqual(['#1 Map', '#2 Map'])
  })

  it('option vide sans série', () => {
    const opt = buildSessionEngagementOption([], { meanLabel: 'M', axisName: 'A' }) as unknown as OptShape
    expect(opt.series).toBeUndefined()
  })
})
