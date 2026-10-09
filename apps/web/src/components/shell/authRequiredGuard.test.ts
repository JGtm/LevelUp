import { describe, it, expect, beforeEach } from 'vitest'
import type { BootstrapResponse } from '@/lib/api/types'
import {
  AUTH_RELOAD_MAX_IN_WINDOW,
  AUTH_RELOAD_STORAGE_KEY,
  AUTH_RELOAD_WINDOW_MS,
  claimAuthReload,
  decideOnAuthRequired,
} from './authRequiredGuard'

function bootstrap(username: string | null): BootstrapResponse {
  return { current_username: username } as unknown as BootstrapResponse
}

describe('claimAuthReload — plafond qui survit au rechargement', () => {
  beforeEach(() => window.sessionStorage.clear())

  it('permet AUTH_RELOAD_MAX_IN_WINDOW rechargements dans la fenêtre, refuse le suivant', () => {
    const t0 = 1_000_000
    for (let i = 0; i < AUTH_RELOAD_MAX_IN_WINDOW; i++) {
      expect(claimAuthReload(window.sessionStorage, t0 + i)).toBe(true)
    }
    expect(claimAuthReload(window.sessionStorage, t0 + 10)).toBe(false)
  })

  it('les instants vivent dans le stockage : un « nouveau chargement » relit le compte', () => {
    const t0 = 2_000_000
    window.sessionStorage.setItem(
      AUTH_RELOAD_STORAGE_KEY,
      JSON.stringify(Array.from({ length: AUTH_RELOAD_MAX_IN_WINDOW }, (_, i) => t0 + i)),
    )
    expect(claimAuthReload(window.sessionStorage, t0 + 100)).toBe(false)
  })

  it('hors fenêtre, le compte repart', () => {
    const t0 = 3_000_000
    for (let i = 0; i < AUTH_RELOAD_MAX_IN_WINDOW; i++) claimAuthReload(window.sessionStorage, t0)
    expect(claimAuthReload(window.sessionStorage, t0 + AUTH_RELOAD_WINDOW_MS)).toBe(true)
  })

  it('valeur illisible en stockage : traitée comme vide', () => {
    window.sessionStorage.setItem(AUTH_RELOAD_STORAGE_KEY, '{pas du json')
    expect(claimAuthReload(window.sessionStorage, 4_000_000)).toBe(true)
  })

  it('sans stockage : permis (le garde /bootstrap reste)', () => {
    expect(claimAuthReload(null, 5_000_000)).toBe(true)
  })
})

describe('decideOnAuthRequired', () => {
  beforeEach(() => window.sessionStorage.clear())

  const deps = (b: () => Promise<BootstrapResponse>, now = 10_000_000) => ({
    fetchBootstrap: b,
    storage: window.sessionStorage,
    now: () => now,
  })

  it('/bootstrap connecté → aucune éjection, aucun rechargement compté', async () => {
    const v = await decideOnAuthRequired(deps(async () => bootstrap('alice')))
    expect(v.kind).toBe('still_authenticated')
    expect(window.sessionStorage.getItem(AUTH_RELOAD_STORAGE_KEY)).toBeNull()
  })

  it('/bootstrap anonyme → rechargement', async () => {
    const v = await decideOnAuthRequired(deps(async () => bootstrap(null)))
    expect(v.kind).toBe('reload')
  })

  it('/bootstrap anonyme, plafond atteint → reload_blocked avec le bootstrap', async () => {
    const anon = bootstrap(null)
    for (let i = 0; i < AUTH_RELOAD_MAX_IN_WINDOW; i++) {
      await decideOnAuthRequired(deps(async () => anon))
    }
    const v = await decideOnAuthRequired(deps(async () => anon))
    expect(v).toEqual({ kind: 'reload_blocked', bootstrap: anon })
  })

  it('/bootstrap injoignable → unknown, rien décidé', async () => {
    const v = await decideOnAuthRequired(deps(async () => Promise.reject(new Error('réseau'))))
    expect(v.kind).toBe('unknown')
  })
})
