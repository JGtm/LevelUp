/**
 * sessionEmpriseText.test.ts — les textes des cartes « Frags et usages » de Sessions : titres, ⓘ et
 * légendes FR de la maquette (`make*`, pleine page `cp = false` et comparaison `cp = true`), mot pour
 * mot ; aucun « Notre camp », aucun « périmètre » (D12, V6).
 */
import { describe, expect, it } from 'vitest'

import { SESSION_CARD_TEXT } from './sessionEmpriseText'

const fr = SESSION_CARD_TEXT.fr

/** Toutes les chaînes d'un objet de textes (fonctions appelées avec des valeurs témoins). */
function strings(o: unknown, out: string[] = [], depth = 0): string[] {
  if (depth > 6 || o == null) return out
  if (typeof o === 'string') out.push(o)
  else if (typeof o === 'function') {
    try {
      const r = (o as (...a: unknown[]) => unknown)(3, 7, 'x', 'y', 'z', 'w')
      if (typeof r === 'string') out.push(r)
    } catch {
      // un formateur qui attend un objet : ignoré
    }
  } else if (typeof o === 'object') for (const v of Object.values(o)) strings(v, out, depth + 1)
  return out
}

describe('SESSION_CARD_TEXT — pleine page (maquette, cp = false)', () => {
  it('A et B : titres et ⓘ', () => {
    expect(fr.full.squad.performanceCharts.fragBreakdownTitle).toBe('Répartition des frags')
    expect(fr.full.squad.performanceCharts.fragBreakdownInfo).toBe(
      'Mes frags de la soirée, par classe d’arme. Le nombre écrit dans un segment est son compte de frags ; le total est au bout de la barre.',
    )
    expect(fr.full.squad.weaponKills.title).toBe('Outils de destruction')
    expect(fr.full.squad.weaponKills.info).toBe(
      'Mes frags, arme par arme, sur la soirée. La pastille devant l’arme est la couleur de sa classe dans la Répartition des frags.',
    )
  })

  it('C, D, E : titres, ⓘ et légendes', () => {
    const e = fr.full.emprise
    expect(e.control.title).toBe('Contrôle des ressources')
    expect(e.control.info).toBe(
      'La part de chaque ressource prise par mon camp face à l’adversaire, sur les matchs de la soirée. Les nombres sont des comptes. Le trait orange marque 50 %, autant que l’adversaire. Les bonus sans ramasseur connu ne comptent dans aucun camp.',
    )
    expect([e.ourSide, e.opponent, e.parity]).toEqual(['Mon camp', 'Adversaire', '50 % : autant que l’adversaire'])
    expect(e.fil.title).toBe('Contrôle des ressources au fil de la session')
    expect(e.fil.info).toBe(
      'Ma part des prises de chaque ressource, cumulée depuis le premier match de la soirée. Les petits points sont la part de chaque match, leur taille son volume. Un match sans la ressource, ou sans film, laisse la courbe filer jusqu’au suivant.',
    )
    expect(e.fil.winLoss).toBe('Victoire, défaite')
    expect(e.grid.title).toBe('Contrôle des ressources, match par match')
    expect(e.grid.info).toBe(
      'Une colonne par match de la soirée, dans l’ordre, avec sa carte, son mode et son résultat. La couleur dit si mon camp a pris plus ou moins que l’adversaire, et sature à trente points d’écart. Le survol d’une case détaille qui l’a prise chez moi.',
    )
    expect([e.grid.more, e.grid.less, e.grid.nothing, e.grid.noFilm]).toEqual(['Plus que l’adversaire', 'Moins', 'Rien à prendre', 'Sans film'])
  })

  it('F, G, H, I', () => {
    expect(fr.full.cards.mine.title).toBe('Mes prises dans mon camp')
    expect(fr.full.cards.mine.info).toBe(
      'Chaque objet pris par mon camp : ma part et celle du reste du camp, en comptes, triés par volume de mon camp. Une répartition, pas un classement. Les bonus perdus sont ceux gardés sans être activés ou lâchés.',
    )
    expect([fr.full.cards.mine.me, fr.full.cards.mine.rest]).toEqual(['Moi', 'Reste de mon camp'])
    expect(fr.full.emprise.production.title).toBe('Frags obtenus avec les ressources')
    expect(fr.full.emprise.production.info).toBe(
      'La barre épaisse partage les frags obtenus grâce à la ressource, la barre fine ce qui les a permis (temps d’effet d’un bonus, prises d’une arme spéciale, temps à bord d’un véhicule), toutes deux sur les matchs où ce qui les a permis est mesuré. Si la coupure de la barre épaisse est à gauche de celle de la fine, on a moins produit qu’on n’a eu.',
    )
    expect(fr.full.emprise.production.thinLegend).toBe('Barre fine : temps d’effet, prises ou temps à bord')
    expect(fr.full.emprise.yield.title).toBe('Rendement face à l’adversaire')
    expect(fr.full.emprise.yield.info).toBe(
      'Combien mon camp produit de plus ou de moins que l’adversaire pour la même exposition : par minute d’effet d’un bonus, par prise d’arme spéciale, par minute à bord d’un véhicule. Zéro veut dire autant que lui. Les deux rendements bruts sont écrits de l’autre côté du zéro.',
    )
    expect([fr.full.emprise.yield.more, fr.full.emprise.yield.less]).toEqual(['Plus productifs que l’adversaire', 'Moins'])
    expect(fr.full.cards.lives.title).toBe('Mes vies : près d’un coéquipier ou seul')
    expect(fr.full.cards.lives.info(3, 0, 0)).toBe(
      'Chaque vie est rangée selon la distance au coéquipier le plus proche au moment de la mort : à moins d’une portée de radar, ou au-delà. La barre épaisse partage mes vies, la barre fine les frags obtenus pendant ces vies. Les vies terminées sans aucun coéquipier situé sont écartées (3 ici).',
    )
  })

  it('J, K, L', () => {
    expect(fr.full.objectif.balance.title).toBe('Rapport de force par famille de mode')
    expect(fr.full.objectif.balance.info).toBe(
      'Pour chaque action de l’objectif, ce que mon camp a fait face à l’adversaire, famille par famille. Le trait orange marque 50 % : autant que l’adversaire. Les prises nettes de drapeau (lues dans le film) ne comptent pas les jonglages.',
    )
    expect(fr.full.sheet.title).toBe('Ma part à l’objectif')
    expect(fr.full.sheet.info).toBe(
      'La fiche du joueur affiché : ce qu’il a fait à l’objectif, action par action. La barre est sa part du total de son camp ; un zéro reste affiché, atténué. Le rôle dominant est celui où il pèse le plus dans son camp.',
    )
    expect(fr.full.cards.equipment.title).toBe('Équipement pris, et ce que j’en ai fait')
    expect(fr.full.cards.equipment.info).toBe(
      'Pour chaque famille, ce que sont devenus mes objets : servis (posé pour le mur, charge consommée pour les autres), gardés sans servir, lâchés. Les comptes portent sur tout l’équipement tenu, celui de réapparition compris ; le sous-libellé dit combien en ont été pris sur la carte. La barre fine donne les mêmes trois parts pour le reste de mon camp. Le répulseur n’a pas de ligne : aucun canal ne mesure son usage.',
    )
  })

  it('couverture de l’intertitre « Ressources de la soirée »', () => {
    expect(fr.full.coverage(6, 7)).toBe('6 matchs filmés sur 7 · frags de la feuille de match sur les 7')
  })
})

