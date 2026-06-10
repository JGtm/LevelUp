/**
 * ChartFromOption.test.tsx — verrouille le contrat anti-régression de la
 * refonte états vides : option=null → bloc TITRÉ avec message vide (jamais de
 * disparition ni de canvas blanc) ; option non-null → rendu ECharts.
 *
 * echarts-for-react est mocké (canvas jsdom instable — cf. convention projet).
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { ChartFromOption } from './ChartFromOption'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

describe('ChartFromOption', () => {
  it('option=null → bloc titré conservé + emptyMessage affiché', async () => {
    render(
      <ChartFromOption
        title="Mon graphe"
        option={null}
        height={240}
        emptyMessage="Aucun point disponible pour cette période."
      />,
    )
    expect(screen.getByText('Mon graphe')).toBeInTheDocument()
    expect(await screen.findByTestId('chart-card-empty')).toHaveTextContent(
      'Aucun point disponible pour cette période.',
    )
  })

  it('option=null sans emptyMessage → défaut ChartCard', async () => {
    render(<ChartFromOption title="T" option={null} height={240} />)
    expect(await screen.findByTestId('chart-card-empty')).toHaveTextContent(
      'Aucune donnée à afficher',
    )
  })

  it('option non-null → pas d état vide (le canvas est rendu)', async () => {
    render(
      <ChartFromOption
        title="Mon graphe"
        option={{ series: [{ type: 'bar', data: [1] }] }}
        height={240}
      />,
    )
    expect(screen.getByText('Mon graphe')).toBeInTheDocument()
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
    expect(screen.queryByTestId('chart-card-empty')).toBeNull()
  })
})
