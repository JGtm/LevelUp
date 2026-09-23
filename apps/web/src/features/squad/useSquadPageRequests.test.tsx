/**
 * Tests — useSquadPageRequests (lot perf L4b, 2026-09-23) : la requête lourde de
 * l'Escouade n'est activée qu'une fois l'ancrage décidé sur la lecture légère des
 * sessions de la composition — donc elle part DÉJÀ sur la bonne session, une seule fois
 * par composition. Oracle : l'ordre et le contenu des requêtes reçues par MSW.
 *
 * Lot perf L9-web (2026-09-23, revue C) : le lien profond de l'accueil (relu au montage
 * par `useSquadDeepLink`) ne vaut que pour le PREMIER ancrage de sa composition ; sans
 * coéquipier, la lourde attend la légère dès qu'une session est pickée.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { server } from '@/test/setup'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import type { TeammatesQueryRequest } from '@/lib/api/types'
import { useSquadPageRequests } from './useSquadPageRequests'

const { searchMock } = vi.hoisted(() => ({
  searchMock: vi.fn<() => Record<string, unknown>>(() => ({})),
}))

// Le lien profond de l'accueil se lit dans l'URL au montage (`useSquadDeepLink`).
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useSearch: () => searchMock() }
})

const SANS_SESSION: string[] = []

function session(label: string) {
  return { label, started_at: '2026-09-20T19:00:00Z', ended_at: '2026-09-20T22:00:00Z', match_count: 2 }
}

/** Réponses de la lecture légère par (composition, option). */
const SESSIONS: Record<string, { composition_sessions: ReturnType<typeof session>[]; latest_composition_session: string }> = {
  'Alice|false': { composition_sessions: [session('A2 (4)'), session('A1 (2)')], latest_composition_session: 'A2 (4)' },
  // Sous l'option, A2 (un autre coéquipier connu était là) disparaît : la dernière est A1.
  'Alice|true': { composition_sessions: [session('A1 (2)')], latest_composition_session: 'A1 (2)' },
  'Alice,Bob|false': { composition_sessions: [session('AB1 (3)')], latest_composition_session: 'AB1 (3)' },
  '|false': { composition_sessions: [session('P1 (5)')], latest_composition_session: '' },
}

interface Requete {
  quoi: 'legere' | 'lourde'
  cle: string
  corps?: TeammatesQueryRequest
}

const journal: Requete[] = []
/** Réponses légères propres à un test (priment sur `SESSIONS`). */
let sessionsDuTest: typeof SESSIONS = {}
let retenirLegere: Promise<void> | null = null
let qcCourant: QueryClient | null = null

beforeEach(() => {
  localStorage.clear()
  useSquadFilterStore.getState().resetFilters()
  useSquadFilterStore.setState({ lastKnownLatestSessionId: null, isAutoSnappingToLatest: false })
  journal.length = 0
  sessionsDuTest = {}
  retenirLegere = null
  searchMock.mockReturnValue({})
  server.use(
    http.get('/api/v1/players/:playerSlug/pages/teammates/sessions', async ({ request }) => {
      const url = new URL(request.url)
      const cle = `${url.searchParams.get('teammates') ?? ''}|${url.searchParams.get('exact')}`
      journal.push({ quoi: 'legere', cle })
      if (retenirLegere) await retenirLegere
      return HttpResponse.json(sessionsDuTest[cle] ?? SESSIONS[cle] ?? { composition_sessions: [], latest_composition_session: '' })
    }),
    http.post('/api/v1/players/:playerSlug/pages/teammates', async ({ request }) => {
      const corps = (await request.json()) as TeammatesQueryRequest
      journal.push({ quoi: 'lourde', cle: (corps.selected_gamertags ?? []).join(','), corps })
      return HttpResponse.json({ options: [], teammates: [], total_matches: 0, session_labels: { solo: [], squad: [] }, friends_count: 0 })
    }),
  )
})

afterEach(() => {
  qcCourant?.clear()
  qcCourant = null
})

interface Props {
  gts: string[]
  exact: boolean
  ready: boolean
}

