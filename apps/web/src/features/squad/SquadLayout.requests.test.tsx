/**
 * Tests — les REQUÊTES de la page Escouade (lot perf L4a, 2026-09-23).
 *
 * Séquence mesurée le 2026-09-23 (`.ai/ETAT_DES_LIEUX_PERF_CHARGEMENTS_2026-09-23.md`
 * §1.2, C3) : une requête teammates SANS coéquipier au premier passage (la
 * composition arrive par GET /friends), puis, à chaque snap ou clic du rail, DEUX
 * requêtes (deux sources de vérité de la session, deux clés : 8,2 s à vide +
 * 26,8 s), et un aperçu /filters/resolve relancé même sans filtre en attente.
 *
 * Oracles : les corps reçus par MSW (ce qui est réellement parti) et le cache
 * TanStack Query — avec le gcTime par défaut il garde TOUTES les clés construites,
 * même abandonnées au rendu suivant : « une seule nouvelle clé » s'y lit.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { server } from '@/test/setup'
import { queryKeys } from '@/lib/query/keys'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import type { FilterContextInput, TeammatesQueryRequest } from '@/lib/api/types'
import { SquadLayout } from './SquadLayout'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'p' }),
    useSearch: () => ({}),
    useMatchRoute: () => () => null,
    useNavigate: () => vi.fn(),
    Outlet: () => <div data-testid="contenu-onglet" />,
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

const TEAMMATES_BASE = {
  options: [],
  teammates: [],
  total_matches: 3,
  session_labels: { solo: [], squad: [] },
  friends_count: 0,
  composition_sessions: [],
  latest_composition_session: '',
  match_history: [],
}

function session(label: string) {
  return { label, started_at: '2026-09-20T19:00:00Z', ended_at: '2026-09-20T22:00:00Z', match_count: 3 }
}

/** Une playlist cochable (pour mettre un filtre EN ATTENTE) + les presets. */
const RESOLVE = {
  effective: {
    filter_mode: 'period',
    period: { start_date: null, end_date: null },
    sessions: { picked_sessions: [], gap_minutes: 120 },
    cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
  },
  available_options: {
    experience_types: [],
    playlists: [{ label: 'Arene classee', value: 'ranked-arena', count: 3 }],
    modes: [],
    maps: [],
  },
  period_presets: [
    { preset_id: '7d', count: 1 },
    { preset_id: '30d', count: 3 },
    { preset_id: '90d', count: 3 },
    { preset_id: 'all', count: 3 },
  ],
}

const corpsTeammates: TeammatesQueryRequest[] = []
const corpsResolve: FilterContextInput[] = []
let reponseTeammates: Record<string, unknown> = TEAMMATES_BASE
let qcCourant: QueryClient | null = null

beforeEach(() => {
  localStorage.clear()
  useSquadFilterStore.getState().resetFilters()
  useSquadFilterStore.setState({ lastKnownLatestSessionId: null, isAutoSnappingToLatest: false })
  useAppShellStore.setState({ locale: 'fr' })
  corpsTeammates.length = 0
  corpsResolve.length = 0
  reponseTeammates = TEAMMATES_BASE
  server.use(
    http.post('/api/v1/players/:playerSlug/pages/teammates', async ({ request }) => {
      corpsTeammates.push((await request.json()) as TeammatesQueryRequest)
      return HttpResponse.json(reponseTeammates)
    }),
    http.post('/api/v1/players/:playerSlug/filters/resolve', async ({ request }) => {
      corpsResolve.push((await request.json()) as FilterContextInput)
      return HttpResponse.json(RESOLVE)
    }),
  )
})

afterEach(() => {
  qcCourant?.clear()
  qcCourant = null
})

function monter(): QueryClient {
  // gcTime par défaut (et non 0) : une clé abandonnée reste lisible dans le cache.
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qcCourant = qc
  render(
    <QueryClientProvider client={qc}>
      <SquadLayout />
    </QueryClientProvider>,
  )
  return qc
}

/** Laisse retomber effets et requêtes en vol (MSW répond en quelques ms). */
async function laisserRetomber() {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 150))
  })
}

/** Toutes les clés teammates jamais construites par le cache. */
const clesTeammates = (qc: QueryClient) =>
  qc.getQueryCache().findAll({ queryKey: queryKeys.teammatesAll }).map((q) => q.queryKey)

describe('Escouade — pas de requête teammates à vide (D4.2)', () => {
  it('amis en attente : aucune requête, « Chargement… » (pas l état vide) ; puis UNE requête avec la composition', async () => {
    let libererAmis: () => void = () => {}
    const amis = new Promise<void>((resolve) => {
      libererAmis = resolve
    })
    server.use(
      http.get('/api/v1/players/:playerSlug/friends', async () => {
        await amis
        return HttpResponse.json({ xuid: 'x-p', gamertags: ['Alice'], can_edit: true })
      }),
    )
    monter()
    await laisserRetomber()
    expect(corpsTeammates).toHaveLength(0)
    expect(screen.getByText('Chargement…')).toBeInTheDocument()

    libererAmis()
    await waitFor(() => expect(corpsTeammates).toHaveLength(1))
    await laisserRetomber()
    expect(corpsTeammates).toHaveLength(1)
    expect(corpsTeammates[0].selected_gamertags).toEqual(['Alice'])
  })

  it('amis résolus et vides : UNE requête, sans coéquipier (exploration)', async () => {
    monter() // handler par défaut : liste d'amis vide
    await waitFor(() => expect(corpsTeammates).toHaveLength(1))
    await laisserRetomber()
    expect(corpsTeammates).toHaveLength(1)
    expect(corpsTeammates[0].selected_gamertags).toBeUndefined()
  })
})

