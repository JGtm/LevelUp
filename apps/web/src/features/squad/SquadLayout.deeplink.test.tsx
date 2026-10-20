/**
 * Test — deep-link accueil → /squad (SquadLayout).
 *
 * Card session escouade → /squad?session=…&teammates=… : SquadLayout doit, au
 * montage, pré-sélectionner la composition (amis de la session) ET épingler la
 * session. L'endpoint teammates est mis en erreur ici pour neutraliser le
 * ré-ancrage composition (qui dépend des données) et isoler la consommation du
 * deep-link — le ré-ancrage a ses propres tests (decideCompositionReanchor).
 *
 * Lot perf L4a (2026-09-23, D4.2) : la composition et la session du lien sont
 * posées AVANT la première requête — même quand une composition restaurée et une
 * session persistée concurrentes existent (ce que vérifie le 3e cas).
 *
 * Lot perf L4b (2026-09-23) : l'ancrage se décide désormais sur la lecture légère des
 * sessions (GET /pages/teammates/sessions). Elle est mise en erreur elle aussi, pour la
 * même raison : sans données de sessions, pas de ré-ancrage, et la requête lourde part
 * aussitôt (repli L4a) avec l'état posé par le lien.
 *
 * Lot perf L9-web (2026-09-23, revue C) : dernier bloc, lecture légère EN SUCCÈS. Un
 * lien vers une session ANCIENNE (carrousel de l'accueil) était écrasé par le snap sur
 * la dernière session de la composition, jamais ancrée — le store ne connaît pas la
 * composition du lien (`lastKnownLatestSessionId` nul, ou celui d'une autre).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import type { FilterContextInput, TeammatesQueryRequest } from '@/lib/api/types'
import { SquadLayout } from './SquadLayout'

const { searchMock } = vi.hoisted(() => ({
  searchMock: vi.fn<() => Record<string, unknown>>(() => ({ session: 'S1', teammates: 'Alice,Bob' })),
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'p' }),
    useSearch: () => searchMock(),
    useMatchRoute: () => () => null,
    useNavigate: () => vi.fn(),
    Outlet: () => null,
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

beforeEach(() => {
  localStorage.clear()
  useSquadFilterStore.getState().resetFilters()
  searchMock.mockReturnValue({ session: 'S1', teammates: 'Alice,Bob' })
  // Neutralise le ré-ancrage : sans données de sessions (légères ou lourdes), l'effet
  // retourne tôt.
  server.use(
    http.post('/api/v1/players/:playerSlug/pages/teammates', () =>
      HttpResponse.json({ error: 'isolate-consume' }, { status: 500 }),
    ),
    http.get('/api/v1/players/:playerSlug/pages/teammates/sessions', () =>
      HttpResponse.json({ error: 'isolate-consume' }, { status: 500 }),
    ),
  )
})

const SESSIONS_DE_LA_COMPOSITION = {
  composition_sessions: [
    { label: 'S2 (3)', started_at: '2026-09-20T19:00:00Z', ended_at: '2026-09-20T22:00:00Z', match_count: 3 },
    { label: 'S1 (2)', started_at: '2026-09-19T19:00:00Z', ended_at: '2026-09-19T21:00:00Z', match_count: 2 },
  ],
  latest_composition_session: 'S2 (3)',
}

/** Légère et lourde EN SUCCÈS ; rend les corps des requêtes lourdes et le journal d'arrivée. */
function servirLaComposition() {
  const corps: TeammatesQueryRequest[] = []
  const journal: string[] = []
  server.use(
    http.get('/api/v1/players/:playerSlug/pages/teammates/sessions', () => {
      journal.push('legere')
      return HttpResponse.json(SESSIONS_DE_LA_COMPOSITION)
    }),
    http.post('/api/v1/players/:playerSlug/pages/teammates', async ({ request }) => {
      journal.push('lourde')
      corps.push((await request.json()) as TeammatesQueryRequest)
      return HttpResponse.json({
        options: [], teammates: [], total_matches: 2, session_labels: { solo: [], squad: [] },
        friends_count: 0, match_history: [], ...SESSIONS_DE_LA_COMPOSITION,
      })
    }),
  )
  return { corps, journal }
}

