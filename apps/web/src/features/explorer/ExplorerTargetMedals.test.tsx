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
  it('affiche le top 5 et un expander quand plus de 5 médailles', () => {
    renderWithProviders(<ExplorerTargetMedals medals={makeMedals(8)} />)
    // Top 5 affichés
    expect(screen.getByText('Medal 0')).toBeInTheDocument()
    expect(screen.getByText('Medal 4')).toBeInTheDocument()
    // La 6e (index 5) est masquée par défaut
    expect(screen.queryByText('Medal 5')).not.toBeInTheDocument()
    // Compteur visible (×100)
    expect(screen.getByText('×100')).toBeInTheDocument()
  })

  it('expander révèle toutes les médailles puis se replie', () => {
    renderWithProviders(<ExplorerTargetMedals medals={makeMedals(8)} />)
    const moreBtn = screen.getByText(/Voir plus|Show more/i)
    fireEvent.click(moreBtn)
    expect(screen.getByText('Medal 7')).toBeInTheDocument()
    fireEvent.click(screen.getByText(/Voir moins|Show less/i))
    expect(screen.queryByText('Medal 7')).not.toBeInTheDocument()
  })

  it('pas d\'expander si <= 5 médailles', () => {
    renderWithProviders(<ExplorerTargetMedals medals={makeMedals(3)} />)
    expect(screen.queryByText(/Voir plus|Show more/i)).not.toBeInTheDocument()
    expect(screen.getByText('Medal 2')).toBeInTheDocument()
  })

  it('ne rend rien si liste vide', () => {
    const { container } = renderWithProviders(<ExplorerTargetMedals medals={[]} />)
    expect(container.querySelector('[data-testid="explorer-target-medals"]')).toBeNull()
  })
})
