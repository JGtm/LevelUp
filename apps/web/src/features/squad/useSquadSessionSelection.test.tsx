/**
 * Tests — useSquadSessionSelection (lot perf L4a, 2026-09-23 : D4.1 et D4.2).
 *
 * L'oracle est l'HISTORIQUE DES RENDUS du hook : la requête teammates part au
 * commit d'un rendu où `teammatesReady` est vrai, avec la composition et la session
 * de CE rendu. Prouver « pas de requête à vide » et « lien profond avant la
 * première requête », c'est donc prouver qu'AUCUN rendu prêt ne porte un contexte
 * provisoire (composition vide en attendant les amis, composition restaurée que
 * le lien profond remplace, session pas encore migrée).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { server } from '@/test/setup'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { useSquadSessionSelection } from './useSquadSessionSelection'

const { searchMock } = vi.hoisted(() => ({
  searchMock: vi.fn<() => Record<string, unknown>>(() => ({})),
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useSearch: () => searchMock() }
})

const FRIENDS_URL = '/api/v1/players/:playerSlug/friends'

function friendsReply(gamertags: string[]) {
  return HttpResponse.json({ xuid: 'x-p', gamertags, can_edit: true })
}

interface Rendu {
  ready: boolean
  selected: string[]
  picked: string[]
}

function monter() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  const rendus: Rendu[] = []
  const utils = renderHook(
    () => {
      const r = useSquadSessionSelection('p')
      rendus.push({ ready: r.teammatesReady, selected: r.selectedGts, picked: r.pickedSquadSessionLabels })
      return r
    },
    { wrapper },
  )
  return { ...utils, rendus }
}

/** Rendus où la requête teammates aurait le droit de partir. */
const prets = (rendus: Rendu[]) => rendus.filter((r) => r.ready)

beforeEach(() => {
  localStorage.clear()
  useSquadFilterStore.getState().resetFilters()
  useSquadFilterStore.setState({ lastKnownLatestSessionId: null, isAutoSnappingToLatest: false })
  searchMock.mockReturnValue({})
})

describe('useSquadSessionSelection — pas de requête à vide (D4.2)', () => {
  it('amis pas encore arrivés : jamais prêt ; arrivés : prêt AVEC la composition des amis', async () => {
    let libererAmis: () => void = () => {}
    const amisEnAttente = new Promise<void>((resolve) => {
      libererAmis = resolve
    })
    server.use(
      http.get(FRIENDS_URL, async () => {
        await amisEnAttente
        return friendsReply(['Alice', 'Bob'])
      }),
    )
    const { result, rendus } = monter()
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 50))
    })
    expect(result.current.teammatesReady).toBe(false)

    libererAmis()
    await waitFor(() => expect(result.current.teammatesReady).toBe(true))
    expect(result.current.selectedGts).toEqual(['Alice', 'Bob'])
    // Aucun rendu prêt avec une composition vide (la requête « sans coéquipier »
    // de 6 s mesurée le 2026-09-23 ne peut plus partir).
    expect(prets(rendus).every((r) => r.selected.length > 0)).toBe(true)
    expect(JSON.parse(localStorage.getItem('squad-teammates-p') ?? '[]')).toEqual(['Alice', 'Bob'])
  })

  it('liste d amis RÉSOLUE et vide : prêt sans coéquipier (exploration légitime)', async () => {
    server.use(http.get(FRIENDS_URL, () => friendsReply([])))
    const { result } = monter()
    await waitFor(() => expect(result.current.teammatesReady).toBe(true))
    expect(result.current.selectedGts).toEqual([])
  })

  it('lecture des amis en échec : vaut liste vide, la page ne reste pas bloquée', async () => {
    server.use(http.get(FRIENDS_URL, () => HttpResponse.json({ error: 'boom' }, { status: 500 })))
    const { result } = monter()
    await waitFor(() => expect(result.current.teammatesReady).toBe(true))
    expect(result.current.selectedGts).toEqual([])
  })

  it('composition restaurée : prêt sans attendre la liste d amis', async () => {
    localStorage.setItem('squad-teammates-p', JSON.stringify(['Carol']))
    server.use(
      http.get(FRIENDS_URL, async () => {
        await new Promise(() => {}) // jamais de réponse
        return friendsReply([])
      }),
    )
    const { result } = monter()
    await waitFor(() => expect(result.current.teammatesReady).toBe(true))
    expect(result.current.selectedGts).toEqual(['Carol'])
  })

  it('vider la composition est un CHOIX : reste prêt même si le joueur a des amis', async () => {
    server.use(http.get(FRIENDS_URL, () => friendsReply(['Alice'])))
    const { result } = monter()
    await waitFor(() => expect(result.current.selectedGts).toEqual(['Alice']))
    act(() => result.current.setSelectedGts([]))
    expect(result.current.selectedGts).toEqual([])
    expect(result.current.teammatesReady).toBe(true)
  })
})

