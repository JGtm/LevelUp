/**
 * useInstanceLock.test.tsx — la logique de l'interrupteur « Instance fermée ».
 *
 * Trois choses à tenir : l'état vient de /bootstrap (store appShell), la bascule
 * envoie `instance_locked` au PATCH /settings, et le succès invalide `bootstrap`
 * — sans quoi l'interrupteur afficherait l'ancienne valeur jusqu'au prochain
 * rechargement de page.
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { useInstanceLock } from './useInstanceLock'

const mutate = vi.fn()
const state = { isPending: false, isError: false }

vi.mock('@/features/settings/queries', () => ({
  useUpdateSettings: () => ({ mutate, isPending: state.isPending, isError: state.isError }),
}))

function makeWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  const invalidate = vi.spyOn(qc, 'invalidateQueries').mockResolvedValue(undefined)
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  return { wrapper, invalidate }
}

beforeEach(() => {
  mutate.mockReset()
  state.isPending = false
  state.isError = false
  useAppShellStore.setState({ instanceLocked: false })
})

describe('useInstanceLock', () => {
  it('lit le verrou depuis le store appShell (alimenté par /bootstrap)', () => {
    useAppShellStore.setState({ instanceLocked: true })
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useInstanceLock(), { wrapper })
    expect(result.current.locked).toBe(true)
  })

  it('bascule avec instance_locked, et invalide bootstrap au succès', () => {
    const { wrapper, invalidate } = makeWrapper()
    const { result } = renderHook(() => useInstanceLock(), { wrapper })

    result.current.setLocked(true)

    expect(mutate).toHaveBeenCalledTimes(1)
    expect(mutate.mock.calls[0][0]).toEqual({ instance_locked: true })

    // Le rappel de succès est ce qui remet l'affichage en phase avec le serveur.
    const options = mutate.mock.calls[0][1] as { onSuccess: () => void }
    options.onSuccess()
    expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.bootstrap })
  })

  it('remonte l envoi en cours et l échec', () => {
    state.isPending = true
    state.isError = true
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useInstanceLock(), { wrapper })
    expect(result.current.isPending).toBe(true)
    expect(result.current.isError).toBe(true)
  })
})
