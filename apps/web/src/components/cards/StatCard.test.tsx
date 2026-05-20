/**
 * Tests StatCard — couvre les 3 variants (default, kpi, metric).
 *
 * P8.13 (revue 2026-04-29 gap #13).
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatCard } from './StatCard'

describe('StatCard', () => {
  it('rend label + value en variant default', () => {
    render(<StatCard label="Win Rate" value="62%" />)
    expect(screen.getByText('Win Rate')).toBeInTheDocument()
    expect(screen.getByText('62%')).toBeInTheDocument()
  })

  it('rend en variant kpi avec text-primary', () => {
    const { container } = render(<StatCard label="K/D" value="1.42" variant="kpi" />)
    const valueEl = screen.getByText('1.42')
    expect(valueEl).toBeInTheDocument()
    expect(valueEl.className).toContain('text-primary')
    expect(container.firstChild).toHaveClass('bg-muted')
  })

  it('variant kpi compact : px-2 au lieu de px-4', () => {
    const { container } = render(<StatCard label="K/D" value="1.42" variant="kpi" compact />)
    expect(container.firstChild).toHaveClass('px-2')
    expect(container.firstChild).not.toHaveClass('px-4')
  })

  it('rend en variant metric avec uppercase tracking', () => {
    render(<StatCard label="Latence p95" value="142ms" variant="metric" />)
    const labelEl = screen.getByText('Latence p95')
    expect(labelEl.className).toContain('uppercase')
    expect(labelEl.className).toContain('tracking-label')
    // value en text-2xl (vs text-xl pour les autres variants)
    expect(screen.getByText('142ms').className).toContain('text-2xl')
  })

  it('rend hint quand fourni', () => {
    render(<StatCard label="Latence p95" value="142ms" hint="API health" variant="metric" />)
    expect(screen.getByText('API health')).toBeInTheDocument()
  })

  it('omet hint quand absent', () => {
    render(<StatCard label="K/D" value="1.42" variant="kpi" />)
    expect(screen.queryByText('API health')).not.toBeInTheDocument()
  })

  it('accepte value numérique', () => {
    render(<StatCard label="Matchs" value={42} />)
    expect(screen.getByText('42')).toBeInTheDocument()
  })
})
