/**
 * Le calque des véhicules honore Seul et Masquer : le véhicule d'un joueur écarté ne se dessine pas, ses
 * occupants écartés ne s'écrivent pas, son explosion de destruction non plus. Le prédicat arrive par le style.
 */
import { describe, expect, it } from 'vitest'

import type { FxInk } from './fxInk'
import { count, recordingContext, type CanvasOp } from '../test/recordingContext'
import type { PlacementView } from './placementShapes'
import type { ReplayVehicleRideReady, ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { drawVehiclesLayer, type VehicleStyle } from './vehiclesPaint'

const VIEW: PlacementView = { bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 }, width: 400, height: 400, pad: 8 }
const SPRITE = { width: 128, height: 128 } as unknown as CanvasImageSource
const INK = { tint: { blast: 'b', neutral: 'n', plasma_hot: 'p' }, core: 'c' } as unknown as FxInk

function track(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 700, gen: 1, t0: 0, t1: 1000, t1max: 1000, end: 'inconnue', family: 'warthog',
    samples: [{ t: 0, x: 50, y: 50, h: 90 }], rides: [], ...over,
  }
}

const ride = (slot: number, seat: number): ReplayVehicleRideReady => ({ t0: 0, t1: 100, slot, seat, src: 'event', aim: [] })

function style(shown?: VehicleStyle['shown']): VehicleStyle {
  return {
    neutralInk: '#n', mapElementInk: '#g', labelStroke: '#c', showNames: true, showAim: true,
    spriteOf: () => SPRITE,
    sizeOf: () => ({ naturalWidthPx: 128, naturalHeightPx: 128, mmPerPx: 10 }),
    kindOf: () => undefined, labelOfFamily: () => null,
    colorOfSlot: () => '#equipe', colorOfXuid: () => null,
    nameOfSlot: (slot) => `N${slot}`, nameOfXuid: () => null,
    explosionInk: INK, reducedMotion: false,
    offscreenLabelOf: (n) => n, offscreenGroupLabelOf: (n) => String(n),
    shown,
  }
}

function paint(tracks: ReplayVehicleTrackReady[], st: VehicleStyle, frame = 50): CanvasOp[] {
  const { ops, ctx } = recordingContext()
  drawVehiclesLayer(ctx, tracks, VIEW, { frame, k: 1, frameMs: 100 }, st)
  return ops
}

const texts = (ops: CanvasOp[]) => ops.filter((o) => o.op === 'fillText').map((o) => String(o.args[0]))
const tout = { vehicleAt: () => true, occupant: () => true, vehicleDestruction: () => true }

describe('drawVehiclesLayer — Seul et Masquer', () => {
  it('un véhicule retiré ne laisse ni sprite, ni nom', () => {
    const t = track({ rides: [ride(1, 0)] })
    expect(count(paint([t], style(tout)), 'drawImage')).toBe(1)
    const ops = paint([t], style({ ...tout, vehicleAt: () => false }))
    expect(count(ops, 'drawImage')).toBe(0)
    expect(texts(ops)).toEqual([])
  })

  it('le prédicat de véhicule reçoit la vie et l’image courante', () => {
    const t = track({ rides: [ride(1, 0)] })
    const vus: unknown[] = []
    paint([t], style({ ...tout, vehicleAt: (v, f) => { vus.push([v.slot, f]); return true } }), 42)
    expect(vus).toEqual([[700, 42]])
  })

  it('un occupant retiré n’est pas écrit, l’autre reste', () => {
    const t = track({ rides: [ride(1, 0), ride(2, 1)] })
    const ops = paint([t], style({ ...tout, occupant: (r) => r.slot !== 2 }))
    expect(texts(ops)).toEqual(['N1'])
    expect(count(ops, 'drawImage')).toBe(1)
  })

  it('l’explosion de destruction suit son propre prédicat', () => {
    const detruit = track({ end: 'destroyed', tEnd: 50, rides: [] })
    const avec = paint([detruit], style(tout), 50)
    const sans = paint([detruit], style({ ...tout, vehicleDestruction: () => false }), 50)
    expect(avec.length).toBeGreaterThan(sans.length)
  })
})
