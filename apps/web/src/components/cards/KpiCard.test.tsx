/**
 * Tests KpiCard — primitive carte KPI unifiée (chrome bg-card + barre d'accent
 * 3px optionnelle + children libre). Fusion des types 2/4 du catalogue UI.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { KpiCard } from './KpiCard'

describe('KpiCard', () => {
  it('rend les children, le chrome bg-card et le testId', () => {
    const { container } = render(
      <KpiCard testId="kpi-x">
        <span>contenu</span>
      </KpiCard>,
    )
    expect(screen.getByTestId('kpi-x')).toBeInTheDocument()
    expect(screen.getByText('contenu')).toBeInTheDocument()
    expect(container.firstChild).toHaveClass('bg-card')
  })

  it("rend la barre d'accent (1er enfant h-[3px] + background inline) quand un token est fourni", () => {
    render(
      <KpiCard accent="outcome-win" testId="kpi-a">
        x
      </KpiCard>,
    )
    const firstChild = screen.getByTestId('kpi-a').firstElementChild
    expect(firstChild).toHaveClass('h-[3px]')
    expect(firstChild).toHaveStyle({ backgroundColor: 'var(--ac-outcome-win)' })
  })

  it("omet la barre d'accent sans token (children = 1er enfant)", () => {
    render(
      <KpiCard testId="kpi-b">
        <span>x</span>
      </KpiCard>,
    )
    expect(screen.getByTestId('kpi-b').firstElementChild?.tagName).toBe('SPAN')
  })
})
