/**
 * Garde-rail (lot 5.8.2) : TOUTE ARME DE VÉHICULE QUI SONNE A UN STYLE D'ÉCLAIR.
 *
 * CE QUI CASSE SANS CE TEST, ET SILENCIEUSEMENT. Une arme de véhicule entre par le SON (une
 * reconstruction Wwise validée, `sound/vehicleShotSound.ts`) et par le MONTAGE (un tag `weap`
 * retrouvé, `vehicleWeaponMounts.ts`). Rien n'oblige à lui donner un STYLE : elle sonne, elle
 * part du bon endroit, et son éclair reste un halo gris pâle — exactement le défaut que
 * l'utilisateur a signalé le 2026-09-19, et qui ne se voit qu'à l'écran.
 *
 * DEUX SOURCES DE VÉRITÉ, ET LE TEST LES JOINT DANS LE BON SENS : toute arme de la table SONORE
 * (10 entrées) doit avoir un style, et tout tag documenté par un MONTAGE aussi. La réciproque
 * n'est PAS exigée — un style peut exister sans son (les missiles du Wasp n'ont aucune
 * reconstruction et se voient pourtant) ni montage (l'éclair part alors du centre du châssis).
 *
 * LES VALEURS SONT CELLES DES DEUX LISTES FERMÉES DU DÉPÔT : un style qui nommerait une famille
 * ou une teinte hors liste tomberait silencieusement sur le neutre côté rendu (`familyOf` /
 * `fxTintOf` sont tolérants par conception). Le test le refuse ici, où ça se voit.
 */
import { describe, expect, it } from 'vitest'

import { fxTintOf } from '../layers/fxInk'
import { familyOf } from '../layers/shotEffects'
import { VEHICLE_SHOT_SOUND_STEMS } from '../sound/vehicleShotSound'
import { vehicleShotStyleOf } from './vehicleShotFx'
import { vehicleWeaponMountOf, vehicleWeapTag } from './vehicleWeaponMounts'

/**
 * LES TAGS QUI SONNENT. La table des tags « ambigus » du lot 5.8.4 a disparu le 2026-09-23 (le
 * tag du Rockethog n'était pas ambigu, lot L1.5) : une seule table, le même compte.
 */
const TAGS_QUI_SONNENT = [...VEHICLE_SHOT_SOUND_STEMS.keys()]

/**
 * Les tags de la table des MONTAGES, relus par leur seule porte publique : la table elle-même
 * n'est pas exportée (et n'a pas à l'être), mais `vehicleWeaponMountOf` répond sur chacun.
 */
const TAGS_MONTES = [
  'c7d50912', // Rockethog
  '00015435', // Ghost
  '0000aa68', // Banshee M1
  '0000aa69', // Banshee M2
  '11725dc4', // Wasp M1
  'd3c407ed', // Wasp M2
  'b40e9618', // Chopper
  '49e40d17', // Scorpion (tag publié)
  '0bb6976b', // Falcon — lance-grenades
].map(vehicleWeapTag)

describe('garde-rail : le style d’éclair des armes de véhicule', () => {
  it('les 10 armes qui SONNENT ont toutes un style', () => {
    expect(TAGS_QUI_SONNENT.length, 'les tables sonores ont changé de taille').toBe(10)
    for (const tag of TAGS_QUI_SONNENT) {
      expect(vehicleShotStyleOf(tag), `aucun style pour ${tag}`).not.toBeNull()
    }
  })

  it('toute arme dont le MONTAGE est documenté a aussi un style', () => {
    for (const tag of TAGS_MONTES) {
      expect(vehicleWeaponMountOf(tag), `montage perdu pour ${tag}`).not.toBeNull()
      expect(vehicleShotStyleOf(tag), `aucun style pour ${tag}`).not.toBeNull()
    }
  })

  it('aucun style ne nomme une famille ni une teinte hors des deux listes fermées', () => {
    for (const tag of TAGS_QUI_SONNENT) {
      const style = vehicleShotStyleOf(tag)
      expect(style, tag).not.toBeNull()
      if (!style) continue
      // `familyOf` / `fxTintOf` rendent le NEUTRE sur une valeur inconnue : l'égalité à
      // l'entrée est donc la preuve que la valeur est bien dans la liste.
      expect(familyOf(style.fx), `famille hors liste pour ${tag}`).toBe(style.fx)
      expect(fxTintOf(style.tint), `teinte hors liste pour ${tag}`).toBe(style.tint)
    }
  })

  it('aucun style n’est la MÊLÉE : un canon de véhicule a un éclair de bouche', () => {
    for (const tag of TAGS_QUI_SONNENT) {
      // `buildShotFx` ÉCARTE la famille `melee` : un style de mêlée rendrait le tir invisible.
      expect(vehicleShotStyleOf(tag)?.fx, tag).not.toBe('melee')
    }
  })

  it('un tag inconnu, ou aucun tag, ne prend JAMAIS le style d’une voisine', () => {
    expect(vehicleShotStyleOf(undefined)).toBeNull()
    expect(vehicleShotStyleOf(vehicleWeapTag('deadbeef'))).toBeNull()
    // Le Shade : aucun tag `weap` documenté — la table ne peut pas le porter (cf. en-tête).
    expect(vehicleShotStyleOf('0xSHADE')).toBeNull()
  })
})
