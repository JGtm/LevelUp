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
import { LOCK_FORCED_CODE, useInstanceLock } from './useInstanceLock'

const mutate = vi.fn()
const state: { isPending: boolean; isError: boolean; error: unknown } = {
  isPending: false,
  isError: false,
  error: null,
}

vi.mock('@/features/settings/queries', () => ({
  useUpdateSettings: () => ({
    mutate,
    isPending: state.isPending,
    isError: state.isError,
    error: state.error,
  }),
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
  state.error = null
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

describe('useInstanceLock — code d erreur (verrou force par l environnement)', () => {
  it('expose instance_lock_forced quand le serveur refuse en 409', () => {
    state.isError = true
    state.error = { code: LOCK_FORCED_CODE, message: 'force' }
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useInstanceLock(), { wrapper })
    expect(result.current.isError).toBe(true)
    expect(result.current.errorCode).toBe(LOCK_FORCED_CODE)
  })

  it('errorCode est null sans erreur, et null sur une erreur sans code', () => {
    const { wrapper } = makeWrapper()
    const ok = renderHook(() => useInstanceLock(), { wrapper })
    expect(ok.result.current.errorCode).toBeNull()

    state.isError = true
    state.error = new Error('reseau')
    const ko = renderHook(() => useInstanceLock(), { wrapper })
    expect(ko.result.current.errorCode).toBeNull()
  })
})
