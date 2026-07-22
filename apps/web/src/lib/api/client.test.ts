/**
 * Non-régression fuite directionnelle inter-titres (PLAN_TITLE_HEADER_LEAK_2026-07),
 * dans les DEUX sens :
 *  - APRÈS hydratation, le client DOIT affirmer le titre courant sur CHAQUE requête
 *    via X-LevelUp-Title, y compris le défaut halo_infinite. Sans header, le backend
 *    retombe sur la session serveur (partagée entre onglets) → une session périmée
 *    sur un autre titre fait fuiter ses données sur le titre affiché (fuite
 *    H5→Infinite).
 *  - AVANT hydratation (slug null), le client NE DOIT PAS envoyer de header : une
 *    requête title-scoped partie avant le bootstrap forcerait sinon halo_infinite et
 *    écraserait une session halo_5 sous une clé de cache sans titre (fuite inverse
 *    Infinite→session).
 * resolveTitleSlug côté backend : header > session > défaut.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api, setApiTitleSlug } from './client'

describe('api client — header X-LevelUp-Title', () => {
  let fetchSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    // État module partagé entre tests : ramener le client à l'état de boot
    // (slug null → aucun header, session serveur autoritaire) avant chaque test.
    setApiTitleSlug(null)
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
    setApiTitleSlug(null) // ramène le client à l'état de boot (état module partagé)
  })

  /** Headers de la dernière requête fetch, normalisés (accès insensible à la casse). */
  function lastRequestHeaders(): Headers {
    const calls = fetchSpy.mock.calls
    expect(calls.length).toBeGreaterThan(0)
    const init = calls[calls.length - 1][1] as RequestInit
    return new Headers(init.headers)
  }

  it('avant toute hydratation (slug null), aucune requête ne porte X-LevelUp-Title', async () => {
    // État de boot (setApiTitleSlug(null) en beforeEach) : la session serveur reste
    // autoritaire, aucun header ne doit partir (fuite inverse Infinite→session).
    await api.get('/home')
    expect(lastRequestHeaders().has('X-LevelUp-Title')).toBe(false)
  })

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
