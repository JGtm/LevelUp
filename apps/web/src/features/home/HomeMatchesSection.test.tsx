/**
 * Tests — HomeMatchesSection (matchs récents / favoris de l'Accueil) : montage limité à la tête,
 * page suivante à 5 tuiles de la fin, navigation sans liste d'identifiants, favoris, favori
 * basculé sans recharger la tête.
 */
import type { ReactNode } from 'react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { server } from '@/test/setup'
import type { HomeMatchesPage, RecentMatchItem } from '@/lib/api/types'

const navigateSpy = vi.fn()

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useNavigate: () => navigateSpy }
})

// jsdom ne mesure rien : le carrousel est remplacé par un témoin qui expose ses props.
vi.mock('@/components/ui/carousel', () => ({
  Carousel: ({
    children,
    onNearEnd,
    loadingMore,
  }: {
    children: ReactNode
    onNearEnd?: () => void
    loadingMore?: boolean
  }) => (
    <div data-testid="carousel" data-loading={loadingMore ? 'oui' : 'non'} data-near-end={onNearEnd ? 'oui' : 'non'}>
      <button type="button" onClick={onNearEnd}>
        fin-proche
      </button>
      {children}
    </div>
  ),
  CarouselItem: ({ children }: { children: ReactNode }) => <div data-testid="carousel-item">{children}</div>,
}))

import { HomeMatchesSection } from './HomeMatchesSection'

const tile = (id: string, fav = false): RecentMatchItem =>
  ({
    match_id: id,
    detail: `Slayer · FDA 1.00 · 50 - 40`,
    started_at: '2026-10-20T20:00:00Z',
    outcome_tone: 'win',
    is_favorite: fav,
    map_ui: `Carte ${id}`,
    mode_ui: 'Slayer',
  }) as unknown as RecentMatchItem

const headTiles = Array.from({ length: 30 }, (_, i) => tile(`h${String(i).padStart(2, '0')}`))

function setup(options: { favorites?: RecentMatchItem[]; headNext?: string | undefined } = {}) {
  const calls: string[] = []
  const favorites = options.favorites ?? []
  server.use(
    http.get('/api/v1/players/:slug/pages/home/matches', ({ request }) => {
      const url = new URL(request.url)
      calls.push(url.search)
      if (url.searchParams.get('kind') === 'favorites') {
        return HttpResponse.json({ items: favorites, has_more: false } satisfies HomeMatchesPage)
      }
      return HttpResponse.json({
        items: [tile('p1a'), tile('p1b')],
        has_more: false,
      } satisfies HomeMatchesPage)
    }),
  )
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const head = {
    recent_matches: headTiles,
    recent_matches_has_more: 'headNext' in options ? !!options.headNext : true,
    recent_matches_next_before: 'headNext' in options ? options.headNext : 'curseur0',
  }
  render(
    <QueryClientProvider client={client}>
      <HomeMatchesSection playerSlug="test-player" head={head} />
    </QueryClientProvider>,
  )
  return { calls, client }
}

beforeEach(() => {
  navigateSpy.mockReset()
  sessionStorage.clear()
})

describe('HomeMatchesSection', () => {
  it('monte seulement la tête (30 tuiles) et ne demande aucune page tant que la fin est loin', async () => {
    const { calls } = setup()
    expect(await screen.findAllByTestId('carousel-item')).toHaveLength(30)
    // Seule la lecture des favoris part au montage.
    await waitFor(() => expect(calls).toEqual(['?kind=favorites']))
  })

  it('à 5 tuiles de la fin : demande la page suivante avec le curseur de la tête, et l\'ajoute', async () => {
    const { calls } = setup()
    await screen.findAllByTestId('carousel-item')
    fireEvent.click(screen.getByText('fin-proche'))
    await waitFor(() => expect(screen.getAllByTestId('carousel-item')).toHaveLength(32))
    expect(calls.some((c) => c.includes('before=curseur0') && c.includes('limit=30'))).toBe(true)
    // Plus de suite annoncée : le carrousel ne demande plus rien.
    expect(screen.getByTestId('carousel').dataset.nearEnd).toBe('non')
  })

  it('sans suite annoncée par la tête : le carrousel ne demande rien', async () => {
    setup({ headNext: undefined })
    await screen.findAllByTestId('carousel-item')
    expect(screen.getByTestId('carousel').dataset.nearEnd).toBe('non')
  })

  it('ouvrir un match récent : AUCUNE liste d\'identifiants, un descripteur « récents » (voisins par l\'API)', async () => {
    setup()
    const cards = await screen.findAllByTestId('carousel-item')
    // Le clic de la carte est porté par la tuile elle-même (pas par son bouton favori).
    fireEvent.click(cards[3].querySelector('button.group') as HTMLElement)
    await waitFor(() => expect(navigateSpy).toHaveBeenCalled())
    const call = navigateSpy.mock.calls[0][0] as {
      params: { matchId: string }
      state: (prev: Record<string, unknown>) => { matchNavContext: { matchIds: string[]; contextDescriptor: unknown; source: string } }
    }
    expect(call.params.matchId).toBe('h03')
    const ctx = call.state({}).matchNavContext
    expect(ctx.matchIds).toEqual([])
    expect(ctx.contextDescriptor).toEqual({ kind: 'recent' })
    expect(ctx.source).toBe('home_recent')
  })

  it('onglet Favoris : compte affiché, tuiles servies par leur propre lecture', async () => {
    setup({ favorites: [tile('f1', true), tile('f2', true)] })
    await screen.findAllByTestId('carousel-item')
    await waitFor(() => expect(screen.getByText('2')).toBeInTheDocument())
    fireEvent.click(screen.getByRole('button', { name: /Favoris/ }))
    expect(await screen.findAllByTestId('carousel-item')).toHaveLength(2)
  })

  it('favori basculé : la tuile change sur place, SEULE la liste des favoris est relue', async () => {
    let patched: unknown = null
    server.use(
      http.patch('/api/v1/players/:slug/matches/:matchId/favorite', async ({ request, params }) => {
        patched = { id: params.matchId, body: await request.json() }
        return HttpResponse.json({ player_slug: 'test-player', match_id: params.matchId, favorited: true })
      }),
    )
    const { calls, client } = setup()
    await screen.findAllByTestId('carousel-item')
    await waitFor(() => expect(calls).toEqual(['?kind=favorites']))
    // La tête est dans le cache de l'accueil (comme en vrai) : on y pose les tuiles affichées.
    const headKey = ['home', 'test-player', 'halo_infinite', 'fr']
    client.setQueryData(headKey, { recent_matches: headTiles })

    fireEvent.click(screen.getAllByRole('button', { name: 'Ajouter aux favoris' })[0])
    await waitFor(() => expect(patched).toEqual({ id: 'h00', body: { favorited: true } }))
    // 2e lecture des favoris, et rien d'autre (ni la tête, ni une page).
    await waitFor(() => expect(calls).toEqual(['?kind=favorites', '?kind=favorites']))
    const cached = client.getQueryData<{ recent_matches: RecentMatchItem[] }>(headKey)
    expect(cached?.recent_matches[0].is_favorite).toBe(true)
    expect(cached?.recent_matches[1].is_favorite).toBe(false)
  })
})
