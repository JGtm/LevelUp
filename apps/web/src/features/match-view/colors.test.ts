import { describe, expect, it } from 'vitest'

import { buildMatchPlayerColors, buildXUIDToGamertagMap } from './colors'
import type {
  MatchKillerVictimPair,
  MatchRosterRow,
  MatchScoreboardRow,
} from '@/lib/api/types'

describe('buildXUIDToGamertagMap', () => {
  const scoreboard: MatchScoreboardRow[] = [
    { xuid: 'X1', gamertag: 'Alice' } as MatchScoreboardRow,
    { xuid: 'X2', gamertag: 'Bob' } as MatchScoreboardRow,
    // Joueur sans gamertag — doit être ignoré (priorité aux sources résolues)
    { xuid: 'X3', gamertag: '' } as MatchScoreboardRow,
  ]

  it('cumule scoreboard + roster + kvPairs', () => {
    const roster: MatchRosterRow[] = [
      // bot non présent au scoreboard
      { xuid: 'BOT1', gamertag: '343 Recruit', is_bot: true } as MatchRosterRow,
      // duplicat — Alice est déjà dans le scoreboard, ne doit pas être écrasé
      { xuid: 'X1', gamertag: 'Alice (alias roster)' } as MatchRosterRow,
    ]
    const pairs: MatchKillerVictimPair[] = [
      // X3 (vide au scoreboard) résolu via kvPair
      {
        killer_xuid: 'X3',
        killer_gamertag: 'Charlie',
        victim_xuid: 'X1',
        victim_gamertag: 'Alice',
        kill_count: 2,
      },
      // Joueur uniquement dans les events/kvPairs (pas au scoreboard ni roster)
      {
        killer_xuid: 'X4',
        killer_gamertag: 'Diana',
        victim_xuid: 'X2',
        victim_gamertag: 'Bob',
        kill_count: 1,
      },
    ]
    const map = buildXUIDToGamertagMap(scoreboard, pairs, roster)
    // priorité scoreboard
    expect(map.get('X1')).toBe('Alice')
    expect(map.get('X2')).toBe('Bob')
    // X3 non résolu au scoreboard → kvPair gagne
    expect(map.get('X3')).toBe('Charlie')
    // X4 absent du scoreboard et roster → kvPair gagne
    expect(map.get('X4')).toBe('Diana')
    // bot du roster
    expect(map.get('BOT1')).toBe('343 Recruit')
  })

  it('fonctionne sans roster ni kvPairs', () => {
    const map = buildXUIDToGamertagMap(scoreboard)
    expect(map.get('X1')).toBe('Alice')
    expect(map.get('X3')).toBeUndefined()
  })

  it('ignore les entrées sans xuid ou gamertag', () => {
    const pairs: MatchKillerVictimPair[] = [
      {
        killer_xuid: '',
        killer_gamertag: 'Anon',
        victim_xuid: 'X9',
        victim_gamertag: '',
        kill_count: 1,
      },
    ]
    const map = buildXUIDToGamertagMap([], pairs)
    expect(map.size).toBe(0)
  })
})

describe('buildMatchPlayerColors — équipes', () => {
  // 4 alliés (team A : main + 3 alliés) vs 4 adversaires (team B)
  const scoreboard: MatchScoreboardRow[] = [
    { xuid: 'ME', gamertag: 'Me', team_side: 'A', is_me: true } as MatchScoreboardRow,
    { xuid: 'A1', gamertag: 'Ally1', team_side: 'A', is_me: false } as MatchScoreboardRow,
    { xuid: 'A2', gamertag: 'Ally2', team_side: 'A', is_me: false } as MatchScoreboardRow,
    { xuid: 'A3', gamertag: 'Ally3', team_side: 'A', is_me: false } as MatchScoreboardRow,
    { xuid: 'E1', gamertag: 'Enemy1', team_side: 'B', is_me: false } as MatchScoreboardRow,
    { xuid: 'E2', gamertag: 'Enemy2', team_side: 'B', is_me: false } as MatchScoreboardRow,
    { xuid: 'E3', gamertag: 'Enemy3', team_side: 'B', is_me: false } as MatchScoreboardRow,
    { xuid: 'E4', gamertag: 'Enemy4', team_side: 'B', is_me: false } as MatchScoreboardRow,
  ]

  it('main player → squad-player-1 quel que soit son team_side', () => {
    const colors = buildMatchPlayerColors(scoreboard, 'ME')
    expect(colors.tokenByXUID.get('ME')).toBe('squad-player-1')
  })

  it('alliés non-amis → palette cool « other » (pas les tokens squad)', () => {
    const colors = buildMatchPlayerColors(scoreboard, 'ME')
    // Sans amis, les 3 alliés non-main reçoivent perf-tier-2,
    // narrative-encounter-ally-plus, narrative-encounter-recrue
    expect(colors.tokenByXUID.get('A1')).toBe('perf-tier-2')
    expect(colors.tokenByXUID.get('A2')).toBe('narrative-encounter-ally-plus')
    expect(colors.tokenByXUID.get('A3')).toBe('narrative-encounter-recrue')
  })

  it('ennemis → palette warm', () => {
    const colors = buildMatchPlayerColors(scoreboard, 'ME')
    // Ordre ENEMY_TOKENS révisé dans 5633afce (palette antagonistes 11 tokens
    // distincts) : outcome-loss → narrative-humiliation → narrative-debacle →
    // narrative-contre-remontada (perf-tier-5 retiré car doublon hex avec
    // outcome-loss).
    expect(colors.tokenByXUID.get('E1')).toBe('outcome-loss')
    expect(colors.tokenByXUID.get('E2')).toBe('narrative-humiliation')
    expect(colors.tokenByXUID.get('E3')).toBe('narrative-debacle')
    expect(colors.tokenByXUID.get('E4')).toBe('narrative-contre-remontada')
  })

  it('amis alliés → tokens squad (squad-player-2 / -3 / -4)', () => {
    const colors = buildMatchPlayerColors(scoreboard, 'ME', ['Ally1', 'Ally3'])
    // Ally1 et Ally3 sont amis alliés → tokens squad cyclés dans l'ordre rencontré
    expect(colors.tokenByXUID.get('A1')).toBe('squad-player-2')
    expect(colors.tokenByXUID.get('A3')).toBe('squad-player-3')
    // Ally2 (non-ami) garde la palette cool « other »
    expect(colors.tokenByXUID.get('A2')).toBe('perf-tier-2')
  })

  it("amis adverses NE prennent PAS la couleur squad — la distinction d'équipe l'emporte", () => {
    const colors = buildMatchPlayerColors(scoreboard, 'ME', ['Enemy1'])
    expect(colors.tokenByXUID.get('E1')).toBe('outcome-loss') // palette warm, pas squad-player-2
  })

  it('matching gamertag amis insensible à la casse / espaces', () => {
    const colors = buildMatchPlayerColors(scoreboard, 'ME', ['  ally1  ', 'ALLY3'])
    expect(colors.tokenByXUID.get('A1')).toBe('squad-player-2')
    expect(colors.tokenByXUID.get('A3')).toBe('squad-player-3')
  })
})
