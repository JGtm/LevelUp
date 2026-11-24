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

// ─── Mock matchMedia (gap jsdom) ─────────────────────────────────────────────
// jsdom n'implémente pas window.matchMedia, requis par useMediaQuery (charts
// responsives). Stub minimal : ne matche jamais (desktop par défaut).
if (typeof window !== 'undefined' && typeof window.matchMedia !== 'function') {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: (query: string): MediaQueryList =>
      ({
        matches: false,
        media: query,
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => false,
      }) as unknown as MediaQueryList,
  })
}

// ─── Mock lecture média (gap jsdom) ──────────────────────────────────────────
// jsdom n'implémente ni play(), ni pause(), ni load() d'un élément média, ni getContext() d'un canvas
// (sans le paquet canvas), ni scrollTo() : chaque appel écrit « Not implemented » sur la console du
// worker. Ces écritures partent par le canal du worker vers le processus principal ; celles encore en vol à la fermeture du worker font échouer tout le lot
// (EnvironmentTeardownError « Closing rpc while onUserConsoleLog was pending »). Stub sans effet :
// aucun test ne dépend du comportement natif (les tests déclenchent eux-mêmes les événements) ; getContext
// rend null, comme jsdom.
if (typeof HTMLMediaElement !== 'undefined') {
  Object.defineProperty(HTMLMediaElement.prototype, 'play', {
    configurable: true, writable: true, value: () => Promise.resolve(),
  })
  Object.defineProperty(HTMLMediaElement.prototype, 'pause', {
    configurable: true, writable: true, value: () => {},
  })
  Object.defineProperty(HTMLMediaElement.prototype, 'load', {
    configurable: true, writable: true, value: () => {},
  })
}
if (typeof HTMLCanvasElement !== 'undefined') {
  Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
    configurable: true, writable: true, value: () => null,
  })
}
if (typeof window !== 'undefined') {
  Object.defineProperty(window, 'scrollTo', { configurable: true, writable: true, value: () => {} })
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
