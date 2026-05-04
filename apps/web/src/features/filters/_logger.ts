/**
 * _logger.ts — Logger namespacé pour la feature filtres.
 *
 * Pattern aligné avec lib/accessibility/_logger.ts et features/squad/_logger.ts :
 * - chaque clé warn/error n'est logguée qu'une fois par session,
 * - debug actif uniquement en dev,
 * - jamais dans les hot paths (rendu pill, popover).
 *
 * Cibles observables :
 *  1. `auto_snap:fired` — un auto-snap a déclenché (transition sync→idle avec
 *     nouvelle session détectée). Diag utile pour vérifier que le flow marche.
 *  2. `auto_snap:skipped:{reason}` — auto-snap court-circuité (pas de session
 *     dispo, ou identique au dernier connu).
 *  3. `hydrate:source:{url|localStorage|defaults}` — quelle source a alimenté
 *     les filtres au mount. Aide à débugger les surprises de "mes filtres
 *     n'ont pas survécu".
 */

const PREFIX = '[filters]'

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
