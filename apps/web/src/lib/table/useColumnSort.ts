/**
 * useColumnSort — tri CLIENT par clic sur les en-têtes d'un tableau HTML simple : la règle de bascule
 * vit ici, une fois, et TanStack Table (modèle de lignes trié) ordonne les lignes.
 *
 * Comportement, identique à celui des tableaux qui le faisaient à la main :
 *  - aucun tri actif au départ (sauf `initial`) : l'ordre reçu est gardé jusqu'au premier clic ;
 *  - premier clic sur une colonne : `firstDir` (`'desc'` par défaut, `'asc'` pour les textes) ;
 *    clic suivant sur la même colonne : bascule du sens, sans jamais revenir à « pas de tri » ;
 *  - une valeur absente (`null` / `undefined`) passe toujours EN DERNIER, dans les deux sens ;
 *  - à valeurs égales, l'ordre reçu est conservé (tri stable) ; avec `reverseTies`, un tri descendant
 *    renverse aussi l'ordre des égaux (comme un tri croissant suivi de `reverse()`) ;
 *  - par défaut deux textes se comparent en ordre naturel insensible à la casse, deux nombres par
 *    leur différence ; `compare` remplace cette règle pour une colonne ;
 *  - `resolve` donne le tri EFFECTIVEMENT appliqué à partir du tri choisi (le choix est conservé tel
 *    quel, seul l'ordre rendu change).
 *
 * `header(key)` rend de quoi poser un `SortableTh` (`active`, `dir`, `onClick`).
 */
import { useCallback, useMemo, useState } from 'react'
import { getCoreRowModel, getSortedRowModel, useReactTable, type ColumnDef } from '@tanstack/react-table'

export type SortValue = string | number | null | undefined
export type SortDirection = 'asc' | 'desc'

export interface SortColumn<T> {
  /** Valeur comparée de la ligne ; `null` / `undefined` = absente, rangée en dernier. */
  value: (row: T) => SortValue
  /** Sens du premier clic (défaut `'desc'`). */
  firstDir?: SortDirection
  /** Comparaison de deux valeurs présentes (défaut : ordre naturel des textes, différence des nombres). */
  compare?: (a: string | number, b: string | number) => number
}

export interface ColumnSortHeader {
  active: boolean
  dir: SortDirection
  onClick: () => void
}

export interface ColumnSortState<K extends string> {
  key: K
  dir: SortDirection
}

export interface ColumnSortOptions<K extends string> {
  /** Tri actif au départ (défaut : aucun). */
  initial?: ColumnSortState<K>
  /** Tri effectivement appliqué à partir du tri choisi (défaut : le tri choisi). */
  resolve?: (chosen: ColumnSortState<K> | null) => ColumnSortState<K> | null
  /** Un tri descendant renverse aussi l'ordre des valeurs égales. */
  reverseTies?: boolean
}

export interface ColumnSort<T, K extends string> {
  /** Lignes dans l'ordre courant. */
  rows: T[]
  /** Tri effectivement appliqué, ou `null` tant qu'aucun en-tête n'a été cliqué. */
  sortKey: K | null
  sortDir: SortDirection
  /** Props d'un `SortableTh` pour la colonne `key`. */
  header: (key: K) => ColumnSortHeader
}

const naturalCompare = (a: string | number, b: string | number): number =>
  typeof a === 'string' && typeof b === 'string'
    ? a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' })
    : (a as number) - (b as number)

export function useColumnSort<T, K extends string>(
  rows: readonly T[],
  columns: Readonly<Record<K, SortColumn<T>>>,
  options: ColumnSortOptions<K> = {},
): ColumnSort<T, K> {
  const { initial, resolve, reverseTies = false } = options
  const [chosen, setChosen] = useState<ColumnSortState<K> | null>(initial ?? null)
  const effective = resolve ? resolve(chosen) : chosen
  const desc = effective?.dir === 'desc'

  const defs = useMemo<ColumnDef<T, SortValue>[]>(
    () =>
      (Object.keys(columns) as K[]).map((id) => {
        const col = columns[id]
        const compare = col.compare ?? naturalCompare
        return {
          id,
          accessorFn: (row) => col.value(row) ?? undefined,
          // Les valeurs absentes sont rangées par sortingFn ci-dessous, pas par la règle par défaut de TanStack.
          sortUndefined: false,
          // TanStack inverse le résultat d'un tri descendant : une valeur absente doit rester derrière une
          // valeur présente dans les deux sens, donc on lui oppose le signe inverse.
          sortingFn: (a, b, columnId) => {
            const va = a.getValue<SortValue>(columnId)
            const vb = b.getValue<SortValue>(columnId)
            if (va == null && vb == null) return 0
            if (va == null) return desc ? -1 : 1
            if (vb == null) return desc ? 1 : -1
            const cmp = compare(va, vb)
            return cmp === 0 && reverseTies ? a.index - b.index : cmp
          },
        }
      }),
    [columns, desc, reverseTies],
  )

  // eslint-disable-next-line react-hooks/incompatible-library -- l'API de TanStack Table n'est pas mémoïsable ; le hook en confine l'usage
  const table = useReactTable({
    data: rows as T[],
    columns: defs,
    state: { sorting: effective ? [{ id: effective.key, desc }] : [] },
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    manualSorting: false,
    enableMultiSort: false,
  })

  const toggle = useCallback(
    (key: K) => {
      setChosen((prev) =>
        prev?.key === key
          ? { key, dir: prev.dir === 'asc' ? 'desc' : 'asc' }
          : { key, dir: columns[key].firstDir ?? 'desc' },
      )
    },
    [columns],
  )

  return {
    rows: table.getRowModel().rows.map((r) => r.original),
    sortKey: effective?.key ?? null,
    sortDir: effective?.dir ?? 'desc',
    header: (key) => ({
      active: effective?.key === key,
      dir: effective?.dir ?? 'desc',
      onClick: () => toggle(key),
    }),
  }
}
