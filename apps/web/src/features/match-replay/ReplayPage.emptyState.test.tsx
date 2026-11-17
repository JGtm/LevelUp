/**
 * Tests — le bloc « aucun rejeu » de la PAGE du rejeu : sans la capacité `can_generate_replays`, la phrase d'état vide
 * seule (ni requête d'état ni bouton) ; avec la capacité, l'action de génération sous la même phrase, et le clic
 * envoie la demande. Monte la vraie page, la vraie requête de l'artefact (404) et le vrai composant : seuls le
 * routeur et le serveur de réseau sont simulés.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'

import { testCapabilities } from '@/test/capabilities'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'

import { ReplayPage } from './ReplayPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    getRouteApi: () => ({
      useParams: () => ({ titleSlug: 'halo_infinite', playerSlug: 'jgtm', matchId: 'match-1' }),
      useSearch: () => ({}),
    }),
    useRouter: () => ({ history: { length: 1, back: vi.fn() }, navigate: vi.fn() }),
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

const API = '/api/v1/players/jgtm/matches/match-1/replay'
const EMPTY_FR = 'Aucun rejeu 2D disponible pour ce match.'

let reads = 0
let posts = 0

beforeEach(() => {
  reads = 0
  posts = 0
  server.use(
    http.get(API, () => HttpResponse.json({ code: 'not_found', message: 'aucun rejeu', retryable: false }, { status: 404 })),
    http.get(`${API}/generation`, () => {
      reads += 1
      return HttpResponse.json({ state: 'generatable' })
    }),
    http.post(`${API}/generate`, () => {
      posts += 1
      return HttpResponse.json({ created: true, status: { state: 'queued', position: 1 } }, { status: 202 })
    }),
  )
  act(() => useAppShellStore.setState({ locale: 'fr' }))
})

afterEach(() => {
  cleanup()
  act(() => useAppShellStore.setState({ capabilities: null }))
})

describe('ReplayPage — bloc « aucun rejeu »', () => {
  it('sans la capacité : la phrase d’état vide seule, aucune lecture de l’état, aucun bouton', async () => {
    act(() => useAppShellStore.setState({ capabilities: testCapabilities({ can_generate_replays: false }) }))
    renderWithProviders(<ReplayPage />)
    expect(await screen.findByText(EMPTY_FR)).toBeInTheDocument()
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(reads).toBe(0)
    expect(screen.queryByRole('button', { name: 'Générer le rejeu' })).not.toBeInTheDocument()
  })

  it('avec la capacité : le bouton sous la phrase ; le clic envoie la demande et montre l’attente', async () => {
    act(() => useAppShellStore.setState({ capabilities: testCapabilities({ can_generate_replays: true }) }))
    renderWithProviders(<ReplayPage />)
    expect(await screen.findByText(EMPTY_FR)).toBeInTheDocument()
    fireEvent.click(await screen.findByRole('button', { name: 'Générer le rejeu' }))
    expect(await screen.findByTestId('analysis-pending-slot')).toHaveTextContent('En file d’attente : position 1.')
    expect(posts).toBe(1)
    expect(screen.queryByRole('button', { name: 'Générer le rejeu' })).not.toBeInTheDocument()
  })
})
