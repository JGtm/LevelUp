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
 *  - LA FENÊTRE D'AFFICHAGE : `t1` date la mise au repos de l'objet et ne referme rien — le
 *    capteur se tient à sa durée officielle, les autres poses vont jusqu'à la fin du rejeu ;
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
  placementEndFrame,
  placementIsDeployedObject,
  placementKind,
  placementOrigin,
  type PlacementToggles,
  countDrawablePlacements,
} from './equipmentPlacementsLayer'
import { placementAt } from './placementHitTest'
import {
  buildSlotOwnership,
  colorResolver,
  colorResolverOrLast,
  type ReplayPlayer,
} from './rosterLogic'
import {
  BEACON_ID,
  DEVICE_ID,
  OVERSHIELD_ID,
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
import { WALL_DURATION_MS } from './placementWall'
import { REVEAL_RADIUS_PX, SENSOR_DURATION_MS, SENSOR_RADIUS_M } from './threatSensor'

/**
 * LES DEUX JEUX DE BASCULES de ce fichier. `RIEN` est le comportement HISTORIQUE du calque (seuls
 * les objets déployés) ; `TOUT` allume les deux commandes du tiroir, ce que fait déjà le comptage
 * (`countDrawablePlacements`) pour savoir si une commande a quelque chose à commander.
 */
const RIEN: PlacementToggles = { showUnnamed: false, showDropped: false }
const TOUT: PlacementToggles = { showUnnamed: true, showDropped: true }

describe('PLACEMENT_RENDER — la table, famille par famille', () => {
  it('les six familles DÉPLOYABLES ont chacune leur règle de rendu', () => {
    expect(PLACEMENT_RENDER.wall).toBe('wall')
    expect(PLACEMENT_RENDER.sensor).toBe('sensor')
    // La FAMILLE garde son nom (identifiant stable du document) ; c est le RENDU qui change.
    expect(PLACEMENT_RENDER.translocator_beacon).toBe('rift')
    expect(PLACEMENT_RENDER.threat_seeker).toBe('seeker')
    expect(PLACEMENT_RENDER.repair_field).toBe('field')
    expect(PLACEMENT_RENDER.other).toBe('unnamed')
    // L ECRAN OCCULTANT a suivi trois etapes distinctes le 2026-08-27, dans cet ordre : nomme
    // (sa banque sonore le dit), sonore, puis DESSINE une fois le verdict rendu sur les trois
    // propositions de la planche — opaque, et les pions au-dessus.
    expect(PLACEMENT_RENDER.shroud_screen).toBe('shroud')
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
      expect(placementKind(pose({ family: f }), TOUT)).toBeNull()
    }
  })

  it('les POWER-UPS sont hors table : aucune règle, pas même `null` (correction 3 de la mesure)', () => {
    // n = 1 par power-up sur les 11 films, les deux avec poseur et `dropped` : rien ne soutient
    // l'hypothèse « objet de la carte sans poseur ». Pas de vocabulaire mort (CLAUDE.md n°7).
    for (const f of ['powerup_overshield', 'powerup_camo']) {
      expect(f in PLACEMENT_RENDER, `${f} doit rester HORS table`).toBe(false)
      expect(placementKind(pose({ family: f }), TOUT)).toBeNull()
    }
  })

  it('une famille inconnue du manifeste ne dessine RIEN', () => {
    expect(placementKind(pose({ family: 'famille_future' }), TOUT)).toBeNull()
    expect(painted([pose({ family: 'famille_future', h: 12 })])).toBe(0)
  })
})

