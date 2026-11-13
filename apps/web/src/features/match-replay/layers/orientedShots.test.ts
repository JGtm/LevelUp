/**
 * orientedShots.test.ts — LE CÂBLAGE DE LA ROTATION DANS LES ÉCLAIRS DE BOUCHE : `drawShotsLayer`
 * lit l'orientation sur le cadrage et la passe à `vehicleShotOrigin`. Un `turns` retiré de
 * `replayDraw.ts` laisse la direction de l'éclair à celle d'une carte non tournée : ces cas échouent.
 * Le peintre d'éclair est remplacé par un témoin qui garde sa forme (position, angle).
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { ShotFxEntry } from '../model/shotFx'
import { QUARTER_TURNS, viewOrientation, type QuarterTurns } from '../model/orientation'
import { projectTo, type CanvasView } from '../model/replayView'
import { recordingContext } from '../test/recordingContext'
import { sameDirection, screenDirection } from '../test/screenDirection'
import { drawMuzzleFlash } from './muzzleFlash'
import { drawShotsLayer } from './replayDraw'

vi.mock('./muzzleFlash', () => ({ drawMuzzleFlash: vi.fn() }))

const SCENE = { minX: 0, minY: 0, maxX: 100, maxY: 100 }
const AT = { x: 50, y: 50 }
const INK = { tint: { kinetic: 'K', plasma_cool: 'K', plasma_hot: 'K', forerunner: 'K', electric: 'K', needle: 'K', blast: 'K', neutral: 'K' }, core: 'C' }

function viewAt(turns: QuarterTurns): CanvasView {
  return { bounds: SCENE, width: 400, height: 400, pad: 8, orient: viewOrientation(turns, SCENE) }
}

type SizeOf = () => { naturalWidthPx: number; naturalHeightPx: number; mmPerPx: number }

function run(view: CanvasView, shot: Partial<ShotFxEntry>, sizeOf?: SizeOf) {
  const { ctx } = recordingContext()
  const entry = { frame: 10, slot: 5, x: AT.x, y: AT.y, h: null, fam: 'bullet', tint: 'kinetic', seed: 3, vehicleShot: null, ...shot } as unknown as ShotFxEntry
  drawShotsLayer(ctx, [entry], view, { frame: 11, hold: 5, frameMs: 33 }, {
    ink: INK as never, k: 1, reducedMotion: false, vehicleSizeOf: sizeOf as never,
  })
  return vi.mocked(drawMuzzleFlash).mock.calls.at(-1)![3]
}

beforeEach(() => vi.mocked(drawMuzzleFlash).mockClear())

describe('drawShotsLayer — la carte tournée', () => {
  it.each(QUARTER_TURNS)('l’éclair d’un tir à pied suit le regard du terrain, à %i quart(s) de tour', (turns) => {
    const view = viewAt(turns)
    const shape = run(view, { h: 30 })
    expect(shape.angle).not.toBeNull()
    expect(sameDirection(shape.angle as number, screenDirection(view, 30, AT))).toBe(true)
  })

  it.each(QUARTER_TURNS)('l’éclair d’une arme de véhicule : direction et bouche suivent le cap, à %i quart(s) de tour', (turns) => {
    const view = viewAt(turns)
    const shape = run(
      view,
      {
        vehicleShot: {
          mount: { classe: 'fixe', ax: 0, ay: -0.5 }, arme: 'vehicule', family: 'warthog', headingDeg: 40, shooterHeadingDeg: null,
        } as never,
      },
      () => ({ naturalWidthPx: 100, naturalHeightPx: 200, mmPerPx: 10 }),
    )
    const centre = projectTo(view, AT)
    expect(sameDirection(shape.angle as number, screenDirection(view, 40, AT))).toBe(true)
    expect(sameDirection(Math.atan2(shape.y - centre.y, shape.x - centre.x), screenDirection(view, 40, AT))).toBe(true)
  })
})
