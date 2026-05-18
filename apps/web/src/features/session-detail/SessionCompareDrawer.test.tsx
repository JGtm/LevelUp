/**
 * Tests ciblés sur l'interactivité du drawer (sans render des charts).
 *
 * On force `compareSession=null` pour rendre l'empty state du body (les charts
 * ne sont pas montés → pas d'erreur ECharts en jsdom). Les 4 charts internes
 * sont déjà couverts par leurs wrappers ECharts (TimeseriesLineChart.test.ts,
 * DonutChart.test.ts, OutcomeSequenceTape via SessionDetailPage.test.tsx).
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'

import { SessionCompareDrawer } from './SessionCompareDrawer'

function baseProps(overrides: Partial<React.ComponentProps<typeof SessionCompareDrawer>> = {}) {
  return {
    open: true,
    onClose: vi.fn(),
    compareSession: null,
    compareMatches: [],
    compareMetrics: [],
    suggestedCompare: null,
    previousLabel: null,
    nextLabel: null,
    onSelectLabel: vi.fn(),
    ...overrides,
  }
}

describe('SessionCompareDrawer', () => {
  it("affiche l'empty state quand compareSession est null", () => {
    renderWithProviders(<SessionCompareDrawer {...baseProps()} />)
    expect(screen.getByText(/Aucune session à comparer pour le moment/i)).toBeInTheDocument()
  })

  it('le bouton précédente appelle onSelectLabel avec previousLabel quand fourni', () => {
    const onSelectLabel = vi.fn()
    renderWithProviders(
      <SessionCompareDrawer
        {...baseProps({ previousLabel: 'S-prev', onSelectLabel })}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Session précédente/i }))
    expect(onSelectLabel).toHaveBeenCalledWith('S-prev')
  })

  it('le bouton suivante appelle onSelectLabel avec nextLabel quand fourni', () => {
    const onSelectLabel = vi.fn()
    renderWithProviders(
      <SessionCompareDrawer
        {...baseProps({ nextLabel: 'S-next', onSelectLabel })}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Session suivante/i }))
    expect(onSelectLabel).toHaveBeenCalledWith('S-next')
  })

  it("précédente / suivante sont disabled quand prev/next sont null", () => {
    renderWithProviders(<SessionCompareDrawer {...baseProps()} />)
    expect(screen.getByRole('button', { name: /Session précédente/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /Session suivante/i })).toBeDisabled()
  })

  it("le bouton suggestion n'apparaît QUE si suggestedCompare est fourni", () => {
    const { rerender } = renderWithProviders(<SessionCompareDrawer {...baseProps()} />)
    expect(
      screen.queryByRole('button', { name: /Utiliser la suggestion similaire/i }),
    ).not.toBeInTheDocument()

    rerender(
      <SessionCompareDrawer
        {...baseProps({
          suggestedCompare: {
            session_label: 'S-suggested',
            strategy: 'category-ranked-volume',
            reason: 'même catégorie ranked',
          },
        })}
      />,
    )
    expect(
      screen.getByRole('button', { name: /Utiliser la suggestion similaire/i }),
    ).toBeInTheDocument()
  })

  it('le bouton suggestion appelle onSelectLabel avec suggestedCompare.session_label', () => {
    const onSelectLabel = vi.fn()
    renderWithProviders(
      <SessionCompareDrawer
        {...baseProps({
          suggestedCompare: {
            session_label: 'S-suggested',
            strategy: 'category-ranked-volume',
            reason: 'même catégorie ranked',
          },
          onSelectLabel,
        })}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Utiliser la suggestion similaire/i }))
    expect(onSelectLabel).toHaveBeenCalledWith('S-suggested')
  })

  it('Escape ferme le drawer (appelle onClose)', () => {
    const onClose = vi.fn()
    renderWithProviders(<SessionCompareDrawer {...baseProps({ onClose })} />)
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalled()
  })

  it("Escape n'a pas d'effet quand le drawer est fermé (pas de listener actif)", () => {
    const onClose = vi.fn()
    renderWithProviders(<SessionCompareDrawer {...baseProps({ open: false, onClose })} />)
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).not.toHaveBeenCalled()
  })

  it('le bouton × du header appelle onClose', () => {
    const onClose = vi.fn()
    renderWithProviders(<SessionCompareDrawer {...baseProps({ onClose })} />)
    fireEvent.click(
      screen.getByRole('button', { name: /Fermer le panneau de comparaison/i }),
    )
    expect(onClose).toHaveBeenCalled()
  })
})
