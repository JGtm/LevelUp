/**
 * queries.delete.test.ts — mutation de suppression définitive d'un média
 * (v7.3 lot 2, item 3.1).
 *
 * Deux garanties :
 *  - ÉCHEC VISIBLE (garde anti-silence, héritée de l'item 1.5) : une suppression
 *    refusée doit le dire. Sans toast, l'utilisateur croit son fichier effacé
 *    alors qu'il est toujours sur le serveur.
 *  - INVALIDATION FRANCHE du préfixe mediaBase : le média disparaît des listes,
 *    donc compteurs, pagination et filtres changent — seul un refetch redonne un
 *    état cohérent (contrairement au like, qui écrit dans le cache).
 */
import { describe, expect, it, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { createElement } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useDeleteMedia } from './queries'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), info: vi.fn(), warning: vi.fn() },
}))

function makeClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })
}

function wrapperFor(client: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client, children })
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.mocked(toast.error).mockClear()
  vi.mocked(toast.success).mockClear()
})

const FILE_URL = '/api/v1/players/JGtm/media/files/JGtm/clip.mp4'

describe('useDeleteMedia', () => {
  it('appelle DELETE avec le file_path encodé en paramètre de requête', async () => {
    const del = vi.spyOn(api, 'delete').mockResolvedValue({
      file_path: FILE_URL,
      deleted: true,
      files_removed: 2,
    })

    const { result } = renderHook(() => useDeleteMedia('JGtm'), {
      wrapper: wrapperFor(makeClient()),
    })
    result.current.mutate(FILE_URL)

    await waitFor(() => expect(result.current.isSuccess).toBe(true), { timeout: 2000 })
    const [path] = vi.mocked(del).mock.calls[0] ?? []
    expect(String(path)).toContain('/players/JGtm/media?file_path=')
    // Le chemin contient des `/` : sans encodage il casserait la query string.
    expect(String(path)).toContain(encodeURIComponent(FILE_URL))
  })

  it('invalide le préfixe mediaBase après une suppression réussie', async () => {
    vi.spyOn(api, 'delete').mockResolvedValue({
      file_path: FILE_URL,
      deleted: true,
      files_removed: 1,
    })
    const client = makeClient()
    const invalidate = vi.spyOn(client, 'invalidateQueries')

    const { result } = renderHook(() => useDeleteMedia('JGtm'), {
      wrapper: wrapperFor(client),
    })
    result.current.mutate(FILE_URL)

    await waitFor(() => expect(result.current.isSuccess).toBe(true), { timeout: 2000 })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.mediaBase('JGtm') })
    expect(toast.success).toHaveBeenCalled()
  })

  it('affiche un toast d’erreur quand le serveur refuse la suppression', async () => {
    vi.spyOn(api, 'delete').mockRejectedValue(
      Object.assign(new Error('Seul le propriétaire de ce média ou un administrateur peut le supprimer.'), {
        code: 'media_delete_forbidden',
      }),
    )

    const { result } = renderHook(() => useDeleteMedia('JGtm'), {
      wrapper: wrapperFor(makeClient()),
    })
    result.current.mutate(FILE_URL)

    await waitFor(() => expect(toast.error).toHaveBeenCalled(), { timeout: 2000 })
    const [message] = vi.mocked(toast.error).mock.calls[0] ?? []
    expect(String(message)).toMatch(/supprimer/i)
    expect(toast.success).not.toHaveBeenCalled()
  })
})
