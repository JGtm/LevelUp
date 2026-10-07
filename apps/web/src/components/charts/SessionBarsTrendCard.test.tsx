/**
 * SessionBarsTrendCard — le composant TRANSMET toutes les options de la frise au module
 * d'option : `baseline` (mode écart) et `hollowLegend` compris. Sans elles, l'axe annonçait
 * « Écart (points) » sous des bâtons tracés en valeur absolue.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { SessionBarsTrendChart } from './SessionBarsTrendCard'

const captured: Array<Record<string, unknown>> = []
vi.mock('echarts-for-react', () => ({
  default: (props: { option: Record<string, unknown> }) => {
    captured.push(props.option)
    return <div data-testid="echarts-mock" />
  },
}))

describe('SessionBarsTrendChart', () => {
  it('passe le mode écart et le témoin des bâtons creux au module d’option', async () => {
    render(
      <SessionBarsTrendChart
        labels={['12/09', '14/09']}
        yAxisLabel="Écart (points)"
        baseline={{ label: 'habituel 41 %', deltaUnit: 'pts' }}
        hollowLegend={{ label: 'Échantillon faible', color: '#111111' }}
        series={[
          {
            name: 'Frags appuyés',
            color: '#111111',
            valuesPct: [50, 30],
            hollow: [false, true],
            usual: { valuePct: 41, label: 'habituel 41 %' },
          },
        ]}
      />,
    )
    await screen.findByTestId('echarts-mock')
    const option = captured[captured.length - 1]
    const barres = (option.series as Array<{ type: string; data: unknown[] }>).filter((s) => s.type === 'bar')
    // Écart au repère : 50 - 41 = 9 (et non 50, la valeur absolue).
    expect(barres[0].data[0]).toBe(9)
    const fmt = (option.yAxis as { axisLabel: { formatter: (v: number) => string } }).axisLabel.formatter
    expect(fmt(10)).toBe('+10')
    expect((option.legend as { data: string[] }).data).toContain('Échantillon faible')
  })
})