describe('SquadLayout — deep-link accueil (card session escouade)', () => {
  it('pré-sélectionne les amis de la session et épingle la session', async () => {
    renderWithProviders(<SquadLayout />)

    await waitFor(() => {
      expect(useSquadFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['S1'])
    })
    expect(JSON.parse(localStorage.getItem('squad-teammates-p') ?? '[]')).toEqual(['Alice', 'Bob'])
  })

  it('sans deep-link (?session absent), ne touche pas la sélection', async () => {
    searchMock.mockReturnValue({})
    renderWithProviders(<SquadLayout />)
    await waitFor(() => expect(document.body).toBeTruthy())
    expect(useSquadFilterStore.getState().filterContext.sessions?.picked_sessions ?? []).toEqual([])
    expect(localStorage.getItem('squad-teammates-p')).toBeNull()
  })

  it('la PREMIÈRE requête porte déjà la composition et la session du lien (état restauré concurrent)', async () => {
    localStorage.setItem('squad-teammates-p', JSON.stringify(['Carol']))
    useSquadFilterStore.getState().setSessions({ picked_sessions: ['S0 (9)'], gap_minutes: 120 })
    const corps: TeammatesQueryRequest[] = []
    const resolutions: FilterContextInput[] = []
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/teammates', async ({ request }) => {
        corps.push((await request.json()) as TeammatesQueryRequest)
        return HttpResponse.json({ error: 'isolate-consume' }, { status: 500 })
      }),
      http.post('/api/v1/players/:playerSlug/filters/resolve', async ({ request }) => {
        resolutions.push((await request.json()) as FilterContextInput)
        return HttpResponse.json({ error: 'isolate-consume' }, { status: 500 })
      }),
    )
    renderWithProviders(<SquadLayout />)

    await waitFor(() => expect(corps.length).toBeGreaterThan(0))
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 150))
    })
    expect(corps).toHaveLength(1)
    expect(corps[0].selected_gamertags).toEqual(['Alice', 'Bob'])
    expect(corps[0].picked_squad_session_labels).toEqual(['S1'])
    expect(corps[0].filters?.sessions?.picked_sessions).toEqual(['S1'])
    // Même verrou pour la résolution escouade : aucune ne part sur la session restaurée.
    expect(resolutions.length).toBeGreaterThan(0)
    expect(resolutions.every((r) => (r.sessions?.picked_sessions ?? []).join() === 'S1')).toBe(true)
  })
})

describe('SquadLayout — lien profond vers une session ANCIENNE, lecture légère en succès (L9-web)', () => {
  it.each([
    ['aucun ancrage connu', null],
    ['ancrage d une AUTRE composition', 'X9 (5)'],
  ])('la session du lien est gardée : UNE lourde, sur elle ; dernière de la composition mémorisée (%s)', async (_nom, ancrage) => {
    searchMock.mockReturnValue({ session: 'S1 (2)', teammates: 'Alice' })
    useSquadFilterStore.setState({ lastKnownLatestSessionId: ancrage, isAutoSnappingToLatest: false })
    const { corps, journal } = servirLaComposition()
    renderWithProviders(<SquadLayout />)

    await waitFor(() => expect(corps).toHaveLength(1))
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 150))
    })
    expect(journal).toEqual(['legere', 'lourde'])
    expect(corps[0].selected_gamertags).toEqual(['Alice'])
    expect(corps[0].picked_squad_session_labels).toEqual(['S1 (2)'])
    expect(corps[0].filters?.sessions?.picked_sessions).toEqual(['S1 (2)'])
    const etat = useSquadFilterStore.getState()
    expect(etat.filterContext.sessions?.picked_sessions).toEqual(['S1 (2)'])
    // La dernière session de la composition est mémorisée : un rechargement ne snappe pas.
    expect(etat.lastKnownLatestSessionId).toBe('S2 (3)')
    expect(etat.isAutoSnappingToLatest).toBe(false)
  })

  it('session du lien inconnue de la composition : snap sur sa dernière session, avant l UNIQUE lourde', async () => {
    searchMock.mockReturnValue({ session: 'S0 (1)', teammates: 'Alice' })
    useSquadFilterStore.setState({ lastKnownLatestSessionId: null, isAutoSnappingToLatest: false })
    const { corps, journal } = servirLaComposition()
    renderWithProviders(<SquadLayout />)

    await waitFor(() => expect(corps).toHaveLength(1))
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 150))
    })
    expect(journal).toEqual(['legere', 'lourde'])
    expect(corps[0].picked_squad_session_labels).toEqual(['S2 (3)'])
    expect(useSquadFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['S2 (3)'])
  })
})
