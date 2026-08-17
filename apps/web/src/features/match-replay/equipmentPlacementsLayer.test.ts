/**
 * Tests — equipmentPlacementsLayer (les poses d'équipement : mur, capteur, objets sans nom).
 *
 * Ce que ces tests verrouillent :
 *  - LA TABLE PAR FAMILLE : une famille absente du manifeste ne dessine RIEN (c'est la règle
 *    qui protège l'arrivée des familles du lot de nommage) ;
 *  - L'ORIENTATION DU MUR : le milieu de l'arc est DEVANT la position, dans la direction du
 *    cap du poseur — donc la concavité regarde le poseur ; sans cap, un cercle pointillé et
 *    aucune orientation inventée ;
 *  - LA FENÊTRE [t0, t1], exacte, sans rémanence (même contrat que la ligne de grappin) ;
 *  - LA PULSATION du capteur comme FONCTION DU TEMPS, figée sous mouvement réduit ;
 *  - LE SURVOL : la plus petite zone gagne, sinon un mur posé dans un capteur serait
 *    inatteignable au pointeur.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import {
  drawEquipmentPlacementsLayer,
  isPlacementActive,
  placementAt,
  placementKind,
  SENSOR_PULSE_MS,
  SENSOR_RADIUS_M,
  sensorPulse,
  wallArcWorld,
  wallRadiusM,
  wallRingWorld,
  WALL_OPENING_RAD,
  WALL_RADIUS_M,
} from './equipmentPlacementsLayer'
import { worldToCanvas } from './replayLogic'
import { recordingContext } from './test/recordingContext'

/** 10 m de côté sur 100 px : 10 px par mètre — le plancher de lisibilité ne mord pas. */
const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 100,
  height: 100,
  pad: 0,
}

const TIME = { frame: 50, frameMs: 100, k: 1, reducedMotion: false, showUnnamed: false }

function pose(over: Partial<ReplayEquipmentPlacement> = {}): ReplayEquipmentPlacement {
  return { t0: 10, t1: 100, x: 5, y: 5, family: 'wall', id: '0x008e2dc5', owner: 3, ...over }
}

const INK = { colorOfSlot: () => 'equipe', neutral: 'neutre' }

describe('placementKind — la table par famille', () => {
  it('une famille inconnue du manifeste ne dessine RIEN', () => {
    expect(placementKind(pose({ family: 'translocateur' }), true)).toBeNull()
  })

  it('mur et capteur ont chacun leur règle', () => {
    expect(placementKind(pose({ family: 'wall' }), false)).toBe('wall')
    expect(placementKind(pose({ family: 'sensor' }), false)).toBe('sensor')
  })

  it('un objet non identifié ne se dessine que si la bascule est allumée', () => {
    expect(placementKind(pose({ family: 'other' }), false)).toBeNull()
    expect(placementKind(pose({ family: 'other' }), true)).toBe('unnamed')
  })
})

describe('isPlacementActive — la fenêtre mesurée, bornes comprises', () => {
  it('vraie sur [t0, t1], fausse d’un cran de part et d’autre', () => {
    const p = pose({ t0: 10, t1: 20 })
    expect([9, 10, 20, 21].map((f) => isPlacementActive(p, f))).toEqual([false, true, true, false])
  })
})

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

describe('sensorPulse — fonction du temps, jamais un état', () => {
  it('le même âge rend toujours la même valeur (un retour en arrière rejoue l’image)', () => {
    expect(sensorPulse(400, false)).toBeCloseTo(sensorPulse(400 + SENSOR_PULSE_MS, false), 10)
  })

  it('reste borné autour du rayon déclaré', () => {
    for (let ms = 0; ms < SENSOR_PULSE_MS; ms += 37) {
      expect(sensorPulse(ms, false)).toBeGreaterThan(0.9)
      expect(sensorPulse(ms, false)).toBeLessThan(1.1)
    }
  })

  it('mouvement réduit : la phase est figée, le disque reste', () => {
    for (const ms of [0, 400, 900, 1_500]) expect(sensorPulse(ms, true)).toBe(1)
  })
})

