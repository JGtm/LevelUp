import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { CapabilityGap } from './CapabilityGap'

describe('CapabilityGap', () => {
  it('mode hide retourne null (pas de DOM)', () => {
    const { container } = render(<CapabilityGap mode="hide" reasonLabel="Indisponible" />)
    expect(container.firstChild).toBeNull()
  })

  it('mode placeholder rend la carte avec reason', () => {
    render(<CapabilityGap mode="placeholder" reasonLabel="Aucune donnée filmée" />)
    const card = screen.getByTestId('capability-gap')
    expect(card.getAttribute('data-mode')).toBe('placeholder')
    expect(screen.getByTestId('capability-gap-reason').textContent).toBe('Aucune donnée filmée')
    expect(card.getAttribute('role')).toBe('status')
  })

  it('mode placeholder affiche un hint si fourni', () => {
    render(
      <CapabilityGap mode="placeholder" reasonLabel="Reason" hintLabel="Activez le sync" />,
    )
    expect(screen.getByTestId('capability-gap-hint').textContent).toBe('Activez le sync')
  })

  it('mode placeholder rend l\'icône optionnelle', () => {
    render(
      <CapabilityGap mode="placeholder" reasonLabel="X" icon={<span>📊</span>} />,
    )
    expect(screen.getByTestId('capability-gap-icon')).toBeTruthy()
  })

  it('mode cta rend un bouton avec href + label', () => {
    render(
      <CapabilityGap
        mode="cta"
        reasonLabel="Reason"
        cta={{ href: '/sync/awards', label: 'Lancer le sync' }}
      />,
    )
    const cta = screen.getByTestId('capability-gap-cta')
    expect(cta.tagName).toBe('A')
    expect(cta.getAttribute('href')).toBe('/sync/awards')
    expect(cta.textContent).toBe('Lancer le sync')
    // Lien interne -> pas de target ni rel
    expect(cta.getAttribute('target')).toBeNull()
    expect(cta.getAttribute('rel')).toBeNull()
  })

  it('mode cta external ouvre un nouvel onglet avec rel safe', () => {
    render(
      <CapabilityGap
        mode="cta"
        reasonLabel="Reason"
        cta={{ href: 'https://example.com', label: 'Doc', external: true }}
      />,
    )
    const cta = screen.getByTestId('capability-gap-cta')
    expect(cta.getAttribute('target')).toBe('_blank')
    expect(cta.getAttribute('rel')).toBe('noopener noreferrer')
  })

  it('mode cta sans cta object (defensive) -> bouton absent', () => {
    render(<CapabilityGap mode="cta" reasonLabel="Reason" />)
    expect(screen.queryByTestId('capability-gap-cta')).toBeNull()
  })

  it('hint absent -> pas d\'élément hint', () => {
    render(<CapabilityGap mode="placeholder" reasonLabel="Reason" />)
    expect(screen.queryByTestId('capability-gap-hint')).toBeNull()
  })
})
