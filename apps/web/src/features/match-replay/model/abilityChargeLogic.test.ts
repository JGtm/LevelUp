/**
 * Tests — abilityChargeLogic (les charges de la capacité portée, lot P6).
 *
 * Ce que ces tests verrouillent, et chacun est une règle de P6.1 :
 *  - AVANT toute lecture : « plein » QUALITATIF, jamais un chiffre inventé (décision
 *    utilisateur du 04/09 — le film ne transmet rien au ramassage) ;
 *  - la VALEUR : la lecture la plus récente <= image courante ; une lecture À VENIR
 *    n'est jamais lue (le rejeu connaît la suite, la vignette n'y a pas droit) ;
 *  - la VIE : jointure par la vie qui COUVRE l'instant (fixture canonique de
 *    `fireMark.test.ts` — deux vies du même slot, [10,20] et [30,40]) : la lecture de la
 *    vie 1 ne colle pas à la vie 2, on réapparaît plein ;
 *  - le RE-RAMASSAGE : une lecture antérieure au dernier changement d'équipement de la vie
 *    décrit l'équipement précédent — le nouveau repart à « plein » ;
 *  - la FAMILLE : jamais mesurée = RIEN d'affirmé (ni chiffre ni « plein ») ; lecture la
 *    plus récente d'une AUTRE famille = RIEN non plus (le canal contredit la vignette,
 *    les charges d'un grappin ne collent pas à un propulseur).
 */
import { describe, expect, it } from 'vitest'

import { abilityChargesAt, hasAbilityChargeLayer, measuredChargeFamilyOf } from './abilityChargeLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { testReplayDoc } from '../test/testDoc'

/** La palette du document : grappin et propulseur mesurés, répulseur jamais (pas de canal). */
const LABELS = {
  '4': { fr: 'grappin', en: 'Grappleshot' },
  '5': { fr: 'propulseur', en: 'Thruster' },
  '6': { fr: 'répulseur', en: 'Repulsor' },
}

/** Deux VIES du même slot : la première couvre [10, 20], la seconde [30, 40]. */
const LIFE_1 = {
  slot: 5,
  team: -1,
  startFrame: 10,
  endFrame: 20,
  points: [{ t: 10, x: 0, y: 0 }],
}
const LIFE_2 = {
  slot: 5,
  team: -1,
  startFrame: 30,
  endFrame: 40,
  points: [{ t: 30, x: 0, y: 0 }],
}

/**
 * La couverture d'un balayage qui a TOURNÉ : c'est elle qui atteste que le calque existe
 * (schéma 38) — sans elle, « plein » serait affirmé sur un artefact où le canal n'a jamais
 * été lu (constat P0 de la revue P6). Le type du contrat exige d'autres blocs obligatoires
 * qui ne jouent pas ici — d'où le cast, même patron que `calqueLu` de
 * `placementTeleport.test.ts` et ses voisines.
 */
function couvertureCharges(componentAbsent?: boolean): ReplayDocumentReady['coverage'] {
  return {
    abilityCharges: {
      reads: 0,
      published: 0,
      beforeOrigin: 0,
      unpublished: 0,
      noIdentity: 0,
      otherFamily: 0,
      noResolver: 0,
      ...(componentAbsent === undefined ? {} : { componentAbsent }),
    },
  } as unknown as ReplayDocumentReady['coverage']
}

/** Une couverture SANS le bloc des charges : l'artefact pré-38, ou un balayage en échec. */
const SANS_CALQUE = {} as unknown as ReplayDocumentReady['coverage']

function doc(over: Parameters<typeof testReplayDoc>[0] = {}) {
  return testReplayDoc({
    abilityLabels: LABELS,
    tracks: [LIFE_1, LIFE_2],
    coverage: couvertureCharges(),
    ...over,
  })
}

describe('measuredChargeFamilyOf', () => {
  it('reconnaît le grappin et le propulseur par la racine du libellé, dans les deux locales', () => {
    expect(measuredChargeFamilyOf(LABELS, 4)).toBe('grapple')
    expect(measuredChargeFamilyOf(LABELS, 5)).toBe('thruster')
    // Une locale suffit : la racine est cherchée dans le texte joint des deux.
    expect(measuredChargeFamilyOf({ '20': { fr: '', en: 'Grappleshot' } }, 20)).toBe('grapple')
    expect(measuredChargeFamilyOf({ '21': { fr: 'propulseur', en: '' } }, 21)).toBe('thruster')
  })

  it('rend null pour une famille jamais mesurée, un rang hors table, ou sans table', () => {
    expect(measuredChargeFamilyOf(LABELS, 6)).toBeNull() // répulseur : pas de canal i56
    expect(measuredChargeFamilyOf(LABELS, 9)).toBeNull() // rang hors table
    expect(measuredChargeFamilyOf(undefined, 4)).toBeNull()
  })
})

