/**
 * Enchaînement réel carrousel + chargement : le VRAI `Carousel` monté avec `useHomeTiles`.
 * Rappel de fin -> page demandée (indicateur de chargement) -> tuiles ajoutées -> plus de rappel tant
 * que la fin est loin -> dernière page -> plus de rappel du tout.
 */
import type { ReactNode } from 'react'
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { act, render, screen, waitFor, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { HomeMatchesPage, RecentMatchItem } from '@/lib/api/types'
import { mockTrackMetrics, type TrackMetricsState } from '@/test/trackMetrics'

const hoisted = vi.hoisted(() => ({
  pages: {} as Record<string, unknown>,
  calls: [] as string[],
  gate: null as Promise<void> | null,
}))

vi.mock('@/lib/api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/client')>()),
  api: {
    get: vi.fn(async (url: string) => {
      hoisted.calls.push(url)
      if (hoisted.gate) await hoisted.gate
      return hoisted.pages[new URL(url, 'http://x').searchParams.get('before') ?? '']
    }),
  },
}))

import { Carousel, CarouselItem } from '@/components/ui/carousel'
import { useHomeTiles } from './queries'

const tile = (id: string) => ({ match_id: id }) as RecentMatchItem
const page = (ids: string[], next?: string): HomeMatchesPage => ({
  items: ids.map(tile),
  has_more: !!next,
  next_before: next,
})

function Harness() {
  const tiles = useHomeTiles('chief', {
    recent_matches: [tile('h1'), tile('h2')],
    recent_matches_has_more: true,
    recent_matches_next_before: 'c0',
  })
  return (
    <Carousel onNearEnd={tiles.hasMore ? tiles.fetchMore : undefined} loadingMore={tiles.loadingMore}>
      {tiles.tiles.map((m) => (
        <CarouselItem key={m.match_id}>
          <span data-testid="tuile">{m.match_id}</span>
        </CarouselItem>
      ))}
    </Carousel>
  )
}

// La piste est près de la fin (2 tuiles de 300 px sur 1 000 px visibles : tout est à portée) jusqu'à
// ce que la page arrive ; alors elle s'allonge et la fin s'éloigne.
const metrics: TrackMetricsState = { scrollWidth: 600, clientWidth: 1000, scrollLeft: 0 }
let unmock: (() => void) | null = null

beforeEach(() => {
  hoisted.calls = []
  hoisted.gate = null
  Object.assign(metrics, { scrollWidth: 600, clientWidth: 1000, scrollLeft: 0 })
  unmock = mockTrackMetrics(metrics)
})
afterEach(() => unmock?.())

function mount(): ReactNode {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return (
    <QueryClientProvider client={client}>
      <Harness />
    </QueryClientProvider>
  )
}

describe('Carousel + useHomeTiles', () => {
  it('rappel de fin -> chargement -> tuiles ajoutées -> fin éloignée : une seule demande par page', async () => {
    let release: () => void = () => {}
    hoisted.gate = new Promise<void>((resolve) => { release = resolve })
    hoisted.pages = { c0: page(['p1', 'p2', 'p3'], 'c1'), c1: page(['p4'], undefined) }
    render(mount())

    // Montage : la fin est à portée -> la page du curseur de la tête est demandée, indicateur affiché.
    await waitFor(() => expect(hoisted.calls).toHaveLength(1))
    expect(hoisted.calls[0]).toContain('before=c0')
    expect(await screen.findByRole('status')).toBeInTheDocument()
    expect(screen.getAllByTestId('tuile')).toHaveLength(2)

    // Pas de nouvelle demande tant que la page est en vol, même si le carrousel se re-mesure.
    act(() => {
      fireEvent.scroll(document.querySelector('.overflow-x-auto') as HTMLElement)
    })
    expect(hoisted.calls).toHaveLength(1)

    // La page arrive : cinq tuiles, plus d'indicateur ; la piste s'allonge, la fin s'éloigne.
    metrics.scrollWidth = 9000
    hoisted.gate = null
    release()
    await waitFor(() => expect(screen.getAllByTestId('tuile')).toHaveLength(5))
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    act(() => {
      fireEvent.scroll(document.querySelector('.overflow-x-auto') as HTMLElement)
    })
    expect(hoisted.calls).toHaveLength(1)

    // L'utilisateur défile à 5 tuiles de la fin : la page suivante (la dernière) est demandée.
    metrics.scrollLeft = 7000
    act(() => {
      fireEvent.scroll(document.querySelector('.overflow-x-auto') as HTMLElement)
    })
    await waitFor(() => expect(screen.getAllByTestId('tuile')).toHaveLength(6))
    expect(hoisted.calls).toHaveLength(2)
    expect(hoisted.calls[1]).toContain('before=c1')

    // Dernière page : plus de suite, plus aucun rappel même tout au bout.
    metrics.scrollLeft = 5000
    act(() => {
      fireEvent.scroll(document.querySelector('.overflow-x-auto') as HTMLElement)
    })
    expect(hoisted.calls).toHaveLength(2)
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })
})
