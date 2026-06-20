/**
 * Tests — buildCombatYieldOption (Phase 3 P3.F).
 */
import { describe, it, expect } from 'vitest'
import { buildCombatYieldOption } from './TimeseriesCombatYield'
import type { ChartSeries } from '@/components/charts/ChartCard'

interface CombatYieldPoint {
  x: string
  y: number | null
}

const labels = {
  ocSeries: 'Offensive (OC)',
  drSeries: 'Defensive (DR)',
  ocReference: 'p80 OC',
  drReference: 'p80 DR',
}

function makeSeries(
  ocPoints: CombatYieldPoint[],
  drPoints: CombatYieldPoint[],
): ChartSeries<CombatYieldPoint>[] {
  return [
    { key: 'combat.oc', meta: { gamertag: labels.ocSeries }, datapoints: ocPoints },
    { key: 'combat.dr', meta: { gamertag: labels.drSeries }, datapoints: drPoints },
  ]
}

interface OptionShape {
  series?: Array<{
    type?: string
    name?: string
    data?: Array<[string, number | null]>
    markLine?: { data?: Array<{ yAxis?: number; name?: string }> }
  }>
  legend?: { data?: string[] }
  xAxis?: { type?: string }
  yAxis?: { type?: string; min?: number }
  backgroundColor?: string
}

describe('buildCombatYieldOption', () => {
  it('retourne option vide si aucune série', () => {
    const opt = buildCombatYieldOption([], labels) as OptionShape
    expect(opt.series).toBeUndefined()
    expect(opt.backgroundColor).toBeDefined()
  })

  it('génère 2 séries (OC + DR) dans l\'ordre', () => {
    const opt = buildCombatYieldOption(
      makeSeries(
        [{ x: '2025-01-01T10:00:00Z', y: 0.7 }],
        [{ x: '2025-01-01T10:00:00Z', y: 1.5 }],
      ),
      labels,
    ) as OptionShape
    expect(opt.series).toHaveLength(2)
    expect(opt.series?.[0].name).toBe(labels.ocSeries)
    expect(opt.series?.[1].name).toBe(labels.drSeries)
  })

  it('mappe les points en tuples [x, y]', () => {
    const opt = buildCombatYieldOption(
      makeSeries(
        [
          { x: '2025-01-01', y: 0.5 },
          { x: '2025-01-02', y: 0.8 },
        ],
        [
          { x: '2025-01-01', y: 1.2 },
          { x: '2025-01-02', y: 1.4 },
        ],
      ),
      labels,
    ) as OptionShape
    expect(opt.series?.[0].data).toEqual([
      ['2025-01-01', 0.5],
      ['2025-01-02', 0.8],
    ])
    expect(opt.series?.[1].data).toEqual([
      ['2025-01-01', 1.2],
      ['2025-01-02', 1.4],
    ])
  })

  it('chaque série a une markLine de référence p80', () => {
    const opt = buildCombatYieldOption(
      makeSeries([{ x: '2025-01-01', y: 0.5 }], [{ x: '2025-01-01', y: 1.0 }]),
      labels,
    ) as OptionShape
    // OC : valeur brute du repère = 0.90.
    expect(opt.series?.[0].markLine?.data?.[0].yAxis).toBe(0.90)
    // DR : la résistance défensive est affichée normalisée depuis 1.0 (cf.
    // DR_DISPLAY_P80 = DR_P80 - 1.0 = 0.65). Le graphe centre la résistance
    // neutre à 0 pour une lecture intuitive "écart vs neutre".
    expect(opt.series?.[1].markLine?.data?.[0].yAxis).toBeCloseTo(0.65, 5)
  })

  it('legend liste les 2 noms de séries', () => {
    const opt = buildCombatYieldOption(
      makeSeries([{ x: '2025-01-01', y: 0.5 }], [{ x: '2025-01-01', y: 1.0 }]),
      labels,
    ) as OptionShape
    expect(opt.legend?.data).toEqual([labels.ocSeries, labels.drSeries])
  })

  it('axe X = time, axe Y commence à 0', () => {
    const opt = buildCombatYieldOption(
      makeSeries([{ x: '2025-01-01', y: 0.5 }], [{ x: '2025-01-01', y: 1.0 }]),
      labels,
    ) as OptionShape
    expect(opt.xAxis?.type).toBe('time')
    expect(opt.yAxis?.type).toBe('value')
    expect(opt.yAxis?.min).toBe(0)
  })
})
