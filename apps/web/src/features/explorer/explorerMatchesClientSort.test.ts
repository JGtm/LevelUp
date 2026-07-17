/**
 * Tests des helpers purs de tri CLIENT du tableau Explorer (mode Matchs).
 * Ces fonctions pilotent le tri par valeur sous-jacente des colonnes ; le tri
 * integre (getSortedRowModel + pagination) est couvert par ExplorerMatchesTable.test.tsx.
 */
import { describe, expect, it } from 'vitest'
import type { Row } from '@tanstack/react-table'

import type { ExplorerMatchRow } from '@/lib/api/types'
import {
  NUMERIC_SORT,
  SORT_ARIA_LABEL_KEYS,
  dateTimeSortingFn,
  localeTextSortingFn,
} from './explorerMatchesClientSort'

// Fausse Row : nos sortingFns ne consultent que getValue(id).
const row = (v: unknown): Row<ExplorerMatchRow> =>
  ({ getValue: () => v }) as unknown as Row<ExplorerMatchRow>
const ID = 'x'

describe('NUMERIC_SORT', () => {
  it('range les valeurs undefined en bas (dans les deux sens) via basic', () => {
    expect(NUMERIC_SORT).toEqual({ sortingFn: 'basic', sortUndefined: 'last' })
  })
})

describe('dateTimeSortingFn', () => {
  it('ordonne sur le timestamp brut (le plus récent en dernier en ascendant)', () => {
    expect(
      dateTimeSortingFn(row('2026-05-01T00:00:00Z'), row('2026-04-01T00:00:00Z'), ID),
    ).toBeGreaterThan(0)
    expect(
      dateTimeSortingFn(row('2026-04-01T00:00:00Z'), row('2026-05-01T00:00:00Z'), ID),
    ).toBeLessThan(0)
  })

  it('timestamps égaux → 0', () => {
    expect(dateTimeSortingFn(row('2026-05-01T00:00:00Z'), row('2026-05-01T00:00:00Z'), ID)).toBe(0)
  })

  it('date invalide → rangée après une date valide', () => {
    expect(dateTimeSortingFn(row('pas-une-date'), row('2026-01-01T00:00:00Z'), ID)).toBe(1)
    expect(dateTimeSortingFn(row('2026-01-01T00:00:00Z'), row('pas-une-date'), ID)).toBe(-1)
  })
})

describe('localeTextSortingFn', () => {
  it('tri alphabétique (Alpha < Zeta)', () => {
    expect(localeTextSortingFn(row('Alpha'), row('Zeta'), ID)).toBeLessThan(0)
    expect(localeTextSortingFn(row('Zeta'), row('Alpha'), ID)).toBeGreaterThan(0)
  })

  it('numérique naturel (Map2 < Map10)', () => {
    expect(localeTextSortingFn(row('Map2'), row('Map10'), ID)).toBeLessThan(0)
  })

  it('insensible à la casse (sensitivity base)', () => {
    expect(localeTextSortingFn(row('alpha'), row('ALPHA'), ID)).toBe(0)
  })

  it('valeur absente coalescée en chaîne vide (pas de crash)', () => {
    expect(localeTextSortingFn(row(undefined), row('Alpha'), ID)).toBeLessThan(0)
  })
})

describe('SORT_ARIA_LABEL_KEYS', () => {
  it('mappe les colonnes de données vers un libellé texte-plein', () => {
    // Colonne numérique : libellé long (pas l'abréviation « F »).
    expect(SORT_ARIA_LABEL_KEYS.kills).toBe('explorer.matches.col_kills_long')
    // Colonne dérivée : libellé de l'en-tête, pas la valeur brute.
    expect(SORT_ARIA_LABEL_KEYS.outcome_code).toBe('explorer.matches.col_outcome')
    expect(SORT_ARIA_LABEL_KEYS.skill_tier_label).toBe('explorer.matches.col_rank')
    expect(SORT_ARIA_LABEL_KEYS.start_time).toBe('explorer.matches.col_date')
  })

  it('la colonne d’ouverture (non triable) n’a pas de libellé de tri', () => {
    expect(SORT_ARIA_LABEL_KEYS.open).toBeUndefined()
  })
})
