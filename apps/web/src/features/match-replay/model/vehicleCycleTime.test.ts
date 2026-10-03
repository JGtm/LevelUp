/// <reference types="node" />
/**
 * Tests — vehicleCycleTime : L'OCCUPATION D'UN EMPLACEMENT DE NAISSANCE ET SON COMPTE À REBOURS.
 *
 * CE QUE CE FICHIER VERROUILLE, clause par clause du lot 5.8 :
 *  - L'APPARIEMENT SE FAIT À LA MAILLE DE LA MESURE, et la maille est celle du producteur Go —
 *    un garde-rail la RELIT dans `vehicle_cycles.go` : deux mailles apparieraient des vies à des
 *    emplacements voisins, et le décalage serait invisible à l'écran.
 *  - UNE VIE NÉE ICI ET ENCORE EN VIGUEUR OCCUPE L'EMPLACEMENT, où qu'elle roule : l'horloge du
 *    jeu ne repart qu'à la destruction.
 *  - L'ORDRE DES SOURCES : la naissance suivante VUE DANS LE FILM l'emporte sur le cycle.
 *  - UNE FIN NON DATÉE OU NON DESTRUCTRICE NE PRÉDIT RIEN (`film_end`, `unknown`) : compter
 *    depuis elle reviendrait à mesurer le recensement des images-clés.
 */
import { describe, expect, it } from 'vitest'

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { racineDuDepot } from '../test/featureFiles'
import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import {
  VEHICLE_CYCLE_CLUSTER_M,
  vehicleCycleLives,
  vehicleCycleOccupantAt,
  vehicleCycleReadingAt,
  vehicleCycleRespawnAt,
  type VehicleCycle,
} from './vehicleCycleTime'

/** Une image de 100 ms : 10 images = 1 s, ce qui rend les comptes lisibles à l'œil nu. */
const FRAME_MS = 100

/** L'emplacement témoin : cycle de 40 s pile, deux écarts mesurés. */
function cycle(over: Partial<VehicleCycle> = {}): VehicleCycle {
  return { x: 5, y: 5, family: 'warthog', medianS: 40, p10S: 38, p90S: 42, gaps: 2, missing: 0, ...over }
}

/**
 * Une vie de véhicule. `t1max` porte la dernière image où le calque la dessinerait ; `tEnd` et
 * `end` disent ce que le film ÉCRIT de sa mort.
 */
function life(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 1,
    gen: 0,
    family: 'warthog',
    end: 'destroyed',
    t0: 0,
    t1: 100,
    t1max: 100,
    tEnd: 100,
    spawn: { x: 5, y: 5 },
    samples: [],
    rides: [],
    ...over,
  } as ReplayVehicleTrackReady
}

describe('vehicleCycleTime — la maille de l appariement', () => {
  it('vaut EXACTEMENT celle du producteur Go (garde-rail : le document ne la publie pas)', () => {
    const go = readFileSync(
      resolve(
        racineDuDepot(),
        'apps/go-api/internal/games/halo_infinite/film/replay/vehicle_cycles.go',
      ),
      'utf8',
    )
    const m = /const vehicleCycleClusterM = ([0-9.]+)/.exec(go)
    expect(m, 'la constante du producteur doit rester trouvable').not.toBeNull()
    expect(Number(m?.[1])).toBe(VEHICLE_CYCLE_CLUSTER_M)
  })

  it('retient les naissances dans le rayon, écarte les autres, et trie par naissance', () => {
    const dedans = life({ slot: 1, t0: 30, spawn: { x: 6, y: 5 } })
    const dehors = life({ slot: 2, t0: 10, spawn: { x: 9, y: 5 } })
    const juste = life({ slot: 3, t0: 20, spawn: { x: 5, y: 5 } })
    const lives = vehicleCycleLives(cycle(), [dedans, dehors, juste])
    expect(lives.map((l) => l.slot)).toEqual([3, 1])
  })

  it('une vie SANS naissance située n entre dans aucun emplacement', () => {
    const sansSpawn = life({ spawn: undefined })
    expect(vehicleCycleLives(cycle(), [sansSpawn])).toEqual([])
  })
})

