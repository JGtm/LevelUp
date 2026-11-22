/**
 * vehiclesPaint.antagonists.test.ts : LES TITRES D'ANTAGONISTE SUR LE NOM D'UN OCCUPANT DE VÉHICULE.
 *
 * Un joueur embarqué n'a plus de pion : son nom est écrit par le calque des véhicules, à l'écran (noms
 * empilés) et sur la flèche hors cadre. Ce nom porte les mêmes titres que celui d'un pion, avec la même
 * mise en page (`replayLabels`) et la même pulsation. L'occupant est désigné par la clé du film que
 * porte son épisode (`ride.xuid`), à défaut par son slot.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { SlotTitle } from '../model/antagonistTitles'
import { edgeMarkFor, OFFSCREEN_MARGIN_PX } from '../model/edgeClamp'
import { recordingContext, type CanvasOp } from '../test/recordingContext'
import type { ReplayVehicleRideReady, ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { createTitleIcons, type AntagonistMarks } from './antagonistMarks'
import type { PlacementView } from './placementShapes'
import { drawVehiclesLayer, type VehicleStyle } from './vehiclesPaint'

const VIEW: PlacementView = { bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 }, width: 400, height: 400, pad: 8 }
const SPRITE = { width: 128, height: 128 } as unknown as CanvasImageSource
const OFFSCREEN_WORLD = { x: 1000, y: 50 }
const MARK = edgeMarkFor(OFFSCREEN_WORLD, VIEW, OFFSCREEN_MARGIN_PX, 1)!
const NEMESIS: SlotTitle = { role: 'nemesis', count: 3 }

function track(over: Partial<ReplayVehicleTrackReady> = {}, at = { x: 50, y: 50 }): ReplayVehicleTrackReady {
  return {
    slot: 700, gen: 1, t0: 0, t1: 1000, t1max: 1000, end: 'inconnue', family: 'warthog',
    samples: [{ t: 0, x: at.x, y: at.y, h: 90 }], rides: [], ...over,
  }
}
const ride = (over: Partial<ReplayVehicleRideReady> = {}): ReplayVehicleRideReady =>
  ({ t0: 0, t1: 100, slot: 1, src: 'event', aim: [], ...over })

function marques(
  titlesOfKey: (key: string, frame: number) => readonly SlotTitle[],
  titlesOfSlot: (slot: number, frame: number) => readonly SlotTitle[] = () => [],
): AntagonistMarks {
  const icons = createTitleIcons()
  icons.setSource('nemesis', { naturalWidth: 714, naturalHeight: 670 } as unknown as HTMLImageElement)
  icons.setSource('victim', { naturalWidth: 1099, naturalHeight: 1369 } as unknown as HTMLImageElement)
  return { titlesOfSlot, titlesOfKey, icons }
}

function style(over: Partial<VehicleStyle> = {}): VehicleStyle {
  return {
    neutralInk: '#neutre', mapElementInk: '#gris-carte', labelStroke: '#contour', showNames: true, showAim: true,
    spriteOf: () => SPRITE,
    sizeOf: () => ({ naturalWidthPx: 128, naturalHeightPx: 128, mmPerPx: 10 }),
    kindOf: () => undefined, labelOfFamily: () => null,
    colorOfSlot: () => '#equipe', colorOfXuid: () => null,
    nameOfSlot: () => 'Brute', nameOfXuid: () => null,
    explosionInk: { tint: {}, core: '' } as unknown as VehicleStyle['explosionInk'],
    reducedMotion: false,
    offscreenLabelOf: (name, meters) => `${name} · ${Math.round(meters)} m`,
    offscreenGroupLabelOf: (n, meters) => `${n} joueurs · ${Math.round(meters)} m`,
    ...over,
  }
}

function paint(tracks: ReplayVehicleTrackReady[], st: VehicleStyle): CanvasOp[] {
  const { ops, ctx } = recordingContext({ charWidth: 6 })
  drawVehiclesLayer(ctx, tracks, VIEW, { frame: 50, k: 1, frameMs: 100 }, st)
  return ops
}

beforeEach(() => {
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => recordingContext().ctx as never)
})
afterEach(() => vi.restoreAllMocks())

/** Les icônes de titre (images autres que le sprite du véhicule) et les textes, dans l'ordre d'écriture. */
const ordre = (ops: CanvasOp[]) =>
  ops
    .filter((o) => o.op === 'fillText' || (o.op === 'drawImage' && o.args[0] !== SPRITE))
    .map((o) => (o.op === 'drawImage' ? 'icone' : o.args[0]))
const arcs = (ops: CanvasOp[]) => ops.filter((o) => o.op === 'arc').length
const largeurIcone = (ops: CanvasOp[]) =>
  (ops.filter((o) => o.op === 'drawImage' && o.args[0] !== SPRITE)[0].args as number[])[3]