describe('hasAbilityChargeLayer — le calque doit exister avant d’affirmer quoi que ce soit', () => {
  it('artefact PRÉ-38 (ni couverture ni lecture) : pas de calque — RIEN, pas même « plein »', () => {
    // Le parc actuel est 100 % pré-38 : « rien transmis » n'y signifie pas « plein », le
    // canal n'a jamais été balayé. C'est le constat P0 de la revue P6.
    const d = doc({ coverage: SANS_CALQUE })
    expect(hasAbilityChargeLayer(d)).toBe(false)
    expect(abilityChargesAt(d, 5, 15, 4)).toBeNull()
  })

  it('couverture posée avec componentAbsent : le film ne transmet pas ce canal — rien', () => {
    const d = doc({ coverage: couvertureCharges(true) })
    expect(hasAbilityChargeLayer(d)).toBe(false)
    expect(abilityChargesAt(d, 5, 15, 4)).toBeNull()
  })

  it('une lecture publiée sert de filet quand la couverture manque', () => {
    // Un artefact antérieur au schéma 38 ne peut pas porter de lecture : si une lecture est
    // là, le calque existe — même construction que hasTranslocationLayer.
    const d = doc({
      coverage: SANS_CALQUE,
      abilityCharges: [{ t: 12, slot: 5, family: 'grapple', charges: 2 }],
    })
    expect(hasAbilityChargeLayer(d)).toBe(true)
    expect(abilityChargesAt(d, 5, 15, 4)).toEqual({ kind: 'count', charges: 2, age: 3 })
  })
})

describe('abilityChargesAt', () => {
  it('avant toute lecture : « plein » qualitatif, jamais un chiffre', () => {
    expect(abilityChargesAt(doc(), 5, 15, 4)).toEqual({ kind: 'full' })
  })

  it('famille jamais mesurée SANS lecture : rien — pas de « plein » pour un répulseur', () => {
    // Le cas de production le plus courant : répulseur (ou camo, surbouclier) porté, zéro
    // lecture — le canal ne porte pas cette famille, « plein » serait une invention. C'est
    // la mutation survivante du constat n°2 de la revue P6, tuée ici.
    expect(abilityChargesAt(doc(), 5, 15, 6)).toBeNull()
  })

  it('affiche la lecture la plus récente <= image courante, avec son âge', () => {
    const d = doc({
      abilityCharges: [
        { t: 12, slot: 5, family: 'grapple', charges: 3 },
        { t: 14, slot: 5, family: 'grapple', charges: 2 },
      ],
    })
    expect(abilityChargesAt(d, 5, 15, 4)).toEqual({ kind: 'count', charges: 2, age: 1 })
  })

  it('ignore une lecture À VENIR : avant elle, on est encore plein', () => {
    const d = doc({ abilityCharges: [{ t: 18, slot: 5, family: 'grapple', charges: 1 }] })
    expect(abilityChargesAt(d, 5, 15, 4)).toEqual({ kind: 'full' })
  })

  it('DEUX VIES du même slot : la lecture de la vie 1 ne colle pas à la vie 2', () => {
    const d = doc({ abilityCharges: [{ t: 15, slot: 5, family: 'grapple', charges: 1 }] })
    // Vie 2 (frame 35) : on a réapparu, l'équipement est plein — jamais le « 1 » de la vie 1.
    expect(abilityChargesAt(d, 5, 35, 4)).toEqual({ kind: 'full' })
    // Contrôle : une lecture de la vie 2 s'affiche, elle.
    const d2 = doc({ abilityCharges: [{ t: 32, slot: 5, family: 'grapple', charges: 4 }] })
    expect(abilityChargesAt(d2, 5, 35, 4)).toEqual({ kind: 'count', charges: 4, age: 3 })
  })

  it('re-ramassage dans la même vie : une lecture antérieure au taken est ignorée, retour à plein', () => {
    const d = doc({
      abilityCharges: [{ t: 12, slot: 5, family: 'grapple', charges: 1 }],
      equipmentChanges: [{ t: 14, slot: 5, kind: 'taken', r: 4, from: 4 }],
    })
    expect(abilityChargesAt(d, 5, 16, 4)).toEqual({ kind: 'full' })
    // Une lecture datée PILE au changement est ambiguë : traitée comme antérieure.
    const dTie = doc({
      abilityCharges: [{ t: 14, slot: 5, family: 'grapple', charges: 1 }],
      equipmentChanges: [{ t: 14, slot: 5, kind: 'taken', r: 4, from: 4 }],
    })
    expect(abilityChargesAt(dTie, 5, 16, 4)).toEqual({ kind: 'full' })
    // Contrôle : une lecture POSTÉRIEURE au taken décrit bien le nouvel équipement.
    const dApres = doc({
      abilityCharges: [{ t: 15, slot: 5, family: 'grapple', charges: 4 }],
      equipmentChanges: [{ t: 14, slot: 5, kind: 'taken', r: 4, from: 4 }],
    })
    expect(abilityChargesAt(dApres, 5, 16, 4)).toEqual({ kind: 'count', charges: 4, age: 1 })
  })

  it('famille jamais mesurée : rien d’affirmé — ni chiffre, ni plein', () => {
    // Même une lecture qui prétendrait cette famille ne crée pas d'affichage : la table
    // des familles mesurées est fermée, le répulseur n'arme jamais i56.
    const d = doc({ abilityCharges: [{ t: 12, slot: 5, family: 'repulsor', charges: 2 }] })
    expect(abilityChargesAt(d, 5, 15, 6)).toBeNull()
  })

  it('famille différente : rien — les charges d’un grappin ne collent pas à un propulseur', () => {
    // La vignette montre le propulseur (rang 5) ; la lecture la plus récente est du grappin.
    // Le canal contredit la vignette : ni le chiffre de l'autre famille, ni un « plein »
    // démenti par la lecture.
    const d = doc({ abilityCharges: [{ t: 12, slot: 5, family: 'grapple', charges: 2 }] })
    expect(abilityChargesAt(d, 5, 15, 5)).toBeNull()
  })

  it('sans vie couvrante à l’image : rien d’affirmé', () => {
    const d = doc({ abilityCharges: [{ t: 12, slot: 5, family: 'grapple', charges: 2 }] })
    expect(abilityChargesAt(d, 5, 25, 4)).toBeNull() // entre les deux vies
  })
})
