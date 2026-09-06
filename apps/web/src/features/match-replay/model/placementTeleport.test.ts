/**
 * LES USAGES DU TRANSLOCATEUR — ce que le document DIT, et ce que le client a le droit d'en
 * affirmer.
 *
 * L'HEURISTIQUE SPATIALE N'EST PLUS TESTÉE PARCE QU'ELLE N'EXISTE PLUS (2026-09-03) : ses deux
 * verrous — le porteur et la porte de carte — cherchaient dans les PISTES ce que l'événement
 * 117 dit désormais lui-même, et tout seuil de distance invente une règle (le corpus porte un
 * saut de 3,24 m, invisible à tous les seuils). Ce fichier vérifie donc les trois choses qui
 * restent : la lecture de l'événement, le REPLI daté des artefacts antérieurs au schéma 38, et
 * le refus d'affirmer sur une identité que le témoin de compteur dit inconnue.
 */
import { describe, expect, it } from 'vitest'

import {
  hasTranslocationLayer,
  identityIsUnknown,
  lastTeleportAge,
  spentTranslocations,
  teleportMoments,
  translocationLinks,
  translocationMoments,
  translocatorRanks,
  type TranslocationMoment,
} from './placementTeleport'
import type { ReplayDocumentReady } from './replayNormalize'
import { testReplayDoc } from '../test/testDoc'

type Translocation = ReplayDocumentReady['translocations'][number]
type Change = ReplayDocumentReady['equipmentChanges'][number]

/** Une téléportation SITUÉE : instant, vie, départ et arrivée. */
function saut(slot: number, t: number, from: [number, number], to: [number, number]): Translocation {
  return { slot, t, fx: from[0], fy: from[1], fz: 0, tx: to[0], ty: to[1], tz: 0 }
}

/** Une téléportation dont la charge n'a pas été lue : datée, jamais située. */
function sautSansPosition(slot: number, t: number): Translocation {
  return { slot, t }
}

function change(over: Partial<Change> & Pick<Change, 'kind'>): Change {
  return { slot: 3, t: 40, r: -1, from: -1, ...over }
}

/**
 * La COUVERTURE réduite au seul témoin qui compte ici : la PRÉSENCE de `translocations`, que le
 * constructeur du schéma 38 pose sans condition. Le type du contrat exige cinq compteurs
 * obligatoires dont aucun ne joue dans ces cas — les fabriquer noierait le fait testé, d'où le
 * cast, comme le font déjà les fixtures de couverture voisines.
 */
function calqueLu(positioned = 0): ReplayDocumentReady['coverage'] {
  return {
    translocations: { events: positioned, published: positioned, beforeOrigin: 0, unpublished: 0, positioned },
  } as unknown as ReplayDocumentReady['coverage']
}

describe('translocationMoments — l instant vient de l événement, avec ou sans position', () => {
  it('date TOUTE translocation publiée, y compris celle sans va-et-vient lisible', () => {
    expect(
      translocationMoments([saut(3, 10, [0, 0], [20, 5]), sautSansPosition(7, 42)]),
    ).toEqual([
      { slot: 3, frame: 10 },
      { slot: 7, frame: 42 },
    ])
  })
})

describe('translocationLinks — le lien se lit sur l événement, jamais sur la piste', () => {
  it('rend les deux bouts du saut, dans l ordre des images', () => {
    const links = translocationLinks([
      saut(3, 50, [10, 10], [40, 12]),
      saut(9, 20, [0, 0], [5, 5]),
    ])
    expect(links.map((l) => l.frame)).toEqual([20, 50])
    expect(links[1]).toEqual({
      slot: 3,
      frame: 50,
      from: { x: 10, y: 10 },
      to: { x: 40, y: 12 },
    })
  })

  it('une translocation SANS positions ne trace RIEN — jamais un saut vers l origine', () => {
    expect(translocationLinks([sautSansPosition(3, 10)])).toEqual([])
  })

  it('une coordonnée à zéro est une VALEUR, pas une absence', () => {
    const links = translocationLinks([saut(3, 10, [0, 0], [0, 0])])
    expect(links).toHaveLength(1)
    expect(links[0].from).toEqual({ x: 0, y: 0 })
  })
})

describe('identityIsUnknown — un `from` sous saut de compteur ne dit plus rien', () => {
  it('une chaîne saine (pas de `gap`) garde son identité', () => {
    expect(identityIsUnknown(change({ kind: 'spent', from: 11 }))).toBe(false)
    expect(identityIsUnknown(change({ kind: 'spent', from: 11, gap: 0 }))).toBe(false)
  })

  it('un saut résiduel rend l identité INCONNUE', () => {
    expect(identityIsUnknown(change({ kind: 'spent', from: 11, gap: 1 }))).toBe(true)
  })
})

