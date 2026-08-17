/**
 * Tests — equipmentPlacementsLayer (les poses d'équipement : mur, capteur, objets sans nom).
 *
 * Ce que ces tests verrouillent :
 *  - LA TABLE PAR FAMILLE : une famille absente du manifeste ne dessine RIEN (c'est la règle
 *    qui protège l'arrivée des familles du lot de nommage) ;
 *  - L'ORIENTATION DU MUR : le milieu de l'arc est DEVANT la position, dans la direction du
 *    cap du poseur — donc la concavité regarde le poseur ; sans cap, un cercle pointillé et
 *    aucune orientation inventée ;
 *  - LA FENÊTRE D'AFFICHAGE : `t1` date la mise au repos de l'objet et ne referme rien — le
 *    capteur se tient à sa durée officielle, les autres poses vont jusqu'à la fin du rejeu ;
 *  - LE TRACÉ DU CAPTEUR : sa zone est FIXE au rayon officiel, seule l'onde du ping bouge —
 *    et elle ne part pas sous mouvement réduit (l'arithmétique du ping, elle, est verrouillée
 *    dans threatSensor.test.ts) ;
 *  - LA MARQUE « RÉVÉLÉ », tracée dans ce calque, à la teinte de l'équipe du POSEUR ;
 *  - LE SURVOL : la plus petite zone gagne, sinon un mur posé dans un capteur serait
 *    inatteignable au pointeur.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayEquipmentPlacement, ReplayPoint } from '@/lib/api/types'

import {
  drawEquipmentPlacementsLayer,
  isPlacementActive,
  placementEndFrame,
  type PlacementScene,
  placementAt,
  placementKind,
  wallArcWorld,
  wallRadiusM,
  wallRingWorld,
  WALL_OPENING_RAD,
  WALL_RADIUS_M,
} from './equipmentPlacementsLayer'
import { worldToCanvas } from './replayLogic'
import type { ReplayTrackReady } from './replayNormalize'
import { recordingContext } from './test/recordingContext'
import { REVEAL_RADIUS_PX, SENSOR_DURATION_MS, SENSOR_RADIUS_M } from './threatSensor'

/** 10 m de côté sur 100 px : 10 px par mètre — le plancher de lisibilité ne mord pas. */
const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 100,
  height: 100,
  pad: 0,
}

const TIME = {
  frame: 50,
  frameMs: 100,
  frames: 600,
  k: 1,
  reducedMotion: false,
  showUnnamed: false,
}

function pose(over: Partial<ReplayEquipmentPlacement> = {}): ReplayEquipmentPlacement {
  return { t0: 10, t1: 100, x: 5, y: 5, family: 'wall', id: '0x008e2dc5', owner: 3, ...over }
}

/**
 * La SCÈNE du calque. Par défaut aucune vie : la révélation n'a alors personne à marquer, et
 * chaque test qui la vise fournit ses propres vies et ses propres camps.
 */
function scene(
  placements: ReplayEquipmentPlacement[],
  over: Partial<PlacementScene> = {},
): PlacementScene {
  return { placements, lives: [], sideOfSlot: () => null, ...over }
}

