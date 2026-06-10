/**
 * SquadContributionsPage.test.tsx — smoke tests.
 *
 * Depuis la refonte des états vides : les graphes sont TOUJOURS montés (chaque
 * ChartCard gère son propre état vide titré) au lieu d'être masqués quand leur
 * source de données est absente. Les tests vérifient donc que les graphes sont
 * présents même sans données, et qu'ils restent présents avec données.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import * as squadContextModule from './SquadContext'
import type { TeammatesPageResponse } from '@/lib/api/types'
import { SquadContributionsPage } from './SquadContributionsPage'

// Stub des charts ECharts pour éviter les erreurs de résolution en env test.
vi.mock('./SquadPerMinuteChart', () => ({
  SquadPerMinuteChart: () => <div data-testid="per-minute-chart" />,
}))
vi.mock('./SquadSynergyRadarChart', () => ({
  SquadSynergyRadarChart: () => <div data-testid="synergy-radar-chart" />,
}))

function mockSquadContext(overrides: Partial<ReturnType<typeof squadContextModule.useSquadContext>>) {
  vi.spyOn(squadContextModule, 'useSquadContext').mockReturnValue({
    selectedRows: [],
    confirmedGamertags: [],
    pageData: null as unknown as TeammatesPageResponse,
    playerSlug: 'test',
    ...overrides,
  })
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('SquadContributionsPage', () => {
  it('monte sans erreur avec pageData null', () => {
    mockSquadContext({})
    const { container } = renderWithProviders(<SquadContributionsPage />)
    expect(container).toBeTruthy()
  })

  it('monte les graphes (état vide) même quand pageData est null', () => {
    mockSquadContext({})
    renderWithProviders(<SquadContributionsPage />)
    // Les graphes ne disparaissent plus : ils sont montés et délèguent leur
    // état vide à ChartCard (bloc titré + message).
    expect(screen.getByTestId('per-minute-chart')).toBeInTheDocument()
    expect(screen.getByTestId('synergy-radar-chart')).toBeInTheDocument()
  })

  it('affiche le per-minute chart quand per_minute_stats est renseigné', () => {
    mockSquadContext({
      confirmedGamertags: ['FriendA'],
      pageData: {
        per_minute_stats: [
          { player: 'test', kills_per_minute: 1, deaths_per_minute: 0.5, assists_per_minute: 0.2, match_count: 5 },
        ],
      } as unknown as TeammatesPageResponse,
    })
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByTestId('per-minute-chart')).toBeInTheDocument()
  })

  it('affiche le synergy radar quand synergy_radar est renseigné', () => {
    mockSquadContext({
      confirmedGamertags: ['FriendA'],
      pageData: {
        synergy_radar: [{ player: 'FriendA', combat: 0.8, survival: 0.6, support: 0.4, score: 0.7, objective: 0.5, impact: 0.9 }],
      } as unknown as TeammatesPageResponse,
    })
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByTestId('synergy-radar-chart')).toBeInTheDocument()
  })
})
