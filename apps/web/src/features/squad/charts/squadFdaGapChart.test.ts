/**
 * squadFdaGapChart.test.ts — « Écart cumulé au FDA attendu » (Lot C, D3/D5).
 *
 * - `cumulativeFdaGapSeries` (pur) : cumul par match_order + trous D5 (report) +
 *   robustesse au désordre / non-fini.
 * - `meanFdaGapPerMatch` (pur) : écart moyen sur les matchs AVEC attendu (pastille).
 * - `buildFdaGapCumulativeOption` (pur) : 1 line/joueur, couleurs, markLine 0, pas d'aire.
 */
import { describe, it, expect } from 'vitest'

import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import {
  buildFdaGapCumulativeOption,
  cumulativeFdaGapSeries,
  meanFdaGapPerMatch,
} from './squadFdaGapChart'

function pt(
  order: number,
  kda: number | undefined,
  kdaExpected: number | undefined,
): SquadPerformanceSeriesPoint {
  return {
    match_id: `m${order}`,
    start_time: '2026-04-30T12:00:00Z',
    match_order: order,
    kills: 10,
    deaths: 5,
    assists: 3,
    kda,
    kda_expected: kdaExpected,
  }
}

interface LineSeries {
  name: string
  type: string
  data: Array<number | null>
  lineStyle: { color: string }
  markLine?: { data: Array<{ yAxis: number }> }
  areaStyle?: unknown
}

describe('cumulativeFdaGapSeries', () => {
  it('cumul du différentiel par match_order croissant', () => {
    const data = cumulativeFdaGapSeries([pt(0, 1.5, 1.0), pt(1, 0.8, 1.2), pt(2, 2.0, 1.0)], 3)
    expect(data).toEqual([0.5, 0.1, 1.1])
  })

  it('report D5 : un match sans attendu saute le cumul (reporte la dernière valeur)', () => {
    const data = cumulativeFdaGapSeries([pt(0, 1.5, 1.0), pt(1, 0.8, undefined), pt(2, 2.0, 1.0)], 3)
    expect(data).toEqual([0.5, 0.5, 1.5])
  })

  it('report D5 côté réel manquant également', () => {
    const data = cumulativeFdaGapSeries([pt(0, 1.5, 1.0), pt(1, undefined, 1.0)], 2)
    expect(data).toEqual([0.5, 0.5])
  })

  it('valeur non-finie (Infinity) traitée comme absente (D5)', () => {
    const data = cumulativeFdaGapSeries([pt(0, 1.5, 1.0), pt(1, 0.8, Infinity)], 2)
    expect(data).toEqual([0.5, 0.5])
  })

  it('match_order désordonné : trie avant de cumuler', () => {
    const data = cumulativeFdaGapSeries([pt(2, 2.0, 1.0), pt(0, 1.5, 1.0), pt(1, 0.8, 1.2)], 3)
    expect(data).toEqual([0.5, 0.1, 1.1])
  })

  it('trou d\'intersection (aucune ligne) reste null', () => {
    const data = cumulativeFdaGapSeries([pt(0, 1.5, 1.0), pt(2, 2.0, 1.0)], 3)
    expect(data).toEqual([0.5, null, 1.5])
  })
})

describe('meanFdaGapPerMatch', () => {
  it('moyenne des écarts sur les matchs avec attendu', () => {
    expect(meanFdaGapPerMatch([pt(0, 1.6, 1.0), pt(1, 1.4, 1.0)])).toBe(0.5)
  })

  it('ignore les matchs sans attendu (D5)', () => {
    expect(meanFdaGapPerMatch([pt(0, 1.6, 1.0), pt(1, 0.8, undefined)])).toBe(0.6)
  })

  it('null si aucun match exploitable', () => {
    expect(meanFdaGapPerMatch([pt(0, 1.0, undefined)])).toBeNull()
    expect(meanFdaGapPerMatch([])).toBeNull()
  })
})

describe('buildFdaGapCumulativeOption', () => {
  const COLORS = { Me: '#aaa', F1: '#bbb' }
  const ORDER = ['Me', 'F1']

  it('vide → option de fond minimale (pas de série)', () => {
    const opt = buildFdaGapCumulativeOption({}, { colorByPlayer: {} })
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
    expect(opt.series).toBeUndefined()
  })

  it('1 série line par joueur : data = cumul, couleur du joueur appliquée', () => {
    const rows = {
      Me: [pt(0, 1.5, 1.0), pt(1, 0.8, 1.2)],
      F1: [pt(0, 2.0, 1.0), pt(1, 1.0, 1.5)],
    }
    const opt = buildFdaGapCumulativeOption(rows, { colorByPlayer: COLORS, playerOrder: ORDER })
    const series = opt.series as unknown as LineSeries[]
    expect(series).toHaveLength(2)
    expect(series.map((s) => s.name)).toEqual(['Me', 'F1'])
    expect(series.every((s) => s.type === 'line')).toBe(true)
    expect(series[0].data).toEqual([0.5, 0.1])
    expect(series[1].data).toEqual([1, 0.5])
    expect(series[0].lineStyle.color).toBe('#aaa')
    expect(series[1].lineStyle.color).toBe('#bbb')
  })

  it('markLine 0 sur le premier joueur uniquement, aucune aire (multi-séries)', () => {
    const rows = { Me: [pt(0, 1.5, 1.0)], F1: [pt(0, 2.0, 1.0)] }
    const opt = buildFdaGapCumulativeOption(rows, { colorByPlayer: COLORS, playerOrder: ORDER })
    const series = opt.series as unknown as LineSeries[]
    expect(series[0].markLine?.data[0].yAxis).toBe(0)
    expect(series[1].markLine).toBeUndefined()
    expect(series[0].areaStyle).toBeUndefined()
    expect(series[1].areaStyle).toBeUndefined()
  })

  it('joueur masqué (hiddenPlayers) → série vidée (null)', () => {
    const rows = {
      Me: [pt(0, 1.5, 1.0), pt(1, 0.8, 1.2)],
      F1: [pt(0, 2.0, 1.0), pt(1, 1.0, 1.5)],
    }
    const opt = buildFdaGapCumulativeOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ORDER,
      hiddenPlayers: new Set(['F1']),
    })
    const series = opt.series as unknown as LineSeries[]
    expect(series[0].data).toEqual([0.5, 0.1])
    expect(series[1].data).toEqual([null, null])
  })
})
