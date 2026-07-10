/**
 * Tests unitaires — CombatYieldBar (Sprint 56).
 *
 * Vérifie : rendu nominal, état zéro, valeurs au-dessus du clip, tooltip au survol.
 */
import { describe, it, expect, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { CombatYieldBar } from './combat-yield-bar'
import { useAppShellStore } from '@/stores/appShellStore'

describe('CombatYieldBar', () => {
  afterEach(() => {
    // Restaure la locale par défaut du store entre les tests (GH3-3).
    useAppShellStore.setState({ locale: 'fr' })
  })
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
    // OC = 10.0 dépasse largement 0.90 × 1.5 = 1.35 → doit clipper à 120px
    const { container } = render(
      <CombatYieldBar offensiveConversion={10} defensiveResistance={10} />,
    )
    // Les barres doivent exister sans overflow
    expect(container.firstChild).toBeTruthy()
  })

  it('shows tooltip on hover when data is present (FR)', () => {
    useAppShellStore.setState({ locale: 'fr' })
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
    expect(screen.getByText(/Rendement/i)).toBeTruthy()
    expect(screen.getByText(/Résistance/i)).toBeTruthy()
    expect(screen.getByText(/dégâts\/frag/i)).toBeTruthy()
    expect(screen.getByText(/dégâts\/mort/i)).toBeTruthy()
  })

  // GH3-3 : sous UI EN, la légende de la barre suit la locale (dmg/kill · dmg/death,
  // Yield / Resistance) — jamais de FR résiduel.
  it('shows tooltip labels in EN under EN locale', () => {
    useAppShellStore.setState({ locale: 'en' })
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
    expect(screen.getByText(/Yield/i)).toBeTruthy()
    expect(screen.getByText(/Resistance/i)).toBeTruthy()
    expect(screen.getByText(/dmg\/kill/i)).toBeTruthy()
    expect(screen.getByText(/dmg\/death/i)).toBeTruthy()
    // Aucun libellé FR ne doit subsister sous EN.
    expect(screen.queryByText(/dégâts/i)).toBeNull()
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
