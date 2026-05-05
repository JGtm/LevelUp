/**
 * _logger.ts — Logger namespacé pour la feature feedback-drawer.
 *
 * Pattern aligné avec features/filters/_logger.ts et features/squad/_logger.ts :
 *   - chaque clé warn/error n'est logguée qu'une fois par session,
 *   - debug actif uniquement en dev,
 *   - jamais dans les hot paths (debounce typing, preview Markdown live).
 *
 * Cibles observables (clés stables) :
 *   - `health:fetch_failed` — /health KO → version "unknown" injectée
 *   - `similar:fetch_failed` — GitHub search API down → section masquée
 *   - `similar:rate_limited` — HTTP 403 GitHub (60 req/h dépassé)
 *   - `url:truncated` — body Markdown > 7000 chars, troncature appliquée
 *   - `clipboard:open_failed` — window.open retourne null (popup bloqué)
 *   - `capture:install_failed` — installGlobalCapture() a throw (très rare)
 */

const PREFIX = '[feedback-drawer]'

const _warned = new Set<string>()
const _errored = new Set<string>()

export const log = {
  info(msg: string, ...args: unknown[]): void {
    console.info(`${PREFIX} ${msg}`, ...args)
  },

  warn(key: string, msg: string, ...args: unknown[]): void {
    if (_warned.has(key)) return
    _warned.add(key)
    console.warn(`${PREFIX} ${msg}`, ...args)
  },

  error(key: string, msg: string, ...args: unknown[]): void {
    if (_errored.has(key)) return
    _errored.add(key)
    console.error(`${PREFIX} ${msg}`, ...args)
  },

  debug(msg: string, ...args: unknown[]): void {
    if (!import.meta.env.DEV) return
    console.debug(`${PREFIX} ${msg}`, ...args)
  },

  /** Réinitialise la déduplication (usage test uniquement). */
  _resetForTests(): void {
    _warned.clear()
    _errored.clear()
  },
}
