/**
 * Tests — equipmentPlacementsLayer : CE QUI SE DESSINE, et ce qui ne se dessine pas.
 *
 * Le calque DÉCIDE ; les formes elles-mêmes sont testées à côté (`placementWall.test.ts` pour
 * le mur, `placementShapes.test.ts` pour la balise, le traqueur et le champ). Ici :
 *  - LA TABLE PAR FAMILLE, famille par famille : les six règles de rendu, les sept familles
 *    PORTÉES à `null` (décision, pas oubli), les power-ups HORS table, et une famille inconnue
 *    qui ne dessine rien ;
 *  - LE FILTRE D'ORIGINE (schéma 10) : seul `deployed` se dessine ; `dropped`, `unknown` et
 *    l'origine ABSENTE (artefact antérieur) ne dessinent rien — sauf l'objet non identifié,
 *    qui reste gouverné par sa seule bascule ;
 *  - LA FENÊTRE [t0, t1], exacte, sans rémanence (même contrat que la ligne de grappin) ;
 *  - LE TRACÉ DU CAPTEUR : sa zone est FIXE au rayon officiel, seule l'onde du ping bouge —
 *    et elle ne part pas sous mouvement réduit (l'arithmétique du ping, elle, est verrouillée
 *    dans threatSensor.test.ts) ;
 *  - LA MARQUE « RÉVÉLÉ », tracée dans ce calque, à la teinte de l'équipe du POSEUR ;
 *  - LE SURVOL : la plus petite zone gagne, sinon un mur posé dans un capteur serait
 *    inatteignable au pointeur — et on ne survole pas ce qui n'est pas dessiné.
 */
import { describe, expect, it } from 'vitest'

import {
  isPlacementActive,
  PLACEMENT_RENDER,
  placementAt,
  placementIsDeployedObject,
  placementKind,
  placementOrigin,
} from './equipmentPlacementsLayer'
import {
  BEACON_ID,
  DEVICE_ID,
  draw,
  FIELD_ID,
  life,
  painted,
  PANEL_ID,
  pose,
  projected,
  SEEKER_ID,
  SENSOR_ID,
  TIME,
  VIEW,
} from './test/placementFixtures'
import { REVEAL_RADIUS_PX, SENSOR_RADIUS_M } from './threatSensor'

describe('PLACEMENT_RENDER — la table, famille par famille', () => {
  it('les cinq familles DÉPLOYABLES ont chacune leur règle de rendu', () => {
    expect(PLACEMENT_RENDER.wall).toBe('wall')
    expect(PLACEMENT_RENDER.sensor).toBe('sensor')
    expect(PLACEMENT_RENDER.translocator_beacon).toBe('beacon')
    expect(PLACEMENT_RENDER.threat_seeker).toBe('seeker')
    expect(PLACEMENT_RENDER.repair_field).toBe('field')
    expect(PLACEMENT_RENDER.other).toBe('unnamed')
  })

  it('les sept familles PORTÉES sont à `null` — connues, et volontairement muettes', () => {
    const portees = [
      'grenade_frag',
      'grenade_plasma',
      'grenade_dynamo',
      'grenade_spike',
      'grapple',
      'thruster',
      'repulsor',
    ]
    for (const f of portees) {
      expect(f in PLACEMENT_RENDER, `${f} doit être DANS la table`).toBe(true)
      expect(PLACEMENT_RENDER[f], `${f} doit valoir null`).toBeNull()
      expect(placementKind(pose({ family: f }), true)).toBeNull()
    }
  })

  it('les POWER-UPS sont hors table : aucune règle, pas même `null` (correction 3 de la mesure)', () => {
    // n = 1 par power-up sur les 11 films, les deux avec poseur et `dropped` : rien ne soutient
    // l'hypothèse « objet de la carte sans poseur ». Pas de vocabulaire mort (CLAUDE.md n°7).
    for (const f of ['powerup_overshield', 'powerup_camo']) {
      expect(f in PLACEMENT_RENDER, `${f} doit rester HORS table`).toBe(false)
      expect(placementKind(pose({ family: f }), true)).toBeNull()
    }
  })

  it('une famille inconnue du manifeste ne dessine RIEN', () => {
    expect(placementKind(pose({ family: 'famille_future' }), true)).toBeNull()
    expect(painted([pose({ family: 'famille_future', h: 12 })])).toBe(0)
  })
})