describe('le nom d’un occupant de véhicule porte ses titres', () => {
  it('à l’écran : icône et compte devant le nom, titres lus par la clé du film de l’épisode', () => {
    const titlesOfKey = vi.fn((key: string) => (key === 'e1' ? [NEMESIS] : []))
    const ops = paint([track({ rides: [ride({ slot: 7, seat: 0, xuid: 'e1' })] })], style({ antagonists: marques(titlesOfKey) }))
    expect(ordre(ops)).toEqual(['icone', '3', 'Brute'])
    expect(titlesOfKey).toHaveBeenCalledWith('e1', 50)
  })

  it('un épisode sans clé : le titre se lit par le slot de l’occupant', () => {
    const titlesOfSlot = vi.fn((slot: number) => (slot === 7 ? [NEMESIS] : []))
    const ops = paint([track({ rides: [ride({ slot: 7, seat: 0 })] })], style({ antagonists: marques(() => [], titlesOfSlot) }))
    expect(ordre(ops)).toEqual(['icone', '3', 'Brute'])
    expect(titlesOfSlot).toHaveBeenCalledWith(7, 50)
  })

  it('chaque occupant porte les siens : deux lignes, un seul antagoniste', () => {
    const ops = paint(
      [track({ rides: [ride({ slot: 7, seat: 0, xuid: 'e1' }), ride({ slot: 8, seat: 1, xuid: 'a1' })] })],
      style({ antagonists: marques((key) => (key === 'e1' ? [NEMESIS] : [])), nameOfSlot: (slot) => (slot === 7 ? 'Brute' : 'Allie') }),
    )
    expect(ordre(ops)).toEqual(['icone', '3', 'Brute', 'Allie'])
  })

  it('hors cadre : l’étiquette de la flèche porte l’icône et le compte', () => {
    const ops = paint(
      [track({ rides: [ride({ slot: 7, seat: 0, xuid: 'e1' })] }, OFFSCREEN_WORLD)],
      style({ antagonists: marques(() => [NEMESIS]) }),
    )
    expect(ordre(ops)).toEqual(['icone', '3', `Brute · ${Math.round(MARK.distanceM)} m`])
  })

  it('hors cadre, plusieurs occupants : le conducteur nommé porte ses titres', () => {
    const ops = paint(
      [track({ rides: [ride({ slot: 8, seat: 1, xuid: 'a1' }), ride({ slot: 7, seat: 0, xuid: 'e1' })] }, OFFSCREEN_WORLD)],
      style({ antagonists: marques((key) => (key === 'e1' ? [NEMESIS] : [])), nameOfSlot: (slot) => (slot === 7 ? 'Brute' : 'Allie') }),
    )
    expect(ordre(ops)).toEqual(['icone', '3', `Brute · ${Math.round(MARK.distanceM)} m`])
  })

  it('hors cadre, conducteur sans nom : le compte « N joueurs » ne porte aucun titre', () => {
    const ops = paint(
      [track({ rides: [ride({ slot: 8, seat: 1, xuid: 'e2' }), ride({ slot: 7, seat: 0, xuid: 'e1' })] }, OFFSCREEN_WORLD)],
      style({ antagonists: marques(() => [NEMESIS]), nameOfSlot: () => null }),
    )
    expect(ordre(ops)).toEqual([`2 joueurs · ${Math.round(MARK.distanceM)} m`])
  })

  it.each([
    ['à l’écran', { x: 50, y: 50 }],
    ['hors cadre', OFFSCREEN_WORLD],
  ])('un occupant sans titre : appels de dessin identiques à ceux d’avant (%s)', (_nom, at) => {
    const t = [track({ rides: [ride({ slot: 7, seat: 0, xuid: 'a1' })] }, at)]
    const avant = JSON.stringify(paint(t, style()))
    expect(JSON.stringify(paint(t, style({ antagonists: marques(() => []) })))).toBe(avant)
  })

  it('le nouveau tenant à bord pulse : deux anneaux autour du véhicule et l’icône qui grossit', () => {
    const t = [track({ rides: [ride({ slot: 7, seat: 0, xuid: 'e1' })] })]
    const repos = paint(t, style({ antagonists: marques(() => [NEMESIS]) }))
    const pulse = paint(t, style({ antagonists: marques(() => [{ ...NEMESIS, motion: { kind: 'gain', progress: 0.5 } }]) }))
    expect(arcs(pulse) - arcs(repos)).toBe(2)
    expect(largeurIcone(pulse)).toBeGreaterThan(largeurIcone(repos))
  })

  it('un occupant masqué ne porte rien', () => {
    const ops = paint(
      [track({ rides: [ride({ slot: 7, seat: 0, xuid: 'e1' })] })],
      style({
        antagonists: marques(() => [NEMESIS]),
        shown: { vehicleAt: () => true, occupant: () => false, vehicleDestruction: () => true },
      }),
    )
    expect(ordre(ops)).toEqual([])
  })
})
