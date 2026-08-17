/**
 * Tests — placementWall (le mur de protection : sa forme, et SUR QUELLE POSE il se dessine).
 *
 * Ce que ces tests verrouillent :
 *  - L'ORIENTATION : le milieu de l'arc est DEVANT la position, dans la direction du cap du
 *    poseur — donc la concavité regarde le poseur ; sans cap, un cercle pointillé et aucune
 *    orientation inventée ;
 *  - LE PLANCHER DE LISIBILITÉ : sur une carte immense, le rayon monde est relevé plutôt que
 *    de laisser l'arc tomber à une éraflure ;
 *  - LES PANNEAUX CONTRE L'APPAREIL : l'arc va sur les panneaux, jamais sur l'appareil, même
 *    déployé — sinon un mur déployé dessinerait DEUX arcs (il produit deux poses).
 *
 * L'accord de la table `WALL_PANEL_IDS` avec le manifeste du titre est un test à part
 * (`placementPanels.guard.test.ts`) : il traverse la frontière Go/TS et lit un TOML.
 */
import { describe, expect, it } from 'vitest'

import { placementIsDeployedObject, placementKind } from './equipmentPlacementsLayer'
import {
  WALL_OPENING_RAD,
  WALL_PANEL_IDS,
  WALL_RADIUS_M,
  wallArcWorld,
  wallRadiusM,
  wallRingWorld,
} from './placementWall'
import {
  DEVICE_ID,
  draw,
  painted,
  PANEL_ID,
  pose,
  projected,
  TIME,
  VIEW,
} from './test/placementFixtures'

describe('wallArcWorld — le milieu de l’arc est devant le poseur', () => {
  it('cap 90° : le milieu de l’arc est à la verticale monde de la position', () => {
    const pts = wallArcWorld({ x: 5, y: 5 }, 90, 2, 16)
    expect(pts).toHaveLength(17)
    const mid = pts[8]
    expect(mid.x).toBeCloseTo(5, 6)
    expect(mid.y).toBeCloseTo(7, 6)
  })

  it('cap 0° : le milieu part vers les X croissants (convention de Point.h)', () => {
    const mid = wallArcWorld({ x: 0, y: 0 }, 0, 3, 16)[8]
    expect(mid.x).toBeCloseTo(3, 6)
    expect(mid.y).toBeCloseTo(0, 6)
  })

  it('tous les points sont à distance constante du centre : la concavité regarde le poseur', () => {
    const c = { x: -2, y: 7 }
    for (const p of wallArcWorld(c, 210, 1.6, 16)) {
      expect(Math.hypot(p.x - c.x, p.y - c.y)).toBeCloseTo(1.6, 6)
    }
  })

  it('l’arc couvre exactement l’ouverture déclarée (110°), et pas davantage', () => {
    const c = { x: 0, y: 0 }
    const pts = wallArcWorld(c, 90, 1, 16)
    const a0 = Math.atan2(pts[0].y, pts[0].x)
    const a1 = Math.atan2(pts[16].y, pts[16].x)
    expect(Math.abs(a1 - a0)).toBeCloseTo(WALL_OPENING_RAD, 6)
  })
})

describe('wallRingWorld — sans cap, aucune orientation inventée', () => {
  it('rend un tour complet : premier et dernier point confondus', () => {
    const pts = wallRingWorld({ x: 1, y: 1 }, 2, 24)
    expect(pts).toHaveLength(25)
    expect(pts[24].x).toBeCloseTo(pts[0].x, 6)
    expect(pts[24].y).toBeCloseTo(pts[0].y, 6)
  })
})

describe('wallRadiusM — le plancher de lisibilité', () => {
  it('à grande échelle, le rayon du plan est servi tel quel', () => {
    expect(wallRadiusM(VIEW)).toBeCloseTo(WALL_RADIUS_M, 6)
  })

  it('sur une carte immense, le rayon monde est relevé pour rester visible', () => {
    // 400 m de côté sur 100 px : 0,25 px/m — 1,6 m ferait 0,4 px de rayon.
    const wide = { ...VIEW, bounds: { minX: 0, minY: 0, maxX: 400, maxY: 400 } }
    expect(wallRadiusM(wide)).toBeGreaterThan(WALL_RADIUS_M)
  })
})

describe('drawWall — l’arc, son halo, et le cercle des poses sans cap', () => {
  it('le mur orienté : une polyligne dont un sommet tombe sur le milieu monde de l’arc', () => {
    const ops = draw([pose({ h: 90 })])
    const mid = projected(5, 5 + wallRadiusM(VIEW))
    const hit = ops.some(
      (o) =>
        o.op === 'lineTo' &&
        Math.abs((o.args[0] as number) - mid.x) < 1e-6 &&
        Math.abs((o.args[1] as number) - mid.y) < 1e-6,
    )
    expect(hit).toBe(true)
    // Trait franc, halo, couleur d'équipe — et JAMAIS de pointillé sur une pose orientée.
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(2)
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'equipe')).toBe(true)
    expect(ops.some((o) => o.op === 'setLineDash')).toBe(false)
  })

  it('le mur sans cap : un cercle FERMÉ et POINTILLÉ, aucune direction affirmée', () => {
    const ops = draw([pose()])
    expect(ops.some((o) => o.op === 'setLineDash')).toBe(true)
    expect(ops.filter((o) => o.op === 'closePath').length).toBeGreaterThan(0)
  })

  it('une pose sans poseur porte l’encre neutre, jamais une couleur d’équipe inventée', () => {
    const ops = draw([pose({ h: 0, owner: -1 })])
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'neutre')).toBe(true)
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'equipe')).toBe(false)
  })

  it('hors de la fenêtre mesurée, rien n’est tracé : la durée EST la pose', () => {
    for (const frame of [9, 101]) {
      expect(painted([pose({ h: 45 })], { ...TIME, frame })).toBe(0)
    }
  })
})

describe('les PANNEAUX du mur contre son APPAREIL', () => {
  it('les deux identifiants de panneaux dessinent, chacun son arc', () => {
    for (const id of WALL_PANEL_IDS) {
      expect(placementIsDeployedObject(pose({ id })), id).toBe(true)
      expect(painted([pose({ id, h: 45 })]), id).toBeGreaterThan(0)
    }
  })

  it("l'APPAREIL déployé ne dessine RIEN : un mur déployé produit deux poses, pas deux arcs", () => {
    const appareil = pose({ id: DEVICE_ID, h: 45 })
    expect(placementIsDeployedObject(appareil)).toBe(false)
    expect(placementKind(appareil, true)).toBeNull()
    expect(painted([appareil])).toBe(0)
  })

  it('un mur déployé (appareil + panneaux) ne rend QU UN arc', () => {
    const ops = draw([pose({ id: DEVICE_ID, h: 45 }), pose({ id: PANEL_ID, h: 45 })])
    // Un arc = halo + trait franc, soit DEUX `stroke`. Quatre diraient deux arcs.
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(2)
  })

  it("la règle des panneaux ne vise QUE le mur : ailleurs, aucun identifiant n'est privilégié", () => {
    expect(placementIsDeployedObject(pose({ family: 'sensor', id: DEVICE_ID }))).toBe(true)
  })
})
