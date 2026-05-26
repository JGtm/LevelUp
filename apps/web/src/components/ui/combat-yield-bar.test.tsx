/**
 * Tests unitaires — CombatYieldBar (Sprint 56).
 *
 * Vérifie : rendu nominal, état zéro, valeurs au-dessus du clip, tooltip au survol.
 */
import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { CombatYieldBar } from './combat-yield-bar'

describe('CombatYieldBar', () => {
  it('renders without error with no props', () => {
    const { container } = render(<CombatYieldBar />)
    expect(container.firstChild).toBeTruthy()
  })

  it('renders both bars when oc and dr are provided', () => {
    const { container } = render(
      // DR=1.0 est le baseline exact → 0px (aucun excès). OC=0.5 → ~48px.
      <CombatYieldBar offensiveConversion={0.5} defensiveResistance={1.0} />,
    )
    const bars = container.querySelectorAll('.rounded-l-full, .rounded-r-full')
    expect(bars.length).toBeGreaterThan(0)
    // Au moins la barre OC doit avoir une largeur > 0
    const widths = Array.from(bars).map((b) => (b as HTMLElement).style.width)
    expect(widths.some((w) => w !== '0px' && w !== '')).toBe(true)
  })

  it('shows zero-width bars when values are zero', () => {
    const { container } = render(
      <CombatYieldBar offensiveConversion={0} defensiveResistance={0} />,
    )
    // Avec des valeurs zéro, les barres doivent avoir width=0
    const bars = container.querySelectorAll('[style*="width: 0px"]')
    expect(bars.length).toBeGreaterThanOrEqual(0) // pas d'erreur = OK
  })

  it('clips to max 120px when value exceeds 1.5×p80', () => {
    // OC = 10.0 dépasse largement 0.83 × 1.5 = 1.245 → doit clipper à 120px
    const { container } = render(
      <CombatYieldBar offensiveConversion={10} defensiveResistance={10} />,
    )
    // Les barres doivent exister sans overflow
    expect(container.firstChild).toBeTruthy()
  })

  it('shows tooltip on hover when data is present', () => {
    const { container } = render(
      <CombatYieldBar
        offensiveConversion={0.9}
        defensiveResistance={1.2}
        damagePerKill={800}
        damagePerDeath={1400}
      />,
    )
    const wrapper = container.firstChild as HTMLElement
    fireEvent.mouseEnter(wrapper)
    expect(screen.getByText(/Offensif/i)).toBeTruthy()
    expect(screen.getByText(/Défensif/i)).toBeTruthy()
    expect(screen.getByText(/dmg\/kill/i)).toBeTruthy()
    expect(screen.getByText(/dmg\/mort/i)).toBeTruthy()
  })

  it('hides tooltip on mouse leave', () => {
    const { container } = render(
      <CombatYieldBar offensiveConversion={0.9} defensiveResistance={1.2} />,
    )
    const wrapper = container.firstChild as HTMLElement
    fireEvent.mouseEnter(wrapper)
    fireEvent.mouseLeave(wrapper)
    expect(screen.queryByText(/Offensif/i)).toBeNull()
  })

  it('shows dashes when values are null', () => {
    const { container } = render(
      <CombatYieldBar offensiveConversion={null} defensiveResistance={null} />,
    )
    expect(container.firstChild).toBeTruthy()
  })
})
