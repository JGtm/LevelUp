import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { ALL_PERIODS, PeriodFilter, type Period } from './PeriodFilter'

const labels: Record<Period, string> = {
  all: 'Toutes',
  '2y': '2 ans',
  '1y': '1 an',
  '1m': '1 mois',
  '1w': '1 semaine',
}

describe('PeriodFilter', () => {
  it('rend les 5 périodes canoniques', () => {
    render(<PeriodFilter value="all" labels={labels} onChange={() => {}} />)
    for (const period of ALL_PERIODS) {
      expect(screen.getByTestId(`period-filter-option-${period}`)).toBeInTheDocument()
    }
  })

  it('marque la période active via aria-checked et data-active', () => {
    render(<PeriodFilter value="1y" labels={labels} onChange={() => {}} />)
    const active = screen.getByTestId('period-filter-option-1y')
    expect(active.getAttribute('aria-checked')).toBe('true')
    expect(active.getAttribute('data-active')).toBe('true')
    const inactive = screen.getByTestId('period-filter-option-all')
    expect(inactive.getAttribute('aria-checked')).toBe('false')
  })

  it('appelle onChange au clic sur une période', () => {
    const onChange = vi.fn()
    render(<PeriodFilter value="all" labels={labels} onChange={onChange} />)
    fireEvent.click(screen.getByTestId('period-filter-option-1m'))
    expect(onChange).toHaveBeenCalledWith('1m')
  })

  it('disabled empêche les clics et applique opacité', () => {
    const onChange = vi.fn()
    render(<PeriodFilter value="all" labels={labels} onChange={onChange} disabled />)
    const opt = screen.getByTestId('period-filter-option-2y')
    expect(opt.hasAttribute('disabled')).toBe(true)
    fireEvent.click(opt)
    expect(onChange).not.toHaveBeenCalled()
  })

  it("expose role=radiogroup avec aria-label par défaut", () => {
    render(<PeriodFilter value="all" labels={labels} onChange={() => {}} />)
    const group = screen.getByRole('radiogroup', { name: 'Filtre période' })
    expect(group).toBeInTheDocument()
  })

  it("aria-label custom remplace le default", () => {
    render(
      <PeriodFilter
        value="all"
        labels={labels}
        onChange={() => {}}
        ariaLabel="Sélection de période historique"
      />,
    )
    expect(
      screen.getByRole('radiogroup', { name: 'Sélection de période historique' }),
    ).toBeInTheDocument()
  })

  it('accepte une className additionnelle', () => {
    render(
      <PeriodFilter
        value="all"
        labels={labels}
        onChange={() => {}}
        className="custom-class"
      />,
    )
    const group = screen.getByTestId('period-filter')
    expect(group.className).toContain('custom-class')
  })
})
