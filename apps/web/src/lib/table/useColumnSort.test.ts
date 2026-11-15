import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { useColumnSort, type SortColumn } from './useColumnSort'

interface Row {
  name: string
  score: number | null
  tag?: string
}

const ROWS: Row[] = [
  { name: 'b', score: 5, tag: 'x10' },
  { name: 'a', score: null, tag: 'x2' },
  { name: 'c', score: 9 },
  { name: 'A', score: 5, tag: 'x1' },
  { name: 'd', score: null },
]

const COLUMNS: Record<'name' | 'score' | 'tag', SortColumn<Row>> = {
  name: { value: (r) => r.name, firstDir: 'asc' },
  score: { value: (r) => r.score },
  tag: { value: (r) => r.tag, firstDir: 'asc' },
}

function setup() {
  return renderHook(() => useColumnSort(ROWS, COLUMNS))
}
const names = (rows: Row[]): string => rows.map((r) => r.name).join('')

describe('useColumnSort', () => {
  it('garde l’ordre reçu tant qu’aucun en-tête n’est cliqué', () => {
    const { result } = setup()
    expect(names(result.current.rows)).toBe('bacAd')
    expect(result.current.sortKey).toBeNull()
    expect(result.current.header('score').active).toBe(false)
  })

  it('premier clic : sens propre à la colonne ; second clic : bascule, sans retour à « pas de tri »', () => {
    const { result } = setup()
    act(() => result.current.header('score').onClick())
    expect(result.current.sortKey).toBe('score')
    expect(result.current.sortDir).toBe('desc')
    expect(names(result.current.rows)).toBe('cbAad')
    act(() => result.current.header('score').onClick())
    expect(result.current.sortDir).toBe('asc')
    expect(names(result.current.rows)).toBe('bAcad')
    act(() => result.current.header('score').onClick())
    expect(result.current.sortDir).toBe('desc')
    expect(result.current.sortKey).toBe('score')
  })

  it('une colonne texte commence en ordre croissant, insensible à la casse, à égalité l’ordre reçu est gardé', () => {
    const { result } = setup()
    act(() => result.current.header('name').onClick())
    expect(result.current.sortDir).toBe('asc')
    // 'a' (index 1) et 'A' (index 3) sont égaux en comparaison insensible à la casse.
    expect(names(result.current.rows)).toBe('aAbcd')
    act(() => result.current.header('name').onClick())
    expect(names(result.current.rows)).toBe('dcbaA')
  })

  it('les valeurs absentes restent en dernier dans les deux sens', () => {
    const { result } = setup()
    act(() => result.current.header('score').onClick())
    expect(result.current.rows.slice(-2).map((r) => r.score)).toEqual([null, null])
    act(() => result.current.header('score').onClick())
    expect(result.current.rows.slice(-2).map((r) => r.score)).toEqual([null, null])
    act(() => result.current.header('tag').onClick())
    expect(result.current.rows.at(-1)?.tag).toBeUndefined()
    expect(names(result.current.rows)).toBe('Aabcd')
  })

  it('l’ordre naturel range x2 avant x10', () => {
    const { result } = setup()
    act(() => result.current.header('tag').onClick())
    expect(result.current.rows.map((r) => r.tag)).toEqual(['x1', 'x2', 'x10', undefined, undefined])
  })

  it('changer de colonne repart du sens de la nouvelle colonne', () => {
    const { result } = setup()
    act(() => result.current.header('name').onClick())
    act(() => result.current.header('name').onClick())
    expect(result.current.sortDir).toBe('desc')
    act(() => result.current.header('score').onClick())
    expect(result.current.sortKey).toBe('score')
    expect(result.current.sortDir).toBe('desc')
    expect(result.current.header('name').active).toBe(false)
  })

  it('accepte une comparaison propre à une colonne', () => {
    const { result } = renderHook(() =>
      useColumnSort(ROWS, {
        name: { value: (r: Row) => r.name, firstDir: 'asc', compare: (a, b) => String(b).localeCompare(String(a)) },
      }),
    )
    act(() => result.current.header('name').onClick())
    expect(names(result.current.rows)).toBe('dcbAa')
  })
})