describe('vehicleCycleTime — l occupation', () => {
  it('une vie née ici et encore en vigueur OCCUPE l emplacement (et il n a pas de compte)', () => {
    const lives = [life({ t0: 0, tEnd: 200, t1max: 200 })]
    expect(vehicleCycleOccupantAt(lives, 50)).toBe(lives[0])
    const lu = vehicleCycleReadingAt(cycle(), lives, 50, FRAME_MS)
    expect(lu.occupant).toBe(lives[0])
    expect(lu.respawn).toBeNull()
  })

  it('une vie détruite ne l occupe plus : le compte à rebours part de sa fin datée', () => {
    const lives = [life({ t0: 0, tEnd: 100, t1max: 100 })]
    expect(vehicleCycleOccupantAt(lives, 150)).toBeNull()
    // 5 s après la destruction, sur un cycle de 40 s : 35 s restent, et c'est une PRÉDICTION.
    const lu = vehicleCycleReadingAt(cycle(), lives, 150, FRAME_MS)
    expect(lu.occupant).toBeNull()
    expect(lu.respawn).toEqual({ seconds: 35, measured: false })
  })

  it('avant la première naissance, rien n occupe et rien ne se prédit', () => {
    const lives = [life({ t0: 100, tEnd: 300, t1max: 300 })]
    const lu = vehicleCycleReadingAt(cycle(), lives, 10, FRAME_MS)
    expect(lu.occupant).toBeNull()
    // La naissance suivante est VUE dans le film : 90 images, soit 9 s, et c'est EXACT.
    expect(lu.respawn).toEqual({ seconds: 9, measured: true })
  })
})

describe('vehicleCycleTime — l ordre des sources', () => {
  it('la naissance suivante VUE dans le film l emporte sur le cycle', () => {
    const lives = [
      life({ slot: 1, t0: 0, tEnd: 100, t1max: 100 }),
      life({ slot: 2, t0: 120, tEnd: 400, t1max: 400 }),
    ]
    // Le cycle prédirait 35 s ; le film montre la naissance dans 2 s. C'est le film qui parle.
    expect(vehicleCycleRespawnAt(cycle(), lives, 100, FRAME_MS)).toEqual({
      seconds: 2,
      measured: true,
    })
  })

  it('la dernière fin datée sert de départ, pas la première', () => {
    const lives = [
      life({ slot: 1, t0: 0, tEnd: 100, t1max: 100 }),
      life({ slot: 2, t0: 200, tEnd: 300, t1max: 300 }),
    ]
    // À l'image 400 : 10 s après la SECONDE mort, donc 30 s restent sur un cycle de 40 s.
    expect(vehicleCycleRespawnAt(cycle(), lives, 400, FRAME_MS)).toEqual({
      seconds: 30,
      measured: false,
    })
  })

  it('une fin NON DESTRUCTRICE ou NON DATÉE ne prédit rien', () => {
    const filmEnd = [life({ t0: 0, end: 'film_end', tEnd: 100, t1max: 100 })]
    expect(vehicleCycleRespawnAt(cycle(), filmEnd, 150, FRAME_MS)).toBeNull()
    const inconnue = [life({ t0: 0, end: 'unknown', tEnd: undefined, t1max: 100 })]
    expect(vehicleCycleRespawnAt(cycle(), inconnue, 150, FRAME_MS)).toBeNull()
  })

  it('un compte ÉPUISÉ ne s écrit pas : le cycle a été dépassé', () => {
    const lives = [life({ t0: 0, tEnd: 100, t1max: 100 })]
    // 60 s après la mort, sur un cycle de 40 s : il ne reste rien à annoncer.
    expect(vehicleCycleRespawnAt(cycle(), lives, 700, FRAME_MS)).toBeNull()
  })

  it('un document sans cadence d image ne prédit rien (jamais un zéro déguisé)', () => {
    const lives = [life({ t0: 0, tEnd: 100, t1max: 100 })]
    expect(vehicleCycleRespawnAt(cycle(), lives, 150, 0)).toBeNull()
  })
})