describe('placementOrigin / placementIsDeployedObject — le filtre du schéma 10', () => {
  it("l'origine ABSENTE se lit `unknown`, JAMAIS `deployed` (parc antérieur au schéma 10)", () => {
    const sansOrigine = pose()
    delete sansOrigine.origin
    expect(placementOrigin(sansOrigine)).toBe('unknown')
    expect(placementIsDeployedObject(sansOrigine)).toBe(false)
    expect(placementKind(sansOrigine, RIEN)).toBeNull()
  })

  it('un LÂCHER ne se dessine pas — 88,6 % du corpus en est', () => {
    for (const origin of ['dropped', 'unknown']) {
      expect(placementIsDeployedObject(pose({ origin })), origin).toBe(false)
      expect(placementKind(pose({ origin }), RIEN), origin).toBeNull()
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
    expect(placementKind(lache, RIEN)).toBeNull()
    expect(placementKind(lache, { showUnnamed: true, showDropped: false })).toBe('unnamed')
    expect(painted([lache])).toBe(0)
    expect(painted([lache], { ...TIME, showUnnamed: true })).toBe(1)
  })
})

describe('placementEndFrame — `t1` n’est PAS la disparition', () => {
  it('rien avant t0, et t1 ne referme RIEN : le film ne date aucune disparition', () => {
    // `t1 = 20` ne ferme pas la fenêtre — c'est la mise au repos, pas la disparition. Ce qui
    // la ferme, pour un mur, c'est sa durée OFFICIELLE (110 avec t0 = 10) ; l'image 21 est
    // donc encore active, l'image 599 ne l'est plus.
    const p = pose({ t0: 10, t1: 20 })
    expect([9, 10, 20, 21, 110].map((f) => isPlacementActive(p, 'wall', f, TIME))).toEqual([
      false, true, true, true, true,
    ])
    expect(isPlacementActive(p, 'wall', 111, TIME)).toBe(false)
    expect(isPlacementActive(p, 'wall', 599, TIME)).toBe(false)
  })

  it('le capteur se tient à sa durée OFFICIELLE : 15 s, soit 150 images de 100 ms', () => {
    const p = pose({ t0: 10, t1: 20, family: 'sensor' })
    expect(placementEndFrame(p, 'sensor', TIME)).toBe(10 + SENSOR_DURATION_MS / TIME.frameMs)
    expect(isPlacementActive(p, 'sensor', 161, TIME)).toBe(false)
  })

  it('le mur se tient à la sienne : une dizaine de secondes, soit 100 images de 100 ms', () => {
    const p = pose({ t0: 10, t1: 20 })
    expect(placementEndFrame(p, 'wall', TIME)).toBe(10 + WALL_DURATION_MS / TIME.frameMs)
  })

  it('la borne MESURÉE l’emporte pour le mur aussi : un mur suivi 19 s reste jusqu’à `t1`', () => {
    expect(placementEndFrame(pose({ t0: 10, t1: 200 }), 'wall', TIME)).toBe(200)
  })

  it('la borne MESURÉE l’emporte quand elle dépasse la durée officielle', () => {
    const p = pose({ t0: 10, t1: 400, family: 'sensor' })
    expect(placementEndFrame(p, 'sensor', TIME)).toBe(400)
  })

  it('une pose sans famille à durée publiée va jusqu’à la dernière image du rejeu', () => {
    expect(placementEndFrame(pose({ t0: 10, t1: 20 }), 'unnamed', TIME)).toBe(599)
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

  it('les objets non identifiés sont muets par défaut, et un point neutre une fois demandés', () => {
    expect(draw([pose({ family: 'other' })]).filter((o) => o.op === 'fill')).toHaveLength(0)
    const on = draw([pose({ family: 'other' })], { ...TIME, showUnnamed: true })
    expect(on.filter((o) => o.op === 'fill')).toHaveLength(1)
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
      wall: 'mur',
      rift: { rim: 'faille-bord', core: 'faille-coeur' },
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

/**
 * RÉGRESSION DE FRONTIÈRE (revue adversariale 2026-08-28) — la couleur d'un objet LÂCHÉ À LA MORT.
 *
 * Un objet `origin='dropped'` porte `t0 = finVie du poseur + 1` : le poseur n'occupe DÉJÀ PLUS le
 * slot à `t0`. Le double `colorOfSlot` doit donc être un résolveur de FRONTIÈRE
 * (`ownerAtFrameOrLast`) : strict, il rendrait `null` → l'objet serait peint en NEUTRE au lieu de
 * la couleur d'équipe du lâcheur. Le double est ici FRAME-DÉPENDANT (un vrai résolveur bâti sur
 * une vie fournie), ce qui démasque le bug qu'un `(slot)=>couleur` figé cachait.
 */
describe('couleur d’un objet LÂCHÉ à la mort (t0 = finVie + 1)', () => {
  const dropTime = { ...TIME, showDropped: true, frame: 60 }
  // Le lâcheur : sa vie sur le slot 3 s'est terminée à l'image 49 (l'objet est lâché à 50).
  const dropper: ReplayPlayer = {
    xuid: 'A',
    lives: [
      { slot: 3, team: -1, startFrame: 1, endFrame: 49, points: [{ t: 1, x: 5, y: 5 }] },
    ] as ReplayPlayer['lives'],
  }
  const own = buildSlotOwnership([dropper])
  const teamColor = (ally: boolean) => (ally ? 'equipe' : 'adverse')
  const dropped = pose({ family: 'sensor', id: SENSOR_ID, origin: 'dropped', owner: 3, t0: 50 })
  const inkWith = (colorOfSlot: (slot: number, frame: number) => string | null) => ({
    colorOfSlot, neutral: 'neutre', wall: 'mur', rift: { rim: 'faille-bord', core: 'faille-coeur' },
  })
  const strokesOf = (ops: ReturnType<typeof draw>) =>
    ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])

  it('résolution STRICTE (ownerAtFrame) : peint en NEUTRE — la régression', () => {
    const strokes = strokesOf(
      draw([dropped], dropTime, {}, inkWith(colorResolver(own, teamColor, () => true, 'neutre'))),
    )
    expect(strokes).toContain('neutre')
    expect(strokes).not.toContain('equipe')
  })

  it('résolution de FRONTIÈRE (ownerAtFrameOrLast) : prend la couleur d’équipe du LÂCHEUR', () => {
    const strokes = strokesOf(
      draw([dropped], dropTime, {}, inkWith(colorResolverOrLast(own, teamColor, () => true, 'neutre'))),
    )
    expect(strokes).toContain('equipe')
    expect(strokes).not.toContain('neutre')
  })
})

describe('placementAt — le survol', () => {
  const hover = {
    frame: 50,
    frameMs: TIME.frameMs,
    frames: TIME.frames,
    showUnnamed: false,
    showDropped: false,
  }
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

/**
 * LES OBJETS DE PUISSANCE LÂCHÉS (décision produit du 2026-08-18) — et le comptage qui décide
 * si la commande du tiroir s'affiche.
 *
 * Ce bloc défend trois choses que rien d'autre ne défend : que la bascule ALLUMÉE ne change
 * RIEN aux déployés (c'est la promesse de non-régression du lot), qu'un lâcher se dessine avec
 * SA forme et non celle de sa famille, et que le comptage passe par la même porte que le tracé.
 */
describe('les objets de PUISSANCE lâchés — la troisième porte de placementKind', () => {
  const LACHE: PlacementToggles = { showUnnamed: false, showDropped: true }

  it('un power-up lâché se dessine — le surbouclier du témoin 01e1f945', () => {
    const p = pose({ family: 'powerup_overshield', id: OVERSHIELD_ID, origin: 'dropped' })
    expect(placementKind(p, LACHE)).toBe('dropped')
    expect(painted([p], { ...TIME, showDropped: true })).toBe(1)
  })

  it('un équipement lâché se dessine, et l’APPAREIL du mur y compris — témoin 000d5950', () => {
    // Les 11 `wall/dropped` du témoin portent l'identifiant de l'APPAREIL : la règle des
    // panneaux ne s'applique qu'au DÉPLOIEMENT, un lâcher ne publie qu'une pose.
    const mur = pose({ family: 'wall', id: DEVICE_ID, origin: 'dropped' })
    const capteur = pose({ family: 'sensor', id: SENSOR_ID, origin: 'dropped' })
    expect(placementKind(mur, LACHE)).toBe('dropped')
    expect(placementKind(capteur, LACHE)).toBe('dropped')
    expect(painted([mur, capteur], { ...TIME, showDropped: true })).toBe(2)
  })

  it('un capteur lâché n’a NI zone NI onde : une seule primitive, pas celle de sa famille', () => {
    // Un capteur déployé émet son disque, son anneau et parfois son ping : au moins trois.
    const lache = pose({ family: 'sensor', id: SENSOR_ID, origin: 'dropped', t0: 48 })
    const deploye = pose({ family: 'sensor', id: SENSOR_ID, origin: 'deployed', t0: 48 })
    expect(painted([lache], { ...TIME, showDropped: true })).toBe(1)
    expect(painted([deploye], { ...TIME, showDropped: true })).toBeGreaterThan(1)
  })

  it('les grenades et les capacités lâchées restent MUETTES, bascule allumée ou non', () => {
    const muets = ['grenade_frag', 'grenade_plasma', 'grenade_dynamo', 'grenade_spike',
      'grapple', 'thruster', 'repulsor']
    for (const family of muets) {
      const p = pose({ family, origin: 'dropped' })
      expect(placementKind(p, LACHE), family).toBeNull()
      expect(painted([p], { ...TIME, showDropped: true }), family).toBe(0)
    }
  })

  it('la bascule ÉTEINTE rend le comportement d’avant le lot, à la primitive près', () => {
    const p = pose({ family: 'powerup_overshield', id: OVERSHIELD_ID, origin: 'dropped' })
    expect(placementKind(p, RIEN)).toBeNull()
    expect(painted([p])).toBe(0)
  })

  it('la bascule ALLUMÉE ne change RIEN aux objets DÉPLOYÉS — la non-régression du lot', () => {
    const attendu: [string, string][] = [
      ['wall', PANEL_ID],
      ['sensor', SENSOR_ID],
      ['translocator_beacon', BEACON_ID],
      ['threat_seeker', SEEKER_ID],
      ['repair_field', FIELD_ID],
    ]
    for (const [family, id] of attendu) {
      const p = pose({ family, id, t0: 48, h: 90 })
      expect(painted([p], { ...TIME, showDropped: true }), family).toBe(painted([p]))
    }
  })

  it('un lâché reste à l’écran jusqu’à la fin du rejeu : le film ne date aucune disparition', () => {
    const p = pose({ family: 'powerup_overshield', id: OVERSHIELD_ID, origin: 'dropped', t0: 10, t1: 20 })
    expect(placementEndFrame(p, 'dropped', TIME)).toBe(TIME.frames - 1)
  })
})

describe('countDrawablePlacements — la porte du comptage est celle du tracé', () => {
  it('compte les deux bascules ALLUMÉES, sinon la commande ne s’afficherait jamais', () => {
    const n = countDrawablePlacements([
      pose(),
      pose({ family: 'other', origin: 'dropped' }),
      pose({ family: 'powerup_overshield', id: OVERSHIELD_ID, origin: 'dropped' }),
      pose({ family: 'sensor', id: SENSOR_ID, origin: 'dropped' }),
      pose({ family: 'grenade_frag', origin: 'dropped' }),
    ])
    // Le mur déployé, l'objet non identifié, le power-up lâché et le capteur lâché : 4.
    // La grenade lâchée n'entre nulle part.
    expect(n).toEqual({ drawable: 4, unnamed: 1, dropped: 2 })
  })

  it('un film SANS lâcher de puissance rend `dropped: 0` — le tiroir n’affiche alors rien', () => {
    const n = countDrawablePlacements([pose(), pose({ family: 'grenade_spike', origin: 'dropped' })])
    expect(n.dropped).toBe(0)
  })
})

describe('le LIEN de téléportation', () => {
  const passage = { slot: 3, frame: 48, from: { x: 1, y: 1 }, to: { x: 9, y: 9 }, viaRift: false }

  /** Les opérations d un tracé de lien : le pointillé est sa signature. */
  function liens(frame: number) {
    return draw([], { ...TIME, frame }, { teleports: [passage] })
      .filter((o) => o.op === 'setLineDash' && Array.isArray(o.args[0]) && o.args[0].length > 0)
  }

  it('se trace pendant les 600 ms qui suivent le passage, et pas au-delà', () => {
    // frameMs vaut 100 dans la fixture : la fenêtre couvre les frames 48 à 53. La frame 54
    // tombe EXACTEMENT sur les 600 ms — l effacement y est complet, donc plus rien n est tracé.
    expect(liens(48)).toHaveLength(1)
    expect(liens(53)).toHaveLength(1)
    expect(liens(54)).toHaveLength(0)
    expect(liens(55)).toHaveLength(0)
  })

  it('ne se trace pas AVANT le passage — un lien n annonce rien', () => {
    expect(liens(47)).toHaveLength(0)
  })

  it('porte les encres de la faille, jamais la couleur d équipe du joueur', () => {
    const encres = draw([], { ...TIME, frame: 49 }, { teleports: [passage] })
      .filter((o) => o.op === 'set strokeStyle')
      .map((o) => o.args[0])
    expect(encres).toContain('faille-coeur')
    expect(encres).not.toContain('equipe')
  })

  it('relie les DEUX positions mesurées : le départ et l arrivée', () => {
    const ops = draw([], { ...TIME, frame: 49 }, { teleports: [passage] })
    const depart = projected(1, 1)
    const arrivee = projected(9, 9)
    const courbe = ops.find((o) => o.op === 'quadraticCurveTo' && o.args[2] === arrivee.x)
    expect(courbe).toBeDefined()
    expect(ops.some((o) => o.op === 'moveTo' && o.args[0] === depart.x && o.args[1] === depart.y)).toBe(true)
  })

  it('sans passage, le calque n émet rien de plus qu avant', () => {
    expect(draw([], TIME, { teleports: [] }).filter((o) => o.op === 'setLineDash')).toHaveLength(0)
  })
})


describe("l ECRAN OCCULTANT — une bulle opaque, et les pions au-dessus", () => {
  const ecran = () => pose({ family: 'shroud_screen', id: 'sh1' })

  it('trace un disque plein a l encre NEUTRE, jamais a celle de l equipe', () => {
    const ops = draw([ecran()])
    const encres = ops.filter((o) => o.op === 'set fillStyle').map((o) => o.args[0])
    expect(encres).toContain('neutre')
    expect(encres).not.toContain('equipe')
  })

  it('le bord est un FONDU, pas une borne : un degrade et AUCUN anneau trace', () => {
    const ops = draw([ecran()])
    expect(ops.filter((o) => o.op === 'createRadialGradient')).toHaveLength(1)
    // Le champ de reparation borne son disque d un pointille parce qu il a une portee dont on
    // doute ; l ecran n a pas de borne du tout, parce qu il n a pas de portee connue.
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(0)
    expect(ops.filter((o) => o.op === 'setLineDash')).toHaveLength(0)
  })

  it('sa zone sensible suit sa bulle — on le designe partout, pas seulement au centre', () => {
    const p = ecran()
    const centre = projected(5, 5)
    // 6 m a 10 px par metre : un point a 4 m du centre est DANS la bulle.
    expect(placementAt([p], VIEW, TIME, { x: centre.x + 40, y: centre.y })?.id).toBe('sh1')
  })
})
