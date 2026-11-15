/**
 * Tests — `HomeFirstSync` relit l'accueil UNE fois par job terminé : la page parente passe une nouvelle fonction de
 * relecture à chaque rendu, et sans garde par identifiant de job chaque rendu relirait l'accueil.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { HomeFirstSync } from './HomeFirstSync'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useNavigate: () => vi.fn() }
})

const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
function Providers({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>
}

function serveJob(jobId: string) {
  server.use(
    http.get('/api/v1/players/:playerSlug/sync/status', ({ params }) =>
      HttpResponse.json({
        player_slug: params.playerSlug,
        syncing: false,
        cooldown_retry_after_s: 0,
        job: { job_id: jobId, job_type: 'initial_sync', status: 'succeeded', player_slug: params.playerSlug },
      }),
    ),
  )
}

describe('HomeFirstSync — relecture de l’accueil', () => {
  afterEach(() => {
    client.clear()
    act(() => useAppShellStore.setState({ currentPlayer: null, writablePlayerSlugs: null }))
  })

  it('une fois par job, même quand la page parente passe une nouvelle fonction', async () => {
    serveJob('job-1')
    const first = vi.fn()
    const { rerender } = render(<HomeFirstSync playerSlug="p1" refetch={first} />, { wrapper: Providers })
    expect(await screen.findByText('Aucun match trouvé')).toBeInTheDocument()
    expect(first).toHaveBeenCalledTimes(1)

    const second = vi.fn()
    rerender(<HomeFirstSync playerSlug="p1" refetch={second} />)
    await new Promise((resolve) => setTimeout(resolve, 100))
    expect(second).not.toHaveBeenCalled()
  })

  it('un nouveau job terminé relit de nouveau l’accueil', async () => {
    serveJob('job-1')
    const refetch = vi.fn()
    render(<HomeFirstSync playerSlug="p1" refetch={refetch} />, { wrapper: Providers })
    await screen.findByText('Aucun match trouvé')
    expect(refetch).toHaveBeenCalledTimes(1)

    serveJob('job-2')
    await act(() => client.invalidateQueries())
    await vi.waitFor(() => expect(refetch).toHaveBeenCalledTimes(2))
  })
})
