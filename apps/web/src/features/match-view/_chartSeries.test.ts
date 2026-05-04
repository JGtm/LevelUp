import { describe, expect, it } from 'vitest'

import {
  TUG_OF_WAR_LABELS,
  allPlayersFragDiffSeries,
  antagonistStackedSeries,
  cadenceSeriesWithGamertags,
  formatBinSeconds,
  tugOfWarStackedSeries,
} from './_chartSeries'
import type {
  MatchHighlightEvent,
  MatchKillerVictimPair,
  MatchScoreboardRow,
  MatchViewCadence,
} from '@/lib/api/types'

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

describe('antagonistStackedSeries', () => {
  it('retourne [] si aucune paire', () => {
    expect(antagonistStackedSeries([])).toEqual([])
  })
  it('agrège par tueur, ordonné par total décroissant', () => {
    const pairs: MatchKillerVictimPair[] = [
      { killer_xuid: 'X_A', killer_gamertag: 'Alice', victim_xuid: 'X_B', victim_gamertag: 'Bob', kill_count: 3 },
      { killer_xuid: 'X_A', killer_gamertag: 'Alice', victim_xuid: 'X_C', victim_gamertag: 'Charlie', kill_count: 1 },
      { killer_xuid: 'X_B', killer_gamertag: 'Bob', victim_xuid: 'X_A', victim_gamertag: 'Alice', kill_count: 2 },
    ]
    const series = antagonistStackedSeries(pairs)
    expect(series).toHaveLength(1)
    const dps = series[0].datapoints
    expect(dps).toHaveLength(2)
    // Alice (4 kills) en tête, Bob (2 kills) ensuite
    expect(dps[0].category).toBe('Alice')
    expect(dps[0].components).toEqual({ Bob: 3, Charlie: 1 })
    expect(dps[1].category).toBe('Bob')
    expect(dps[1].components).toEqual({ Alice: 2 })
  })
  it('utilise xuid si gamertag manque', () => {
    const pairs: MatchKillerVictimPair[] = [
      { killer_xuid: 'X1', killer_gamertag: '', victim_xuid: 'X2', victim_gamertag: '', kill_count: 1 },
    ]
    const series = antagonistStackedSeries(pairs)
    expect(series[0].datapoints[0].category).toBe('X1')
    expect(series[0].datapoints[0].components).toEqual({ X2: 1 })
  })
})

describe('allPlayersFragDiffSeries', () => {
  const scoreboard: MatchScoreboardRow[] = [
    { xuid: 'X1', gamertag: 'Alice', is_me: true } as MatchScoreboardRow,
    { xuid: 'X2', gamertag: 'Bob', is_me: false } as MatchScoreboardRow,
  ]

  it('retourne [] si pas d\'events', () => {
    expect(allPlayersFragDiffSeries([], scoreboard, 'X1')).toEqual([])
  })

  it('calcule cumulatif kill +1 / death -1, ordre joueur principal en premier', () => {
    const events: MatchHighlightEvent[] = [
      { event_type: 'kill', event_time_ms: 1000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'death', event_time_ms: 2000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 3000, actor_xuid: 'X2', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 4000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 5000, actor_xuid: 'X2', target_xuid: null, weapon_id: null },
    ]
    const series = allPlayersFragDiffSeries(events, scoreboard, 'X1')
    expect(series).toHaveLength(2)
    // X1 (joueur principal) en premier
    expect(series[0].meta?.gamertag).toBe('Alice')
    expect(series[0].datapoints).toEqual([
      { x: 0, y: 0 }, // point initial
      { x: 1, y: 1 }, // kill → +1
      { x: 2, y: 0 }, // death → 0
      { x: 4, y: 1 }, // kill → +1
    ])
    // X2 ensuite
    expect(series[1].meta?.gamertag).toBe('Bob')
    expect(series[1].datapoints).toEqual([
      { x: 0, y: 0 },
      { x: 3, y: 1 },
      { x: 5, y: 2 },
    ])
  })

  it('ignore les events sans time_ms ou actor_xuid', () => {
    const events: MatchHighlightEvent[] = [
      { event_type: 'kill', event_time_ms: 1000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: null, actor_xuid: 'X2', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 2000, actor_xuid: null, target_xuid: null, weapon_id: null },
    ]
    const series = allPlayersFragDiffSeries(events, scoreboard, null)
    expect(series).toHaveLength(1)
    expect(series[0].datapoints).toEqual([
      { x: 0, y: 0 },
      { x: 1, y: 1 },
    ])
  })

  it('ignore les events autres que kill/death', () => {
    const events: MatchHighlightEvent[] = [
      { event_type: 'kill', event_time_ms: 1000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'medal', event_time_ms: 1500, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'death', event_time_ms: 2000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
    ]
    const series = allPlayersFragDiffSeries(events, scoreboard, 'X1')
    expect(series[0].datapoints).toEqual([
      { x: 0, y: 0 },
      { x: 1, y: 1 },
      { x: 2, y: 0 },
    ])
  })
})
