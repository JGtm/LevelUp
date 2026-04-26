/**
 * _logger.ts — Logger namespacé pour la feature Escouade.
 *
 * Pattern aligné avec lib/accessibility/_logger.ts :
 * - chaque clé d'avertissement n'est logguée qu'une fois par session,
 * - jamais de log dans les hot paths (rendu de chaque card / chaque axe).
 *
 * Cible deux signaux observables :
 *  1. `field_missing:{slug}:{key}` — un FieldKey listé dans SQUAD_*_METRICS
 *     est absent du fields.toml du titre courant. Indique une dégradation
 *     gracieuse plutôt qu'un bug.
 *  2. `invalid_selection:{slug}` — un gamertag confirmé n'a matché aucun
 *     teammate côté backend (cause racine du bug "Comparaison inactive même
 *     après sélection").
 */

const PREFIX = '[squad]'

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
