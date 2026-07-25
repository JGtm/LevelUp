/**
 * ChartLegend.test.tsx — légende HTML réutilisable (pied de card). Vérifie le rendu
 * des entrées, l'état vide, l'estompage (survol lié) et le callback de survol.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { ChartLegend, type ChartLegendItem } from './ChartLegend'

const ITEMS: ChartLegendItem[] = [
  { key: 'a', label: 'Épaule', color: '#112233' },
  { key: 'b', label: 'Mêlée', color: '#445566', dimmed: true },
]

describe('ChartLegend', () => {
  it('rend une entrée par item (pastille + libellé)', () => {
    render(<ChartLegend items={ITEMS} />)
    const legend = screen.getByTestId('chart-legend')
    expect(legend.querySelectorAll('li')).toHaveLength(2)
    expect(screen.getByText('Épaule')).toBeInTheDocument()
    expect(screen.getByText('Mêlée')).toBeInTheDocument()
  })

  it('liste vide → ne rend rien', () => {
    const { container } = render(<ChartLegend items={[]} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('estompe l\'entrée dimmed (survol lié)', () => {
    render(<ChartLegend items={ITEMS} />)
    const lis = screen.getByTestId('chart-legend').querySelectorAll('li')
    expect((lis[0] as HTMLElement).style.opacity).toBe('1')
    expect((lis[1] as HTMLElement).style.opacity).toBe('0.28')
  })

  it('remonte la clé au survol quand onItemHover est fourni', () => {
    const onHover = vi.fn()
    render(<ChartLegend items={ITEMS} onItemHover={onHover} />)
    const first = screen.getByTestId('chart-legend').querySelectorAll('li')[0]
    fireEvent.mouseEnter(first)
    expect(onHover).toHaveBeenCalledWith('a')
    fireEvent.mouseLeave(first)
    expect(onHover).toHaveBeenCalledWith(null)
  })
})