/** Le layout réduit à ce que le hook consomme : corps de la requête lourde lu dans le store. */
function useHarnais({ gts, exact, ready }: Props) {
  const { filterContext, filterContextHash } = useSquadFilterStore()
  const picked = filterContext.sessions?.picked_sessions ?? SANS_SESSION
  const request: TeammatesQueryRequest = {
    filters: { ...filterContext, match_context: 'squad' },
    selected_gamertags: gts.length > 0 ? gts : undefined,
    picked_squad_session_labels: picked.length > 0 ? picked : undefined,
    locale: 'fr',
    filter_exact_composition: exact,
  }
  return useSquadPageRequests({
    playerSlug: 'p',
    request,
    filterContextHash,
    selectedGts: gts,
    exactComposition: exact,
    teammatesReady: ready,
    pickedSquadSessionLabels: picked,
    applySessionLabels: (labels) =>
      useSquadFilterStore.getState().setSessions({ picked_sessions: labels, gap_minutes: 120 }),
  })
}

function monter(initial: Props) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qcCourant = qc
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  return renderHook((props: Props) => useHarnais(props), { wrapper, initialProps: initial })
}

async function laisserRetomber() {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 120))
  })
}

const lourdes = () => journal.filter((r) => r.quoi === 'lourde')

describe('useSquadPageRequests — la requête lourde attend la décision d ancrage', () => {
  it('composition initiale inconnue (verrou de montage L4a) : ni légère ni lourde ; connue : la légère, puis la lourde', async () => {
    const { rerender } = monter({ gts: ['Alice'], exact: false, ready: false })
    await laisserRetomber()
    expect(journal).toEqual([])

    rerender({ gts: ['Alice'], exact: false, ready: true })
    await waitFor(() => expect(lourdes()).toHaveLength(1))
    await laisserRetomber()
    expect(journal.map((r) => r.quoi)).toEqual(['legere', 'lourde'])
    expect(lourdes()[0].corps?.picked_squad_session_labels).toEqual(['A2 (4)'])
  })

  it('changer de composition : une légère pour la nouvelle, puis UNE lourde déjà sur SA dernière session', async () => {
    const { rerender, result } = monter({ gts: ['Alice'], exact: false, ready: true })
    await waitFor(() => expect(lourdes()).toHaveLength(1))
    await laisserRetomber()

    rerender({ gts: ['Alice', 'Bob'], exact: false, ready: true })
    await waitFor(() => expect(lourdes()).toHaveLength(2))
    await laisserRetomber()

    expect(journal.map((r) => `${r.quoi}:${r.cle}`)).toEqual([
      'legere:Alice|false',
      'lourde:Alice',
      'legere:Alice,Bob|false',
      'lourde:Alice,Bob',
    ])
    expect(lourdes()[1].corps?.picked_squad_session_labels).toEqual(['AB1 (3)'])
    expect(lourdes()[1].corps?.filters?.sessions?.picked_sessions).toEqual(['AB1 (3)'])
    // Le sélecteur lit les sessions de la NOUVELLE composition.
    expect(result.current.compositionSessions.map((s) => s.label)).toEqual(['AB1 (3)'])
  })

  it('basculer l option composition exacte : une légère, puis UNE lourde sur la dernière session de l option', async () => {
    const { rerender } = monter({ gts: ['Alice'], exact: false, ready: true })
    await waitFor(() => expect(lourdes()).toHaveLength(1))
    await laisserRetomber()

    rerender({ gts: ['Alice'], exact: true, ready: true })
    await waitFor(() => expect(lourdes()).toHaveLength(2))
    await laisserRetomber()

    expect(journal.map((r) => r.quoi)).toEqual(['legere', 'lourde', 'legere', 'lourde'])
    expect(lourdes()[1].corps?.filter_exact_composition).toBe(true)
    expect(lourdes()[1].corps?.picked_squad_session_labels).toEqual(['A1 (2)'])
  })

  it('sélection manuelle encore valide : la lourde part sur la session choisie, sans snap', async () => {
    useSquadFilterStore.getState().setSessions({ picked_sessions: ['A1 (2)'], gap_minutes: 120 })
    useSquadFilterStore.setState({ lastKnownLatestSessionId: 'A2 (4)' })
    monter({ gts: ['Alice'], exact: false, ready: true })
    await waitFor(() => expect(lourdes()).toHaveLength(1))
    await laisserRetomber()

    expect(journal.map((r) => r.quoi)).toEqual(['legere', 'lourde'])
    expect(lourdes()[0].corps?.picked_squad_session_labels).toEqual(['A1 (2)'])
    expect(useSquadFilterStore.getState().isAutoSnappingToLatest).toBe(false)
  })

  it('sans coéquipier ni session pickée : la lourde part sans attendre la légère (décision connue d avance)', async () => {
    let liberer: () => void = () => {}
    retenirLegere = new Promise<void>((resolve) => {
      liberer = resolve
    })
    const { result } = monter({ gts: [], exact: false, ready: true })
    await waitFor(() => expect(lourdes()).toHaveLength(1))
    // La légère est toujours en vol : la lourde ne l'a pas attendue.
    expect(journal.map((r) => r.quoi)).toEqual(['legere', 'lourde'])
    expect(result.current.compositionSessions).toEqual([])

    liberer()
    await waitFor(() => expect(result.current.compositionSessions.map((s) => s.label)).toEqual(['P1 (5)']))
    expect(lourdes()).toHaveLength(1)
  })
})

