import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { KPIStrip, type KPICardData } from './KPIStrip'

describe('KPIStrip', () => {
  const baseCard: KPICardData = {
    id: 'kpi.test',
    label: 'Mon KPI',
    primary: '42',
  }

  it('rend toutes les cartes fournies', () => {
    const cards: KPICardData[] = [
      { id: 'a', label: 'A', primary: '1' },
      { id: 'b', label: 'B', primary: '2' },
      { id: 'c', label: 'C', primary: '3' },
    ]
    render(<KPIStrip cards={cards} />)
    expect(screen.getAllByTestId('kpi-card')).toHaveLength(3)
  })

  it('affiche label + primary', () => {
    render(<KPIStrip cards={[baseCard]} />)
    expect(screen.getByText('Mon KPI')).toBeTruthy()
    expect(screen.getByTestId('kpi-primary').textContent).toBe('42')
  })

  it('affiche secondary si fourni', () => {
    render(<KPIStrip cards={[{ ...baseCard, secondary: '/min' }]} />)
    expect(screen.getByTestId('kpi-secondary').textContent).toBe('/min')
  })

  it('omet secondary si absent', () => {
    render(<KPIStrip cards={[baseCard]} />)
    expect(screen.queryByTestId('kpi-secondary')).toBeNull()
  })

  it('affiche flèche above ▲', () => {
    render(<KPIStrip cards={[{ ...baseCard, trend: 'above' }]} />)
    const trend = screen.getByTestId('kpi-trend')
    expect(trend.textContent).toBe('▲')
    expect(trend.getAttribute('data-trend')).toBe('above')
  })

  it('affiche flèche below ▼', () => {
    render(<KPIStrip cards={[{ ...baseCard, trend: 'below' }]} />)
    expect(screen.getByTestId('kpi-trend').textContent).toBe('▼')
  })

  it('affiche near =', () => {
    render(<KPIStrip cards={[{ ...baseCard, trend: 'near' }]} />)
    expect(screen.getByTestId('kpi-trend').textContent).toBe('=')
  })

  it('omet la flèche pour trend=none ou absent', () => {
    render(<KPIStrip cards={[{ ...baseCard, trend: 'none' }]} />)
    expect(screen.queryByTestId('kpi-trend')).toBeNull()
    render(<KPIStrip cards={[baseCard]} />)
    expect(screen.queryAllByTestId('kpi-trend')).toHaveLength(0)
  })

  it('rend le slot custom si fourni', () => {
    render(
      <KPIStrip
        cards={[{ ...baseCard, custom: <div data-testid="custom-bar">stack</div> }]}
      />,
    )
    expect(screen.getByTestId('custom-bar')).toBeTruthy()
  })

  it('applique la classe wide pour les cartes wide=true', () => {
    render(<KPIStrip cards={[{ ...baseCard, wide: true }]} />)
    const card = screen.getByTestId('kpi-card')
    expect(card.className).toContain('lg:col-span-2')
  })

  it('applique la couleur via CSS variable selon le trend', () => {
    render(<KPIStrip cards={[{ ...baseCard, trend: 'above' }]} />)
    const trend = screen.getByTestId('kpi-trend') as HTMLElement
    // Style inline contient la CSS variable narrative-trend-positive
    expect(trend.style.color).toContain('--narrative-trend-positive')
  })
})
