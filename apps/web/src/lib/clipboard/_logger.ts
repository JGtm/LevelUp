/**
 * _logger.ts — Logger namespacé pour le module presse-papier.
 *
 * Même pattern que `lib/accessibility/_logger.ts` : chaque clé n'est logguée
 * qu'une fois par session (un presse-papier refusé l'est à chaque clic, on ne
 * veut pas noyer la console).
 *
 * Clés stables :
 *   - `clipboard:write_failed` — `navigator.clipboard.writeText` a rejeté
 *     (contexte non sécurisé, permission refusée, API absente).
 */

const PREFIX = '[clipboard]'

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
