/**
 * Tests — useReliefRender : le journal des replis sur le plan. Une vue quittée (ou une carte changée) pendant le
 * chargement ferme le calculateur ; la lecture en cours rejette, mais aucun repli n'a lieu : rien ne doit être
 * journalisé. Une vraie panne, vue encore montée, l'est une fois.
 */
import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { fourBlocksScene } from './relief/reliefScene.testutil'
import { useReliefRender } from './useReliefRender'

const mocks = vi.hoisted(() => ({ read: vi.fn(), create: vi.fn() }))
vi.mock('./relief', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./relief')>()),
  readReliefImage: mocks.read,
  createReliefComputer: mocks.create,
}))

const DATA = fourBlocksScene().data
const BLOB = new Blob()
const REQUEST = { turns: 0 as const, cut: null, base: null, lines: false, masks: false }

/** Une promesse que le test résout ou rejette à son heure. */
function deferred<T>() {
  let resolve!: (v: T) => void
  let reject!: (e: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

const computer = () => ({ load: vi.fn(), render: vi.fn(), dispose: vi.fn() })

beforeEach(() => {
  mocks.read.mockReset()
  mocks.create.mockReset()
  vi.spyOn(console, 'warn').mockImplementation(() => {})
})
afterEach(() => vi.restoreAllMocks())

describe('useReliefRender : le journal des replis', () => {
  it('un démontage pendant le chargement ne journalise aucun repli', async () => {
    const pending = deferred<never>()
    mocks.read.mockReturnValue(pending.promise)
    mocks.create.mockReturnValue(computer())
    const { unmount } = renderHook(() => useReliefRender(DATA, BLOB, REQUEST))
    unmount()
    pending.reject(new Error('calculateur fermé'))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(console.warn).not.toHaveBeenCalled()
  })

  it('un changement de carte pendant le chargement ne journalise pas non plus l’ancienne', async () => {
    const first = deferred<never>()
    mocks.read.mockReturnValueOnce(first.promise).mockReturnValue(new Promise(() => {}))
    mocks.create.mockImplementation(computer)
    const { rerender, result } = renderHook(({ data }) => useReliefRender(data, BLOB, REQUEST), { initialProps: { data: DATA } })
    rerender({ data: fourBlocksScene().data })
    first.reject(new Error('calculateur fermé'))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(console.warn).not.toHaveBeenCalled()
    expect(result.current.status).toBe('loading')
  })

  it('une vraie panne du chargement, vue montée, est journalisée une fois et l’état passe à failed', async () => {
    mocks.read.mockRejectedValue(new Error('image illisible'))
    mocks.create.mockReturnValue(computer())
    const { result } = renderHook(() => useReliefRender(DATA, BLOB, REQUEST))
    await waitFor(() => expect(result.current.status).toBe('failed'))
    expect(console.warn).toHaveBeenCalledTimes(1)
    expect(vi.mocked(console.warn).mock.calls[0][0]).toContain('chargement du relief impossible, repli sur le plan : image illisible')
  })

  it('un rendu qui rejette après le démontage n’est pas journalisé', async () => {
    const render = deferred<never>()
    const c = computer()
    c.load.mockResolvedValue({ width: 40, height: 10, hasMesh: true })
    c.render.mockReturnValue(render.promise)
    mocks.read.mockResolvedValue(fourBlocksScene().image)
    mocks.create.mockReturnValue(c)
    const { result, unmount } = renderHook(() => useReliefRender(DATA, BLOB, REQUEST))
    await waitFor(() => expect(result.current.status).toBe('ready'))
    await waitFor(() => expect(c.render).toHaveBeenCalled())
    unmount()
    render.reject(new Error('calculateur fermé'))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(console.warn).not.toHaveBeenCalled()
  })
})
