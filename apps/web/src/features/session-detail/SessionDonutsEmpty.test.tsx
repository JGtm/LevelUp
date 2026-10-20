/**
 * Les deux anneaux de « Bilan » (issues, frags / morts / assistances) gardent leur carte quand la session
 * n'a rien à compter : titre et état vide dans le bloc, plus de trou à côté de l'anneau voisin.
 */
import { describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'

import { SessionKillsDonut } from './SessionKillsDonut'
import { SessionOutcomeDonut } from './SessionOutcomeDonut'

vi.mock('echarts-for-react', () => ({ default: () => <div data-testid="echarts-mock" /> }))

describe('anneaux de Bilan sans donnée', () => {
  it('issues : titre et état vide dans la carte', async () => {
    renderWithProviders(<SessionOutcomeDonut title="Issues des matchs" matches={[]} />)
    expect(await screen.findByText('Issues des matchs')).toBeInTheDocument()
    expect(screen.getByTestId('chart-card-empty').querySelector('.border-dashed')).not.toBeNull()
  })

  it('frags / morts / assistances : titre et état vide dans la carte', async () => {
    renderWithProviders(<SessionKillsDonut title="Frags, morts, assistances" matches={[]} kda={null} />)
    expect(await screen.findByText('Frags, morts, assistances')).toBeInTheDocument()
    expect(screen.getByTestId('chart-card-empty').querySelector('.border-dashed')).not.toBeNull()
  })
})
