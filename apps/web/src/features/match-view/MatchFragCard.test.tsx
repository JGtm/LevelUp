/**
 * MatchFragCard.test.tsx — survol LIÉ entre les deux cartes séparées (FIX 2) : un état
 * `hoveredClass` partagé est remonté au parent. Les composants enfants sont mockés pour
 * exposer les props/callbacks du câblage (le rendu SVG/ECharts est testé chez eux).
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { MatchFragCard } from './MatchFragCard'
import type { FragDistribution, MatchWeaponKill } from '@/lib/api/types'

// FragSunburst mocké : bouton pour simuler le survol d'une classe + affiche la classe
// externe reçue (réciproque : pilotée par le breakdown).
vi.mock('@/components/charts/FragSunburst', () => ({
  FragSunburst: ({
    externalHoveredClass,
    onClassHover,
  }: {
    externalHoveredClass?: string | null
    onClassHover?: (c: string | null) => void
  }) => (
    <div data-testid="sunburst" data-external={externalHoveredClass ?? ''}>
      <button data-testid="sun-hover" onClick={() => onClassHover?.('melee')} />
      <button data-testid="sun-leave" onClick={() => onClassHover?.(null)} />
    </div>
  ),
}))

// FragWeaponBreakdown mocké : bouton pour simuler le survol d'une barre + affiche la
// classe survolée reçue (le breakdown estompe les autres à partir de cette prop).
vi.mock('@/components/charts/FragWeaponBreakdown', () => ({
  FragWeaponBreakdown: ({
    hoveredClass,
    onClassHover,
  }: {
    hoveredClass?: string | null
    onClassHover?: (c: string | null) => void
  }) => (
    <div data-testid="breakdown" data-hovered={hoveredClass ?? ''}>
      <button data-testid="brk-hover" onClick={() => onClassHover?.('shoulder')} />
    </div>
  ),
}))

const DIST: FragDistribution = {
  total_kills: 8,
  classes: [
    { class: 'shoulder', kills: 5, authoritative: false },
    { class: 'melee', kills: 3, authoritative: true },
  ],
}
const WEAPONS: MatchWeaponKill[] = [
  { weapon_label: 'BR75', kill_count: 5, class: 'shoulder' } as MatchWeaponKill,
]

describe('MatchFragCard (survol lié)', () => {
  it('monte les deux cartes ; hoveredClass initial vide', () => {
    render(<MatchFragCard distribution={DIST} weapons={WEAPONS} />)
    expect(screen.getByTestId('sunburst')).toBeInTheDocument()
    expect(screen.getByTestId('breakdown')).toHaveAttribute('data-hovered', '')
  })

  it('survol sunburst → propage hoveredClass au breakdown (dim des non-matchs)', () => {
    render(<MatchFragCard distribution={DIST} weapons={WEAPONS} />)
    fireEvent.click(screen.getByTestId('sun-hover'))
    expect(screen.getByTestId('breakdown')).toHaveAttribute('data-hovered', 'melee')
    fireEvent.click(screen.getByTestId('sun-leave'))
    expect(screen.getByTestId('breakdown')).toHaveAttribute('data-hovered', '')
  })

  it('réciproque : survol barre → propage la classe au sunburst', () => {
    render(<MatchFragCard distribution={DIST} weapons={WEAPONS} />)
    fireEvent.click(screen.getByTestId('brk-hover'))
    expect(screen.getByTestId('sunburst')).toHaveAttribute('data-external', 'shoulder')
  })

  it('aucune donnée → rend null', () => {
    const { container } = render(
      <MatchFragCard distribution={{ total_kills: 0, classes: [] }} weapons={[]} />,
    )
    expect(container).toBeEmptyDOMElement()
  })
})
