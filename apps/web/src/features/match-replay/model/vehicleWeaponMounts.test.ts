/**
 * vehicleWeaponMounts.test.ts — la géométrie pure (rotation d'ancre aux quatre points cardinaux,
 * distinction fixe/tourelle). Aucun canvas. Les ANCRES elles-mêmes viennent du registre du titre
 * depuis le schéma 69 : leur lecture est testée dans `vehicleWeaponRegistry.test.ts`.
 */
import { describe, expect, it } from 'vitest'

import { vehicleSpriteScale } from './vehiclesLayer'
import { vehicleShotPlacement, type VehicleWeaponMount } from './vehicleWeaponMounts'

describe('vehicleShotPlacement — rotation de l’ancre par le cap du véhicule', () => {
  const size = { naturalWidthPx: 100, naturalHeightPx: 200, mmPerPx: 10 }
  // `vehicleSpriteScale` n'est PAS l'identité (plancher/plafond doux de `vehiclesLayer.ts`) :
  // la même primitive que le tracé du sprite, donc le même facteur ici — jamais recalculé.
  // L'ECHELLE DU CADRAGE : 20 px/m. Le sprite mesure 200 px x 10 mm/px = 2 m de long, donc
  // 40 px a l'ecran — au-dessus du minimum garanti (18,48 px), le terme REALISTE s'applique
  // et ces tests de geometrie portent bien sur le cas nominal (cf. model/screenSizes.ts).
  const ECHELLE = 20
  const SCALE = vehicleSpriteScale(size.naturalHeightPx, size.mmPerPx, ECHELLE)
  /**
   * CAPS — LE CAS SANS VISÉE DE TIREUR, c'est-à-dire l'état d'avant le lot 5.5 : ces tests de
   * GÉOMÉTRIE ne portent que sur le cap du châssis, le seul qui place une ancre. Les tests de
   * DIRECTION posent leurs deux caps explicitement.
   */
  // `arme: 'vehicule'` est le cas nominal de ces tests (lot 5.8.5) : une arme DE VEHICULE, donc
  // un chassis dont le cap est un repli legitime quand le montage manque.
  const CAPS = (chassisDeg: number) => ({ chassisDeg, tireurDeg: null, arme: 'vehicule' as const })
  // Ancre nez pur (ax=0, ay=-0,5) : au bord haut du sprite AVANT rotation, comme
  // `drawRotatedSprite` dessine `drawImage(img, -w/2, -h/2, w, h)` (ay=-0,5 -> y local = -h/2).
  const nose: VehicleWeaponMount = { classe: 'fixe', ax: 0, ay: -0.5 }
  // Ancre latérale pure (ax=+0,5, ay=0) : au bord droit du sprite avant rotation.
  const rightSide: VehicleWeaponMount = { classe: 'fixe', ax: 0.5, ay: 0 }

  it('cap 90° (vehicleScreenAngle = 0) : repère local = repère écran, sans rotation', () => {
    const p = vehicleShotPlacement(nose, CAPS(90), size, 1, ECHELLE)
    // localY = -0,5 * 200 = -100 ; screenAngle(90) = 0 -> offset = (0, -100) * SCALE.
    expect(p.offset.x).toBeCloseTo(0, 6)
    expect(p.offset.y).toBeCloseTo(-100 * SCALE, 6)
  })

  it('cap 0° (monde +X = droite écran) : le nez pointe vers +X', () => {
    const p = vehicleShotPlacement(nose, CAPS(0), size, 1, ECHELLE)
    expect(p.offset.x).toBeCloseTo(100 * SCALE, 6)
    expect(p.offset.y).toBeCloseTo(0, 6)
  })

  it('cap 180° (monde -X = gauche écran) : le nez pointe vers -X', () => {
    const p = vehicleShotPlacement(nose, CAPS(180), size, 1, ECHELLE)
    expect(p.offset.x).toBeCloseTo(-100 * SCALE, 6)
    expect(p.offset.y).toBeCloseTo(0, 6)
  })

  it('cap 270° (monde -Y = bas écran) : le nez pointe vers +Y écran (bas)', () => {
    const p = vehicleShotPlacement(nose, CAPS(270), size, 1, ECHELLE)
    expect(p.offset.x).toBeCloseTo(0, 6)
    expect(p.offset.y).toBeCloseTo(100 * SCALE, 6)
  })

  it('une ancre latérale tourne comme une ancre longitudinale (même transform)', () => {
    const p = vehicleShotPlacement(rightSide, CAPS(0), size, 1, ECHELLE)
    // localX = 0,5 * 100 = 50 ; screenAngle(0) = 90° -> (localX*cos90 - localY*sin90, localX*sin90+...)
    // = (0 - 0, 50 + 0) = (0, 50) * SCALE.
    expect(p.offset.x).toBeCloseTo(0, 6)
    expect(p.offset.y).toBeCloseTo(50 * SCALE, 6)
  })

  it('la densité k met le décalage à l’échelle (aucune rotation supplémentaire)', () => {
    const p1 = vehicleShotPlacement(nose, CAPS(90), size, 1, ECHELLE)
    const p2 = vehicleShotPlacement(nose, CAPS(90), size, 2, ECHELLE)
    expect(p2.offset.y).toBeCloseTo(p1.offset.y * 2, 6)
  })

  /**
   * CLASSE TOURELLE : LE CAP DU CHÂSSIS NE L'ORIENTE JAMAIS, et c'est la règle qui SURVIT au lot
   * 5.5 — une tourelle tourne indépendamment du corps. Ce qui change, c'est qu'une visée de
   * TIREUR, quand elle est lue, l'oriente désormais (cas suivant).
   */
  it('classe tourelle SANS visée de tireur : direction null, quel que soit le cap du châssis', () => {
    const turret: VehicleWeaponMount = { classe: 'tourelle', ax: 0, ay: 0 }
    expect(vehicleShotPlacement(turret, CAPS(45), size, 1, ECHELLE).angle).toBeNull()
    expect(vehicleShotPlacement(turret, CAPS(270), size, 1, ECHELLE).angle).toBeNull()
  })

  /**
   * LE POINT DU LOT 5.5 : la visée MESURÉE du tireur oriente la tourelle, et elle ne doit RIEN
   * devoir au cap du châssis — d'où un cap de châssis franchement différent dans ce cas.
   */
  it('classe tourelle AVEC visée de tireur : direction = visée du tireur, pas le cap du châssis', () => {
    const turret: VehicleWeaponMount = { classe: 'tourelle', ax: 0, ay: 0 }
    const p = vehicleShotPlacement(turret, { chassisDeg: 10, tireurDeg: 200, arme: 'vehicule' as const }, size, 1, ECHELLE)
    expect(p.angle).toBeCloseTo((-200 * Math.PI) / 180, 10)
  })

  /**
   * LE MONTAGE, LUI, SUIT LE CHÂSSIS : l'ancre est un point du SPRITE, donc elle tourne avec
   * l'image. Un décalage calculé sur la visée du tireur sortirait l'éclair du châssis.
   */
  it('classe tourelle : le DÉCALAGE suit le châssis, jamais la visée du tireur', () => {
    const turret: VehicleWeaponMount = { classe: 'tourelle', ax: 0, ay: -0.5 }
    const sansVisee = vehicleShotPlacement(turret, CAPS(90), size, 1, ECHELLE)
    const avecVisee = vehicleShotPlacement(turret, { chassisDeg: 90, tireurDeg: 200, arme: 'vehicule' as const }, size, 1, ECHELLE)
    expect(avecVisee.offset).toEqual(sansVisee.offset)
  })

  /**
   * UNE ARME FIXE NE CHANGE PAS DE SOURCE, et le test le dit : sur ces familles le cap du
   * châssis EST déjà la visée du conducteur depuis 5.2a.6 (écart mesuré 0,0 degré), donc lui
   * substituer la visée du tireur n'apporterait rien et brouillerait la règle.
   */
  it('classe fixe : la visée du tireur NE prend PAS le pas sur le cap du châssis', () => {
    const p = vehicleShotPlacement(nose, { chassisDeg: 33, tireurDeg: 200, arme: 'vehicule' as const }, size, 1, ECHELLE)
    expect(p.angle).toBeCloseTo((-33 * Math.PI) / 180, 10)
  })

  /**
   * MONTAGE INCONNU (2026-09-20) : l'éclair reste au CENTRE — la seule position que le document
   * donne —, mais il garde la DIRECTION du véhicule. Sans cela, un tir d'arme de véhicule non
   * documentée perdait toute direction et tombait sur la bouffée ronde, invisible sur un
   * châssis. Le Wraith, le Gungoose et le Falcon sont mesurés depuis le lot 5.8.3 ; le cas
   * reste vivant pour le Shade, dont le tag `weap` n'est pas documenté.
   */
  it('montage INCONNU : aucun décalage, mais le cap du véhicule donne la direction', () => {
    const p = vehicleShotPlacement(null, CAPS(90), size, 1, ECHELLE)
    expect(p.offset).toEqual({ x: 0, y: 0 })
    expect(p.angle).toBeCloseTo((-90 * Math.PI) / 180, 10)
  })

  /**
   * LOT 5.8.5 — SANS MONTAGE, LA DIRECTION DÉPEND DE CE QUI A TIRÉ.
   *
   * Une arme DE VÉHICULE est solidaire du corps : le cap du châssis est son repli légitime, et il
   * ne change pas. Une arme DE JOUEUR tirée d'un siège ne l'est pas — le passager vise où il veut
   * (règle du lot 5.2a.5, qui SURVIT) : elle n'accepte que la visée mesurée de son tireur.
   */
  it('sans montage, ARME DE VÉHICULE : le cap du châssis, inchangé depuis 5.2a.5', () => {
    const p = vehicleShotPlacement(
      null,
      { chassisDeg: 90, tireurDeg: 200, arme: 'vehicule' },
      size,
      1,
      ECHELLE,
    )
    expect(p.angle).toBeCloseTo((-90 * Math.PI) / 180, 10)
  })

  it('sans montage, ARME DE JOUEUR : la visée du tireur, JAMAIS le cap du châssis', () => {
    const p = vehicleShotPlacement(
      null,
      { chassisDeg: 90, tireurDeg: 200, arme: 'joueur' },
      size,
      1,
      ECHELLE,
    )
    expect(p.angle).toBeCloseTo((-200 * Math.PI) / 180, 10)
  })

  it('sans montage, ARME DE JOUEUR sans visée lue : aucune direction (jamais le châssis)', () => {
    const p = vehicleShotPlacement(
      null,
      { chassisDeg: 90, tireurDeg: null, arme: 'joueur' },
      size,
      1,
      ECHELLE,
    )
    expect(p.angle).toBeNull()
    expect(p.offset).toEqual({ x: 0, y: 0 })
  })

  it('classe fixe : direction = vehicleAimAngle(cap), jamais null', () => {
    const p = vehicleShotPlacement(nose, CAPS(33), size, 1, ECHELLE)
    expect(p.angle).not.toBeNull()
    expect(p.angle).toBeCloseTo((-33 * Math.PI) / 180, 10)
  })
})
