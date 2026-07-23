/**
 * squadEngagementGapChart.test.ts — « Écart d'engagement cumulé » par joueur.
 *
 * - `cumulativeEngagementGapSeries` (pur) : cumul de (pace_observed −
 *   team_expected) × durée/60, report D5.
 * - `buildSquadEngagementGapOption` (pur) : 1 line/joueur, couleurs, markLine 0.
 */
import { describe, it, expect } from 'vitest'

import type { SquadEngagementSessionAPI } from '@/lib/api/types'
import {
  buildSquadEngagementGapOption,
  cumulativeEngagementGapSeries,
} from './squadEngagementGapChart'

interface LineSeries {
  name: string
  type: string
  data: Array<number | null>
  lineStyle: { color: string }
  markLine?: { data: Array<{ yAxis: number }> }
  areaStyle?: unknown
}

describe('cumulativeEngagementGapSeries', () => {
  it('cumul de (observé − attendu) × durée/60 (événements)', () => {
    // résidu [3, -1] évén./min ; durée 600 s (10 min) → contrib [30, -10] ; cumul [30, 20].
    const data = cumulativeEngagementGapSeries([5, 3], [2, 4], [600, 600])
    expect(data).toEqual([30, 20])
  })

  it('report D5 : durée nulle (0) donne une contribution 0, pas null', () => {
    // durée 0 → contribution 0 (résidu × 0), le cumul ne bouge pas.
    const data = cumulativeEngagementGapSeries([5, 3], [2, 4], [600, 0])
    expect(data).toEqual([30, 30])
  })
})

describe('buildSquadEngagementGapOption', () => {
  const COLORS = { Me: '#aaa', F1: '#bbb' }

  function session(): SquadEngagementSessionAPI {
    return {
      labels: ['M1', 'M2'],
      map_names: ['Aquarius', 'Live Fire'],
      lobby_per_player: [10, 12],
      team_expected: [2, 4],
      team_observed: [5, 6],
      durations_seconds: [600, 600],
      players: [
        { xuid: 'x1', gamertag: 'Me', pace_observed: [5, 3] }, // cumul [30, 20]
        { xuid: 'x2', gamertag: 'F1', pace_observed: [4, 6] }, // résidu [2,2] → contrib [20,20] → [20,40]
      ],
    }
  }

  it('vide → option de fond minimale (pas de série)', () => {
    const empty: SquadEngagementSessionAPI = {
      labels: [],
      map_names: [],
      lobby_per_player: [],
      team_expected: [],
      team_observed: [],
      durations_seconds: [],
      players: [],
    }
    const opt = buildSquadEngagementGapOption(empty, { colorByPlayer: {} })
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
    expect(opt.series).toBeUndefined()
  })

  it('1 série line par joueur : data = cumul, couleur du joueur', () => {
    const opt = buildSquadEngagementGapOption(session(), { colorByPlayer: COLORS })
    const series = opt.series as unknown as LineSeries[]
    expect(series).toHaveLength(2)
    expect(series.map((s) => s.name)).toEqual(['Me', 'F1'])
    expect(series[0].data).toEqual([30, 20])
    expect(series[1].data).toEqual([20, 40])
    expect(series[0].lineStyle.color).toBe('#aaa')
    expect(series[1].lineStyle.color).toBe('#bbb')
  })

  it('markLine 0 sur le premier joueur uniquement, aucune aire', () => {
    const opt = buildSquadEngagementGapOption(session(), { colorByPlayer: COLORS })
    const series = opt.series as unknown as LineSeries[]
    expect(series[0].markLine?.data[0].yAxis).toBe(0)
    expect(series[1].markLine).toBeUndefined()
    expect(series[0].areaStyle).toBeUndefined()
  })
})
