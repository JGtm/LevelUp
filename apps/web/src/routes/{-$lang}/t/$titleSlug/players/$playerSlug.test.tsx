/**
 * Tests — layout joueur (PlayerLayout) : la résolution des filtres SOLO et le suivi
 * de la dernière session solo ne tournent que là où la barre solo est rendue
 * (lot perf L4a, D4.4, 2026-09-23).
 *
 * Mesure d'origine (`.ai/ETAT_DES_LIEUX_PERF_CHARGEMENTS_2026-09-23.md` C6) : un
 * POST /filters/resolve solo sur TOUTES les pages joueur, alors que seules les
 * pages Stats le lisent. Oracle : les POST reçus par MSW. NavL2 est neutralisée —
 * elle monte FilterOmnibar et son propre aperçu, hors sujet : on ne compte ici que
 * les résolutions de PlayerLayout.
 */
import type { ComponentType } from 'react'
import { describe, it, expect, beforeAll, beforeEach, vi } from 'vitest'
import { act, screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import type { FilterContextInput, FilterContextResolved, SessionOption } from '@/lib/api/types'

const paramsRef = { titleSlug: 'halo_infinite', playerSlug: 'p' }
const pathRef = { current: '/t/halo_infinite/players/p/home' }
// Abonnements au routeur (useRouterState) pendant le rendu : un composant qui rend
// `<Navigate>` ne doit JAMAIS s'y abonner (cf. le test « slug inconnu »).
const routerStateCalls = { n: 0 }

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    createFileRoute: () => (opts: Record<string, unknown>) => ({
      ...opts,
      useParams: () => paramsRef,
    }),
    useRouterState: (opts?: { select?: (s: { location: { pathname: string } }) => unknown }) => {
      routerStateCalls.n += 1
      const state = { location: { pathname: pathRef.current } }
      return opts?.select ? opts.select(state) : state
    },
    Navigate: () => <div data-testid="player-navigate" />,
    Outlet: () => <div data-testid="player-outlet" />,
  }
})

vi.mock('@/components/shell/NavL2', () => ({ NavL2: () => null }))

// Import APRÈS les mocks (le module lit createFileRoute au chargement).
import { Route } from './$playerSlug'

// autoCodeSplitting : le composant de route est un wrapper lazy — préchargé une fois.
const routeComponent = (
  Route as unknown as { component: ComponentType & { preload?: () => Promise<unknown> } }
).component
const PlayerLayout: ComponentType = routeComponent

beforeAll(async () => {
  await routeComponent.preload?.()
})

function soloSession(sessionId: string, label: string): SessionOption {
  return {
    session_id: sessionId,
    label,
    match_count: 3,
    match_count_filtered: 3,
    is_squad: false,
    started_at_utc: '2026-09-20T19:00:00Z',
    ended_at_utc: '2026-09-20T22:00:00Z',
  }
}

function resolved(all: SessionOption[]): FilterContextResolved {
  return {
    effective: {
      filter_mode: 'period',
      period: { start_date: null, end_date: null },
      sessions: {
        picked_sessions: [],
        gap_minutes: 120,
        picked_session_label: null,
        picked_solo_session_label: null,
        picked_squad_session_label: null,
      },
      cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
    },
    available_options: { experience_types: [], playlists: [], modes: [], maps: [] },
    session_options: { all_sessions: all, solo_labels: [], squad_labels: [] },
    counts: { total_matches_before_filters: 3, total_matches_after_filters: 3 },
    period_presets: [],
  }
}

const corpsResolve: FilterContextInput[] = []
let reponseResolve: FilterContextResolved = resolved([soloSession('s-new', 'NEW (2)')])

beforeEach(() => {
  corpsResolve.length = 0
  reponseResolve = resolved([soloSession('s-new', 'NEW (2)')])
  pathRef.current = '/t/halo_infinite/players/p/home'
  paramsRef.playerSlug = 'p'
  routerStateCalls.n = 0
  useAppShellStore.setState({
    isBootstrapped: true,
    locale: 'fr',
    currentTitleSlug: 'halo_infinite',
    availablePlayers: [
      { player_slug: 'p', gamertag: 'P', xuid: '1', waypoint_player: 'P', is_demo: false },
    ] as ReturnType<typeof useAppShellStore.getState>['availablePlayers'],
    activeSyncJobId: null,
  })
  useSoloFilterStore.getState().resetFilters()
  useSoloFilterStore.setState({ lastKnownLatestSessionId: null, isAutoSnappingToLatest: false })
  server.use(
    http.post('/api/v1/players/:playerSlug/filters/resolve', async ({ request }) => {
      corpsResolve.push((await request.json()) as FilterContextInput)
      return HttpResponse.json(reponseResolve)
    }),
  )
})

