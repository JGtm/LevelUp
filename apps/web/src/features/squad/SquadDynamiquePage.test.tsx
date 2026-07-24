/**
 * SquadDynamiquePage.test.tsx — smoke tests.
 *
 * L'onglet Dynamique regroupe les sections déplacées depuis Contributions :
 * intensité, rendement/résistance et engagement. Les charts ECharts sont
 * stubés (résolution jsdom) ; on vérifie qu'ils sont montés même sans données,
 * et que la section engagement reçoit les match_ids du scope en ordre
 * chronologique ASC (cap 15).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import * as squadContextModule from './SquadContext'
import type { TeammatesPageResponse } from '@/lib/api/types'
import { SquadDynamiquePage } from './SquadDynamiquePage'

// Stub des charts ECharts pour éviter les erreurs de résolution en env test.
vi.mock('./SquadIntensityProfileChart', () => ({
  SquadIntensityProfileChart: () => <div data-testid="intensity-chart" />,
}))
vi.mock('./SquadEfficiencyChart', () => ({
  SquadEfficiencyChart: () => <div data-testid="efficiency-chart" />,
}))

const engagementProps: { matchIds?: string[] }[] = []
vi.mock('@/features/engagement/SquadEngagementSection', () => ({
  SquadEngagementSection: (props: { matchIds?: string[] }) => {
    engagementProps.push(props)
    return <div data-testid="engagement-section" />
  },
}))

function mockSquadContext(overrides: Partial<ReturnType<typeof squadContextModule.useSquadContext>>) {
  vi.spyOn(squadContextModule, 'useSquadContext').mockReturnValue({
    selectedRows: [],
    confirmedGamertags: [],
    pageData: null as unknown as TeammatesPageResponse,
    playerSlug: 'test',
    currentPlayerXuid: '',
    ...overrides,
  })
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('SquadDynamiquePage', () => {
  it('monte sans erreur avec pageData null', () => {
    mockSquadContext({})
    const { container } = renderWithProviders(<SquadDynamiquePage />)
    expect(container).toBeTruthy()
  })

  it('monte les sections (état vide) même quand pageData est null', () => {
    mockSquadContext({})
    renderWithProviders(<SquadDynamiquePage />)
    expect(screen.getByTestId('intensity-chart')).toBeInTheDocument()
    expect(screen.getByTestId('efficiency-chart')).toBeInTheDocument()
    expect(screen.getByTestId('engagement-section')).toBeInTheDocument()
  })

  it('passe a l engagement les match_ids du scope en ordre chronologique ASC, cap 15', () => {
    engagementProps.length = 0
    // match_history arrive DESC (recent d'abord) du backend : m17..m1.
    const history = Array.from({ length: 17 }, (_, i) => ({
      match_id: 'm' + (17 - i),
      start_time: '2026-06-0' + ((i % 9) + 1) + 'T10:00:00Z',
      outcome: 2,
    }))
    mockSquadContext({
      confirmedGamertags: ['FriendA'],
      pageData: { match_history: history } as unknown as TeammatesPageResponse,
    })
    renderWithProviders(<SquadDynamiquePage />)
    const got = engagementProps.at(-1)?.matchIds ?? []
    // Cap 15 (aligne sur le fallback handler) sur les PLUS RECENTS (m17..m3),
    // puis reverse → chronologique ASC : m3 d'abord, m17 en dernier.
    expect(got).toHaveLength(15)
    expect(got[0]).toBe('m3')
    expect(got.at(-1)).toBe('m17')
  })
})
