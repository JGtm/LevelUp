/**
 * hsPkChart.test.ts — buildHsPkSeries produit un ChartSeries<ChartPointStacked>
 * sans libellé hardcodé.
 */
import { describe, it, expect } from 'vitest'
import { buildHsPkSeries } from './hsPkChart'
import { SQUAD_HSPK_METRICS } from '../metrics'
import type { TeammateRow } from '@/lib/api/types'

const makeRow = (gamertag: string, hs: number, pk: number): TeammateRow => ({
  gamertag,
  xuid: 'x',
  encounter_count: 5,
  last_seen_at: null,
  with_kpis: {
    match_count: 5,
    wins: 3,
    kd_ratio: 1.2,
    win_rate: 0.6,
    accuracy: 0.4,
    kills_per_game: 10,
    assists_per_game: 4,
    headshot_kills_per_game: hs,
    perfect_kills_per_game: pk,
  },
  without_kpis: null,
})

describe('buildHsPkSeries', () => {
  const args = {
    hsMetric: SQUAD_HSPK_METRICS.hs,
    pkMetric: SQUAD_HSPK_METRICS.pk,
    hsLabel: 'HS Label',
    pkLabel: 'PK Label',
  }

  it('retourne [] pour rows vide', () => {
    expect(buildHsPkSeries({ rows: [], ...args })).toEqual([])
  })

  it('retourne 1 série avec 1 datapoint par gamertag', () => {
    const result = buildHsPkSeries({
      rows: [makeRow('A', 3, 1), makeRow('B', 4, 0)],
      ...args,
    })
    expect(result).toHaveLength(1)
    expect(result[0].datapoints).toHaveLength(2)
    expect(result[0].datapoints[0].category).toBe('A')
    expect(result[0].datapoints[1].category).toBe('B')
  })

  it('utilise les labels passés comme clés de composants', () => {
    const result = buildHsPkSeries({ rows: [makeRow('A', 3, 1)], ...args })
    const dp = result[0].datapoints[0]
    expect(dp.components).toHaveProperty('HS Label')
    expect(dp.components).toHaveProperty('PK Label')
  })

  it('utilise les extracteurs SquadMetric pour les valeurs', () => {
    const result = buildHsPkSeries({
      rows: [makeRow('A', 4, 2), makeRow('B', 0, 0)],
      ...args,
    })
    expect(result[0].datapoints[0].components['HS Label']).toBe(4)
    expect(result[0].datapoints[0].components['PK Label']).toBe(2)
    expect(result[0].datapoints[1].components['HS Label']).toBe(0)
    expect(result[0].datapoints[1].components['PK Label']).toBe(0)
  })

  it('aucun libellé hardcodé en français résiduel', () => {
    const result = buildHsPkSeries({ rows: [makeRow('A', 3, 1)], ...args })
    const json = JSON.stringify(result)
    expect(json).not.toMatch(/Headshot kills\/partie/)
    expect(json).not.toMatch(/Perfect kills\/partie/)
  })
})
