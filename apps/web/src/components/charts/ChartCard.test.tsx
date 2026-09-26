import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'

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
    expect(screen.getByTestId('chart-card-empty').textContent).toContain('Pas de matchs')
  })

  // L'etat vide d'une carte de graphe se dessine comme TOUS les autres de l'app
  // (`components/ui/empty-state.tsx`) : titre en gras + description grise dans un cadre
  // pointille. Decision utilisateur du 2026-09-22.
  it("l'etat vide rend le gabarit canonique : titre en gras + description grise", () => {
    render(<ChartCard {...baseProps} emptyTitle="Rien ici" emptyMessage="Pas de matchs" />)
    const empty = screen.getByTestId('chart-card-empty')

    const titleEl = screen.getByText('Rien ici')
    expect(titleEl.className).toContain('font-semibold')
    expect(titleEl.className).toContain('text-foreground')

    const descEl = screen.getByText('Pas de matchs')
    expect(descEl.className).toContain('text-muted-foreground')

    // Le cadre pointille du gabarit canonique, et la hauteur du graphe conservee.
    expect(empty.querySelector('.border-dashed')).not.toBeNull()
    expect(empty.style.minHeight).toBe('320px')
  })

  it("etat vide sans props : defauts FR de la locale du shell", () => {
    useAppShellStore.setState({ locale: 'fr' })
    render(<ChartCard {...baseProps} />)
    const empty = screen.getByTestId('chart-card-empty')
    expect(empty.textContent).toContain('Aucune donnée')
    expect(empty.textContent).toContain('Aucune donnée à afficher pour cette sélection')
  })

  it("etat vide sans props : defauts EN quand la locale est 'en'", () => {
    useAppShellStore.setState({ locale: 'en' })
    try {
      render(<ChartCard {...baseProps} />)
      const empty = screen.getByTestId('chart-card-empty')
      expect(empty.textContent).toContain('No data')
      expect(empty.textContent).toContain('No data to display for this selection')
      expect(empty.textContent).not.toContain('Aucune')
    } finally {
      useAppShellStore.setState({ locale: 'fr' })
    }
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
