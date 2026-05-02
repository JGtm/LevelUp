import { describe, expect, it } from 'vitest'

import {
  TUG_OF_WAR_LABELS,
  cadenceSeriesWithGamertags,
  formatBinSeconds,
  tugOfWarStackedSeries,
} from './_chartSeries'
import type { MatchScoreboardRow, MatchViewCadence } from '@/lib/api/types'

describe('formatBinSeconds', () => {
  it('formate sous 60s', () => {
    expect(formatBinSeconds(45)).toBe('0:45')
  })
  it('formate au-dessus de 60s', () => {
    expect(formatBinSeconds(75)).toBe('1:15')
    expect(formatBinSeconds(125)).toBe('2:05')
  })
  it('formate 0', () => {
    expect(formatBinSeconds(0)).toBe('0:00')
  })
})

describe('tugOfWarStackedSeries', () => {
  it('retourne [] sur entrée vide', () => {
    expect(tugOfWarStackedSeries([])).toEqual([])
  })
  it('produit des composants divergents (équipe positive, adverses négatives)', () => {
    const series = tugOfWarStackedSeries([
      { bin_start: 0, bin_end: 30, team_kills: 3, enemy_kills: 1, net_kills: 2 },
      { bin_start: 30, bin_end: 60, team_kills: 0, enemy_kills: 4, net_kills: -2 },
    ])
    expect(series).toHaveLength(1)
    const dps = series[0].datapoints
    expect(dps[0].category).toBe('0:00')
    expect(dps[0].components[TUG_OF_WAR_LABELS.team]).toBe(3)
    expect(dps[0].components[TUG_OF_WAR_LABELS.enemy]).toBe(-1)
    expect(dps[1].category).toBe('0:30')
    expect(dps[1].components[TUG_OF_WAR_LABELS.team]).toBe(0)
    expect(dps[1].components[TUG_OF_WAR_LABELS.enemy]).toBe(-4)
  })
})

describe('cadenceSeriesWithGamertags', () => {
  const scoreboard: MatchScoreboardRow[] = [
    { xuid: 'X1', gamertag: 'Alice', is_me: true } as MatchScoreboardRow,
    { xuid: 'X2', gamertag: 'Bob', is_me: false } as MatchScoreboardRow,
  ]
  it('retourne [] si cadence absente', () => {
    expect(cadenceSeriesWithGamertags(null, scoreboard)).toEqual([])
    expect(cadenceSeriesWithGamertags(undefined, scoreboard)).toEqual([])
  })
  it('retourne [] si datapoints vides', () => {
    const cad: MatchViewCadence = {
      key: 'k',
      datapoints: [],
    }
    expect(cadenceSeriesWithGamertags(cad, scoreboard)).toEqual([])
  })
  it('remappe les xuid en gamertags et drop les zéros', () => {
    const cad: MatchViewCadence = {
      key: 'cadence',
      label_key: 'cadence_lbl',
      datapoints: [
        { category: '0-60s', components: { X1: 2, X2: 0, X3: 1 } },
        { category: '60-120s', components: { X1: 0, X2: 3 } },
      ],
      meta: { phase_seconds: 60 },
    }
    const series = cadenceSeriesWithGamertags(cad, scoreboard)
    expect(series).toHaveLength(1)
    expect(series[0].key).toBe('cadence')
    expect(series[0].labelKey).toBe('cadence_lbl')
    expect(series[0].meta).toEqual({ phase_seconds: 60 })
    expect(series[0].datapoints[0].components).toEqual({ Alice: 2, X3: 1 })
    expect(series[0].datapoints[1].components).toEqual({ Bob: 3 })
  })
})
