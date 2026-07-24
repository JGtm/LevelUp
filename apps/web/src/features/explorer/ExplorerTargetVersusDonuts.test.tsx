/**
 * Tests ExplorerTargetVersusDonuts — rendu conditionnel de la dernière rangée de
 * la section « matchs joués ensemble ». Les briques réutilisées du hub Relations
 * (donut SVG + chart ECharts) sont MOQUÉES : on teste la logique d'assemblage
 * (2 donuts, repère perso, présence/absence du graphe, état vide d'un rôle).
 */
import { describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerEncounterStats } from '@/lib/api/types'

vi.mock('@/features/palmares/RelationWinRateDonut', () => ({
  RelationWinRateDonut: ({
    winRate,
    personalAvg,
    caption,
  }: {
    winRate: number | null
    personalAvg?: number | null
    caption?: string
  }) => (
    <div data-testid="donut" data-personal-avg={personalAvg ?? ''}>
      {caption}:{winRate}
    </div>
  ),
}))
vi.mock('@/features/palmares/CumulativeFragGapChart', () => ({
  CumulativeFragGapChart: ({ points }: { points: unknown[] }) => (
    <div data-testid="frag-chart">{points.length}</div>
  ),
}))

import { ExplorerTargetVersusDonuts } from './ExplorerTargetVersusDonuts'

const BASE: ExplorerEncounterStats = {
  count_together: 10,
  ally_count: 6,
  enemy_count: 4,
  winrate_as_ally: 0.66,
  winrate_vs_enemy: 0.5,
  player_win_rate: 0.55,
  kills_dealt: 12,
  deaths_suffered: 8,
  frag_gap_series: [
    { cumulative: 2, outcome: 'win' },
    { cumulative: -1, outcome: 'loss' },
  ],
}

describe('ExplorerTargetVersusDonuts', () => {
  it('rend les deux donuts (repère perso) + le graphe écart de frags', () => {
    renderWithProviders(<ExplorerTargetVersusDonuts encounterStats={BASE} />)
    const donuts = screen.getAllByTestId('donut')
    expect(donuts).toHaveLength(2)
    // Repère « moyenne perso historique » propagé aux deux donuts.
    donuts.forEach((d) => expect(d.getAttribute('data-personal-avg')).toBe('0.55'))
    // Graphe présent avec ses 2 points + titre.
    expect(screen.getByTestId('frag-chart')).toHaveTextContent('2')
    expect(screen.getByText('Écart de frags cumulé')).toBeInTheDocument()
  })

  it('masque le graphe quand aucun duel (frag_gap_series vide)', () => {
    renderWithProviders(
      <ExplorerTargetVersusDonuts encounterStats={{ ...BASE, frag_gap_series: [] }} />,
    )
    expect(screen.getAllByTestId('donut')).toHaveLength(2)
    expect(screen.queryByTestId('frag-chart')).not.toBeInTheDocument()
    expect(screen.queryByText('Écart de frags cumulé')).not.toBeInTheDocument()
  })

  it('affiche « — » pour un rôle sans taux (jamais allié)', () => {
    renderWithProviders(
      <ExplorerTargetVersusDonuts
        encounterStats={{ ...BASE, winrate_as_ally: undefined, frag_gap_series: [] }}
      />,
    )
    // Un seul vrai donut (face à lui) ; « ensemble » tombe en état vide « — ».
    expect(screen.getAllByTestId('donut')).toHaveLength(1)
    expect(screen.getByText('Taux de victoires ensemble')).toBeInTheDocument()
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('ne rend rien si aucun taux ni duel', () => {
    renderWithProviders(
      <ExplorerTargetVersusDonuts
        encounterStats={{
          count_together: 0,
          winrate_as_ally: undefined,
          winrate_vs_enemy: undefined,
          frag_gap_series: [],
        }}
      />,
    )
    expect(screen.queryByTestId('explorer-target-versus')).not.toBeInTheDocument()
  })
})
