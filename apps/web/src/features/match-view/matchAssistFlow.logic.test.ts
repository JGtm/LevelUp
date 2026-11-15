/**
 * matchAssistFlow.logic.test — la projection des paires d'assistances d'un match vers le
 * diagramme de flux : ennemis en tête, total décroissant, bots nommés par le film, repli masqué
 * d'un finisseur sans gamertag, part moyenne et assistances « plus de dégâts » conservées.
 */
import { describe, expect, it } from 'vitest'

import type { MatchAssistPair, MatchScoreboardRow } from '@/lib/api/types'

import { matchAssistFlow, matchAssistantOrder, matchFlowPairs } from './matchAssistFlow.logic'

const pair = (over: Partial<MatchAssistPair> = {}): MatchAssistPair => ({
  assist_xuid: 'X_A',
  assist_gamertag: 'Alice',
  killer_xuid: 'X_B',
  killer_gamertag: 'Bob',
  assist_count: 2,
  stolen_count: 0,
  ...over,
})

const sb = (rows: Array<[string, string]>) =>
  rows.map(([xuid, team_side]) => ({ xuid, team_side })) as unknown as MatchScoreboardRow[]

describe('matchAssistFlow', () => {
  it('sans paire : aucun nœud ni ruban', () => {
    expect(matchAssistFlow([]).links).toEqual([])
    expect(matchAssistFlow([]).nodes).toEqual([])
  })

  it('agrège par ASSISTANT (et non par finisseur), total décroissant', () => {
    const flow = matchAssistFlow([
      pair({ assist_xuid: 'X_A', assist_gamertag: 'Alice', killer_xuid: 'X_B', killer_gamertag: 'Bob', assist_count: 1 }),
      pair({ assist_xuid: 'X_C', assist_gamertag: 'Carol', killer_xuid: 'X_B', killer_gamertag: 'Bob', assist_count: 4 }),
      pair({ assist_xuid: 'X_A', assist_gamertag: 'Alice', killer_xuid: 'X_D', killer_gamertag: 'Dave', assist_count: 2 }),
    ])
    const left = flow.nodes.filter((n) => n.side === 'assistant')
    // Carol (4) devant Alice (1 + 2 = 3).
    expect(left.map((n) => [n.name, n.total])).toEqual([['Carol', 4], ['Alice', 3]])
    const right = flow.nodes.filter((n) => n.side === 'finisher')
    // Bob a conclu 5 frags sur une assistance, Dave 2.
    expect(right.map((n) => [n.name, n.total])).toEqual([['Bob', 5], ['Dave', 2]])
  })

  it('met les ENNEMIS en tête, comme le graphe des antagonistes', () => {
    const scoreboard = sb([['X_ME', 'ally'], ['X_ALLY', 'ally'], ['X_FOE', 'enemy']])
    const flow = matchAssistFlow(
      [
        pair({ assist_xuid: 'X_ALLY', assist_gamertag: 'Ally', assist_count: 9 }),
        pair({ assist_xuid: 'X_FOE', assist_gamertag: 'Foe', assist_count: 1 }),
      ],
      scoreboard,
      'X_ME',
    )
    expect(flow.nodes.filter((n) => n.side === 'assistant').map((n) => n.name)).toEqual(['Foe', 'Ally'])
  })

  it('compte les BOTS : assistant sans xuid nommé par le film, rangé au camp des finisseurs qu’il assiste', () => {
    const scoreboard = sb([['X_ME', 'ally'], ['X_FOE', 'enemy']])
    const flow = matchAssistFlow(
      [
        pair({ assist_xuid: '', assist_gamertag: '343 Oscar [bot]', killer_xuid: 'X_ME', killer_gamertag: 'Moi', assist_count: 5 }),
        pair({ assist_xuid: '', assist_gamertag: '343 Ritzy [bot]', killer_xuid: 'X_FOE', killer_gamertag: 'Foe', assist_count: 1 }),
        pair({ assist_xuid: 'X_ME', assist_gamertag: 'Moi', killer_xuid: '', killer_gamertag: '343 Cosmo [bot]', assist_count: 2 }),
      ],
      scoreboard,
      'X_ME',
    )
    // Ritzy assiste un ennemi : en tête ; puis les alliés, total décroissant (le suffixe de bot ne s'écrit pas).
    expect(flow.nodes.filter((n) => n.side === 'assistant').map((n) => n.name)).toEqual(['343 Ritzy', '343 Oscar', 'Moi'])
    expect(flow.nodes.filter((n) => n.side === 'finisher').map((n) => n.name)).toContain('343 Cosmo')
  })

  it('deux assistants sans xuid de même nom de film restent un seul joueur ; un xuid et un bot restent distincts', () => {
    const flow = matchAssistFlow([
      pair({ assist_xuid: '', assist_gamertag: 'Bot', assist_count: 1 }),
      pair({ assist_xuid: '', assist_gamertag: 'Bot', killer_xuid: 'X_C', killer_gamertag: 'Carl', assist_count: 2 }),
      pair({ assist_xuid: 'X_BOT', assist_gamertag: 'Bot', assist_count: 3 }),
    ])
    expect(flow.nodes.filter((n) => n.side === 'assistant')).toHaveLength(2)
  })

  it('replie un finisseur sans gamertag sur le nom masqué, jamais sur le xuid brut', () => {
    const flow = matchAssistFlow([pair({ killer_gamertag: undefined, killer_xuid: 'X_SECRET1234' })])
    const finisher = flow.nodes.find((n) => n.side === 'finisher')!
    expect(finisher.name).not.toBe('X_SECRET1234')
    expect(finisher.name).toContain('1234')
  })

  it('aucune part (pas de total de référence sur la page Match)', () => {
    expect(matchAssistFlow([pair()]).links.every((l) => l.share === null)).toBe(true)
  })
})

describe('matchFlowPairs', () => {
  it('conserve les assistances « plus de dégâts » et la part moyenne SANS la plafonner ni la fabriquer', () => {
    const [a, b] = matchFlowPairs([
      pair({ stolen_count: 2, avg_assist_pct: 228 }),
      pair({ assist_gamertag: 'Carol', assist_xuid: 'X_C' }), // pas de part servie : jamais « 0 % »
    ])
    expect(a).toMatchObject({ stolen: 2, avgPct: 228 })
    expect(b.avgPct).toBeUndefined()
  })
})

describe('matchAssistantOrder', () => {
  it('sans meXUID ni scoreboard, nul n’est ennemi : l’ordre est celui du total', () => {
    expect(
      matchAssistantOrder([
        pair({ assist_xuid: 'X_A', assist_count: 1 }),
        pair({ assist_xuid: 'X_B', assist_gamertag: 'B', assist_count: 5 }),
      ]),
    ).toEqual(['X_B', 'X_A'])
  })
})
