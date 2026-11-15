/**
 * orientedHeadings.test.ts — LES CAPS TOURNENT AVEC LA CARTE.
 *
 * Une position monde tourne par `projectTo` ; un CAP (le regard d'un joueur, le nez d'un véhicule,
 * la bouche d'une arme) est un angle que le dessin convertit lui-même, sans passer par la
 * projection : il faut lui ajouter la rotation de la carte. L'oracle de ces cas ne réécrit aucune
 * formule d'angle : il fait avancer un point de quelques mètres le long du cap, le projette par le
 * cadrage tourné, et exige que le dessin pointe exactement dans la direction de ce déplacement à
 * l'écran — pour les quatre orientations et plusieurs caps.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayBounds } from '@/lib/api/types'

import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import type { ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import { QUARTER_TURNS, viewOrientation, type QuarterTurns } from '@/lib/replay/orientation'
import type { CanvasView } from '../model/replayView'
import { vehicleAimAngle, vehicleScreenAngle } from '../model/vehiclesLayer'
import { vehicleShotOrigin, type VehicleWeaponMountEntry } from '../model/vehicleWeaponMounts'
import { recordingContext } from '../test/recordingContext'
import { sameDirection, screenDirection as direction } from '../test/screenDirection'
import { drawTracksLayer, type MarkerStyle } from './replayMarkers'

const SCENE: ReplayBounds = { minX: 0, minY: 0, maxX: 40, maxY: 20 }
const HEADINGS = [0, 30, 90, 137, 180, 250, 315]
const ORIGIN = { x: 17, y: 9 }

function viewAt(turns: QuarterTurns): CanvasView {
  return { bounds: SCENE, width: 800, height: 800, pad: 20, orient: viewOrientation(turns, SCENE) }
}

const screenDirection = (view: CanvasView, h: number) => direction(view, h, ORIGIN)

describe('les caps suivent la rotation de la carte', () => {
  it.each(QUARTER_TURNS)('cône de visée d’un pion, à %i quart(s) de tour', (turns) => {
    for (const h of HEADINGS) {
      const view = viewAt(turns)
      const track: ReplayTrackReady = {
        slot: 1, team: -1, xuid: 'A', startFrame: 0, endFrame: 100,
        points: [{ t: 50, x: ORIGIN.x, y: ORIGIN.y, z: 0, h }],
      }
      const style: MarkerStyle = {
        colorOfSlot: () => 'rgb(1 2 3)', ink: 'rgb(9 9 9)', frame: 50,
        timing: { trail: 30, aimHold: 60, death: 20, spawn: 5 }, z: { min: 0, max: 10 }, k: 1,
        showAim: true, markOfSlot: (): PlayerMarkKind | undefined => undefined, nameOfSlot: () => null,
        showTrail: false, selfInk: 'rgb(4 4 4)', deathInk: 'rgb(5 5 5)', labelStroke: 'rgb(8 12 18)',
        offscreenLabelOf: () => '',
      }
      const { ops, ctx } = recordingContext()
      drawTracksLayer(ctx, [track], view, style)
      // Le secteur de visée est le seul arc de grand rayon (52 px) ; son axe est le milieu de l'arc.
      const arcs = ops.filter((o) => o.op === 'arc' && (o.args[2] as number) > 30)
      expect(arcs).toHaveLength(1)
      const [, , , start, end] = arcs[0].args as number[]
      expect(sameDirection((start + end) / 2, screenDirection(view, h))).toBe(true)
    }
  })

  it.each(QUARTER_TURNS)('sprite de véhicule (nez en haut) et cône de véhicule, à %i quart(s) de tour', (turns) => {
    const view = viewAt(turns)
    for (const h of HEADINGS) {
      const dir = screenDirection(view, h)
      // Le sprite est dessiné nez en haut : après `rotate(a)`, le nez pointe vers (sin a, -cos a).
      const a = vehicleScreenAngle(h, turns)
      expect(sameDirection(Math.atan2(-Math.cos(a), Math.sin(a)), dir)).toBe(true)
      // Le cône est tracé à l'angle nu.
      expect(sameDirection(vehicleAimAngle(h, turns), dir)).toBe(true)
    }
  })

  it.each(QUARTER_TURNS)('direction et bouche d’une arme de véhicule, à %i quart(s) de tour', (turns) => {
    const view = viewAt(turns)
    const nose: VehicleWeaponMountEntry = { classe: 'fixe', ax: 0, ay: -0.5 }
    const size = { naturalWidthPx: 100, naturalHeightPx: 200, mmPerPx: 10 }
    for (const h of HEADINGS) {
      const dir = screenDirection(view, h)
      const { origin, angle } = vehicleShotOrigin({
        h: null,
        vehicleShot: { mount: nose, arme: 'vehicule', family: 'warthog', headingDeg: h, shooterHeadingDeg: null },
        center: { x: 0, y: 0 },
        sizeOf: () => size,
        k: 1,
        scalePxPerM: 20,
        turns,
      })
      // La direction de la décharge ET la bouche (ancre au nez) pointent dans le sens du cap, à l'écran.
      expect(angle).not.toBeNull()
      expect(sameDirection(angle as number, dir)).toBe(true)
      expect(sameDirection(Math.atan2(origin.y, origin.x), dir)).toBe(true)
    }
  })

  it.each(QUARTER_TURNS)('éclair d’un tir à pied (cap du regard), à %i quart(s) de tour', (turns) => {
    const view = viewAt(turns)
    for (const h of HEADINGS) {
      const { angle } = vehicleShotOrigin({
        h, vehicleShot: null, center: { x: 5, y: 5 }, sizeOf: undefined, k: 1, scalePxPerM: 20, turns,
      })
      expect(sameDirection(angle as number, screenDirection(view, h))).toBe(true)
    }
  })

  it('à 0 quart de tour, rien ne change : les angles d’avant', () => {
    expect(vehicleScreenAngle(30)).toBe(vehicleScreenAngle(30, 0))
    expect(vehicleAimAngle(30)).toBe(vehicleAimAngle(30, 0))
    expect(vehicleScreenAngle(90)).toBeCloseTo(0, 12)
    expect(vehicleAimAngle(90)).toBeCloseTo(-Math.PI / 2, 12)
  })
})
