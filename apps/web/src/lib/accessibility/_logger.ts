/**
 * _logger.ts — Logger namespacé pour le module accessibilité.
 *
 * Principe : logger uniquement les transitions d'état (changements de palette,
 * erreurs de résolution). Jamais dans les hot paths (useColor, resolveToken).
 *
 * Déduplication : chaque clé d'erreur n'est logguée qu'une fois par session
 * pour éviter de polluer la console lors de renders répétés.
 */

const PREFIX = '[accessibility]'

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
