/**
 * replayMarkers.gaps.test.ts — LE PION PENDANT UNE LACUNE DE RÉPLICATION : tenu, et PÂLI.
 *
 * RETOURS DU REJEU 2026-09-23 (lot L1.4, décision Q14 « tenu, pâli »). Entre deux points dont le
 * second porte `g` (le film n'a rien répliqué), le pion reste à sa dernière position connue et
 * se dessine pâli : il dit « on ne sait pas où il est depuis », pas « il est là ». Aucun segment
 * de traînée ne relie les deux côtés de la lacune, et aucun cône de visée ne s'y dessine (la
 * visée non plus n'est pas répliquée).
 */
import { describe, expect, it } from 'vitest'

import { drawTracksLayer, type MarkerStyle } from './replayMarkers'
import type { ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import { count, recordingContext, type CanvasOp } from '../test/recordingContext'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 }, width: 200, height: 200, pad: 4 }

/** Vie 0 → 100 : continue jusqu'à 40, lacune de 6 s, reprise au point de 80 (loin). */
const TRACK: ReplayTrackReady = {
  slot: 512,
  team: -1,
  xuid: 'A',
  startFrame: 0,
  endFrame: 100,
  points: [
    { t: 0, x: 10, y: 10, z: 0, h: 0 },
    { t: 40, x: 12, y: 10, z: 0, h: 0 },
    { t: 80, x: 90, y: 90, z: 0, h: 0, g: 6000 },
    { t: 90, x: 91, y: 90, z: 0, h: 0 },
  ],
}

function style(frame: number): MarkerStyle {
  return {
    colorOfSlot: () => 'rgb(1 2 3)',
    ink: 'rgb(9 9 9)',
    frame,
    timing: { trail: 200, aimHold: 600, death: 20, spawn: 5 },
    z: { min: 0, max: 10 },
    k: 1,
    showAim: true,
    markOfSlot: () => undefined,
    nameOfSlot: () => null,
    showTrail: true,
    selfInk: 'rgb(4 4 4)',
    deathInk: 'rgb(5 5 5)',
    labelStroke: 'rgb(8 12 18)',
    offscreenLabelOf: (name) => name,
  }
}

function paint(frame: number): CanvasOp[] {
  const { ops, ctx } = recordingContext()
  drawTracksLayer(ctx, [TRACK], VIEW, style(frame))
  return ops
}

/** L'opacité en vigueur à chaque `fill` (le noyau et son liseré). */
function alphasAtFill(ops: CanvasOp[]): number[] {
  let alpha = 1
  const out: number[] = []
  for (const o of ops) {
    if (o.op === 'set globalAlpha') alpha = Number(o.args[0])
    if (o.op === 'fill') out.push(alpha)
  }
  return out
}

/** Les abscisses (toile) des `lineTo` émis : la traînée, segment par segment. */
function lineToXs(ops: CanvasOp[]): number[] {
  return ops.filter((o) => o.op === 'lineTo').map((o) => Number(o.args[0]))
}

describe('drawTracksLayer — le pion pendant une lacune (Q14 : tenu, pâli)', () => {
  it('hors lacune, le noyau est plein', () => {
    expect(Math.max(...alphasAtFill(paint(20)))).toBe(1)
  })

  it('pendant la lacune, le noyau ET son liseré sont pâlis', () => {
    const alphas = alphasAtFill(paint(60))
    expect(alphas.length).toBeGreaterThan(0)
    expect(Math.max(...alphas)).toBeLessThan(1)
  })

  it('pendant la lacune, le pion est à sa dernière position (x = 12 dans le monde), sans cône', () => {
    const tenu = paint(60)
    const avant = paint(40)
    // Même position à l'écran qu'au dernier point avant la lacune : les arcs du noyau le disent.
    const arcs = (ops: CanvasOp[]) => ops.filter((o) => o.op === 'arc').map((o) => [o.args[0], o.args[1]])
    expect(arcs(tenu)[0]).toEqual(arcs(avant)[0])
    expect(count(tenu, 'createRadialGradient')).toBe(0)
  })

  it('après la lacune, aucun segment de traînée ne relie les deux côtés', () => {
    const ops = paint(90)
    // Toutes les extrémités de segment sont du côté de la reprise (x monde ≥ 90, toile > 150).
    for (const x of lineToXs(ops)) expect(x).toBeGreaterThan(150)
  })
})
