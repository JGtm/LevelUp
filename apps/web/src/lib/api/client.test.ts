/**
 * Non-régression fuite directionnelle H5 → Infinite (PLAN_TITLE_HEADER_LEAK_2026-07).
 *
 * Le client DOIT affirmer le titre courant sur CHAQUE requête via X-LevelUp-Title,
 * y compris pour le titre par défaut halo_infinite. Sans header, le backend retombe
 * sur la session serveur (partagée entre onglets) → une session périmée sur un autre
 * titre fait fuiter ses données sur le titre affiché (resolveTitleSlug : header >
 * session > défaut). Le cœur du test : le défaut halo_infinite porte bien le header.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api, setApiTitleSlug } from './client'

describe('api client — header X-LevelUp-Title', () => {
  let fetchSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    // Court-circuite MSW (le fetch patché) : on n'observe que les headers sortants.
    fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({}), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })

  afterEach(() => {
    fetchSpy.mockRestore() // rend la main au fetch patché par MSW
    setApiTitleSlug('halo_infinite') // restaure le défaut (état module partagé)
  })

  /** Headers de la dernière requête fetch, normalisés (accès insensible à la casse). */
  function lastRequestHeaders(): Headers {
    const calls = fetchSpy.mock.calls
    expect(calls.length).toBeGreaterThan(0)
    const init = calls[calls.length - 1][1] as RequestInit
    return new Headers(init.headers)
  }

  it('affirme halo_infinite (titre par défaut) — cœur de la non-régression', async () => {
    setApiTitleSlug('halo_infinite')
    await api.get('/home')
    expect(lastRequestHeaders().get('X-LevelUp-Title')).toBe('halo_infinite')
  })

  it('affirme halo_5 (titre non-défaut) — comportement inchangé', async () => {
    setApiTitleSlug('halo_5')
    await api.get('/home')
    expect(lastRequestHeaders().get('X-LevelUp-Title')).toBe('halo_5')
  })
})
