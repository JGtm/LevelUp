/**
 * Tests — padControlColumns (le pivot lignes -> colonnes du contrôle des armes spéciales).
 *
 * CE QU'ILS PROTÈGENT, et ce sont les promesses de la forme 4.A (D18) :
 *   1. L'ORDRE DES COLONNES est celui des groupes passés — puissance PUIS terrain (D2) — et
 *      l'ordre interne d'un groupe n'est jamais rejoué.
 *   2. LES GROUPES portent leur `span` : c'est lui qui place le trait de séparation et centre
 *      les titres. Un groupe vide ne produit ni colonne ni titre.
 *   3. LA HAUTEUR D'UNE COLONNE est le nombre de prises NOMMÉES : les totaux ne comptent
 *      jamais ce qui n'a pas de ramasseur, qui s'annonce en note sous la colonne.
 *   4. UN JOUEUR GARDE SA SOUS-CLÉ d'une colonne à l'autre : c'est la seule façon de le suivre
 *      à l'œil, et la condition pour que la pile ECharts l'empile au même endroit.
 */
import { describe, expect, it } from 'vitest'

import { buildPadColumns } from './padControlColumns'
import type { PadBarRow, PadBarSegment } from './padControlChart'

function seg(over: Partial<PadBarSegment>): PadBarSegment {
  return {
    xuid: 'a1',
    name: 'Alpha',
    side: 't0',
    sideLabel: 'Eagle',
    count: 1,
    tint: 100,
    color: 'var(--team-ally)',
    fraction: 1,
    startsSide: false,
    ...over,
  }
}

function row(label: string, segments: PadBarSegment[], unnamed = 0): PadBarRow {
  return {
    weapon: label,
    label,
    total: segments.reduce((s, x) => s + x.count, 0),
    segments,
    unnamed,
  }
}

const unnamedFmt = (n: number) => `+ ${n} sans nom`

describe('buildPadColumns', () => {
  const puissance = [
    row('Sniper', [seg({ count: 5 }), seg({ xuid: 'b1', name: 'Charlie', side: 't1', count: 3 })], 4),
    row('Épée', [seg({ xuid: 'b1', name: 'Charlie', side: 't1', count: 2 })]),
  ]
  const terrain = [row('Hydra', [seg({ count: 1 })])]

  const model = buildPadColumns({
    groups: [
      { label: 'Puissance · 10 prises', rows: puissance },
      { label: 'Terrain · 1 prise', rows: terrain },
    ],
    unnamedFmt,
  })

  it('range les colonnes dans l’ordre des groupes : puissance puis terrain', () => {
    expect(model.datapoints.map((d) => d.category)).toEqual(['Sniper', 'Épée', 'Hydra'])
  })

  it('donne à chaque groupe son titre et sa largeur — de quoi placer trait et titres', () => {
    expect(model.groups).toEqual([
      { label: 'Puissance · 10 prises', span: 2 },
      { label: 'Terrain · 1 prise', span: 1 },
    ])
  })

  it('la hauteur d’une colonne est le total NOMMÉ, et le manque part en note', () => {
    expect(model.totals).toEqual([8, 2, 1])
    expect(model.notes).toEqual({ Sniper: '+ 4 sans nom' })
  })

  it('un joueur garde la MÊME sous-clé d’une colonne à l’autre', () => {
    expect(model.componentOrder).toEqual(['Alpha', 'Charlie'])
    expect(model.datapoints[0].components).toEqual({ Alpha: 5, Charlie: 3 })
    expect(model.datapoints[1].components).toEqual({ Charlie: 2 })
    expect(model.datapoints[2].components).toEqual({ Alpha: 1 })
  })

  it('porte le camp, l’éclaircissement et l’encre DOM de chaque joueur', () => {
    expect(model.players[0]).toMatchObject({ xuid: 'a1', side: 't0', tint: 100 })
    expect(model.players[0].cssColor).toBe('var(--team-ally)')
  })

  it('un groupe SANS ligne ne produit ni colonne ni titre', () => {
    const vide = buildPadColumns({
      groups: [
        { label: 'Puissance', rows: [] },
        { label: 'Terrain · 1 prise', rows: terrain },
      ],
      unnamedFmt,
    })
    expect(vide.groups).toEqual([{ label: 'Terrain · 1 prise', span: 1 }])
    expect(vide.datapoints).toHaveLength(1)
  })

  it('aucun groupe peuplé = aucune colonne : l’appelant y lit l’état vide', () => {
    const vide = buildPadColumns({ groups: [{ label: 'Puissance', rows: [] }], unnamedFmt })
    expect(vide.datapoints).toHaveLength(0)
    expect(vide.groups).toHaveLength(0)
  })

  it('coupe un nom de socle trop long, et garde deux coupures DISTINCTES', () => {
    const long = buildPadColumns({
      groups: [
        {
          label: '',
          rows: [
            row('Fusil de précision longue portée', [seg({})]),
            row('Fusil de précision courte portée', [seg({})]),
          ],
        },
      ],
      unnamedFmt,
    })
    const [a, b] = long.datapoints.map((d) => d.category)
    expect(a).toBe('Fusil de préci…')
    expect(a).not.toBe(b)
  })
})
