/**
 * TimeseriesPage.usages.test — le câblage du graphe « Prises par camp, cumul par match » sur
 * l'onglet Usages : l'axe PÉRIODE (D12), donc une légende sans encoche de dominance. `ChartCard` est
 * rendu pour de bon (seul ECharts est doublé) : sa légende de pied est celle que la page lui passe.
 */
import { describe, expect, it, vi } from 'vitest'
import { screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { TimeseriesPageResponse } from '@/lib/api/types'

import { TimeseriesUsagesTab } from './TimeseriesPage.usages'
import { MATCH_ROWS, soloEmprise } from './usages/usages.fixtures'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

describe('Onglet Usages — « cumul par match » sur une période', () => {
  it('légende : ressources, victoire / défaite, 50 % — sans encoche de dominance', () => {
    const data = { total_matches: 4, match_rows: MATCH_ROWS, emprise: soloEmprise() } as unknown as TimeseriesPageResponse
    renderWithProviders(<TimeseriesUsagesTab data={data} locale="fr" t={(k) => k} />)
    const fil = screen.getByTestId('emprise-fil')
    const legend = within(fil).getByRole('list', { name: 'Prises par camp, cumul par match' })
    expect(within(legend).getByText('Victoire, défaite')).toBeInTheDocument()
    expect(within(legend).queryByText('Drapeau de dominance')).not.toBeInTheDocument()
  })
})
