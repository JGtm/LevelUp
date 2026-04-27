import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { FloatingLegend } from './FloatingLegend'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `var(${token})`,
}))

describe('FloatingLegend', () => {
  it('rend une pastille par joueur dans l\'ordre stable', () => {
    render(<FloatingLegend squadOrder={['main', 'f1', 'f2']} />)
    const legend = screen.getByTestId('floating-legend')
    expect(legend).toBeTruthy()
    expect(legend.textContent).toContain('main')
    expect(legend.textContent).toContain('f1')
    expect(legend.textContent).toContain('f2')
    // Ordre dans le DOM
    const text = legend.textContent ?? ''
    expect(text.indexOf('main')).toBeLessThan(text.indexOf('f1'))
    expect(text.indexOf('f1')).toBeLessThan(text.indexOf('f2'))
  })

  it("applique la couleur chart-series-N a chaque pastille", () => {
    render(<FloatingLegend squadOrder={['main', 'f1']} />)
    const dot0 = screen.getByTestId('floating-legend-dot-0')
    const dot1 = screen.getByTestId('floating-legend-dot-1')
    expect(dot0.style.backgroundColor).toBe('var(chart-series-1)')
    expect(dot1.style.backgroundColor).toBe('var(chart-series-2)')
  })

  it('squadOrder vide → composant absent', () => {
    const { container } = render(<FloatingLegend squadOrder={[]} />)
    expect(container.firstChild).toBeNull()
  })

  it('cycle modulo 8 sur les couleurs si > 8 joueurs', () => {
    // Edge case : on peut avoir plus que 8 (théoriquement squad limité à 4
    // mais la prop accepte plus). La couleur du 9e doit cycler à chart-series-1.
    const longSquad = Array.from({ length: 9 }, (_, i) => `p${i}`)
    render(<FloatingLegend squadOrder={longSquad} />)
    const dot8 = screen.getByTestId('floating-legend-dot-8')
    expect(dot8.style.backgroundColor).toBe('var(chart-series-1)')
  })
})
