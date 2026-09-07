import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { equipmentChangeSoundEvents } from './equipmentChangeSound'
import { WEAPON_CHANGE_SOUND_STEMS, weaponChangeSoundEvents } from './weaponChangeSound'

/**
 * weaponChangeSound.test.ts — les gestes datés du chantier ramassage (schémas 25-26).
 *
 * La règle est courte parce que la source l'est : le calque PUBLIE les instants, il n'y a rien
 * à déduire. Ce qui se fige ici, ce sont les CHOIX — un échange sonne le ramassage seul, une
 * consommation d'équipement ne sonne rien — et les bords vides.
 */

/** Un document minimal : le pas de grille et les canaux de changements suffisent. */
function doc(
  weaponChanges: unknown[],
  equipmentChanges: unknown[] = [],
  pickups: unknown[] = [],
): ReplayDocumentReady {
  return {
    frameIntervalMs: 100,
    weaponChanges,
    equipmentChanges,
    pickups,
  } as unknown as ReplayDocumentReady
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

describe('le canal NATIF comble les trous, et ne double JAMAIS', () => {
  it('un ramassage natif déjà vu par weaponChanges ne sonne pas une seconde fois', () => {
    const d = doc(
      [{ t: 10, slot: 512, kind: 'taken', w: '1111aaaa' }],
      [],
      [{ t: 12, slot: 512, kind: 'weapon', w: '1111aaaa', class: 0 }],
    )
    expect(weaponChangeSoundEvents(d)).toEqual([
      { ms: 1000, stem: WEAPON_CHANGE_SOUND_STEMS.taken },
    ])
  })

  it('un ramassage natif que weaponChanges RATE sonne — c est le trou de rappel comblé', () => {
    const d = doc([], [], [{ t: 12, slot: 512, kind: 'weapon', w: '1111aaaa', class: 0 }])
    expect(weaponChangeSoundEvents(d)).toEqual([
      { ms: 1200, stem: WEAPON_CHANGE_SOUND_STEMS.taken },
    ])
  })

  it('hors fenêtre d appariement, c est un AUTRE geste : il sonne', () => {
    const d = doc(
      [{ t: 10, slot: 512, kind: 'taken', w: '1111aaaa' }],
      [],
      [{ t: 40, slot: 512, kind: 'weapon', w: '1111aaaa', class: 0 }],
    )
    expect(weaponChangeSoundEvents(d)).toEqual([
      { ms: 1000, stem: WEAPON_CHANGE_SOUND_STEMS.taken },
      { ms: 4000, stem: WEAPON_CHANGE_SOUND_STEMS.taken },
    ])
  })

  it('une AUTRE vie au même instant n est pas le même geste : elle sonne', () => {
    const d = doc(
      [{ t: 10, slot: 512, kind: 'taken', w: '1111aaaa' }],
      [],
      [{ t: 10, slot: 999, kind: 'weapon', w: '1111aaaa', class: 0 }],
    )
    expect(weaponChangeSoundEvents(d)).toHaveLength(2)
  })

  it('la FAMILLE compte : une AUTRE arme prise dans la fenêtre sonne quand même', () => {
    // Le joueur prend l arme A (vue par weaponChanges) puis l arme B moins de 500 ms plus tard
    // (vue du seul canal natif). Sans la clé de famille dans la déduplication, B serait
    // avalée par A et le geste deviendrait MUET.
    const d = doc(
      [{ t: 10, slot: 512, kind: 'taken', w: '1111aaaa' }],
      [],
      [{ t: 12, slot: 512, kind: 'weapon', w: '2222bbbb', class: 0 }],
    )
    expect(weaponChangeSoundEvents(d)).toEqual([
      { ms: 1000, stem: WEAPON_CHANGE_SOUND_STEMS.taken },
      { ms: 1200, stem: WEAPON_CHANGE_SOUND_STEMS.taken },
    ])
  })

  it('la BORNE de la fenêtre est nette : 5 frames = même geste, 6 = un autre', () => {
    const base = { slot: 512, kind: 'taken', w: '1111aaaa' }
    const natif = (t: number) => [{ t, slot: 512, kind: 'weapon', w: '1111aaaa', class: 0 }]
    // Δ = 5 frames (500 ms) : c'est le MÊME ramassage, vu deux fois.
    expect(weaponChangeSoundEvents(doc([{ ...base, t: 10 }], [], natif(15)))).toHaveLength(1)
    // Δ = 6 frames : au-delà de la tolérance mesurée, ce sont deux gestes.
    expect(weaponChangeSoundEvents(doc([{ ...base, t: 10 }], [], natif(16)))).toHaveLength(2)
  })

  it('un ramassage NON-ARME ne sonne pas l arme — son bruit est celui de l équipement', () => {
    const d = doc([], [], [{ t: 12, slot: 512, kind: 'item', w: 'deadbeef', class: 2 }])
    expect(weaponChangeSoundEvents(d)).toEqual([])
  })

  it('un artefact ANTÉRIEUR au schéma 30 n a pas de `pickups` : on se tait, on ne lève pas', () => {
    const d = { frameIntervalMs: 100, weaponChanges: [], equipmentChanges: [] } as unknown as ReplayDocumentReady
    expect(weaponChangeSoundEvents(d)).toEqual([])
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
