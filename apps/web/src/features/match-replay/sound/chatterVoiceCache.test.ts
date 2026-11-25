import { afterEach, describe, expect, it, vi } from 'vitest'

import { CHATTER_VOICE_CACHE_MAX, ChatterVoiceCache, voicesInWindow } from './chatterVoiceCache'
import type { ReplaySoundEvent } from './replaySoundVariants'

const buf = (name: string) => ({ name }) as unknown as AudioBuffer

/** Un chargeur témoin : chaque demande attend qu'on la résolve à la main, et note son signal. */
function loader() {
  const calls: Array<{ url: string; signal: AbortSignal; resolve: (b: AudioBuffer) => void; reject: (e: unknown) => void }> = []
  const load = vi.fn(
    (url: string, signal: AbortSignal) =>
      new Promise<AudioBuffer>((resolve, reject) => {
        calls.push({ url, signal, resolve, reject })
      }),
  )
  return { calls, load }
}

const flush = () => new Promise<void>((r) => setTimeout(r, 0))

afterEach(() => vi.restoreAllMocks())

describe('ChatterVoiceCache', () => {
  it('une seule requête par URL, même demandée plusieurs fois', async () => {
    const { calls, load } = loader()
    const c = new ChatterVoiceCache(load, 4)
    c.request('a')
    c.request('a')
    expect(calls).toHaveLength(1)
    calls[0].resolve(buf('a'))
    await flush()
    c.request('a') // déjà prête
    expect(load).toHaveBeenCalledTimes(1)
    expect(c.take('a')).toEqual(buf('a'))
  })

  it("rend `undefined` pour une voix pas encore prête : elle n'est pas jouée en retard", async () => {
    const { calls, load } = loader()
    const c = new ChatterVoiceCache(load, 4)
    c.request('a')
    expect(c.take('a')).toBeUndefined()
    expect(calls).toHaveLength(1) // `take` ne lance aucune requête
    expect(c.take('jamais-demandee')).toBeUndefined()
    expect(load).toHaveBeenCalledTimes(1)
  })

  it('borné : la plus ancienne voix est libérée à la (N+1)e', async () => {
    const { calls, load } = loader()
    const c = new ChatterVoiceCache(load, 3)
    for (const u of ['a', 'b', 'c', 'd']) c.request(u)
    for (const call of calls) call.resolve(buf(call.url))
    await flush()
    expect(c.size).toBe(3)
    expect(c.take('a')).toBeUndefined()
    expect(c.take('d')).toBeDefined()
  })

  it('une voix rejouée reste : l’éviction prend la moins récemment utilisée', async () => {
    const { calls, load } = loader()
    const c = new ChatterVoiceCache(load, 2)
    c.request('a')
    c.request('b')
    for (const call of calls) call.resolve(buf(call.url))
    await flush()
    c.take('a') // a redevient la plus récente
    c.request('c')
    calls[2].resolve(buf('c'))
    await flush()
    expect(c.take('b')).toBeUndefined()
    expect(c.take('a')).toBeDefined()
  })

  it('un échec est mémorisé : pas de nouvelle tentative', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    const { calls, load } = loader()
    const c = new ChatterVoiceCache(load, 2)
    c.request('a')
    calls[0].reject(new Error('404'))
    await flush()
    c.request('a')
    expect(load).toHaveBeenCalledTimes(1)
    expect(c.take('a')).toBeUndefined()
  })

  it('clear annule les requêtes en cours et libère tout ; la réponse arrivée après coup ne se range pas', async () => {
    const { calls, load } = loader()
    const c = new ChatterVoiceCache(load, 4)
    c.request('a')
    c.request('b')
    calls[0].resolve(buf('a'))
    await flush()
    c.clear()
    expect(calls[1].signal.aborted).toBe(true)
    expect(c.size).toBe(0)
    calls[1].resolve(buf('b'))
    await flush()
    expect(c.take('b')).toBeUndefined()
    expect(c.size).toBe(0)
  })

  it("un rejet après annulation ne laisse aucune trace (ni échec mémorisé, ni journal)", async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const { calls, load } = loader()
    const c = new ChatterVoiceCache(load, 4)
    c.request('a')
    c.clear()
    calls[0].reject(new DOMException('annulé', 'AbortError'))
    await flush()
    expect(warn).not.toHaveBeenCalled()
    c.request('a') // une nouvelle demande repart : l'annulation n'est pas un échec
    expect(load).toHaveBeenCalledTimes(2)
    warn.mockRestore()
  })

  it('la borne est de 16 voix', () => {
    expect(CHATTER_VOICE_CACHE_MAX).toBe(16)
  })
})

describe('voicesInWindow', () => {
  const ev = (ms: number, stem: string, chatter = true): ReplaySoundEvent => (chatter ? { ms, stem, chatter: true } : { ms, stem })
  const events = [ev(500, 'tir', false), ev(1000, 'a'), ev(3000, 'b'), ev(3100, 'a'), ev(6000, 'c'), ev(9000, 'd')]

  it("ne rend que les voix de réplique de la fenêtre ]de, de+avance], sans doublon", () => {
    expect(voicesInWindow(events, 1000, 4000)).toEqual(['b', 'a'])
  })

  it('une fenêtre vide ne demande rien', () => {
    expect(voicesInWindow(events, 6500, 1000)).toEqual([])
    expect(voicesInWindow([], 0, 4000)).toEqual([])
  })
})
