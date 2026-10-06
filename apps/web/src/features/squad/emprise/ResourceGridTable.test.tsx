/**
 * ResourceGridTable.test.tsx — la table partagée de la grille « Contrôle des ressources » : une
 * colonne par en-tête reçu, et l'infobulle de chaque case ouverte par l'en-tête de SA colonne (la
 * grille par carte des Séries temporelles monte la même table avec des cartes pour colonnes).
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { GridSection } from './emprise.logic'
import { EMPRISE_TEXT } from './empriseStrings'
import { ResourceGridTable, type GridColumn } from './ResourceGridTable'

const T = EMPRISE_TEXT.fr

const columns: GridColumn[] = [
  { key: 'aquarius', head: <div data-testid="head-aquarius">Aquarius</div>, tipHead: 'Aquarius · 12 matchs' },
  { key: 'recharge', head: <div data-testid="head-recharge">Recharge</div>, tipHead: 'Recharge · 9 matchs' },
]

const sections: GridSection[] = [
  {
    resource: 'powerup',
    summary: {
      cells: [
        { kind: 'value', us: 5, them: 2, share: 5 / 7, who: [{ xuid: 'x1', taken: 5 }] },
        { kind: 'value', us: 1, them: 3, share: 1 / 4, who: [] },
      ],
    },
    items: [],
    kills: null,
  },
]

function renderTable() {
  render(
    <ResourceGridTable columns={columns} sections={sections} itemName={() => ''} whoText={(w) => `Moi ${w[0].taken}`} t={T} />,
  )
  return Array.from(screen.getByTestId('emprise-grid-table').querySelectorAll('[data-cell="value"]'))
}

describe('ResourceGridTable', () => {
  it('un en-tête par colonne reçue, dans l’ordre', () => {
    renderTable()
    expect(screen.getByTestId('head-aquarius')).toBeTruthy()
    expect(screen.getByTestId('head-recharge')).toBeTruthy()
  })

  it('l’infobulle d’une case s’ouvre sur l’en-tête de SA colonne', () => {
    const cells = renderTable()
    expect(cells.map((c) => c.textContent)).toEqual(['5–2', '1–3'])
    fireEvent.mouseEnter(cells[1].parentElement!)
    const tip = screen.getByRole('tooltip').textContent ?? ''
    expect(tip).toContain('Recharge · 9 matchs')
    expect(tip).not.toContain('Aquarius')
  })

  it('« qui chez nous » dans l’infobulle d’une case prise par l’escouade', () => {
    const cells = renderTable()
    fireEvent.mouseEnter(cells[0].parentElement!)
    const tip = screen.getByRole('tooltip').textContent ?? ''
    expect(tip).toContain('Aquarius · 12 matchs')
    expect(tip).toContain('Moi 5')
  })
})
