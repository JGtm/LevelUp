/**
 * squadAssistPairs.logic.test — la projection des paires d'« Appui » vers le diagramme de flux :
 * identité par gamertag, ordre du roster, parts sur le total serveur.
 */
import { describe, expect, it } from 'vitest'

import type { SquadAssistPair, SquadAssistPairs } from '@/lib/api/types'

import { squadAssistFlow, squadFlowPairs } from './squadAssistPairs.logic'

function paire(assist: string, killer: string, count: number, stolen = 0): SquadAssistPair {
  return {
    assist_xuid: `x_${assist}`,
    assist_gamertag: assist,
    killer_xuid: `x_${killer}`,
    killer_gamertag: killer,
    assist_count: count,
    stolen_count: stolen,
  }
}

const PAIRES = [paire('Bob', 'Alice', 30, 12), paire('Alice', 'Bob', 20), paire('Alice', 'Carol', 10, 4)]

const BLOC: SquadAssistPairs = {
  matches_measured: 8,
  matches_total: 10,
  total_assists: 100,
  pairs: PAIRES,
}

describe('squadFlowPairs', () => {
  it('l’identité d’un joueur est son gamertag, des deux côtés', () => {
    expect(squadFlowPairs(BLOC)[0]).toEqual({
      assistantKey: 'Bob',
      assistantName: 'Bob',
      finisherKey: 'Alice',
      finisherName: 'Alice',
      count: 30,
      stolen: 12,
    })
  })

  it('`pairs: null` (tableau nullable du contrat) vaut une liste vide', () => {
    expect(squadFlowPairs({ ...BLOC, pairs: null })).toEqual([])
  })
})

describe('squadAssistFlow', () => {
  it('suit l’ordre du ROSTER, pas celui des paires servies, des deux côtés', () => {
    // Les paires arrivent triées par volume (Bob en tête) ; la page ordonne ses joueurs de la même façon partout.
    const flow = squadAssistFlow(BLOC, ['Alice', 'Bob', 'Carol'])
    expect(flow.nodes.filter((n) => n.side === 'assistant').map((n) => n.name)).toEqual(['Alice', 'Bob'])
    expect(flow.nodes.filter((n) => n.side === 'finisher').map((n) => n.name)).toEqual(['Alice', 'Bob', 'Carol'])
  })

  it('garde un assistant HORS roster, à la suite — la mesure l’a vu', () => {
    const flow = squadAssistFlow({ ...BLOC, pairs: [...PAIRES, paire('Dave', 'Alice', 5)] }, ['Alice', 'Bob'])
    expect(flow.nodes.filter((n) => n.side === 'assistant').map((n) => n.name)).toEqual(['Alice', 'Bob', 'Dave'])
  })

  it('divise par le TOTAL SERVEUR, jamais par la somme des rubans visibles', () => {
    // 30 sur les 100 assistances mesurées de l'escouade, et non sur les 60 affichées.
    const flow = squadAssistFlow(BLOC, ['Alice', 'Bob'])
    expect(flow.total).toBe(100)
    expect(flow.links.find((l) => l.assistantKey === 'Bob')!.share).toBe(0.3)
  })

  it('total à zéro : aucune part, jamais une division par zéro', () => {
    const flow = squadAssistFlow({ ...BLOC, total_assists: 0 }, ['Alice', 'Bob'])
    expect(flow.links.every((l) => l.share === null)).toBe(true)
  })

  it('aucune paire : aucun ruban (l’état vide est celui de la carte)', () => {
    expect(squadAssistFlow({ ...BLOC, pairs: [] }, []).links).toEqual([])
  })
})
