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
  WALL_DURATION_MS,
  WALL_OPENING_RAD,
  WALL_PANEL_IDS,
  WALL_RADIUS_M,
  wallArcWorld,
  wallHeading,
  wallRadiusM,
  wallRingWorld,
} from './placementWall'
import type { ReplayTrackReady } from './replayNormalize'
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
    // Trait franc, halo — et JAMAIS de pointillé sur une pose orientée.
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(2)
    expect(ops.some((o) => o.op === 'setLineDash')).toBe(false)
  })

  /**
   * R2-5 (2026-08-18) — LE MUR PORTE UN TOKEN FIXE, PLUS LA COULEUR D'ÉQUIPE.
   *
   * « Je préférerais un orange doré pour sa couleur » : c'est `warning` qui a été retenu
   * entre les deux tokens proposés. Ce mur est donc le SEUL objet posé dont la couleur dit
   * ce qu'il EST au lieu de dire qui l'a posé — l'utilisateur a explicitement accepté de
   * perdre le camp sur celui-là. Le test tient les deux moitiés de la règle.
   */
  it('l’arc prend l’encre FIXE du mur, jamais celle de l’équipe du poseur', () => {
    const ops = draw([pose({ h: 0 })])
    const encres = ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])
    expect(encres).toContain('mur')
    expect(encres).not.toContain('equipe')
  })

  it('le mur sans AUCUNE source de cap : le cercle fermé, dernier repli (0/62 mesuré)', () => {
    // Aucune vie dans la scène : le poseur n'a pas de piste, donc ni trajectoire ni visée.
    // C'est le seul cas où le cercle subsiste — cf. la chaîne en tête de placementWall.ts.
    const ops = draw([pose()])
    expect(ops.some((o) => o.op === 'setLineDash')).toBe(true)
    expect(ops.filter((o) => o.op === 'closePath').length).toBeGreaterThan(0)
  })

  it('une pose de mur sans poseur garde la MÊME encre : elle ne dépend pas du poseur', () => {
    const ops = draw([pose({ h: 0, owner: -1 })])
    const encres = ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])
    expect(encres).toContain('mur')
    expect(encres).not.toContain('equipe')
  })

  /**
   * LA FENÊTRE DU MUR, et elle se referme — règle changée le 2026-08-20 (demande utilisateur).
   *
   * `t1` date la mise au repos de l'objet, PAS sa disparition, que le film ne porte nulle part
   * (mesure du 2026-08-18, cf. placementEndFrame) : le mur survit donc à `t1`. Ce qui le
   * referme, c'est la durée OFFICIELLE — comme le capteur avant lui. La fixture bat à 100 ms
   * par image et pose le mur à `t0 = 10` : la fenêtre court jusqu'à l'image
   * 10 + 10 000/100 = 110.
   */
  it('le mur survit à `t1` mais s’efface au terme de sa durée officielle', () => {
    const mur = pose({ h: 45 }) // t0 = 10, t1 = 100
    const fin = 10 + WALL_DURATION_MS / TIME.frameMs
    expect(fin).toBe(110)
    // Avant la pose : rien.
    expect(painted([mur], { ...TIME, frame: 9 })).toBe(0)
    // Après `t1` (100) : le mur est TOUJOURS là — la mise au repos n'est pas la disparition.
    expect(painted([mur], { ...TIME, frame: 101 })).toBeGreaterThan(0)
    // À la dernière image de sa vie officielle : encore là.
    expect(painted([mur], { ...TIME, frame: fin })).toBeGreaterThan(0)
    // L'image d'après : plus rien. C'est le changement de règle.
    expect(painted([mur], { ...TIME, frame: fin + 1 })).toBe(0)
    expect(painted([mur], { ...TIME, frame: 600 })).toBe(0)
  })

  /**
   * LA BORNE MESURÉE L'EMPORTE. Un mur suivi 18,9 s existe au témoin : effacer à 10 s
   * effacerait un objet que la mesure montre encore vivant. La fenêtre couvre donc toujours
   * au moins `t1`, exactement comme pour le capteur.
   */
  it('un mur suivi PLUS LONGTEMPS que sa durée officielle reste jusqu’à `t1`', () => {
    const suiviLongtemps = pose({ h: 45, t1: 200 }) // 19 s de suivi, contre 10 s officielles
    expect(painted([suiviLongtemps], { ...TIME, frame: 150 })).toBeGreaterThan(0)
    expect(painted([suiviLongtemps], { ...TIME, frame: 200 })).toBeGreaterThan(0)
    expect(painted([suiviLongtemps], { ...TIME, frame: 201 })).toBe(0)
  })

  /** Sans cadence d'images, aucune durée ne se convertit : on ne referme rien au hasard. */
  it('sans `frameMs`, la fenêtre reste ouverte plutôt que de se fermer sur une conversion fausse', () => {
    expect(painted([pose({ h: 45 })], { ...TIME, frame: 400, frameMs: 0 })).toBeGreaterThan(0)
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
    expect(placementKind(appareil, { showUnnamed: true, showDropped: true })).toBeNull()
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

/**
 * V3 (retour utilisateur du 2026-08-18) — LA CHAÎNE DE CAP DU MUR.
 *
 * « Sans cap je préférerais qu'on tente de corréler la visée ou la trajectoire du joueur, un
 * mur portatif rond serait trop troublant. » La chaîne est donc : cap de la POSE, puis
 * TRAJECTOIRE du poseur (dernier déplacement d'au moins 0,5 m), puis VISÉE de la dernière
 * image qui en porte une. Le rond ne reste que pour un poseur sans piste — 0 cas sur 62.
 */
describe('wallHeading — la chaîne des trois sources (V3, 2026-08-18)', () => {
  /** Une vie qui va vers l'EST (X croissants) puis s'arrête juste avant la pose. */
  function versEst(slot: number): ReplayTrackReady {
    return {
      slot,
      team: -1,
      startFrame: 0,
      endFrame: 20,
      points: [
        { t: 0, x: 0, y: 5 },
        { t: 5, x: 3, y: 5 },
        // Deux derniers points quasi confondus : le bruit d'un joueur à l'arrêt, que le
        // seuil de 0,5 m doit refuser au profit du segment qui précède.
        { t: 9, x: 4.95, y: 5 },
        { t: 10, x: 5, y: 5 },
      ],
    }
  }

  it('cap de la POSE : il prime sur tout le reste, même quand la trajectoire existe', () => {
    const h = wallHeading(pose({ h: 90, owner: 3 }), [versEst(3)])
    expect(h).toEqual({ deg: 90, source: 'placement' })
  })

  it('cap NUL de la pose est une VALEUR, pas une absence (un poseur peut viser 0°)', () => {
    expect(wallHeading(pose({ h: 0, owner: 3 }), [versEst(3)])).toEqual({
      deg: 0,
      source: 'placement',
    })
  })

  it('sans cap : la TRAJECTOIRE du poseur, prise sur le dernier segment d au moins 0,5 m', () => {
    const h = wallHeading(pose({ owner: 3 }), [versEst(3)])
    expect(h?.source).toBe('trajectory')
    // Le joueur arrivait vers les X croissants : 0° dans la convention de `Point.h`.
    expect(h?.deg).toBeCloseTo(0, 6)
  })

  it('le seuil de 0,5 m ÉCARTE le bruit : deux points à 5 cm ne disent aucune direction', () => {
    const immobile: ReplayTrackReady = {
      slot: 3, team: -1, startFrame: 0, endFrame: 10,
      points: [{ t: 9, x: 4.95, y: 5 }, { t: 10, x: 5, y: 5, h: 180 }],
    }
    const h = wallHeading(pose({ owner: 3 }), [immobile])
    // Faute de déplacement mesurable, on tombe sur la VISÉE — jamais sur un angle de bruit.
    expect(h).toEqual({ deg: 180, source: 'aim' })
  })

  it('ni cap ni trajectoire ni visée : null — et c est là, et seulement là, que le rond sert', () => {
    const sansRien: ReplayTrackReady = {
      slot: 3, team: -1, startFrame: 0, endFrame: 10,
      points: [{ t: 10, x: 5, y: 5 }],
    }
    expect(wallHeading(pose({ owner: 3 }), [sansRien])).toBeNull()
    expect(wallHeading(pose({ owner: 3 }), [])).toBeNull()
  })

  it('la vie lue est celle du SLOT du poseur, jamais celle d un voisin', () => {
    expect(wallHeading(pose({ owner: 7 }), [versEst(3)])).toBeNull()
  })

  it('les points POSTÉRIEURS à la pose ne comptent pas : on n oriente pas par l avenir', () => {
    const apres: ReplayTrackReady = {
      slot: 3, team: -1, startFrame: 0, endFrame: 40,
      points: [
        { t: 0, x: 0, y: 5 },
        { t: 5, x: 3, y: 5 },
        { t: 10, x: 5, y: 5 },
        // Après t0 = 10, le joueur repart plein NORD : cela ne doit rien changer.
        { t: 20, x: 5, y: 9 },
      ],
    }
    const h = wallHeading(pose({ owner: 3, t0: 10 }), [apres])
    expect(h?.source).toBe('trajectory')
    expect(h?.deg).toBeCloseTo(0, 6)
  })
})

describe('drawWall — un cap DÉDUIT se voit (V3)', () => {
  it('cap de la pose : arc FRANC, aucun pointillé', () => {
    const ops = draw([pose({ h: 90, owner: 3 })])
    expect(ops.some((o) => o.op === 'setLineDash')).toBe(false)
    expect(ops.filter((o) => o.op === 'closePath')).toHaveLength(0)
  })

  it('cap déduit de la trajectoire : un ARC (jamais un cercle), mais POINTILLÉ', () => {
    const vie: ReplayTrackReady = {
      slot: 3, team: -1, startFrame: 0, endFrame: 20,
      points: [{ t: 0, x: 0, y: 5 }, { t: 10, x: 5, y: 5 }],
    }
    const ops = draw([pose({ owner: 3 })], TIME, { lives: [vie] })
    expect(ops.some((o) => o.op === 'setLineDash')).toBe(true)
    // OUVERT : un cercle se referme, un arc non. C'est la différence que l'utilisateur voit.
    expect(ops.filter((o) => o.op === 'closePath')).toHaveLength(0)
    // Et le milieu de l'arc est bien devant le poseur, dans la direction où il arrivait.
    const mid = projected(5 + wallRadiusM(VIEW), 5)
    expect(
      ops.some(
        (o) =>
          o.op === 'lineTo' &&
          Math.abs((o.args[0] as number) - mid.x) < 1e-6 &&
          Math.abs((o.args[1] as number) - mid.y) < 1e-6,
      ),
    ).toBe(true)
  })
})
