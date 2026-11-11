/**
 * Tests — la position que « Suivre » donne à la carte (lot 4 du plan « maquettes du rejeu »).
 *
 * Ce qu'ils protègent :
 *  1. LA POSITION SUIT L'IMAGE, y compris fractionnaire : la lecture et l'interpolation du film
 *     donnent des images à virgule, la carte ne saute pas d'une image entière à l'autre.
 *  2. UN JOUEUR EMBARQUÉ se suit sur son véhicule, jamais sur son bipède figé.
 *  3. SANS POSITION, RIEN : joueur mort depuis plus que la fenêtre après-mort, lacune avant la
 *     première vie — `null`, et la carte reste où elle est. Aucune position n'est inventée.
 *  4. UN BOT SE SUIT PAR SES VIES : il n'a pas de xuid.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { FOLLOW_MIN_LEVEL, followPositionAt, sameSpot } from './followLogic'

const NO_VEHICLE = () => null
const DEATH_FRAMES = 15

function life(slot: number, start: number, end: number, points: Array<[number, number, number]>) {
  return {
    slot, team: 0, xuid: 'A', startFrame: start, endFrame: end,
    points: points.map(([t, x, y]) => ({ t, x, y })),
  }
}

function player(lives: ReturnType<typeof life>[], extra: Partial<ReplayPlayer> = {}): ReplayPlayer {
  return { xuid: 'A', lives: lives as never, ...extra }
}

describe('followPositionAt', () => {
  const p = player([life(1, 0, 100, [[0, 0, 0], [100, 100, 50]])])

  it('lit la position à une image fractionnaire, par interpolation', () => {
    expect(followPositionAt(p, 25.5, NO_VEHICLE, DEATH_FRAMES)).toEqual({ x: 25.5, y: 12.75 })
  })

  it('un joueur embarqué se suit sur son véhicule', () => {
    const embarked = (xuid: string, frame: number) => (xuid === 'A' && frame === 40 ? { x: 7, y: 9 } : null)
    expect(followPositionAt(p, 40, embarked, DEATH_FRAMES)).toEqual({ x: 7, y: 9 })
    expect(followPositionAt(p, 41, embarked, DEATH_FRAMES)).toEqual({ x: 41, y: 20.5 })
  })

  it('un joueur mort depuis peu garde sa dernière position, puis plus rien', () => {
    const dead = player([life(1, 0, 50, [[0, 0, 0], [50, 10, 10]])])
    expect(followPositionAt(dead, 55, NO_VEHICLE, DEATH_FRAMES)).toEqual({ x: 10, y: 10 })
    expect(followPositionAt(dead, 80, NO_VEHICLE, DEATH_FRAMES)).toBeNull()
  })

  it('avant sa première vie : null', () => {
    const late = player([life(1, 60, 100, [[60, 5, 5], [100, 9, 9]])])
    expect(followPositionAt(late, 10, NO_VEHICLE, DEATH_FRAMES)).toBeNull()
  })

  it('un bot se suit par ses vies, sans xuid', () => {
    const bot = player([life(7, 0, 100, [[0, 4, 4], [100, 4, 4]])], { xuid: 'bot:Marc', bot: true })
    expect(followPositionAt(bot, 10, NO_VEHICLE, DEATH_FRAMES)).toEqual({ x: 4, y: 4 })
  })
})

describe('sameSpot', () => {
  it('deux positions identiques, ou deux absences, sont le même endroit', () => {
    expect(sameSpot({ x: 1, y: 2 }, { x: 1, y: 2 })).toBe(true)
    expect(sameSpot(null, null)).toBe(true)
    expect(sameSpot({ x: 1, y: 2 }, { x: 1, y: 2.5 })).toBe(false)
    expect(sameSpot(null, { x: 0, y: 0 })).toBe(false)
  })
})

describe('FOLLOW_MIN_LEVEL', () => {
  it('est un palier du zoom : Suivre zoome au moins jusque-là', () => {
    expect(FOLLOW_MIN_LEVEL).toBe(2)
  })
})
