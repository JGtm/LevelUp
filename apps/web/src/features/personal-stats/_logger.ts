/**
 * _logger.ts — Logger namespacé pour la feature Personal Stats.
 *
 * Pattern aligné avec features/squad/_logger.ts :
 * - chaque clé d'avertissement n'est logguée qu'une fois par session,
 * - jamais de log dans les hot paths (rendu de chaque card / chaque axe).
 */

const PREFIX = '[personal-stats]'

const _warned = new Set<string>()

export const log = {
  warn(key: string, msg: string, ...args: unknown[]): void {
    if (_warned.has(key)) return
    _warned.add(key)
    console.warn(`${PREFIX} ${msg}`, ...args)
  },

  /** Réinitialise la déduplication (usage test uniquement). */
  _resetForTests(): void {
    _warned.clear()
  },
}