describe('placementOrigin / placementIsDeployedObject — le filtre du schéma 10', () => {
  it("l'origine ABSENTE se lit `unknown`, JAMAIS `deployed` (parc antérieur au schéma 10)", () => {
    const sansOrigine = pose()
    delete sansOrigine.origin
    expect(placementOrigin(sansOrigine)).toBe('unknown')
    expect(placementIsDeployedObject(sansOrigine)).toBe(false)
    expect(placementKind(sansOrigine, false)).toBeNull()
  })

  it('un LÂCHER ne se dessine pas — 88,6 % du corpus en est', () => {
    for (const origin of ['dropped', 'unknown']) {
      expect(placementIsDeployedObject(pose({ origin })), origin).toBe(false)
      expect(placementKind(pose({ origin }), false), origin).toBeNull()
      expect(painted([pose({ origin, h: 90 })]), origin).toBe(0)
    }
  })

  it('un DÉPLOIEMENT se dessine, pour chacune des familles déployables', () => {
    const attendu: [string, string][] = [
      ['wall', PANEL_ID],
      ['sensor', SENSOR_ID],
      ['translocator_beacon', BEACON_ID],
      ['threat_seeker', SEEKER_ID],
      ['repair_field', FIELD_ID],
    ]
    // `t0: 48` place l'image courante (50) à 200 ms de la pose : le traqueur y est au milieu
    // de son unique impulsion, les quatre autres sont indifférentes à leur âge.
    for (const [family, id] of attendu) {
      const p = pose({ family, id, t0: 48, h: 90 })
      expect(placementIsDeployedObject(p), family).toBe(true)
      expect(painted([p]), family).toBeGreaterThan(0)
    }
  })

  it("l'objet non identifié ÉCHAPPE au filtre d'origine : seule sa bascule le gouverne", () => {
    // Sa bascule est un outil de diagnostic — on cherche à voir ce qu'on ne sait pas nommer,
    // d'où que l'objet vienne. Comportement inchangé par ce lot.
    const lache = pose({ family: 'other', origin: 'dropped' })
    expect(placementKind(lache, false)).toBeNull()
    expect(placementKind(lache, true)).toBe('unnamed')
    expect(painted([lache])).toBe(0)
    expect(painted([lache], { ...TIME, showUnnamed: true })).toBe(1)
  })
})

describe('isPlacementActive — la fenêtre mesurée, bornes comprises', () => {
  it('vraie sur [t0, t1], fausse d’un cran de part et d’autre', () => {
    const p = pose({ t0: 10, t1: 20 })
    expect([9, 10, 20, 21].map((f) => isPlacementActive(p, f))).toEqual([false, true, true, false])
  })
})

describe('drawSensor — la zone officielle et son onde', () => {
  const sensor = (over = {}) => pose({ family: 'sensor', id: SENSOR_ID, t0: 50, ...over })

  it('un disque rempli et son anneau, au rayon OFFICIEL', () => {
    const ops = draw([sensor()])
    const c = projected(5, 5)
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
    // 2 images après t0, soit 200 ms : l'onde est à mi-course de ses 400 ms.
    const radii = draw([sensor()], { ...TIME, frame: 52 })
      .filter((o) => o.op === 'arc')
      .map((o) => o.args[2] as number)
    expect(radii).toHaveLength(3)
    const zone = SENSOR_RADIUS_M * 10
    // La ZONE est servie deux fois au rayon officiel, l'onde une fois en deçà.
    expect(radii[0]).toBeCloseTo(zone, 6)
    expect(radii[1]).toBeCloseTo(zone, 6)
    expect(radii[2]).toBeGreaterThan(0)
    expect(radii[2]).toBeLessThan(zone)
  })

  it('entre deux pings, plus d’onde : la zone seule, au rayon officiel', () => {
    // 10 images après t0 = 1 000 ms : l'onde des 400 ms a fini sa course.
    expect(draw([sensor()], { ...TIME, frame: 60 }).filter((o) => o.op === 'arc')).toHaveLength(2)
  })

  it('mouvement réduit : la zone reste, l’onde ne part jamais', () => {
    const time = { ...TIME, frame: 52, reducedMotion: true }
    expect(draw([sensor()], time).filter((o) => o.op === 'arc')).toHaveLength(2)
  })
})

