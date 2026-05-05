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

// Listeners + originaux mémorisés pour pouvoir uninstall en tests.
// Ne sont jamais nettoyés en prod (l'install est définitive).
interface InstallationState {
  errorListener?: EventListener
  rejectionListener?: EventListener
  originalConsoleError?: typeof console.error
  originalConsoleWarn?: typeof console.warn
  originalFetch?: typeof window.fetch
}
let _state: InstallationState = {}

export function installGlobalCapture(): void {
  if ((globalThis as Record<string, unknown>)[FLAG_KEY]) return
  ;(globalThis as Record<string, unknown>)[FLAG_KEY] = true

  // Chaque wrap est isolé pour qu'une exception sur l'un n'empêche pas
  // les autres de s'installer. L'app continue de fonctionner même si la
  // capture est partiellement KO.
  safeRun('wrapConsole(error)', () => wrapConsole('error'))
  safeRun('wrapConsole(warn)', () => wrapConsole('warn'))
  safeRun('wrapWindowErrors', () => wrapWindowErrors())
  safeRun('wrapFetch', () => wrapFetch())
}

function safeRun(label: string, fn: () => void): void {
  try {
    fn()
  } catch (err) {
    // Pas d'import du logger feature ici (évite cycle) — log direct console.
    // Clé stable [capture:install_failed] pour grep en prod.
    console.error(`[global-capture] capture:install_failed step=${label}`, err)
  }
}

/** Réinitialise le flag pour les tests Vitest (interne — ne pas exporter en prod). */
export function _resetInstallFlagForTests(): void {
  ;(globalThis as Record<string, unknown>)[FLAG_KEY] = undefined
}

/**
 * Restaure tout ce que installGlobalCapture a wrappé : console.error/warn,
 * window.fetch, et retire les listeners 'error' / 'unhandledrejection'.
 *
 * Réservé aux tests Vitest pour éviter la pollution cross-fichiers (un
 * worker partage le même `window` jsdom entre tests). Sans ce reset, les
 * listeners s'accumulent entre les fichiers de tests et ralentissent les
 * suites avec beaucoup de re-renders.
 */
export function _uninstallGlobalCaptureForTests(): void {
  if (typeof window !== 'undefined') {
    if (_state.errorListener) {
      window.removeEventListener('error', _state.errorListener)
    }
    if (_state.rejectionListener) {
      window.removeEventListener('unhandledrejection', _state.rejectionListener)
    }
    if (_state.originalFetch) {
      window.fetch = _state.originalFetch
    }
  }
  if (_state.originalConsoleError) {
    console.error = _state.originalConsoleError
  }
  if (_state.originalConsoleWarn) {
    console.warn = _state.originalConsoleWarn
  }
  _state = {}
  _resetInstallFlagForTests()
}

function wrapConsole(level: 'error' | 'warn'): void {
  // Stocker la référence native (pas un .bind, sinon chaque cycle
  // install/uninstall accumule un niveau de bind).
  const original = console[level]
  if (level === 'error') _state.originalConsoleError = original
  else _state.originalConsoleWarn = original
  const callOriginal = original.bind(console)
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
    callOriginal(...args)
  }
}

function wrapWindowErrors(): void {
  if (typeof window === 'undefined') return
  const onError = ((e: ErrorEvent) => {
    recordConsole({
      level: 'error',
      message: e.message,
      timestamp: Date.now(),
      stack: e.error instanceof Error ? e.error.stack : undefined,
    })
  }) as EventListener
  const onRejection = ((e: PromiseRejectionEvent) => {
    const reason = e.reason
    recordConsole({
      level: 'error',
      message: reason instanceof Error ? reason.message : String(reason),
      timestamp: Date.now(),
      stack: reason instanceof Error ? reason.stack : undefined,
    })
  }) as EventListener
  window.addEventListener('error', onError)
  window.addEventListener('unhandledrejection', onRejection)
  _state.errorListener = onError
  _state.rejectionListener = onRejection
}

function wrapFetch(): void {
  if (typeof window === 'undefined' || typeof window.fetch !== 'function') return
  // Stocker la référence native (pas un .bind, sinon chaque cycle
  // install/uninstall accumule un niveau de bind).
  const original = window.fetch
  _state.originalFetch = original
  const callOriginal = original.bind(window)
  window.fetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const method = init?.method ?? (input instanceof Request ? input.method : 'GET')
    const url = resolveUrl(input)
    try {
      const response = await callOriginal(input, init)
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
