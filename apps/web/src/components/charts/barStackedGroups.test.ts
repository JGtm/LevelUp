/**
 * Tests — barStackedGroups (le trait entre deux groupes de colonnes, et leurs titres).
 *
 * CE QU'ILS PROTÈGENT : le trait tombe ENTRE deux colonnes (index fractionnaire), il n'existe
 * pas quand il n'y a qu'un groupe, et chaque titre est centré dans SA part de la bande — pas
 * au milieu du graphe.
 */
import { describe, expect, it } from 'vitest'

import { groupBoundaries, groupSeparatorMarkLine, groupTitleGraphic } from './barStackedGroups'

const GROUPES = [
  { label: 'Puissance · 24 prises', span: 3 },
  { label: 'Terrain · 19 prises', span: 2 },
]

describe('groupBoundaries', () => {
  it('pose une frontière entre les groupes, à l’index fractionnaire de la saignée', () => {
    expect(groupBoundaries(GROUPES)).toEqual([2.5])
    expect(groupBoundaries([{ label: 'a', span: 2 }, { label: 'b', span: 2 }, { label: 'c', span: 1 }])).toEqual([1.5, 3.5])
  })

  it('un seul groupe = aucune frontière : un trait au bord ne sépare rien', () => {
    expect(groupBoundaries([{ label: 'a', span: 4 }])).toEqual([])
  })
})

describe('groupSeparatorMarkLine', () => {
  it('rend un trait muet, sans symbole ni étiquette, à l’encre reçue', () => {
    const ml = groupSeparatorMarkLine(GROUPES, 'var(--border)')
    expect(ml?.data).toEqual([{ xAxis: 2.5 }])
    expect(ml?.silent).toBe(true)
    expect(ml?.label.show).toBe(false)
    expect(ml?.lineStyle.color).toBe('var(--border)')
  })

  it('pas de markLine avec un seul groupe', () => {
    expect(groupSeparatorMarkLine([{ label: 'a', span: 2 }], 'x')).toBeUndefined()
  })
})

describe('groupTitleGraphic', () => {
  it('centre chaque titre dans SA part de la bande de tracé', () => {
    const g = groupTitleGraphic(GROUPES, 'var(--muted-foreground)')
    expect(g?.type).toBe('group')
    expect(g?.children.map((c) => c.style.text)).toEqual([
      'Puissance · 24 prises',
      'Terrain · 19 prises',
    ])
    const x = g?.children.map((c) => Number.parseFloat(c.left)) ?? []
    // Le premier groupe couvre 3 colonnes sur 5 : son centre est à gauche du second.
    expect(x[0]).toBeLessThan(x[1])
    expect(x[0]).toBeGreaterThan(10)
    expect(x[1]).toBeLessThan(97)
  })

  it('aucun titre nommé = aucun graphic (l’appelant peut ne vouloir que le trait)', () => {
    expect(groupTitleGraphic([{ label: '', span: 2 }], 'x')).toBeUndefined()
    expect(groupTitleGraphic([], 'x')).toBeUndefined()
  })
})
