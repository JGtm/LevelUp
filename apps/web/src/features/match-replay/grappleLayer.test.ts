/**
 * Tests — grappleLayer (la ligne de grappin : jointure, fenêtre, projection).
 *
 * Ce que ces tests verrouillent :
 *  - la JOINTURE : une traction ne se dessine que si sa vie est publiée (des points
 *    existent) et que sa fenêtre est non vide — jamais une ligne sans joueur à relier ;
 *  - la FENÊTRE : la ligne n'existe que pour frame dans [t0, t1] — la durée EST la
 *    traction mesurée, aucune rémanence ;
 *  - la GÉOMÉTRIE : le segment part de la position INTERPOLÉE du joueur (positionAt,
 *    la même chaîne que le marqueur) et va à l'ancre, tous deux projetés par
 *    worldToCanvas — la même projection que les tracks ;
 *  - l'ENCRE : celle passée par l'appelant (le thème), jamais une couleur d'ici.
 */
import { describe, expect, it } from 'vitest'

import { buildGrappleFx, drawGrappleLayer } from './grappleLayer'
import { worldToCanvas } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { recordingContext } from './test/recordingContext'

const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 100,
  height: 100,
  pad: 0,
}

function docWith(over: Partial<ReplayDocumentReady>): ReplayDocumentReady {
  return {
    grappleLines: [],
    tracks: [],
    ...over,
  } as ReplayDocumentReady
}

const TRACK = {
  slot: 5,
  team: -1,
  points: [
    { t: 10, x: 2, y: 2 },
    { t: 20, x: 4, y: 2 },
  ],
}

describe('buildGrappleFx', () => {
  it('joint chaque traction aux points de SA vie', () => {
    const doc = docWith({
      tracks: [TRACK] as ReplayDocumentReady['tracks'],
      grappleLines: [{ slot: 5, t0: 12, t1: 18, ax: 8, ay: 2 }],
    })
    const fx = buildGrappleFx(doc)
    expect(fx).toHaveLength(1)
    expect(fx[0].points).toBe(TRACK.points)
    expect(fx[0].anchor).toEqual({ x: 8, y: 2 })
  })

  it('écarte une traction dont la vie n’est pas publiée : aucun joueur à relier', () => {
    const doc = docWith({
      tracks: [TRACK] as ReplayDocumentReady['tracks'],
      grappleLines: [{ slot: 99, t0: 12, t1: 18, ax: 8, ay: 2 }],
    })
    expect(buildGrappleFx(doc)).toHaveLength(0)
  })

  it('écarte une fenêtre vide : rien à tracer', () => {
    const doc = docWith({
      tracks: [TRACK] as ReplayDocumentReady['tracks'],
      grappleLines: [{ slot: 5, t0: 18, t1: 18, ax: 8, ay: 2 }],
    })
    expect(buildGrappleFx(doc)).toHaveLength(0)
  })
})

describe('drawGrappleLayer', () => {
  const entry = () =>
    buildGrappleFx(
      docWith({
        tracks: [TRACK] as ReplayDocumentReady['tracks'],
        grappleLines: [{ slot: 5, t0: 12, t1: 18, ax: 8, ay: 2 }],
      }),
    )

  it('trace le segment de la position INTERPOLÉE du joueur vers l’ancre', () => {
    const { ops, ctx } = recordingContext()
    drawGrappleLayer(ctx, entry(), VIEW, 15, 'encre')
    // À t=15, le joueur est à mi-chemin entre (2,2) et (4,2) : (3,2).
    const p = worldToCanvas({ x: 3, y: 2 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    const a = worldToCanvas({ x: 8, y: 2 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    const moveTo = ops.find((o) => o.op === 'moveTo')
    const lineTo = ops.find((o) => o.op === 'lineTo')
    expect(moveTo?.args).toEqual([p.x, p.y])
    expect(lineTo?.args).toEqual([a.x, a.y])
    expect(ops.some((o) => o.op === 'stroke')).toBe(true)
    // Le point d'accroche : un disque à l'ancre.
    const arc = ops.find((o) => o.op === 'arc')
    expect(arc?.args?.slice(0, 2)).toEqual([a.x, a.y])
    // L'encre est celle de l'appelant.
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'encre')).toBe(true)
  })

  it('ne trace RIEN hors de la fenêtre mesurée : la durée est la traction, pas une rémanence', () => {
    for (const frame of [11, 19]) {
      const { ops, ctx } = recordingContext()
      drawGrappleLayer(ctx, entry(), VIEW, frame, 'encre')
      expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(0)
    }
  })

  it('ne trace rien avant le premier point de la vie : pas de position, pas de ligne', () => {
    const fx = buildGrappleFx(
      docWith({
        tracks: [TRACK] as ReplayDocumentReady['tracks'],
        // Fenêtre ouvrant AVANT le premier échantillon de la track (t=10).
        grappleLines: [{ slot: 5, t0: 5, t1: 18, ax: 8, ay: 2 }],
      }),
    )
    const { ops, ctx } = recordingContext()
    drawGrappleLayer(ctx, fx, VIEW, 7, 'encre')
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(0)
  })
})
