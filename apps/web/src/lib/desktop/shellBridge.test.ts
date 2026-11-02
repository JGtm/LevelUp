import { afterEach, describe, expect, it, vi } from 'vitest'
import { requestUpdateCheck, SHELL_CHECK_UPDATE_PATH } from './shellBridge'

describe('requestUpdateCheck', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("appelle le protocole embarqué de la coquille, sans lire la réponse ni quitter la page", async () => {
    const fetchSpy = vi.fn().mockResolvedValue(new Response(null))
    vi.stubGlobal('fetch', fetchSpy)

    await expect(requestUpdateCheck()).resolves.toBe(true)

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit]
    expect(url).toBe(`http://levelup-shell.localhost${SHELL_CHECK_UPDATE_PATH}`)
    expect(SHELL_CHECK_UPDATE_PATH).toBe('/check-update')
    expect(init.mode).toBe('no-cors')
    expect(init.cache).toBe('no-store')
  })

  it("rend faux quand la coquille est injoignable (page ouverte hors de l'application)", async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    await expect(requestUpdateCheck()).resolves.toBe(false)
  })
})
