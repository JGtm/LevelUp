/**
 * Tests unitaires — TipsTicker.
 *
 * Vérifie : rendu vide, duplication des tips pour la boucle, attributs href,
 * aria-hidden sur les copies, paramétrage de la durée via CSS variable.
 */
import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { TipsTicker, type Tip } from './tips-ticker'

const SAMPLE_TIPS: Tip[] = [
  { id: 't1', term: 'Streak', shortDef: 'Suite continue', href: '/help#streak' },
  { id: 't2', term: 'Record', shortDef: 'Meilleur perso', href: '/help#record' },
]

describe('TipsTicker', () => {
  it('renders nothing when no tips', () => {
    const { container } = render(<TipsTicker tips={[]} />)
    expect(container.firstChild).toBeNull()
  })

  it('duplicates tips for seamless looping', () => {
    const { container } = render(<TipsTicker tips={SAMPLE_TIPS} />)
    // 2 tips × 2 copies = 4 pills
    const pills = container.querySelectorAll('a, span[aria-hidden]')
    const linkCount = container.querySelectorAll('a').length
    expect(linkCount).toBe(4)
    expect(pills.length).toBeGreaterThanOrEqual(4)
  })

  it('flags duplicated tips as aria-hidden for screen readers', () => {
    const { container } = render(<TipsTicker tips={SAMPLE_TIPS} />)
    const links = Array.from(container.querySelectorAll('a'))
    // first 2 must be visible to AT, last 2 must be aria-hidden
    expect(links[0].getAttribute('aria-hidden')).toBeNull()
    expect(links[1].getAttribute('aria-hidden')).toBeNull()
    expect(links[2].getAttribute('aria-hidden')).toBe('true')
    expect(links[3].getAttribute('aria-hidden')).toBe('true')
  })

  it('sets href on each tip link', () => {
    const { container } = render(<TipsTicker tips={SAMPLE_TIPS} />)
    const links = container.querySelectorAll('a')
    expect(links[0].getAttribute('href')).toBe('/help#streak')
    expect(links[1].getAttribute('href')).toBe('/help#record')
  })

  it('renders as span (not link) when href is missing', () => {
    const tips: Tip[] = [{ id: 'x', term: 'NoLink', shortDef: 'sans href' }]
    const { container } = render(<TipsTicker tips={tips} />)
    expect(container.querySelector('a')).toBeNull()
    expect(container.querySelectorAll('span').length).toBeGreaterThan(0)
  })

  it('exposes durationSeconds as CSS variable on the track', () => {
    const { container } = render(<TipsTicker tips={SAMPLE_TIPS} durationSeconds={30} />)
    // Find the inner flex track (sibling of <style>)
    const track = container.querySelector('[style*="--ticker-duration"]') as HTMLElement | null
    expect(track).not.toBeNull()
    expect(track?.style.getPropertyValue('--ticker-duration')).toBe('30s')
  })

  it('applies aria-label on the region wrapper', () => {
    const { container } = render(
      <TipsTicker tips={SAMPLE_TIPS} ariaLabel="Tips Ascension" />,
    )
    const region = container.querySelector('[role="region"]')
    expect(region?.getAttribute('aria-label')).toBe('Tips Ascension')
  })
})
