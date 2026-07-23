/**
 * squadNetLivesChart.test.ts — « Balance des dégâts cumulée » par joueur.
 *
 * - `cumulativeNetLivesSeries` (pur) : cumul par match_order + report D5 (dégâts
 *   absents) + robustesse au désordre / trou d'intersection, division par le barème.
 * - `buildNetLivesCumulativeOption` (pur) : 1 line/joueur, couleurs, markLine 0, pas d'aire.
 */
import { describe, it, expect } from 'vitest'

import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import {
  buildNetLivesCumulativeOption,
  cumulativeNetLivesSeries,
} from './squadNetLivesChart'

const HP = 225

function pt(
  order: number,
  damageDealt: number | undefined,
  damageTaken: number | undefined,
): SquadPerformanceSeriesPoint {
  return {
    match_id: `m${order}`,
    start_time: '2026-04-30T12:00:00Z',
    match_order: order,
    kills: 10,
    deaths: 5,
    assists: 3,
    damage_dealt: damageDealt,
    damage_taken: damageTaken,
  } as SquadPerformanceSeriesPoint
}

interface LineSeries {
  name: string
  type: string
  data: Array<number | null>
  lineStyle: { color: string }
  markLine?: { data: Array<{ yAxis: number }> }
  areaStyle?: unknown
}

describe('cumulativeNetLivesSeries', () => {
  it('cumul de la balance par match_order croissant (÷ PV-pour-tuer)', () => {
    // m0 : (900-450)/225 = +2 ; m1 : (450-900)/225 = -2 ; m2 : (675-225)/225 = +2.
    const data = cumulativeNetLivesSeries(
      [pt(0, 900, 450), pt(1, 450, 900), pt(2, 675, 225)],
      3,
      HP,
    )
    expect(data).toEqual([2, 0, 2])
  })

  it('report D5 : un match sans dégâts subis saute le cumul (reporte)', () => {
    const data = cumulativeNetLivesSeries(
      [pt(0, 900, 450), pt(1, 450, undefined), pt(2, 675, 225)],
      3,
      HP,
    )
    expect(data).toEqual([2, 2, 4])
  })

  it('match_order désordonné : trie avant de cumuler', () => {
    const data = cumulativeNetLivesSeries(
      [pt(2, 675, 225), pt(0, 900, 450), pt(1, 450, 900)],
      3,
      HP,
    )
    expect(data).toEqual([2, 0, 2])
  })

  it('trou d\'intersection (aucun point) reste null', () => {
    const data = cumulativeNetLivesSeries([pt(0, 900, 450), pt(2, 675, 225)], 3, HP)
    expect(data).toEqual([2, null, 4])
  })
})

describe('buildNetLivesCumulativeOption', () => {
  const COLORS = { Me: '#aaa', F1: '#bbb' }
  const ORDER = ['Me', 'F1']

  it('vide → option de fond minimale (pas de série)', () => {
    const opt = buildNetLivesCumulativeOption({}, { colorByPlayer: {}, hp: HP })
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
    expect(opt.series).toBeUndefined()
  })

  it('1 série line par joueur : data = cumul, couleur du joueur appliquée', () => {
    const rows = {
      Me: [pt(0, 900, 450), pt(1, 450, 900)], // +2, 0
      F1: [pt(0, 675, 225), pt(1, 900, 675)], // +2, +3
    }
    const opt = buildNetLivesCumulativeOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ORDER,
      hp: HP,
    })
    const series = opt.series as unknown as LineSeries[]
    expect(series).toHaveLength(2)
    expect(series.map((s) => s.name)).toEqual(['Me', 'F1'])
    expect(series.every((s) => s.type === 'line')).toBe(true)
    expect(series[0].data).toEqual([2, 0])
    expect(series[1].data).toEqual([2, 3])
    expect(series[0].lineStyle.color).toBe('#aaa')
    expect(series[1].lineStyle.color).toBe('#bbb')
  })

  it('markLine 0 sur le premier joueur uniquement, aucune aire (multi-séries)', () => {
    const rows = { Me: [pt(0, 900, 450)], F1: [pt(0, 675, 225)] }
    const opt = buildNetLivesCumulativeOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ORDER,
      hp: HP,
    })
    const series = opt.series as unknown as LineSeries[]
    expect(series[0].markLine?.data[0].yAxis).toBe(0)
    expect(series[1].markLine).toBeUndefined()
    expect(series[0].areaStyle).toBeUndefined()
    expect(series[1].areaStyle).toBeUndefined()
  })

  it('joueur masqué (hiddenPlayers) → série vidée (null)', () => {
    const rows = {
      Me: [pt(0, 900, 450), pt(1, 450, 900)],
      F1: [pt(0, 675, 225), pt(1, 900, 675)],
    }
    const opt = buildNetLivesCumulativeOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ORDER,
      hp: HP,
      hiddenPlayers: new Set(['F1']),
    })
    const series = opt.series as unknown as LineSeries[]
    expect(series[0].data).toEqual([2, 0])
    expect(series[1].data).toEqual([null, null])
  })
})