describe('useSquadPageRequests — sans coéquipier, session pickée (L9-web)', () => {
  it('la lourde attend la légère (réconciliation du suffixe « (N) ») : UNE lourde, sur la forme courante', async () => {
    let liberer: () => void = () => {}
    retenirLegere = new Promise<void>((resolve) => {
      liberer = resolve
    })
    useSquadFilterStore.getState().setSessions({ picked_sessions: ['P1 (3)'], gap_minutes: 120 })
    monter({ gts: [], exact: false, ready: true })
    await laisserRetomber()
    // Légère en vol : la lourde ne part pas sur le label au suffixe périmé.
    expect(journal.map((r) => r.quoi)).toEqual(['legere'])

    liberer()
    await waitFor(() => expect(lourdes()).toHaveLength(1))
    await laisserRetomber()
    expect(journal.map((r) => r.quoi)).toEqual(['legere', 'lourde'])
    expect(lourdes()[0].corps?.picked_squad_session_labels).toEqual(['P1 (5)'])
  })
})

describe('useSquadPageRequests — lien profond de l accueil (L9-web)', () => {
  function lienVers(session: string, teammates: string) {
    searchMock.mockReturnValue({ session, teammates })
    // État de montage posé par useSquadSessionSelection : la session du lien, dans le store.
    useSquadFilterStore.getState().setSessions({ picked_sessions: [session], gap_minutes: 120 })
  }

  it('le PREMIER ancrage garde la session du lien ; un ancrage ultérieur (nouvelle session) suit les règles ordinaires', async () => {
    lienVers('A1 (2)', 'Alice')
    monter({ gts: ['Alice'], exact: false, ready: true })
    await waitFor(() => expect(lourdes()).toHaveLength(1))
    await laisserRetomber()
    expect(lourdes()[0].corps?.picked_squad_session_labels).toEqual(['A1 (2)'])
    expect(useSquadFilterStore.getState().lastKnownLatestSessionId).toBe('A2 (4)')

    // Une nouvelle soirée de la composition arrive (sync) : relecture de la légère.
    sessionsDuTest['Alice|false'] = {
      composition_sessions: [session('A3 (1)'), session('A2 (4)'), session('A1 (2)')],
      latest_composition_session: 'A3 (1)',
    }
    const titre = useAppShellStore.getState().currentTitleSlug
    await act(async () => {
      await qcCourant?.invalidateQueries({ queryKey: queryKeys.compositionSessions('p', titre, ['Alice'], false) })
    })
    await waitFor(() => expect(lourdes()).toHaveLength(2))
    await laisserRetomber()
    expect(lourdes()[1].corps?.picked_squad_session_labels).toEqual(['A3 (1)'])
    expect(useSquadFilterStore.getState().isAutoSnappingToLatest).toBe(true)
  })

  it('lien d une AUTRE composition que la courante : règles ordinaires (snap sur la dernière)', async () => {
    lienVers('A1 (2)', 'Bob')
    monter({ gts: ['Alice'], exact: false, ready: true })
    await waitFor(() => expect(lourdes()).toHaveLength(1))
    await laisserRetomber()
    expect(lourdes()[0].corps?.picked_squad_session_labels).toEqual(['A2 (4)'])
  })
})
