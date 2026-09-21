/**
 * vehicleWeaponMounts.test.ts — la table (bornes, repli tag inconnu) et la géométrie pure
 * (rotation d'ancre aux quatre points cardinaux, distinction fixe/tourelle). Aucun canvas.
 */
import { describe, expect, it } from 'vitest'

import { vehicleSpriteScale } from './vehiclesLayer'
import {
  vehicleShotPlacement,
  vehicleWeaponMountOf,
  type VehicleWeaponMount,
} from './vehicleWeaponMounts'

// Gabarit RÉEL de `Shot.w` pour une arme de véhicule, vérifié en direct (artefact `0d76e8f1` :
// `0xC7D5091200000000` pour le Warthog, `0x11725DC400000000` pour le Wasp — moitié haute =
// tag `weap` de V3F_TIRS_COVENANT_2026-09-02.md, moitié basse nulle). Répété ici plutôt
// qu'importé (`vehicleWeapTag` n'est pas exportée : c'est un détail d'assemblage de la table,
// pas une API publique) pour garder le test capable de détecter un changement de gabarit.
function shotW(weap8hex: string): string {
  return `0x${weap8hex.toUpperCase()}00000000`
}

describe('vehicleWeaponMountOf — la table et son repli', () => {
  it('un tag inconnu rend null (repli centre, comportement d’avant ce fichier)', () => {
    expect(vehicleWeaponMountOf('0xFFFFFFFF00000000')).toBeNull()
    expect(vehicleWeaponMountOf(undefined)).toBeNull()
    expect(vehicleWeaponMountOf('')).toBeNull()
  })

  it('une arme de JOUEUR tirée depuis un véhicule (passager) rend null : pas de montage', () => {
    // Les deux armes vues dans l'artefact `0d76e8f1` sur des tirs `v` : Disrupteur et
    // CQS48 Bulldog — un passager qui tire SA PROPRE arme, sans siège mesuré à lui affirmer.
    expect(vehicleWeaponMountOf('0x84BD29ED42C9679F')).toBeNull()
    expect(vehicleWeaponMountOf('0xB619D84A42C9679F')).toBeNull()
  })

  it('le Warthog (témoin, vérifié en direct) est TOURELLE, ancre du plateau arrière', () => {
    const m = vehicleWeaponMountOf(shotW('c7d50912'))
    expect(m).not.toBeNull()
    expect(m?.classe).toBe('tourelle')
    expect(m?.ax).toBeCloseTo(0, 10)
    expect(m?.ay).toBeCloseTo(0.26, 10)
  })

  it('le Wasp M1 (vérifié en direct) est fixe : le pilote est le viseur', () => {
    expect(vehicleWeaponMountOf(shotW('11725dc4'))?.classe).toBe('fixe')
  })

  it('Ghost / Banshee (×2) / Chopper : montage fixe (documentés par V3F, non observés ici)', () => {
    for (const weap of ['00015435', '0000aa68', '0000aa69', 'b40e9618']) {
      expect(vehicleWeaponMountOf(shotW(weap))?.classe).toBe('fixe')
    }
  })

  it('le Scorpion est tourelle : le plateau tourne indépendamment des chenilles', () => {
    expect(vehicleWeaponMountOf(shotW('00015cfa'))?.classe).toBe('tourelle')
  })

  it('le Shade reste sans montage : c est son TAG qui manque, pas sa position', () => {
    // Zéro occurrence de sa famille dans V3F_TIRS_COVENANT — pas de tag à indexer, pas d'entrée
    // fabriquée. Les tags jpt de labels.tsv qu'une version antérieure de ce fichier utilisait par
    // erreur (mauvais espace d'identifiants) ne sont plus dans la table.
    expect(vehicleWeaponMountOf('099377af')).toBeNull() // Shade (jpt, PAS un tag weap de Shot.w).
  })

  /**
   * LOT 5.8.3 — LES TROIS MONTAGES MESURÉS SUR LE SPRITE. Leur TAG était connu depuis le lot des
   * sons (2026-09-04) ; c'est leur POSITION qui manquait, et l'éclair partait donc du centre du
   * châssis — pour le Wraith, un mètre et demi derrière la bouche de son mortier.
   */
  it('Wraith, Gungoose, Falcon : montage présent, et la CLASSE suit FAMILLES_ARME_FIXE (5.2a.6)', () => {
    const wraith = vehicleWeaponMountOf(shotW('121b4009'))
    const gungoose = vehicleWeaponMountOf(shotW('0042678e'))
    const falcon = vehicleWeaponMountOf(shotW('00015cd3'))
    // Le Wraith et le Gungoose sont des châssis à arme FIXE (viser, c'est tourner le véhicule) ;
    // la mitrailleuse de PORTE du Falcon est servie par un passager, donc une tourelle.
    expect(wraith?.classe).toBe('fixe')
    expect(gungoose?.classe).toBe('fixe')
    expect(falcon?.classe).toBe('tourelle')
  })

  it('les trois ancres neuves sont AVANT ou ARRIÈRE selon ce que le sprite montre', () => {
    // Le mortier du Wraith et les canons du Gungoose sont à l'AVANT (ay < 0, le nez est en haut) ;
    // les postes latéraux du Falcon sont EN ARRIÈRE du poste de pilotage (ay > 0).
    expect(vehicleWeaponMountOf(shotW('121b4009'))?.ay).toBeLessThan(0)
    expect(vehicleWeaponMountOf(shotW('0042678e'))?.ay).toBeLessThan(0)
    expect(vehicleWeaponMountOf(shotW('00015cd3'))?.ay).toBeGreaterThan(0)
    // Le mortier du Wraith est SUR L'AXE : son fût est centré à un demi-pixel de l'axe du châssis.
    expect(vehicleWeaponMountOf(shotW('121b4009'))?.ax).toBe(0)
  })

  it('toutes les ancres de la table tiennent dans [-0,5 ; +0,5]', () => {
    const weaps = ['c7d50912', '00015435', '0000aa68', '0000aa69', '11725dc4', 'd3c407ed',
      'b40e9618', '00015cfa', '121b4009', '0042678e', '00015cd3']
    for (const weap of weaps) {
      const m = vehicleWeaponMountOf(shotW(weap))
      expect(m).not.toBeNull()
      expect(Math.abs(m!.ax)).toBeLessThanOrEqual(0.5)
      expect(Math.abs(m!.ay)).toBeLessThanOrEqual(0.5)
    }
  })
})

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
  const CAPS = (chassisDeg: number) => ({ chassisDeg, tireurDeg: null })
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
    const p = vehicleShotPlacement(turret, { chassisDeg: 10, tireurDeg: 200 }, size, 1, ECHELLE)
    expect(p.angle).toBeCloseTo((-200 * Math.PI) / 180, 10)
  })

  /**
   * LE MONTAGE, LUI, SUIT LE CHÂSSIS : l'ancre est un point du SPRITE, donc elle tourne avec
   * l'image. Un décalage calculé sur la visée du tireur sortirait l'éclair du châssis.
   */
  it('classe tourelle : le DÉCALAGE suit le châssis, jamais la visée du tireur', () => {
    const turret: VehicleWeaponMount = { classe: 'tourelle', ax: 0, ay: -0.5 }
    const sansVisee = vehicleShotPlacement(turret, CAPS(90), size, 1, ECHELLE)
    const avecVisee = vehicleShotPlacement(turret, { chassisDeg: 90, tireurDeg: 200 }, size, 1, ECHELLE)
    expect(avecVisee.offset).toEqual(sansVisee.offset)
  })

  /**
   * UNE ARME FIXE NE CHANGE PAS DE SOURCE, et le test le dit : sur ces familles le cap du
   * châssis EST déjà la visée du conducteur depuis 5.2a.6 (écart mesuré 0,0 degré), donc lui
   * substituer la visée du tireur n'apporterait rien et brouillerait la règle.
   */
  it('classe fixe : la visée du tireur NE prend PAS le pas sur le cap du châssis', () => {
    const p = vehicleShotPlacement(nose, { chassisDeg: 33, tireurDeg: 200 }, size, 1, ECHELLE)
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

  it('classe fixe : direction = vehicleAimAngle(cap), jamais null', () => {
    const p = vehicleShotPlacement(nose, CAPS(33), size, 1, ECHELLE)
    expect(p.angle).not.toBeNull()
    expect(p.angle).toBeCloseTo((-33 * Math.PI) / 180, 10)
  })
})
