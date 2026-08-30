/**
 * Tests — changeRefine : LA DATATION FINE, ET CE QU'ELLE REFUSE DE FAIRE.
 *
 * CE QUE CE FICHIER VERROUILLE :
 *  - une substitution d'arme survenue APRÈS le relevé bascule la rangée À SA FRAME, et rajeunit
 *    l'âge de la lecture — c'est tout l'objet du lot ;
 *  - ce qui est DÉJÀ dans le relevé n'est pas rejoué, et ce qui est À VENIR n'est jamais lu ;
 *  - la rangée garde sa LONGUEUR et l'ORDRE de ses emplacements : lâchers et prises sur
 *    emplacement vide sont volontairement inappliqués (cf. l'en-tête du module) ;
 *  - une lecture À VENIR (âge négatif) n'est jamais raffinée ;
 *  - côté capacité, la plus RÉCENTE des deux sources gagne, et une consommation rend `null` —
 *    le joueur ne porte plus rien, et la fiche doit cesser de montrer l'équipement dépensé.
 */
import { describe, expect, it } from 'vitest'

import {
  REPLAY_NO_ABILITY_RANK,
  type ReplayEquipmentChange,
  type ReplayWeaponChange,
} from '@/lib/api/types'

import { ABILITY_SRC_CHANGE, refineAbilityReading, refineWeaponsReading } from './changeRefine'

const BR75 = '2b1824d5'
const SNIPER = '0a1992bc'
const SPNKR = '5eb0f0d1'

function chg(over: Partial<ReplayWeaponChange> = {}): ReplayWeaponChange {
  return { t: 50, slot: 1, kind: 'swapped', w: SNIPER, from: BR75, ...over }
}

describe('refineWeaponsReading — la bascule à la frame de l’événement', () => {
  it('substitue l’arme quand le changement suit le relevé, et rajeunit la lecture', () => {
    // Relevé à l'image 40 (âge 20 sur l'image 60), échange à l'image 50.
    const out = refineWeaponsReading({ weapons: [BR75, SPNKR], age: 20 }, [chg({ t: 50 })], 1, 60)
    expect(out.weapons).toEqual([SNIPER, SPNKR])
    expect(out.age).toBe(10)
  })

  it('ne rejoue PAS un changement déjà compris dans le relevé', () => {
    // Relevé à l'image 40 ; l'échange date de l'image 30, il est donc DANS la lecture.
    const base = { weapons: [SNIPER, SPNKR], age: 20 }
    expect(refineWeaponsReading(base, [chg({ t: 30 })], 1, 60)).toBe(base)
  })

  it('ne lit JAMAIS un changement à venir — le rejeu connaît la suite, la fiche non', () => {
    const base = { weapons: [BR75, SPNKR], age: 20 }
    expect(refineWeaponsReading(base, [chg({ t: 80 })], 1, 60)).toBe(base)
  })

  it('ignore les changements d’un AUTRE slot', () => {
    const base = { weapons: [BR75, SPNKR], age: 20 }
    expect(refineWeaponsReading(base, [chg({ t: 50, slot: 7 })], 1, 60)).toBe(base)
  })

  it('enchaîne deux substitutions DANS L’ORDRE, même servies à l’envers', () => {
    const out = refineWeaponsReading(
      { weapons: [BR75, SPNKR], age: 40 },
      [chg({ t: 55, from: SNIPER, w: SPNKR }), chg({ t: 45, from: BR75, w: SNIPER })],
      1,
      60,
    )
    expect(out.weapons).toEqual([SPNKR, SPNKR])
    expect(out.age).toBe(5)
  })
})

