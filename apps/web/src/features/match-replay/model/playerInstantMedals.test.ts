/**
 * playerInstantMedals.test.ts — LES MÉDAILLES D'UN JOUEUR, GROUPÉES : un groupe par médaille avec son compte, dans
 * l'ordre de la première obtention, image, sprite et rareté lus dans la table du document, aucun visuel inventé.
 */
import { describe, expect, it } from 'vitest'

import type { MedalEvent } from './killFeedLogic'
import { medalGroupsOf } from './playerInstantMedals'

const medal = (name: string, label = '', imageUrl = ''): MedalEvent => ({
  tMs: 0, xuid: 'A', gamertag: '', name, label, description: '', imageUrl,
})

const DOC = {
  medalLabels: {
    Double: { en: 'Double Kill', fr: 'Doublé', img: '/d.png', difficulty: 'Normal' },
    Perfect: { en: 'Perfect', fr: 'Parfait', difficulty: 'Mythic', sprite: { sheet: '/s.png', left: 1, top: 2, width: 74, height: 74 } },
  },
}

describe('medalGroupsOf', () => {
  it('un groupe par médaille, dans l’ordre de la première obtention, avec son compte', () => {
    const groups = medalGroupsOf([medal('Perfect', 'Parfait'), medal('Double', 'Doublé', '/d.png'), medal('Perfect', 'Parfait')], DOC)
    expect(groups.map((g) => [g.name, g.count])).toEqual([['Perfect', 2], ['Double', 1]])
  })

  it('l’image du fil, la découpe de sprites et la rareté de la table', () => {
    const [parfait, double] = medalGroupsOf([medal('Perfect', 'Parfait'), medal('Double', 'Doublé', '/d.png')], DOC)
    expect(parfait).toMatchObject({ sprite: { sheet: '/s.png', left: 1, top: 2, width: 74, height: 74 }, difficulty: 'Mythic', imageUrl: '' })
    expect(double).toMatchObject({ imageUrl: '/d.png', sprite: null, difficulty: 'Normal' })
  })

  it('une médaille hors de la table garde son nom : ni image, ni sprite, ni rareté', () => {
    const [m] = medalGroupsOf([medal('Mystery')], DOC)
    expect(m).toMatchObject({ label: 'Mystery', imageUrl: '', sprite: null, difficulty: null })
  })

  it('une table de libellés absente : les médailles gardent leur nom', () => {
    expect(medalGroupsOf([medal('Double', 'Doublé')], {})[0]).toMatchObject({ label: 'Doublé', sprite: null, difficulty: null })
  })

  it('aucune liste, ou une liste vide : aucun groupe', () => {
    expect(medalGroupsOf(null, DOC)).toEqual([])
    expect(medalGroupsOf([], DOC)).toEqual([])
  })
})
