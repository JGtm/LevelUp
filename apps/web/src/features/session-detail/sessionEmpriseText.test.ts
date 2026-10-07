/**
 * sessionEmpriseText.test.ts — les textes des cartes « Frags et usages » de Sessions : les cartes de
 * l'Escouade et des Séries temporelles gardent leurs titres et leurs aides ; seules les aides propres à
 * la page (cartes de frags, « Contribution aux prises » portée sur la soirée, vue compacte en parts)
 * sont écrites ici. Aucune portée « du périmètre » (V6). L'absence de personne est tenue par
 * `timeseries/usages/textesSansPersonne.test.ts`.
 */
import { describe, expect, it } from 'vitest'

import { EMPRISE_TEXT } from '@/features/squad/emprise/empriseStrings'
import { OBJECTIF_TEXT } from '@/features/squad/objectif/objectifStrings'
import { USAGES_TEXT } from '@/features/timeseries/usages/usagesText'

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

describe('SESSION_CARD_TEXT — pleine page', () => {
  it('A et B : titres de l’Escouade, aides propres à la page', () => {
    expect(fr.full.squad.performanceCharts.fragBreakdownTitle).toBe('Répartition des frags')
    expect(fr.full.squad.performanceCharts.fragBreakdownInfo).toBe(
      'Frags du joueur sur la soirée : anneau intérieur par classe d’arme, anneau extérieur par rôle ; total au centre, part de chaque classe en légende.',
    )
    expect(fr.full.squad.weaponKills.title).toBe('Outils de destruction')
    expect(fr.full.squad.weaponKills.info).toBe(
      'Frags du joueur sur la soirée, arme par arme ; couleur de la barre : classe de l’arme, celle de la Répartition des frags.',
    )
  })

  it('C à H : l’Emprise de l’Escouade, sans variante', () => {
    for (const locale of ['fr', 'en'] as const) {
      expect(SESSION_CARD_TEXT[locale].full.emprise).toBe(EMPRISE_TEXT[locale])
    }
    const e = fr.full.emprise
    expect([e.control.title, e.fil.title, e.grid.title]).toEqual([
      'Contrôle des ressources',
      'Contrôle des ressources, cumul par match',
      'Contrôle des ressources, par match',
    ])
    expect([e.production.title, e.yield.title]).toEqual(['Frags par ressource', 'Rendement par ressource'])
    expect([e.ourSide, e.opponent]).toEqual(['Équipe', 'Adversaire'])
  })

  it('F, I, K, L : les cartes des Séries temporelles ; F porte sur la soirée', () => {
    for (const locale of ['fr', 'en'] as const) {
      const v = SESSION_CARD_TEXT[locale].full
      const u = USAGES_TEXT[locale]
      expect(v.cards.lives).toBe(u.cards.lives)
      expect(v.cards.equipment).toBe(u.cards.equipment)
      expect(v.sheet).toBe(u.sheet)
      expect({ ...v.cards.mine, info: '' }).toEqual({ ...u.cards.mine, info: '' })
    }
    expect([fr.full.cards.mine.title, fr.full.cards.lives.title, fr.full.sheet.title, fr.full.cards.equipment.title]).toEqual([
      'Contribution aux prises',
      'Isolement',
      'Part du joueur à l’objectif',
      'Usage d’équipements',
    ])
    expect(fr.full.cards.mine.info).toBe(
      'Objets pris par l’équipe sur les matchs filmés de la soirée : part du joueur et du reste de l’équipe, en comptes, par volume décroissant. Bonus perdus : gardés sans être activés, ou lâchés.',
    )
  })

  it('J : l’Objectif de l’Escouade, sans variante', () => {
    expect(fr.full.objectif).toBe(OBJECTIF_TEXT.fr)
    expect(fr.full.objectif.balance.title).toBe('Rapport de force')
  })

  it('couverture de l’intertitre « Ressources »', () => {
    expect(fr.full.coverage(6, 7)).toBe('6 matchs filmés sur 7 · frags de la feuille de match sur les 7')
  })
})

