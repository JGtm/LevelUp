/**
 * installGlobalCapture — point d'entrée des side-effects globaux du
 * drawer feedback (wrap console.error/warn, window error handlers,
 * intercepteur fetch pour les requêtes échouées).
 *
 * Idempotence sous HMR Vite : un flag module-level serait reset à chaque
 * recharge de module, ce qui re-instrumenterait les wraps et doublerait
 * (puis triplerait) la capture. On stocke donc le flag sur `globalThis`,
 * qui survit au HMR.
 *
 * Appelé une seule fois en tête de main.tsx (avant createRoot).
 */
import {
  extractStackFromArgs,
  recordConsole,
  recordFailedRequest,
  stringifyConsoleArgs,
} from './buffers'

const FLAG_KEY = '__levelup_global_capture_installed__'

declare global {
  var __levelup_global_capture_installed__: boolean | undefined
}

export function installGlobalCapture(): void {
  if ((globalThis as Record<string, unknown>)[FLAG_KEY]) return
  ;(globalThis as Record<string, unknown>)[FLAG_KEY] = true

  wrapConsole('error')
  wrapConsole('warn')
  wrapWindowErrors()
  wrapFetch()
}

/** Réinitialise le flag pour les tests Vitest (interne — ne pas exporter en prod). */
export function _resetInstallFlagForTests(): void {
  ;(globalThis as Record<string, unknown>)[FLAG_KEY] = undefined
}

function wrapConsole(level: 'error' | 'warn'): void {
  const original = console[level].bind(console)
  console[level] = (...args: unknown[]) => {
    try {
      recordConsole({
        level,
        message: stringifyConsoleArgs(args),
        timestamp: Date.now(),
        stack: extractStackFromArgs(args),
      })
    } catch {
      // ne jamais casser console.error — fail silently côté capture
    }
    original(...args)
  }
}

function wrapWindowErrors(): void {
  if (typeof window === 'undefined') return
  window.addEventListener('error', (e: ErrorEvent) => {
    recordConsole({
      level: 'error',
      message: e.message,
      timestamp: Date.now(),
      stack: e.error instanceof Error ? e.error.stack : undefined,
    })
  })
  window.addEventListener('unhandledrejection', (e: PromiseRejectionEvent) => {
    const reason = e.reason
    recordConsole({
      level: 'error',
      message: reason instanceof Error ? reason.message : String(reason),
      timestamp: Date.now(),
      stack: reason instanceof Error ? reason.stack : undefined,
    })
  })
}

function wrapFetch(): void {
  if (typeof window === 'undefined' || typeof window.fetch !== 'function') return
  const originalFetch = window.fetch.bind(window)
  window.fetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const method = init?.method ?? (input instanceof Request ? input.method : 'GET')
    const url = resolveUrl(input)
    try {
      const response = await originalFetch(input, init)
      if (!response.ok) {
        recordFailedRequest({ url, method, status: response.status, timestamp: Date.now() })
      }
      return response
    } catch (err) {
      // Erreur réseau (DNS, offline, CORS…) : status=0 par convention
      recordFailedRequest({ url, method, status: 0, timestamp: Date.now() })
      throw err
    }
  }
}

function resolveUrl(input: RequestInfo | URL): string {
  if (typeof input === 'string') return input
  if (input instanceof URL) return input.toString()
  return input.url
}
