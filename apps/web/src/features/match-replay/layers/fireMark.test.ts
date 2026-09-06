/**
 * Tests — fireMark (le « ! » dans le point du tireur).
 *
 * Ce que ces tests verrouillent :
 *  - la JOINTURE : un tir marque la vie qui COUVRE son instant (le slot est réattribué à
 *    chaque réapparition), jamais une autre vie du même slot, jamais la mêlée ;
 *  - la FENÊTRE : le glyphe n'existe que pendant [frame, frame + hold] — la même rémanence
 *    que l'éclair de bouche, deux effets du même événement ;
 *  - les PORTES : une vie fermée ne marque plus rien (sa croix de mort n'est pas un joueur
 *    qui tire), une vie sans couleur n'a pas de marqueur donc pas de « ! » ;
 *  - la GÉOMÉTRIE : le glyphe est CENTRÉ sur la position interpolée du marqueur
 *    (positionAt + worldToCanvas, la même chaîne que les tracks) ;
 *  - l'ENCRE : celle passée par l'appelant (le thème), jamais une couleur d'ici.
 */
import { describe, expect, it } from 'vitest'

import { buildFireMarks, drawFireMarks } from './fireMark'
import { worldToCanvas } from '../replayLogic'
import type { ReplayDocumentReady } from '../replayNormalize'
import { recordingContext } from '../test/recordingContext'

const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 100,
  height: 100,
  pad: 0,
}

function docWith(over: Partial<ReplayDocumentReady>): ReplayDocumentReady {
  return {
    shots: [],
    tracks: [],
    ...over,
  } as ReplayDocumentReady
}

/** Deux VIES du même slot : la première couvre [10, 20], la seconde [30, 40]. */
const LIFE_1 = {
  slot: 5,
  team: -1,
  points: [
    { t: 10, x: 2, y: 2 },
    { t: 20, x: 4, y: 2 },
  ],
}
const LIFE_2 = {
  slot: 5,
  team: -1,
  points: [
    { t: 30, x: 8, y: 8 },
    { t: 40, x: 8, y: 8 },
  ],
}

const STYLE = {
  hold: 4,
  colorOfSlot: () => 'teinte',
  ink: 'encre',
  k: 1,
}

describe('buildFireMarks', () => {
  it('joint chaque tir à la vie qui COUVRE son instant, pas à une autre vie du slot', () => {
    const doc = docWith({
      tracks: [LIFE_1, LIFE_2] as ReplayDocumentReady['tracks'],
      shots: [{ t: 32, x: 8, y: 8, slot: 5 }] as ReplayDocumentReady['shots'],
    })
    const marks = buildFireMarks(doc)
    expect(marks).toHaveLength(1)
    expect(marks[0].track).toBe(LIFE_2)
  })

  it('écarte un tir sans vie couvrante : personne à marquer', () => {
    const doc = docWith({
      tracks: [LIFE_1] as ReplayDocumentReady['tracks'],
      shots: [{ t: 25, x: 4, y: 2, slot: 5 }] as ReplayDocumentReady['shots'],
    })
    expect(buildFireMarks(doc)).toHaveLength(0)
  })

  it('écarte la mêlée : un coup de marteau n’est pas un tir (même règle que l’éclair)', () => {
    const doc = docWith({
      tracks: [LIFE_1] as ReplayDocumentReady['tracks'],
      shots: [{ t: 12, x: 2, y: 2, slot: 5, w: 'marteau' }] as ReplayDocumentReady['shots'],
      weaponLabels: { marteau: { en: 'Hammer', fr: 'Marteau', fx: 'melee' } },
    })
    expect(buildFireMarks(doc)).toHaveLength(0)
  })
})

describe('drawFireMarks', () => {
  const marks = () =>
    buildFireMarks(
      docWith({
        tracks: [LIFE_1] as ReplayDocumentReady['tracks'],
        shots: [{ t: 12, x: 2, y: 2, slot: 5 }] as ReplayDocumentReady['shots'],
      }),
    )

  it('centre le glyphe sur la position INTERPOLÉE du marqueur, à l’encre de l’appelant', () => {
    const { ops, ctx } = recordingContext()
    drawFireMarks(ctx, marks(), VIEW, { ...STYLE, frame: 15 })
    // À t=15, le joueur est à mi-chemin entre (2,2) et (4,2) : (3,2) — le glyphe le SUIT.
    const c = worldToCanvas({ x: 3, y: 2 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    const moveTo = ops.find((o) => o.op === 'moveTo')
    expect(moveTo?.args[0]).toBe(c.x)
    const arc = ops.find((o) => o.op === 'arc')
    expect(arc?.args?.[0]).toBe(c.x)
    // Barre AU-DESSUS du centre, point EN DESSOUS : le « ! » est centré, pas posé à côté.
    expect(moveTo!.args[1]).toBeLessThan(c.y)
    expect(arc!.args[1]).toBeGreaterThan(c.y)
    expect(ops.some((o) => o.op === 'stroke')).toBe(true)
    expect(ops.some((o) => o.op === 'fill')).toBe(true)
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'encre')).toBe(true)
  })

  it('ne dessine RIEN hors de la fenêtre du tir : la rémanence est celle de l’éclair', () => {
    for (const frame of [11, 17]) {
      const { ops, ctx } = recordingContext()
      drawFireMarks(ctx, marks(), VIEW, { ...STYLE, frame })
      expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(0)
    }
  })

  it('ne dessine rien sur une vie FERMÉE : la croix de mort n’est pas un joueur qui tire', () => {
    const fx = buildFireMarks(
      docWith({
        tracks: [LIFE_1] as ReplayDocumentReady['tracks'],
        // Tir à la toute fin de la vie : la fenêtre (hold 4) survit à la vie (fin t=20).
        shots: [{ t: 19, x: 4, y: 2, slot: 5 }] as ReplayDocumentReady['shots'],
      }),
    )
    const { ops, ctx } = recordingContext()
    drawFireMarks(ctx, fx, VIEW, { ...STYLE, frame: 22 })
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(0)
  })

  it('ne dessine rien pour une vie SANS couleur : pas de marqueur, pas de « ! »', () => {
    const { ops, ctx } = recordingContext()
    drawFireMarks(ctx, marks(), VIEW, { ...STYLE, frame: 15, colorOfSlot: () => null })
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(0)
  })
})
