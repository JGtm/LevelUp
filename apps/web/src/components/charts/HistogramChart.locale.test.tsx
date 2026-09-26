/**
 * Le libellé par défaut de l'axe Y d'un histogramme est BILINGUE (2026-09-22).
 *
 * Il valait `'Matchs'` en dur dans le builder pur — une string UI sans parité EN au cœur
 * d'un wrapper monté par toutes les pages de distribution. Il est désormais résolu par le
 * composant depuis `common.charts.axis_matches`, dans la locale du shell.
 */
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { useAppShellStore } from '@/stores/appShellStore'

import { HistogramChart } from './HistogramChart'

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

const series = [
  { key: 'h', datapoints: [{ binStart: 0, binEnd: 1, count: 3 }] },
]

async function yAxisName(): Promise<string> {
  const stub = await screen.findByTestId('echarts-stub')
  const opt = JSON.parse(stub.textContent ?? '{}') as { yAxis?: { name?: string } }
  return opt.yAxis?.name ?? ''
}

afterEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

describe("HistogramChart — libellé par défaut de l'axe Y", () => {
  it('locale fr : « Matchs »', async () => {
    useAppShellStore.setState({ locale: 'fr' })
    render(<HistogramChart series={series} />)
    expect(await yAxisName()).toBe('Matchs')
  })

  it('locale en : « Matches »', async () => {
    useAppShellStore.setState({ locale: 'en' })
    render(<HistogramChart series={series} />)
    expect(await yAxisName()).toBe('Matches')
  })

  it("un libellé explicite l'emporte sur le défaut", async () => {
    useAppShellStore.setState({ locale: 'fr' })
    render(
      <HistogramChart series={series} yAxisLabel="Parties" />,
    )
    expect(await yAxisName()).toBe('Parties')
  })
})
