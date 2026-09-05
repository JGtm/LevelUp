/**
 * Tests — abilityImpulseSound (le son de l'usage d'une capacité qui pousse son porteur).
 *
 * Ce que ces tests verrouillent :
 *  - UN SON PAR IMPULSION PUBLIÉE, à SA frame, convertie en ms par l'horloge du document :
 *    le calque date, on joue — rien d'autre ;
 *  - LA TABLE EST PAR FAMILLE : une famille sans stem reste MUETTE, jamais le son d'une
 *    voisine (le répulseur n'est pas dans ce canal et n'y sera pas) ;
 *  - LES VARIANTES SONT ATTACHÉES : le jeu tire un fichier parmi trois à chaque dash, donc
 *    l'événement porte les trois — un événement « nu » rejouerait toujours le même ;
 *  - LA PISTE ENTIÈRE LE PORTE, sous la catégorie ÉQUIPEMENT : couper cette catégorie coupe
 *    le dash comme elle coupe le grappin.
 */
import { describe, expect, it } from 'vitest'

import {
  ABILITY_IMPULSE_SOUND_STEMS,
  abilityImpulseSoundEvents,
} from './abilityImpulseSound'
import { buildSoundTimeline } from './replaySound'
import { SOUND_VARIANTS } from './replaySoundVariants'
import { testReplayDoc } from './test/testDoc'

const CATEGORIES = {
  weapon: false,
  grenade: false,
  melee: false,
  equipment: true,
  objective: false,
}

function docWith(impulses: { t: number; slot: number; family: string }[]) {
  return testReplayDoc({ frameIntervalMs: 100, abilityImpulses: impulses })
}

describe('abilityImpulseSoundEvents', () => {
  it('un son par impulsion publiée, à sa frame convertie par l’horloge du document', () => {
    const events = abilityImpulseSoundEvents(
      docWith([
        { t: 10, slot: 5, family: 'thruster' },
        { t: 25, slot: 5, family: 'thruster' },
      ]),
    )
    expect(events.map((e) => e.ms)).toEqual([1_000, 2_500])
    expect(events.every((e) => e.stem === 'thruster_activate')).toBe(true)
  })

  it('une famille sans stem reste MUETTE — jamais le son d’une voisine', () => {
    const events = abilityImpulseSoundEvents(
      docWith([
        { t: 10, slot: 5, family: 'repulsor' },
        { t: 20, slot: 5, family: 'grapple' },
      ]),
    )
    expect(events).toEqual([])
  })

  it('l’événement porte les TROIS variantes : le jeu en tire une à chaque dash', () => {
    const [ev] = abilityImpulseSoundEvents(docWith([{ t: 10, slot: 5, family: 'thruster' }]))
    expect(ev.variants).toEqual(SOUND_VARIANTS.thruster_activate)
    expect(ev.variants).toHaveLength(3)
  })

  it('le stem du propulseur est celui que le manifeste de variantes déclare en tête', () => {
    expect(ABILITY_IMPULSE_SOUND_STEMS.thruster).toBe(SOUND_VARIANTS.thruster_activate?.[0])
  })

  it('sans impulsion, aucun son : le document normalisé garantit un tableau', () => {
    expect(abilityImpulseSoundEvents(docWith([]))).toEqual([])
  })
})

describe('buildSoundTimeline — le dash entre par la catégorie ÉQUIPEMENT', () => {
  const doc = docWith([{ t: 10, slot: 5, family: 'thruster' }])

  it('la piste porte le son quand la catégorie est allumée', () => {
    const piste = buildSoundTimeline(doc, [], CATEGORIES)
    expect(piste.filter((e) => e.stem === 'thruster_activate')).toHaveLength(1)
  })

  it('couper « Équipements » coupe le dash, comme elle coupe le grappin', () => {
    const piste = buildSoundTimeline(doc, [], { ...CATEGORIES, equipment: false })
    expect(piste.filter((e) => e.stem === 'thruster_activate')).toEqual([])
  })
})
