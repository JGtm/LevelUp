/**
 * markerOutline.test.ts — LE CERNE DEUX TONS DU PION, vérifié sans navigateur (contexte enregistreur).
 *
 * Ce que ce fichier garde :
 *   - le cerne se peint AVANT le noyau, filet clair d'abord (un anneau, rempli en pair-impair) puis
 *     bande sombre, aux épaisseurs nommées de `markerOutline.ts` ; le noyau garde la couleur d'équipe ;
 *   - les encres du cerne viennent du style (jetons), jamais d'un littéral ;
 *   - rien ne se superpose au cerne : ni les anneaux d'étage, ni le double contour du joueur de la
 *     page ne se posent sur le filet clair, et les deux familles d'anneaux ne se mêlent pas.
 * Le contraste des deux encres contre les fonds de carte est gardé par
 * `lib/accessibility/mapReliefTokens.test.ts`.
 */
import { describe, expect, it } from 'vitest'

import {
  FLOOR_RING_CLEARANCE_PX,
  MARKER_BAND_PX,
  MARKER_OUTLINE_PX,
  MARKER_RIM_OVERLAP_PX,
  MARKER_RIM_PX,
  firstFloorRingRadius,
  markerCorePx,
  markerOuterEdge,
  markerOutlineRadius,
} from './markerOutline'
import { drawTracksLayer, type MarkerStyle } from './replayMarkers'
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import type { ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import { recordingContext, type CanvasOp } from '../test/recordingContext'
import { CORE_RADIUS, OUTLINE_PAD } from '@/lib/replay/screenSizes'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 200, height: 100, pad: 4 }

const BAND = 'bande-sombre'
const RIM = 'filet-clair'
const CORE = 'couleur-equipe'

/** Les trois étages d'une plage `{ min: 0, max: 10 }` : z = 0, 5 et 10. */
const Z_OF_FLOOR = [0, 5, 10] as const

function track(z: number): ReplayTrackReady {
  return {
    slot: 512,
    team: -1,
    xuid: 'A',
    startFrame: 0,
    endFrame: 100,
    points: [{ t: 50, x: 5, y: 5, z, h: 90 }],
  }
}

function paint(floor: 0 | 1 | 2, mark?: PlayerMarkKind): CanvasOp[] {
  const style: MarkerStyle = {
    colorOfSlot: () => CORE,
    outline: { band: BAND, rim: RIM },
    frame: 50,
    timing: { trail: 30, aimHold: 60, death: 20, spawn: 5 },
    z: { min: 0, max: 10 },
    k: 1,
    showAim: false,
    markOfSlot: () => mark,
    nameOfSlot: () => null,
    showTrail: false,
    selfInk: 'moi',
    deathInk: 'mort',
    labelStroke: 'nom',
    offscreenLabelOf: (name) => name,
  }
  const { ops, ctx } = recordingContext()
  drawTracksLayer(ctx, [track(Z_OF_FLOOR[floor])], VIEW, style)
  return ops
}

/** Les rayons des arcs émis dans le chemin ouvert par chaque `beginPath`, à chaque `fill`. */
function fills(ops: CanvasOp[]): Array<{ style: unknown; rule: unknown; radii: number[]; alpha: number }> {
  const out: Array<{ style: unknown; rule: unknown; radii: number[]; alpha: number }> = []
  let style: unknown
  let alpha = 1
  let radii: number[] = []
  for (const o of ops) {
    if (o.op === 'set fillStyle') style = o.args[0]
    else if (o.op === 'set globalAlpha') alpha = o.args[0] as number
    else if (o.op === 'beginPath') radii = []
    else if (o.op === 'arc') radii.push(o.args[2] as number)
    else if (o.op === 'fill') out.push({ style, rule: o.args[0], radii: [...radii], alpha })
  }
  return out
}

/** Les anneaux TRACÉS (stroke) : rayon de la ligne médiane et épaisseur, dans l'ordre d'émission. */
function strokedRings(ops: CanvasOp[]): Array<{ r: number; w: number }> {
  const out: Array<{ r: number; w: number }> = []
  let width = 1
  let r = 0
  for (const o of ops) {
    if (o.op === 'set lineWidth') width = o.args[0] as number
    else if (o.op === 'arc') r = o.args[2] as number
    else if (o.op === 'stroke') out.push({ r, w: width })
  }
  return out
}

