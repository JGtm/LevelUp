import { describe, expect, it } from 'vitest'

import {
  allPlayersFragDiffSeries,
  antagonistStackedSeries,
  cadenceTeamSeries,
  formatBinSeconds,
  kdCumulSeries,
  tugOfWarStackedSeries,
} from './_chartSeries'
import type {
  MatchHighlightEvent,
  MatchKDTimelinePoint,
  MatchKillerVictimPair,
  MatchScoreboardRow,
  MatchTugOfWarBin,
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

describe('kdCumulSeries', () => {
  const labels = { kills: 'Frags', deaths: 'Morts' }

  it('retourne [] si aucun point', () => {
    expect(kdCumulSeries([], labels)).toEqual([])
  })

  it('produit 2 séries (frags / morts) avec point initial (0, 0) et tokens couleur', () => {
    const points: MatchKDTimelinePoint[] = [
      { time_seconds: 30, kills: 1, deaths: 0 },
      { time_seconds: 90, kills: 1, deaths: 1 },
      { time_seconds: 150, kills: 2, deaths: 1 },
    ]
    const series = kdCumulSeries(points, labels)
    expect(series).toHaveLength(2)
    expect(series[0].meta?.gamertag).toBe('Frags')
    expect(series[0].colorToken).toBe('compare-a')
    expect(series[0].datapoints).toEqual([
      { x: 0, y: 0 },
      { x: 30, y: 1 },
      { x: 150, y: 2 },
    ])
    expect(series[1].meta?.gamertag).toBe('Morts')
    expect(series[1].colorToken).toBe('outcome-loss')
    expect(series[1].datapoints).toEqual([
      { x: 0, y: 0 },
      { x: 90, y: 1 },
    ])
  })

  it('trie par time_seconds croissant avant déduplication', () => {
    const points: MatchKDTimelinePoint[] = [
      { time_seconds: 90, kills: 1, deaths: 1 },
      { time_seconds: 30, kills: 1, deaths: 0 },
    ]
    const series = kdCumulSeries(points, labels)
    expect(series[0].datapoints.map((p) => p.x)).toEqual([0, 30])
    expect(series[1].datapoints.map((p) => p.x)).toEqual([0, 90])
  })
})

describe('tugOfWarStackedSeries', () => {
  const labels = { team: 'Mon équipe', enemy: 'Adversaires' }

  it('retourne [] si aucun bin', () => {
    expect(tugOfWarStackedSeries([], labels)).toEqual([])
  })

  it('catégorie = mm:ss du milieu, components team/enemy', () => {
    const bins: MatchTugOfWarBin[] = [
      { bin_start: 0, bin_end: 60, team_kills: 3, enemy_kills: 1, net_kills: 2 },
      { bin_start: 60, bin_end: 120, team_kills: 2, enemy_kills: 4, net_kills: -2 },
    ]
    const series = tugOfWarStackedSeries(bins, labels)
    expect(series).toHaveLength(1)
    expect(series[0].datapoints).toEqual([
      { category: '0:30', components: { 'Mon équipe': 3, Adversaires: 1 } },
      { category: '1:30', components: { 'Mon équipe': 2, Adversaires: 4 } },
    ])
  })
})

describe('cadenceTeamSeries', () => {
  const labels = { team: 'Mon équipe', enemy: 'Adversaires' }
  const scoreboard = (rows: Array<Partial<MatchScoreboardRow>>): MatchScoreboardRow[] =>
    rows.map(
      (r) =>
        ({
          xuid: r.xuid ?? '',
          gamertag: r.gamertag ?? '',
          team_side: r.team_side ?? null,
          is_me: r.is_me ?? false,
          rank: null,
          score: null,
          kills: null,
          deaths: null,
          assists: null,
          shots_fired: null,
          shots_hit: null,
          shots_accuracy: null,
          damage_dealt: null,
          damage_taken: null,
          average_life: null,
          headshot_kills: null,
          max_killing_spree: null,
          perfect_kills: null,
          melee_kills: null,
          power_weapon_kills: null,
          outcome_label: '',
        }) as MatchScoreboardRow,
    )

  it('retourne [] si cadence absente ou vide', () => {
    expect(cadenceTeamSeries(null, [], null, labels)).toEqual([])
    expect(cadenceTeamSeries(undefined, [], null, labels)).toEqual([])
    const emptyCadence: MatchViewCadence = { key: 'k', datapoints: [] }
    expect(cadenceTeamSeries(emptyCadence, [], null, labels)).toEqual([])
  })

  it('agrège par équipe (allié vs adverse) avec catégorie au milieu de phase', () => {
    const sb = scoreboard([
      { xuid: 'me', team_side: 't0', is_me: true },
      { xuid: 'ally1', team_side: 't0' },
      { xuid: 'enemy1', team_side: 't1' },
      { xuid: 'enemy2', team_side: 't1' },
    ])
    const cadence: MatchViewCadence = {
      key: 'cadence',
      datapoints: [
        { category: 'phase_00', components: { me: 1, ally1: 1, enemy1: 2, enemy2: 0 } },
        { category: 'phase_01', components: { me: 0, ally1: 0, enemy1: 1, enemy2: 1 } },
      ],
      meta: { phase_seconds: 60 },
    }
    const series = cadenceTeamSeries(cadence, sb, 'me', labels)
    expect(series).toHaveLength(1)
    expect(series[0].datapoints).toEqual([
      { category: '0:30', components: { 'Mon équipe': 2, Adversaires: 2 } },
      { category: '1:30', components: { 'Mon équipe': 0, Adversaires: 2 } },
    ])
  })

  it('utilise phase_seconds=60 par défaut si meta absent', () => {
    const sb = scoreboard([
      { xuid: 'me', team_side: 't0', is_me: true },
      { xuid: 'enemy', team_side: 't1' },
    ])
    const cadence: MatchViewCadence = {
      key: 'cadence',
      datapoints: [{ category: 'phase_00', components: { me: 2, enemy: 1 } }],
    }
    const series = cadenceTeamSeries(cadence, sb, 'me', labels)
    expect(series[0].datapoints[0].category).toBe('0:30')
  })

  it("traite les xuids inconnus comme adverses si meXUID a un team_side", () => {
    const sb = scoreboard([{ xuid: 'me', team_side: 't0', is_me: true }])
    const cadence: MatchViewCadence = {
      key: 'cadence',
      datapoints: [{ category: 'phase_00', components: { me: 1, ghost: 3 } }],
      meta: { phase_seconds: 60 },
    }
    const series = cadenceTeamSeries(cadence, sb, 'me', labels)
    expect(series[0].datapoints[0].components).toEqual({
      'Mon équipe': 1,
      Adversaires: 3,
    })
  })
})

