/**
 * Tests — TimeseriesSkillProgression.buildProgressionSeries : mapping des valeurs
 * par index de match, ruptures de saison, isolation des placements.
 * (Aucun rendu ECharts — on teste la fonction pure, pas le canvas. Le cadrage Y
 * frameToData est testé dans @/lib/charts/skillTierBands.test.ts.)
 */
import { describe, it, expect } from 'vitest'
import { buildProgressionSeries } from './progressionSeries'
import type { TimeseriesMatchRow } from '@/lib/api/types'

function makeRow({ index, ...rest }: Partial<TimeseriesMatchRow> & { index: number }): TimeseriesMatchRow {
  return {
    match_id: `m${index}`,
    index,
    start_time: `2025-01-${String(index + 1).padStart(2, '0')}T10:00:00Z`,
    kills: 0,
    deaths: 0,
    assists: 0,
    accuracy: null,
    outcome: null,
    personal_score: null,
    damage_dealt: null,
    damage_taken: null,
    perf_score: null,
    rank: null,
    playlist_name: '',
    time_played_seconds: null,
    ...rest,
  }
}

describe('buildProgressionSeries', () => {
  it('place les valeurs à l\'index de match (null hors matchs notés)', () => {
    const rows = [
      makeRow({ index: 0, skill_rating_value: 1500, skill_rating_type: 'lusr', skill_playlist_group: 'ranked' }),
      makeRow({ index: 1 }), // match non noté
      makeRow({ index: 2, skill_rating_value: 1520, skill_rating_type: 'lusr', skill_playlist_group: 'ranked' }),
    ]
    const series = buildProgressionSeries(rows, 'fr')
    expect(series).toHaveLength(1)
    expect(series[0].values).toEqual([1500, null, 1520])
    expect(series[0].ratingType).toBe('lusr')
  })

  it('sépare CSR et LUSR en deux séries distinctes', () => {
    const rows = [
      makeRow({ index: 0, skill_rating_value: 1500, skill_rating_type: 'lusr', skill_playlist_group: 'open' }),
      makeRow({ index: 1, skill_rating_value: 1200, skill_rating_type: 'csr', skill_playlist_group: 'arena' }),
    ]
    expect(buildProgressionSeries(rows, 'fr')).toHaveLength(2)
  })

  it('marque une rupture de saison à l\'index du nouveau match', () => {
    const rows = [
      makeRow({ index: 0, skill_rating_value: 1500, skill_rating_type: 'lusr', skill_playlist_group: 'ranked', skill_season_id: 'S1' }),
      makeRow({ index: 1, skill_rating_value: 1480, skill_rating_type: 'lusr', skill_playlist_group: 'ranked', skill_season_id: 'S2' }),
    ]
    expect(buildProgressionSeries(rows, 'fr')[0].seasonBreaks).toEqual([1])
  })

  it('isole les matchs de placement en points scatter (midpoint), hors de la ligne', () => {
    const rows = [
      makeRow({ index: 0, skill_rating_value: 1400, skill_rating_type: 'lusr', skill_playlist_group: 'ranked', skill_measurement_remaining: 0 }),
      makeRow({ index: 1, skill_rating_value: 0, skill_rating_type: 'lusr', skill_playlist_group: 'ranked', skill_measurement_remaining: 3 }),
      makeRow({ index: 2, skill_rating_value: 1600, skill_rating_type: 'lusr', skill_playlist_group: 'ranked', skill_measurement_remaining: 0 }),
    ]
    const series = buildProgressionSeries(rows, 'fr')
    expect(series[0].values).toEqual([1400, null, 1600]) // placement laissé null dans la ligne
    expect(series[0].placementPoints).toEqual([[1, 1500]]) // midpoint des valeurs réelles
  })

  it('retourne [] si aucun match noté', () => {
    expect(buildProgressionSeries([makeRow({ index: 0 }), makeRow({ index: 1 })], 'fr')).toEqual([])
  })
})
