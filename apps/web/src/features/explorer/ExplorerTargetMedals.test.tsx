/**
 * Tests ExplorerTargetMedals — top 5 + expander jusqu'à 20.
 */
import { describe, expect, it } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { MedalDigestItem } from '@/lib/api/types'

import { ExplorerTargetMedals } from './ExplorerTargetMedals'

function makeMedals(n: number): MedalDigestItem[] {
  return Array.from({ length: n }, (_, i) => ({
    medal_id: 1000 + i,
    label: `Medal ${i}`,
    description: `Desc ${i}`,
    image_url: `/static/medals/halo_infinite/${1000 + i}.png`,
    total_count: 100 - i, // décroissant
    match_count: 0,
  }))
}

describe('ExplorerTargetMedals', () => {
  it('affiche le top 18 et un expander quand plus de 18 médailles', () => {
    renderWithProviders(<ExplorerTargetMedals medals={makeMedals(20)} />)
    // Top 18 affichés (index 0..17)
    expect(screen.getByText('Medal 0')).toBeInTheDocument()
    expect(screen.getByText('Medal 17')).toBeInTheDocument()
    // La 19e (index 18) est masquée par défaut
    expect(screen.queryByText('Medal 18')).not.toBeInTheDocument()
    // Compteur visible (×100)
    expect(screen.getByText('×100')).toBeInTheDocument()
  })

  it('expander révèle toutes les médailles puis se replie', () => {
    renderWithProviders(<ExplorerTargetMedals medals={makeMedals(20)} />)
    const moreBtn = screen.getByText(/Voir plus|Show more/i)
    fireEvent.click(moreBtn)
    expect(screen.getByText('Medal 19')).toBeInTheDocument()
    fireEvent.click(screen.getByText(/Voir moins|Show less/i))
    expect(screen.queryByText('Medal 19')).not.toBeInTheDocument()
  })

  it('pas d\'expander si <= 18 médailles', () => {
    renderWithProviders(<ExplorerTargetMedals medals={makeMedals(3)} />)
    expect(screen.queryByText(/Voir plus|Show more/i)).not.toBeInTheDocument()
    expect(screen.getByText('Medal 2')).toBeInTheDocument()
  })

  it('ne rend rien si liste vide', () => {
    const { container } = renderWithProviders(<ExplorerTargetMedals medals={[]} />)
    expect(container.querySelector('[data-testid="explorer-target-medals"]')).toBeNull()
  })
})
