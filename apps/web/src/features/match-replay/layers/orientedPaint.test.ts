/**
 * orientedPaint.test.ts — LE CÂBLAGE DE LA ROTATION DANS LES CALQUES DES VÉHICULES : le sprite,
 * le pictogramme d'un élément de carte et le cône d'un occupant se tournent avec la carte, quand
 * `drawVehiclesLayer` lit l'orientation sur le cadrage. Un `turns` retiré de `vehiclesPaint.ts`
 * (sprite, tourelle, cône) laisse l'angle à celui d'une carte non tournée : ces cas échouent.
 *
 * L'oracle est le déplacement projeté par le cadrage tourné (`test/screenDirection`).
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleSample } from '@/lib/api/types'
import type { ReplayVehicleRideReady, ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { QUARTER_TURNS, viewOrientation, type QuarterTurns } from '@/lib/replay/orientation'
import type { CanvasView } from '../model/replayView'
import { recordingContext } from '../test/recordingContext'
import { sameDirection, screenDirection } from '../test/screenDirection'
import type { FxInk } from './fxInk'
import { drawVehiclesLayer, type VehicleStyle } from './vehiclesPaint'

const SCENE = { minX: 0, minY: 0, maxX: 100, maxY: 100 }
const AT = { x: 50, y: 50 }
const FX: FxInk = {
  tint: { kinetic: 'K', plasma_cool: 'PC', plasma_hot: 'PH', forerunner: 'F', electric: 'E', needle: 'N', blast: 'B', neutral: 'X' },
  core: 'C',
}

function viewAt(turns: QuarterTurns): CanvasView {
  return { bounds: SCENE, width: 400, height: 400, pad: 8, orient: viewOrientation(turns, SCENE) }
}

function track(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 700, gen: 1, t0: 0, t1: 1000, t1max: 1000, end: 'inconnue', family: 'warthog',
    samples: [{ t: 0, x: AT.x, y: AT.y, h: 90 } as ReplayVehicleSample], rides: [], ...over,
  }
}

function style(over: Partial<VehicleStyle> = {}): VehicleStyle {
  return {
    neutralInk: '#neutre', mapElementInk: '#gris', labelStroke: '#contour', showNames: false, showAim: true,
    spriteOf: () => null, sizeOf: () => null, kindOf: () => undefined, labelOfFamily: () => null,
    colorOfSlot: () => '#equipe', colorOfXuid: () => null, nameOfSlot: () => null, nameOfXuid: () => null,
    explosionInk: FX, reducedMotion: false,
    offscreenLabelOf: () => '', offscreenGroupLabelOf: () => '',
    ...over,
  }
}

function paint(view: CanvasView, tracks: ReplayVehicleTrackReady[], st: VehicleStyle) {
  const { ops, ctx } = recordingContext()
  drawVehiclesLayer(ctx, tracks, view, { frame: 50, k: 1, frameMs: 100 }, st)
  return ops
}

/** La direction du nez d'un sprite dessiné nez en haut puis tourné de `angle` (canevas). */
const noseDirection = (angle: number) => Math.atan2(-Math.cos(angle), Math.sin(angle))

describe('drawVehiclesLayer — la carte tournée', () => {
  it.each(QUARTER_TURNS)('le sprite d’un véhicule (cap nord) pointe comme le terrain, à %i quart(s) de tour', (turns) => {
    const view = viewAt(turns)
    const image = { width: 20, height: 40 } as unknown as CanvasImageSource
    const ops = paint(view, [track()], style({
      spriteOf: () => image,
      sizeOf: () => ({ naturalWidthPx: 20, naturalHeightPx: 40, mmPerPx: 50 }),
    }))
    const rotate = ops.find((o) => o.op === 'rotate')
    expect(rotate).toBeDefined()
    expect(sameDirection(noseDirection(rotate!.args[0] as number), screenDirection(view, 90, AT))).toBe(true)
  })

  it.each(QUARTER_TURNS)('le pictogramme d’un élément de carte (cap par défaut) se tourne aussi, à %i quart(s) de tour', (turns) => {
    const view = viewAt(turns)
    const tourelle = track({ slot: 768, family: 'tourelle_auto_bannie', samples: [], spawn: { x: AT.x, y: AT.y } })
    const ops = paint(view, [tourelle], style({ kindOf: () => 'map_element' }))
    const rotate = ops.find((o) => o.op === 'rotate')
    expect(rotate).toBeDefined()
    // Cap par défaut du film : le nord (90°).
    expect(sameDirection(noseDirection(rotate!.args[0] as number), screenDirection(view, 90, AT))).toBe(true)
  })

  it.each(QUARTER_TURNS)('le cône d’un occupant suit sa visée mesurée, à %i quart(s) de tour', (turns) => {
    const view = viewAt(turns)
    const ride = { t0: 0, t1: 100, slot: 7, src: 'event', seat: 0, aim: [{ t: 50, h: 30 }] } as unknown as ReplayVehicleRideReady
    const ops = paint(view, [track({ rides: [ride] })], style())
    const secteurs = ops.filter((o) => o.op === 'arc' && (o.args[2] as number) > 30)
    expect(secteurs).toHaveLength(1)
    const [, , , start, end] = secteurs[0].args as number[]
    expect(sameDirection((start + end) / 2, screenDirection(view, 30, AT))).toBe(true)
  })
})
