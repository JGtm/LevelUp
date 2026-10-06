/**
 * usagesText.test.ts — les textes de l'onglet « Usages » : titres et aides ⓘ FR copiés de la maquette v4
 * (`.ai/V7.5/MAQUETTE_TIMESERIES_USAGES_2026-10-05.html`, `renderApres`), « Mon camp » partout, jamais
 * « Notre camp » / « Our side » ni « nous / eux ». La parité FR / EN est tenue par le typage.
 */
import { describe, expect, it } from 'vitest'

import { EMPRISE_TEXT } from '@/features/squad/emprise/empriseStrings'

import { EMPRISE_TEXT_SOLO, OBJECTIF_TEXT_SOLO, USAGES_TEXT } from './usagesText'

const E = EMPRISE_TEXT_SOLO.fr
const O = OBJECTIF_TEXT_SOLO.fr
const U = USAGES_TEXT.fr

describe('textes FR = maquette v4', () => {
  it('intertitres de l’onglet, dans l’ordre', () => {
    expect([U.sections.bilan, U.sections.carte, U.sections.mine, U.sections.prendre, U.sections.lives, U.sections.objectif, U.sections.equipment]).toEqual([
      'Bilan du périmètre',
      'Carte par carte',
      'Mes prises',
      'Prendre, et s’en servir',
      'Près d’un coéquipier ou seul',
      'Objectif',
      'Équipement',
    ])
    expect(U.sections.bilanCoverage(62, 90)).toBe('62 matchs filmés sur 90 · frags de la feuille de match sur les 90')
  })

  it('une seule source des intertitres : les textes de l’Emprise solo ne surchargent pas ceux de l’Escouade', () => {
    for (const locale of ['fr', 'en'] as const) expect(EMPRISE_TEXT_SOLO[locale].sections).toBe(EMPRISE_TEXT[locale].sections)
  })

  it('Contrôle des ressources et au fil des matchs', () => {
    expect(E.control.title).toBe('Contrôle des ressources')
    expect(E.control.info).toBe(
      'La part de chaque ressource prise par mon camp face à l’adversaire, sur les matchs du périmètre. Les nombres sont des comptes ; le trait orange marque 50 %, autant que l’adversaire. Les bonus sans ramasseur connu ne comptent dans aucun camp.',
    )
    expect(E.fil.title).toBe('Contrôle des ressources au fil des matchs')
    expect(E.fil.info).toBe(
      'Ma part des prises de chaque ressource, cumulée depuis le premier match du périmètre. Les petits points sont la part de chaque match, leur taille son volume. Un match sans la ressource, ou sans film, laisse la courbe filer jusqu’au suivant.',
    )
    expect(U.cards.filCaption(90, 62)).toBe('90 matchs, dont 62 filmés')
  })

  it('Carte par carte', () => {
    expect(E.grid.title).toBe('Contrôle des ressources, carte par carte')
    expect(E.grid.info).toBe(
      'Une colonne par carte jouée, la plus jouée à gauche, avec son nombre de matchs et ses résultats. La couleur dit si mon camp a pris plus ou moins que l’adversaire, et sature à trente points d’écart. Le survol d’une case détaille qui l’a prise chez moi.',
    )
    expect([E.grid.more, E.grid.less, E.grid.nothing, E.grid.noFilm]).toEqual(['Plus que l’adversaire', 'Moins', 'Rien à prendre', 'Sans film'])
    expect(E.grid.untieredTip).toBe('Niveaux de socle non mesurés sur ces matchs : armes spéciales et armes de râtelier ne se séparent pas.')
    expect(U.cards.maps.tipHead('Carte Alpha', 12, 1)).toBe('Carte Alpha (12 matchs, 1 filmé)')
    expect(U.cards.maps.others).toBe('Autres cartes')
  })

  it('Mes prises dans mon camp', () => {
    expect(U.cards.mine.title).toBe('Mes prises dans mon camp')
    expect(U.cards.mine.info).toBe(
      'Chaque objet pris par mon camp : ma part et celle du reste du camp, en comptes, triés par volume de mon camp. Une répartition, pas un classement. Les bonus perdus sont ceux gardés sans être activés ou lâchés.',
    )
    expect([U.cards.mine.me, U.cards.mine.rest]).toEqual(['Moi', 'Reste de mon camp'])
  })

  it('Prendre, et s’en servir', () => {
    expect(E.production.title).toBe('Frags obtenus avec les ressources')
    expect(E.production.info).toBe(
      'La barre épaisse partage les frags obtenus grâce à la ressource, la barre fine ce qui les a permis (temps d’effet d’un bonus, prises d’une arme spéciale, temps à bord d’un véhicule), toutes deux sur les matchs où ce qui les a permis est mesuré. Si la coupure de la barre épaisse est à gauche de celle de la fine, on a moins produit qu’on n’a eu. Les frags de toute la période se lisent carte par carte.',
    )
    expect(E.yield.title).toBe('Rendement face à l’adversaire')
    expect(E.yield.info).toBe(
      'Combien mon camp produit de plus ou de moins que l’adversaire pour la même exposition : par minute d’effet d’un bonus, par prise d’arme spéciale, par minute à bord d’un véhicule. Zéro veut dire autant que lui. Les deux rendements bruts sont écrits de l’autre côté du zéro.',
    )
    expect(E.yield.tip('Bonus', 'frags par minute d’effet', 3, 2.7, '+11 %')).toBe('Bonus · frags par minute d’effet\nMon camp 3,0, adversaire 2,7 : +11 %')
  })

  it('Mes vies : près d’un coéquipier ou seul', () => {
    expect(U.cards.lives.title).toBe('Mes vies : près d’un coéquipier ou seul')
    expect(U.cards.lives.info(22, 0, 0)).toBe(
      'Chaque vie est rangée selon la distance au coéquipier le plus proche au moment de la mort : à moins d’une portée de radar, ou au-delà. La barre épaisse partage mes vies, la barre fine les frags obtenus pendant ces vies. Les vies terminées sans aucun coéquipier situé sont écartées (22 ici).',
    )
    expect(U.cards.lives.info(22, 5, 0)).toContain('sans portée de radar connue (5)')
    expect([U.cards.lives.near, U.cards.lives.alone, U.cards.lives.thinLegend]).toEqual([
      'Près d’un coéquipier',
      'Seul',
      'Barre fine : mes frags pendant ces vies',
    ])
  })

  it('Objectif', () => {
    expect(O.balance.title).toBe('Rapport de force par famille de mode')
    expect(O.balance.info).toBe(
      'Pour chaque action de l’objectif, ce que mon camp a fait face à l’adversaire, famille par famille. Le trait orange marque 50 % : autant que l’adversaire.',
    )
    expect(U.sheet.title).toBe('Ma part à l’objectif')
    expect(U.sheet.info).toBe(
      'La fiche du joueur affiché : ce qu’il a fait à l’objectif, action par action. La barre est sa part du total de son camp ; un zéro reste affiché, atténué. Le rôle dominant est celui où il pèse le plus dans son camp.',
    )
  })

  it('Équipement', () => {
    expect(U.cards.equipment.title).toBe('Équipement pris, et ce que j’en ai fait')
    expect(U.cards.equipment.info).toBe(
      'Pour chaque famille, ce que sont devenus mes objets : servis (posé pour le mur, charge consommée pour les autres), gardés sans servir, lâchés. Les comptes portent sur tout l’équipement tenu, celui de réapparition compris ; le sous-libellé dit combien en ont été pris sur la carte. La barre fine donne les mêmes trois parts pour le reste de mon camp. Le répulseur n’a pas de ligne : aucun canal ne mesure son usage. Seules les familles tenues par au moins un joueur du lobby sont listées.',
    )
    expect(USAGES_TEXT.en.cards.equipment.info).toMatch(/Only families held by at least one player in the lobby are listed\.$/)
    expect([U.cards.equipment.used, U.cards.equipment.kept, U.cards.equipment.dropped, U.cards.equipment.thinLegend]).toEqual([
      'Servi',
      'Gardé sans servir',
      'Lâché',
      'Barre fine : reste de mon camp',
    ])
    expect(U.cards.equipment.unmeasured).toBe('Non mesuré : ni prise ni usage publiés pour cette famille')
    expect(U.cards.equipment.restLine(146, 7, 151)).toBe('reste de mon camp : 146 servis · 7 gardés · 151 lâchés')
  })
})

