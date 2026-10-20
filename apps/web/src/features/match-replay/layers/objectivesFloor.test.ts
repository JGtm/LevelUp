/**
 * Tests — L'ALTITUDE DES OBJECTIFS SE DIT COMME CELLE DES JOUEURS (2026-09-18).
 *
 * Un marqueur porte `fl` anneaux concentriques AU-DELÀ de son anneau de livraison (8 px) ; une
 * zone porte `fl` contours concentriques EXTÉRIEURS, dilatés d'un pas constant en pixels — à
 * l'extérieur parce que le calque vivant repeint l'intérieur de la forme. L'étage vient de
 * `floorOf` sur l'amplitude du document ; un terrain PLAT n'en donne aucun.
 *
 * FICHIER À PART de `objectivesLayer.test.ts` : celui-ci était à son plafond de lignes ; la
 * responsabilité « étage » a le sien. Le contexte enregistreur est le double partagé du rejeu.
 *
 * Sur VIEW (48 px/m) et Z = 0..10 : z=1 -> étage 0, z=5 -> étage 1, z=9 -> étage 2.
 */
import { describe, expect, it } from 'vitest'

import { drawObjectivesLayer, normalizeMapObjectives } from './objectivesLayer'
import { recordingContext, type CanvasOp } from '../test/recordingContext'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 480 + 48, height: 480 + 48, pad: 24 }
const STYLE = { colorOfTeam: () => '#123456', neutralOutline: '#000000', z: { min: 0, max: 10 } }

const arcsOf = (ops: CanvasOp[]) => ops.filter((o) => o.op === 'arc').map((o) => o.args[2] as number)
const inksOf = (ops: CanvasOp[]) => ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0] as string)
/** Largeur et opacité EN VIGUEUR à chaque `stroke` : ce sont des propriétés lues à cet instant. */
function strokesOf(ops: CanvasOp[]): { width: number; alpha: number }[] {
  const out: { width: number; alpha: number }[] = []
  let width = 1
  let alpha = 1
  for (const o of ops) {
    if (o.op === 'set lineWidth') width = o.args[0] as number
    else if (o.op === 'set globalAlpha') alpha = o.args[0] as number
    else if (o.op === 'stroke') out.push({ width, alpha })
  }
  return out
}

describe("l'ÉTAGE des marqueurs — anneaux concentriques au-delà de l'anneau de livraison", () => {
  it('une livraison à l’étage 2 porte DEUX anneaux au-delà des 8 px de livraison, au sol aucun', () => {
    const haut = recordingContext()
    drawObjectivesLayer(
      haut.ctx,
      normalizeMapObjectives({ markers: [{ role: 'flag_delivery', team: 0, x: 1, y: 8, z: 9 }] }),
      VIEW,
      STYLE,
    )
    const rayons = arcsOf(haut.ops)
    expect(rayons).toHaveLength(3)
    const etage = rayons.filter((r) => r > 8).sort((a, b) => a - b)
    expect(etage).toHaveLength(2)
    // Le second anneau est plus loin que le premier : ils se COMPTENT.
    expect(etage[1]).toBeGreaterThan(etage[0])

    const sol = recordingContext()
    drawObjectivesLayer(
      sol.ctx,
      normalizeMapObjectives({ markers: [{ role: 'flag_delivery', team: 0, x: 1, y: 8, z: 1 }] }),
      VIEW,
      STYLE,
    )
    expect(arcsOf(sol.ops)).toEqual([8])
  })

  it('un socle (sans anneau de livraison) à l’étage 1 porte UN anneau, dans la couleur de l’objectif', () => {
    const { ctx, ops } = recordingContext()
    drawObjectivesLayer(
      ctx,
      normalizeMapObjectives({ markers: [{ role: 'flag_spawn', team: 1, x: 1, y: 9, z: 5 }] }),
      VIEW,
      { ...STYLE, colorOfTeam: () => '#0F0F0F', neutralOutline: '#ABCDEF' },
    )
    const rayons = arcsOf(ops)
    expect(rayons).toHaveLength(1)
    expect(rayons[0]).toBeGreaterThan(8)
    // Objectif TENU : aucune encre de liseré, seule la couleur du camp trace l'anneau.
    expect(inksOf(ops)).toContain('#0F0F0F')
    expect(inksOf(ops)).not.toContain('#ABCDEF')
  })
})

