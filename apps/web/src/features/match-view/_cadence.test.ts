/**
 * _cadence.test.ts — la géométrie de la CADENCE DES FRAGS, sur des oracles écrits à la main.
 *
 * CE QUE CES TESTS PROTÈGENT, ET QUI N'ÉTAIT EXERCÉ PAR RIEN (registre 2026-09-05, N1) :
 * le seul test qui nommait ce graphe remplaçait le composant par un stub — zéro ligne
 * exécutée. Sont vérifiés ici : la répartition des frags entre les deux camps, la moyenne
 * mobile et sa fenêtre expansive au démarrage, le pic, et le pas de temps des abscisses.
 *
 * Les moyennes attendues sont posées à la main : `[2, 0, 4, 2]` donne `2`, `(2+0)/2 = 1`,
 * `(2+0+4)/3 = 2` et `(0+4+2)/3 = 2` — le module ne sert pas d'oracle à lui-même.
 */
import { describe, expect, it } from 'vitest'

import type { MatchViewCadence } from '@/lib/api/types'

import {
  buildCadence,
  CADENCE_PHASE_SECONDS_DEFAULT,
  movingAverage,
  type CadenceInput,
} from './_cadence'
import type { XuidMeta } from './xuidMeta'

/** `A` est allié, `E` adverse ; `Z` n'a aucune ligne de scoreboard. */
const META: XuidMeta = new Map([
  ['A', { gamertag: 'Alpha', ally: true }],
  ['E', { gamertag: 'Echo', ally: false }],
])

/** Une cadence servie par l'API : une entrée par phase, `components` par xuid. */
function cadence(
  phases: Array<Record<string, number>>,
  phaseSeconds?: number,
): MatchViewCadence {
  return {
    key: 'match_view.combat.cadence',
    datapoints: phases.map((components, i) => ({ label: `p${i}`, components })),
    ...(phaseSeconds !== undefined ? { meta: { phase_seconds: phaseSeconds } } : {}),
  } as unknown as MatchViewCadence
}

function input(over: Partial<CadenceInput> = {}): CadenceInput {
  return {
    // Quatre phases : les alliés font 2, 0, 4, 2 ; les adversaires 1, 1, 0, 3.
    cadence: cadence([
      { A: 2, E: 1 },
      { E: 1 },
      { A: 4 },
      { A: 2, E: 3 },
    ]),
    meXUID: 'A',
    xuidMeta: META,
    ...over,
  }
}

describe('buildCadence — quand le graphe existe', () => {
  it('rend null sans cadence servie', () => {
    expect(buildCadence(input({ cadence: null }))).toBeNull()
    expect(buildCadence(input({ cadence: undefined }))).toBeNull()
  })

  it('rend null sans aucune phase : un histogramme sans barre est un cadre vide', () => {
    expect(buildCadence(input({ cadence: cadence([]) }))).toBeNull()
  })

  it('rend un modèle même quand PERSONNE n’a fragué — « zéro partout » est une lecture', () => {
    const m = buildCadence(input({ cadence: cadence([{}, {}]) }))!
    expect(m.ally).toEqual([0, 0])
    expect(m.enemy).toEqual([0, 0])
    expect(m.peak).toEqual({ index: 0, total: 0 })
  })
})

describe('buildCadence — la répartition entre les deux camps', () => {
  it('somme les frags de chaque camp, phase par phase', () => {
    const m = buildCadence(input())!
    expect(m.ally).toEqual([2, 0, 4, 2])
    expect(m.enemy).toEqual([1, 1, 0, 3])
  })

  it('compte comme ADVERSE un tueur qu’aucune ligne de scoreboard ne rattache', () => {
    // C'est le comportement historique et il est délibéré : la cadence oppose deux camps,
    // et un frag qui n'est pas le nôtre est, pour ce graphe, celui d'en face.
    const m = buildCadence(input({ cadence: cadence([{ A: 1, Z: 5 }]) }))!
    expect(m.ally).toEqual([1])
    expect(m.enemy).toEqual([5])
  })

  it('garde le joueur de la page de SON côté même sans ligne au scoreboard', () => {
    const m = buildCadence(input({ cadence: cadence([{ inconnu: 3 }]), meXUID: 'inconnu' }))!
    expect(m.ally).toEqual([3])
    expect(m.enemy).toEqual([0])
  })
})

describe('buildCadence — la moyenne mobile', () => {
  it('part dès la PREMIÈRE phase, avec une fenêtre réduite', () => {
    const m = buildCadence(input())!
    // 2 ; (2+0)/2 = 1 ; (2+0+4)/3 = 2 ; (0+4+2)/3 = 2
    expect(m.allyMA).toEqual([2, 1, 2, 2])
    // 1 ; (1+1)/2 = 1 ; (1+1+0)/3 = 0.7 ; (1+0+3)/3 = 1.3
    expect(m.enemyMA).toEqual([1, 1, 0.7, 1.3])
  })

  it('arrondit au dixième — la moyenne d’un compte entier n’a pas plus de précision', () => {
    expect(movingAverage([1, 0, 0])).toEqual([1, 0.5, 0.3])
    expect(movingAverage([2, 2, 2, 5])).toEqual([2, 2, 2, 3])
  })

  it('rend une série vide sur une entrée vide, sans jamais lever', () => {
    expect(movingAverage([])).toEqual([])
  })
})

describe('buildCadence — le pic', () => {
  it('désigne la phase la plus meurtrière, camps confondus', () => {
    // Totaux par phase : 3, 1, 4, 5 -> la dernière.
    expect(buildCadence(input())!.peak).toEqual({ index: 3, total: 5 })
  })

  it('garde la PREMIÈRE à égalité : le pic est celui qu’on a vu arriver d’abord', () => {
    const m = buildCadence(input({ cadence: cadence([{ A: 3 }, { E: 3 }]) }))!
    expect(m.peak).toEqual({ index: 0, total: 3 })
  })
})

describe('buildCadence — les abscisses', () => {
  it('date chaque phase par son DÉBUT, au pas déclaré par le serveur', () => {
    const m = buildCadence(input({ cadence: cadence([{}, {}, {}], 45) }))!
    expect(m.categories).toEqual(['0m00s', '0m45s', '1m30s'])
  })

  it('retombe sur trente secondes quand le serveur ne déclare pas le pas', () => {
    expect(CADENCE_PHASE_SECONDS_DEFAULT).toBe(30)
    expect(buildCadence(input())!.categories).toEqual(['0m00s', '0m30s', '1m00s', '1m30s'])
  })
})