describe('Escouade — une seule source de vérité pour la session (D4.1)', () => {
  it('snap sur la dernière session de la composition : UNE nouvelle clé, UNE requête, pas d intermédiaire', async () => {
    localStorage.setItem('squad-teammates-p', JSON.stringify(['Alice']))
    reponseTeammates = {
      ...TEAMMATES_BASE,
      composition_sessions: [session('S2 (3)'), session('S1 (2)')],
      latest_composition_session: 'S2 (3)',
    }
    const qc = monter()
    await waitFor(() =>
      expect(useSquadFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['S2 (3)']),
    )
    await waitFor(() => expect(corpsTeammates).toHaveLength(2))
    await laisserRetomber()

    expect(useSquadFilterStore.getState().isAutoSnappingToLatest).toBe(true)
    // Avant : tout l'historique. Après le snap : la session, dans les DEUX champs.
    expect(corpsTeammates).toHaveLength(2)
    expect(corpsTeammates[0].picked_squad_session_labels).toBeUndefined()
    expect(corpsTeammates[1].picked_squad_session_labels).toEqual(['S2 (3)'])
    expect(corpsTeammates[1].filters?.sessions?.picked_sessions).toEqual(['S2 (3)'])
    // Deux clés en tout : celle d'avant et celle du snap. Une requête intermédiaire
    // (filters.sessions posé, picked_squad_session_labels pas encore) en ferait trois.
    expect(clesTeammates(qc)).toHaveLength(2)
    // Et pas d'aperçu : deux résolutions, celles du commité (montage, snap).
    expect(corpsResolve).toHaveLength(2)
  })

  it('changement de session (clic du rail, sélecteur) : UNE clé, UNE requête, champs alignés', async () => {
    localStorage.setItem('squad-teammates-p', JSON.stringify(['Alice']))
    reponseTeammates = {
      ...TEAMMATES_BASE,
      composition_sessions: [session('S2 (3)'), session('S1 (2)')],
      latest_composition_session: 'S2 (3)',
    }
    // Déjà ancré sur la dernière : aucun snap au montage.
    useSquadFilterStore.getState().setSessions({ picked_sessions: ['S2 (3)'], gap_minutes: 120 })
    useSquadFilterStore.setState({ lastKnownLatestSessionId: 'S2 (3)' })
    const qc = monter()
    await waitFor(() => expect(corpsTeammates).toHaveLength(1))
    await laisserRetomber()

    act(() => {
      useSquadFilterStore.getState().setSessions({ picked_sessions: ['S1 (2)'], gap_minutes: 120 })
    })
    await waitFor(() => expect(corpsTeammates).toHaveLength(2))
    await laisserRetomber()

    expect(corpsTeammates).toHaveLength(2)
    expect(corpsTeammates[1].picked_squad_session_labels).toEqual(['S1 (2)'])
    expect(corpsTeammates[1].filters?.sessions?.picked_sessions).toEqual(['S1 (2)'])
    expect(clesTeammates(qc)).toHaveLength(2)
  })
})

describe('Escouade — aperçu et résolu (D4.3)', () => {
  it('aucun aperçu sans filtre en attente ; résolu du store en match_context squad ; aperçu dès qu un filtre attend', async () => {
    localStorage.setItem('squad-teammates-p', JSON.stringify(['Alice']))
    const user = userEvent.setup()
    monter()
    await waitFor(() => expect(corpsTeammates).toHaveLength(1))
    await laisserRetomber()

    // Montage : UNE résolution — celle du commité, en squad.
    expect(corpsResolve).toHaveLength(1)
    expect(corpsResolve[0].match_context).toBe('squad')

    // Session changée : le commité change, une résolution ; toujours aucun aperçu.
    act(() => {
      useSquadFilterStore.getState().setSessions({ picked_sessions: ['S1 (2)'], gap_minutes: 120 })
    })
    await laisserRetomber()
    expect(corpsResolve).toHaveLength(2)
    expect(corpsResolve[1].sessions?.picked_sessions).toEqual(['S1 (2)'])

    // Une playlist cochée EN ATTENTE : l'aperçu part (et lui seul).
    await user.click(screen.getByRole('button', { name: /Filtres/ }))
    await user.click(await screen.findByLabelText(/Arene classee/))
    await laisserRetomber()
    expect(corpsResolve).toHaveLength(3)
    expect(corpsResolve[2].cascade?.playlists).toEqual(['ranked-arena'])
    expect(corpsResolve[2].match_context).toBe('squad')
    // Rien n'est commité : la requête lourde n'est pas repartie.
    expect(corpsTeammates).toHaveLength(2)
  })
})