describe('la marque « révélé », tracée dans ce calque', () => {
  /** Le capteur pinge à l'image 50 ; le poseur (slot 3) est du camp « t0 ». */
  const sensorPose = pose({ family: 'sensor', id: SENSOR_ID, t0: 50, x: 5, y: 5 })
  const sides: Record<number, string | null> = { 3: 't0', 4: 't0', 7: 't1' }
  const sideOfSlot = (slot: number) => sides[slot] ?? null
  const arcsOf = (ops: ReturnType<typeof draw>) => ops.filter((o) => o.op === 'arc')

  it('un adversaire dans le rayon reçoit un halo, à la position du JOUEUR', () => {
    const foe = life(7, 6, 5) // 1 m du capteur, camp adverse
    const ops = draw([sensorPose], TIME, { lives: [foe], sideOfSlot })
    const c = projected(6, 5)
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
    const foe = life(7, 6, 5)
    // Une encre par slot : le poseur est le slot 3, la cible le slot 7.
    const ops = draw([sensorPose], TIME, { lives: [foe], sideOfSlot }, {
      colorOfSlot: (slot: number) => `slot${slot}`,
      neutral: 'neutre',
    })
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'slot3')).toBe(true)
    expect(ops.some((o) => o.op === 'set strokeStyle' && o.args[0] === 'slot7')).toBe(false)
  })

  it('un coéquipier du poseur n’est pas marqué : rien de plus que la zone', () => {
    const mate = life(4, 6, 5)
    expect(arcsOf(draw([sensorPose], TIME, { lives: [mate], sideOfSlot }))).toHaveLength(2)
  })

  it('sans camp connu, aucune marque — le ping se dessine quand même', () => {
    const foe = life(7, 6, 5)
    // Poseur non mesuré : le capteur existe et pinge, mais il ne révèle personne.
    const sansPoseur = pose({ family: 'sensor', id: SENSOR_ID, t0: 50, owner: -1 })
    expect(arcsOf(draw([sansPoseur], TIME, { lives: [foe], sideOfSlot }))).toHaveLength(2)
  })

  it('un mur seul ne révèle rien : la révélation appartient au capteur', () => {
    const foe = life(7, 6, 5)
    expect(arcsOf(draw([pose({ h: 90 })], TIME, { lives: [foe], sideOfSlot }))).toHaveLength(0)
  })

  it('un capteur LÂCHÉ ne révèle personne : il n’est même pas dessiné', () => {
    const foe = life(7, 6, 5)
    const lache = pose({ family: 'sensor', id: SENSOR_ID, t0: 50, origin: 'dropped' })
    expect(arcsOf(draw([lache], TIME, { lives: [foe], sideOfSlot }))).toHaveLength(0)
  })
})

describe('placementAt — le survol', () => {
  const hover = { frame: 50, frameMs: 100, showUnnamed: false }
  const center = projected(5, 5)

  it('la PLUS PETITE zone gagne : un mur posé dans un capteur reste atteignable', () => {
    const wall = pose({ family: 'wall', id: PANEL_ID })
    const sensor = pose({ family: 'sensor', id: SENSOR_ID })
    // Dans les DEUX ordres : c'est la taille qui tranche, jamais le rang dans la liste.
    expect(placementAt([sensor, wall], VIEW, hover, center)?.id).toBe(PANEL_ID)
    expect(placementAt([wall, sensor], VIEW, hover, center)?.id).toBe(PANEL_ID)
    // Et le capteur reste atteignable partout ailleurs dans sa zone (30 px = 3 m, à
    // l'intérieur des 4,25 m officiels).
    expect(placementAt([sensor, wall], VIEW, hover, { x: center.x + 30, y: center.y })?.id).toBe(
      SENSOR_ID,
    )
  })

  it('rien sous le curseur = null, jamais la pose la plus proche par défaut', () => {
    expect(placementAt([pose()], VIEW, hover, { x: 99, y: 99 })).toBeNull()
  })

  it('une pose éteinte par sa bascule ne se survole pas non plus', () => {
    const p = pose({ family: 'other' })
    expect(placementAt([p], VIEW, hover, center)).toBeNull()
    expect(placementAt([p], VIEW, { ...hover, showUnnamed: true }, center)?.id).toBe(p.id)
  })

  it('une pose ÉCARTÉE PAR SON ORIGINE ne se survole pas : le survol suit le dessin', () => {
    expect(placementAt([pose({ origin: 'dropped' })], VIEW, hover, center)).toBeNull()
    expect(placementAt([pose({ id: DEVICE_ID })], VIEW, hover, center)).toBeNull()
  })

  it("le TRAQUEUR ne se survole que pendant son impulsion — on n'inspecte pas ce qui n'est plus là", () => {
    const seeker = pose({ family: 'threat_seeker', id: SEEKER_ID, t0: 50, t1: 200 })
    expect(placementAt([seeker], VIEW, { ...hover, frame: 52 }, center)?.id).toBe(SEEKER_ID)
    expect(placementAt([seeker], VIEW, { ...hover, frame: 60 }, center)).toBeNull()
  })
})