describe('drawEquipmentPlacementsLayer — ce qui est tracé, et ce qui ne l’est pas', () => {
  it('le mur orienté : une polyligne dont un sommet tombe sur le milieu monde de l’arc', () => {
    const { ops, ctx } = recordingContext()
    drawEquipmentPlacementsLayer(ctx, [pose({ h: 90 })], VIEW, TIME, INK)
    const mid = worldToCanvas(
      { x: 5, y: 5 + wallRadiusM(VIEW) },
      VIEW.bounds,
      VIEW.width,
      VIEW.height,
      VIEW.pad,
    )
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
    const { ops, ctx } = recordingContext()
    drawEquipmentPlacementsLayer(ctx, [pose()], VIEW, TIME, INK)
    expect(ops.some((o) => o.op === 'setLineDash')).toBe(true)
    expect(ops.filter((o) => o.op === 'closePath').length).toBeGreaterThan(0)
  })

  it('le capteur : un disque rempli et son anneau, au rayon déclaré', () => {
    const { ops, ctx } = recordingContext()
    drawEquipmentPlacementsLayer(ctx, [pose({ family: 'sensor', t0: 50 })], VIEW, TIME, INK)
    const c = worldToCanvas({ x: 5, y: 5 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    const arcs = ops.filter((o) => o.op === 'arc')
    expect(arcs).toHaveLength(2)
    expect(arcs[0].args.slice(0, 2)).toEqual([c.x, c.y])
    // Âge nul à t0 : le rayon vaut exactement le rayon déclaré, à l'échelle (10 px/m).
    expect(arcs[0].args[2]).toBeCloseTo(SENSOR_RADIUS_M * 10, 6)
    expect(ops.some((o) => o.op === 'fill')).toBe(true)
    expect(ops.some((o) => o.op === 'stroke')).toBe(true)
  })

  it('une pose sans poseur porte l’encre neutre, jamais une couleur d’équipe inventée', () => {
    const { ops, ctx } = recordingContext()
    drawEquipmentPlacementsLayer(ctx, [pose({ h: 0, owner: -1 })], VIEW, TIME, INK)
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'neutre')).toBe(true)
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'equipe')).toBe(false)
  })

  it('une famille inconnue ne trace rien du tout', () => {
    const { ops, ctx } = recordingContext()
    drawEquipmentPlacementsLayer(ctx, [pose({ family: 'propulseur', h: 12 })], VIEW, TIME, INK)
    expect(ops.filter((o) => o.op === 'stroke' || o.op === 'fill')).toHaveLength(0)
  })

  it('les objets non identifiés sont muets par défaut, et un point neutre une fois demandés', () => {
    const off = recordingContext()
    drawEquipmentPlacementsLayer(off.ctx, [pose({ family: 'other' })], VIEW, TIME, INK)
    expect(off.ops.filter((o) => o.op === 'fill')).toHaveLength(0)

    const on = recordingContext()
    drawEquipmentPlacementsLayer(
      on.ctx,
      [pose({ family: 'other' })],
      VIEW,
      { ...TIME, showUnnamed: true },
      INK,
    )
    expect(on.ops.filter((o) => o.op === 'fill')).toHaveLength(1)
  })

  it('hors de la fenêtre mesurée, rien n’est tracé : la durée EST la pose', () => {
    for (const frame of [9, 101]) {
      const { ops, ctx } = recordingContext()
      drawEquipmentPlacementsLayer(ctx, [pose({ h: 45 })], VIEW, { ...TIME, frame }, INK)
      expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(0)
    }
  })
})

describe('placementAt — le survol', () => {
  const hover = { frame: 50, showUnnamed: false }

  it('la PLUS PETITE zone gagne : un mur posé dans un capteur reste atteignable', () => {
    const wall = pose({ family: 'wall', id: '0xmur', x: 5, y: 5 })
    const sensor = pose({ family: 'sensor', id: '0xcapteur', x: 5, y: 5 })
    const c = worldToCanvas({ x: 5, y: 5 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    // Dans les DEUX ordres : c'est la taille qui tranche, jamais le rang dans la liste.
    expect(placementAt([sensor, wall], VIEW, hover, c)?.id).toBe('0xmur')
    expect(placementAt([wall, sensor], VIEW, hover, c)?.id).toBe('0xmur')
    // Et le capteur reste atteignable partout ailleurs dans sa zone.
    expect(placementAt([sensor, wall], VIEW, hover, { x: c.x + 40, y: c.y })?.id).toBe('0xcapteur')
  })

  it('rien sous le curseur = null, jamais la pose la plus proche par défaut', () => {
    expect(placementAt([pose()], VIEW, hover, { x: 99, y: 99 })).toBeNull()
  })

  it('une pose éteinte par sa bascule ne se survole pas non plus', () => {
    const p = pose({ family: 'other' })
    const c = worldToCanvas({ x: 5, y: 5 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    expect(placementAt([p], VIEW, hover, c)).toBeNull()
    expect(placementAt([p], VIEW, { ...hover, showUnnamed: true }, c)?.id).toBe(p.id)
  })
})