describe('SESSION_CARD_TEXT — comparaison (vue compacte)', () => {
  it('aides propres à la vue compacte : A, B, C, E, F, J, K, L', () => {
    const c = fr.compact
    // A : le même anneau dans les deux vues, la même aide.
    expect(c.squad.performanceCharts.fragBreakdownInfo).toBe(fr.full.squad.performanceCharts.fragBreakdownInfo)
    expect(c.squad.weaponKills.info).toBe(
      'Les six armes les plus meurtrières du joueur sur la soirée, en part des frags du joueur ; couleur de la barre : classe de l’arme, compte au survol.',
    )
    expect(c.emprise.control.info).toBe(
      'Prises de chaque ressource par l’équipe et par l’adversaire, en parts (comptes au survol), sur les matchs filmés de la soirée ; trait orange : 50 %. Les bonus sans ramasseur connu ne comptent dans aucune équipe.',
    )
    expect(c.emprise.grid.info).toBe(
      'Une colonne par match de la soirée ; case : part de l’équipe dans la ressource (vert au-dessus de l’adversaire, rouge en dessous, saturée à trente points d’écart), comptes et preneurs au survol.',
    )
    expect([c.emprise.grid.more, c.emprise.grid.less, c.emprise.grid.nothing, c.emprise.grid.noFilm]).toEqual([
      'Plus de 50 %',
      'Moins de 50 %',
      'Rien à prendre',
      'Sans film, non mesuré',
    ])
    expect(c.cards.mine.info).toBe(
      'Prises de l’équipe par ressource sur les matchs filmés de la soirée : part du joueur et du reste de l’équipe, en pourcentage (comptes au survol). Bonus perdus : gardés sans être activés, ou lâchés, en part des bonus pris par chaque équipe.',
    )
    expect(c.objectif.balance.info).toBe(
      'Part de l’équipe face à l’adversaire pour chaque rôle de l’objectif (somme de ses actions ; Tenir en durée), par famille de mode, comptes au survol ; trait orange : 50 %.',
    )
    expect(c.sheet.info).toBe(
      'Actions de l’objectif du joueur, rôle par rôle ; barre et nombre : part du total de l’équipe (compte au survol), un zéro reste affiché, atténué. Rôle dominant : celui où la part du joueur dans l’équipe est la plus forte.',
    )
    expect(c.sheet.pctFmt(31.8)).toBe('32 %')
    expect(c.cards.equipment.info).toBe(
      'Équipement tenu par le joueur (réapparition comprise), par famille : servi (mur posé, charge consommée), gardé sans servir, lâché, en part des objets du joueur (comptes au survol) ; barre fine : reste de l’équipe. Seules les familles tenues dans le lobby sont listées ; le répulseur, sans mesure d’usage, n’a pas de ligne.',
    )
  })

  it('D, G, H, I : mêmes aides que la pleine page ; titres inchangés', () => {
    expect(fr.compact.emprise.fil).toBe(fr.full.emprise.fil)
    expect(fr.compact.emprise.production).toBe(fr.full.emprise.production)
    expect(fr.compact.emprise.yield).toBe(fr.full.emprise.yield)
    expect(fr.compact.cards.lives).toBe(fr.full.cards.lives)
    expect(fr.compact.emprise.control.title).toBe(fr.full.emprise.control.title)
    expect(fr.compact.sheet.title).toBe(fr.full.sheet.title)
  })

  it('formateurs compacts : sous-libellés et lignes', () => {
    const k = SESSION_CARD_TEXT.fr.compactCards
    expect(k.production.exposureLine('temps d’effet', '58 %')).toBe('temps d’effet : 58 %')
    expect(k.mine.resourceSub).toBe('prises de l’équipe')
    expect(k.equipment.sub(84)).toBe('84 objets')
    expect(k.equipment.sub(0)).toBe('0 objet')
    expect(k.equipment.restUsed('48 %')).toBe('reste de l’équipe : 48 % servis')
    expect(k.lives.killsLine('90 %', '0,7')).toBe('frags : 90 % · 0,7 par vie')
    expect(k.lives.killsLineAlone('0,2')).toBe('0,2 par vie')
  })
})

describe('SESSION_CARD_TEXT — portée de la page (V6)', () => {
  it('aucune portée « du périmètre » ; ni « in scope » ni « over the scope » en anglais', () => {
    // Les dictionnaires de l'Escouade portent toute leur page (fiches de prises, habitude…) : on lit ce
    // que les cartes de Sessions écrivent.
    const ours = (v: typeof fr.full) => {
      const e = v.emprise
      return [e.control, e.fil, e.production, e.yield, e.grid, v.objectif.balance, v.cards, v.sheet, v.squad.performanceCharts.fragBreakdownInfo, v.squad.weaponKills.info]
    }
    const frStrings = [...strings(ours(fr.full)), ...strings(ours(fr.compact))].join('\n')
    expect(frStrings).not.toMatch(/(du|sur le) périmètre/i)
    const en = SESSION_CARD_TEXT.en
    const enStrings = [...strings(ours(en.full)), ...strings(ours(en.compact))].join('\n')
    expect(enStrings).not.toMatch(/(in|over the) scope|scope total/i)
  })
})
