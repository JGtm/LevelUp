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
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import * as squadContextModule from './SquadContext'
import type { TeammatesPageResponse } from '@/lib/api/types'
import { SquadContributionsPage } from './SquadContributionsPage'
import { block0709, history0709, history0709Evenings } from './objectif/objectif.fixtures'
import { impactSessionsFixture } from './impact-sessions/impactSessions.fixtures'

// Porte data-level de la section d'impact : posée par chaque test (le titre déclare, ou non, la timeline).
const capability = vi.hoisted(() => ({ timeline: true }))
vi.mock('@/lib/capabilities/dataCapabilities', () => ({ useDataCapability: () => capability.timeline }))

// Stub des charts ECharts pour éviter les erreurs de résolution en env test.
vi.mock('./SquadPerMinuteChart', () => ({
  SquadPerMinuteChart: ({ title }: { title?: React.ReactNode }) => <div data-testid="per-minute-chart">{title}</div>,
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
    currentPlayerXuid: '',
    ...overrides,
  })
}

beforeEach(() => {
  capability.timeline = true
  useSquadFilterStore.getState().setSessions({ picked_sessions: [], gap_minutes: 30 })
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

  it('« Stats par minute » : le titre nomme les dégâts seulement quand un joueur en porte', () => {
    const base = { player: 'test', kills_per_minute: 1, deaths_per_minute: 0.5, assists_per_minute: 0.2, match_count: 5 }
    mockSquadContext({ pageData: { per_minute_stats: [base] } as unknown as TeammatesPageResponse })
    const { unmount } = renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByTestId('per-minute-chart')).toHaveTextContent(/^Stats par minute — Frags \/ Morts \/ Assistances$/)
    unmount()

    mockSquadContext({
      pageData: { per_minute_stats: [{ ...base, damage_dealt_per_minute: 310 }] } as unknown as TeammatesPageResponse,
    })
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByTestId('per-minute-chart')).toHaveTextContent(
      'Stats par minute — Frags / Morts / Assistances / Dégâts',
    )
  })

  // LOT 3 « sections » (2026-09-22) : l'impact des coéquipiers et les médailles ont
  // quitté Synergies pour Contributions — une contribution par joueur, pas une
  // production de la composition. Sections non-graphes TOUJOURS montées (titre + état
  // vide géré par le composant), donc elles répondent présentes même sans données.
  it('monte « Impact des coéquipiers » et « Médailles » (arrivés de Synergies)', () => {
    mockSquadContext({})
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByText('Impact des coéquipiers')).toBeInTheDocument()
    expect(screen.getByText(/^Médailles/)).toBeInTheDocument()
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

  // LOT L2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 : « Répartition des frags »
  // puis « Outils de destruction » arrivent d'Usages, entre la rangée Stats par minute /
  // Radar synergie et la section Performance.
  it('monte « Répartition des frags » puis « Outils de destruction », avant « Performance »', () => {
    mockSquadContext({
      confirmedGamertags: ['FriendA'],
      pageData: {
        main_player: 'test',
        frag_classes: { test: [{ class: 'shoulder', kills: 12, authoritative: false }] },
        weapon_tools: {
          players: ['test'],
          lines: [
            { kind: 'weapon', weapon_key: 'hinf_br75', label: 'BR75', label_en: 'BR75', class: 'shoulder', kills_by_player: { test: 12 }, total_squad: 12 },
          ],
        },
      } as unknown as TeammatesPageResponse,
    })
    const { container } = renderWithProviders(<SquadContributionsPage />)
    const text = container.textContent ?? ''
    const section = text.indexOf('Frags et armes')
    const breakdown = text.indexOf('Répartition des frags')
    const tools = text.indexOf('Outils de destruction')
    const perf = text.indexOf('Performance')
    expect(breakdown).toBeGreaterThan(-1)
    expect(tools).toBeGreaterThan(-1)
    expect(section).toBeLessThan(breakdown)
    expect(breakdown).toBeLessThan(tools)
    expect(tools).toBeLessThan(perf)
    expect(screen.getByTestId('squad-frag-breakdown')).toBeInTheDocument()
  })

  it('monte les deux cartes frags même sans données (état vide de chaque carte)', () => {
    mockSquadContext({})
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByText('Répartition des frags')).toBeInTheDocument()
    expect(screen.getByText('Outils de destruction')).toBeInTheDocument()
  })

  // « Impact des coéquipiers » : un seul tableau par session, sous son titre de section, avant les médailles.
  it('monte le tableau d’impact par session sous son titre, avant les médailles', () => {
    mockSquadContext({
      confirmedGamertags: ['joueur_117', 'joueur_165'],
      pageData: { main_player: 'JGtm', squad_impact_sessions: impactSessionsFixture } as unknown as TeammatesPageResponse,
    })
    const { container } = renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByTestId('squad-impact-sessions')).toBeInTheDocument()
    expect(screen.getAllByTestId('squad-impact-row')).toHaveLength(4)
    const text = container.textContent ?? ''
    expect(text.indexOf('Impact des coéquipiers')).toBeLessThan(text.indexOf('Toutes les sessions'))
    expect(text.indexOf('Toutes les sessions')).toBeLessThan(text.search(/Médailles/))
    // L'ancienne matrice et le graphe par soirée ont disparu.
    expect(screen.queryByTestId('squad-impact-scoreboard')).toBeNull()
    expect(screen.queryByTestId('squad-impact-history')).toBeNull()
  })

  it('ouvre d’office la session sélectionnée dans le filtre de la page', () => {
    useSquadFilterStore.getState().setSessions({ picked_sessions: ['07/11/2026 18:30–19:40 (9)'], gap_minutes: 30 })
    mockSquadContext({
      confirmedGamertags: ['joueur_117', 'joueur_165'],
      pageData: { main_player: 'JGtm', squad_impact_sessions: impactSessionsFixture } as unknown as TeammatesPageResponse,
    })
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByRole('table', { name: /Rôles d’impact de la session 07\/11/ })).toBeInTheDocument()
    expect(screen.queryByRole('table', { name: /Rôles d’impact de la session 14\/11/ })).toBeNull()
  })

  it('sans le bloc, garde le titre et montre le cadre vide', () => {
    mockSquadContext({ confirmedGamertags: ['FriendA'], pageData: { main_player: 'test' } as unknown as TeammatesPageResponse })
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.getByText('Impact des coéquipiers')).toBeInTheDocument()
    expect(screen.getByTestId('squad-impact-sessions')).toBeInTheDocument()
    expect(screen.queryByTestId('squad-impact-row')).toBeNull()
  })

  // Fonction non déclarée par le titre (match.events.timeline absente) : la section entière, titre compris,
  // est absente ; les autres sections de la page restent.
  it('sans la capacité match.events.timeline, la section « Impact des coéquipiers » est absente', () => {
    capability.timeline = false
    mockSquadContext({
      confirmedGamertags: ['joueur_117', 'joueur_165'],
      pageData: { main_player: 'JGtm', squad_impact_sessions: impactSessionsFixture } as unknown as TeammatesPageResponse,
    })
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.queryByText('Impact des coéquipiers')).toBeNull()
    expect(screen.queryByTestId('squad-impact-sessions')).toBeNull()
    expect(screen.getByText(/^Médailles/)).toBeInTheDocument()
  })

  // L'objectif est parti dans l'onglet Emprise : même avec des matchs à objectif, Contributions
  // n'en monte plus aucune carte.
  it('ne monte plus la section « Objectif » (partie dans l’onglet Emprise)', () => {
    mockSquadContext({
      confirmedGamertags: ['Joueur90001', 'Joueur90002'],
      pageData: {
        main_player: 'JGtm',
        formes_retenues: block0709(),
        match_history: history0709(),
        squad_objective_history: history0709Evenings(),
      } as unknown as TeammatesPageResponse,
    })
    renderWithProviders(<SquadContributionsPage />)
    expect(screen.queryByTestId('squad-objective-section')).toBeNull()
    expect(screen.queryByText('Répartition de l’objectif dans l’escouade')).toBeNull()
  })
})