describe('spentTranslocations — le REPLI des artefacts antérieurs au schéma 38', () => {
  it('retient les `spent` dont le rang consommé est un translocateur, et eux seuls', () => {
    const moments = spentTranslocations(
      [
        change({ kind: 'spent', from: 11 }),
        change({ kind: 'spent', from: 4 }),
        change({ kind: 'taken', from: 11, slot: 5, t: 60 }),
      ],
      new Set([11]),
    )
    expect(moments).toEqual([{ slot: 3, frame: 40 }])
  })

  it('suit la table des rangs — famille B comprise', () => {
    expect(
      spentTranslocations([change({ kind: 'spent', from: 21, slot: 7, t: 12 })], new Set([21])),
    ).toEqual([{ slot: 7, frame: 12 }])
  })

  it('S ABSTIENT sous `gap` — le cas mesuré de JGtm, dont le `spent` porte le grappin', () => {
    expect(
      spentTranslocations([change({ kind: 'spent', from: 11, gap: 1 })], new Set([11])),
    ).toEqual([])
  })

  it('les moments nourrissent lastTeleportAge comme ceux de l événement', () => {
    const moments = spentTranslocations([change({ kind: 'spent', from: 11 })], new Set([11]))
    expect(lastTeleportAge(moments, 3, 45)).toBe(5)
  })
})

describe('teleportMoments — un seul canal à la fois', () => {
  it('l ÉVÉNEMENT gagne dès que le calque a tourné, même à zéro téléportation', () => {
    const doc = testReplayDoc({
      coverage: calqueLu(),
      translocations: [],
      equipmentChanges: [change({ kind: 'spent', from: 11 })],
      abilityLabels: { '11': { fr: 'Translocateur quantique', en: 'Quantum Translocator' } },
    })
    expect(hasTranslocationLayer(doc)).toBe(true)
    // Le `spent` est là, il porte le bon rang, et il n'allume RIEN : le calque publié fait foi.
    expect(teleportMoments(doc)).toEqual([])
  })

  it('une téléportation publiée SANS couverture atteste le calque à elle seule', () => {
    const doc = testReplayDoc({ translocations: [saut(3, 10, [0, 0], [20, 0])] })
    expect(hasTranslocationLayer(doc)).toBe(true)
  })

  it('sans le calque (artefact < 38), le REPLI du `spent` prend la main', () => {
    const doc = testReplayDoc({
      equipmentChanges: [change({ kind: 'spent', from: 11 })],
      abilityLabels: { '11': { fr: 'Translocateur quantique', en: 'Quantum Translocator' } },
    })
    expect(hasTranslocationLayer(doc)).toBe(false)
    expect(teleportMoments(doc)).toEqual([{ slot: 3, frame: 40 }])
  })

  it('avec le calque, les instants sont ceux de l événement', () => {
    const doc = testReplayDoc({
      coverage: calqueLu(1),
      translocations: [saut(3, 10, [0, 0], [20, 0])],
      equipmentChanges: [change({ kind: 'spent', from: 11, t: 175 })],
      abilityLabels: { '11': { fr: 'Translocateur quantique', en: 'Quantum Translocator' } },
    })
    // 165 frames = 16,5 s de retard : c'est exactement ce que le `spent` coûtait.
    expect(teleportMoments(doc)).toEqual([{ slot: 3, frame: 10 }])
  })
})

describe('lastTeleportAge — l âge du passage le plus récent d un slot', () => {
  const passage = (slot: number, frame: number): TranslocationMoment => ({ slot, frame })

  it('rend -1 quand ce slot n a aucun passage advenu', () => {
    expect(lastTeleportAge([], 3, 50)).toBe(-1)
    expect(lastTeleportAge([passage(7, 10)], 3, 50)).toBe(-1)
  })

  it('un passage À VENIR ne compte pas', () => {
    expect(lastTeleportAge([passage(3, 60)], 3, 50)).toBe(-1)
  })

  it('rend l âge du plus récent quand il y en a plusieurs', () => {
    expect(lastTeleportAge([passage(3, 10), passage(3, 40)], 3, 50)).toBe(10)
  })
})

describe('translocatorRanks — la table du document nomme les rangs, le littéral n est qu un repli', () => {
  it('lit la table `abilityLabels` sur la racine du mot, dans les deux locales', () => {
    expect(
      translocatorRanks({
        '11': { fr: 'Translocateur quantique', en: 'Quantum Translocator' },
        '3': { fr: 'Grappin', en: 'Grappleshot' },
      }),
    ).toEqual(new Set([11]))
    expect(
      translocatorRanks({
        '21': { fr: 'Translocateur quantique', en: 'Quantum Translocator' },
      }),
    ).toEqual(new Set([21]))
  })

  it('retombe sur le rang 11 (famille A) sans table, ou sans rang reconnu', () => {
    expect(translocatorRanks(undefined)).toEqual(new Set([11]))
    expect(translocatorRanks({ '20': { fr: 'Grappin', en: 'Grappleshot' } })).toEqual(new Set([11]))
  })
})
