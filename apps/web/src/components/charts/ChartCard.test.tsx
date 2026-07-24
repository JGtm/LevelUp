import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

import { ChartCard, type ChartSeries } from './ChartCard'

// Mock echarts-for-react pour eviter le cout d'instancier le canvas en jsdom
// (jsdom ne supporte pas le canvas par defaut, et echarts en a besoin).
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

describe('ChartCard', () => {
  const baseProps = {
    series: [] as ChartSeries[],
    buildOption: vi.fn(() => ({ xAxis: { type: 'category' as const } })),
  }

  it('rend l\'etat loading quand loading=true', () => {
    render(<ChartCard {...baseProps} loading />)
    expect(screen.getByTestId('chart-card-loading')).toBeTruthy()
    expect(baseProps.buildOption).not.toHaveBeenCalled()
  })

  it('rend l\'etat error quand error est present', () => {
    const error = new Error('boom')
    render(<ChartCard {...baseProps} error={error} />)
    const el = screen.getByTestId('chart-card-error')
    expect(el.textContent).toBe('boom')
    expect(el.getAttribute('role')).toBe('alert')
  })

  it('error sans message affiche fallback humain', () => {
    const error = new Error()
    render(<ChartCard {...baseProps} error={error} />)
    expect(screen.getByTestId('chart-card-error').textContent).toBe('Erreur de chargement')
  })

  it('rend l\'etat empty quand series est vide', () => {
    render(<ChartCard {...baseProps} emptyMessage="Pas de matchs" />)
    expect(screen.getByTestId('chart-card-empty').textContent).toBe('Pas de matchs')
  })

  it('rend ECharts quand series contient des datapoints', async () => {
    const buildOption = vi.fn(() => ({ xAxis: { type: 'category' as const } }))
    render(
      <ChartCard
        series={[{ key: 's1', datapoints: [{ x: 1, y: 2 }] }]}
        buildOption={buildOption}
      />,
    )
    // Suspense -> lazy import -> stub mock
    await waitFor(() => {
      expect(screen.getByTestId('echarts-stub')).toBeTruthy()
    })
    expect(buildOption).toHaveBeenCalledTimes(1)
  })

  it('affiche le titre quand fourni', () => {
    render(<ChartCard {...baseProps} title="Mon chart" />)
    expect(screen.getByText('Mon chart')).toBeTruthy()
  })

  it('rend les enfants en dessous du chart', async () => {
    render(
      <ChartCard {...baseProps}>
        <div data-testid="footer-note">Note pied</div>
      </ChartCard>,
    )
    expect(screen.getByTestId('footer-note').textContent).toBe('Note pied')
  })

  it('respecte la hauteur custom', () => {
    render(<ChartCard {...baseProps} height={500} loading />)
    const loading = screen.getByTestId('chart-card-loading')
    expect(loading.style.minHeight).toBe('500px')
  })

  it('rend la légende en pied de card quand la prop legend est fournie', () => {
    render(
      <ChartCard {...baseProps} legend={<div data-testid="legend-content">Légende</div>} />,
    )
    // Footer dédié avec chrome (bordure + padding) appliqué par ChartCard.
    expect(screen.getByTestId('chart-card-legend')).toBeTruthy()
    expect(screen.getByTestId('legend-content').textContent).toBe('Légende')
  })

  it('pas de pied de légende quand la prop legend est absente', () => {
    render(<ChartCard {...baseProps} />)
    expect(screen.queryByTestId('chart-card-legend')).toBeNull()
  })
})
