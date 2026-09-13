/**
 * squadAssistPairs.logic.test — les décisions PURES du graphe « Assistances dans
 * l'escouade », qui a remplacé le tableau le 2026-09-13.
 */
import { describe, expect, it } from 'vitest'

import type { SquadAssistPair, SquadAssistPairs } from '@/lib/api/types'

import {
  assistBeneficiaires,
  assistCle,
  assistPairsSeries,
  assistPartParCouple,
  assistVoleesParCouple,
} from './squadAssistPairs.logic'

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

const PAIRES = [
  paire('Bob', 'Alice', 30, 12),
  paire('Alice', 'Bob', 20),
  paire('Alice', 'Carol', 10, 4),
]

const BLOC: SquadAssistPairs = {
  matches_measured: 8,
  matches_total: 10,
  total_assists: 100,
  pairs: PAIRES,
}

describe('assistPairsSeries', () => {
  it('une barre par ASSISTANT, un segment par BÉNÉFICIAIRE', () => {
    const [serie] = assistPairsSeries(PAIRES, ['Alice', 'Bob', 'Carol'])
    expect(serie.datapoints).toEqual([
      { category: 'Alice', components: { Bob: 20, Carol: 10 } },
      { category: 'Bob', components: { Alice: 30 } },
    ])
  })

  it('suit l’ordre du ROSTER, pas celui des paires servies', () => {
    // Les paires arrivent triées par volume (Bob en tête) ; la page, elle, ordonne ses
    // joueurs de la même façon partout — joueur principal d'abord.
    const [serie] = assistPairsSeries(PAIRES, ['Alice', 'Bob'])
    expect(serie.datapoints.map((d) => d.category)).toEqual(['Alice', 'Bob'])
  })

  it('garde un assistant HORS roster, à la suite — la mesure l’a vu', () => {
    const [serie] = assistPairsSeries([...PAIRES, paire('Dave', 'Alice', 5)], ['Alice', 'Bob'])
    expect(serie.datapoints.map((d) => d.category)).toEqual(['Alice', 'Bob', 'Dave'])
  })

  it('aucune paire : AUCUNE série (l’état vide est celui du wrapper)', () => {
    expect(assistPairsSeries([], [])).toEqual([])
  })
})

describe('assistBeneficiaires', () => {
  it('rend les bénéficiaires dans l’ordre du roster', () => {
    expect(assistBeneficiaires(PAIRES, ['Alice', 'Bob', 'Carol'])).toEqual([
      'Alice',
      'Bob',
      'Carol',
    ])
  })
})

describe('assistPartParCouple', () => {
  it('divise par le TOTAL SERVEUR, jamais par la somme des barres visibles', () => {
    // 30 sur les 100 assistances mesurées de l'escouade, et non sur les 60 affichées :
    // dériver le dénominateur de l'affichage le ferait mentir dès qu'une paire serait
    // filtrée.
    expect(assistPartParCouple(BLOC).get(assistCle('Bob', 'Alice'))).toBe(0.3)
  })

  it('total à zéro : aucune part (et surtout pas une division par zéro)', () => {
    expect(assistPartParCouple({ ...BLOC, total_assists: 0 }).size).toBe(0)
  })
})

describe('assistVoleesParCouple', () => {
  it('indexe les éliminations volées par couple', () => {
    const volees = assistVoleesParCouple(PAIRES)
    expect(volees.get(assistCle('Bob', 'Alice'))).toBe(12)
    expect(volees.get(assistCle('Alice', 'Bob'))).toBe(0)
  })
})
