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

  it('Gungoose, Shade, tourelle LMG : aucun tag weap documenté, donc repli centre', () => {
    // Zéro occurrence de leur famille dans V3F_TIRS_COVENANT — pas de tag à indexer, pas
    // d'entrée fabriquée. Les tags jpt de labels.tsv qu'une version antérieure de ce fichier
    // utilisait par erreur (mauvais espace d'identifiants) ne sont plus dans la table.
    expect(vehicleWeaponMountOf('099377af')).toBeNull() // Shade (jpt, PAS un tag weap de Shot.w).
  })

  it('toutes les ancres de la table tiennent dans [-0,5 ; +0,5]', () => {
    const weaps = ['c7d50912', '00015435', '0000aa68', '0000aa69', '11725dc4', 'd3c407ed', 'b40e9618', '00015cfa']
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
  const SCALE = vehicleSpriteScale(size.naturalHeightPx, size.mmPerPx)
  // Ancre nez pur (ax=0, ay=-0,5) : au bord haut du sprite AVANT rotation, comme
  // `drawRotatedSprite` dessine `drawImage(img, -w/2, -h/2, w, h)` (ay=-0,5 -> y local = -h/2).
  const nose: VehicleWeaponMount = { classe: 'fixe', ax: 0, ay: -0.5 }
  // Ancre latérale pure (ax=+0,5, ay=0) : au bord droit du sprite avant rotation.
  const rightSide: VehicleWeaponMount = { classe: 'fixe', ax: 0.5, ay: 0 }

  it('cap 90° (vehicleScreenAngle = 0) : repère local = repère écran, sans rotation', () => {
    const p = vehicleShotPlacement(nose, 90, size, 1)
    // localY = -0,5 * 200 = -100 ; screenAngle(90) = 0 -> offset = (0, -100) * SCALE.
    expect(p.offset.x).toBeCloseTo(0, 6)
    expect(p.offset.y).toBeCloseTo(-100 * SCALE, 6)
  })

  it('cap 0° (monde +X = droite écran) : le nez pointe vers +X', () => {
    const p = vehicleShotPlacement(nose, 0, size, 1)
    expect(p.offset.x).toBeCloseTo(100 * SCALE, 6)
    expect(p.offset.y).toBeCloseTo(0, 6)
  })

  it('cap 180° (monde -X = gauche écran) : le nez pointe vers -X', () => {
    const p = vehicleShotPlacement(nose, 180, size, 1)
    expect(p.offset.x).toBeCloseTo(-100 * SCALE, 6)
    expect(p.offset.y).toBeCloseTo(0, 6)
  })

  it('cap 270° (monde -Y = bas écran) : le nez pointe vers +Y écran (bas)', () => {
    const p = vehicleShotPlacement(nose, 270, size, 1)
    expect(p.offset.x).toBeCloseTo(0, 6)
    expect(p.offset.y).toBeCloseTo(100 * SCALE, 6)
  })

  it('une ancre latérale tourne comme une ancre longitudinale (même transform)', () => {
    const p = vehicleShotPlacement(rightSide, 0, size, 1)
    // localX = 0,5 * 100 = 50 ; screenAngle(0) = 90° -> (localX*cos90 - localY*sin90, localX*sin90+...)
    // = (0 - 0, 50 + 0) = (0, 50) * SCALE.
    expect(p.offset.x).toBeCloseTo(0, 6)
    expect(p.offset.y).toBeCloseTo(50 * SCALE, 6)
  })

  it('la densité k met le décalage à l’échelle (aucune rotation supplémentaire)', () => {
    const p1 = vehicleShotPlacement(nose, 90, size, 1)
    const p2 = vehicleShotPlacement(nose, 90, size, 2)
    expect(p2.offset.y).toBeCloseTo(p1.offset.y * 2, 6)
  })

  it('classe tourelle : direction TOUJOURS null, quel que soit le cap', () => {
    const turret: VehicleWeaponMount = { classe: 'tourelle', ax: 0, ay: 0 }
    expect(vehicleShotPlacement(turret, 45, size, 1).angle).toBeNull()
    expect(vehicleShotPlacement(turret, 270, size, 1).angle).toBeNull()
  })

  it('classe fixe : direction = vehicleAimAngle(cap), jamais null', () => {
    const p = vehicleShotPlacement(nose, 33, size, 1)
    expect(p.angle).not.toBeNull()
    expect(p.angle).toBeCloseTo((-33 * Math.PI) / 180, 10)
  })
})