describe("l'ÉTAGE des zones — contours concentriques EXTÉRIEURS", () => {
  it('un cylindre à l’étage 1 gagne UN contour, dilaté du pas d’étage en pixels', () => {
    const { ctx, ops } = recordingContext()
    drawObjectivesLayer(
      ctx,
      normalizeMapObjectives({
        zones: [{ role: 'koth_hill', team: -1, x: 8, y: 2, z: 5, family: 'cylinder', radius: 3, fwdX: 1, fwdY: 0 }],
      }),
      VIEW,
      STYLE,
    )
    const rayons = arcsOf(ops)
    expect(rayons).toHaveLength(2)
    expect(rayons[0]).toBeCloseTo(3 * 48, 5)
    // Le contour d'étage est à un pas CONSTANT en pixels (2,8) — pas proportionnel au rayon.
    expect(rayons[1]).toBeCloseTo(3 * 48 + 2.8, 5)
  })

  it('une boîte à l’étage 2 gagne DEUX tracés dont les coins sont AGRANDIS, jamais rétrécis', () => {
    const { ctx, ops } = recordingContext()
    drawObjectivesLayer(
      ctx,
      normalizeMapObjectives({
        zones: [{ role: 'strongholds_zone', team: -1, x: 5, y: 5, z: 9, family: 'box', halfX: 2, halfY: 1, fwdX: 0, fwdY: 1 }],
      }),
      VIEW,
      STYLE,
    )
    // Trois tracés de quatre coins : la forme, puis deux contours d'étage.
    const pts = ops.filter((o) => o.op === 'moveTo' || o.op === 'lineTo').map((o) => o.args as number[])
    expect(pts).toHaveLength(12)
    const emprise = (quad: number[][]) => {
      const xs = quad.map((p) => p[0])
      const ys = quad.map((p) => p[1])
      return { w: Math.max(...xs) - Math.min(...xs), h: Math.max(...ys) - Math.min(...ys) }
    }
    const forme = emprise(pts.slice(0, 4))
    const un = emprise(pts.slice(4, 8))
    const deux = emprise(pts.slice(8, 12))
    // Forme : 2 m × 4 m -> 96 × 192 px. Chaque contour dilate de 2,8 px de chaque côté.
    expect(forme).toEqual({ w: 96, h: 192 })
    expect(un.w).toBeCloseTo(96 + 2 * 2.8, 5)
    expect(un.h).toBeCloseTo(192 + 2 * 2.8, 5)
    expect(deux.w).toBeCloseTo(96 + 4 * 2.8, 5)
    expect(deux.h).toBeCloseTo(192 + 4 * 2.8, 5)
  })

  it('les contours d’étage sont FINS (1 px) et pâlissent vers l’extérieur', () => {
    const { ctx, ops } = recordingContext()
    drawObjectivesLayer(
      ctx,
      normalizeMapObjectives({
        zones: [{ role: 'koth_hill', team: 0, x: 8, y: 2, z: 9, family: 'cylinder', radius: 3, fwdX: 1, fwdY: 0 }],
      }),
      VIEW,
      STYLE,
    )
    // Contour de la zone (1,5 px), puis deux contours d'étage à 1 px, le second plus pâle.
    const traits = strokesOf(ops)
    expect(traits.map((t) => t.width)).toEqual([1.5, 1, 1])
    expect(traits[2].alpha).toBeLessThan(traits[1].alpha)
  })
})

describe("l'ÉTAGE des objectifs — les gardes", () => {
  it('TERRAIN PLAT : aucun anneau ni contour, quel que soit le z', () => {
    const { ctx, ops } = recordingContext()
    drawObjectivesLayer(
      ctx,
      normalizeMapObjectives({
        zones: [{ role: 'koth_hill', team: -1, x: 8, y: 2, z: 5, family: 'cylinder', radius: 3, fwdX: 1, fwdY: 0 }],
        markers: [{ role: 'flag_delivery', team: 0, x: 1, y: 8, z: 5 }],
      }),
      VIEW,
      { ...STYLE, z: { min: 4, max: 4 } },
    )
    // Le cercle de la zone et l'anneau de livraison, rien d'autre.
    expect(arcsOf(ops)).toHaveLength(2)
  })

  it('anneaux et contours n’écrivent AUCUN texte — le garde du calque tient à l’étage aussi', () => {
    const { ctx, ops } = recordingContext()
    const hauts = normalizeMapObjectives({
      zones: [
        { role: 'strongholds_zone', team: -1, x: 5, y: 5, z: 9, family: 'box', halfX: 2, halfY: 1, fwdX: 0, fwdY: 1 },
        { role: 'flag_delivery', team: 1, x: 8, y: 2, z: 9, family: 'cylinder', radius: 3, fwdX: 1, fwdY: 0 },
      ],
      markers: [
        { role: 'flag_spawn', team: 0, x: 1, y: 9, z: 9 },
        { role: 'flag_delivery', team: 0, x: 1, y: 8, z: 9 },
      ],
    })
    drawObjectivesLayer(ctx, hauts, VIEW, STYLE)
    expect(ops.filter((o) => o.op === 'fillText' || o.op === 'strokeText')).toHaveLength(0)
  })
})
