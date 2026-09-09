/**
 * Heatmap2DChart — rendu du composant (pas seulement l'option ECharts pure,
 * couverte par Heatmap2DChart.test.ts) : la légende « Aucune mesure sur cet axe »
 * (décision D3, plan vague C formes 2026-09-08) n'apparaît QUE si la série
 * contient au moins une case `value: null`, et respecte la parité FR/EN.
 *
 * echarts-for-react est mocké (comme FirstBloodLanes.test.tsx, OutcomeSequenceTape)
 * pour éviter de payer le rendu canvas réel — seul le pied de card (`ChartCard`
 * prop `legend`) nous intéresse ici.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'

import { Heatmap2DChart, type ChartPointHeatmap } from './Heatmap2DChart'
import type { ChartSeries } from './ChartCard'

vi.mock('@/lib/accessibility', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/accessibility')>()),
  resolveToken: (token: string) => `tok:${token}`,
}))

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="heatmap-echarts-stub" />,
}))

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

const SERIES_AVEC_CASE_VIDE: ChartSeries<ChartPointHeatmap>[] = [
  {
    key: 'matrice',
    datapoints: [
      { x: 'A', y: 'A', value: null },
      { x: 'B', y: 'A', value: 4 },
    ],
  },
]

const SERIES_SANS_CASE_VIDE: ChartSeries<ChartPointHeatmap>[] = [
  {
    key: 'heatmap-map',
    datapoints: [{ x: 'Aquarius', y: 'main', value: 75 }],
  },
]

describe('Heatmap2DChart — légende de la case vide (D3)', () => {
  it('affiche la légende FR quand une case est vide', async () => {
    render(<Heatmap2DChart series={SERIES_AVEC_CASE_VIDE} />)
    await screen.findByTestId('heatmap-echarts-stub')
    expect(screen.getByTestId('heatmap-empty-cell-legend').textContent).toBe(
      'Aucune mesure sur cet axe',
    )
  })

  it('affiche la légende EN quand la locale est en', async () => {
    useAppShellStore.setState({ locale: 'en' })
    render(<Heatmap2DChart series={SERIES_AVEC_CASE_VIDE} />)
    await screen.findByTestId('heatmap-echarts-stub')
    expect(screen.getByTestId('heatmap-empty-cell-legend').textContent).toBe(
      'No measurement on this axis',
    )
  })

  it('NE l’affiche PAS quand aucune case n’est vide (les consommateurs sans case vide héritent sans rien voir de nouveau)', async () => {
    render(<Heatmap2DChart series={SERIES_SANS_CASE_VIDE} />)
    await screen.findByTestId('heatmap-echarts-stub')
    expect(screen.queryByTestId('heatmap-empty-cell-legend')).toBeNull()
    expect(screen.queryByTestId('chart-card-legend')).toBeNull()
  })
})
