/**
 * Heatmap2DChart — rendu du composant (pas seulement l'option ECharts pure,
 * couverte par Heatmap2DChart.test.ts) : AUCUNE légende ne nomme une case vide
 * (`value: null`), quel que soit le mode — aucun inconnu n'est écrit à l'écran ; la
 * légende posée par l'appelant passe telle quelle.
 *
 * echarts-for-react est mocké (comme FirstBloodLanes.test.tsx, OutcomeSequenceTape)
 * pour éviter de payer le rendu canvas réel — seul le pied de card (`ChartCard`
 * prop `legend`) nous intéresse ici.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { Heatmap2DChart, type ChartPointHeatmap } from './Heatmap2DChart'
import type { ChartSeries } from './ChartCard'

vi.mock('@/lib/accessibility', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/accessibility')>()),
  resolveToken: (token: string) => `tok:${token}`,
}))

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="heatmap-echarts-stub" />,
}))

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

describe('Heatmap2DChart — aucune légende de case vide', () => {
  for (const emptyCells of [undefined, 'hidden', 'blank'] as const) {
    it(`mode ${emptyCells ?? 'par défaut'} : une case vide ne fait naître aucune légende`, async () => {
      render(<Heatmap2DChart series={SERIES_AVEC_CASE_VIDE} emptyCells={emptyCells} />)
      await screen.findByTestId('heatmap-echarts-stub')
      expect(screen.queryByTestId('chart-card-legend')).toBeNull()
    })
  }

  it('sans case vide : aucune légende non plus', async () => {
    render(<Heatmap2DChart series={SERIES_SANS_CASE_VIDE} />)
    await screen.findByTestId('heatmap-echarts-stub')
    expect(screen.queryByTestId('chart-card-legend')).toBeNull()
  })

  it('la légende posée par l’appelant passe telle quelle', async () => {
    render(<Heatmap2DChart series={SERIES_AVEC_CASE_VIDE} legend={<p data-testid="legende-appelant">paliers</p>} />)
    await screen.findByTestId('heatmap-echarts-stub')
    expect(screen.getByTestId('legende-appelant').textContent).toBe('paliers')
  })
})
