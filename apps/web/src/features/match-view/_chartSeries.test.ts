import { describe, expect, it } from 'vitest'

import {
  allPlayersFragDiffSeries,
  antagonistStackedSeries,
  cadenceSeriesFromEvents,
  formatBinSeconds,
} from './_chartSeries'
import type {
  MatchHighlightEvent,
  MatchKillerVictimPair,
  MatchScoreboardRow,
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

  it("retourne [] si pas d'events", () => {
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

  it("propage le colorToken sur chaque série quand colorByXUID est fourni", () => {
    const events: MatchHighlightEvent[] = [
      { event_type: 'kill', event_time_ms: 1000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 2000, actor_xuid: 'X2', target_xuid: null, weapon_id: null },
    ]
    const colorByXUID = new Map<string, 'compare-a' | 'outcome-loss'>([
      ['X1', 'compare-a'],
      ['X2', 'outcome-loss'],
    ])
    const series = allPlayersFragDiffSeries(events, scoreboard, 'X1', colorByXUID)
    expect(series[0].colorToken).toBe('compare-a')
    expect(series[1].colorToken).toBe('outcome-loss')
  })

  it("fallback `Joueur XXXX` quand l'xuid n'est pas dans le scoreboard", () => {
    const events: MatchHighlightEvent[] = [
      { event_type: 'kill', event_time_ms: 1000, actor_xuid: 'XX_unknown_aaaa', target_xuid: null, weapon_id: null },
    ]
    const series = allPlayersFragDiffSeries(events, [], null)
    expect(series[0].meta?.gamertag).toBe('Joueur aaaa')
  })
})

describe('cadenceSeriesFromEvents', () => {
  const scoreboard: MatchScoreboardRow[] = [
    { xuid: 'X1', gamertag: 'Alice', is_me: true } as MatchScoreboardRow,
    { xuid: 'X2', gamertag: 'Bob', is_me: false } as MatchScoreboardRow,
  ]

  it('retourne [] sans events ou phase invalide', () => {
    expect(cadenceSeriesFromEvents([], scoreboard, 60)).toEqual([])
    const events: MatchHighlightEvent[] = [
      { event_type: 'kill', event_time_ms: 1000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
    ]
    expect(cadenceSeriesFromEvents(events, scoreboard, 0)).toEqual([])
  })

  it('agrège par phase de 60s, label m:ss, ne compte que kill', () => {
    const events: MatchHighlightEvent[] = [
      // phase 0 (0-60s)
      { event_type: 'kill', event_time_ms: 5_000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'death', event_time_ms: 30_000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      // phase 1 (60-120s)
      { event_type: 'kill', event_time_ms: 65_000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 90_000, actor_xuid: 'X2', target_xuid: null, weapon_id: null },
      // phase 2 (120-180s) — Alice tue 2x, Bob 1x
      { event_type: 'kill', event_time_ms: 125_000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 150_000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 175_000, actor_xuid: 'X2', target_xuid: null, weapon_id: null },
    ]
    const series = cadenceSeriesFromEvents(events, scoreboard, 60)
    expect(series).toHaveLength(1)
    const dps = series[0].datapoints
    expect(dps).toHaveLength(3)
    expect(dps[0].category).toBe('0:00')
    expect(dps[0].components).toEqual({ Alice: 1 })
    expect(dps[1].category).toBe('1:00')
    expect(dps[1].components).toEqual({ Alice: 1, Bob: 1 })
    expect(dps[2].category).toBe('2:00')
    expect(dps[2].components).toEqual({ Alice: 2, Bob: 1 })
  })

  it("crée des phases vides entre la première et la dernière phase active", () => {
    const events: MatchHighlightEvent[] = [
      { event_type: 'kill', event_time_ms: 5_000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 185_000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
    ]
    const series = cadenceSeriesFromEvents(events, scoreboard, 60)
    const dps = series[0].datapoints
    expect(dps.map((d) => d.category)).toEqual(['0:00', '1:00', '2:00', '3:00'])
    // phase 1 et 2 sont vides, phase 0 et 3 ont 1 kill chacun
    expect(dps[1].components).toEqual({})
    expect(dps[2].components).toEqual({})
  })
})
