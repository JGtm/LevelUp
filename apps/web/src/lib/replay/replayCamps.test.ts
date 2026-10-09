/**
 * Tests — replayCamps (les camps du rejeu : le FILM décide, la FEUILLE nomme).
 *
 * CE QU'ILS TIENNENT, décision du 2026-10-06 (« une section sans équipe n'existe pas ») :
 *   - un camp par DÉSIGNATEUR du film, dans l'ordre des désignateurs ;
 *   - un élément sans désignateur n'entre dans AUCUN camp, et n'en crée aucun ;
 *   - le côté de feuille ne fait que NOMMER — et c'est celui de la majorité des membres ;
 *   - le nom d'un camp n'est jamais « sans équipe » : son plancher est « Équipe N » du
 *     désignateur.
 */
import { describe, expect, it } from 'vitest'

import { campLabel, campSideOf, groupByCamp, type CampSheetRow } from './replayCamps'

const FR = { teamLabelFmt: (n: string) => `Équipe ${n}`, teamNumberedFmt: (n: number) => `Équipe ${n}` }

/** Un élément de test : un nom, un désignateur (ou aucun), une ligne de feuille (ou aucune). */
interface Item {
  nom: string
  team?: number
  row?: CampSheetRow
}

const ligne = (side: string | null, team_name: string | null = null): CampSheetRow => ({
  team_side: side,
  team_name,
})

function camps(items: Item[]) {
  return groupByCamp(items, (i) => i.team, (i) => [i.row])
}

describe('groupByCamp — le film décide', () => {
  it('un camp par désignateur, dans l’ordre des désignateurs, l’ordre d’entrée gardé dedans', () => {
    const groupes = camps([
      { nom: 'B1', team: 1 },
      { nom: 'A1', team: 0 },
      { nom: 'B2', team: 1 },
      { nom: 'A2', team: 0 },
    ])
    expect(groupes.map((g) => [g.team, g.members.map((m) => m.nom)])).toEqual([
      [0, ['A1', 'A2']],
      [1, ['B1', 'B2']],
    ])
  })

  it('un élément SANS désignateur n’entre dans aucun camp, et n’en crée aucun', () => {
    const groupes = camps([
      { nom: 'A', team: 0, row: ligne('t0') },
      { nom: 'Muet', row: ligne('t1') }, // la feuille le connaît : elle ne le range pas pour autant
      { nom: 'B', team: 1, row: ligne('t1') },
    ])
    expect(groupes).toHaveLength(2)
    expect(groupes.flatMap((g) => g.members.map((m) => m.nom))).toEqual(['A', 'B'])
  })

  it('« aucune équipe » (-1, mode sans camps) est une LECTURE du film : elle regroupe comme une autre', () => {
    expect(camps([{ nom: 'X', team: -1 }, { nom: 'Y', team: -1 }]).map((g) => g.team)).toEqual([-1])
  })

  it('un membre SANS ligne de feuille reste dans le camp que le film lui donne', () => {
    const [camp] = camps([{ nom: 'Humain', team: 0, row: ligne('t0') }, { nom: 'Bot', team: 0 }])
    expect(camp.members.map((m) => m.nom)).toEqual(['Humain', 'Bot'])
    expect(camp.side).toBe('t0')
  })

  it('sans aucune ligne de feuille, les camps du film restent — seul leur côté est inconnu', () => {
    const groupes = camps([{ nom: 'A', team: 0 }, { nom: 'B', team: 1 }])
    expect(groupes.map((g) => [g.team, g.side])).toEqual([
      [0, null],
      [1, null],
    ])
  })
})

describe('campSideOf — le côté qui nomme, celui de la majorité', () => {
  it('la majorité gagne : un membre que la feuille contredit ne renomme pas son camp', () => {
    expect(campSideOf([ligne('t1'), ligne('t0'), ligne('t0')])).toBe('t0')
  })

  it('à égalité, le côté qui a atteint ce compte le premier', () => {
    expect(campSideOf([ligne('t1'), ligne('t0')])).toBe('t1')
    expect(campSideOf([ligne('t0'), ligne('t1'), ligne('t1'), ligne('t0')])).toBe('t1')
  })

  it('ni ligne, ni côté, ni chaîne vide : `null`', () => {
    expect(campSideOf([])).toBeNull()
    expect(campSideOf([undefined, ligne(null), ligne('')])).toBeNull()
  })
})

describe('campLabel — le nom d’un camp, jamais « sans équipe »', () => {
  it('le côté de feuille nomme : `t0` -> « Équipe Eagle »', () => {
    expect(campLabel({ team: 0, side: 't0' }, [ligne('t0')], FR)).toBe('Équipe Eagle')
  })

  it('le `team_name` publié par le backend prime, lu sur les lignes DU côté du camp seulement', () => {
    const feuille = [ligne('t1', 'Bleu'), ligne('t0', 'Rouge')]
    expect(campLabel({ team: 0, side: 't0' }, feuille, FR)).toBe('Équipe Rouge')
    expect(campLabel({ team: 1, side: 't1' }, feuille, FR)).toBe('Équipe Bleu')
  })

  it('aucune ligne de feuille : le PLANCHER est « Équipe N » du désignateur', () => {
    expect(campLabel({ team: 1, side: null }, [], FR)).toBe('Équipe 1')
    expect(campLabel({ team: 0, side: null }, [undefined], FR)).toBe('Équipe 0')
  })

  it('un côté illisible ne se traduit pas en camp inventé : plancher du désignateur', () => {
    expect(campLabel({ team: 1, side: 'bizarre' }, [ligne('bizarre')], FR)).toBe('Équipe 1')
  })
})
