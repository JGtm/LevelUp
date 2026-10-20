/**
 * Tests — grappleLayer (la ligne de grappin : jointure, fenêtre, projection).
 *
 * Ce que ces tests verrouillent :
 *  - la JOINTURE : une traction ne se dessine que si sa vie est publiée (des points
 *    existent) et que sa fenêtre est non vide — jamais une ligne sans joueur à relier ;
 *  - la JOINTURE PAR VIE : un slot de biped est RÉATTRIBUÉ à chaque réapparition, donc la
 *    traction prend la vie qui COUVRE son départ — jamais la dernière piste du slot ;
 *  - la FENÊTRE : la ligne n'existe que pour frame dans [t0, t1] — la durée EST la
 *    traction mesurée, aucune rémanence ;
 *  - la GÉOMÉTRIE : le segment part de la position INTERPOLÉE du joueur (positionAt,
 *    la même chaîne que le marqueur) et va à l'ancre, tous deux projetés par
 *    worldToCanvas — la même projection que les tracks ;
 *  - l'ENCRE : celle passée par l'appelant (le thème), jamais une couleur d'ici.
 */
import { describe, expect, it } from 'vitest'

import { buildGrappleFx, drawGrappleLayer } from './grappleLayer'
import { worldToCanvas } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { recordingContext } from '../test/recordingContext'

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

/**
 * DEUX VIES DU MÊME SLOT, aux trajectoires ORTHOGONALES et aux fenêtres DISJOINTES : la
 * première file plein est en bas à gauche, la seconde plein nord en haut à droite. La
 * confusion des deux ne peut donc pas passer inaperçue.
 */
const VIE_1 = {
  slot: 5,
  team: -1,
  startFrame: 10,
  endFrame: 20,
  points: [
    { t: 10, x: 2, y: 2 },
    { t: 20, x: 4, y: 2 },
  ],
}
const VIE_2 = {
  slot: 5,
  team: -1,
  startFrame: 30,
  endFrame: 40,
  points: [
    { t: 30, x: 8, y: 8 },
    { t: 40, x: 8, y: 9 },
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

  /**
   * LE CAS QUI A MOTIVÉ LA CORRECTION (lot D1). Un slot de biped est RÉATTRIBUÉ à chaque
   * réapparition : indexer `slot -> points` ne garde que la DERNIÈRE piste du slot. Une
   * accroche de la PREMIÈRE vie irait alors chercher les points de la seconde — ici la
   * seconde vie est ailleurs (plein nord, en haut à droite) et commence APRÈS la traction,
   * donc le câble disparaîtrait purement et simplement de la carte ; si les fenêtres se
   * chevauchaient, il se peindrait à la position d'UN AUTRE JOUEUR, tendu vers une ancre
   * qui n'est pas la sienne.
   */
  it('DEUX VIES du même slot : chaque traction prend la vie qui COUVRE son départ', () => {
    const doc = docWith({
      tracks: [VIE_1, VIE_2] as unknown as ReplayDocumentReady['tracks'],
      grappleLines: [
        { slot: 5, t0: 12, t1: 18, ax: 8, ay: 2 },
        { slot: 5, t0: 32, t1: 38, ax: 2, ay: 9 },
      ],
    })
    const fx = buildGrappleFx(doc)
    expect(fx).toHaveLength(2)
    // La première traction tient les points de la PREMIÈRE vie — pas ceux de la seconde.
    expect(fx[0].points).toBe(VIE_1.points)
    expect(fx[0].anchor).toEqual({ x: 8, y: 2 })
    // La seconde tient bien la seconde vie.
    expect(fx[1].points).toBe(VIE_2.points)
  })

  it('écarte une traction qu’AUCUNE vie du slot ne couvre : entre deux vies', () => {
    const doc = docWith({
      tracks: [VIE_1, VIE_2] as unknown as ReplayDocumentReady['tracks'],
      grappleLines: [{ slot: 5, t0: 24, t1: 27, ax: 8, ay: 2 }],
    })
    expect(buildGrappleFx(doc)).toHaveLength(0)
  })
})

describe('drawGrappleLayer — deux vies du même slot', () => {
  /**
   * LA MUTATION QUE CE TEST TUE : reprendre la dernière piste du slot. La traction part de la
   * PREMIÈRE vie, qui file plein est ; la seconde vie est ailleurs et ne commence qu'après.
   * Avec l'ancienne jointure, `positionAt` rendrait `null` et RIEN ne serait tracé.
   */
  it('trace le câble depuis la position de la vie qui a tiré, pas de la vie suivante', () => {
    const fx = buildGrappleFx(
      docWith({
        tracks: [VIE_1, VIE_2] as unknown as ReplayDocumentReady['tracks'],
        grappleLines: [{ slot: 5, t0: 12, t1: 18, ax: 8, ay: 2 }],
      }),
    )
    const { ops, ctx } = recordingContext()
    drawGrappleLayer(ctx, fx, VIEW, 15, 'encre')
    // À t=15, la PREMIÈRE vie est à mi-chemin entre (2,2) et (4,2) : (3,2). La seconde, elle,
    // n'existe pas encore — et se trouverait de toute façon en (8,8).
    const p = worldToCanvas({ x: 3, y: 2 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    const a = worldToCanvas({ x: 8, y: 2 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    expect(ops.find((o) => o.op === 'moveTo')?.args).toEqual([p.x, p.y])
    expect(ops.find((o) => o.op === 'lineTo')?.args).toEqual([a.x, a.y])
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(1)
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
