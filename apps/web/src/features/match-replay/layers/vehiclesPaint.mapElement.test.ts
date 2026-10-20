/**
 * vehiclesPaint.mapElement.test.ts — l'ENCRE et le LIBELLÉ d'un élément de carte non jouable
 * (tourelle automatique bannie) face à la tourelle fixe jouable, sur le contexte enregistreur.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleSample } from '@/lib/api/types'

import { count, recordingContext, type CanvasOp } from '../test/recordingContext'
import type { FxInk } from './fxInk'
import type { PlacementView } from './placementShapes'
import type { ReplayVehicleRideReady, ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { drawVehiclesLayer, type VehicleStyle } from './vehiclesPaint'

const VIEW: PlacementView = {
  bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 },
  width: 400,
  height: 400,
  pad: 8,
}

const FX: FxInk = {
  tint: {
    kinetic: 'K', plasma_cool: 'PC', plasma_hot: 'PH', forerunner: 'F', electric: 'E', needle: 'N',
    blast: 'B', neutral: 'X',
  },
  core: 'C',
}

function track(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 700, gen: 1, t0: 0, t1: 1000, t1max: 1000, end: 'inconnue', family: 'warthog',
    samples: [{ t: 0, x: 50, y: 50, h: 90 } as ReplayVehicleSample], rides: [], ...over,
  }
}

function ride(over: Partial<ReplayVehicleRideReady> = {}): ReplayVehicleRideReady {
  return { t0: 0, t1: 100, slot: 1, src: 'event', aim: [], ...over }
}

function style(over: Partial<VehicleStyle> = {}): VehicleStyle {
  return {
    neutralInk: '#neutre',
    mapElementInk: '#gris-carte',
    labelStroke: '#contour',
    showNames: true,
    showAim: true,
    spriteOf: () => null,
    sizeOf: () => null,
    kindOf: () => undefined,
    labelOfFamily: () => null,
    colorOfSlot: () => '#equipe',
    colorOfXuid: () => null,
    nameOfSlot: () => 'PION-BRIDGE',
    nameOfXuid: () => null,
    explosionInk: FX,
    reducedMotion: false,
    offscreenLabelOf: (name, meters) => `${name} · ${Math.round(meters)} m`,
    offscreenGroupLabelOf: (n, meters) => `${n} joueurs · ${Math.round(meters)} m`,
    ...over,
  }
}

function paint(tracks: ReplayVehicleTrackReady[], st: VehicleStyle): CanvasOp[] {
  const { ops, ctx } = recordingContext()
  drawVehiclesLayer(ctx, tracks, VIEW, { frame: 50, k: 1, frameMs: 100 }, st)
  return ops
}

const paintedColors = (ops: CanvasOp[]): unknown[] =>
  ops
    .filter((o) => o.op === 'set fillStyle' || o.op === 'set strokeStyle' || o.op === 'addColorStop')
    .map((o) => (o.op === 'addColorStop' ? o.args[1] : o.args[0]))

const texts = (ops: CanvasOp[]): string[] =>
  ops.filter((o) => o.op === 'fillText').map((o) => String(o.args[0]))

describe('drawVehiclesLayer — encre et libellé d’un élément de carte', () => {
  const styleNomme = (over: Partial<VehicleStyle> = {}): VehicleStyle =>
    style({
      sizeOf: () => null,
      spriteOf: () => null,
      kindOf: () => 'map_element',
      labelOfFamily: () => 'Tourelle automatique bannie',
      ...over,
    })

  const tourelle = (): ReplayVehicleTrackReady =>
    track({ slot: 768, family: 'tourelle_auto_bannie', samples: [], spawn: { x: 40, y: 60 } })

  const tourelleFixe = (): ReplayVehicleTrackReady =>
    track({ slot: 769, family: 'tourelle_fixe', samples: [], spawn: { x: 40, y: 60 } })

  const styleFixe = (over: Partial<VehicleStyle> = {}): VehicleStyle =>
    styleNomme({ kindOf: () => 'fixed_turret', ...over })

  it('un ÉLÉMENT DE CARTE ne porte JAMAIS de libellé, calque des noms allumé', () => {
    expect(texts(paint([tourelle()], styleNomme()))).toEqual([])
  })

  it('calque des noms ÉTEINT : le pictogramme reste, aucun libellé', () => {
    const ops = paint([tourelle()], styleNomme({ showNames: false }))
    expect(texts(ops)).toEqual([])
    expect(count(ops, 'arc')).toBe(2)
  })

  it('un ÉLÉMENT DE CARTE se peint à l’encre grise, jamais à l’encre neutre ni à celle d’un camp', () => {
    const st = styleNomme()
    const couleurs = paintedColors(paint([tourelle()], st))
    expect(couleurs).toContain(st.mapElementInk)
    expect(couleurs).not.toContain(st.neutralInk)
    expect(couleurs).not.toContain('#equipe')
  })

  it('la TOURELLE FIXE vide garde son libellé du document, à l’encre neutre', () => {
    const st = styleFixe({ colorOfSlot: () => null })
    const ops = paint([tourelleFixe()], st)
    expect(texts(ops)).toEqual(['Tourelle automatique bannie'])
    expect(paintedColors(ops)).toContain(st.neutralInk)
    expect(paintedColors(ops)).not.toContain(st.mapElementInk)
  })

  it('la TOURELLE FIXE sans libellé publié : rien n’est écrit — jamais un littéral du calque', () => {
    expect(texts(paint([tourelleFixe()], styleFixe({ labelOfFamily: () => null })))).toEqual([])
  })

  it('la TOURELLE FIXE occupée prend la couleur de son occupant', () => {
    const st = styleFixe()
    const t = track({
      slot: 769, family: 'tourelle_fixe', samples: [], spawn: { x: 40, y: 60 },
      rides: [ride({ slot: 7, seat: 0 })],
    })
    const couleurs = paintedColors(paint([t], st))
    expect(couleurs).toContain('#equipe')
    expect(couleurs).not.toContain(st.mapElementInk)
  })

  it('un VÉHICULE garde ses noms d’occupants, jamais le nom de sa famille', () => {
    const ops = paint([track({ rides: [ride({ slot: 7, seat: 0 })] })],
      style({ labelOfFamily: () => 'NE DOIT PAS APPARAITRE' }))
    expect(texts(ops)).toEqual(['PION-BRIDGE'])
  })
})
