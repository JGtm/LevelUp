/**
 * Tests — reliefFollowSample : ce que « Suivre le joueur ouvert » lit du film à une image.
 *
 * Ce qu'ils protègent : (1) le temps est celui du REJEU, en millisecondes (`frameIntervalMs`), jamais l'image
 * brute ; (2) une vie vivante rend sa position et la hauteur de ses pieds, et son rang dans `lives` nomme la
 * vie ; (3) mort ou pas encore réapparu : un échantillon silencieux, sans fenêtre après-mort ; (4) embarqué :
 * la position ET la hauteur du véhicule, jamais le z du bipède ; un véhicule sans hauteur rend un échantillon
 * silencieux plutôt qu'un étage inventé.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { testReplayDoc } from '../test/testDoc'
import { buildFollowSampler } from './reliefFollowSample'

const FRAME_MS = 100

/** Deux vies du joueur A : de 0 à 20 (au sol, x de 0 à 20), puis de 50 à 70 (en haut, z = 2,2). */
const TRACKS = [
  {
    slot: 1, team: 0, xuid: 'A', startFrame: 0, endFrame: 20,
    points: [{ t: 0, x: 0, y: 5, z: 0 }, { t: 20, x: 20, y: 5, z: 0 }],
  },
  {
    slot: 1, team: 0, xuid: 'A', startFrame: 50, endFrame: 70,
    points: [{ t: 50, x: 7, y: 5, z: 2.2 }, { t: 70, x: 9, y: 5, z: 2.2 }],
  },
]

/** Un Warthog conduit par A de 5 à 15 : il monte en Y, à z = 4 (ou sans hauteur quand `z` est absent). */
function vehicle(withZ: boolean) {
  const sample = (t: number, y: number) => (withZ ? { t, x: 0, y, z: 4 } : { t, x: 0, y })
  return {
    slot: 700, gen: 1, t0: 0, t1: 100, t1max: 100, end: 'unknown', family: 'warthog',
    samples: [sample(0, 0), sample(100, 100)],
    rides: [{ t0: 5, t1: 15, slot: 1, seat: 0, src: 'event', xuid: 'A' }],
  }
}

function playerOf(vehicles: unknown[] = []) {
  const doc = testReplayDoc({ frameIntervalMs: FRAME_MS, tracks: TRACKS as never, vehicles: vehicles as never })
  const player = { xuid: 'A', lives: doc.tracks } as unknown as ReplayPlayer
  return { doc, sample: buildFollowSampler(doc, player) }
}

describe('buildFollowSampler — le joueur vivant', () => {
  it('rend sa position, la hauteur de ses pieds et l’instant du rejeu en millisecondes', () => {
    const { sample } = playerOf()
    expect(sample(10)).toEqual({ alive: true, lifeKey: 0, x: 10, y: 5, z: 0, atMs: 10 * FRAME_MS })
  })

  it('une image fractionnaire donne un instant fractionnaire (la lecture n’est pas calée sur les images)', () => {
    const { sample } = playerOf()
    expect(sample(2.5)).toMatchObject({ x: 2.5, atMs: 250 })
  })

  it('la vie est son rang dans la liste : une autre vie est une autre clé', () => {
    const { sample } = playerOf()
    expect(sample(60)).toEqual({ alive: true, lifeKey: 1, x: 8, y: 5, z: 2.2, atMs: 60 * FRAME_MS })
  })
})

describe('buildFollowSampler — mort et réapparition', () => {
  it('entre deux vies : silencieux, sans fenêtre après-mort', () => {
    const { sample } = playerOf()
    for (const frame of [21, 30, 49]) expect(sample(frame)).toMatchObject({ alive: false, atMs: frame * FRAME_MS })
  })

  it('avant la première vie et après la dernière : silencieux', () => {
    const { doc } = playerOf()
    const lateStart = { xuid: 'A', lives: doc.tracks.slice(1) } as unknown as ReplayPlayer
    expect(buildFollowSampler(doc, lateStart)(10).alive).toBe(false)
    expect(playerOf().sample(71).alive).toBe(false)
  })
})

describe('buildFollowSampler — le joueur embarqué', () => {
  it('prend la position ET la hauteur du véhicule, jamais le z du bipède', () => {
    const { sample } = playerOf([vehicle(true)])
    // Le bipède, lui, serait en (10, 5) à z = 0 : la monture est en (0, 10) à z = 4.
    expect(sample(10)).toMatchObject({ alive: true, x: 0, y: 10, z: 4 })
  })

  it('un véhicule sans hauteur lue : silencieux, la coupe ne se pose pas sur le z du bipède', () => {
    const { sample } = playerOf([vehicle(false)])
    expect(sample(10).alive).toBe(false)
  })

  it('hors de l’épisode, le bipède répond de nouveau', () => {
    const { sample } = playerOf([vehicle(true)])
    expect(sample(18)).toMatchObject({ alive: true, x: 18, y: 5, z: 0 })
  })
})
