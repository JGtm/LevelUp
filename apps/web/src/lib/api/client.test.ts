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
import { api, setApiTitleSlug, TITLE_MISMATCH_CODE } from './client'

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

/**
 * Garde anti-fuite cross-titre V72-29 (défense en profondeur décisive) : le serveur
 * échoie le titre résolu dans X-LevelUp-Title-Resolved ; le client REJETTE toute
 * réponse dont ce titre diverge du titre qu'il considère actif. Une réponse d'un autre
 * titre (course de bascule) est ainsi jetée AVANT d'entrer dans le cache TanStack.
 */
describe('api client — garde anti-fuite cross-titre (X-LevelUp-Title-Resolved)', () => {
  let fetchSpy: ReturnType<typeof vi.spyOn>

  function mockResolved(resolved: string | null, body: unknown = {}): void {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' }
    if (resolved !== null) headers['X-LevelUp-Title-Resolved'] = resolved
    fetchSpy.mockResolvedValue(
      new Response(JSON.stringify(body), { status: 200, headers }),
    )
  }

  beforeEach(() => {
    setApiTitleSlug(null)
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    fetchSpy = vi.spyOn(globalThis, 'fetch')
  })

  afterEach(() => {
    vi.restoreAllMocks()
    setApiTitleSlug(null)
  })

  it('REJETTE (title_mismatch) une réponse dont le titre résolu diverge du titre actif', async () => {
    setApiTitleSlug('halo_infinite')
    mockResolved('halo_5', { leaked: true })
    await expect(api.get('/players/p/pages/home')).rejects.toMatchObject({
      code: TITLE_MISMATCH_CODE,
    })
  })

  it('ACCEPTE une réponse dont le titre résolu correspond au titre actif', async () => {
    setApiTitleSlug('halo_infinite')
    mockResolved('halo_infinite', { ok: true })
    await expect(api.get('/players/p/pages/home')).resolves.toEqual({ ok: true })
  })

  it("n'applique PAS la garde sans header envoyé (page agnostique au boot, session autoritaire)", async () => {
    // slug null → aucun header X-LevelUp-Title : la session serveur fait autorité, on
    // ne rejette pas même si l'écho diverge (flux CLI / SSO / boot non gardés).
    mockResolved('halo_5', { ok: true })
    await expect(api.get('/home')).resolves.toEqual({ ok: true })
  })

  it("n'applique PAS la garde si le serveur n'échoie aucun titre (rétro-compat)", async () => {
    setApiTitleSlug('halo_infinite')
    mockResolved(null, { ok: true })
    await expect(api.get('/players/p/pages/home')).resolves.toEqual({ ok: true })
  })
})