describe('« Mon camp », jamais « Notre camp »', () => {
  /** Toutes les chaînes et toutes les sorties de fonction (appelées avec des valeurs témoins) d'un texte. */
  function strings(v: unknown): string[] {
    if (typeof v === 'string') return [v]
    if (typeof v === 'function') {
      // Deux jeux de témoins : des nombres d'abord, puis une forme « (qui, quoi, n, part, total, pct) ».
      const f = v as (...a: unknown[]) => unknown
      return [[3, 2, 1, '1 %', 'x', 'y'], ['x', 'y', 3, 'used', 4, '1 %']].flatMap((args) => {
        try {
          return strings(f(...args))
        } catch {
          return []
        }
      })
    }
    if (v && typeof v === 'object') return Object.values(v).flatMap(strings)
    return []
  }

  it('libellés de camp', () => {
    expect(E.ourSide).toBe('Mon camp')
    expect(O.ourSide).toBe('Mon camp')
    expect(EMPRISE_TEXT_SOLO.en.ourSide).toBe('My side')
    expect(OBJECTIF_TEXT_SOLO.en.ourSide).toBe('My side')
  })

  it.each([
    ['grille', () => [E.grid, E.fil, E.control, E.production, E.yield]],
    ['objectif', () => [O.balance, O.ourSide]],
    ['cartes', () => [U]],
  ])('aucun « Notre camp », « pour nous », « Chez nous » ni « Our side » (%s)', (_, pick) => {
    const all = strings(pick()).join('\n')
    expect(all).not.toMatch(/Notre camp|notre camp|pour nous|Chez nous|Our side|our side|for us\b/)
  })

  it.each([
    ['grille', () => [EMPRISE_TEXT_SOLO.en.grid, EMPRISE_TEXT_SOLO.en.fil, EMPRISE_TEXT_SOLO.en.control, EMPRISE_TEXT_SOLO.en.production, EMPRISE_TEXT_SOLO.en.yield]],
    ['cartes', () => [USAGES_TEXT.en]],
  ])('anglais : ni « Our side » ni « for us » (%s)', (_, pick) => {
    expect(strings(pick()).join('\n')).not.toMatch(/Our side|our side|for us\b|On our side/)
  })
})
