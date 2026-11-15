/**
 * Tests — livesPosition (la relecture de la position d'un BIPÈDE à une image).
 *
 * CES TROIS CAS VIENNENT DE `killFx.test.ts` : ils y étaient parce que `posOfPlayerAt` y était
 * définie. La primitive a été rapatriée dans son module canonique le 2026-09-05 (le cycle
 * d'imports qui la retenait dans killFx.ts n'existe plus) ; ses tests l'ont suivie, à
 * l'identique — un test qui reste derrière une fonction déplacée devient un test que personne
 * ne relie plus à ce qu'il protège.
 *
 * CE QU'ILS PROTÈGENT : une vie vivante rend sa position interpolée ; une vie CLOSE la rend
 * encore, mais seulement dans la fenêtre après-mort (la victime vient de mourir, par
 * construction) ; au-delà, rien — aucune position inventée.
 */
import { describe, expect, it } from 'vitest'

import { altitudeOfPlayerAt, posOfPlayerAt } from './livesPosition'
import { testReplayDoc } from '../test/testDoc'

/** La victime : vie CLOSE à la frame 20, deux échantillons. */
const doc = testReplayDoc({
  frameIntervalMs: 100,
  tracks: [
    {
      slot: 2,
      team: -1,
      xuid: 'V',
      points: [
        { t: 0, x: 5, y: 5 },
        { t: 20, x: 5, y: 4 },
      ],
      startFrame: 0,
      endFrame: 20,
    },
  ] as never,
})

describe('posOfPlayerAt', () => {
  const lives = doc.tracks.filter((t) => t.xuid === 'V')

  it('rend la position exacte pendant la vie', () => {
    expect(posOfPlayerAt(lives, 10, 15)).toEqual({ x: 5, y: 4.5 })
  })

  it('rend la DERNIÈRE position dans la fenêtre DEATH après la fin de vie', () => {
    expect(posOfPlayerAt(lives, 30, 15)).toEqual({ x: 5, y: 4 })
  })

  it("ne rend RIEN au-delà de la fenêtre — aucune position inventée", () => {
    expect(posOfPlayerAt(lives, 40, 15)).toBeNull()
  })
})

describe('altitudeOfPlayerAt — la même règle de vie que la position', () => {
  const avecZ = testReplayDoc({
    tracks: [
      {
        slot: 2,
        team: -1,
        xuid: 'V',
        points: [
          { t: 0, x: 5, y: 5, z: 0 },
          { t: 20, x: 5, y: 4, z: 4 },
        ],
        startFrame: 0,
        endFrame: 20,
      },
    ] as never,
  }).tracks
  const sansZ = testReplayDoc({
    tracks: [{ slot: 3, team: -1, xuid: 'W', points: [{ t: 0, x: 1, y: 1 }, { t: 20, x: 2, y: 2 }], startFrame: 0, endFrame: 20 }] as never,
  }).tracks

  it('rend l’altitude interpolée pendant la vie', () => {
    expect(altitudeOfPlayerAt(avecZ, 10, 15)).toBeCloseTo(2)
  })

  it('rend la dernière altitude dans la fenêtre après-mort, rien au-delà', () => {
    expect(altitudeOfPlayerAt(avecZ, 30, 15)).toBe(4)
    expect(altitudeOfPlayerAt(avecZ, 40, 15)).toBeNull()
  })

  it('une vie dont aucun point ne porte de z rend null, jamais un 0 inventé', () => {
    expect(altitudeOfPlayerAt(sansZ, 10, 15)).toBeNull()
    expect(altitudeOfPlayerAt(sansZ, 30, 15)).toBeNull()
  })

  it('joueur inconnu : null', () => {
    expect(altitudeOfPlayerAt(undefined, 10, 15)).toBeNull()
  })
})
