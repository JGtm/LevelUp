import { describe, expect, it } from 'vitest'

import { equippedWeapons, loadoutSwapAt } from './equippedLogic'
import { testReplayDoc as doc } from './test/testDoc'

// Un slot, deux armes lues à l'image-clé t=10, et un inventaire dont seul le sélecteur
// d'emplacement varie d'un cas à l'autre : c'est lui qui est sous test.
function docWithSelector(d?: number) {
  return doc({
    loadouts: [{ t: 10, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
    inventory: [{ t: 10, slot: 512, ...(d === undefined ? {} : { d }) }],
  })
}

describe('equippedWeapons — le sélecteur d’emplacement ordonne la rangée', () => {
  it('D=0 : l’emplacement 0 est EN MAIN, à gauche ; l’autre suit', () => {
    expect(equippedWeapons(docWithSelector(0), 512, 60)).toEqual({
      weapons: [
        { id: '0xAAAA', inHand: true },
        { id: '0xBBBB', inHand: false },
      ],
      age: 50,
      holstered: false,
    })
  })

  it('D=1 : l’emplacement 1 passe devant, marqué en main', () => {
    expect(equippedWeapons(docWithSelector(1), 512, 60)).toEqual({
      weapons: [
        { id: '0xBBBB', inHand: true },
        { id: '0xAAAA', inHand: false },
      ],
      age: 50,
      holstered: false,
    })
  })

  it('D=2 : RIEN de dégainé — une mesure, pas une lacune : ordre d’emplacement, aucun marquage', () => {
    // C'est l'état des huit joueurs à la première image-clé, avant le départ.
    expect(equippedWeapons(docWithSelector(2), 512, 60)).toEqual({
      weapons: [
        { id: '0xAAAA', inHand: false },
        { id: '0xBBBB', inHand: false },
      ],
      age: 50,
      holstered: true,
    })
  })

  it('sélecteur non lu : ordre d’emplacement, aucun marquage, et ce n’est PAS « rangées »', () => {
    const r = equippedWeapons(docWithSelector(undefined), 512, 60)
    expect(r?.weapons.every((w) => !w.inHand)).toBe(true)
    expect(r?.holstered).toBe(false)
  })

  it('sélecteur désappariété (D pointe hors du loadout lu) : la mise en valeur s’abstient', () => {
    // Loadout à UNE arme, sélecteur disant « emplacement 1 » : les deux scans ne se
    // recouvrent pas — marquer une main serait afficher une certitude qu'on n'a pas.
    const d = doc({
      loadouts: [{ t: 10, slot: 512, w: ['0xAAAA'] }],
      inventory: [{ t: 10, slot: 512, d: 1 }],
    })
    expect(equippedWeapons(d, 512, 60)).toEqual({
      weapons: [{ id: '0xAAAA', inHand: false }],
      age: 50,
      holstered: false,
    })
  })

  it('le sélecteur d’un AUTRE slot ne fuit pas', () => {
    const d = doc({
      loadouts: [{ t: 10, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
      inventory: [{ t: 10, slot: 513, d: 1 }],
    })
    expect(equippedWeapons(d, 512, 60)?.weapons.every((w) => !w.inHand)).toBe(true)
  })

  it('sans loadout lu, rend null — la rangée affiche sa lacune, pas une liste vide', () => {
    expect(equippedWeapons(doc(), 512, 60)).toBeNull()
  })
})

describe('loadoutSwapAt — le diff des deux dernières lectures d’un slot', () => {
  it('date un échange : ce qui est entré, ce qui est sorti, l’âge de la lecture courante', () => {
    const d = doc({
      loadouts: [
        { t: 10, slot: 512, w: ['0xAAAA', '0xBBBB'] },
        { t: 30, slot: 512, w: ['0xAAAA', '0xCCCC'] },
      ],
    })
    expect(loadoutSwapAt(d, 512, 50)).toEqual({
      picked: ['0xCCCC'],
      dropped: ['0xBBBB'],
      age: 20,
    })
  })

  it('deux lectures identiques : aucun échange, null', () => {
    const d = doc({
      loadouts: [
        { t: 10, slot: 512, w: ['0xAAAA', '0xBBBB'] },
        { t: 30, slot: 512, w: ['0xAAAA', '0xBBBB'] },
      ],
    })
    expect(loadoutSwapAt(d, 512, 50)).toBeNull()
  })

  it('l’ordre des emplacements n’est PAS un échange', () => {
    // Un swap de MAIN réordonne la rangée via le sélecteur ; la dotation, elle, n'a pas changé.
    const d = doc({
      loadouts: [
        { t: 10, slot: 512, w: ['0xAAAA', '0xBBBB'] },
        { t: 30, slot: 512, w: ['0xBBBB', '0xAAAA'] },
      ],
    })
    expect(loadoutSwapAt(d, 512, 50)).toBeNull()
  })

  it('le diff est un MULTIENSEMBLE : perdre une copie d’une arme portée en double se voit', () => {
    const d = doc({
      loadouts: [
        { t: 10, slot: 512, w: ['0xAAAA', '0xAAAA'] },
        { t: 30, slot: 512, w: ['0xAAAA', '0xBBBB'] },
      ],
    })
    expect(loadoutSwapAt(d, 512, 50)).toEqual({
      picked: ['0xBBBB'],
      dropped: ['0xAAAA'],
      age: 20,
    })
  })

  it('sans DEUX lectures au plus tard à l’image, rien à comparer : null', () => {
    const d = doc({
      loadouts: [
        { t: 10, slot: 512, w: ['0xAAAA'] },
        { t: 90, slot: 512, w: ['0xBBBB'] }, // dans le futur de frame=50
        { t: 10, slot: 513, w: ['0xCCCC'] }, // un autre slot ne compte pas
      ],
    })
    expect(loadoutSwapAt(d, 512, 50)).toBeNull()
  })

  it('compare les deux DERNIÈRES lectures, pas la première venue', () => {
    const d = doc({
      loadouts: [
        { t: 5, slot: 512, w: ['0xDDDD'] },
        { t: 10, slot: 512, w: ['0xAAAA'] },
        { t: 30, slot: 512, w: ['0xAAAA'] },
      ],
    })
    // Entre t=10 et t=30, rien n'a changé : le vieux 0xDDDD de t=5 ne doit pas ressurgir.
    expect(loadoutSwapAt(d, 512, 50)).toBeNull()
  })
})
