/**
 * Setup Vitest + Testing Library + MSW.
 * Importé automatiquement via `setupFiles` dans vite.config.ts.
 *
 * Node.js v22+ expose un `globalThis.localStorage` natif qui n'implémente
 * pas l'interface Storage complète (pas de setItem/clear) quand aucun fichier
 * n'est fourni via --localstorage-file. On injecte un mock in-memory complet
 * AVANT que jsdom ou Zustand n'accèdent au storage.
 */
import '@testing-library/jest-dom'
import { setupServer } from 'msw/node'
import { handlers } from './handlers'

// ─── Mock Storage (Node.js v22+ compat) ──────────────────────────────────────

function createStorageMock(): Storage {
  const store: Record<string, string> = {}
  return {
    getItem:    (key: string)          => store[key] ?? null,
    setItem:    (key: string, val: string) => { store[key] = String(val) },
    removeItem: (key: string)          => { delete store[key] },
    clear:      ()                     => { for (const k in store) delete store[k] },
    get length()                       { return Object.keys(store).length },
    key:        (i: number)            => Object.keys(store)[i] ?? null,
  }
}

if (typeof localStorage === 'undefined' || typeof localStorage.setItem !== 'function') {
  Object.defineProperty(globalThis, 'localStorage', {
    value: createStorageMock(), writable: true, configurable: true,
  })
}
if (typeof sessionStorage === 'undefined' || typeof sessionStorage.setItem !== 'function') {
  Object.defineProperty(globalThis, 'sessionStorage', {
    value: createStorageMock(), writable: true, configurable: true,
  })
}

// ─── MSW ─────────────────────────────────────────────────────────────────────

export const server = setupServer(...handlers)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => {
  server.resetHandlers()
  if (typeof localStorage !== 'undefined' && typeof localStorage.clear === 'function') {
    localStorage.clear()
  }
})
afterAll(() => server.close())
