/**
 * Tests ExplorerTargetSeasonMatches — barres verticales (total par saison) +
 * badge CSR. Le builder pur buildSeasonMatchesOption est testé sans monter
 * ECharts ; le rendu mocke echarts-for-react (jsdom n'a pas de canvas).
 */
import { describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { SeasonMatchCount } from '@/lib/api/types'

import { ExplorerTargetSeasonMatches, buildSeasonMatchesOption } from './ExplorerTargetSeasonMatches'

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

interface OptShape {
  series: Array<{ name: string; data: number[]; markPoint?: { data: unknown[] } }>
  xAxis: { data: string[] }
}

describe('buildSeasonMatchesOption', () => {
  it('une barre par saison (total) + badge CSR en markPoint quand le rang est connu', () => {
    const seasons: SeasonMatchCount[] = [
      {
        season_id: 'season6',
        season_name: 'S6',
        matches: 20,
        csr_tier: 'Onyx',
        csr_badge_image_url: '/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png',
      },
      { season_id: 'season7', season_name: 'S7', matches: 9 },
    ]
    const opt = buildSeasonMatchesOption(seasons, 'Parties') as unknown as OptShape
    // 1 série (total), axe X = saisons, valeurs = totaux.
    expect(opt.series).toHaveLength(1)
    expect(opt.series[0].name).toBe('Parties')
    expect(opt.series[0].data).toEqual([20, 9])
    expect(opt.xAxis.data).toEqual(['S6', 'S7'])
    // Un seul badge CSR (S6) — S7 n'a pas de rang.
    expect(opt.series[0].markPoint?.data).toHaveLength(1)
  })

  it('pas de markPoint quand aucune saison n’a de badge CSR', () => {
    const seasons: SeasonMatchCount[] = [
      { season_id: 'season1', season_name: 'S1', matches: 42 },
    ]
    const opt = buildSeasonMatchesOption(seasons, 'Parties') as unknown as {
      series: Array<{ markPoint?: unknown }>
    }
    expect(opt.series[0].markPoint).toBeUndefined()
  })
})

describe('ExplorerTargetSeasonMatches', () => {
  it('rend le conteneur avec le titre', () => {
    const seasons: SeasonMatchCount[] = [{ season_id: 'season13', season_name: 'S13', matches: 5 }]
    renderWithProviders(<ExplorerTargetSeasonMatches seasons={seasons} title="Matchs par saison" />)
    expect(screen.getByTestId('explorer-target-season-matches')).toBeInTheDocument()
    expect(screen.getByText('Matchs par saison')).toBeInTheDocument()
  })

  it('rend un placeholder titré (jamais masqué) si liste vide', () => {
    renderWithProviders(
      <ExplorerTargetSeasonMatches seasons={[]} title="Matchs par saison" />,
    )
    // Jamais d'espace blanc silencieux : conteneur + barre de titre restent rendus,
    // et ChartCard affiche son état « empty » (testid stable, indépendant de la locale).
    expect(screen.getByTestId('explorer-target-season-matches')).toBeInTheDocument()
    expect(screen.getByText('Matchs par saison')).toBeInTheDocument()
    expect(screen.getByTestId('chart-card-empty')).toBeInTheDocument()
  })

  // Lot A3 (fin de la dégradation muette) : badge discret dans la barre de titre
  // quand liveStatus explique le repli local (ex. Lot A2 non traité, bucketing
  // local sans service record par saison).
  it('affiche le badge live_status quand local_partial', () => {
    renderWithProviders(
      <ExplorerTargetSeasonMatches seasons={[]} title="Matchs par saison" liveStatus="local_partial" />,
    )
    expect(screen.getByTestId('explorer-live-status-badge-local_partial')).toBeInTheDocument()
    expect(screen.getByText('Live partiel')).toBeInTheDocument()
  })

  it('n\'affiche aucun badge sans liveStatus (compat antérieure au Lot A3)', () => {
    renderWithProviders(
      <ExplorerTargetSeasonMatches seasons={[{ season_id: 's1', season_name: 'S1', matches: 3 }]} title="Matchs par saison" />,
    )
    expect(screen.queryByTestId(/explorer-live-status-badge-/)).not.toBeInTheDocument()
  })
})
