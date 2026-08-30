import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady } from './replayNormalize'
import { equipmentChangeSoundEvents } from './equipmentChangeSound'
import { WEAPON_CHANGE_SOUND_STEMS, weaponChangeSoundEvents } from './weaponChangeSound'

/**
 * weaponChangeSound.test.ts — les gestes datés du chantier ramassage (schémas 25-26).
 *
 * La règle est courte parce que la source l'est : le calque PUBLIE les instants, il n'y a rien
 * à déduire. Ce qui se fige ici, ce sont les CHOIX — un échange sonne le ramassage seul, une
 * consommation d'équipement ne sonne rien — et les bords vides.
 */

/** Un document minimal : le pas de grille et les deux canaux de changements suffisent. */
function doc(weaponChanges: unknown[], equipmentChanges: unknown[] = []): ReplayDocumentReady {
  return { frameIntervalMs: 100, weaponChanges, equipmentChanges } as unknown as ReplayDocumentReady
}

describe('ramassage et lâcher d arme — le calque date, on joue', () => {
  it('une prise sonne le ramassage, un lâcher le lâcher, chacun à sa frame', () => {
    const d = doc([
      { t: 10, slot: 512, kind: 'taken', w: '0x1111aaaa' },
      { t: 40, slot: 512, kind: 'dropped', from: '0x1111aaaa' },
    ])
    expect(weaponChangeSoundEvents(d)).toEqual([
      { ms: 1000, stem: WEAPON_CHANGE_SOUND_STEMS.taken },
      { ms: 4000, stem: WEAPON_CHANGE_SOUND_STEMS.dropped },
    ])
  })

  it('un ÉCHANGE sonne le ramassage seul — pas les deux fichiers superposés', () => {
    const d = doc([{ t: 20, slot: 512, kind: 'swapped', w: '0x2222bbbb', from: '0x1111aaaa' }])
    expect(weaponChangeSoundEvents(d)).toEqual([
      { ms: 2000, stem: WEAPON_CHANGE_SOUND_STEMS.taken },
    ])
  })

  it('MUET sans changement', () => {
    expect(weaponChangeSoundEvents(doc([]))).toEqual([])
  })
})

describe('ramassage d équipement — `taken` sonne, `spent` se tait', () => {
  it('un ramassage sonne à sa frame', () => {
    const d = doc([], [{ t: 30, slot: 512, kind: 'taken', r: 2, from: -1 }])
    expect(equipmentChangeSoundEvents(d)).toEqual([
      { ms: 3000, stem: 'objective_pad_pickup' },
    ])
  })

  it('une CONSOMMATION ne sonne pas : l usage sonne déjà par sa famille', () => {
    const d = doc([], [{ t: 30, slot: 512, kind: 'spent', r: -1, from: 2 }])
    expect(equipmentChangeSoundEvents(d)).toEqual([])
  })

  it('MUET sans changement', () => {
    expect(equipmentChangeSoundEvents(doc([]))).toEqual([])
  })
})