describe('le cerne deux tons — géométrie nommée', () => {
  it('la bande garde le liseré historique, le filet s’y ajoute vers l’extérieur', () => {
    expect(MARKER_BAND_PX).toBe(OUTLINE_PAD)
    expect(MARKER_OUTLINE_PX).toBeCloseTo(MARKER_BAND_PX + MARKER_RIM_PX, 12)
    expect(markerCorePx(0)).toBe(CORE_RADIUS)
    expect(markerOutlineRadius(0)).toBeCloseTo(CORE_RADIUS + MARKER_OUTLINE_PX, 12)
  })

  it('le noyau grossit d’un cran par étage, le cerne le suit', () => {
    expect(markerCorePx(2) - markerCorePx(0)).toBeCloseTo(1.4, 12)
    expect(markerOutlineRadius(2) - markerCorePx(2)).toBeCloseTo(MARKER_OUTLINE_PX, 12)
  })
})

describe('le cerne deux tons — ce que le pion émet', () => {
  it('peint le filet (anneau pair-impair), puis la bande, puis le noyau', () => {
    const f = fills(paint(0))
    expect(f.map((x) => x.style)).toEqual([RIM, BAND, CORE])
    expect(f[0].rule).toBe('evenodd')
    expect(f[1].rule).toBeUndefined()
    // Filet : du bord extérieur du cerne jusque sous la bande (recouvrement anticrénelage).
    expect(f[0].radii).toEqual([CORE_RADIUS + MARKER_OUTLINE_PX, CORE_RADIUS + MARKER_BAND_PX - MARKER_RIM_OVERLAP_PX])
    expect(f[1].radii).toEqual([CORE_RADIUS + MARKER_BAND_PX])
    expect(f[2].radii).toEqual([CORE_RADIUS])
  })

  it('est opaque pour un pion plein, et le noyau garde la couleur d’équipe', () => {
    const f = fills(paint(0))
    expect(f.map((x) => x.alpha)).toEqual([1, 1, 1])
    expect(f[2].style).toBe(CORE)
  })

  it.each([1, 2] as const)('suit le rayon du noyau à l’étage %i', (floor) => {
    const [rim, band, core] = fills(paint(floor))
    expect(rim.radii[0]).toBeCloseTo(markerOutlineRadius(floor), 12)
    expect(band.radii[0]).toBeCloseTo(markerCorePx(floor) + MARKER_BAND_PX, 12)
    expect(core.radii[0]).toBeCloseTo(markerCorePx(floor), 12)
  })
})

describe('anneaux d’étage et double contour — jamais sur le filet', () => {
  it('le premier anneau d’étage est détaché du cerne de FLOOR_RING_CLEARANCE_PX', () => {
    for (const floor of [1, 2] as const) {
      expect(firstFloorRingRadius(floor) - markerOutlineRadius(floor)).toBeCloseTo(FLOOR_RING_CLEARANCE_PX, 12)
    }
  })

  it.each([0, 1, 2] as const)('étage %i : aucun trait ne chevauche le cerne ni un autre trait (pion du joueur de la page)', (floor) => {
    const rings = strokedRings(paint(floor, 'me')).sort((a, b) => a.r - b.r)
    const outer = markerOutlineRadius(floor)
    let edge = outer
    for (const { r, w } of rings) {
      // Un jour franc (≥ 0,4 px) entre le trait et ce qui le précède : le filet, ou l'anneau d'à côté.
      expect(r - w / 2 - edge, `anneau r=${r} (largeur ${w}) à l’étage ${floor}`).toBeGreaterThanOrEqual(0.4)
      edge = r + w / 2
    }
    // Les anneaux d'étage (un par étage) puis le double contour (deux) : tout est là.
    expect(rings).toHaveLength(floor + 2)
  })

  it.each([0, 1, 2] as const)('étage %i : le double contour part du bord extérieur du pion', (floor) => {
    const rings = strokedRings(paint(floor, 'me')).sort((a, b) => a.r - b.r)
    const selfFirst = rings[floor]
    expect(selfFirst.r - selfFirst.w / 2).toBeGreaterThan(markerOuterEdge(floor))
  })
})
