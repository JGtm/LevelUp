/**
 * SquadDynamiquePage.test.tsx — smoke tests.
 *
 * L'onglet Dynamique regroupe les sections déplacées depuis Contributions :
 * intensité, rendement/résistance, « Premier frag / première mort » et
 * engagement. Les charts ECharts sont stubés (résolution jsdom) ; on vérifie
 * qu'ils sont montés même sans données, et que la section engagement reçoit les
 * match_ids du scope en ordre chronologique ASC (cap 15). Lot L1
 * (PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : « Écart cumulé au FDA attendu »
 * sur la même rangée que « Balance des dégâts cumulée », absent sans la capability
 * expected_stats (la balance prend alors toute la rangée).
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

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
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
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  vi.restoreAllMocks()
})

function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      {
        slug: 'test_title',
        name: 'Test',
        status: 'active',
        capabilities: caps,
        is_default: true,
        effective_hp_to_kill: 225,
        provides_damage_taken: true,
        provides_team_mmr: true,
        provides_max_killing_spree: true,
        offensive_conversion_p80: 0.9,
        defensive_resistance_p80: 1.65,
      },
    ],
  })
}

const PERF_PAGE = {
  main_player: 'Main',
  performance_series: {
    Main: [
      {
        match_id: 'm1',
        start_time: '2026-09-22T20:00:00Z',
        match_order: 0,
        kills: 10,
        deaths: 5,
        assists: 3,
        kda: 1.5,
        kda_expected: 1.0,
        damage_dealt: 3000,
        damage_taken: 2500,
      },
    ],
  },
} as unknown as TeammatesPageResponse

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

  it('monte le bloc « Premier frag / première mort » (état vide sans données)', () => {
    mockSquadContext({})
    renderWithProviders(<SquadDynamiquePage />)
    expect(screen.getByText('Premier frag / première mort')).toBeInTheDocument()
  })

  it('alimente les bandes depuis first_blood du payload (une bande par joueur)', () => {
    mockSquadContext({
      confirmedGamertags: ['FriendA'],
      pageData: {
        main_player: 'Main',
        first_blood: [
          {
            player: 'Main',
            matches: [{ match_id: 'm1', first_kill_sec: 20, first_death_sec: 55 }],
          },
          {
            player: 'FriendA',
            matches: [{ match_id: 'm1', first_kill_sec: 35, first_death_sec: 40 }],
          },
        ],
      } as unknown as TeammatesPageResponse,
    })
    renderWithProviders(<SquadDynamiquePage />)
    // Le chart est rendu (pas l'état vide) dès qu'une bande porte un événement.
    expect(screen.getByText('Premier frag / première mort')).toBeInTheDocument()
    expect(
      screen.queryByText('Aucun premier frag ni première mort sur ce périmètre'),
    ).not.toBeInTheDocument()
  })

  it('monte « Écart cumulé au FDA attendu » sur la rangée de « Balance des dégâts cumulée »', () => {
    setTitleCaps(['expected_stats', 'damage_taken'])
    mockSquadContext({ confirmedGamertags: [], pageData: PERF_PAGE, playerSlug: 'main' })
    renderWithProviders(<SquadDynamiquePage />)
    const row = screen.getByTestId('squad-cumulative-row')
    const fda = screen.getByText('Écart cumulé au FDA attendu')
    const balance = screen.getByText('Balance des dégâts cumulée')
    expect(row).toContainElement(fda)
    expect(row).toContainElement(balance)
    // Deux colonnes sur desktop, même rangée (grid stretch → même hauteur).
    expect(row.className).toContain('md:grid-cols-2')
    expect(row.children).toHaveLength(2)
  })

  it('sans expected_stats : la carte FDA est absente, la balance reste seule sur sa rangée', () => {
    setTitleCaps(['damage_taken'])
    mockSquadContext({ confirmedGamertags: [], pageData: PERF_PAGE, playerSlug: 'main' })
    renderWithProviders(<SquadDynamiquePage />)
    expect(screen.queryByText('Écart cumulé au FDA attendu')).toBeNull()
    const row = screen.getByTestId('squad-cumulative-row')
    expect(row).toContainElement(screen.getByText('Balance des dégâts cumulée'))
    expect(row.children).toHaveLength(1)
    // La survivante prend toute la rangée.
    expect(row.className).toContain('md:[&>*:only-child]:col-span-2')
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