describe('useSquadSessionSelection — lien profond de l accueil', () => {
  it('pose composition ET session avant le premier rendu prêt, malgré un état restauré concurrent', async () => {
    searchMock.mockReturnValue({ session: 'S1 (4)', teammates: 'Alice, Bob' })
    localStorage.setItem('squad-teammates-p', JSON.stringify(['Carol']))
    useSquadFilterStore.getState().setSessions({ picked_sessions: ['S0 (9)'], gap_minutes: 120 })

    const { result, rendus } = monter()
    await waitFor(() => expect(result.current.teammatesReady).toBe(true))

    expect(prets(rendus).length).toBeGreaterThan(0)
    for (const r of prets(rendus)) {
      expect(r.selected).toEqual(['Alice', 'Bob'])
      expect(r.picked).toEqual(['S1 (4)'])
    }
    expect(useSquadFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['S1 (4)'])
    expect(JSON.parse(localStorage.getItem('squad-teammates-p') ?? '[]')).toEqual(['Alice', 'Bob'])
  })
})

describe('useSquadSessionSelection — source unique de la session (D4.1)', () => {
  it('les sessions pickées SONT celles du store, et applySessionLabels écrit dans le store', async () => {
    localStorage.setItem('squad-teammates-p', JSON.stringify(['Alice']))
    useSquadFilterStore.getState().setSessions({ picked_sessions: [], gap_minutes: 60 })
    const { result } = monter()
    await waitFor(() => expect(result.current.teammatesReady).toBe(true))

    act(() => result.current.applySessionLabels(['S2 (3)']))
    const sessions = useSquadFilterStore.getState().filterContext.sessions
    expect(sessions?.picked_sessions).toEqual(['S2 (3)'])
    expect(sessions?.gap_minutes).toBe(60) // réglage de regroupement préservé
    expect(result.current.pickedSquadSessionLabels).toEqual(['S2 (3)'])

    // Un changement venu d'ailleurs (rail, snap) est vu tel quel : pas de copie locale.
    act(() => {
      useSquadFilterStore.getState().autoSnapToLatestSession({ session_id: 'S3 (1)', label: 'S3 (1)' }, true)
    })
    expect(result.current.pickedSquadSessionLabels).toEqual(['S3 (1)'])
    expect(localStorage.getItem('squad-sessions-p')).toBeNull()
  })

  it('migre l ancienne clé locale dans un store VIDE avant le premier rendu prêt, puis la retire', async () => {
    localStorage.setItem('squad-teammates-p', JSON.stringify(['Alice']))
    localStorage.setItem('squad-sessions-p', JSON.stringify(['S1 (2)']))
    const { result, rendus } = monter()
    await waitFor(() => expect(result.current.teammatesReady).toBe(true))

    for (const r of prets(rendus)) expect(r.picked).toEqual(['S1 (2)'])
    expect(useSquadFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['S1 (2)'])
    expect(localStorage.getItem('squad-sessions-p')).toBeNull()
  })

  it('store déjà renseigné : il prime, l ancienne clé est retirée sans être appliquée', async () => {
    localStorage.setItem('squad-teammates-p', JSON.stringify(['Alice']))
    useSquadFilterStore.getState().setSessions({ picked_sessions: ['S9 (4)'], gap_minutes: 120 })
    localStorage.setItem('squad-sessions-p', JSON.stringify(['S1 (2)']))
    const { result } = monter()
    await waitFor(() => expect(result.current.teammatesReady).toBe(true))

    expect(result.current.pickedSquadSessionLabels).toEqual(['S9 (4)'])
    expect(localStorage.getItem('squad-sessions-p')).toBeNull()
  })

  it('ancienne clé illisible : retirée, ignorée', async () => {
    localStorage.setItem('squad-teammates-p', JSON.stringify(['Alice']))
    localStorage.setItem('squad-sessions-p', '{pas du json')
    const { result } = monter()
    await waitFor(() => expect(result.current.teammatesReady).toBe(true))

    expect(result.current.pickedSquadSessionLabels).toEqual([])
    expect(localStorage.getItem('squad-sessions-p')).toBeNull()
  })
})
