import { describe, expect, it } from 'vitest'

import {
  allPlayersFragDiffSeries,
  antagonistStackedSeries,
  formatBinSeconds,
} from './_chartSeries'
import type {
  MatchHighlightEvent,
  MatchKillerVictimPair,
} from '@/lib/api/types'

describe('formatBinSeconds', () => {
  // Format "MmSSs" depuis la refonte onglet Combat (97199831 / 5633afce) —
  // remplace l'ancien "M:SS" pour éviter la confusion avec les notations CSR.
  it('formate sous 60s', () => {
    expect(formatBinSeconds(45)).toBe('0m45s')
  })
  it('formate au-dessus de 60s', () => {
    expect(formatBinSeconds(75)).toBe('1m15s')
    expect(formatBinSeconds(125)).toBe('2m05s')
  })
  it('formate 0', () => {
    expect(formatBinSeconds(0)).toBe('0m00s')
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
  it('utilise un libellé masqué (jamais le xuid brut) si gamertag manque', () => {
    const pairs: MatchKillerVictimPair[] = [
      { killer_xuid: 'X1', killer_gamertag: '', victim_xuid: 'X2', victim_gamertag: '', kill_count: 1 },
    ]
    const series = antagonistStackedSeries(pairs)
    expect(series[0].datapoints[0].category).toBe('Joueur X1')
    expect(series[0].datapoints[0].components).toEqual({ 'Joueur X2': 1 })
  })
})

describe('allPlayersFragDiffSeries', () => {
  const xuidToGamertag = new Map<string, string>([
    ['X1', 'Alice'],
    ['X2', 'Bob'],
  ])

  it("retourne [] si pas d'events", () => {
    expect(allPlayersFragDiffSeries([], xuidToGamertag, 'X1')).toEqual([])
  })

  it('calcule cumulatif kill +1 / death -1, ordre joueur principal en premier', () => {
    const events: MatchHighlightEvent[] = [
      { event_type: 'kill', event_time_ms: 1000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'death', event_time_ms: 2000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 3000, actor_xuid: 'X2', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 4000, actor_xuid: 'X1', target_xuid: null, weapon_id: null },
      { event_type: 'kill', event_time_ms: 5000, actor_xuid: 'X2', target_xuid: null, weapon_id: null },
    ]
    const series = allPlayersFragDiffSeries(events, xuidToGamertag, 'X1')
    expect(series).toHaveLength(2)
    expect(series[0].meta?.gamertag).toBe('Alice')
    expect(series[0].datapoints).toEqual([
      { x: 0, y: 0 },
      { x: 1, y: 1 },
      { x: 2, y: 0 },
      { x: 4, y: 1 },
    ])
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
    const series = allPlayersFragDiffSeries(events, xuidToGamertag, null)
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
    const series = allPlayersFragDiffSeries(events, xuidToGamertag, 'X1')
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
    const series = allPlayersFragDiffSeries(events, xuidToGamertag, 'X1', colorByXUID)
    expect(series[0].colorToken).toBe('compare-a')
    expect(series[1].colorToken).toBe('outcome-loss')
  })

  it("fallback `Joueur XXXX` quand l'xuid n'est pas dans la map", () => {
    const events: MatchHighlightEvent[] = [
      { event_type: 'kill', event_time_ms: 1000, actor_xuid: 'XX_unknown_aaaa', target_xuid: null, weapon_id: null },
    ]
    const series = allPlayersFragDiffSeries(events, new Map(), null)
    expect(series[0].meta?.gamertag).toBe('Joueur aaaa')
  })
})
