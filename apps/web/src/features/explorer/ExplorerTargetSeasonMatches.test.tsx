/**
 * Tests ExplorerTargetSeasonMatches — barres horizontales matchs par saison.
 */
import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { SeasonMatchCount } from '@/lib/api/types'

import { ExplorerTargetSeasonMatches } from './ExplorerTargetSeasonMatches'

describe('ExplorerTargetSeasonMatches', () => {
  it('rend une ligne par saison avec libellé + compteur', () => {
    const seasons: SeasonMatchCount[] = [
      { season_id: 'season12', season_name: 'S12', matches: 63 },
      { season_id: 'season13', season_name: 'S13', matches: 4 },
    ]
    renderWithProviders(<ExplorerTargetSeasonMatches seasons={seasons} title="Matchs par saison" />)
    expect(screen.getByTestId('explorer-target-season-matches')).toBeInTheDocument()
    expect(screen.getByText('S12')).toBeInTheDocument()
    expect(screen.getByText('63')).toBeInTheDocument()
    expect(screen.getByText('S13')).toBeInTheDocument()
    expect(screen.getByText('4')).toBeInTheDocument()
  })

  it('ne rend rien si liste vide', () => {
    const { container } = renderWithProviders(
      <ExplorerTargetSeasonMatches seasons={[]} title="Matchs par saison" />,
    )
    expect(container.querySelector('[data-testid="explorer-target-season-matches"]')).toBeNull()
  })
})
