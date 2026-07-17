/**
 * Tests logique pure du tri serveur Explorer (mapping colonne → cle serveur +
 * conversions sortKey ⇄ SortingState + toggle de direction). Aucun rendu :
 * ces fonctions pilotent le <select> et les en-tetes cliquables du tableau.
 */
import { describe, expect, it } from 'vitest'

import {
  EXPLORER_COLUMN_SORT_KEYS,
  EXPLORER_SORT_LABEL_KEYS,
  isSortableColumn,
  sortKeyToSorting,
  sortingToSortKey,
} from './explorerMatchesSort'

describe('explorerMatchesSort — mapping colonne → cle serveur', () => {
  it('les 5 colonnes triables mappent vers leur cle serveur backend', () => {
    expect(EXPLORER_COLUMN_SORT_KEYS).toEqual({
      start_time: 'start_time',
      perf_score: 'performance_score_relative',
      kills: 'kills',
      kda: 'kda',
      delta_mmr: 'delta_mmr',
    })
  })

  it('chaque colonne triable a un libelle aria', () => {
    for (const col of Object.keys(EXPLORER_COLUMN_SORT_KEYS)) {
      expect(EXPLORER_SORT_LABEL_KEYS[col]).toBeDefined()
    }
  })

  it('isSortableColumn : vrai pour les colonnes mappees, faux sinon', () => {
    expect(isSortableColumn('kda')).toBe(true)
    expect(isSortableColumn('perf_score')).toBe(true)
    expect(isSortableColumn('delta_mmr')).toBe(true)
    // outcome_code : le select l'expose mais le backend ne l'honore pas → non triable.
    expect(isSortableColumn('outcome_code')).toBe(false)
    expect(isSortableColumn('map_ui')).toBe(false)
    expect(isSortableColumn(undefined)).toBe(false)
    expect(isSortableColumn(null)).toBe(false)
  })
})

describe('explorerMatchesSort — sortKey → SortingState', () => {
  it('convertit champ + direction (accessorKey ≠ cle serveur inclus)', () => {
    expect(sortKeyToSorting('start_time:desc')).toEqual([{ id: 'start_time', desc: true }])
    expect(sortKeyToSorting('performance_score_relative:asc')).toEqual([
      { id: 'perf_score', desc: false },
    ])
    expect(sortKeyToSorting('kda:asc')).toEqual([{ id: 'kda', desc: false }])
  })

  it('direction absente ou non-asc = descendant', () => {
    expect(sortKeyToSorting('kills')).toEqual([{ id: 'kills', desc: true }])
    expect(sortKeyToSorting('kills:desc')).toEqual([{ id: 'kills', desc: true }])
  })

  it('colonne sans cle serveur (outcome) ou entree vide → []', () => {
    expect(sortKeyToSorting('outcome:desc')).toEqual([])
    expect(sortKeyToSorting('')).toEqual([])
    expect(sortKeyToSorting(undefined)).toEqual([])
    expect(sortKeyToSorting(null)).toEqual([])
  })
})

describe('explorerMatchesSort — SortingState → sortKey', () => {
  it('convertit vers "{champ serveur}:{dir}"', () => {
    expect(sortingToSortKey([{ id: 'perf_score', desc: true }])).toBe(
      'performance_score_relative:desc',
    )
    expect(sortingToSortKey([{ id: 'kda', desc: false }])).toBe('kda:asc')
    expect(sortingToSortKey([{ id: 'start_time', desc: true }])).toBe('start_time:desc')
  })

  it('etat vide ou colonne non serveur → null', () => {
    expect(sortingToSortKey([])).toBe(null)
    expect(sortingToSortKey([{ id: 'map_ui', desc: true }])).toBe(null)
  })
})

describe('explorerMatchesSort — toggle de direction (aller-retour)', () => {
  it('inverser desc preserve la colonne et bascule asc ⇄ desc', () => {
    const desc = sortKeyToSorting('kda:desc') // [{ id:'kda', desc:true }]
    const asc = desc.map((s) => ({ ...s, desc: !s.desc }))
    expect(sortingToSortKey(asc)).toBe('kda:asc')
    const back = asc.map((s) => ({ ...s, desc: !s.desc }))
    expect(sortingToSortKey(back)).toBe('kda:desc')
  })
})
