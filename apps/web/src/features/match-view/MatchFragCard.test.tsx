/**
 * MatchFragCard.test.tsx — la rangée « Répartition des frags » | « Outils de destruction » (cartes A et
 * B de la Vue match). Les deux enfants sont mockés : seule la MISE EN PAGE est testée ici (deux
 * colonnes égales quand les deux cartes existent, une carte seule pleine largeur, rien sans aucune).
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { MatchFragCard } from './MatchFragCard'
import type { FragDistribution, SquadWeaponTools } from '@/lib/api/types'

vi.mock('@/components/charts/FragSunburst', () => ({
  FragSunburst: () => <div data-testid="sunburst" />,
}))
vi.mock('./MatchToolsCard', () => ({
  MatchToolsCard: () => <div data-testid="outils" />,
}))

const DIST: FragDistribution = {
  total_kills: 8,
  classes: [
    { class: 'shoulder', kills: 5, authoritative: false },
    { class: 'melee', kills: 3, authoritative: true },
  ],
}
const TOOLS: SquadWeaponTools = {
  players: ['Alpha'],
  lines: [{ kind: 'weapon', class: 'shoulder', label: 'BR75', kills_by_player: { Alpha: 5 }, total_squad: 5 }],
}

describe('MatchFragCard — anneau et outils de destruction', () => {
  it('les deux cartes : deux colonnes égales', () => {
    const { container } = render(<MatchFragCard distribution={DIST} tools={TOOLS} locale="fr" />)
    expect(screen.getByTestId('sunburst')).toBeInTheDocument()
    expect(screen.getByTestId('outils')).toBeInTheDocument()
    expect(container.firstElementChild?.className).toContain('lg:grid-cols-2')
  })

  it('anneau seul (aucun outil) : pleine largeur, sans grille', () => {
    const { container } = render(<MatchFragCard distribution={DIST} tools={null} locale="fr" />)
    expect(screen.queryByTestId('outils')).toBeNull()
    expect(container.firstElementChild?.className).not.toContain('grid-cols-2')
  })

  it('outils seuls (anneau vide, miroir de FragSunburst) : pleine largeur, sans grille', () => {
    const { container } = render(<MatchFragCard distribution={{ total_kills: 5, classes: [] }} tools={TOOLS} locale="fr" />)
    expect(screen.queryByTestId('sunburst')).toBeNull()
    expect(screen.getByTestId('outils')).toBeInTheDocument()
    expect(container.firstElementChild?.className).not.toContain('grid-cols-2')
  })

  it('aucune des deux : rien', () => {
    const { container } = render(<MatchFragCard distribution={{ total_kills: 0, classes: [] }} tools={{ players: [], lines: [] }} locale="fr" />)
    expect(container).toBeEmptyDOMElement()
  })
})