/** Une vie IMMOBILE, pour la révélation : deux échantillons à la même position. */
function life(slot: number, x: number, y: number): ReplayTrackReady {
  const points: ReplayPoint[] = [
    { t: 0, x, y },
    { t: 200, x, y },
  ]
  return { slot, team: -1, startFrame: 0, endFrame: 200, points }
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

describe('placementEndFrame — `t1` n’est PAS la disparition', () => {
  it('rien avant t0, et t1 ne referme RIEN : le film ne date aucune disparition', () => {
    const p = pose({ t0: 10, t1: 20 })
    expect([9, 10, 20, 21, 599].map((f) => isPlacementActive(p, 'wall', f, TIME))).toEqual([
      false, true, true, true, true,
    ])
    expect(isPlacementActive(p, 'wall', 600, TIME)).toBe(false)
  })

  it('le capteur se tient à sa durée OFFICIELLE : 15 s, soit 150 images de 100 ms', () => {
    const p = pose({ t0: 10, t1: 20, family: 'sensor' })
    expect(placementEndFrame(p, 'sensor', TIME)).toBe(10 + SENSOR_DURATION_MS / TIME.frameMs)
    expect(isPlacementActive(p, 'sensor', 161, TIME)).toBe(false)
  })

  it('la borne MESURÉE l’emporte quand elle dépasse la durée officielle', () => {
    const p = pose({ t0: 10, t1: 400, family: 'sensor' })
    expect(placementEndFrame(p, 'sensor', TIME)).toBe(400)
  })

  it('une pose sans famille à durée publiée va jusqu’à la dernière image du rejeu', () => {
    expect(placementEndFrame(pose({ t0: 10, t1: 20 }), 'unnamed', TIME)).toBe(599)
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

describe('drawEquipmentPlacementsLayer — ce qui est tracé, et ce qui ne l’est pas', () => {
  it('le mur orienté : une polyligne dont un sommet tombe sur le milieu monde de l’arc', () => {
    const { ops, ctx } = recordingContext()
    drawEquipmentPlacementsLayer(ctx, scene([pose({ h: 90 })]), VIEW, TIME, INK)
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
    drawEquipmentPlacementsLayer(ctx, scene([pose()]), VIEW, TIME, INK)
    expect(ops.some((o) => o.op === 'setLineDash')).toBe(true)
    expect(ops.filter((o) => o.op === 'closePath').length).toBeGreaterThan(0)
  })

  it('le capteur : un disque rempli et son anneau, au rayon OFFICIEL', () => {
    const { ops, ctx } = recordingContext()
    drawEquipmentPlacementsLayer(ctx, scene([pose({ family: 'sensor', t0: 50 })]), VIEW, TIME, INK)
    const c = worldToCanvas({ x: 5, y: 5 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    const arcs = ops.filter((o) => o.op === 'arc')
    // Âge nul à t0 : l'onde du ping part du centre (rayon nul, rien à tracer), il ne reste
    // que la ZONE — son remplissage et son anneau.
    expect(arcs).toHaveLength(2)
    expect(arcs[0].args.slice(0, 2)).toEqual([c.x, c.y])
    expect(arcs[0].args[2]).toBeCloseTo(SENSOR_RADIUS_M * 10, 6)
    expect(arcs[1].args[2]).toBeCloseTo(SENSOR_RADIUS_M * 10, 6)
    expect(ops.some((o) => o.op === 'fill')).toBe(true)
    expect(ops.some((o) => o.op === 'stroke')).toBe(true)
  })

  it('LE PING : une onde en plus, entre le centre et le rayon — la zone ne bouge pas', () => {
    const { ops, ctx } = recordingContext()
    // 2 images après t0, soit 200 ms : l'onde est à mi-course de ses 400 ms.
    const time = { ...TIME, frame: 52 }
    drawEquipmentPlacementsLayer(ctx, scene([pose({ family: 'sensor', t0: 50 })]), VIEW, time, INK)
    const radii = ops.filter((o) => o.op === 'arc').map((o) => o.args[2] as number)
    expect(radii).toHaveLength(3)
    const zone = SENSOR_RADIUS_M * 10
    // La ZONE est servie deux fois au rayon officiel, l'onde une fois en deçà.
    expect(radii[0]).toBeCloseTo(zone, 6)
    expect(radii[1]).toBeCloseTo(zone, 6)
    expect(radii[2]).toBeGreaterThan(0)
    expect(radii[2]).toBeLessThan(zone)
  })

  it('entre deux pings, plus d’onde : la zone seule, au rayon officiel', () => {
    const { ops, ctx } = recordingContext()
    // 10 images après t0 = 1 000 ms : l'onde des 400 ms a fini sa course.
    const time = { ...TIME, frame: 60 }
    drawEquipmentPlacementsLayer(ctx, scene([pose({ family: 'sensor', t0: 50 })]), VIEW, time, INK)
    expect(ops.filter((o) => o.op === 'arc')).toHaveLength(2)
  })

  it('mouvement réduit : la zone reste, l’onde ne part jamais', () => {
    const { ops, ctx } = recordingContext()
    const time = { ...TIME, frame: 52, reducedMotion: true }
    drawEquipmentPlacementsLayer(ctx, scene([pose({ family: 'sensor', t0: 50 })]), VIEW, time, INK)
    expect(ops.filter((o) => o.op === 'arc')).toHaveLength(2)
  })

  it('une pose sans poseur porte l’encre neutre, jamais une couleur d’équipe inventée', () => {
    const { ops, ctx } = recordingContext()
    drawEquipmentPlacementsLayer(ctx, scene([pose({ h: 0, owner: -1 })]), VIEW, TIME, INK)
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'neutre')).toBe(true)
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'equipe')).toBe(false)
  })

  it('une famille inconnue ne trace rien du tout', () => {
    const { ops, ctx } = recordingContext()
    drawEquipmentPlacementsLayer(ctx, scene([pose({ family: 'propulseur', h: 12 })]), VIEW, TIME, INK)
    expect(ops.filter((o) => o.op === 'stroke' || o.op === 'fill')).toHaveLength(0)
  })

  it('les objets non identifiés sont muets par défaut, et un point neutre une fois demandés', () => {
    const off = recordingContext()
    drawEquipmentPlacementsLayer(off.ctx, scene([pose({ family: 'other' })]), VIEW, TIME, INK)
    expect(off.ops.filter((o) => o.op === 'fill')).toHaveLength(0)

    const on = recordingContext()
    drawEquipmentPlacementsLayer(
      on.ctx,
      scene([pose({ family: 'other' })]),
      VIEW,
      { ...TIME, showUnnamed: true },
      INK,
    )
    expect(on.ops.filter((o) => o.op === 'fill')).toHaveLength(1)
  })

  it('avant la pose rien n’est tracé, après `t1` le mur reste : il ne disparaît pas', () => {
    const traced = (frame: number) => {
      const { ops, ctx } = recordingContext()
      drawEquipmentPlacementsLayer(ctx, scene([pose({ h: 45 })]), VIEW, { ...TIME, frame }, INK)
      return ops.filter((o) => o.op === 'stroke').length
    }
    expect(traced(9)).toBe(0)
    expect(traced(101)).toBeGreaterThan(0) // t1 = 100 : rien ne se referme là
    expect(traced(600)).toBe(0) // au-delà de la dernière image du rejeu
  })
})

describe('la marque « révélé », tracée dans ce calque', () => {
  /** Le capteur pinge à l'image 50 ; le poseur (slot 3) est du camp « t0 ». */
  const sensorPose = pose({ family: 'sensor', t0: 50, x: 5, y: 5 })
  const sides: Record<number, string | null> = { 3: 't0', 4: 't0', 7: 't1' }
  const sideOfSlot = (slot: number) => sides[slot] ?? null

  it('un adversaire dans le rayon reçoit un halo, à la position du JOUEUR', () => {
    const { ops, ctx } = recordingContext()
    const foe = life(7, 6, 5) // 1 m du capteur, camp adverse
    drawEquipmentPlacementsLayer(
      ctx,
      scene([sensorPose], { lives: [foe], sideOfSlot }),
      VIEW,
      TIME,
      INK,
    )
    const c = worldToCanvas({ x: 6, y: 5 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    const marks = ops.filter(
      (o) =>
        o.op === 'arc' &&
        Math.abs((o.args[0] as number) - c.x) < 1e-6 &&
        Math.abs((o.args[1] as number) - c.y) < 1e-6,
    )
    // Halo + liseré, tous deux au rayon d'écran de la marque.
    expect(marks).toHaveLength(2)
    for (const m of marks) expect(m.args[2]).toBeCloseTo(REVEAL_RADIUS_PX, 6)
  })

  it('la marque porte la teinte de l’équipe du POSEUR — c’est son camp qui voit', () => {
    const { ops, ctx } = recordingContext()
    const foe = life(7, 6, 5)
    drawEquipmentPlacementsLayer(
      ctx,
      scene([sensorPose], { lives: [foe], sideOfSlot }),
      VIEW,
      TIME,
      // Une encre par slot : le poseur est le slot 3, la cible le slot 7.
      { colorOfSlot: (slot) => `slot${slot}`, neutral: 'neutre' },
    )
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'slot3')).toBe(true)
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'slot7')).toBe(false)
  })

  it('un coéquipier du poseur n’est pas marqué : rien de plus que la zone', () => {
    const { ops, ctx } = recordingContext()
    const mate = life(4, 6, 5)
    drawEquipmentPlacementsLayer(
      ctx,
      scene([sensorPose], { lives: [mate], sideOfSlot }),
      VIEW,
      TIME,
      INK,
    )
    expect(ops.filter((o) => o.op === 'arc')).toHaveLength(2)
  })

  it('sans camp connu, aucune marque — le ping se dessine quand même', () => {
    const { ops, ctx } = recordingContext()
    const foe = life(7, 6, 5)
    // Poseur non mesuré : le capteur existe et pinge, mais il ne révèle personne.
    drawEquipmentPlacementsLayer(
      ctx,
      scene([pose({ family: 'sensor', t0: 50, x: 5, y: 5, owner: -1 })], {
        lives: [foe],
        sideOfSlot,
      }),
      VIEW,
      TIME,
      INK,
    )
    expect(ops.filter((o) => o.op === 'arc')).toHaveLength(2)
  })

  it('un mur seul ne révèle rien : la révélation appartient au capteur', () => {
    const { ops, ctx } = recordingContext()
    const foe = life(7, 6, 5)
    drawEquipmentPlacementsLayer(
      ctx,
      scene([pose({ h: 90 })], { lives: [foe], sideOfSlot }),
      VIEW,
      TIME,
      INK,
    )
    expect(ops.filter((o) => o.op === 'arc')).toHaveLength(0)
  })
})

describe('placementAt — le survol', () => {
  const hover = { frame: 50, showUnnamed: false, frameMs: TIME.frameMs, frames: TIME.frames }

  it('la PLUS PETITE zone gagne : un mur posé dans un capteur reste atteignable', () => {
    const wall = pose({ family: 'wall', id: '0xmur', x: 5, y: 5 })
    const sensor = pose({ family: 'sensor', id: '0xcapteur', x: 5, y: 5 })
    const c = worldToCanvas({ x: 5, y: 5 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    // Dans les DEUX ordres : c'est la taille qui tranche, jamais le rang dans la liste.
    expect(placementAt([sensor, wall], VIEW, hover, c)?.id).toBe('0xmur')
    expect(placementAt([wall, sensor], VIEW, hover, c)?.id).toBe('0xmur')
    // Et le capteur reste atteignable partout ailleurs dans sa zone (30 px = 3 m, à
    // l'intérieur des 4,25 m officiels).
    expect(placementAt([sensor, wall], VIEW, hover, { x: c.x + 30, y: c.y })?.id).toBe('0xcapteur')
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
