import { describe, it, expect } from 'vitest'

import { extractCtfCaptures } from './_objectiveCaptures'
import type { MatchObjectiveEvent, MatchScoreboardRow } from '@/lib/api/types'

function sbRow(partial: Partial<MatchScoreboardRow>): MatchScoreboardRow {
  return {
    xuid: 'x',
    gamertag: 'GT',
    team_side: null,
    is_me: false,
    rank: null,
    score: null,
    kills: null,
    deaths: null,
    assists: null,
    shots_fired: null,
    shots_hit: null,
    accuracy: null,
    damage_dealt: null,
    damage_taken: null,
    average_life: null,
    headshot_kills: null,
    max_killing_spree: null,
    perfect_kills: null,
    power_weapon_kills: null,
    melee_kills: null,
    outcome_label: '',
    ...partial,
  } as MatchScoreboardRow
}

function flagCapture(partial: Partial<MatchObjectiveEvent>): MatchObjectiveEvent {
  return {
    matchId: 'm1',
    seq: 0,
    objectiveType: 'flag',
    eventType: 'capture',
    source: 'film',
    confidence: 'high',
    players: [],
    ...partial,
  } as MatchObjectiveEvent
}

const SCOREBOARD: MatchScoreboardRow[] = [
  sbRow({ xuid: 'me', gamertag: 'Me', team_side: '0', is_me: true }),
  sbRow({ xuid: 'ally1', gamertag: 'AllyOne', team_side: '0' }),
  sbRow({ xuid: 'enemy1', gamertag: 'EnemyOne', team_side: '1' }),
]

describe('extractCtfCaptures', () => {
  it('renvoie [] quand objectiveEvents est null/undefined/vide', () => {
    expect(extractCtfCaptures(null, SCOREBOARD, 'me')).toEqual([])
    expect(extractCtfCaptures(undefined, SCOREBOARD, 'me')).toEqual([])
    expect(extractCtfCaptures([], SCOREBOARD, 'me')).toEqual([])
  })

  it('renvoie [] quand aucun event flag/capture (mode non-CTF)', () => {
    const events: MatchObjectiveEvent[] = [
      flagCapture({ objectiveType: 'oddball', eventType: 'score', teamId: 0, timeMs: 1000, players: [{ xuid: 'me', role: 'scorer' }] }),
      flagCapture({ objectiveType: 'flag', eventType: 'pickup', teamId: 0, timeMs: 2000, players: [{ xuid: 'me', role: 'carrier' }] }),
    ]
    expect(extractCtfCaptures(events, SCOREBOARD, 'me')).toEqual([])
  })

  it('mappe teamId == allyTeam → ally:true, sinon ally:false', () => {
    const events: MatchObjectiveEvent[] = [
      flagCapture({ teamId: 0, timeMs: 5000, players: [{ xuid: 'ally1', role: 'scorer' }] }),
      flagCapture({ teamId: 1, timeMs: 9000, players: [{ xuid: 'enemy1', role: 'scorer' }] }),
    ]
    const out = extractCtfCaptures(events, SCOREBOARD, 'me')
    expect(out).toHaveLength(2)
    expect(out[0]).toMatchObject({ tMs: 5000, ally: true, scorer: 'AllyOne' })
    expect(out[1]).toMatchObject({ tMs: 9000, ally: false, scorer: 'EnemyOne' })
  })

  it('résout le scorer en gamertag via le scoreboard', () => {
    const events = [flagCapture({ teamId: 0, timeMs: 1000, players: [{ xuid: 'me', role: 'scorer' }] })]
    const out = extractCtfCaptures(events, SCOREBOARD, 'me')
    expect(out[0].scorer).toBe('Me')
  })

  it('fallback xuid masqué quand le scorer est hors scoreboard', () => {
    const events = [flagCapture({ teamId: 0, timeMs: 1000, players: [{ xuid: '2533274801234567', role: 'scorer' }] })]
    const out = extractCtfCaptures(events, SCOREBOARD, 'me')
    // displayPlayerName masque un xuid brut (>=15 chiffres) → "Joueur 4567"
    expect(out[0].scorer).toBe('Joueur 4567')
  })

  it('ignore les events sans timeMs ou sans teamId', () => {
    const events: MatchObjectiveEvent[] = [
      flagCapture({ teamId: 0, players: [{ xuid: 'me', role: 'scorer' }] }), // pas de timeMs
      flagCapture({ timeMs: 3000, players: [{ xuid: 'me', role: 'scorer' }] }), // pas de teamId
      flagCapture({ teamId: 1, timeMs: 4000, players: [{ xuid: 'enemy1', role: 'scorer' }] }),
    ]
    const out = extractCtfCaptures(events, SCOREBOARD, 'me')
    expect(out).toHaveLength(1)
    expect(out[0].tMs).toBe(4000)
  })

  it('trie les captures par temps croissant', () => {
    const events: MatchObjectiveEvent[] = [
      flagCapture({ teamId: 0, timeMs: 9000, players: [{ xuid: 'me', role: 'scorer' }] }),
      flagCapture({ teamId: 1, timeMs: 3000, players: [{ xuid: 'enemy1', role: 'scorer' }] }),
      flagCapture({ teamId: 0, timeMs: 6000, players: [{ xuid: 'ally1', role: 'scorer' }] }),
    ]
    const out = extractCtfCaptures(events, SCOREBOARD, 'me')
    expect(out.map((c) => c.tMs)).toEqual([3000, 6000, 9000])
  })

  it('ally:false quand le joueur courant est absent (allyTeam null)', () => {
    const events = [flagCapture({ teamId: 0, timeMs: 1000, players: [{ xuid: 'ally1', role: 'scorer' }] })]
    const out = extractCtfCaptures(events, SCOREBOARD, null)
    expect(out[0].ally).toBe(false)
  })
})
