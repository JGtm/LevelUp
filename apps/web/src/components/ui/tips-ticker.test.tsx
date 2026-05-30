/**
 * Tests unitaires — TipsTicker (fondu enchaîné).
 *
 * Vérifie : rendu vide, pill unique visible, href, span sans lien, aria-label.
 * Le cycle (setInterval + setTimeout) est testé via fake timers.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import { act, render } from '@testing-library/react'
import { TipsTicker, type Tip } from './tips-ticker'

const SAMPLE_TIPS: Tip[] = [
  { id: 't1', term: 'Streak', shortDef: 'Suite continue', href: '/help#streak' },
  { id: 't2', term: 'Record', shortDef: 'Meilleur perso', href: '/help#record' },
]

afterEach(() => {
  vi.useRealTimers()
})

describe('TipsTicker', () => {
  it('renders nothing when no tips', () => {
    const { container } = render(<TipsTicker tips={[]} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders exactly one tip pill initially', () => {
    const { container } = render(<TipsTicker tips={SAMPLE_TIPS} />)
    expect(container.querySelectorAll('a').length).toBe(1)
    expect(container.querySelector('a')?.getAttribute('href')).toBe('/help#streak')
  })

  it('renders as span (not link) when href is missing', () => {
    const tips: Tip[] = [{ id: 'x', term: 'NoLink', shortDef: 'sans href' }]
    const { container } = render(<TipsTicker tips={tips} />)
    expect(container.querySelector('a')).toBeNull()
    expect(container.querySelectorAll('span').length).toBeGreaterThan(0)
  })

  it('applies aria-label on the region wrapper', () => {
    const { container } = render(
      <TipsTicker tips={SAMPLE_TIPS} ariaLabel="Tips Ascension" />,
    )
    const region = container.querySelector('[role="region"]')
    expect(region?.getAttribute('aria-label')).toBe('Tips Ascension')
  })

  it('advances to the next tip after one full cycle', async () => {
    vi.useFakeTimers()
    const { container } = render(
      <TipsTicker tips={SAMPLE_TIPS} displaySeconds={6} transitionSeconds={0.5} />,
    )
    expect(container.querySelector('a')?.getAttribute('href')).toBe('/help#streak')

    // cycleMs = (6 + 0.5*2) * 1000 = 7000ms → setInterval fires → setVisible(false)
    await act(async () => { vi.advanceTimersByTime(7000) })
    // transitionSeconds*1000 = 500ms → setTimeout fires → index++ + setVisible(true)
    await act(async () => { vi.advanceTimersByTime(500) })

    expect(container.querySelector('a')?.getAttribute('href')).toBe('/help#record')
  })
})
