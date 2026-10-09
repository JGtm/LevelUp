/**
 * Tests — groundWeaponTime : LES TROIS BORNES, ET CE QU'ON N'AFFIRME PAS ENTRE ELLES.
 *
 * CE QUE CE FICHIER VERROUILLE, clause par clause du schéma 27 :
 *  - avant `t0`, RIEN : l'objet n'existe pas encore ;
 *  - de `t0` à `t1`, PLEIN : une preuve de présence tient ;
 *  - de `t1` à `t1max` (fins `seen` seulement), une DESCENTE : plus rien ne prouve, rien ne
 *    réfute encore. Elle ne saute pas au plancher et n'y arrive qu'à la dernière image ;
 *  - après `t1max`, RIEN : la borne est une preuve d'ABSENCE, on ne la franchit pas ;
 *  - sur `pickup` et `open` (`t1max == t1`), il n'y a PAS de descente — la fin est exacte.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayGroundWeapon } from '@/lib/api/types'

import {
  GROUND_WEAPON_ALPHA_FADED,
  GROUND_WEAPON_ALPHA_FULL,
  groundWeaponPresenceAt,
  groundWeaponsAt,
} from './groundWeaponTime'

/** Une arme au sol dont la disparition est un INTERVALLE de 10 images (fin `seen`). */
function vue(over: Partial<ReplayGroundWeapon> = {}): ReplayGroundWeapon {
  return {
    t0: 10,
    t1: 40,
    t1max: 50,
    x: 5,
    y: 5,
    w: '0a1992bc',
    origin: 'dropped',
    dropper: 3,
    end: 'seen',
    picker: -1,
    ...over,
  }
}

/** Une arme RAMASSÉE : fin exacte, donc `t1max == t1` (cf. le contrat du schéma 27). */
const RAMASSEE = vue({ t1: 40, t1max: 40, end: 'pickup', picker: 5 })

describe('groundWeaponPresenceAt — les trois bornes', () => {
  it('ne dessine RIEN avant l’apparition', () => {
    expect(groundWeaponPresenceAt(vue(), 9)).toBeNull()
  })

  it('dessine PLEIN de t0 à t1 inclus — une preuve de présence tient', () => {
    for (const frame of [10, 25, 40]) {
      expect(groundWeaponPresenceAt(vue(), frame), `image ${frame}`).toEqual({
        alpha: GROUND_WEAPON_ALPHA_FULL,
        vanishing: false,
      })
    }
  })

  it('ESTOMPE entre t1 et t1max, sans jamais sauter au plancher', () => {
    const mi = groundWeaponPresenceAt(vue(), 45)
    expect(mi?.vanishing).toBe(true)
    // À mi-intervalle, l'opacité est à mi-chemin : la descente est LINÉAIRE parce que le film
    // ne dit rien de la façon dont l'objet a disparu (cf. l'en-tête du module).
    expect(mi?.alpha).toBeCloseTo((GROUND_WEAPON_ALPHA_FULL + GROUND_WEAPON_ALPHA_FADED) / 2, 6)
    const tot = groundWeaponPresenceAt(vue(), 41)
    expect(tot?.alpha).toBeGreaterThan(mi!.alpha)
    expect(tot?.alpha).toBeLessThan(GROUND_WEAPON_ALPHA_FULL)
  })

  it('atteint le plancher À la première preuve d’absence, et pas avant', () => {
    const bout = groundWeaponPresenceAt(vue(), 50)
    expect(bout?.vanishing).toBe(true)
    expect(bout?.alpha).toBeCloseTo(GROUND_WEAPON_ALPHA_FADED, 6)
    // NON NUL à `t1max` : à zéro, l'objet s'éteindrait avant la borne mesurée et la borne
    // haute ne se lirait plus à l'écran.
    expect(GROUND_WEAPON_ALPHA_FADED).toBeGreaterThan(0)
  })

  it('ne dessine RIEN après la première preuve d’absence', () => {
    expect(groundWeaponPresenceAt(vue(), 51)).toBeNull()
    expect(groundWeaponPresenceAt(vue(), 9999)).toBeNull()
  })
})

describe('groundWeaponPresenceAt — les fins EXACTES n’ont pas de réserve à porter', () => {
  it('un RAMASSAGE passe du plein au néant : aucune image estompée', () => {
    expect(groundWeaponPresenceAt(RAMASSEE, 40)).toEqual({
      alpha: GROUND_WEAPON_ALPHA_FULL,
      vanishing: false,
    })
    expect(groundWeaponPresenceAt(RAMASSEE, 41)).toBeNull()
  })

  it('une fin OUVERTE tient plein jusqu’à sa dernière image', () => {
    const ouverte = vue({ t1: 199, t1max: 199, end: 'open' })
    expect(groundWeaponPresenceAt(ouverte, 199)?.vanishing).toBe(false)
    expect(groundWeaponPresenceAt(ouverte, 200)).toBeNull()
  })

  it('un objet qui n’a jamais été revu (t1 == t0) reste affiché SON image', () => {
    // Le document publie `t1 == t0` quand la vie de l'objet est plus courte qu'un intervalle
    // d'image-clé : sans ce cas, une arme lâchée puis reprise aussitôt n'aurait aucune image.
    const bref = vue({ t0: 30, t1: 30, t1max: 30, end: 'pickup', picker: 2 })
    expect(groundWeaponPresenceAt(bref, 30)?.alpha).toBe(GROUND_WEAPON_ALPHA_FULL)
    expect(groundWeaponPresenceAt(bref, 31)).toBeNull()
  })
})

describe('groundWeaponsAt — la sélection d’une image', () => {
  it('ne rend que les armes visibles, chacune avec SA présence', () => {
    const items = [
      vue({ t0: 0, t1: 10, t1max: 10, end: 'pickup', w: 'aaaaaaaa' }),
      vue({ t0: 0, t1: 100, t1max: 100, end: 'open', w: 'bbbbbbbb' }),
      vue({ t0: 90, t1: 120, t1max: 120, end: 'open', w: 'cccccccc' }),
    ]
    const a20 = groundWeaponsAt(items, 20)
    expect(a20.map((v) => v.item.w)).toEqual(['bbbbbbbb'])
    const a95 = groundWeaponsAt(items, 95)
    expect(a95.map((v) => v.item.w)).toEqual(['bbbbbbbb', 'cccccccc'])
  })

  it('rend une liste VIDE quand rien n’est au sol — jamais null', () => {
    expect(groundWeaponsAt([], 0)).toEqual([])
    expect(groundWeaponsAt([vue()], 0)).toEqual([])
  })
})
