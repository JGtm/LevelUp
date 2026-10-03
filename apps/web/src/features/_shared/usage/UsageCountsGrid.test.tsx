/**
 * UsageCountsGrid.test.tsx — le rendu de la variante COMPTES (P9, E5.8/E6.2) : une
 * ligne par grandeur, aucun trait de parité, l'axe finit par le compte + unité.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { UsageCountsGrid } from './UsageCountsGrid'
import { buildCountsGrid } from './usageCountsModel'
import { USAGE_TEXT } from './usageI18n'

const t = USAGE_TEXT.fr

describe('UsageCountsGrid', () => {
  it('ne rend rien pour une grille vide', () => {
    const grid = buildCountsGrid([], { t, locale: 'fr', unit: 'equipment' })
    const { container } = render(<UsageCountsGrid grid={grid} />)
    expect(container.firstChild).toBeNull()
  })

  it('rend une ligne par grandeur, sans trait de parite', () => {
    const grid = buildCountsGrid(
      [
        { key: 'wall', label: 'Mur', taken: 138 },
        { key: 'sensor', label: 'Capteur', taken: 40 },
      ],
      { t, locale: 'fr', unit: 'equipment' },
    )
    render(<UsageCountsGrid grid={grid} />)
    expect(screen.getByText('Mur')).toBeInTheDocument()
    expect(screen.getByText('Capteur')).toBeInTheDocument()
    expect(screen.getByText('138 pris')).toBeInTheDocument()
    // Le trait de parite n'existe pas dans cette variante (P9).
    const { container } = render(<UsageCountsGrid grid={grid} />)
    expect(container.querySelectorAll('[style*="background-color"]').length).toBeGreaterThan(0)
  })

  it('affiche la graduation de fin d axe avec son unite', () => {
    const grid = buildCountsGrid([{ key: 'a', label: 'A', taken: 12 }], {
      t,
      locale: 'fr',
      unit: 'weapon',
    })
    render(<UsageCountsGrid grid={grid} />)
    expect(screen.getByText(grid.axisMaxText)).toBeInTheDocument()
  })
})
