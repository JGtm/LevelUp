/**
 * usagesText.test.ts — les textes de l'onglet « Usages » : intertitres et titres concis et factuels,
 * aides ⓘ d'une ou deux phrases (ce qui est mesuré, sur quel périmètre), aucune personne (la garde
 * générale est `textesSansPersonne.test.ts`). La parité FR / EN est tenue par le typage.
 */
import { describe, expect, it } from 'vitest'

import { EMPRISE_TEXT } from '@/features/squad/emprise/empriseStrings'

import { EMPRISE_TEXT_SOLO, OBJECTIF_TEXT_SOLO, USAGES_TEXT } from './usagesText'

const E = EMPRISE_TEXT_SOLO.fr
const O = OBJECTIF_TEXT_SOLO.fr
const U = USAGES_TEXT.fr

/** Le nombre de phrases d'une aide (point final, ou point suivi d'un blanc). */
const sentences = (s: string) => s.split(/\.(?:\s|$)/).filter((p) => p.trim() !== '').length

describe('textes FR', () => {
  it('intertitres de l’onglet, dans l’ordre', () => {
    expect([U.sections.bilan, U.sections.carte, U.sections.mine, U.sections.prendre, U.sections.lives, U.sections.objectif, U.sections.equipment]).toEqual([
      'Ressources',
      'Par carte',
      'Prises',
      'Rendement des ressources',
      'Isolement',
      'Objectif',
      'Équipement',
    ])
  })

  it('une seule source des intertitres : les textes de l’Emprise solo ne surchargent pas ceux de l’Escouade', () => {
    for (const locale of ['fr', 'en'] as const) expect(EMPRISE_TEXT_SOLO[locale].sections).toBe(EMPRISE_TEXT[locale].sections)
  })

  it('titres des cartes', () => {
    expect([E.control.title, E.fil.title, E.grid.title, U.cards.mine.title, E.production.title, E.yield.title]).toEqual([
      'Contrôle des ressources',
      'Contrôle des ressources, cumul par match',
      'Contrôle des ressources, par carte',
      'Contribution aux prises',
      'Frags par ressource',
      'Rendement par ressource',
    ])
    expect([U.cards.lives.title, O.balance.title, U.sheet.title, U.cards.equipment.title]).toEqual([
      'Isolement',
      'Rapport de force',
      'Part du joueur à l’objectif',
      'Usage d’équipements',
    ])
  })

  it('aides ⓘ : deux phrases au plus, périmètre du solo, sans phrase de lecture', () => {
    const aides = [E.control.info, E.fil.info, E.grid.info, U.cards.mine.info, E.production.info, E.yield.info, O.balance.info, U.sheet.info]
    for (const a of aides) expect(sentences(a)).toBeLessThanOrEqual(2)
    expect(E.control.info).toContain('sur les matchs filmés du périmètre')
    expect(E.production.info).not.toMatch(/coupure|moins produit/)
    expect(U.cards.mine.info).not.toMatch(/classement/)
  })

  it('infobulles et légendes : équipe, adversaire, reste de l’équipe', () => {
    expect(E.fil.pointTip({ match: 'M', outcome: null, resource: 'Bonus', us: 3, them: 2, pct: '60 %', cumUs: 3, cumTotal: 5, cumPct: '60 %' })).toBe(
      'M\nBonus : équipe 3, adversaire 2 (60 %)\nCumul : 3 sur 5 (60 %)',
    )
    expect(E.yield.tip('Bonus', 'frags par minute d’effet', 3, 2.7, '+11 %')).toBe('Bonus · frags par minute d’effet\nÉquipe 3,0, adversaire 2,7 : +11 %')
    expect(E.grid.cellTip('Camouflage', 4, 1, '80 %')).toBe('Camouflage : équipe 4, adversaire 1 (80 %)')
    expect(U.cards.mine.meTip('JGtm', 'Camouflage', 3, 7)).toBe('JGtm · Camouflage\n3 des 7 prises de l’équipe')
    expect(U.cards.mine.rest).toBe('Reste de l’équipe')
    expect(U.sheet.lineTip('JGtm', 'Drapeau', 'Captures', '2', '5', '40 %')).toBe('JGtm · Drapeau\nCaptures : 2 des 5 de l’équipe (40 %)')
  })

  it('carte par carte', () => {
    expect([E.grid.more, E.grid.less, E.grid.nothing]).toEqual(['Plus que l’adversaire', 'Moins', 'Rien à prendre'])
    expect(U.cards.maps.tipHead('Carte Alpha', 12)).toBe('Carte Alpha (12 matchs)')
    expect(U.cards.maps.others).toBe('Autres cartes')
  })

  it('vies : l’aide dit ce que la carte range, sans compte d’écartées', () => {
    expect(U.cards.lives.info).toBe(
      'Vies terminées par une mort, rangées selon la distance au coéquipier le plus proche à l’instant de la mort (à portée de radar ou au-delà) ; barre fine : frags obtenus pendant ces vies.',
    )
    expect([U.cards.lives.near, U.cards.lives.alone, U.cards.lives.thinLegend]).toEqual([
      'À portée d’un coéquipier',
      'Isolée',
      'Barre fine : frags du joueur pendant ces vies',
    ])
  })

  it('équipement', () => {
    expect(U.cards.equipment.info).toContain('Seules les familles tenues dans le lobby sont listées')
    expect(USAGES_TEXT.en.cards.equipment.info).toContain('Only families held in the lobby are listed')
    expect([U.cards.equipment.used, U.cards.equipment.kept, U.cards.equipment.dropped, U.cards.equipment.thinLegend]).toEqual([
      'Servi',
      'Gardé sans servir',
      'Lâché',
      'Barre fine : reste de l’équipe',
    ])
    expect(U.cards.equipment.restLine(146, 7, 151)).toBe('reste de l’équipe : 146 servis · 7 gardés · 151 lâchés')
    expect(U.cards.equipment.zeroTip('JGtm', 'Mur')).toBe('Mur : 0 objet pour JGtm')
  })
})

describe('textes EN', () => {
  const EN = USAGES_TEXT.en
  it('intertitres et titres', () => {
    expect([EN.sections.bilan, EN.sections.carte, EN.sections.mine, EN.sections.prendre, EN.sections.lives]).toEqual([
      'Resources',
      'By map',
      'Pickups',
      'Resource efficiency',
      'Isolation',
    ])
    expect([EMPRISE_TEXT_SOLO.en.control.title, EMPRISE_TEXT_SOLO.en.grid.title, EN.cards.mine.title, EN.cards.equipment.title]).toEqual([
      'Resource control',
      'Resource control, by map',
      'Pickup contribution',
      'Equipment use',
    ])
  })

  it('libellés de camp', () => {
    expect([E.ourSide, O.ourSide]).toEqual(['Équipe', 'Équipe'])
    expect([EMPRISE_TEXT_SOLO.en.ourSide, OBJECTIF_TEXT_SOLO.en.ourSide]).toEqual(['Team', 'Team'])
  })
})