async function monterSur(pathname: string) {
  pathRef.current = pathname
  renderWithProviders(<PlayerLayout />)
  expect(screen.getByTestId('player-outlet')).toBeInTheDocument()
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 150))
  })
}

describe('PlayerLayout — résolution solo seulement sous la barre solo (D4.4)', () => {
  it.each([
    '/t/halo_infinite/players/p/squad/synergies',
    '/t/halo_infinite/players/p/career',
    '/t/halo_infinite/players/p/home',
    '/t/halo_infinite/players/p/stats/synthesis',
  ])('%s : aucune résolution solo', async (pathname) => {
    await monterSur(pathname)
    expect(corpsResolve).toHaveLength(0)
  })

  it('/stats/timeseries : UNE résolution solo, corps inchangé (sans match_context)', async () => {
    // Déjà ancré sur la dernière session : pas de snap, donc pas de 2e résolution
    // (un snap change le commité et relance légitimement la résolution).
    useSoloFilterStore.getState().setSessions({ picked_sessions: ['NEW (2)'], gap_minutes: 120 })
    useSoloFilterStore.setState({ isAutoSnappingToLatest: true, lastKnownLatestSessionId: 's-new' })
    await monterSur('/t/halo_infinite/players/p/stats/timeseries')
    expect(corpsResolve).toHaveLength(1)
    expect(corpsResolve[0].match_context).toBeUndefined()
  })

  it('hors Stats, un résolu resté dans le store ne fait pas snapper le filtre solo', async () => {
    useSoloFilterStore.getState().setResolvedContext(resolved([soloSession('s-old', 'OLD (3)')]))
    await monterSur('/t/halo_infinite/players/p/home')
    expect(useSoloFilterStore.getState().filterContext.sessions?.picked_sessions ?? []).toEqual([])
    expect(useSoloFilterStore.getState().isAutoSnappingToLatest).toBe(false)
  })

  it('sur Stats, le suivi n agit que sur le résolu de la requête courante (jamais sur un résolu périmé)', async () => {
    // Résolu périmé laissé dans le store (autre page, autre joueur) ; le serveur dit NEW.
    useSoloFilterStore.getState().setResolvedContext(resolved([soloSession('s-old', 'OLD (3)')]))
    const vus: string[][] = []
    const stop = useSoloFilterStore.subscribe((s) => vus.push(s.filterContext.sessions?.picked_sessions ?? []))
    try {
      await monterSur('/t/halo_infinite/players/p/stats/timeseries')
    } finally {
      stop()
    }
    expect(useSoloFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['NEW (2)'])
    expect(vus.some((picked) => picked.includes('OLD (3)'))).toBe(false)
  })
})

describe('PlayerLayout — slug inconnu : redirection SANS abonnement au routeur', () => {
  // Régression du 2026-09-23 (CI E2E, PR vers main, après le lot perf L4a) : PlayerLayout
  // s'abonnait au routeur (useRouterState) ET rendait `<Navigate params={{...}}>` sur un
  // slug inconnu. Navigate re-navigue dès que l'identité de ses props change (objet
  // `params` neuf à chaque rendu) et la navigation re-rend l'abonné : boucle, onglet
  // figé. L'abonnement vit désormais dans un enfant monté seulement sur un slug valide.
  it('slug inconnu : <Navigate> rendu, aucun useRouterState, aucune résolution', async () => {
    paramsRef.playerSlug = 'inconnu'
    renderWithProviders(<PlayerLayout />)
    expect(screen.getByTestId('player-navigate')).toBeInTheDocument()
    expect(screen.queryByTestId('player-outlet')).not.toBeInTheDocument()
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 150))
    })
    expect(routerStateCalls.n).toBe(0)
    expect(corpsResolve).toHaveLength(0)
  })

  it('slug connu : l abonnement au routeur existe (témoin du garde-fou)', async () => {
    await monterSur('/t/halo_infinite/players/p/stats/timeseries')
    expect(routerStateCalls.n).toBeGreaterThan(0)
  })
})
