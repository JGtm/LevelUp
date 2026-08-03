/**
 * queries.likeToast.test.ts — garde anti-silence du like média (plan v7.3 lot 2,
 * item 1.5).
 *
 * La mutation de like est optimiste : le cœur se remplit avant la réponse du
 * serveur. Sans notification d'échec, un refus serveur (401 sans session, 503
 * base occupée, 404 média inconnu) se traduit par un cœur qui revient en arrière
 * sans la moindre explication — c'est précisément la forme « cassée sans
 * message » du bug d'origine.
 */
import { describe, expect, it, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { createElement } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/api/client'
import { useToggleMediaLike } from './queries'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), info: vi.fn(), warning: vi.fn() },
}))

function makeWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client, children })
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.mocked(toast.error).mockClear()
})

describe('useToggleMediaLike — échec visible', () => {
  it('affiche un toast d’erreur quand le serveur refuse le like', async () => {
    vi.spyOn(api, 'patch').mockRejectedValue(
      Object.assign(new Error('Aucun joueur courant en session'), {
        code: 'like_requires_session',
      }),
    )

    const { result } = renderHook(() => useToggleMediaLike('JGtm'), {
      wrapper: makeWrapper(),
    })

    result.current.mutate({
      file_path: '/api/v1/players/JGtm/media/files/JGtm/clip.mp4',
      liked: true,
    })

    await waitFor(() => expect(toast.error).toHaveBeenCalled(), { timeout: 2000 })
    const [message] = vi.mocked(toast.error).mock.calls[0] ?? []
    expect(String(message)).toMatch(/like/i)
  })

  it('n’affiche aucun toast quand le like réussit', async () => {
    vi.spyOn(api, 'patch').mockResolvedValue({
      file_path: '/api/v1/players/JGtm/media/files/JGtm/clip.mp4',
      liked: true,
      like_count: 1,
    })

    const { result } = renderHook(() => useToggleMediaLike('JGtm'), {
      wrapper: makeWrapper(),
    })

    result.current.mutate({
      file_path: '/api/v1/players/JGtm/media/files/JGtm/clip.mp4',
      liked: true,
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true), { timeout: 2000 })
    expect(toast.error).not.toHaveBeenCalled()
  })
})