describe('refineWeaponsReading — ce qu’elle refuse d’appliquer', () => {
  it('n’applique pas un LÂCHER : la longueur de la rangée ne bouge pas', () => {
    // Retirer une entrée décalerait les indices que le sélecteur d'emplacement dégainé adresse.
    const base = { weapons: [BR75, SPNKR], age: 20 }
    const out = refineWeaponsReading(base, [chg({ t: 50, kind: 'dropped', w: '', from: BR75 })], 1, 60)
    expect(out).toBe(base)
  })

  it('n’applique pas une PRISE sur emplacement vide : rien n’est ajouté', () => {
    const base = { weapons: [BR75], age: 20 }
    const out = refineWeaponsReading(base, [chg({ t: 50, kind: 'taken', w: SNIPER, from: '' })], 1, 60)
    expect(out).toBe(base)
  })

  it('s’abstient quand la rangée lue ne NOMME PAS l’arme remplacée', () => {
    // Lectures désappariées : le relevé ne portait pas cette arme. On ne devine pas laquelle
    // des deux emplacements changer.
    const base = { weapons: [SPNKR], age: 20 }
    expect(refineWeaponsReading(base, [chg({ t: 50 })], 1, 60)).toBe(base)
  })

  it('ne raffine PAS une lecture À VENIR (âge négatif)', () => {
    const base = { weapons: [BR75, SPNKR], age: -30 }
    expect(refineWeaponsReading(base, [chg({ t: 50 })], 1, 60)).toBe(base)
  })

  it('rend la lecture telle quelle sur un artefact sans changements', () => {
    const base = { weapons: [BR75], age: 5 }
    expect(refineWeaponsReading(base, [], 1, 60)).toBe(base)
  })
})

function equip(over: Partial<ReplayEquipmentChange> = {}): ReplayEquipmentChange {
  return { t: 50, slot: 1, kind: 'taken', r: 20, from: REPLAY_NO_ABILITY_RANK, ...over }
}

describe('refineAbilityReading — la plus récente des deux sources gagne', () => {
  it('le CHANGEMENT l’emporte quand il est plus récent que le relevé', () => {
    const out = refineAbilityReading(
      { rank: 11, age: 30, src: 'kf' },
      [equip({ t: 50, r: 20 })],
      1,
      60,
    )
    expect(out).toEqual({ rank: 20, age: 10, src: ABILITY_SRC_CHANGE })
  })

  it('le RELEVÉ l’emporte quand il est plus récent — il a déjà vu l’effet', () => {
    const base = { rank: 20, age: 5, src: 'delta' }
    expect(refineAbilityReading(base, [equip({ t: 50 })], 1, 60)).toBe(base)
  })

  it('le RELEVÉ l’emporte à ÉGALITÉ d’âge', () => {
    const base = { rank: 20, age: 10, src: 'kf' }
    expect(refineAbilityReading(base, [equip({ t: 50 })], 1, 60)).toBe(base)
  })

  it('un événement PASSÉ prime une lecture à venir', () => {
    const out = refineAbilityReading({ rank: 11, age: -8, src: 'kf' }, [equip({ t: 50 })], 1, 60)
    expect(out?.rank).toBe(20)
  })

  it('sert la capacité même sans aucun relevé', () => {
    expect(refineAbilityReading(null, [equip({ t: 50 })], 1, 60)?.rank).toBe(20)
  })
})

describe('refineAbilityReading — la consommation est une MESURE', () => {
  it('rend null après un `spent` : le joueur ne porte plus rien', () => {
    const consomme = equip({ t: 50, kind: 'spent', r: REPLAY_NO_ABILITY_RANK, from: 20 })
    expect(refineAbilityReading({ rank: 20, age: 30, src: 'kf' }, [consomme], 1, 60)).toBeNull()
  })

  it('mais laisse le relevé PLUS RÉCENT reprendre la main', () => {
    const consomme = equip({ t: 50, kind: 'spent', r: REPLAY_NO_ABILITY_RANK, from: 20 })
    const base = { rank: 21, age: 2, src: 'delta' }
    expect(refineAbilityReading(base, [consomme], 1, 60)).toBe(base)
  })

  it('ignore les changements à venir et ceux d’un autre slot', () => {
    const base = { rank: 11, age: 30, src: 'kf' }
    expect(refineAbilityReading(base, [equip({ t: 80 })], 1, 60)).toBe(base)
    expect(refineAbilityReading(base, [equip({ t: 50, slot: 4 })], 1, 60)).toBe(base)
  })

  it('rend le relevé tel quel sur un artefact sans changements', () => {
    const base = { rank: 11, age: 30, src: 'kf' }
    expect(refineAbilityReading(base, [], 1, 60)).toBe(base)
  })
})
