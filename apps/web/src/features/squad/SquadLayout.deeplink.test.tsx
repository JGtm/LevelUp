/**
 * Test — deep-link accueil → /squad (SquadLayout).
 *
 * Card session escouade → /squad?session=…&teammates=… : SquadLayout doit, au
 * montage, pré-sélectionner la composition (amis de la session) ET épingler la
 * session. L'endpoint teammates est mis en erreur ici pour neutraliser le
 * ré-ancrage composition (qui dépend des données) et isoler la consommation du
 * deep-link — le ré-ancrage a ses propres tests (decideCompositionReanchor).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
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
  // Neutralise le ré-ancrage : sans données teammates, l'effet retourne tôt.
  server.use(
    http.post('/api/v1/players/:playerSlug/pages/teammates', () =>
      HttpResponse.json({ error: 'isolate-consume' }, { status: 500 }),
    ),
  )
})

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
})
