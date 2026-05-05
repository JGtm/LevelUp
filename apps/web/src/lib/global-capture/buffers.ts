/**
 * Ring buffers de capture globale — alimentés par install.ts.
 *
 * Stockent en mémoire (FIFO) les dernières erreurs console + requêtes fetch
 * échouées, pour enrichir les issues GitHub envoyées via le drawer feedback.
 *
 * Sécurité PII : les URLs des requêtes échouées sont systématiquement
 * strippées de leur query string avant stockage (`url.split('?')[0]`),
 * pour éviter de leak des tokens, gamertags ou IDs en clair dans le body
 * de l'issue (qui sera publié sur un repo public).
 */

export type ConsoleLevel = 'error' | 'warn'

export interface ConsoleEntry {
  level: ConsoleLevel
  message: string
  timestamp: number
  stack?: string
}

export interface FailedRequest {
  url: string
  method: string
  status: number
  timestamp: number
}

const CONSOLE_MAX = 20
const NETWORK_MAX = 5

const consoleBuffer: ConsoleEntry[] = []
const networkBuffer: FailedRequest[] = []

function pushBounded<T>(buf: T[], item: T, max: number): void {
  buf.push(item)
  if (buf.length > max) buf.shift()
}

export function recordConsole(entry: ConsoleEntry): void {
  pushBounded(consoleBuffer, entry, CONSOLE_MAX)
}

export function recordFailedRequest(req: FailedRequest): void {
  // Strip query string : pas de fuite de tokens / gamertags / IDs.
  const sanitized: FailedRequest = { ...req, url: stripQueryString(req.url) }
  pushBounded(networkBuffer, sanitized, NETWORK_MAX)
}

export function getRecentConsoleEntries(): ConsoleEntry[] {
  return consoleBuffer.slice()
}

export function getRecentFailedRequests(): FailedRequest[] {
  return networkBuffer.slice()
}

export function resetCaptureBuffersForTests(): void {
  consoleBuffer.length = 0
  networkBuffer.length = 0
}

/**
 * Extrait `.stack` d'un argument de console.error/warn :
 * - Error → stack lisible
 * - autre → undefined (le caller fera un stringify safe pour le message)
 */
export function extractStackFromArgs(args: unknown[]): string | undefined {
  for (const a of args) {
    if (a instanceof Error && typeof a.stack === 'string') return a.stack
  }
  return undefined
}

export function stringifyConsoleArgs(args: unknown[]): string {
  return args
    .map((a) => {
      if (a instanceof Error) return a.message
      if (typeof a === 'string') return a
      try {
        return JSON.stringify(a)
      } catch {
        return String(a)
      }
    })
    .join(' ')
}

function stripQueryString(url: string): string {
  const qIdx = url.indexOf('?')
  return qIdx === -1 ? url : url.slice(0, qIdx)
}