describe('SESSION_CARD_TEXT — comparaison (maquette, cp = true)', () => {
  it('ⓘ propres à la vue compacte : A, B, C, E, F, J, K, L', () => {
    const c = fr.compact
    expect(c.squad.performanceCharts.fragBreakdownInfo).toBe(
      'Mes frags de la soirée, par classe d’arme. Chaque segment porte sa part de mes frags ; le compte est au survol.',
    )
    expect(c.squad.weaponKills.info).toBe(
      'Mes six outils les plus meurtriers de la soirée, en part de mes frags. La pastille devant l’arme est la couleur de sa classe ; le compte est au survol.',
    )
    expect(c.emprise.control.info).toBe(
      'La part de chaque ressource prise par mon camp face à l’adversaire, sur les matchs de la soirée. Les segments portent les parts ; les comptes sont au survol. Le trait orange marque 50 %, autant que l’adversaire. Les bonus sans ramasseur connu ne comptent dans aucun camp.',
    )
    expect(c.emprise.grid.info).toBe(
      'Une colonne par match de la soirée. Chaque case est la part de mon camp dans la ressource (plus que l’adversaire en vert, moins en rouge, saturation à trente points d’écart) ; les comptes et qui l’a prise chez moi sont au survol.',
    )
    expect([c.emprise.grid.more, c.emprise.grid.less, c.emprise.grid.nothing, c.emprise.grid.noFilm]).toEqual([
      'Plus de 50 %',
      'Moins de 50 %',
      'Rien à prendre',
      'Sans film, non mesuré',
    ])
    expect(c.cards.mine.info).toBe(
      'Pour chaque ressource, ma part des prises de mon camp et celle du reste du camp, en pourcentage ; les comptes sont au survol. Les bonus perdus sont ceux gardés sans être activés ou lâchés, en part des bonus pris par chaque camp.',
    )
    expect(c.objectif.balance.info).toBe(
      'Pour chaque rôle de l’objectif (somme de ses actions ; Tenir en durée), la part de mon camp face à l’adversaire, famille par famille ; les comptes sont au survol. Le trait orange marque 50 % : autant que l’adversaire. Les prises nettes de drapeau (lues dans le film) ne comptent pas les jonglages.',
    )
    expect(c.sheet.info).toBe(
      'La fiche du joueur affiché : ce qu’il a fait à l’objectif, action par action. La barre est sa part du total de son camp, et le nombre à droite aussi (compte au survol) ; un zéro reste affiché, atténué. Le rôle dominant est celui où il pèse le plus dans son camp.',
    )
    expect(c.sheet.pctFmt(31.8)).toBe('32 %')
    expect(c.cards.equipment.info).toBe(
      'Pour chaque famille, ce que sont devenus mes objets : servis (posé pour le mur, charge consommée pour les autres), gardés sans servir, lâchés, en part de mes objets (comptes au survol). Les comptes portent sur tout l’équipement tenu, celui de réapparition compris ; le sous-libellé dit combien en ont été pris sur la carte. La barre fine donne les mêmes trois parts pour le reste de mon camp. Le répulseur n’a pas de ligne : aucun canal ne mesure son usage.',
    )
  })

  it('D, G, H, I : mêmes ⓘ que la pleine page', () => {
    expect(fr.compact.emprise.fil.info).toBe(fr.full.emprise.fil.info)
    expect(fr.compact.emprise.production.info).toBe(fr.full.emprise.production.info)
    expect(fr.compact.emprise.yield.info).toBe(fr.full.emprise.yield.info)
    expect(fr.compact.cards.lives.info(3, 0, 0)).toBe(fr.full.cards.lives.info(3, 0, 0))
  })

  it('formateurs compacts : sous-libellés et lignes de la maquette', () => {
    const k = SESSION_CARD_TEXT.fr.compactCards
    expect(k.frag.totalSub(65)).toBe('65 frags')
    expect(k.frag.pctFmt(33.8)).toBe('34 %')
    expect(k.production.exposureLine('temps d’effet', '58 %')).toBe('temps d’effet : 58 %')
    expect(k.mine.resourceSub).toBe('prises de mon camp')
    expect(k.equipment.sub(84)).toBe('84 objets')
    expect(k.equipment.sub(0)).toBe('0 objet')
    expect(k.equipment.restUsed('48 %')).toBe('reste de mon camp : 48 % servis')
    expect(k.lives.killsLine('90 %', '0,7')).toBe('frags : 90 % · 0,7 par vie')
    expect(k.lives.killsLineAlone('0,2')).toBe('0,2 par vie')
  })
})

describe('SESSION_CARD_TEXT — vocabulaire de la page (V6)', () => {
  it('aucun « Notre camp », aucun « périmètre » ; ni « Our side » ni « scope » en anglais', () => {
    // Les dictionnaires de l'Escouade portent toute leur page (fiches de prises, habitude…) : on lit ce
    // que les cartes de Sessions écrivent — les parties de l'Emprise montées par C à H, et les deux ⓘ
    // surchargées de A et B.
    const ours = (v: typeof fr.full) => {
      const e = v.emprise
      return [e.control, e.fil, e.production, e.yield, e.grid, e.ourSide, v.objectif.balance, v.objectif.ourSide, v.cards, v.sheet, v.squad.performanceCharts.fragBreakdownInfo, v.squad.weaponKills.info]
    }
    const frStrings = [...strings(ours(fr.full)), ...strings(ours(fr.compact))].join('\n')
    expect(frStrings).not.toMatch(/Notre camp|notre camp|périmètre/)
    const en = SESSION_CARD_TEXT.en
    const enStrings = [...strings(ours(en.full)), ...strings(ours(en.compact))].join('\n')
    expect(enStrings).not.toMatch(/Our side|our side|\bscope\b/)
  })
})
