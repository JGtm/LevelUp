/**
 * Les graphes de la Synthèse tirés de la feuille de match gardent leur carte (titre + état vide dans le
 * bloc) quand la sélection ne leur donne rien : la page les monte toujours, plus de trou sans explication.
 */
import { describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'

import { SynthesisOutcomesByGroupChart } from './SynthesisOutcomesByGroupChart'
import { SynthesisTopWeeksChart } from './SynthesisTopWeeksChart'

vi.mock('echarts-for-react', () => ({ default: () => <div data-testid="echarts-mock" /> }))

describe('Synthèse — cartes sans donnée', () => {
  it('meilleures semaines : titre et état vide dans la carte', async () => {
    renderWithProviders(<SynthesisTopWeeksChart title="Matchs Top vs Total par semaine" weeks={[]} />)
    expect(await screen.findByText('Matchs Top vs Total par semaine')).toBeInTheDocument()
    expect(screen.getByTestId('chart-card-empty').querySelector('.border-dashed')).not.toBeNull()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })

  it('résultats par carte : titre et état vide dans la carte', async () => {
    renderWithProviders(<SynthesisOutcomesByGroupChart title="Par carte" entries={[]} />)
    expect(await screen.findByText('Par carte')).toBeInTheDocument()
    expect(screen.getByTestId('chart-card-empty').querySelector('.border-dashed')).not.toBeNull()
  })
})
